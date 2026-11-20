# Engineering contract

Pinned decisions for the sportsbook. The frontend and backend are built against this document;
do not change a signature or a price formula silently. If something here is wrong, fix the
document first, then the code.

## 1. Money

All money is an integer amount of minor currency units (kopecks). Currency is `RUB`.
`2 500 000` minor units = 25 000,00 ₽. Floats never touch a balance, a stake, a payout or SQL.

Rules:

- Balance is stored on `players.balance_minor BIGINT NOT NULL CHECK (balance_minor >= 0)`.
- Every balance change writes an immutable `ledger_entries` row in the same transaction, with
  `balance_after_minor` and a signed `amount_minor`.
- A bet is recorded and the stake debited in one database transaction, under `SELECT … FOR UPDATE`
  on the player row.
- Idempotency: `bets UNIQUE (player_id, idempotency_key)`. A replayed request returns the original
  bet and balance with `replayed: true`, with no second debit. The replay lookup runs **before** the
  price, stake-limit and balance checks: a client that lost the response retries with the old price
  and must get its bet back, not a 409 that invites a second bet.
- Potential payout = `stake_minor * price_milli / 1000`, integer division, floored. Never rounded up.
  The remainder is the house's; it is not a rounding gift to the player.
- Stake limits: minimum `10_000` minor (100 ₽), maximum `500_000` minor (5 000 ₽), balance
  `2_500_000` minor (25 000 ₽).
- A bet is rejected with `insufficient_funds` if `stake > balance` (the balance check is inside
  the locked transaction, not only in Go).

## 2. Odds

Odds are decimal (European) prices stored as integer thousandths: `priceMilli = 1950` means
`1.95`. Valid prices are `1050 … 20000` (1.05 … 20.00), always a multiple of 10 (two decimals).

### Line pipeline

For each match and outcome:

1. Each source (`pulse`, `sharp`) produces a simulated decimal price. Source overrounds differ
   (about 3% and about 6%), so the sources are not directly comparable.
2. De-vig each source independently: `p_i = (1 / price_i) / Σ_j (1 / price_j)`. Fair probabilities
   per source sum to 1.
3. Blend the two source vectors with weights `0.6` (sharp) and `0.4` (pulse). Weights sum to 1.
4. Apply the house margin: `adj_i = p_i * (1 + margin)`, with `margin = 0.05` (500 bps).
   `Σ adj_i = 1.05` exactly before rounding.
5. Convert back: `price_i = 1 / adj_i`, floor to two decimals (`priceMilli = floor(1000 / adj_i / 10) * 10`),
   then clamp to `[1050, 20000]`.

Flooring prices never reduces the achieved margin: a smaller price means a larger implied
probability, so the implied total stays at or above `1 + margin`. The achieved margin is
`Σ (1000/priceMilli) - 1` and is exposed as `marginBps` per snapshot (the target is `500`); the
customer UI intentionally does not surface it.

The intra-source price is generated from a mean-reverting fair probability per outcome with small
per-source noise, so the consensus drifts by a few hundredths between ticks without jumping
wildly. Every tick recomputes both sources from scratch; nothing is random per request.

Refresh cadence: one tick every 3000 ms. `NewEngine` computes the initial snapshot with `seq = 0`;
every `Update` returns the next sequence (`1, 2, …`). A subscriber channel is buffered; a slow
subscriber must never block the ticker (drop the oldest pending snapshot instead).

## 3. Database

Single migration file, applied at startup, idempotent.

```sql
create table if not exists players (
  id            uuid primary key,
  name          text not null,
  currency      text not null default 'RUB',
  balance_minor bigint not null check (balance_minor >= 0),
  created_at    timestamptz not null default now()
);

create table if not exists bets (
  id                     uuid primary key,
  player_id              uuid not null references players(id),
  idempotency_key        text not null,
  match_id               text not null,
  match_label            text not null,
  outcome                text not null check (outcome in ('home','draw','away')),
  outcome_label          text not null,
  stake_minor            bigint not null check (stake_minor > 0),
  price_milli            bigint not null check (price_milli between 1050 and 20000),
  potential_payout_minor bigint not null check (potential_payout_minor >= stake_minor),
  status                 text not null default 'pending'
                           check (status in ('pending','won','lost','void')),
  placed_at              timestamptz not null default now(),
  unique (player_id, idempotency_key)
);

create index if not exists bets_player_placed_idx on bets (player_id, placed_at desc);

create table if not exists ledger_entries (
  id                  bigserial primary key,
  player_id           uuid not null references players(id),
  bet_id              uuid references bets(id),
  kind                text not null check (kind in ('deposit','bet_stake','bet_win','bet_refund')),
  amount_minor        bigint not null,
  balance_after_minor bigint not null check (balance_after_minor >= 0),
  created_at          timestamptz not null default now()
);
```

Seed (in the same migration, `on conflict do nothing`): player
`11111111-1111-4111-8111-111111111111`, name `Игрок`, balance `2500000`, plus an initial
`deposit` ledger row of `2500000`. A subsequent reference to the same player must not reset the
balance.

## 4. HTTP API

Base path `/api`. JSON in, JSON out, UTF-8. All errors share one envelope:

```json
{ "error": { "code": "odds_changed", "message": "Коэффициент изменился", "details": { "priceMilli": 1920 } } }
```

`details` is optional. Status mapping:

| code | status |
| --- | --- |
| `validation_error` | 422 |
| `stake_out_of_range` | 422 |
| `insufficient_funds` | 422 |
| `odds_changed` | 409 |
| `market_not_found` | 404 |
| `not_found` | 404 |
| `internal` | 500 |

There is no auth: the server always uses the single demo player.

### GET /api/session

```json
{
  "player": { "id": "1111…", "name": "Игрок", "balanceMinor": 2500000, "currency": "RUB" },
  "limits": { "minStakeMinor": 10000, "maxStakeMinor": 500000 },
  "serverTime": "2026-11-20T18:00:00Z"
}
```

### GET /api/line

```json
{
  "seq": 41,
  "updatedAt": "2026-11-20T18:00:03Z",
  "marginBps": 500,
  "matches": [
    {
      "id": "epl-ars-che",
      "sport": "Футбол",
      "league": "АПЛ",
      "startsAt": "2026-11-20T20:00:00Z",
      "homeTeam": "Арсенал",
      "awayTeam": "Челси",
      "outcomes": [
        {
          "key": "home",
          "label": "П1",
          "priceMilli": 1910,
          "trend": "down",
          "sources": [
            { "id": "pulse", "label": "PulseBet", "priceMilli": 1950 },
            { "id": "sharp", "label": "SharpLine", "priceMilli": 1930 }
          ]
        },
        { "key": "draw", "label": "Х", "priceMilli": 3550, "trend": "flat", "sources": [] },
        { "key": "away", "label": "П2", "priceMilli": 4050, "trend": "up", "sources": [] }
      ]
    }
  ]
}
```

`trend` is the server-side comparison with the previous tick: `up`, `down` or `flat` (the first
tick is all `flat`). Outcome order is always `home`, `draw`, `away`.

### POST /api/bets

```json
{
  "matchId": "epl-ars-che",
  "outcome": "home",
  "stakeMinor": 50000,
  "expectedPriceMilli": 1910,
  "idempotencyKey": "3f3c1f9e-6a3e-4c31-9c0e-6c2b0b9a1b77"
}
```

Success `201`:

```json
{
  "bet": {
    "id": "…", "matchId": "epl-ars-che", "matchLabel": "Арсенал — Челси",
    "outcome": "home", "outcomeLabel": "П1",
    "stakeMinor": 50000, "priceMilli": 1910, "potentialPayoutMinor": 95500,
    "status": "pending", "placedAt": "2026-11-20T18:00:04Z"
  },
  "balanceMinor": 2450000,
  "currency": "RUB",
  "replayed": false
}
```

A replay of the same `idempotencyKey` returns `200` with the original bet, the current balance and
`"replayed": true`.

`409 odds_changed` details: `{ "expectedPriceMilli": 1910, "priceMilli": 1920 }` where `priceMilli`
is the current price. The client must show the new price and let the user confirm; the retry sends
the fresh key (a new bet attempt) and the new expected price.

Validation: `stakeMinor` is a positive integer; `priceMilli` guess is an integer; `idempotencyKey`
is 8–100 characters (UUID recommended); `outcome` is one of `home|draw|away`; the match must exist
in the current line. Unknown fields are rejected with `validation_error` (Fiber `BodyParser` with a
strict check is fine; at minimum validate the listed fields).

### GET /api/bets?limit=20

```json
{ "bets": [ { "id": "…", "matchId": "…", "matchLabel": "…", "outcome": "home",
  "outcomeLabel": "П1", "stakeMinor": 50000, "priceMilli": 1910,
  "potentialPayoutMinor": 95500, "status": "pending", "placedAt": "…" } ] }
```

Newest first, `limit` clamped to 1–50, default 20.

### GET /api/stream (SSE)

Headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`,
`X-Accel-Buffering: no`.

On connect: `event: snapshot` with the current `GET /api/line` payload as one-line JSON, then:

- `event: line` — the same full snapshot on every tick, `id: <seq>`.
- `event: balance` — `{"balanceMinor": 2450000, "currency": "RUB"}` after any bet by the player.
- A comment heartbeat `: ping` every 20 s.

No replay buffer: clients reconnect (EventSource does it automatically) and receive a fresh
`snapshot`. Every `line` event is a complete state, so dropping one is harmless.

### GET /api/healthz

`200 {"status":"ok"}` after a database ping.

## 5. Go package contract (backend)

Module `sportsbook/backend`, Go 1.22. Dependencies: Fiber v2, pgx v5, `google/uuid`.

```go
// internal/money
func Payout(stakeMinor, priceMilli int64) (int64, error) // floor; ErrOverflow on overflow
func ValidateStake(stake, min, max int64) error            // ErrStakeOutOfRange
var ErrOverflow, ErrStakeOutOfRange error

// internal/line
type SourceQuote struct { ID, Label string; PriceMilli int64 }
type Outcome struct { Key string; Label string; PriceMilli int64; Trend string; Sources []SourceQuote }
type Match struct { ID, Sport, League, HomeTeam, AwayTeam string; StartsAt time.Time; Outcomes []Outcome }
type Snapshot struct { Seq int64; UpdatedAt time.Time; MarginBps int; Matches []Match }

func Devig(pricesMilli []int64) []float64                       // normalized, sums to 1
func WithMargin(fair []float64, marginBps int) []int64          // prices, floored to 10, clamped
func Blend(sources [][]float64, weights []float64) []float64    // weighted mean, normalized

type Config struct { Seed int64; MarginBps int; Tick time.Duration; Now func() time.Time }
func NewEngine(cfg Config) *Engine
func (e *Engine) Snapshot() Snapshot
func (e *Engine) Update() Snapshot                              // one deterministic tick
func (e *Engine) Run(ctx context.Context)                       // ticker loop, broadcasts
func (e *Engine) Subscribe() (<-chan Snapshot, func())          // buffered channel; cancel func

// internal/store
const DemoPlayerID = "11111111-1111-4111-8111-111111111111"
type Player struct { ID, Name, Currency string; BalanceMinor int64 }
type Bet struct { ID, MatchID, MatchLabel, Outcome, OutcomeLabel string; StakeMinor, PriceMilli, PotentialPayoutMinor int64; Status string; PlacedAt time.Time }
type PlaceBetInput struct { ID, PlayerID, IdempotencyKey, MatchID, MatchLabel, Outcome, OutcomeLabel string; StakeMinor, PriceMilli, PotentialPayoutMinor int64; Now time.Time }
func Migrate(ctx context.Context, pool *pgxpool.Pool) error
func New(pool *pgxpool.Pool) *Store
func (s *Store) Player(ctx context.Context, id string) (Player, error)
func (s *Store) Bets(ctx context.Context, playerID string, limit int) ([]Bet, error)
func (s *Store) PlaceBet(ctx context.Context, in PlaceBetInput) (Bet, int64, bool, error) // bet, balance, replayed, err
var ErrInsufficientFunds, ErrNotFound error

// internal/betting
type Deps struct { Store *store.Store; Engine *line.Engine; PlayerID string; MinStakeMinor, MaxStakeMinor int64; Now func() time.Time }
func New(d Deps) *Service
type PlaceInput struct { MatchID, Outcome string; StakeMinor, ExpectedPriceMilli int64; IdempotencyKey string }
type PlaceResult struct { Bet store.Bet; BalanceMinor int64; Replayed bool }
func (s *Service) Place(ctx context.Context, in PlaceInput) (PlaceResult, error)
func (s *Service) Session(ctx context.Context) (store.Player, error)
func (s *Service) History(ctx context.Context, limit int) ([]store.Bet, error)
// SubscribeBalance fans out the player's balance after every accepted bet
// (including idempotent replays). Slow subscribers are dropped, never blocked.
type BalanceEvent struct { BalanceMinor int64; Currency string }
func (s *Service) SubscribeBalance() (<-chan BalanceEvent, func())
var ErrMarketNotFound, ErrOddsChanged error
type OddsChangedError struct { Expected, Current int64 }
var ErrInsufficientFunds, ErrStakeOutOfRange, ErrInvalidRequest error

// internal/app
type Limits struct { MinStakeMinor, MaxStakeMinor int64 }
type Deps struct { Service *betting.Service; Engine *line.Engine; Limits Limits; Now func() time.Time }
func New(d Deps) *fiber.App // routes under /api, JSON errors, SSE
```

Env for `cmd/server`: `DATABASE_URL`, `PORT` (default `8080`), `LINE_TICK_MS` (default `3000`),
`MARGIN_BPS` (default `500`), `LINE_SEED` (default `7`).

## 6. Frontend contract

- API base: same origin `/api` (Vite dev proxy to `http://localhost:8080`; nginx in Docker).
- `src/api/` is the only place that knows URLs, JSON shapes, status codes, and EventSource.
  Components and composables call the API module, never `fetch` directly.
- Domain state lives in composables (`src/composables/`): session/balance, line + SSE status,
  bet slip, bets history. Pinia is not needed.
- Money and odds format helpers in `src/lib/format.ts` (integer → `ru-RU` currency / decimal odds).
- UI copy is Russian; code identifiers are English.
- shadcn-vue components live in `src/components/ui/`; no custom re-implementation of a button,
  dialog or input. Toasts come from `@yusufceng1z/ding-vue`.
- Ported Magic UI effects live in `src/components/magic/` with a comment naming the source.
  Micro-interactions use Vue `<Transition>`/CSS transitions, not a new animation library.
- The client must handle: loading skeletons, empty bet history, network failure on load (retry
  action), SSE disconnect (status pill, betting disabled once the line is stale), 409 odds change
  (confirm dialog with the new price), 422 insufficient funds (toast + balance refresh),
  in-flight bet (double submit blocked, idempotency key reused on timeout).

## 7. Definition of done

- `docker compose up` serves the UI on `http://localhost:8080` with no manual steps.
- `go test ./...` green (integration tests skip without `TEST_DATABASE_URL`).
- `go vet ./...`, `npm run build`, `vue-tsc --noEmit` clean.
- Concurrency test proves no negative balance and no double debit under parallel bets.
- A short README explains the line, money safety, AI/human split, and what was deliberately cut.
