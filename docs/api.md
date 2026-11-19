# API reference

The backend exposes a small JSON API under `/api`. Types and rules are defined in
[`engineering.md`](./engineering.md); this file is the cheat sheet.

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/session` | Demo player, balance, currency, stake limits |
| GET | `/api/line` | Current line snapshot (all matches, both source prices) |
| GET | `/api/bets?limit=20` | Player's bets, newest first |
| POST | `/api/bets` | Place a bet (idempotent, price-checked) |
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

## Errors

```json
{ "error": { "code": "odds_changed", "message": "Коэффициент изменился",
  "details": { "expectedPriceMilli": 1910, "priceMilli": 1920 } } }
```
