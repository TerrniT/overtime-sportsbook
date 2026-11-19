package app

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"sportsbook/backend/internal/betting"
	"sportsbook/backend/internal/line"
)

// Limits are the public stake limits.
type Limits struct {
	MinStakeMinor int64 `json:"minStakeMinor"`
	MaxStakeMinor int64 `json:"maxStakeMinor"`
}

// Deps wires the HTTP app.
type Deps struct {
	Service *betting.Service
	Engine  *line.Engine
	Limits  Limits
	Now     func() time.Time
}

// New builds the Fiber app with every /api route.
func New(d Deps) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "overtime"})
	// TODO: implement routes (http agent).
	_ = d
	return app
}
