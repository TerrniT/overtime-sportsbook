import type { ErrorCode } from './types'

export type ClientErrorCode = ErrorCode | 'network' | 'timeout'

const STATUS_FALLBACK: Record<number, ErrorCode> = {
  404: 'not_found',
  409: 'odds_changed',
  422: 'validation_error',
  500: 'internal',
}

const FALLBACK_MESSAGE: Record<ClientErrorCode, string> = {
  validation_error: 'Проверьте параметры запроса',
  stake_out_of_range: 'Сумма ставки вне допустимого диапазона',
  insufficient_funds: 'Недостаточно средств на балансе',
  odds_changed: 'Коэффициент изменился',
  market_not_found: 'Рынок больше не доступен',
  bet_not_found: 'Ставка не найдена',
  bet_not_pending: 'Ставка уже рассчитана',
  not_found: 'Сервер не нашёл запрошенный ресурс',
  internal: 'Внутренняя ошибка сервера',
  network: 'Нет связи с сервером',
  timeout: 'Сервер не ответил вовремя',
}

export class ApiError extends Error {
  readonly code: ClientErrorCode
  readonly status: number
  readonly details: Record<string, unknown> | undefined

  constructor(code: ClientErrorCode, message: string, status = 0, details?: Record<string, unknown>) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.details = details
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}

/** The server message when there is one, otherwise the caller's load-specific fallback. */
export function apiErrorMessage(error: unknown, fallback: string): string {
  return isApiError(error) ? error.message : fallback
}

/** Network and timeout failures are safe to retry with the same idempotency key. */
export function isRetryableTransportError(error: unknown): boolean {
  return isApiError(error) && (error.code === 'network' || error.code === 'timeout')
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/**
 * Turns any non-2xx response into the shared ApiError.
 * Exported separately so the envelope contract can be tested without fetch.
 */
export function apiErrorFromResponse(status: number, bodyText: string): ApiError {
  let parsed: unknown
  try {
    parsed = bodyText ? JSON.parse(bodyText) : undefined
  } catch {
    parsed = undefined
  }

  if (isRecord(parsed) && isRecord(parsed.error)) {
    const envelope = parsed.error
    const rawCode = typeof envelope.code === 'string' ? envelope.code : ''
    const code = (rawCode || STATUS_FALLBACK[status] || 'internal') as ClientErrorCode
    const message =
      typeof envelope.message === 'string' && envelope.message.trim()
        ? envelope.message
        : FALLBACK_MESSAGE[code] ?? FALLBACK_MESSAGE.internal
    const details = isRecord(envelope.details) ? envelope.details : undefined
    return new ApiError(code, message, status, details)
  }

  const code = STATUS_FALLBACK[status] ?? 'internal'
  return new ApiError(code, FALLBACK_MESSAGE[code], status)
}

const DEFAULT_TIMEOUT_MS = 12_000

interface RequestOptions {
  method?: 'GET' | 'POST'
  body?: unknown
  timeoutMs?: number
}

/**
 * The only fetch in the client. URL building, status handling and the JSON
 * error envelope stay inside src/api.
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? DEFAULT_TIMEOUT_MS)

  let response: Response
  try {
    response = await fetch(`/api${path}`, {
      method: options.method ?? 'GET',
      headers: {
        Accept: 'application/json',
        ...(options.body === undefined ? {} : { 'Content-Type': 'application/json' }),
      },
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      signal: controller.signal,
    })
  } catch {
    if (controller.signal.aborted) {
      throw new ApiError('timeout', FALLBACK_MESSAGE.timeout)
    }
    throw new ApiError('network', FALLBACK_MESSAGE.network)
  } finally {
    clearTimeout(timeout)
  }

  const bodyText = await response.text()

  if (!response.ok) {
    throw apiErrorFromResponse(response.status, bodyText)
  }

  if (!bodyText) {
    return undefined as T
  }

  try {
    return JSON.parse(bodyText) as T
  } catch {
    throw new ApiError('internal', 'Сервер вернул некорректный ответ', response.status)
  }
}
