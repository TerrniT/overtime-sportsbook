import { ref } from 'vue'
import { getSession } from '@/api/client'
import { isApiError } from '@/api/http'
import type { Limits, Player } from '@/api/types'

const player = ref<Player | null>(null)
const limits = ref<Limits | null>(null)
const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)

let pending: Promise<void> | null = null
let pendingBalance: number | null = null

function messageFor(cause: unknown): string {
  if (isApiError(cause)) return cause.message
  return 'Не удалось загрузить баланс'
}

async function fetchSession(): Promise<void> {
  if (pending) return pending

  loading.value = true
  error.value = null

  pending = (async () => {
    try {
      const data = await getSession()
      player.value = {
        ...data.player,
        balanceMinor: pendingBalance ?? data.player.balanceMinor,
      }
      limits.value = data.limits
      loaded.value = true
      pendingBalance = null
    } catch (cause) {
      error.value = messageFor(cause)
    } finally {
      loading.value = false
      pending = null
    }
  })()

  return pending
}

/** Balance events and POST responses are newer than the session snapshot. */
function setBalance(balanceMinor: number): void {
  if (player.value) {
    player.value = { ...player.value, balanceMinor }
  } else {
    pendingBalance = balanceMinor
  }
}

export function useSession() {
  return {
    player,
    limits,
    loading,
    loaded,
    error,
    fetchSession,
    setBalance,
  }
}
