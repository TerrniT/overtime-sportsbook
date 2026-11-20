import type { BalanceEvent, LineSnapshot } from './types'

export interface LineStreamHandlers {
  onOpen?: () => void
  onError?: () => void
  onSnapshot?: (snapshot: LineSnapshot) => void
  onLine?: (snapshot: LineSnapshot) => void
  onBalance?: (event: BalanceEvent) => void
}

export interface LineStream {
  close: () => void
}

function parseEvent<T>(event: MessageEvent<string>): T | null {
  try {
    return JSON.parse(event.data) as T
  } catch {
    return null
  }
}

/**
 * EventSource is confined to src/api. Reconnection is left to the browser
 * (EventSource retries on its own); the composable watches for staleness.
 */
export function createLineStream(handlers: LineStreamHandlers): LineStream {
  const source = new EventSource('/api/stream')

  source.addEventListener('open', () => {
    handlers.onOpen?.()
  })

  source.addEventListener('error', () => {
    handlers.onError?.()
  })

  source.addEventListener('snapshot', (event) => {
    const snapshot = parseEvent<LineSnapshot>(event as MessageEvent<string>)
    if (snapshot) handlers.onSnapshot?.(snapshot)
  })

  source.addEventListener('line', (event) => {
    const snapshot = parseEvent<LineSnapshot>(event as MessageEvent<string>)
    if (snapshot) handlers.onLine?.(snapshot)
  })

  source.addEventListener('balance', (event) => {
    const balance = parseEvent<BalanceEvent>(event as MessageEvent<string>)
    if (balance) handlers.onBalance?.(balance)
  })

  return {
    close: () => source.close(),
  }
}
