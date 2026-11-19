package betting

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"sportsbook/backend/internal/line"
	"sportsbook/backend/internal/money"
	"sportsbook/backend/internal/store"
)

var (
	ErrMarketNotFound    = errors.New("betting: market not found")
	ErrInvalidRequest    = errors.New("betting: invalid request")
	ErrStakeOutOfRange   = errors.New("betting: stake out of range")
	ErrInsufficientFunds = errors.New("betting: insufficient funds")
	ErrOddsChanged       = errors.New("betting: odds changed")
)

// OddsChangedError is returned when the client's expected price no longer
// matches the live line.
type OddsChangedError struct {
	Expected int64
	Current  int64
}

func (e *OddsChangedError) Error() string { return "betting: odds changed" }

// Is lets errors.Is(err, ErrOddsChanged) work for the typed error too.
func (e *OddsChangedError) Is(target error) bool { return target == ErrOddsChanged }

// Deps wires the betting service.
type Deps struct {
	Store         *store.Store
	Engine        *line.Engine
	PlayerID      string
	MinStakeMinor int64
	MaxStakeMinor int64
	Now           func() time.Time
}

// Service applies betting rules on top of the store and the live line.
type Service struct {
	deps Deps

	subsMu sync.Mutex
	subs   map[chan BalanceEvent]struct{}
}

// New creates the service.
func New(d Deps) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Service{deps: d, subs: make(map[chan BalanceEvent]struct{})}
}

// PlaceInput is a validated bet request from the HTTP layer.
type PlaceInput struct {
	MatchID            string
	Outcome            string
	StakeMinor         int64
	ExpectedPriceMilli int64
	IdempotencyKey     string
}

// PlaceResult is the outcome of a successful bet.
type PlaceResult struct {
	Bet          store.Bet
	BalanceMinor int64
	Replayed     bool
}

// Place validates the market and price, then records the bet.
func (s *Service) Place(ctx context.Context, in PlaceInput) (PlaceResult, error) {
	key := strings.TrimSpace(in.IdempotencyKey)
	if in.MatchID == "" || in.StakeMinor <= 0 || in.ExpectedPriceMilli <= 0 ||
		len(key) < 8 || len(key) > 100 {
		return PlaceResult{}, ErrInvalidRequest
	}
	switch in.Outcome {
	case "home", "draw", "away":
	default:
		return PlaceResult{}, ErrInvalidRequest
	}

	snapshot := s.deps.Engine.Snapshot()
	match, ok := findMatch(snapshot, in.MatchID)
	if !ok {
		return PlaceResult{}, ErrMarketNotFound
	}
	outcome, ok := findOutcome(match, in.Outcome)
	if !ok {
		return PlaceResult{}, ErrMarketNotFound
	}

	if err := money.ValidateStake(in.StakeMinor, s.deps.MinStakeMinor, s.deps.MaxStakeMinor); err != nil {
		return PlaceResult{}, ErrStakeOutOfRange
	}

	current := outcome.PriceMilli
	if in.ExpectedPriceMilli != current {
		return PlaceResult{}, &OddsChangedError{Expected: in.ExpectedPriceMilli, Current: current}
	}

	payout, err := money.Payout(in.StakeMinor, current)
	if err != nil {
		return PlaceResult{}, ErrInvalidRequest
	}

	bet, balance, replayed, err := s.deps.Store.PlaceBet(ctx, store.PlaceBetInput{
		ID:                   uuid.NewString(),
		PlayerID:             s.deps.PlayerID,
		IdempotencyKey:       key,
		MatchID:              match.ID,
		MatchLabel:           match.HomeTeam + " — " + match.AwayTeam,
		Outcome:              outcome.Key,
		OutcomeLabel:         outcome.Label,
		StakeMinor:           in.StakeMinor,
		PriceMilli:           current,
		PotentialPayoutMinor: payout,
		Now:                  s.deps.Now(),
	})
	switch {
	case errors.Is(err, store.ErrInsufficientFunds):
		return PlaceResult{}, ErrInsufficientFunds
	case errors.Is(err, store.ErrNotFound):
		return PlaceResult{}, ErrMarketNotFound
	case err != nil:
		return PlaceResult{}, err
	}

	s.publish(BalanceEvent{BalanceMinor: balance, Currency: "RUB"})
	return PlaceResult{Bet: bet, BalanceMinor: balance, Replayed: replayed}, nil
}

// Session returns the demo player.
func (s *Service) Session(ctx context.Context) (store.Player, error) {
	return s.deps.Store.Player(ctx, s.deps.PlayerID)
}

// History returns the player's bets, newest first.
func (s *Service) History(ctx context.Context, limit int) ([]store.Bet, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	return s.deps.Store.Bets(ctx, s.deps.PlayerID, limit)
}

// BalanceEvent is published after a bet changes the player's balance.
type BalanceEvent struct {
	BalanceMinor int64  `json:"balanceMinor"`
	Currency     string `json:"currency"`
}

// SubscribeBalance returns a channel of balance updates and a cancel function.
// Like the line feed, delivery is best-effort: a slow subscriber never blocks
// a bet, and the channel is left open on cancel.
func (s *Service) SubscribeBalance() (<-chan BalanceEvent, func()) {
	ch := make(chan BalanceEvent, 1)

	s.subsMu.Lock()
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			s.subsMu.Lock()
			delete(s.subs, ch)
			s.subsMu.Unlock()
		})
	}
	return ch, cancel
}

func (s *Service) publish(event BalanceEvent) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()

	for ch := range s.subs {
		select {
		case ch <- event:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- event:
			default:
			}
		}
	}
}

func findMatch(snapshot line.Snapshot, id string) (line.Match, bool) {
	for _, match := range snapshot.Matches {
		if match.ID == id {
			return match, true
		}
	}
	return line.Match{}, false
}

func findOutcome(match line.Match, key string) (line.Outcome, bool) {
	for _, outcome := range match.Outcomes {
		if outcome.Key == key {
			return outcome, true
		}
	}
	return line.Outcome{}, false
}
