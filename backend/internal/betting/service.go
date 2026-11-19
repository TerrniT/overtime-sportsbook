package betting

import (
	"context"
	"errors"
	"time"

	"sportsbook/backend/internal/line"
	"sportsbook/backend/internal/store"
)

var (
	ErrMarketNotFound   = errors.New("betting: market not found")
	ErrInvalidRequest   = errors.New("betting: invalid request")
	ErrStakeOutOfRange  = errors.New("betting: stake out of range")
	ErrInsufficientFunds = errors.New("betting: insufficient funds")
)

// OddsChangedError is returned when the client's expected price no longer
// matches the live line.
type OddsChangedError struct {
	Expected int64
	Current  int64
}

func (e *OddsChangedError) Error() string { return "betting: odds changed" }

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
}

// New creates the service.
func New(d Deps) *Service { return &Service{deps: d} }

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
	// TODO: implement (betting agent).
	return PlaceResult{}, errors.New("not implemented")
}

// Session returns the demo player.
func (s *Service) Session(ctx context.Context) (store.Player, error) {
	// TODO: implement (betting agent).
	return store.Player{}, errors.New("not implemented")
}

// History returns the player's bets, newest first.
func (s *Service) History(ctx context.Context, limit int) ([]store.Bet, error) {
	// TODO: implement (betting agent).
	return nil, errors.New("not implemented")
}

// BalanceEvent is published after a bet changes the player's balance.
type BalanceEvent struct {
	BalanceMinor int64  `json:"balanceMinor"`
	Currency     string `json:"currency"`
}

// SubscribeBalance returns a channel of balance updates and a cancel function.
func (s *Service) SubscribeBalance() (<-chan BalanceEvent, func()) {
	// TODO: implement (betting agent).
	ch := make(chan BalanceEvent, 1)
	return ch, func() {}
}
