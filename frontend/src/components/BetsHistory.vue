<script setup lang="ts">
import { computed } from 'vue'
import { Dices, RotateCw } from 'lucide-vue-next'
import { toast } from '@yusufceng1z/ding-vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { isApiError } from '@/api/http'
import { useBets } from '@/composables/useBets'
import { useSession } from '@/composables/useSession'
import type { Bet, BetStatus } from '@/api/types'
import { formatMoney, formatOdds, formatPlacedAt } from '@/lib/format'
import EmptyState from './EmptyState.vue'

const { bets, loading, loaded, error, settlingId, fetchBets, settleDemo } = useBets()
const session = useSession()

const statusMap: Record<BetStatus, { label: string; class: string }> = {
  pending: { label: 'В игре', class: 'border-sodium/35 bg-sodium/[0.07] text-sodium' },
  won: { label: 'Выиграла', class: 'border-[#3d5a35] bg-[#121a10] text-[#a9cf98]' },
  lost: { label: 'Проиграла', class: 'border-[#5a2c2c] bg-[#1c1112] text-[#f0b4b4]' },
  void: { label: 'Возврат', class: 'border-line bg-[#17171a] text-muted-foreground' },
}

const showSkeleton = computed(() => loading.value && !loaded.value)
const showEmpty = computed(() => !loading.value && loaded.value && bets.value.length === 0 && !error.value)

function outcomeLabel(bet: Bet): string {
  return bet.outcomeLabel || bet.outcome
}

function payoutLabel(bet: Bet): string {
  if (bet.status === 'won') return 'выигрыш'
  if (bet.status === 'lost') return 'проигрыш'
  if (bet.status === 'void') return 'возврат'
  return 'возможный выигрыш'
}

function payoutAmount(bet: Bet): number {
  return bet.status === 'lost' ? 0 : bet.potentialPayoutMinor
}

async function runDemoSettlement(bet: Bet): Promise<void> {
  if (settlingId.value) return

  try {
    const result = await settleDemo(bet.id)
    if (!result) return

    session.setBalance(result.balanceMinor)
    if (result.won) {
      toast.success('Выигрыш зачислен', {
        description: `${formatMoney(result.bet.potentialPayoutMinor)} · демо-расчёт`,
        duration: 5_000,
      })
    } else {
      toast.info('Ставка проиграла', {
        description: 'Демо-расчёт: баланс не изменился.',
        duration: 5_000,
      })
    }
  } catch (cause) {
    if (isApiError(cause)) {
      toast.error(cause.message)
      if (cause.code === 'bet_not_pending') void fetchBets()
    } else {
      toast.error('Не удалось разыграть ставку')
    }
  }
}
</script>

<template>
  <section aria-labelledby="history-heading">
    <div class="mb-2.5 flex items-end justify-between gap-4">
      <div class="flex items-baseline gap-2.5">
        <h2
          id="history-heading"
          class="font-display text-[20px] font-medium uppercase leading-none tracking-[0.1em] text-foreground"
        >
          Мои ставки
        </h2>
        <span v-if="loaded" class="tabular text-[10px] text-faint">{{ bets.length }}</span>
      </div>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        class="h-7 gap-1.5 px-2 text-[10px] text-muted-foreground hover:bg-[#17171a] hover:text-foreground"
        :disabled="loading"
        @click="fetchBets()"
      >
        <RotateCw class="size-3" :class="loading ? 'animate-spin' : ''" aria-hidden="true" />
        Обновить
      </Button>
    </div>

    <p class="mb-3 max-w-[74ch] text-[10px] leading-relaxed text-faint">
      Демо: ставка сразу списывается с баланса. Исход можно разыграть вручную кнопкой
      «Демо-исход»; автоматического расчёта после матча нет.
    </p>

    <div class="overflow-hidden rounded-[10px] border border-line bg-panel">
      <div v-if="showSkeleton" class="divide-y divide-line" aria-hidden="true">
        <div v-for="index in 3" :key="index" class="flex items-center justify-between gap-4 px-4 py-4">
          <div class="space-y-2">
            <Skeleton class="h-3.5 w-44 max-w-full rounded-[2px]" />
            <Skeleton class="h-2.5 w-28 rounded-[2px]" />
          </div>
          <Skeleton class="h-6 w-16 rounded-[3px]" />
        </div>
      </div>

      <EmptyState
        v-else-if="showEmpty"
        compact
        title="Ставок пока нет"
        description="Здесь появятся ваши ставки."
      />

      <div v-else-if="error && bets.length === 0" class="px-4 py-6 text-center">
        <p class="text-[12px] text-[#f0b4b4]">{{ error }}</p>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="mt-3 border-line bg-transparent text-[11px]"
          @click="fetchBets()"
        >
          Повторить
        </Button>
      </div>

      <template v-else>
        <div v-if="error" class="flex flex-wrap items-center justify-between gap-2 border-b border-line px-4 py-2">
          <p class="text-[10px] text-[#f0b4b4]">{{ error }}</p>
          <button
            type="button"
            class="text-[10px] font-medium text-sodium underline-offset-2 hover:underline"
            @click="fetchBets()"
          >
            Повторить
          </button>
        </div>
        <div
          v-for="bet in bets"
          :key="bet.id"
          class="grid gap-2.5 border-b border-line px-4 py-3.5 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <Badge
                variant="outline"
                class="h-[18px] rounded-[3px] border-line bg-transparent px-1.5 text-[9px] font-medium uppercase tracking-[0.1em] text-muted-foreground"
              >
                {{ outcomeLabel(bet) }}
              </Badge>
              <p class="truncate text-[12px] font-medium text-foreground">{{ bet.matchLabel }}</p>
            </div>
            <p class="mt-1 text-[10px] tabular text-faint">
              {{ formatPlacedAt(bet.placedAt) }} · {{ formatMoney(bet.stakeMinor) }} ×
              {{ formatOdds(bet.priceMilli) }}
            </p>
          </div>

          <div class="flex items-center justify-between gap-4 sm:justify-end">
            <div class="text-right">
              <p class="font-display text-[16px] leading-none tracking-[0.02em] text-foreground tabular">
                {{ formatMoney(payoutAmount(bet)) }}
              </p>
              <p class="mt-0.5 text-[9px] uppercase tracking-[0.1em] text-faint">
                {{ payoutLabel(bet) }}
              </p>
            </div>
            <div class="flex flex-col items-end gap-1.5">
              <Badge
                variant="outline"
                class="h-[20px] rounded-[3px] px-1.5 text-[9px] font-medium uppercase tracking-[0.1em]"
                :class="statusMap[bet.status].class"
              >
                <span
                  v-if="bet.status === 'pending'"
                  class="mr-1 size-1 rounded-full bg-sodium animate-pulse-soft"
                  aria-hidden="true"
                />
                {{ statusMap[bet.status].label }}
              </Badge>
              <Button
                v-if="bet.status === 'pending'"
                type="button"
                variant="outline"
                size="sm"
                :disabled="Boolean(settlingId)"
                class="h-6 gap-1 rounded-[3px] border-line bg-transparent px-2 text-[9px] font-medium uppercase tracking-[0.08em] text-muted-foreground hover:bg-[#17171a] hover:text-foreground disabled:opacity-50"
                :aria-label="`Разыграть демо-исход для ставки ${bet.matchLabel}`"
                @click="runDemoSettlement(bet)"
              >
                <Dices class="size-3" aria-hidden="true" />
                {{ settlingId === bet.id ? 'Разыгрываем…' : 'Демо-исход' }}
              </Button>
            </div>
          </div>
        </div>
      </template>
    </div>
  </section>
</template>
