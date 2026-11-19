# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Go (Fiber) + PostgreSQL API, Vue 3 + TypeScript + Vite + Tailwind + shadcn-vue web client,
everything started with `docker compose up`. Fixed by the brief; not an open decision.

## Users

One recreational bettor, alone, looking at a live football line on a laptop or phone. They are
not a professional: they scan a handful of matches, compare prices, pick an outcome and stake a
small amount from a demo balance. The product is also read by reviewers evaluating engineering
judgment, so the reasoning behind the price must be legible, not hidden.

## Product Purpose

A single-player sportsbook slice: six to eight football matches with a 1X2 line that moves on its
own, a balance, and a bet slip. Success is a bettor who can see the price change, understand where
it came from, and place a bet without ever being able to spend money they do not have or be
charged twice.

## Positioning

The line shows its work. Every price exposes the two source prices it was derived from and the
house margin, so the product is a demonstrator of a priced market rather than an opaque casino.
No registration, no settlement, no admin: the line and the bet are the whole product.

## Operating Context

Used in short sessions while matches approach kickoff. Odds refresh every few seconds and the
page must stay honest when the connection drops: stale prices are marked stale and betting is
blocked rather than accepting a price the server no longer honors. Bets stay pending; the brief
explicitly excludes settling them after the match.

## Capabilities and Constraints

- 1X2 (П1 / Х / П2) on 6–8 simulated football matches across several leagues.
- Two independent simulated price sources per outcome; the house line is their de-vigged,
  weighted consensus plus a fixed margin. Sources drift on a mean-reverting random walk.
- Real-time delivery over Server-Sent Events (SSE), one full line snapshot per tick.
- One seeded demo player with a balance in integer kopecks. Stake limits and a 1.01 minimum price.
- A bet is accepted at the price the client saw or rejected with the new price (409), never
  silently re-priced. Requests are idempotent by client key; balance is debited in the same
  transaction that records the bet; no negative balance is possible (DB CHECK + row lock).
- Money is never a float: BIGINT minor units on the server, integer `priceMilli` for odds,
  integer arithmetic for payout, `Intl.NumberFormat` for display.
- Russian UI copy. No auth, registration, settlement, cash-out, or admin.
- Must run locally with a single `docker compose up`, no external services or keys.

## Brand Commitments

Working name: **Овертайм** (Overtime). Dark, near-black surfaces with a single hot-orange accent;
the feel of a night-match broadcast scoreboard rather than a generic SaaS dashboard. Micro
interactions use ordinary CSS/Vue transitions; larger ambient effects are limited and purposeful.

## Evidence on Hand

All prices, matches, teams and results are simulated. There is no real sportsbook feed, no real
money, no real user data, and no claim of betting accuracy. The README must say so plainly.

## Product Principles

1. **The price is the product.** Show the consensus, the sources and the margin; never fake a
   price the backend would not honor.
2. **Money fails closed.** If the balance, the price or the connection is uncertain, refuse the
   bet and explain why in the user's language.
3. **One screen, one decision.** Scan the line, pick an outcome, stake an amount. Everything else
   is secondary.
4. **Calm until it matters.** Motion marks a price change or a confirmed action, not decoration.
5. **Honest demo.** Label simulated data and demo balance instead of dressing it as live betting.
