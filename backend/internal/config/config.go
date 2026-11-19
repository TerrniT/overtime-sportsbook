// Package config reads the server configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the complete runtime configuration.
type Config struct {
	DatabaseURL string
	Port        int
	LineTick    time.Duration
	MarginBps   int
	Seed        int64
}

// Load reads and validates every supported environment variable.
func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	port, err := intEnv("PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535, got %d", port)
	}

	tickMS, err := intEnv("LINE_TICK_MS", 3000)
	if err != nil {
		return Config{}, err
	}
	if tickMS <= 0 {
		return Config{}, fmt.Errorf("LINE_TICK_MS must be positive, got %d", tickMS)
	}

	marginBps, err := intEnv("MARGIN_BPS", 500)
	if err != nil {
		return Config{}, err
	}
	if marginBps < 0 || marginBps >= 10_000 {
		return Config{}, fmt.Errorf("MARGIN_BPS must be in [0, 10000), got %d", marginBps)
	}

	seed, err := int64Env("LINE_SEED", 7)
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL: databaseURL,
		Port:        port,
		LineTick:    time.Duration(tickMS) * time.Millisecond,
		MarginBps:   marginBps,
		Seed:        seed,
	}, nil
}

func intEnv(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", name, raw)
	}
	return value, nil
}

func int64Env(name string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", name, raw)
	}
	return value, nil
}
