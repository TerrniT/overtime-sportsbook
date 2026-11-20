/**
 * Wire types. These mirror docs/api.md and docs/engineering.md exactly.
 * Amounts are integer kopecks; prices are integer thousandths of decimal odds.
 */

export type OutcomeKey = 'home' | 'draw' | 'away'
export type Trend = 'up' | 'down' | 'flat'
export type BetStatus = 'pending' | 'won' | 'lost' | 'void'
export type Currency = 'RUB'

export interface SourceQuote {
  id: string
  label: string
  priceMilli: number
}

export interface Outcome {
  key: OutcomeKey
  label: string
  priceMilli: number
  trend: Trend
  sources: SourceQuote[]
}

export interface Match {
  id: string
  sport: string
  league: string
  startsAt: string
  homeTeam: string
  awayTeam: string
  outcomes: Outcome[]
}

export interface LineSnapshot {
  seq: number
  updatedAt: string
  marginBps: number
  matches: Match[]
}

export interface Player {
  id: string
  name: string
  balanceMinor: number
  currency: Currency
}

export interface Limits {
  minStakeMinor: number
  maxStakeMinor: number
}

export interface SessionResponse {
  player: Player
  limits: Limits
  serverTime: string
}

export interface Bet {
  id: string
  matchId: string
  matchLabel: string
  outcome: OutcomeKey
  outcomeLabel: string
  stakeMinor: number
  priceMilli: number
  potentialPayoutMinor: number
  status: BetStatus
  placedAt: string
}

export interface PlaceBetRequest {
  matchId: string
  outcome: OutcomeKey
  stakeMinor: number
  expectedPriceMilli: number
  idempotencyKey: string
}

export interface PlaceBetResponse {
  bet: Bet
  balanceMinor: number
  currency: Currency
  replayed: boolean
}

export interface BetsResponse {
  bets: Bet[]
}

export interface SettleDemoResponse {
  bet: Bet
  balanceMinor: number
  currency: Currency
  won: boolean
}

export interface BalanceEvent {
  balanceMinor: number
  currency: Currency
}

export interface OddsChangedDetails {
  expectedPriceMilli?: number
  priceMilli?: number
  [key: string]: unknown
}

export type ErrorCode =
  | 'validation_error'
  | 'stake_out_of_range'
  | 'insufficient_funds'
  | 'odds_changed'
  | 'market_not_found'
  | 'bet_not_found'
  | 'bet_not_pending'
  | 'not_found'
  | 'internal'
