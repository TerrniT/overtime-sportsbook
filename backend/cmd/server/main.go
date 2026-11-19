package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sportsbook/backend/internal/app"
	"sportsbook/backend/internal/betting"
	"sportsbook/backend/internal/config"
	"sportsbook/backend/internal/line"
	"sportsbook/backend/internal/store"
)

const (
	minStakeMinor = int64(10_000)
	maxStakeMinor = int64(500_000)
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := connectDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	migrateCtx, cancelMigrate := context.WithTimeout(ctx, 30*time.Second)
	err = store.Migrate(migrateCtx, pool)
	cancelMigrate()
	if err != nil {
		slog.Error("database migration failed", "err", err)
		os.Exit(1)
	}

	engine := line.NewEngine(line.Config{
		Seed:      cfg.Seed,
		MarginBps: cfg.MarginBps,
		Tick:      cfg.LineTick,
		Now:       time.Now,
	})
	engineDone := make(chan struct{})
	go func() {
		defer close(engineDone)
		engine.Run(ctx)
	}()

	service := betting.New(betting.Deps{
		Store:         store.New(pool),
		Engine:        engine,
		PlayerID:      store.DemoPlayerID,
		MinStakeMinor: minStakeMinor,
		MaxStakeMinor: maxStakeMinor,
		Now:           time.Now,
	})
	fiberApp := app.New(app.Deps{
		Service: service,
		Engine:  engine,
		Limits:  app.Limits{MinStakeMinor: minStakeMinor, MaxStakeMinor: maxStakeMinor},
		Now:     time.Now,
		Pool:    pool,
	})

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	listenErr := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", addr, "seed", cfg.Seed,
			"marginBps", cfg.MarginBps, "tick", cfg.LineTick)
		listenErr <- fiberApp.Listen(addr)
	}()

	select {
	case err := <-listenErr:
		if err != nil {
			slog.Error("http server stopped", "err", err)
		}
		stop()
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	if err := fiberApp.ShutdownWithTimeout(10 * time.Second); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
	<-engineDone
	slog.Info("server stopped")
}

// connectDatabase retries for up to 30 seconds so the API survives a database
// container that is still starting.
func connectDatabase(ctx context.Context, url string) (*pgxpool.Pool, error) {
	const timeout = 30 * time.Second
	deadline := time.Now().Add(timeout)

	var lastErr error
	for {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("not reachable within %s: %w", timeout, lastErr)
		}
		slog.Warn("database not ready, retrying", "err", lastErr)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
