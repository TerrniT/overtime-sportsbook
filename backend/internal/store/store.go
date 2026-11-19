package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

const betColumns = `id::text, match_id, match_label, outcome, outcome_label,
	stake_minor, price_milli, potential_payout_minor, status, placed_at`

func scanBet(row pgx.Row) (Bet, error) {
	var bet Bet
	err := row.Scan(&bet.ID, &bet.MatchID, &bet.MatchLabel, &bet.Outcome, &bet.OutcomeLabel,
		&bet.StakeMinor, &bet.PriceMilli, &bet.PotentialPayoutMinor, &bet.Status, &bet.PlacedAt)
	return bet, err
}

// Player loads one player.
func (s *Store) Player(ctx context.Context, id string) (Player, error) {
	var player Player
	err := s.pool.QueryRow(ctx, `
		select id::text, name, currency, balance_minor
		from players where id = $1`, id).
		Scan(&player.ID, &player.Name, &player.Currency, &player.BalanceMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return Player{}, ErrNotFound
	}
	if err != nil {
		return Player{}, err
	}
	return player, nil
}

// Bets lists a player's bets, newest first. The limit is clamped to 1..50;
// a non-positive limit means the default of 20.
func (s *Store) Bets(ctx context.Context, playerID string, limit int) ([]Bet, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
		select `+betColumns+`
		from bets
		where player_id = $1
		order by placed_at desc, id desc
		limit $2`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bets := make([]Bet, 0, limit)
	for rows.Next() {
		bet, err := scanBet(rows)
		if err != nil {
			return nil, err
		}
		bets = append(bets, bet)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bets, nil
}

// FindBet looks up an accepted bet by (player, idempotency key). The second
// result is false when no such bet exists. It runs before price, stake and
// balance checks so that a retry of an accepted bet always replays, even if
// the line or the balance has moved since the first attempt.
func (s *Store) FindBet(ctx context.Context, playerID, key string) (Bet, bool, error) {
	bet, err := scanBet(s.pool.QueryRow(ctx, `
		select `+betColumns+`
		from bets
		where player_id = $1 and idempotency_key = $2`, playerID, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return Bet{}, false, nil
	}
	if err != nil {
		return Bet{}, false, err
	}
	return bet, true, nil
}

// PlaceBet records the bet, debits the stake and writes a ledger entry in one
// transaction. It returns the stored bet, the new balance, and whether the
// request was an idempotent replay.
func (s *Store) PlaceBet(ctx context.Context, in PlaceBetInput) (Bet, int64, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Bet{}, 0, false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// Lock the player first: every bet for this player serializes here, which
	// makes the balance check and the idempotency lookup race-free.
	var balance int64
	err = tx.QueryRow(ctx,
		`select balance_minor from players where id = $1 for update`, in.PlayerID,
	).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bet{}, 0, false, ErrNotFound
	}
	if err != nil {
		return Bet{}, 0, false, err
	}

	existing, err := scanBet(tx.QueryRow(ctx, `
		select `+betColumns+`
		from bets
		where player_id = $1 and idempotency_key = $2`, in.PlayerID, in.IdempotencyKey))
	switch {
	case err == nil:
		return existing, balance, true, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return Bet{}, 0, false, err
	}

	if balance < in.StakeMinor {
		return Bet{}, 0, false, ErrInsufficientFunds
	}

	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}

	bet, err := scanBet(tx.QueryRow(ctx, `
		insert into bets (
			id, player_id, idempotency_key, match_id, match_label,
			outcome, outcome_label, stake_minor, price_milli,
			potential_payout_minor, status, placed_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', $11)
		returning `+betColumns,
		in.ID, in.PlayerID, in.IdempotencyKey, in.MatchID, in.MatchLabel,
		in.Outcome, in.OutcomeLabel, in.StakeMinor, in.PriceMilli,
		in.PotentialPayoutMinor, now))
	if err != nil {
		// The unique (player_id, idempotency_key) index is the last line of
		// defence. Read the winner of the race and report a replay instead of
		// debiting twice.
		if isUniqueViolation(err) {
			_ = tx.Rollback(ctx)
			committed = true // rollback already happened; skip the deferred one
			return s.replayBet(ctx, in.PlayerID, in.IdempotencyKey, err)
		}
		return Bet{}, 0, false, err
	}

	err = tx.QueryRow(ctx, `
		update players
		set balance_minor = balance_minor - $1
		where id = $2 and balance_minor >= $1
		returning balance_minor`, in.StakeMinor, in.PlayerID,
	).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bet{}, 0, false, ErrInsufficientFunds
	}
	if err != nil {
		return Bet{}, 0, false, err
	}

	if _, err := tx.Exec(ctx, `
		insert into ledger_entries (player_id, bet_id, kind, amount_minor, balance_after_minor)
		values ($1, $2, 'bet_stake', $3, $4)`,
		in.PlayerID, in.ID, -in.StakeMinor, balance,
	); err != nil {
		return Bet{}, 0, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Bet{}, 0, false, err
	}
	committed = true
	return bet, balance, false, nil
}

// replayBet reads an already stored bet and the player's current balance. It
// runs after the unique-violation rollback, so it cannot reuse the failed
// transaction.
func (s *Store) replayBet(ctx context.Context, playerID, key string, cause error) (Bet, int64, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Bet{}, 0, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	bet, err := scanBet(tx.QueryRow(ctx, `
		select `+betColumns+`
		from bets
		where player_id = $1 and idempotency_key = $2`, playerID, key))
	if err != nil {
		return Bet{}, 0, false, cause
	}

	var balance int64
	if err := tx.QueryRow(ctx,
		`select balance_minor from players where id = $1`, playerID,
	).Scan(&balance); err != nil {
		return Bet{}, 0, false, err
	}
	return bet, balance, true, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
