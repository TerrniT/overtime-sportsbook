package app_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"sportsbook/backend/internal/app"
	"sportsbook/backend/internal/betting"
	"sportsbook/backend/internal/line"
	"sportsbook/backend/internal/store"
	"sportsbook/backend/internal/testsupport"
)

var fixedTime = time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC)

type testEnv struct {
	app        *fiber.App
	engine     *line.Engine
	service    *betting.Service
	store      *store.Store
	pool       *pgxpool.Pool
	settleRoll *int
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	pool := testsupport.NewDB(t)
	testsupport.ResetDemo(t, pool)

	engine := line.NewEngine(line.Config{
		Seed:      7,
		MarginBps: 500,
		Tick:      20 * time.Millisecond,
		Now:       func() time.Time { return fixedTime },
	})

	// Placed-at timestamps increase by a second per service call, which makes
	// "newest first" deterministic without touching the database directly.
	var clock atomic.Int64
	serviceNow := func() time.Time {
		return fixedTime.Add(time.Duration(clock.Add(1)-1) * time.Second)
	}

	settleRoll := 0

	svc := betting.New(betting.Deps{
		Store:         store.New(pool),
		Engine:        engine,
		PlayerID:      store.DemoPlayerID,
		MinStakeMinor: 10_000,
		MaxStakeMinor: 500_000,
		Now:           serviceNow,
		SettleRoll:    func() (int, error) { return settleRoll, nil },
	})

	fiberApp := app.New(app.Deps{
		Service: svc,
		Engine:  engine,
		Limits:  app.Limits{MinStakeMinor: 10_000, MaxStakeMinor: 500_000},
		Now:     func() time.Time { return fixedTime },
	})
	t.Cleanup(func() { _ = fiberApp.ShutdownWithTimeout(2 * time.Second) })

	return &testEnv{
		app:        fiberApp,
		engine:     engine,
		service:    svc,
		store:      store.New(pool),
		pool:       pool,
		settleRoll: &settleRoll,
	}
}

// --- HTTP helpers ---------------------------------------------------------

func request(t *testing.T, fiberApp *fiber.App, method, path string, body any) (int, []byte, http.Header) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := fiberApp.Test(req, 5000)
	if err != nil {
		t.Fatalf("%s %s: app.Test error: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read response: %v", method, path, err)
	}
	return resp.StatusCode, data, resp.Header
}

func requireJSONContentType(t *testing.T, header http.Header) {
	t.Helper()
	if ct := header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
}

func object(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("response is not a JSON object: %v\nbody: %s", err, data)
	}
	return obj
}

func requireKeys(t *testing.T, obj map[string]json.RawMessage, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := obj[key]; !ok {
			t.Fatalf("response object is missing JSON field %q; has %v", key, mapKeys(obj))
		}
	}
}

func mapKeys(obj map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	return keys
}

func decode[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("decode JSON: %v\nbody: %s", err, data)
	}
	return v
}

type apiError struct {
	Error struct {
		Code    string                     `json:"code"`
		Message string                     `json:"message"`
		Details map[string]json.RawMessage `json:"details"`
	} `json:"error"`
}

func expectError(t *testing.T, status int, body []byte, wantStatus int, wantCode string) apiError {
	t.Helper()
	if status != wantStatus {
		t.Fatalf("status = %d, want %d\nbody: %s", status, wantStatus, body)
	}
	top := object(t, body)
	requireKeys(t, top, "error")
	errObj := object(t, top["error"])
	requireKeys(t, errObj, "code", "message")

	apiErr := decode[apiError](t, body)
	if apiErr.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q\nbody: %s", apiErr.Error.Code, wantCode, body)
	}
	if strings.TrimSpace(apiErr.Error.Message) == "" {
		t.Fatalf("error message is empty for code %q", wantCode)
	}
	return apiErr
}

type betRequest struct {
	MatchID            string `json:"matchId"`
	Outcome            string `json:"outcome"`
	StakeMinor         int64  `json:"stakeMinor"`
	ExpectedPriceMilli int64  `json:"expectedPriceMilli"`
	IdempotencyKey     string `json:"idempotencyKey"`
}

type betDTO struct {
	ID                   string `json:"id"`
	MatchID              string `json:"matchId"`
	MatchLabel           string `json:"matchLabel"`
	Outcome              string `json:"outcome"`
	OutcomeLabel         string `json:"outcomeLabel"`
	StakeMinor           int64  `json:"stakeMinor"`
	PriceMilli           int64  `json:"priceMilli"`
	PotentialPayoutMinor int64  `json:"potentialPayoutMinor"`
	Status               string `json:"status"`
	PlacedAt             string `json:"placedAt"`
}

type betResponse struct {
	Bet          betDTO `json:"bet"`
	BalanceMinor int64  `json:"balanceMinor"`
	Currency     string `json:"currency"`
	Replayed     bool   `json:"replayed"`
}

type settleResponse struct {
	Bet          betDTO `json:"bet"`
	BalanceMinor int64  `json:"balanceMinor"`
	Currency     string `json:"currency"`
	Won          bool   `json:"won"`
}

func pickHome(t *testing.T, env *testEnv) (line.Match, line.Outcome) {
	t.Helper()
	snap := env.engine.Snapshot()
	if len(snap.Matches) == 0 {
		t.Fatal("engine snapshot has no matches")
	}
	match := snap.Matches[0]
	if len(match.Outcomes) < 1 {
		t.Fatalf("match %q has no outcomes", match.ID)
	}
	return match, match.Outcomes[0]
}

func betBody(match line.Match, outcome line.Outcome, stake int64, key string) betRequest {
	return betRequest{
		MatchID:            match.ID,
		Outcome:            outcome.Key,
		StakeMinor:         stake,
		ExpectedPriceMilli: outcome.PriceMilli,
		IdempotencyKey:     key,
	}
}

func countQuery(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var n int64
	if err := pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		t.Fatalf("query failed: %v\nquery: %s", err, query)
	}
	return n
}

// --- Tests ----------------------------------------------------------------

func TestSession(t *testing.T) {
	env := newTestEnv(t)
	status, body, header := request(t, env.app, http.MethodGet, "/api/session", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /api/session status = %d, want 200\nbody: %s", status, body)
	}
	requireJSONContentType(t, header)

	top := object(t, body)
	requireKeys(t, top, "player", "limits", "serverTime")
	playerObj := object(t, top["player"])
	requireKeys(t, playerObj, "id", "name", "balanceMinor", "currency")
	limitsObj := object(t, top["limits"])
	requireKeys(t, limitsObj, "minStakeMinor", "maxStakeMinor")

	resp := decode[struct {
		Player struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			BalanceMinor int64  `json:"balanceMinor"`
			Currency     string `json:"currency"`
		} `json:"player"`
		Limits struct {
			MinStakeMinor int64 `json:"minStakeMinor"`
			MaxStakeMinor int64 `json:"maxStakeMinor"`
		} `json:"limits"`
		ServerTime string `json:"serverTime"`
	}](t, body)

	if resp.Player.ID != store.DemoPlayerID {
		t.Fatalf("player id = %q, want %q", resp.Player.ID, store.DemoPlayerID)
	}
	if resp.Player.Name != "Игрок" {
		t.Fatalf("player name = %q, want Игрок", resp.Player.Name)
	}
	if resp.Player.BalanceMinor != 2_500_000 {
		t.Fatalf("balance = %d, want 2500000", resp.Player.BalanceMinor)
	}
	if resp.Player.Currency != "RUB" {
		t.Fatalf("currency = %q, want RUB", resp.Player.Currency)
	}
	if resp.Limits.MinStakeMinor != 10_000 || resp.Limits.MaxStakeMinor != 500_000 {
		t.Fatalf("limits = %+v, want min 10000 max 500000", resp.Limits)
	}
	if _, err := time.Parse(time.RFC3339, resp.ServerTime); err != nil {
		t.Fatalf("serverTime %q is not RFC3339: %v", resp.ServerTime, err)
	}
}

func TestLine(t *testing.T) {
	env := newTestEnv(t)
	status, body, header := request(t, env.app, http.MethodGet, "/api/line", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /api/line status = %d, want 200\nbody: %s", status, body)
	}
	requireJSONContentType(t, header)

	top := object(t, body)
	requireKeys(t, top, "seq", "updatedAt", "marginBps", "matches")

	var matches []map[string]json.RawMessage
	if err := json.Unmarshal(top["matches"], &matches); err != nil {
		t.Fatalf("matches is not an array: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("line has no matches")
	}

	for _, matchObj := range matches {
		requireKeys(t, matchObj, "id", "sport", "league", "startsAt", "homeTeam", "awayTeam", "outcomes")
		var outcomes []map[string]json.RawMessage
		if err := json.Unmarshal(matchObj["outcomes"], &outcomes); err != nil {
			t.Fatalf("outcomes is not an array: %v", err)
		}
		if len(outcomes) != 3 {
			t.Fatalf("match has %d outcomes, want 3", len(outcomes))
		}
		for _, outcomeObj := range outcomes {
			requireKeys(t, outcomeObj, "key", "label", "priceMilli", "trend", "sources")
			var sources []map[string]json.RawMessage
			if err := json.Unmarshal(outcomeObj["sources"], &sources); err != nil {
				t.Fatalf("sources is not an array: %v", err)
			}
			if len(sources) != 2 {
				t.Fatalf("outcome has %d sources, want 2", len(sources))
			}
			for _, sourceObj := range sources {
				requireKeys(t, sourceObj, "id", "label", "priceMilli")
			}
		}
	}

	snap := decode[line.Snapshot](t, body)
	if snap.Seq < 0 {
		t.Fatalf("seq = %d, want >= 0", snap.Seq)
	}
	if snap.UpdatedAt.IsZero() {
		t.Fatal("updatedAt is zero")
	}
	if snap.MarginBps <= 0 {
		t.Fatalf("marginBps = %d, want > 0", snap.MarginBps)
	}
	if len(snap.Matches) == 0 {
		t.Fatal("decoded line has no matches")
	}
	for _, match := range snap.Matches {
		if len(match.Outcomes) != 3 {
			t.Fatalf("match %q has %d outcomes, want 3", match.ID, len(match.Outcomes))
		}
		wantKeys := []string{"home", "draw", "away"}
		for i, outcome := range match.Outcomes {
			if outcome.Key != wantKeys[i] {
				t.Fatalf("match %q outcome %d = %q, want %q", match.ID, i, outcome.Key, wantKeys[i])
			}
			if outcome.PriceMilli < 1050 || outcome.PriceMilli > 20_000 || outcome.PriceMilli%10 != 0 {
				t.Fatalf("match %q outcome %q has invalid price %d", match.ID, outcome.Key, outcome.PriceMilli)
			}
			if outcome.Trend != "up" && outcome.Trend != "down" && outcome.Trend != "flat" {
				t.Fatalf("match %q outcome %q has invalid trend %q", match.ID, outcome.Key, outcome.Trend)
			}
		}
	}
}

func TestPlaceBetHappyPath(t *testing.T) {
	env := newTestEnv(t)
	match, outcome := pickHome(t, env)

	status, body, header := request(t, env.app, http.MethodPost, "/api/bets",
		betBody(match, outcome, 50_000, testsupport.UUID(t)))
	if status != http.StatusCreated {
		t.Fatalf("POST /api/bets status = %d, want 201\nbody: %s", status, body)
	}
	requireJSONContentType(t, header)

	top := object(t, body)
	requireKeys(t, top, "bet", "balanceMinor", "currency", "replayed")
	betObj := object(t, top["bet"])
	requireKeys(t, betObj, "id", "matchId", "matchLabel", "outcome", "outcomeLabel",
		"stakeMinor", "priceMilli", "potentialPayoutMinor", "status", "placedAt")

	resp := decode[betResponse](t, body)
	if resp.Replayed {
		t.Fatal("first bet reported replayed=true")
	}
	if resp.BalanceMinor != 2_450_000 {
		t.Fatalf("balance = %d, want 2450000", resp.BalanceMinor)
	}
	if resp.Currency != "RUB" {
		t.Fatalf("currency = %q, want RUB", resp.Currency)
	}
	if resp.Bet.ID == "" {
		t.Fatal("bet id is empty")
	}
	if resp.Bet.MatchID != match.ID || resp.Bet.Outcome != outcome.Key {
		t.Fatalf("bet market = %q/%q, want %q/%q", resp.Bet.MatchID, resp.Bet.Outcome, match.ID, outcome.Key)
	}
	if !strings.Contains(resp.Bet.MatchLabel, match.HomeTeam) || !strings.Contains(resp.Bet.MatchLabel, match.AwayTeam) {
		t.Fatalf("matchLabel = %q, want both team names %q and %q", resp.Bet.MatchLabel, match.HomeTeam, match.AwayTeam)
	}
	if resp.Bet.OutcomeLabel != outcome.Label {
		t.Fatalf("outcomeLabel = %q, want %q", resp.Bet.OutcomeLabel, outcome.Label)
	}
	if resp.Bet.StakeMinor != 50_000 {
		t.Fatalf("stake = %d, want 50000", resp.Bet.StakeMinor)
	}
	if resp.Bet.PriceMilli != outcome.PriceMilli {
		t.Fatalf("price = %d, want current %d", resp.Bet.PriceMilli, outcome.PriceMilli)
	}
	wantPayout := int64(50_000) * outcome.PriceMilli / 1000
	if resp.Bet.PotentialPayoutMinor != wantPayout {
		t.Fatalf("potential payout = %d, want floor(%d * %d / 1000) = %d",
			resp.Bet.PotentialPayoutMinor, 50_000, outcome.PriceMilli, wantPayout)
	}
	if resp.Bet.Status != "pending" {
		t.Fatalf("status = %q, want pending", resp.Bet.Status)
	}
	if _, err := time.Parse(time.RFC3339, resp.Bet.PlacedAt); err != nil {
		t.Fatalf("placedAt %q is not RFC3339: %v", resp.Bet.PlacedAt, err)
	}

	if got := countQuery(t, env.pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != 2_450_000 {
		t.Fatalf("db balance = %d, want 2450000", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 1 {
		t.Fatalf("db bets = %d, want 1", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from ledger_entries where kind = 'bet_stake' and amount_minor = -50000 and balance_after_minor = 2450000`); got != 1 {
		t.Fatalf("bet_stake ledger rows = %d, want 1 with amount -50000 and balance_after 2450000", got)
	}
}

func TestPlaceBetIdempotentReplay(t *testing.T) {
	env := newTestEnv(t)
	match, outcome := pickHome(t, env)
	body := betBody(match, outcome, 50_000, testsupport.UUID(t))

	status, firstBody, _ := request(t, env.app, http.MethodPost, "/api/bets", body)
	if status != http.StatusCreated {
		t.Fatalf("first POST status = %d, want 201\nbody: %s", status, firstBody)
	}
	first := decode[betResponse](t, firstBody)

	status, secondBody, _ := request(t, env.app, http.MethodPost, "/api/bets", body)
	if status != http.StatusOK {
		t.Fatalf("replay POST status = %d, want 200\nbody: %s", status, secondBody)
	}
	second := decode[betResponse](t, secondBody)
	if !second.Replayed {
		t.Fatal("replay response replayed=false, want true")
	}
	if second.Bet.ID != first.Bet.ID {
		t.Fatalf("replay bet id = %q, want %q", second.Bet.ID, first.Bet.ID)
	}
	if second.BalanceMinor != 2_450_000 {
		t.Fatalf("replay balance = %d, want unchanged 2450000", second.BalanceMinor)
	}

	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 1 {
		t.Fatalf("db bets = %d, want 1", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); got != 1 {
		t.Fatalf("bet_stake ledger rows = %d, want 1", got)
	}
}

func TestPlaceBetOddsChanged(t *testing.T) {
	env := newTestEnv(t)
	match, outcome := pickHome(t, env)
	body := betBody(match, outcome, 50_000, testsupport.UUID(t))
	body.ExpectedPriceMilli = outcome.PriceMilli + 10

	status, respBody, _ := request(t, env.app, http.MethodPost, "/api/bets", body)
	apiErr := expectError(t, status, respBody, http.StatusConflict, "odds_changed")

	if apiErr.Error.Details == nil {
		t.Fatal("odds_changed has no details")
	}
	if _, ok := apiErr.Error.Details["expectedPriceMilli"]; !ok {
		t.Fatalf("details missing expectedPriceMilli: %s", respBody)
	}
	if _, ok := apiErr.Error.Details["priceMilli"]; !ok {
		t.Fatalf("details missing priceMilli: %s", respBody)
	}
	var expected, current int64
	if err := json.Unmarshal(apiErr.Error.Details["expectedPriceMilli"], &expected); err != nil {
		t.Fatalf("decode expectedPriceMilli: %v", err)
	}
	if err := json.Unmarshal(apiErr.Error.Details["priceMilli"], &current); err != nil {
		t.Fatalf("decode priceMilli: %v", err)
	}
	if expected != outcome.PriceMilli+10 {
		t.Fatalf("details.expectedPriceMilli = %d, want %d", expected, outcome.PriceMilli+10)
	}
	if current != outcome.PriceMilli {
		t.Fatalf("details.priceMilli = %d, want current %d", current, outcome.PriceMilli)
	}

	// A price rejection must not move money or leave a bet behind.
	if got := countQuery(t, env.pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != 2_500_000 {
		t.Fatalf("balance = %d after odds_changed, want 2500000", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 0 {
		t.Fatalf("bets = %d after odds_changed, want 0", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); got != 0 {
		t.Fatalf("bet_stake rows = %d after odds_changed, want 0", got)
	}
}

func TestPlaceBetInsufficientFunds(t *testing.T) {
	env := newTestEnv(t)
	testsupport.SetBalance(t, env.pool, 40_000)
	match, outcome := pickHome(t, env)

	status, respBody, _ := request(t, env.app, http.MethodPost, "/api/bets",
		betBody(match, outcome, 50_000, testsupport.UUID(t)))
	expectError(t, status, respBody, http.StatusUnprocessableEntity, "insufficient_funds")

	if got := countQuery(t, env.pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != 40_000 {
		t.Fatalf("balance = %d after insufficient_funds, want 40000", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 0 {
		t.Fatalf("bets = %d after insufficient_funds, want 0", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); got != 0 {
		t.Fatalf("bet_stake rows = %d after insufficient_funds, want 0", got)
	}
}

func TestPlaceBetStakeOutOfRange(t *testing.T) {
	env := newTestEnv(t)
	match, outcome := pickHome(t, env)

	for _, stake := range []int64{9_999, 500_001} {
		t.Run(strconv.FormatInt(stake, 10), func(t *testing.T) {
			status, body, _ := request(t, env.app, http.MethodPost, "/api/bets",
				betBody(match, outcome, stake, testsupport.UUID(t)))
			expectError(t, status, body, http.StatusUnprocessableEntity, "stake_out_of_range")
		})
	}

	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 0 {
		t.Fatalf("bets = %d after stake_out_of_range, want 0", got)
	}
	if got := countQuery(t, env.pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != 2_500_000 {
		t.Fatalf("balance = %d after stake_out_of_range, want 2500000", got)
	}
}

func TestPlaceBetValidationErrors(t *testing.T) {
	env := newTestEnv(t)
	match, outcome := pickHome(t, env)

	valid := betBody(match, outcome, 50_000, testsupport.UUID(t))

	asMap := func(mutate func(map[string]any)) map[string]any {
		m := map[string]any{
			"matchId":            valid.MatchID,
			"outcome":            valid.Outcome,
			"stakeMinor":         valid.StakeMinor,
			"expectedPriceMilli": valid.ExpectedPriceMilli,
			"idempotencyKey":     valid.IdempotencyKey,
		}
		mutate(m)
		return m
	}

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "bad outcome",
			body: asMap(func(m map[string]any) { m["outcome"] = "over" }),
		},
		{
			name: "missing idempotency key",
			body: asMap(func(m map[string]any) { delete(m, "idempotencyKey") }),
		},
		{
			name: "short idempotency key",
			body: asMap(func(m map[string]any) { m["idempotencyKey"] = "short" }),
		},
		{
			name: "long idempotency key",
			body: asMap(func(m map[string]any) { m["idempotencyKey"] = strings.Repeat("k", 101) }),
		},
		{
			name: "negative stake",
			body: asMap(func(m map[string]any) { m["stakeMinor"] = -1 }),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			status, body, _ := request(t, env.app, http.MethodPost, "/api/bets", tt.body)
			expectError(t, status, body, http.StatusUnprocessableEntity, "validation_error")
		})
	}

	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 0 {
		t.Fatalf("bets = %d after validation errors, want 0", got)
	}
	if got := countQuery(t, env.pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != 2_500_000 {
		t.Fatalf("balance = %d after validation errors, want 2500000", got)
	}
}

func TestPlaceBetMarketNotFound(t *testing.T) {
	env := newTestEnv(t)
	match, outcome := pickHome(t, env)
	body := betBody(match, outcome, 50_000, testsupport.UUID(t))
	body.MatchID = "no-such-match"

	status, respBody, _ := request(t, env.app, http.MethodPost, "/api/bets", body)
	expectError(t, status, respBody, http.StatusNotFound, "market_not_found")

	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 0 {
		t.Fatalf("bets = %d after market_not_found, want 0", got)
	}
	if got := countQuery(t, env.pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != 2_500_000 {
		t.Fatalf("balance = %d after market_not_found, want 2500000", got)
	}
}

func TestBetsHistoryNewestFirstAndLimitClamped(t *testing.T) {
	env := newTestEnv(t)
	match, outcome := pickHome(t, env)

	const placed = 55
	ids := make([]string, 0, placed)
	for i := 0; i < placed; i++ {
		status, body, _ := request(t, env.app, http.MethodPost, "/api/bets",
			betBody(match, outcome, 10_000, testsupport.UUID(t)))
		if status != http.StatusCreated {
			t.Fatalf("bet %d status = %d, want 201\nbody: %s", i, status, body)
		}
		ids = append(ids, decode[betResponse](t, body).Bet.ID)
	}

	// A bet that belongs to a different player must never appear.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	foreignPlayer := testsupport.UUID(t)
	foreignBet := testsupport.UUID(t)
	if _, err := env.pool.Exec(ctx, `
		insert into players (id, name, currency, balance_minor)
		values ($1, 'Другой', 'RUB', 1000000)`, foreignPlayer); err != nil {
		t.Fatalf("insert foreign player: %v", err)
	}
	if _, err := env.pool.Exec(ctx, `
		insert into bets (id, player_id, idempotency_key, match_id, match_label,
		                  outcome, outcome_label, stake_minor, price_milli,
		                  potential_payout_minor, status, placed_at)
		values ($1, $2, $3, $4, $5, 'home', 'П1', 10000, $6, $7, 'pending', $8)`,
		foreignBet, foreignPlayer, testsupport.UUID(t), match.ID, "Другой — Матч",
		outcome.PriceMilli, 10*outcome.PriceMilli, fixedTime.Add(240*time.Hour)); err != nil {
		t.Fatalf("insert foreign bet: %v", err)
	}

	fetch := func(t *testing.T, query string) []betDTO {
		t.Helper()
		status, body, header := request(t, env.app, http.MethodGet, "/api/bets"+query, nil)
		if status != http.StatusOK {
			t.Fatalf("GET /api/bets%s status = %d, want 200\nbody: %s", query, status, body)
		}
		requireJSONContentType(t, header)
		top := object(t, body)
		requireKeys(t, top, "bets")
		resp := decode[struct {
			Bets []betDTO `json:"bets"`
		}](t, body)
		return resp.Bets
	}

	max := fetch(t, "?limit=999")
	if len(max) != 50 {
		t.Fatalf("limit=999 returned %d bets, want clamped 50", len(max))
	}
	if max[0].ID != ids[len(ids)-1] {
		t.Fatalf("history[0] = %q, want newest bet %q", max[0].ID, ids[len(ids)-1])
	}
	for _, bet := range max {
		if bet.ID == foreignBet {
			t.Fatal("history contains another player's bet")
		}
	}
	for i := 1; i < len(max); i++ {
		prev, err := time.Parse(time.RFC3339, max[i-1].PlacedAt)
		if err != nil {
			t.Fatalf("parse placedAt %q: %v", max[i-1].PlacedAt, err)
		}
		cur, err := time.Parse(time.RFC3339, max[i].PlacedAt)
		if err != nil {
			t.Fatalf("parse placedAt %q: %v", max[i].PlacedAt, err)
		}
		if cur.After(prev) {
			t.Fatalf("history is not newest first at index %d: %s after %s", i, max[i].PlacedAt, max[i-1].PlacedAt)
		}
	}

	def := fetch(t, "?limit=0")
	if len(def) != 20 {
		t.Fatalf("limit=0 returned %d bets, want default 20", len(def))
	}
	if def[0].ID != ids[len(ids)-1] {
		t.Fatalf("default history[0] = %q, want newest bet %q", def[0].ID, ids[len(ids)-1])
	}
	for _, bet := range def {
		if bet.ID == foreignBet {
			t.Fatal("default history contains another player's bet")
		}
	}
}

func TestHealthz(t *testing.T) {
	env := newTestEnv(t)
	status, body, header := request(t, env.app, http.MethodGet, "/api/healthz", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /api/healthz status = %d, want 200\nbody: %s", status, body)
	}
	requireJSONContentType(t, header)
	resp := decode[struct {
		Status string `json:"status"`
	}](t, body)
	if resp.Status != "ok" {
		t.Fatalf("healthz status = %q, want ok", resp.Status)
	}
}

func TestPlaceBetReplayAfterPriceChange(t *testing.T) {
	env := newTestEnv(t)
	match, outcome := pickHome(t, env)
	body := betBody(match, outcome, 50_000, testsupport.UUID(t))

	status, firstBody, _ := request(t, env.app, http.MethodPost, "/api/bets", body)
	if status != http.StatusCreated {
		t.Fatalf("first POST status = %d, want 201\nbody: %s", status, firstBody)
	}
	first := decode[betResponse](t, firstBody)

	// Move the line until this outcome's price differs from the accepted one.
	changed := false
	for i := 0; i < 100; i++ {
		snap := env.engine.Update()
		for _, current := range snap.Matches[0].Outcomes {
			if current.Key == outcome.Key && current.PriceMilli != outcome.PriceMilli {
				changed = true
			}
		}
		if changed {
			break
		}
	}
	if !changed {
		t.Fatal("engine never moved the price; cannot test replay after a price change")
	}

	// A retry after a lost response still carries the old price. It must
	// replay the original bet, not fail with 409 and tempt the client to place
	// a second bet at the new price.
	status, secondBody, _ := request(t, env.app, http.MethodPost, "/api/bets", body)
	if status != http.StatusOK {
		t.Fatalf("replay after price change status = %d, want 200\nbody: %s", status, secondBody)
	}
	second := decode[betResponse](t, secondBody)
	if !second.Replayed {
		t.Fatal("replay after price change reported replayed=false")
	}
	if second.Bet.ID != first.Bet.ID {
		t.Fatalf("replay bet id = %q, want original %q", second.Bet.ID, first.Bet.ID)
	}
	if second.BalanceMinor != 2_450_000 {
		t.Fatalf("replay balance = %d, want 2450000", second.BalanceMinor)
	}
	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 1 {
		t.Fatalf("bets = %d after replay, want 1", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from ledger_entries where kind = 'bet_stake'`); got != 1 {
		t.Fatalf("bet_stake ledger rows = %d after replay, want 1", got)
	}
}

func TestSettleDemoBetWin(t *testing.T) {
	env := newTestEnv(t)
	*env.settleRoll = 0

	match, outcome := pickHome(t, env)
	status, body, _ := request(t, env.app, http.MethodPost, "/api/bets",
		betBody(match, outcome, 50_000, testsupport.UUID(t)))
	if status != http.StatusCreated {
		t.Fatalf("place bet status = %d, want 201\nbody: %s", status, body)
	}
	placed := decode[betResponse](t, body)

	status, body, _ = request(t, env.app, http.MethodPost, "/api/bets/"+placed.Bet.ID+"/settle-demo", nil)
	if status != http.StatusOK {
		t.Fatalf("settle status = %d, want 200\nbody: %s", status, body)
	}
	settled := decode[settleResponse](t, body)
	if !settled.Won {
		t.Fatal("settle reported won=false, want true")
	}
	if settled.Bet.Status != "won" {
		t.Fatalf("settled status = %q, want won", settled.Bet.Status)
	}
	wantBalance := int64(2_500_000-50_000) + placed.Bet.PotentialPayoutMinor
	if settled.BalanceMinor != wantBalance {
		t.Fatalf("settled balance = %d, want %d", settled.BalanceMinor, wantBalance)
	}
	if got := countQuery(t, env.pool,
		`select count(*) from ledger_entries where kind = 'bet_win' and bet_id = $1 and amount_minor = $2 and balance_after_minor = $3`,
		placed.Bet.ID, placed.Bet.PotentialPayoutMinor, wantBalance); got != 1 {
		t.Fatalf("bet_win ledger rows = %d, want 1", got)
	}

	// A second settlement must not credit the payout twice.
	status, body, _ = request(t, env.app, http.MethodPost, "/api/bets/"+placed.Bet.ID+"/settle-demo", nil)
	expectError(t, status, body, http.StatusConflict, "bet_not_pending")
	if got := countQuery(t, env.pool, `select balance_minor from players where id = $1`, store.DemoPlayerID); got != wantBalance {
		t.Fatalf("balance after double settle = %d, want %d", got, wantBalance)
	}
}

func TestSettleDemoBetLoss(t *testing.T) {
	env := newTestEnv(t)
	*env.settleRoll = 9_999

	match, outcome := pickHome(t, env)
	status, body, _ := request(t, env.app, http.MethodPost, "/api/bets",
		betBody(match, outcome, 50_000, testsupport.UUID(t)))
	if status != http.StatusCreated {
		t.Fatalf("place bet status = %d, want 201\nbody: %s", status, body)
	}
	placed := decode[betResponse](t, body)

	status, body, _ = request(t, env.app, http.MethodPost, "/api/bets/"+placed.Bet.ID+"/settle-demo", nil)
	if status != http.StatusOK {
		t.Fatalf("settle status = %d, want 200\nbody: %s", status, body)
	}
	settled := decode[settleResponse](t, body)
	if settled.Won {
		t.Fatal("settle reported won=true, want false")
	}
	if settled.Bet.Status != "lost" {
		t.Fatalf("settled status = %q, want lost", settled.Bet.Status)
	}
	if settled.BalanceMinor != 2_450_000 {
		t.Fatalf("settled balance = %d, want 2450000", settled.BalanceMinor)
	}
	if got := countQuery(t, env.pool, `select count(*) from ledger_entries where kind = 'bet_win' and bet_id = $1`, placed.Bet.ID); got != 0 {
		t.Fatalf("bet_win ledger rows after loss = %d, want 0", got)
	}
}

func TestSettleDemoUnknownBet(t *testing.T) {
	env := newTestEnv(t)

	status, body, _ := request(t, env.app, http.MethodPost, "/api/bets/"+testsupport.UUID(t)+"/settle-demo", nil)
	expectError(t, status, body, http.StatusNotFound, "bet_not_found")

	status, body, _ = request(t, env.app, http.MethodPost, "/api/bets/not-a-uuid/settle-demo", nil)
	expectError(t, status, body, http.StatusUnprocessableEntity, "validation_error")
}

func TestBetAfterInsufficientFundsStillWorks(t *testing.T) {
	env := newTestEnv(t)
	testsupport.SetBalance(t, env.pool, 40_000)
	match, outcome := pickHome(t, env)

	status, body, _ := request(t, env.app, http.MethodPost, "/api/bets",
		betBody(match, outcome, 50_000, testsupport.UUID(t)))
	expectError(t, status, body, http.StatusUnprocessableEntity, "insufficient_funds")

	status, body, _ = request(t, env.app, http.MethodPost, "/api/bets",
		betBody(match, outcome, 10_000, testsupport.UUID(t)))
	if status != http.StatusCreated {
		t.Fatalf("valid bet after insufficient_funds status = %d, want 201\nbody: %s", status, body)
	}
	resp := decode[betResponse](t, body)
	if resp.BalanceMinor != 30_000 {
		t.Fatalf("balance after valid bet = %d, want 30000", resp.BalanceMinor)
	}
	if got := countQuery(t, env.pool, `select count(*) from bets`); got != 1 {
		t.Fatalf("bets = %d, want 1", got)
	}
	if got := countQuery(t, env.pool, `select count(*) from ledger_entries where kind = 'bet_stake' and amount_minor = -10000`); got != 1 {
		t.Fatalf("bet_stake rows = %d, want 1", got)
	}
}

// --- SSE ------------------------------------------------------------------

type sseEvent struct {
	Name string
	ID   string
	Data string
}

func TestSSESnapshotLineAndBalance(t *testing.T) {
	env := newTestEnv(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serveDone := make(chan struct{})
	go func() {
		defer close(serveDone)
		_ = env.app.Listener(ln)
	}()
	t.Cleanup(func() {
		_ = env.app.ShutdownWithTimeout(2 * time.Second)
		select {
		case <-serveDone:
		case <-time.After(time.Second):
		}
	})

	baseURL := "http://" + ln.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/stream", nil)
	if err != nil {
		t.Fatalf("new SSE request: %v", err)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("connect SSE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("SSE Content-Type = %q, want text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
		t.Fatalf("SSE Cache-Control = %q, want no-cache", cc)
	}
	if accel := resp.Header.Get("X-Accel-Buffering"); accel != "no" {
		t.Fatalf("SSE X-Accel-Buffering = %q, want no", accel)
	}

	events := make(chan sseEvent, 64)
	streamErrs := make(chan error, 1)
	readerDone := make(chan struct{})
	go func() {
		streamErrs <- readSSE(resp.Body, events, readerDone)
	}()
	defer close(readerDone)

	// 1. A fresh connection gets a full snapshot immediately.
	snapshotEvent := waitEvent(t, events, streamErrs, "snapshot", time.Second)
	snapshot := decode[line.Snapshot](t, []byte(snapshotEvent.Data))
	if len(snapshot.Matches) == 0 {
		t.Fatal("SSE snapshot has no matches")
	}
	for _, match := range snapshot.Matches {
		if len(match.Outcomes) != 3 {
			t.Fatalf("SSE snapshot match %q has %d outcomes", match.ID, len(match.Outcomes))
		}
	}

	// 2. Engine updates arrive as complete line events with an increasing seq.
	// Repeated updates tolerate the tiny race between the HTTP response and the
	// handler registering its subscription.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var lineEvent sseEvent
	var lineSnap line.Snapshot
	lineDeadline := time.After(time.Second)
waitLine:
	for {
		select {
		case ev := <-events:
			if ev.Name == "line" {
				lineEvent = ev
				lineSnap = decode[line.Snapshot](t, []byte(ev.Data))
				break waitLine
			}
		case <-ticker.C:
			env.engine.Update()
		case err := <-streamErrs:
			t.Fatalf("SSE stream ended while waiting for line event: %v", err)
		case <-lineDeadline:
			t.Fatal("no SSE line event arrived within 1s while calling engine.Update")
		}
	}
	if lineSnap.Seq <= snapshot.Seq {
		t.Fatalf("line event seq = %d, want > snapshot seq %d", lineSnap.Seq, snapshot.Seq)
	}
	if len(lineSnap.Matches) == 0 {
		t.Fatal("SSE line event has no matches")
	}
	if lineEvent.ID != strconv.FormatInt(lineSnap.Seq, 10) {
		t.Fatalf("line event id = %q, want seq %d", lineEvent.ID, lineSnap.Seq)
	}

	// 3. Placing a bet broadcasts the new balance on the same stream.
	current := env.engine.Snapshot()
	match := current.Matches[0]
	outcome := match.Outcomes[0]
	payload, err := json.Marshal(betBody(match, outcome, 50_000, testsupport.UUID(t)))
	if err != nil {
		t.Fatalf("marshal bet: %v", err)
	}
	betClient := &http.Client{Timeout: 2 * time.Second}
	betResp, err := betClient.Post(baseURL+"/api/bets", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST bet over HTTP: %v", err)
	}
	betBodyBytes, err := io.ReadAll(betResp.Body)
	betResp.Body.Close()
	if err != nil {
		t.Fatalf("read bet response: %v", err)
	}
	if betResp.StatusCode != http.StatusCreated {
		t.Fatalf("bet status = %d, want 201\nbody: %s", betResp.StatusCode, betBodyBytes)
	}

	balanceEvent := waitEvent(t, events, streamErrs, "balance", time.Second)
	balance := decode[struct {
		BalanceMinor int64  `json:"balanceMinor"`
		Currency     string `json:"currency"`
	}](t, []byte(balanceEvent.Data))
	if balance.BalanceMinor != 2_450_000 {
		t.Fatalf("SSE balance = %d, want 2450000", balance.BalanceMinor)
	}
	if balance.Currency != "RUB" {
		t.Fatalf("SSE balance currency = %q, want RUB", balance.Currency)
	}
}

func waitEvent(t *testing.T, events <-chan sseEvent, errs <-chan error, name string, timeout time.Duration) sseEvent {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-events:
			if ev.Name == name {
				return ev
			}
		case err := <-errs:
			t.Fatalf("SSE stream ended while waiting for %q: %v", name, err)
		case <-deadline:
			t.Fatalf("timed out waiting for SSE event %q", name)
		}
	}
}

func readSSE(r io.Reader, events chan<- sseEvent, done <-chan struct{}) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var (
		name string
		id   string
		data []string
	)
	emit := func() error {
		if name == "" && len(data) == 0 {
			return nil
		}
		ev := sseEvent{Name: name, ID: id, Data: strings.Join(data, "\n")}
		select {
		case events <- ev:
		case <-done:
			return io.EOF
		}
		name, id, data = "", "", nil
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if err := emit(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue // heartbeat comment
		}
		field, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			name = value
		case "id":
			id = value
		case "data":
			data = append(data, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}
