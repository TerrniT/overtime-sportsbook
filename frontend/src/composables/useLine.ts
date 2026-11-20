import { computed, ref, watch, type WatchStopHandle } from 'vue'
import { toast } from '@yusufceng1z/ding-vue'
import { getLine } from '@/api/client'
import { createLineStream, type LineStream } from '@/api/stream'
import { isApiError } from '@/api/http'
import type { LineSnapshot } from '@/api/types'
import { useSession } from './useSession'

/** Three missed ticks at 3 s each. */
export const STALE_AFTER_MS = 9_000

type ConnectionPhase = 'connecting' | 'live' | 'reconnecting' | 'offline'

const snapshot = ref<LineSnapshot | null>(null)
const loading = ref(true)
const loadError = ref<string | null>(null)
const phase = ref<ConnectionPhase>('connecting')
const lastEventAt = ref(0)
const clock = ref(Date.now())

let stream: LineStream | null = null
let clockTimer: ReturnType<typeof setInterval> | null = null
let stopStaleWatch: WatchStopHandle | null = null
let started = false
let disconnected = false

const stale = computed(() => {
  if (!snapshot.value) return false
  return clock.value - lastEventAt.value > STALE_AFTER_MS
})

const isBettingPaused = computed(() => !snapshot.value || stale.value)

function applySnapshot(next: LineSnapshot): void {
  snapshot.value = next
  lastEventAt.value = Date.now()
  clock.value = lastEventAt.value
  loading.value = false
  loadError.value = null
  if (phase.value === 'connecting' || phase.value === 'offline') {
    phase.value = 'live'
  }
}

function messageFor(cause: unknown): string {
  if (isApiError(cause)) return cause.message
  return 'Не удалось загрузить линию'
}

async function fetchLine(silent = false): Promise<void> {
  if (!silent && !snapshot.value) loading.value = true
  try {
    const data = await getLine()
    applySnapshot(data)
  } catch (cause) {
    if (!snapshot.value) {
      loadError.value = messageFor(cause)
      phase.value = 'offline'
    }
  } finally {
    loading.value = false
  }
}

function connectStream(): void {
  stream?.close()
  stream = createLineStream({
    onOpen: () => {
      phase.value = 'live'
      disconnected = false
      // A reconnect has no replay buffer; ask for a fresh snapshot.
      void fetchLine(true)
    },
    onError: () => {
      if (!disconnected) {
        disconnected = true
        if (phase.value === 'live') {
          toast.warning('Переподключение к линии', {
            description: 'Пытаемся восстановить связь.',
          })
        }
      }
      phase.value = 'reconnecting'
    },
    onSnapshot: (data) => {
      phase.value = 'live'
      disconnected = false
      applySnapshot(data)
    },
    onLine: (data) => {
      phase.value = 'live'
      disconnected = false
      applySnapshot(data)
    },
    onBalance: (event) => {
      useSession().setBalance(event.balanceMinor)
    },
  })
}

function handleVisibility(): void {
  if (document.visibilityState !== 'visible') return
  clock.value = Date.now()
  void fetchLine(true)
}

function start(): void {
  if (started) return
  started = true

  loading.value = !snapshot.value
  void fetchLine()
  connectStream()

  clockTimer = setInterval(() => {
    clock.value = Date.now()
  }, 1000)

  document.addEventListener('visibilitychange', handleVisibility)

  stopStaleWatch = watch(stale, (isStale) => {
    if (isStale) {
      toast.error('Линия устарела', {
        description: 'Ставки приостановлены до обновления линии.',
        duration: 6_000,
      })
    } else if (snapshot.value) {
      toast.success('Связь восстановлена', {
        description: 'Ставки снова доступны.',
        duration: 4_000,
      })
    }
  })
}

function stop(): void {
  started = false
  stream?.close()
  stream = null
  if (clockTimer) clearInterval(clockTimer)
  clockTimer = null
  document.removeEventListener('visibilitychange', handleVisibility)
  stopStaleWatch?.()
  stopStaleWatch = null
}

export function useLine() {
  return {
    snapshot,
    loading,
    loadError,
    phase,
    stale,
    isBettingPaused,
    lastEventAt,
    start,
    stop,
    refresh: () => fetchLine(false),
  }
}
