package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DemoPlayerID is the only player in the demo.
const DemoPlayerID = "11111111-1111-4111-8111-111111111111"

var (
	ErrNotFound          = errors.New("store: not found")
	ErrInsufficientFunds = errors.New("store: insufficient funds")
)

// Player is the demo bettor.
type Player struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Currency     string `json:"currency"`
	BalanceMinor int64  `json:"balanceMinor"`
}

// Bet is a stored, pending wager.
type Bet struct {
	ID                   string    `json:"id"`
	MatchID              string    `json:"matchId"`
	MatchLabel           string    `json:"matchLabel"`
	Outcome              string    `json:"outcome"`
	OutcomeLabel         string    `json:"outcomeLabel"`
	StakeMinor           int64     `json:"stakeMinor"`
	PriceMilli           int64     `json:"priceMilli"`
	PotentialPayoutMinor int64     `json:"potentialPayoutMinor"`
	Status               string    `json:"status"`
	PlacedAt             time.Time `json:"placedAt"`
}

// PlaceBetInput is one atomic bet attempt.
type PlaceBetInput struct {
	ID                   string
	PlayerID             string
	IdempotencyKey       string
	MatchID              string
	MatchLabel           string
	Outcome              string
	OutcomeLabel         string
	StakeMinor           int64
	PriceMilli           int64
	PotentialPayoutMinor int64
	Now                  time.Time
}

// Store is the PostgreSQL data access layer.
type Store struct {
	pool *pgxpool.Pool
}

// New wraps a connection pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Migrate applies the embedded schema migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// TODO: implement (store agent).
	return errors.New("not implemented")
}

// Player loads one player.
func (s *Store) Player(ctx context.Context, id string) (Player, error) {
	// TODO: implement (store agent).
	return Player{}, errors.New("not implemented")
}

// Bets lists a player's bets, newest first.
func (s *Store) Bets(ctx context.Context, playerID string, limit int) ([]Bet, error) {
	// TODO: implement (store agent).
	return nil, errors.New("not implemented")
}

// PlaceBet records the bet, debits the stake and writes a ledger entry in one
// transaction. It returns the stored bet, the new balance, and whether the
// request was an idempotent replay.
func (s *Store) PlaceBet(ctx context.Context, in PlaceBetInput) (Bet, int64, bool, error) {
	// TODO: implement (store agent).
	return Bet{}, 0, false, errors.New("not implemented")
}
