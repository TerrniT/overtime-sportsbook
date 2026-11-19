import { describe, expect, it } from 'vitest'
import {
  createIdempotencyKey,
  idempotencyKeyAfter,
  payoutMinor,
  validateStake,
} from './bet'

const limits = { minStakeMinor: 10_000, maxStakeMinor: 500_000 }

describe('payout', () => {
  it('floors stake × price to the kopeck, never rounds up', () => {
    expect(payoutMinor(50_000, 1910)).toBe(95_500)
    expect(payoutMinor(33_333, 1950)).toBe(64_999)
    expect(payoutMinor(100, 1050)).toBe(105)
    expect(payoutMinor(0, 1910)).toBe(0)
  })
})

describe('stake validation', () => {
  it('enforces min, max, balance and the effective maximum', () => {
    expect(validateStake(50_000, 2_500_000, limits).ok).toBe(true)
    expect(validateStake(9_999, 2_500_000, limits).error).toBe('min')
    expect(validateStake(500_001, 2_500_000, limits).error).toBe('max')
    expect(validateStake(300_000, 250_000, limits).error).toBe('insufficient')
    expect(validateStake(null, 2_500_000, limits).error).toBe('invalid')
    expect(validateStake(50_000, null, limits).error).toBe('unavailable')
    expect(validateStake(5_000, 9_999, limits).error).toBe('insufficient')
    expect(validateStake(200_000, 250_000, limits).effectiveMaxMinor).toBe(250_000)
    expect(validateStake(200_000, 2_500_000, limits).effectiveMaxMinor).toBe(500_000)
  })
})

describe('idempotency key lifecycle', () => {
  it('reuses the key on transport failure and resets it for a new attempt', () => {
    const key = '3f3c1f9e-6a3e-4c31-9c0e-6c2b0b9a1b77'
    expect(idempotencyKeyAfter('transport', key)).toBe(key)
    expect(idempotencyKeyAfter('transport', null)).toMatch(/^[0-9a-f-]{8,100}$/)
    expect(idempotencyKeyAfter('odds_changed', key)).toBeNull()
    expect(idempotencyKeyAfter('insufficient_funds', key)).toBeNull()
    expect(idempotencyKeyAfter('success', key)).toBeNull()

    const generated = createIdempotencyKey()
    expect(generated.length).toBeGreaterThanOrEqual(8)
    expect(generated.length).toBeLessThanOrEqual(100)
    expect(generated).not.toBe(createIdempotencyKey())
  })
})
