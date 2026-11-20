# API reference

The backend exposes a small JSON API under `/api`. Types and rules are defined in
[`engineering.md`](./engineering.md); this file is the cheat sheet.

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/session` | Demo player, balance, currency, stake limits |
| GET | `/api/line` | Current line snapshot (all matches, both source prices) |
| GET | `/api/bets?limit=20` | Player's bets, newest first |
| POST | `/api/bets` | Place a bet (idempotent, price-checked) |
| POST | `/api/bets/{id}/settle-demo` | Demo-only: resolve a pending bet as won/lost |
| GET | `/api/stream` | SSE: `snapshot`, `line`, `balance` events |
| GET | `/api/healthz` | Liveness + database ping |

All amounts are integer minor units (kopecks). All prices are integer thousandths of decimal odds
(`1950` = `1.95`).

## Place a bet

```bash
curl -s http://localhost:8080/api/bets \
  -H 'content-type: application/json' \
  -d '{
    "matchId": "epl-ars-che",
    "outcome": "home",
    "stakeMinor": 50000,
    "expectedPriceMilli": 1910,
    "idempotencyKey": "6f1a2f52-7a8a-4d8f-9d3e-7a1b2c3d4e5f"
  }'
```

- `201` — bet stored, balance debited, payout frozen.
- `200` — same idempotency key replayed; no second debit.
- `409 odds_changed` — `details.priceMilli` is the current price; re-confirm and retry.
- `422 insufficient_funds` / `stake_out_of_range` / `validation_error`.
- `404 market_not_found` — the match or outcome is not in the current line.

## Resolve a bet in the demo

```bash
curl -s -X POST http://localhost:8080/api/bets/<bet-id>/settle-demo
```

This is a manual demo simulation, not automatic settlement after a match. The server draws a result
using the accepted price as the win probability (`1 / price`). A win credits
`potentialPayoutMinor` to the balance and writes a `bet_win` ledger row; a loss only changes the
status.

```json
{ "bet": { "id": "…", "status": "won", "potentialPayoutMinor": 95500, "…": "…" },
  "balanceMinor": 2545500, "currency": "RUB", "won": true }
```

- `200` — bet resolved; `won` tells the UI which toast to show.
- `409 bet_not_pending` — the bet is already won, lost, or void; no second credit.
- `404 bet_not_found` — no such bet for the demo player.
- `422 validation_error` — the id is not a UUID.

## Errors

```json
{ "error": { "code": "odds_changed", "message": "Коэффициент изменился",
  "details": { "expectedPriceMilli": 1910, "priceMilli": 1920 } } }
```
