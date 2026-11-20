import { ref } from 'vue'
import { getBets } from '@/api/client'
import { apiErrorMessage } from '@/api/http'
import type { Bet } from '@/api/types'

const bets = ref<Bet[]>([])
const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)

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

export function useBets() {
  return {
    bets,
    loading,
    loaded,
    error,
    fetchBets,
    prependBet,
  }
}
