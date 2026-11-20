import { ref } from 'vue'
import { getBets, settleBetDemo } from '@/api/client'
import { apiErrorMessage } from '@/api/http'
import type { Bet, SettleDemoResponse } from '@/api/types'

const bets = ref<Bet[]>([])
const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)
const settlingId = ref<string | null>(null)

let pending: Promise<void> | null = null

async function fetchBets(limit = 20): Promise<void> {
  // A request that started before a bet was placed may resolve without it.
  // Wait for it, then fetch again so the new bet cannot disappear from history.
  if (pending) {
    await pending
    return fetchBets(limit)
  }

  loading.value = true
  error.value = null

  pending = (async () => {
    try {
      const data = await getBets(limit)
      bets.value = data.bets
      loaded.value = true
    } catch (cause) {
      error.value = apiErrorMessage(cause, 'Не удалось загрузить историю ставок')
    } finally {
      loading.value = false
      pending = null
    }
  })()

  return pending
}

/** Show the server-confirmed bet immediately, then reconcile with the history. */
function prependBet(bet: Bet): void {
  bets.value = [bet, ...bets.value.filter((item) => item.id !== bet.id)].slice(0, 20)
  loaded.value = true
}

/** Demo-only settlement: replace the pending bet with its won/lost copy. */
async function settleDemo(betId: string): Promise<SettleDemoResponse | null> {
  if (settlingId.value) return null
  settlingId.value = betId
  try {
    const response = await settleBetDemo(betId)
    bets.value = bets.value.map((bet) => (bet.id === betId ? response.bet : bet))
    return response
  } finally {
    settlingId.value = null
  }
}

export function useBets() {
  return {
    bets,
    loading,
    loaded,
    error,
    settlingId,
    fetchBets,
    prependBet,
    settleDemo,
  }
}
