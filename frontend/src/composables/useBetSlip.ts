import { computed, ref, watch } from 'vue'
import { toast } from '@yusufceng1z/ding-vue'
import { placeBet } from '@/api/client'
import { ApiError, isApiError, isRetryableTransportError } from '@/api/http'
import type { Match, OddsChangedDetails, Outcome, OutcomeKey, PlaceBetResponse } from '@/api/types'
import {
  createIdempotencyKey,
  idempotencyKeyAfter,
  payoutMinor,
  validateStake,
  type StakeError,
} from '@/lib/bet'
import { formatMoneyWhole, formatStakeInput, parseStakeToMinor } from '@/lib/format'
import { useBets } from './useBets'
import { useLine } from './useLine'
import { useSession } from './useSession'

export interface SlipSelection {
  matchId: string
  outcome: OutcomeKey
}

export interface ActiveSelection {
  match: Match
  outcome: Outcome
  priceMilli: number
}

export interface OddsConflict {
  expectedPriceMilli: number
  currentPriceMilli: number
}

export type SubmitErrorKind = 'network' | 'insufficient_funds' | 'rejected' | 'stale'

export interface SubmitError {
  kind: SubmitErrorKind
  message: string
  retryable: boolean
}

interface Attempt {
  matchId: string
  outcome: OutcomeKey
  stakeMinor: number
  expectedPriceMilli: number
}

const line = useLine()
const session = useSession()
const history = useBets()

const selection = ref<SlipSelection | null>(null)
const stakeInput = ref('')
const touched = ref(false)
const inFlight = ref(false)
const submitError = ref<SubmitError | null>(null)
const conflict = ref<OddsConflict | null>(null)
const resolvingConflict = ref(false)
const priceNotice = ref<{ from: number; to: number } | null>(null)

let attemptKey: string | null = null
let lastAttempt: Attempt | null = null
let noticeTimer: ReturnType<typeof setTimeout> | null = null
let activeKey = ''

const active = computed<ActiveSelection | null>(() => {
  const current = line.snapshot.value
  if (!current || !selection.value) return null

  const match = current.matches.find((item) => item.id === selection.value?.matchId)
  const outcome = match?.outcomes.find((item) => item.key === selection.value?.outcome)
  if (!match || !outcome) return null

  return { match, outcome, priceMilli: outcome.priceMilli }
})

const stakeMinor = computed<number | null>(() => parseStakeToMinor(stakeInput.value))

// While a 409 confirmation is open, the slip shows the price from the dialog.
// The live line keeps ticking underneath, but two visible "new" prices would be
// confusing and would make the accept button disagree with its own totals.
const displayPriceMilli = computed<number | null>(() => {
  if (!active.value) return null
  if (conflict.value) return conflict.value.currentPriceMilli
  // A retry repeats the exact attempt, including the price the user agreed to.
  if (submitError.value?.retryable && lastAttempt) return lastAttempt.expectedPriceMilli
  return active.value.priceMilli
})

const validation = computed(() =>
  validateStake(stakeMinor.value, session.player.value?.balanceMinor ?? null, session.limits.value),
)

const effectiveMaxMinor = computed(() => validation.value.effectiveMaxMinor)

const potentialPayoutMinor = computed<number | null>(() => {
  if (displayPriceMilli.value === null || stakeMinor.value === null || stakeMinor.value <= 0) return null
  return payoutMinor(stakeMinor.value, displayPriceMilli.value)
})

const remainingBalanceMinor = computed<number | null>(() => {
  const balance = session.player.value?.balanceMinor
  if (balance === undefined || stakeMinor.value === null) return null
  if (stakeMinor.value <= 0 || stakeMinor.value > balance) return null
  return balance - stakeMinor.value
})

function stakeErrorMessage(error: StakeError | undefined): string | null {
  if (!error) return null

  const limits = session.limits.value
  switch (error) {
    case 'invalid':
      return stakeInput.value.trim() === ''
        ? 'Введите сумму ставки'
        : 'Введите сумму в рублях, например 500 или 500,50'
    case 'min':
      return limits
        ? `Минимальная ставка — ${formatMoneyWhole(limits.minStakeMinor)}`
        : 'Сумма ниже минимальной'
    case 'max':
      return limits
        ? `Максимальная ставка — ${formatMoneyWhole(limits.maxStakeMinor)}`
        : 'Сумма выше максимальной'
    case 'insufficient': {
      const balance = session.player.value?.balanceMinor
      if (balance !== null && balance !== undefined && limits && balance < limits.minStakeMinor) {
        return 'На демо-балансе меньше минимальной ставки'
      }
      return balance === null || balance === undefined
        ? 'Недостаточно средств на балансе'
        : `Недостаточно средств: доступно ${formatMoneyWhole(balance)}`
    }
    case 'unavailable':
      return 'Не удалось загрузить баланс'
  }
}

const fieldError = computed<string | null>(() => {
  if (!touched.value || validation.value.ok) return null
  return stakeErrorMessage(validation.value.error)
})

const submitBlockedReason = computed<string | null>(() => {
  if (inFlight.value) return 'Отправляем ставку…'
  if (!active.value) return 'Выберите исход в линии'
  if (line.stale.value) return 'Линия устарела. Ждём обновления.'
  if (!session.player.value) {
    return session.loading.value ? 'Загружаем баланс…' : 'Баланс недоступен'
  }
  if (!validation.value.ok) {
    if (validation.value.error === 'invalid') return 'Укажите сумму ставки'
    return stakeErrorMessage(validation.value.error)
  }
  return null
})

const canSubmit = computed(
  () =>
    !inFlight.value &&
    !conflict.value &&
    Boolean(active.value) &&
    !line.stale.value &&
    Boolean(session.player.value) &&
    validation.value.ok,
)

const hasUnresolvedAttempt = computed(() => Boolean(submitError.value?.retryable && lastAttempt))

const submitLabel = computed(() => {
  if (inFlight.value) return 'Отправляем…'
  if (submitError.value?.retryable && lastAttempt) return 'Повторить отправку'
  if (validation.value.ok && stakeMinor.value !== null) {
    return `Поставить ${formatMoneyWhole(stakeMinor.value)}`
  }
  return 'Поставить'
})

function clearPriceNotice(): void {
  priceNotice.value = null
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = null
}

function showPriceNotice(from: number, to: number): void {
  priceNotice.value = { from, to }
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => {
    priceNotice.value = null
    noticeTimer = null
  }, 5_000)
}

function resetAttempt(): void {
  attemptKey = null
  lastAttempt = null
}

function resetSlipState(): void {
  resetAttempt()
  submitError.value = null
  conflict.value = null
  touched.value = false
  clearPriceNotice()
}

function setStakeInput(value: string): void {
  if (hasUnresolvedAttempt.value) return
  stakeInput.value = value
  submitError.value = null
  if (!inFlight.value) resetAttempt()
}

function setStakeTouched(): void {
  touched.value = true
}

function setStakeChip(minor: number): void {
  if (minor <= 0) return
  setStakeInput(formatStakeInput(minor))
  touched.value = true
}

function select(match: Match, outcome: Outcome): void {
  if (line.stale.value || hasUnresolvedAttempt.value) return

  const same = selection.value?.matchId === match.id && selection.value.outcome === outcome.key
  if (same) {
    clearSelection()
    return
  }

  selection.value = { matchId: match.id, outcome: outcome.key }
  resetSlipState()
}

function clearSelection(): void {
  if (hasUnresolvedAttempt.value) return
  selection.value = null
  resetSlipState()
}

function onPlaced(response: PlaceBetResponse): void {
  const { bet } = response
  const summary = `${bet.matchLabel} · ${bet.outcomeLabel} · ${formatMoneyWhole(bet.stakeMinor)}`

  toast.success(response.replayed ? 'Ставка уже принята' : 'Ставка принята', {
    description: response.replayed ? `${summary}. Деньги не списаны повторно.` : summary,
  })

  session.setBalance(response.balanceMinor)
  history.prependBet(bet)
  void history.fetchBets()

  selection.value = null
  stakeInput.value = ''
  resetSlipState()
}

function numberDetail(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function handleFailure(cause: unknown, fromConflict: boolean): void {
  const error = isApiError(cause)
    ? cause
    : new ApiError('internal', 'Не удалось отправить ставку')

  if (error.code === 'odds_changed') {
    const details = (error.details ?? {}) as OddsChangedDetails
    const expected =
      numberDetail(details.expectedPriceMilli) ??
      lastAttempt?.expectedPriceMilli ??
      active.value?.priceMilli ??
      0
    const current =
      numberDetail(details.priceMilli) ?? active.value?.priceMilli ?? expected

    conflict.value = { expectedPriceMilli: expected, currentPriceMilli: current }
    resetAttempt()
    submitError.value = null
    clearPriceNotice()

    if (fromConflict) {
      toast.info('Коэффициент обновился ещё раз', {
        description: 'Подтвердите новую цену.',
      })
    }
    return
  }

  if (error.code === 'insufficient_funds') {
    conflict.value = null
    resetAttempt()
    submitError.value = {
      kind: 'insufficient_funds',
      message: 'Недостаточно средств. Мы обновили баланс — уменьшите сумму ставки.',
      retryable: false,
    }
    toast.error('Недостаточно средств', {
      description: 'Баланс обновлён. Уменьшите сумму ставки.',
    })
    void session.fetchSession()
    return
  }

  if (isRetryableTransportError(error) || error.code === 'internal') {
    if (fromConflict) conflict.value = null
    attemptKey = idempotencyKeyAfter('transport', attemptKey)
    submitError.value = {
      kind: 'network',
      message:
        error.code === 'timeout'
          ? 'Сервер не ответил. Повторите отправку — ставка не спишется дважды.'
          : 'Нет связи с сервером. Повторите отправку — ставка не спишется дважды.',
      retryable: true,
    }
    toast.error(error.code === 'timeout' ? 'Сервер не ответил' : 'Нет связи с сервером', {
      description: 'Ставка не подтверждена. Повторите отправку.',
    })
    return
  }

  conflict.value = null
  resetAttempt()
  submitError.value = {
    kind: 'rejected',
    message: error.message,
    retryable: false,
  }
  toast.error(error.message)

  if (error.code === 'market_not_found') {
    void line.refresh()
    void session.fetchSession()
  } else if (error.code === 'stake_out_of_range') {
    void session.fetchSession()
  }
}

async function sendAttempt(key: string, attempt: Attempt, fromConflict = false): Promise<void> {
  inFlight.value = true
  submitError.value = null

  try {
    const response = await placeBet({
      matchId: attempt.matchId,
      outcome: attempt.outcome,
      stakeMinor: attempt.stakeMinor,
      expectedPriceMilli: attempt.expectedPriceMilli,
      idempotencyKey: key,
    })
    conflict.value = null
    onPlaced(response)
  } catch (cause) {
    handleFailure(cause, fromConflict)
  } finally {
    inFlight.value = false
    resolvingConflict.value = false
  }
}

async function submit(): Promise<void> {
  touched.value = true
  if (inFlight.value || !canSubmit.value) return

  // A transport failure retries the exact same attempt and key. The expected
  // price is deliberately the one the user agreed to, not the latest tick.
  if (submitError.value?.retryable && attemptKey && lastAttempt) {
    await sendAttempt(attemptKey, lastAttempt)
    return
  }

  if (!active.value || stakeMinor.value === null) return

  attemptKey = attemptKey ?? createIdempotencyKey()
  lastAttempt = {
    matchId: active.value.match.id,
    outcome: active.value.outcome.key,
    stakeMinor: stakeMinor.value,
    expectedPriceMilli: active.value.priceMilli,
  }

  await sendAttempt(attemptKey, lastAttempt)
}

async function acceptNewPrice(): Promise<void> {
  if (!conflict.value || resolvingConflict.value || inFlight.value) return
  if (!active.value || stakeMinor.value === null) return
  if (line.stale.value) {
    toast.error('Линия устарела', { description: 'Дождитесь обновления линии и повторите.' })
    return
  }

  resolvingConflict.value = true
  attemptKey = createIdempotencyKey()
  lastAttempt = {
    matchId: active.value.match.id,
    outcome: active.value.outcome.key,
    stakeMinor: stakeMinor.value,
    expectedPriceMilli: conflict.value.currentPriceMilli,
  }

  await sendAttempt(attemptKey, lastAttempt, true)
}

function cancelConflict(): void {
  if (resolvingConflict.value) return
  conflict.value = null
  resetAttempt()
}

watch(active, (next, previous) => {
  if (!next) {
    activeKey = ''
    clearPriceNotice()
    return
  }

  if (conflict.value || submitError.value?.retryable) return

  const key = `${next.match.id}:${next.outcome.key}`
  if (previous && activeKey === key && previous.priceMilli !== next.priceMilli) {
    showPriceNotice(previous.priceMilli, next.priceMilli)
  }
  activeKey = key
})

watch(
  () => line.stale.value,
  (isStale) => {
    if (isStale) {
      submitError.value = {
        kind: 'stale',
        message: 'Ставка не отправлена. Дождитесь обновления линии.',
        retryable: false,
      }
    } else if (submitError.value?.kind === 'stale') {
      submitError.value = null
    }
  },
)

export function useBetSlip() {
  return {
    selection,
    active,
    displayPriceMilli,
    stakeInput,
    stakeMinor,
    touched,
    inFlight,
    submitError,
    conflict,
    resolvingConflict,
    priceNotice,
    effectiveMaxMinor,
    validation,
    fieldError,
    submitBlockedReason,
    submitLabel,
    canSubmit,
    hasUnresolvedAttempt,
    potentialPayoutMinor,
    remainingBalanceMinor,
    select,
    clearSelection,
    setStakeInput,
    setStakeChip,
    setStakeTouched,
    submit,
    acceptNewPrice,
    cancelConflict,
    clearPriceNotice,
  }
}
