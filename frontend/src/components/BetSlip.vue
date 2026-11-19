<script setup lang="ts">
import { computed } from 'vue'
import { AlertTriangle, Loader2, WifiOff, X } from 'lucide-vue-next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import ShineBorder from '@/components/magic/ShineBorder.vue'
import { useBetSlip } from '@/composables/useBetSlip'
import { useLine } from '@/composables/useLine'
import { useSession } from '@/composables/useSession'
import { formatKickoff, formatOdds } from '@/lib/format'
import EmptyState from './EmptyState.vue'
import PotentialPayout from './PotentialPayout.vue'
import StakeInput from './StakeInput.vue'

withDefaults(defineProps<{ embedded?: boolean; hideHeader?: boolean }>(), {
  embedded: false,
  hideHeader: false,
})

const {
  active,
  displayPriceMilli,
  stakeInput,
  stakeMinor,
  inFlight,
  hasUnresolvedAttempt,
  submitError,
  priceNotice,
  fieldError,
  submitBlockedReason,
  submitLabel,
  canSubmit,
  effectiveMaxMinor,
  setStakeInput,
  setStakeTouched,
  submit,
  clearSelection,
} = useBetSlip()
const { player } = useSession()
const { stale } = useLine()

const balanceMinor = computed(() => player.value?.balanceMinor ?? null)
const stakeDisabled = computed(() => inFlight.value || hasUnresolvedAttempt.value)
const errorIsFunds = computed(() => submitError.value?.kind === 'insufficient_funds')
</script>

<template>
  <Card
    :class="
      cn(
        'relative gap-0 overflow-hidden rounded-[10px] border border-line bg-panel py-0 text-foreground shadow-none ring-0',
        embedded ? 'rounded-none border-0 bg-transparent' : '',
      )
    "
  >
    <ShineBorder :active="Boolean(active)" class="opacity-60" />

    <div v-if="!hideHeader" class="relative flex items-center justify-between border-b border-line px-4 py-3">
      <div class="flex items-baseline gap-2.5">
        <h2 class="font-display text-[19px] font-medium uppercase leading-none tracking-[0.12em] text-foreground">
          Купон
        </h2>
        <span class="text-[9px] uppercase tracking-[0.14em] text-faint">
          {{ active ? '1 исход' : '1X2' }}
        </span>
      </div>
      <span class="flex items-center gap-1.5 text-[9px] uppercase tracking-[0.12em] text-faint">
        <span class="size-1 rounded-full" :class="stale ? 'bg-[#f26d6d]' : 'bg-sodium'" aria-hidden="true" />
        {{ stale ? 'пауза' : 'демо' }}
      </span>
    </div>

    <template v-if="!active">
      <EmptyState
        title="Купон пуст"
        description="Нажмите на коэффициент в линии — исход, цена и сумма появятся здесь."
      />
      <p
        class="border-t border-line px-4 py-3 text-center text-[9px] uppercase leading-relaxed tracking-[0.12em] text-faint"
      >
        1X2 · одиночные ставки · демо-баланс
      </p>
    </template>

    <form v-else novalidate @submit.prevent="submit">
      <div class="px-4 pt-4 pb-3.5">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-[9px] uppercase tracking-[0.14em] text-faint">
              {{ active.match.league }} · {{ formatKickoff(active.match.startsAt) }}
            </p>
            <h3
              class="mt-1 text-balance break-words font-display text-[17px] font-medium uppercase leading-[1.2] tracking-[0.02em] text-foreground"
            >
              {{ active.match.homeTeam }} — {{ active.match.awayTeam }}
            </h3>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="-mt-1 -mr-1.5 shrink-0 gap-1 px-1.5 text-[9px] font-medium uppercase tracking-[0.1em] text-faint hover:bg-[#1b1b1e] hover:text-foreground disabled:pointer-events-none disabled:opacity-40"
            :disabled="hasUnresolvedAttempt"
            aria-label="Убрать исход из купона"
            @click="clearSelection"
          >
            <X class="size-3" aria-hidden="true" />
            Убрать
          </Button>
        </div>

        <div class="mt-3 flex items-center justify-between gap-3 border-t border-line pt-3">
          <div class="flex items-center gap-2">
            <span
              class="inline-flex h-6 min-w-8 items-center justify-center rounded-[3px] bg-sodium px-1.5 font-display text-[11px] font-semibold uppercase leading-none text-ink"
            >
              {{ active.outcome.label }}
            </span>
            <span class="text-[11px] text-muted-foreground">Одиночная ставка</span>
          </div>
          <span class="font-display text-[26px] leading-none tracking-[0.02em] text-foreground tabular">
            {{ displayPriceMilli !== null ? formatOdds(displayPriceMilli) : '—' }}
          </span>
        </div>

        <Transition name="slip-notice">
          <div
            v-if="priceNotice"
            class="mt-2.5 flex items-center gap-2 rounded-[6px] border border-sodium/25 bg-sodium/[0.06] px-2.5 py-1.5"
            role="status"
            aria-live="polite"
          >
            <span class="size-1 shrink-0 rounded-full bg-sodium" aria-hidden="true" />
            <p class="text-[10px] leading-relaxed text-[#ffc4a8]">
              Коэффициент изменился: {{ formatOdds(priceNotice.from) }} →
              {{ formatOdds(priceNotice.to) }}
            </p>
          </div>
        </Transition>
      </div>

      <Separator class="bg-line" />
      <div class="px-4 py-3.5">
        <StakeInput
          :model-value="stakeInput"
          :balance-minor="balanceMinor"
          :effective-max-minor="effectiveMaxMinor"
          :error="fieldError"
          :disabled="stakeDisabled"
          @update:model-value="setStakeInput"
          @blur="setStakeTouched"
        />
      </div>

      <Separator class="bg-line" />
      <div class="px-4 py-3.5">
        <PotentialPayout
          :stake-minor="stakeMinor"
          :price-milli="displayPriceMilli"
          :balance-minor="balanceMinor"
        />
      </div>

      <Separator class="bg-line" />
      <div class="px-4 pt-3.5 pb-[max(14px,env(safe-area-inset-bottom))]">
        <div
          v-if="submitError"
          class="mb-3 flex items-start gap-2 rounded-[6px] border px-2.5 py-2"
          :class="
            errorIsFunds
              ? 'border-[#5a2c2c] bg-[#1c1112]'
              : 'border-[#5b3520] bg-[#1b120c]'
          "
          role="alert"
        >
          <WifiOff
            v-if="submitError.kind === 'network'"
            class="mt-0.5 size-3.5 shrink-0 text-sodium"
            aria-hidden="true"
          />
          <AlertTriangle v-else class="mt-0.5 size-3.5 shrink-0 text-sodium" aria-hidden="true" />
          <p
            class="text-[10px] leading-relaxed"
            :class="errorIsFunds ? 'text-[#f0b4b4]' : 'text-[#e8c0a8]'"
          >
            {{ submitError.message }}
          </p>
        </div>

        <Button
          type="submit"
          :disabled="!canSubmit"
          class="h-11 w-full rounded-lg bg-sodium font-display text-[15px] font-medium uppercase tracking-[0.1em] text-ink shadow-none transition-colors hover:bg-[#ff7c45] disabled:bg-[#241a14] disabled:text-[#a4836f] disabled:opacity-100"
        >
          <Loader2 v-if="inFlight" class="size-4 animate-spin" aria-hidden="true" />
          {{ submitLabel }}
        </Button>

        <p v-if="submitBlockedReason" class="mt-2 text-center text-[10px] leading-relaxed text-faint">
          {{ submitBlockedReason }}
        </p>
        <p v-else class="mt-2 text-center text-[10px] leading-relaxed text-faint">
          Цена фиксируется в момент ставки · демо-режим
        </p>
      </div>
    </form>
  </Card>
</template>
