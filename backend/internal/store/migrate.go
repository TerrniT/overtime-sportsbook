package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrateLockKey serializes concurrent migration attempts in one database.
const migrateLockKey int64 = 0x73706f727473 // "sports"

// Migrate applies the embedded schema migrations in filename order.
//
// Applied files are recorded in schema_migrations, and the whole run happens
// in one transaction under an advisory lock, so several API instances (or
// parallel test schemas) cannot apply the same migration twice.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin migration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock($1)`, migrateLockKey); err != nil {
		return fmt.Errorf("store: acquire migration lock: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		create table if not exists schema_migrations (
			filename   text primary key,
			applied_at timestamptz not null default now()
		)`); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}

	files, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("store: list migrations: %w", err)
	}
	sort.Strings(files)

	for _, name := range files {
		var applied bool
		if err := tx.QueryRow(ctx,
			`select exists (select 1 from schema_migrations where filename = $1)`, name,
		).Scan(&applied); err != nil {
			return fmt.Errorf("store: check migration %s: %w", name, err)
		}
		if applied {
			continue
		}

		sql, err := migrationsFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("store: read migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("store: apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx,
			`insert into schema_migrations (filename) values ($1)`, name,
		); err != nil {
			return fmt.Errorf("store: record migration %s: %w", name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit migrations: %w", err)
	}
	return nil
}
