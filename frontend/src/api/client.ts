import { request } from './http'
import type {
  BetsResponse,
  LineSnapshot,
  PlaceBetRequest,
  PlaceBetResponse,
  SessionResponse,
  SettleDemoResponse,
} from './types'

export function getSession(): Promise<SessionResponse> {
  return request<SessionResponse>('/session')
}

export function getLine(): Promise<LineSnapshot> {
  return request<LineSnapshot>('/line')
}

export function getBets(limit = 20): Promise<BetsResponse> {
  const safeLimit = Math.min(50, Math.max(1, Math.trunc(limit)))
  return request<BetsResponse>(`/bets?limit=${safeLimit}`)
}

export function placeBet(payload: PlaceBetRequest): Promise<PlaceBetResponse> {
  return request<PlaceBetResponse>('/bets', { method: 'POST', body: payload })
}

/** Demo-only: resolve a pending bet immediately and return the won/lost state. */
export function settleBetDemo(betId: string): Promise<SettleDemoResponse> {
  return request<SettleDemoResponse>(`/bets/${encodeURIComponent(betId)}/settle-demo`, {
    method: 'POST',
  })
}
