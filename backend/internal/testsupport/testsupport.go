// Package testsupport contains helpers shared by the database-backed test
// suites. It is only imported from _test.go files and never by cmd/server.
package testsupport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sportsbook/backend/internal/store"
)

// DemoStartBalanceMinor is the seeded demo balance.
const DemoStartBalanceMinor int64 = 2_500_000

// DatabaseURL returns TEST_DATABASE_URL or skips the calling test when the
// variable is unset. Integration tests never fall back to a default database.
func DatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	return url
}

// NewDB creates an isolated PostgreSQL schema for one test, applies the real
// migrations into it and returns a pool whose connections have
// search_path=<schema>. The schema is dropped by t.Cleanup.
//
// A unique schema per test keeps `go test ./...` safe: packages run in
// parallel and no test ever touches the public schema.
func NewDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := DatabaseURL(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("testsupport: open admin pool: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Fatalf("testsupport: ping database: %v", err)
	}

	schema := "test_" + randomHex(t)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "create schema "+quoted); err != nil {
		admin.Close()
		t.Fatalf("testsupport: create schema %s: %v", schema, err)
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		dropSchema(admin, quoted)
		admin.Close()
		t.Fatalf("testsupport: parse database url: %v", err)
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		dropSchema(admin, quoted)
		admin.Close()
		t.Fatalf("testsupport: open schema pool: %v", err)
	}

	if err := store.Migrate(ctx, pool); err != nil {
		pool.Close()
		dropSchema(admin, quoted)
		admin.Close()
		t.Fatalf("testsupport: migrate schema %s: %v", schema, err)
	}

	t.Cleanup(func() {
		pool.Close()
		if err := dropSchema(admin, quoted); err != nil {
			t.Errorf("testsupport: drop schema %s: %v", schema, err)
		}
		admin.Close()
	})

	return pool
}

// ResetDemo restores the state the HTTP tests expect: no bets, only the seed
// deposit in the ledger and the demo balance back at 2_500_000.
func ResetDemo(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// pgx uses the extended protocol for parameterized statements, which does
	// not allow several commands in one Exec. Keep each statement separate.
	if _, err := pool.Exec(ctx, `truncate table ledger_entries, bets restart identity`); err != nil {
		t.Fatalf("testsupport: truncate demo state: %v", err)
	}
	if _, err := pool.Exec(ctx, `update players set balance_minor = $1 where id = $2`, DemoStartBalanceMinor, store.DemoPlayerID); err != nil {
		t.Fatalf("testsupport: reset demo balance: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into ledger_entries (player_id, kind, amount_minor, balance_after_minor)
		select $1, 'deposit', $2, $2
		where not exists (
			select 1 from ledger_entries where player_id = $1 and kind = 'deposit'
		);
	`, store.DemoPlayerID, DemoStartBalanceMinor); err != nil {
		t.Fatalf("testsupport: restore demo deposit: %v", err)
	}
}

// SetBalance changes the demo player's balance directly. Tests use it to set
// up insufficient-funds scenarios without placing five real bets.
func SetBalance(t *testing.T, pool *pgxpool.Pool, balanceMinor int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tag, err := pool.Exec(ctx, `update players set balance_minor = $1 where id = $2`, balanceMinor, store.DemoPlayerID)
	if err != nil {
		t.Fatalf("testsupport: set balance: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("testsupport: set balance: demo player not found")
	}
}

// UUID returns a random RFC 4122 version 4 UUID string using only stdlib.
func UUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("testsupport: random uuid: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

func randomHex(t *testing.T) string {
	t.Helper()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("testsupport: random schema suffix: %v", err)
	}
	return hex.EncodeToString(b[:])
}

func dropSchema(admin *pgxpool.Pool, quotedSchema string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := admin.Exec(ctx, "drop schema if exists "+quotedSchema+" cascade")
	return err
}
