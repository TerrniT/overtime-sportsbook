package line

import (
	"context"
	"sync"
	"time"
)

// SourceQuote is one sportsbook's price for an outcome.
type SourceQuote struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	PriceMilli int64  `json:"priceMilli"`
}

// Outcome is one leg of the 1X2 market.
type Outcome struct {
	Key        string        `json:"key"`
	Label      string        `json:"label"`
	PriceMilli int64         `json:"priceMilli"`
	Trend      string        `json:"trend"`
	Sources    []SourceQuote `json:"sources"`
}

// Match is a football fixture with its 1X2 market.
type Match struct {
	ID       string    `json:"id"`
	Sport    string    `json:"sport"`
	League   string    `json:"league"`
	StartsAt time.Time `json:"startsAt"`
	HomeTeam string    `json:"homeTeam"`
	AwayTeam string    `json:"awayTeam"`
	Outcomes []Outcome `json:"outcomes"`
}

// Snapshot is the complete public line at one tick.
type Snapshot struct {
	Seq       int64     `json:"seq"`
	UpdatedAt time.Time `json:"updatedAt"`
	MarginBps int       `json:"marginBps"`
	Matches   []Match   `json:"matches"`
}

// Config controls the simulated line.
type Config struct {
	Seed      int64
	MarginBps int
	Tick      time.Duration
	Now       func() time.Time
}

// Engine owns the current line and broadcasts each tick.
type Engine struct {
	cfg Config

	mu       sync.RWMutex
	snapshot Snapshot
	prev     Snapshot

	subsMu sync.Mutex
	subs   map[chan Snapshot]struct{}
}

// NewEngine creates the engine and computes the first snapshot.
func NewEngine(cfg Config) *Engine {
	// TODO: implement (line agent).
	return &Engine{cfg: cfg}
}

// Snapshot returns the current line.
func (e *Engine) Snapshot() Snapshot {
	// TODO: implement (line agent).
	return Snapshot{}
}

// Update advances the simulation by one tick and returns the new snapshot.
func (e *Engine) Update() Snapshot {
	// TODO: implement (line agent).
	return Snapshot{}
}

// Run ticks the engine until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	// TODO: implement (line agent).
	<-ctx.Done()
}

// Subscribe returns a channel of snapshots and a cancel function.
func (e *Engine) Subscribe() (<-chan Snapshot, func()) {
	// TODO: implement (line agent).
	ch := make(chan Snapshot, 1)
	return ch, func() {}
}
