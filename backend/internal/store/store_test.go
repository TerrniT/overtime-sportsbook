package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sportsbook/backend/internal/store"
	"sportsbook/backend/internal/testsupport"
)

const (
	demoBalance int64 = 2_500_000
	stakeMinor  int64 = 50_000
	priceMilli  int64 = 1910
)

func newTestStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := testsupport.NewDB(t)
	return store.New(pool), pool
}

func testInput(t *testing.T, id, key string, stake, price int64) store.PlaceBetInput {
	t.Helper()
	return store.PlaceBetInput{
		ID:                   id,
		PlayerID:             store.DemoPlayerID,
		IdempotencyKey:       key,
		MatchID:              "epl-ars-che",
		MatchLabel:           "Арсенал — Челси",
		Outcome:              "home",
		OutcomeLabel:         "П1",
		StakeMinor:           stake,
		PriceMilli:           price,
		PotentialPayoutMinor: stake * price / 1000,
		Now:                  time.Now().UTC(),
	}
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var n int64
	if err := pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		t.Fatalf("count query failed: %v\nquery: %s", err, query)
	}
	return n
}

func TestPlayerReturnsSeededDemoPlayer(t *testing.T) {
	s, _ := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	player, err := s.Player(ctx, store.DemoPlayerID)
	if err != nil {
		t.Fatalf("Player(demo) error: %v", err)
	}
	if player.ID != store.DemoPlayerID {
		t.Fatalf("player id = %q, want %q", player.ID, store.DemoPlayerID)
	}
	if player.Name != "Игрок" {
		t.Fatalf("player name = %q, want Игрок", player.Name)
	}
	if player.Currency != "RUB" {
		t.Fatalf("player currency = %q, want RUB", player.Currency)
	}
	if player.BalanceMinor != demoBalance {
		t.Fatalf("player balance = %d, want %d", player.BalanceMinor, demoBalance)
	}
}

func TestPlayerNotFound(t *testing.T) {
	s, _ := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := s.Player(ctx, testsupport.UUID(t)); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Player(unknown) error = %v, want store.ErrNotFound", err)
	}
}

func TestPlaceBetDebitsOnceAndWritesLedger(t *testing.T) {
	s, pool := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	in := testInput(t, testsupport.UUID(t), testsupport.UUID(t), stakeMinor, priceMilli)
	bet, balance, replayed, err := s.PlaceBet(ctx, in)
	if err != nil {
		t.Fatalf("PlaceBet error: %v", err)
	}
	if replayed {
		t.Fatal("first PlaceBet reported replayed=true")
	}
	if balance != demoBalance-stakeMinor {
		t.Fatalf("balance = %d, want %d", balance, demoBalance-stakeMinor)
	}

	if bet.ID != in.ID {
		t.Fatalf("bet id = %q, want %q", bet.ID, in.ID)
	}
	if bet.MatchID != in.MatchID || bet.MatchLabel != in.MatchLabel {
		t.Fatalf("bet match fields = %+v, want match %q / %q", bet, in.MatchID, in.MatchLabel)
	}
	if bet.Outcome != in.Outcome || bet.OutcomeLabel != in.OutcomeLabel {
		t.Fatalf("bet outcome fields = %+v, want %q / %q", bet, in.Outcome, in.OutcomeLabel)
	}
	if bet.StakeMinor != stakeMinor || bet.PriceMilli != priceMilli {
		t.Fatalf("bet stake/price = %d/%d, want %d/%d", bet.StakeMinor, bet.PriceMilli, stakeMinor, priceMilli)
	}
	const wantPayout int64 = 95_500
	if bet.PotentialPayoutMinor != wantPayout {
		t.Fatalf("potential payout = %d, want %d", bet.PotentialPayoutMinor, wantPayout)
	}
	if bet.Status != "pending" {
		t.Fatalf("bet status = %q, want pending", bet.Status)
	}
	if bet.PlacedAt.IsZero() {
		t.Fatal("bet placedAt is zero")
	}

	// The row really is in the database, not just in memory.
	if n := countRows(t, pool, `select count(*) from bets`); n != 1 {
		t.Fatalf("bets count = %d, want 1", n)
	}
	if n := countRows(t, pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); n != 1 {
		t.Fatalf("bet_stake ledger rows = %d, want exactly 1", n)
	}
	var (
		dbID, dbMatch, dbLabel, dbOutcome, dbOutcomeLabel, dbStatus string
		dbStake, dbPrice, dbPayout                                  int64
		dbPlacedAt                                                  time.Time
	)
	err = pool.QueryRow(ctx, `
		select id::text, match_id, match_label, outcome, outcome_label,
		       stake_minor, price_milli, potential_payout_minor, status, placed_at
		from bets where id = $1`, in.ID).
		Scan(&dbID, &dbMatch, &dbLabel, &dbOutcome, &dbOutcomeLabel, &dbStake, &dbPrice, &dbPayout, &dbStatus, &dbPlacedAt)
	if err != nil {
		t.Fatalf("reading stored bet: %v", err)
	}
	if dbID != in.ID || dbMatch != in.MatchID || dbLabel != in.MatchLabel ||
		dbOutcome != in.Outcome || dbOutcomeLabel != in.OutcomeLabel ||
		dbStake != stakeMinor || dbPrice != priceMilli || dbPayout != wantPayout || dbStatus != "pending" {
		t.Fatalf("stored bet differs from input: %+v", []any{dbID, dbMatch, dbLabel, dbOutcome, dbOutcomeLabel, dbStake, dbPrice, dbPayout, dbStatus, dbPlacedAt})
	}
	if !dbPlacedAt.Equal(bet.PlacedAt) {
		t.Fatalf("stored placedAt = %s, returned placedAt = %s", dbPlacedAt, bet.PlacedAt)
	}

	var (
		kind         string
		amount       int64
		balanceAfter int64
		betID        string
	)
	err = pool.QueryRow(ctx, `
		select kind, amount_minor, balance_after_minor, coalesce(bet_id::text, '')
		from ledger_entries where kind = 'bet_stake' and player_id = $1`, store.DemoPlayerID).
		Scan(&kind, &amount, &balanceAfter, &betID)
	if err != nil {
		t.Fatalf("reading ledger entry: %v", err)
	}
	if kind != "bet_stake" {
		t.Fatalf("ledger kind = %q, want bet_stake", kind)
	}
	if amount != -stakeMinor {
		t.Fatalf("ledger amount = %d, want %d", amount, -stakeMinor)
	}
	if balanceAfter != demoBalance-stakeMinor {
		t.Fatalf("ledger balance_after = %d, want %d", balanceAfter, demoBalance-stakeMinor)
	}
	if betID != "" && betID != in.ID {
		t.Fatalf("ledger bet_id = %q, want %q (or empty)", betID, in.ID)
	}

	player, err := s.Player(ctx, store.DemoPlayerID)
	if err != nil {
		t.Fatalf("Player after bet: %v", err)
	}
	if player.BalanceMinor != balance {
		t.Fatalf("player balance = %d, PlaceBet returned %d", player.BalanceMinor, balance)
	}
}

func TestPlaceBetReplayDoesNotDebitAgain(t *testing.T) {
	s, pool := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	key := testsupport.UUID(t)
	first, firstBalance, replayed, err := s.PlaceBet(ctx, testInput(t, testsupport.UUID(t), key, stakeMinor, priceMilli))
	if err != nil || replayed {
		t.Fatalf("first PlaceBet: replayed=%v err=%v", replayed, err)
	}

	// A retry after a client timeout builds a fresh bet id but reuses the key.
	second, secondBalance, replayed, err := s.PlaceBet(ctx, testInput(t, testsupport.UUID(t), key, 60_000, 1950))
	if err != nil {
		t.Fatalf("replay PlaceBet error: %v", err)
	}
	if !replayed {
		t.Fatal("replay PlaceBet reported replayed=false")
	}
	if second.ID != first.ID {
		t.Fatalf("replay bet id = %q, want original %q", second.ID, first.ID)
	}
	if secondBalance != firstBalance {
		t.Fatalf("replay balance = %d, want unchanged %d", secondBalance, firstBalance)
	}
	if second.StakeMinor != stakeMinor || second.PriceMilli != priceMilli {
		t.Fatalf("replay returned a rewritten bet: stake/price = %d/%d, want original %d/%d",
			second.StakeMinor, second.PriceMilli, stakeMinor, priceMilli)
	}

	if n := countRows(t, pool, `select count(*) from bets`); n != 1 {
		t.Fatalf("bets count after replay = %d, want 1", n)
	}
	if n := countRows(t, pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); n != 1 {
		t.Fatalf("bet_stake ledger rows after replay = %d, want 1", n)
	}
	if got := countRows(t, pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != demoBalance-stakeMinor {
		t.Fatalf("balance after replay = %d, want %d", got, demoBalance-stakeMinor)
	}
}

func TestPlaceBetConcurrentSameKeyDebitsOnce(t *testing.T) {
	s, pool := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const goroutines = 10
	key := testsupport.UUID(t)

	type result struct {
		bet      store.Bet
		balance  int64
		replayed bool
		err      error
	}
	inputs := make([]store.PlaceBetInput, goroutines)
	for i := range inputs {
		inputs[i] = testInput(t, testsupport.UUID(t), key, stakeMinor, priceMilli)
	}
	results := make([]result, goroutines)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			bet, balance, replayed, err := s.PlaceBet(ctx, inputs[i])
			results[i] = result{bet: bet, balance: balance, replayed: replayed, err: err}
		}(i)
	}
	close(start)
	wg.Wait()

	originalID := ""
	for i, res := range results {
		if res.err != nil {
			t.Fatalf("goroutine %d: PlaceBet error: %v", i, res.err)
		}
		if res.bet.ID == "" {
			t.Fatalf("goroutine %d: empty bet id", i)
		}
		if originalID == "" {
			originalID = res.bet.ID
		} else if res.bet.ID != originalID {
			t.Fatalf("goroutine %d returned bet id %q, want %q", i, res.bet.ID, originalID)
		}
		if res.balance != demoBalance-stakeMinor {
			t.Fatalf("goroutine %d balance = %d, want %d", i, res.balance, demoBalance-stakeMinor)
		}
	}

	if n := countRows(t, pool, `select count(*) from bets`); n != 1 {
		t.Fatalf("bets count = %d, want 1 (unique idempotency key)", n)
	}
	if n := countRows(t, pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); n != 1 {
		t.Fatalf("bet_stake ledger rows = %d, want exactly one debit", n)
	}
	if got := countRows(t, pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != demoBalance-stakeMinor {
		t.Fatalf("balance = %d, want %d", got, demoBalance-stakeMinor)
	}
}

func TestPlaceBetInsufficientFunds(t *testing.T) {
	s, pool := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	in := testInput(t, testsupport.UUID(t), testsupport.UUID(t), demoBalance+1, priceMilli)
	_, _, _, err := s.PlaceBet(ctx, in)
	if !errors.Is(err, store.ErrInsufficientFunds) {
		t.Fatalf("PlaceBet(stake over balance) error = %v, want ErrInsufficientFunds", err)
	}

	if n := countRows(t, pool, `select count(*) from bets`); n != 0 {
		t.Fatalf("bets count = %d after rejected bet, want 0", n)
	}
	if n := countRows(t, pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); n != 0 {
		t.Fatalf("bet_stake rows = %d after rejected bet, want 0", n)
	}
	if got := countRows(t, pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != demoBalance {
		t.Fatalf("balance = %d after rejected bet, want %d", got, demoBalance)
	}

	// stake == balance is allowed: only stake > balance is rejected.
	exact := testInput(t, testsupport.UUID(t), testsupport.UUID(t), demoBalance, priceMilli)
	_, balance, replayed, err := s.PlaceBet(ctx, exact)
	if err != nil {
		t.Fatalf("PlaceBet(stake == balance) error: %v", err)
	}
	if replayed {
		t.Fatal("PlaceBet(stake == balance) reported replayed=true")
	}
	if balance != 0 {
		t.Fatalf("balance after betting the whole balance = %d, want 0", balance)
	}
}

func TestConcurrentBetsNeverGoNegative(t *testing.T) {
	s, pool := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const (
		goroutines = 20
		stake      = int64(200_000)
		wantOK     = 12 // 2_500_000 / 200_000, rounded down
	)

	type result struct {
		bet     store.Bet
		balance int64
		err     error
	}
	inputs := make([]store.PlaceBetInput, goroutines)
	for i := range inputs {
		inputs[i] = testInput(t, testsupport.UUID(t), testsupport.UUID(t), stake, priceMilli)
	}
	results := make([]result, goroutines)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			bet, balance, _, err := s.PlaceBet(ctx, inputs[i])
			results[i] = result{bet: bet, balance: balance, err: err}
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	ids := map[string]bool{}
	for i, res := range results {
		if res.err != nil {
			if !errors.Is(res.err, store.ErrInsufficientFunds) {
				t.Fatalf("goroutine %d: unexpected error %v (want nil or ErrInsufficientFunds)", i, res.err)
			}
			continue
		}
		successes++
		if res.bet.ID == "" {
			t.Fatalf("goroutine %d: successful bet has empty id", i)
		}
		if ids[res.bet.ID] {
			t.Fatalf("goroutine %d: duplicate bet id %q", i, res.bet.ID)
		}
		ids[res.bet.ID] = true
		if res.balance < 0 {
			t.Fatalf("goroutine %d: negative balance %d returned", i, res.balance)
		}
	}

	if successes != wantOK {
		t.Fatalf("successful bets = %d, want exactly %d (each debits 200_000 from 2_500_000)", successes, wantOK)
	}

	wantBalance := demoBalance - stake*int64(successes)
	if got := countRows(t, pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != wantBalance {
		t.Fatalf("final balance = %d, want %d", got, wantBalance)
	}
	if got := countRows(t, pool, `select count(*) from bets`); got != int64(successes) {
		t.Fatalf("bets count = %d, want %d", got, successes)
	}
	if got := countRows(t, pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); got != int64(successes) {
		t.Fatalf("bet_stake ledger rows = %d, want %d", got, successes)
	}

	var ledgerSum int64
	if err := pool.QueryRow(ctx, `select coalesce(sum(amount_minor), 0) from ledger_entries where kind = 'bet_stake'`).Scan(&ledgerSum); err != nil {
		t.Fatalf("sum ledger amounts: %v", err)
	}
	if want := -stake * int64(successes); ledgerSum != want {
		t.Fatalf("ledger sum = %d, want %d", ledgerSum, want)
	}
}

func TestDatabaseRejectsNegativeBalance(t *testing.T) {
	_, pool := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, `update players set balance_minor = -1 where id = $1`, store.DemoPlayerID); err == nil {
		t.Fatal("direct UPDATE to a negative balance succeeded; the CHECK constraint is missing")
	}

	if got := countRows(t, pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != demoBalance {
		t.Fatalf("balance = %d after failed negative update, want %d", got, demoBalance)
	}
}
