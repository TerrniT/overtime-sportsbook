package line

import (
	"context"
	"math"
	"math/rand"
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

type source struct {
	id           string
	label        string
	overroundBps int
}

var sources = [...]source{
	{id: "pulse", label: "PulseBet", overroundBps: 600},
	{id: "sharp", label: "SharpLine", overroundBps: 300},
}

// blendWeights pairs with sources: 40% pulse, 60% sharp.
var blendWeights = []float64{0.4, 0.6}

var outcomeKeys = [...]string{"home", "draw", "away"}
var outcomeLabels = [...]string{"П1", "Х", "П2"}

type fixture struct {
	id     string
	league string
	home   string
	away   string
	base   [3]float64
}

// fixtures are fixed so IDs, teams and kickoff times are stable across ticks.
var fixtures = [...]fixture{
	{id: "epl-ars-che", league: "АПЛ", home: "Арсенал", away: "Челси", base: [3]float64{0.46, 0.27, 0.27}},
	{id: "lal-rma-bar", league: "Ла Лига", home: "Реал Мадрид", away: "Барселона", base: [3]float64{0.42, 0.28, 0.30}},
	{id: "bund-bay-bvb", league: "Бундеслига", home: "Бавария", away: "Боруссия Дортмунд", base: [3]float64{0.52, 0.25, 0.23}},
	{id: "serie-int-juv", league: "Серия A", home: "Интер", away: "Ювентус", base: [3]float64{0.44, 0.30, 0.26}},
	{id: "ligue-psg-om", league: "Лига 1", home: "ПСЖ", away: "Марсель", base: [3]float64{0.55, 0.24, 0.21}},
	{id: "rpl-zen-spa", league: "РПЛ", home: "Зенит", away: "Спартак", base: [3]float64{0.47, 0.28, 0.25}},
	{id: "epl-liv-mci", league: "АПЛ", home: "Ливерпуль", away: "Манчестер Сити", base: [3]float64{0.38, 0.27, 0.35}},
}

// kickoffOffsets are relative to the moment the engine starts.
var kickoffOffsets = [...]time.Duration{
	2 * time.Hour, 4 * time.Hour, 6 * time.Hour, 26 * time.Hour,
	28 * time.Hour, 48 * time.Hour, 50 * time.Hour,
}

const (
	driftDecay = 0.65
	driftSigma = 0.03
	driftLimit = 0.30
)

// Engine owns the current line and broadcasts each tick.
type Engine struct {
	cfg      Config
	rng      *rand.Rand
	drift    [2][]float64 // per source, flattened match*3+outcome
	startsAt []time.Time

	mu       sync.RWMutex
	snapshot Snapshot

	subsMu sync.Mutex
	subs   map[chan Snapshot]struct{}
}

// NewEngine creates the engine and computes the first snapshot.
func NewEngine(cfg Config) *Engine {
	e := &Engine{
		cfg:      cfg,
		rng:      rand.New(rand.NewSource(cfg.Seed)),
		startsAt: make([]time.Time, len(fixtures)),
		subs:     make(map[chan Snapshot]struct{}),
	}

	now := e.now()
	for i := range fixtures {
		e.startsAt[i] = now.Add(kickoffOffsets[i])
	}

	size := len(fixtures) * len(outcomeKeys)
	e.drift = [2][]float64{make([]float64, size), make([]float64, size)}
	e.advanceDrift()
	e.snapshot = e.render(0, now)
	return e
}

// Snapshot returns the current line. The result shares no memory with the
// engine, so a caller may keep or modify it freely.
func (e *Engine) Snapshot() Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return cloneSnapshot(e.snapshot)
}

// Update advances the simulation by one tick and returns the new snapshot.
func (e *Engine) Update() Snapshot {
	e.mu.Lock()
	e.advanceDrift()
	snap := e.render(e.snapshot.Seq+1, e.now())
	e.snapshot = snap
	e.mu.Unlock()

	e.broadcast(snap)
	return cloneSnapshot(snap)
}

// Run ticks the engine until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	tick := e.cfg.Tick
	if tick <= 0 {
		tick = 3 * time.Second
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.Update()
		}
	}
}

// Subscribe returns a channel of snapshots and a cancel function.
// Delivery is best-effort: a slow subscriber gets the newest snapshot and
// loses older pending ones, so a tick never blocks. The channel is not
// closed on cancel (the engine cannot know whether a send is in flight).
func (e *Engine) Subscribe() (<-chan Snapshot, func()) {
	ch := make(chan Snapshot, 1)

	e.subsMu.Lock()
	e.subs[ch] = struct{}{}
	e.subsMu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			e.subsMu.Lock()
			delete(e.subs, ch)
			e.subsMu.Unlock()
		})
	}
	return ch, cancel
}

func (e *Engine) now() time.Time {
	if e.cfg.Now != nil {
		return e.cfg.Now()
	}
	return time.Now()
}

// advanceDrift moves every per-source, per-outcome bias one mean-reverting
// step. The iteration order is fixed by the slices, never a map, and the
// random source is seeded once, so a seed always produces the same line.
func (e *Engine) advanceDrift() {
	for s := range e.drift {
		for i := range e.drift[s] {
			next := e.drift[s][i]*driftDecay + e.rng.NormFloat64()*driftSigma
			if next > driftLimit {
				next = driftLimit
			}
			if next < -driftLimit {
				next = -driftLimit
			}
			e.drift[s][i] = next
		}
	}
}

// render builds the complete line for one tick. The caller holds e.mu.
func (e *Engine) render(seq int64, now time.Time) Snapshot {
	snap := Snapshot{
		Seq:       seq,
		UpdatedAt: now,
		Matches:   make([]Match, len(fixtures)),
	}

	var impliedTotal float64
	for mi, f := range fixtures {
		match := Match{
			ID:       f.id,
			Sport:    "Футбол",
			League:   f.league,
			StartsAt: e.startsAt[mi],
			HomeTeam: f.home,
			AwayTeam: f.away,
			Outcomes: make([]Outcome, len(outcomeKeys)),
		}

		// Each source quotes the same base event with its own drift and its
		// own overround, so deleting the vig is the only way to compare them.
		sourcePrices := make([][]int64, len(sources))
		for si := range sources {
			probs := make([]float64, len(outcomeKeys))
			var total float64
			for oi := range outcomeKeys {
				p := f.base[oi] * math.Exp(e.drift[si][mi*len(outcomeKeys)+oi])
				probs[oi] = p
				total += p
			}
			for oi := range probs {
				probs[oi] /= total
			}
			sourcePrices[si] = WithMargin(probs, sources[si].overroundBps)
		}

		fair := make([][]float64, len(sources))
		for si := range sources {
			fair[si] = Devig(sourcePrices[si])
		}
		ourPrices := WithMargin(Blend(fair, blendWeights), e.cfg.MarginBps)

		var matchImplied float64
		for oi, key := range outcomeKeys {
			price := ourPrices[oi]
			matchImplied += 1000 / float64(price)

			trend := "flat"
			if seq > 0 && mi < len(e.snapshot.Matches) && oi < len(e.snapshot.Matches[mi].Outcomes) {
				previous := e.snapshot.Matches[mi].Outcomes[oi].PriceMilli
				switch {
				case price > previous:
					trend = "up"
				case price < previous:
					trend = "down"
				}
			}

			quotes := make([]SourceQuote, len(sources))
			for si := range sources {
				quotes[si] = SourceQuote{
					ID:         sources[si].id,
					Label:      sources[si].label,
					PriceMilli: sourcePrices[si][oi],
				}
			}

			match.Outcomes[oi] = Outcome{
				Key:        key,
				Label:      outcomeLabels[oi],
				PriceMilli: price,
				Trend:      trend,
				Sources:    quotes,
			}
		}
		snap.Matches[mi] = match
		impliedTotal += matchImplied
	}

	// marginBps is the average achieved margin across markets, measured on the
	// final rounded prices rather than the pre-rounding target. Flooring
	// prices can only raise it; clamping a heavy favourite is the one case
	// that could lower it, hence the floor at the configured target.
	snap.MarginBps = int(math.Round((impliedTotal/float64(len(fixtures)) - 1) * 10_000))
	if snap.MarginBps < e.cfg.MarginBps {
		snap.MarginBps = e.cfg.MarginBps
	}
	return snap
}

func (e *Engine) broadcast(snap Snapshot) {
	e.subsMu.Lock()
	defer e.subsMu.Unlock()

	for ch := range e.subs {
		copied := cloneSnapshot(snap)
		select {
		case ch <- copied:
		default:
			// Drop the oldest pending snapshot, then offer the newest one.
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- copied:
			default:
			}
		}
	}
}

func cloneSnapshot(s Snapshot) Snapshot {
	out := s
	out.Matches = make([]Match, len(s.Matches))
	for i, match := range s.Matches {
		out.Matches[i] = match
		out.Matches[i].Outcomes = make([]Outcome, len(match.Outcomes))
		for j, outcome := range match.Outcomes {
			out.Matches[i].Outcomes[j] = outcome
			if outcome.Sources != nil {
				out.Matches[i].Outcomes[j].Sources = append([]SourceQuote(nil), outcome.Sources...)
			}
		}
	}
	return out
}
