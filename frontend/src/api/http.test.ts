import { describe, expect, it } from 'vitest'
import { ApiError, apiErrorFromResponse, isRetryableTransportError } from './http'

describe('API error envelope', () => {
  it('maps a 409 odds_changed response with its price details', () => {
    const error = apiErrorFromResponse(
      409,
      JSON.stringify({
        error: {
          code: 'odds_changed',
          message: 'Коэффициент изменился',
          details: { expectedPriceMilli: 1910, priceMilli: 1920 },
        },
      }),
    )

    expect(error.code).toBe('odds_changed')
    expect(error.status).toBe(409)
    expect(error.message).toBe('Коэффициент изменился')
    expect(error.details?.priceMilli).toBe(1920)
  })

  it('falls back to a readable internal error for a non-JSON body', () => {
    const error = apiErrorFromResponse(500, '<html>oops</html>')
    expect(error.code).toBe('internal')
    expect(error.status).toBe(500)
    expect(error.message).toBe('Внутренняя ошибка сервера')
  })

  it('classifies transport failures as retryable but rejects odds changes', () => {
    expect(isRetryableTransportError(new ApiError('network', 'нет сети'))).toBe(true)
    expect(isRetryableTransportError(new ApiError('timeout', 'таймаут'))).toBe(true)
    expect(isRetryableTransportError(new ApiError('odds_changed', 'цена', 409))).toBe(false)
    expect(isRetryableTransportError(new Error('other'))).toBe(false)
  })
})
