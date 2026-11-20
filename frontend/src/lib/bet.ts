export interface StakeLimits {
  minStakeMinor: number
  maxStakeMinor: number
}

export type StakeError = 'invalid' | 'min' | 'max' | 'insufficient' | 'unavailable'

export interface StakeValidation {
  ok: boolean
  error?: StakeError
  effectiveMaxMinor: number
}

/**
 * Client-side mirror of the server rules. The server remains authoritative;
 * this only prevents avoidable round trips.
 */
export function validateStake(
  stakeMinor: number | null,
  balanceMinor: number | null,
  limits: StakeLimits | null,
): StakeValidation {
  if (!limits || balanceMinor === null) {
    return { ok: false, error: 'unavailable', effectiveMaxMinor: 0 }
  }

  const effectiveMaxMinor = Math.max(0, Math.min(limits.maxStakeMinor, balanceMinor))

  if (balanceMinor < limits.minStakeMinor) {
    return { ok: false, error: 'insufficient', effectiveMaxMinor }
  }

  if (stakeMinor === null) {
    return { ok: false, error: 'invalid', effectiveMaxMinor }
  }

  if (stakeMinor <= 0) {
    return { ok: false, error: 'min', effectiveMaxMinor }
  }

  if (stakeMinor < limits.minStakeMinor) {
    return { ok: false, error: 'min', effectiveMaxMinor }
  }

  if (stakeMinor > limits.maxStakeMinor) {
    return { ok: false, error: 'max', effectiveMaxMinor }
  }

  if (stakeMinor > balanceMinor) {
    return { ok: false, error: 'insufficient', effectiveMaxMinor }
  }

  return { ok: true, effectiveMaxMinor }
}

/** stake × price / 1000, floored to the kopeck. Never rounded up. */
export function payoutMinor(stakeMinor: number, priceMilli: number): number {
  if (!Number.isFinite(stakeMinor) || !Number.isFinite(priceMilli)) return 0
  if (stakeMinor <= 0 || priceMilli <= 0) return 0
  const product = Math.trunc(stakeMinor) * Math.trunc(priceMilli)
  if (!Number.isSafeInteger(product)) {
    return Math.floor(Number.MAX_SAFE_INTEGER / 1000)
  }
  return Math.floor(product / 1000)
}

export type AttemptFailure = 'transport' | 'odds_changed' | 'insufficient_funds' | 'rejected' | 'success'

/**
 * Idempotency lifecycle. A transport failure must retry with the same key, so
 * a request that reached the server but lost its response is replayed instead
 * of debited twice. A new bet (including an accepted new price) gets a new key.
 */
export function idempotencyKeyAfter(failure: AttemptFailure, current: string | null): string | null {
  if (failure === 'transport') return current ?? createIdempotencyKey()
  return null
}

function randomHexPart(): string {
  return Math.floor(Math.random() * 0xffffffff).toString(16).padStart(8, '0')
}

export function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    const bytes = crypto.getRandomValues(new Uint8Array(16))
    bytes[6] = (bytes[6] & 0x0f) | 0x40
    bytes[8] = (bytes[8] & 0x3f) | 0x80
    const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
  }

  return `${randomHexPart()}-${randomHexPart().slice(0, 4)}-4${randomHexPart().slice(0, 3)}-a${randomHexPart().slice(0, 3)}-${randomHexPart()}${randomHexPart().slice(0, 4)}`
}
