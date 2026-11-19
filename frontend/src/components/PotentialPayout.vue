<script setup lang="ts">
import { computed } from 'vue'
import { Info } from 'lucide-vue-next'
import NumberTicker from '@/components/magic/NumberTicker.vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { payoutMinor } from '@/lib/bet'
import { formatMoney, formatOdds, formatPayoutValue } from '@/lib/format'

const props = defineProps<{
  stakeMinor: number | null
  priceMilli: number | null
  balanceMinor: number | null
}>()

const payout = computed(() => {
  if (props.stakeMinor === null || props.priceMilli === null) return null
  if (props.stakeMinor <= 0) return null
  return payoutMinor(props.stakeMinor, props.priceMilli)
})

const remaining = computed(() => {
  if (props.balanceMinor === null || props.stakeMinor === null) return null
  if (props.stakeMinor <= 0 || props.stakeMinor > props.balanceMinor) return null
  return props.balanceMinor - props.stakeMinor
})

const formula = computed(() => {
  if (props.stakeMinor === null || props.priceMilli === null) return null
  return `${formatMoney(props.stakeMinor)} × ${formatOdds(props.priceMilli)}`
})
</script>

<template>
  <div>
    <div class="flex items-center justify-between gap-2">
      <span class="text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
        Возможный выигрыш
      </span>
      <Tooltip>
        <TooltipTrigger as-child>
          <button
            type="button"
            class="inline-flex rounded-[2px] text-faint outline-none hover:text-foreground focus-visible:text-sodium focus-visible:outline focus-visible:outline-1 focus-visible:outline-offset-2 focus-visible:outline-sodium"
            aria-label="Как считается выигрыш"
          >
            <Info class="size-3" aria-hidden="true" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="top" :side-offset="6" class="max-w-[260px] leading-relaxed">
          Ставка умножается на коэффициент, результат округляется вниз до копейки. В демо-режиме
          выплата не начисляется.
        </TooltipContent>
      </Tooltip>
    </div>

    <p v-if="payout !== null" class="mt-1.5 leading-none">
      <NumberTicker
        :value="payout / 100"
        :format="formatPayoutValue"
        :duration="480"
        class="font-display text-[30px] leading-none tracking-[0.01em] text-foreground"
      />
    </p>
    <p v-else class="mt-1.5 font-display text-[30px] leading-none text-[#4e4b46]">—</p>

    <p v-if="formula" class="mt-1 text-[10px] tabular text-faint">{{ formula }}</p>
    <p v-else class="mt-1 text-[10px] text-faint">Введите сумму, чтобы увидеть расчёт</p>

    <div v-if="remaining !== null" class="mt-2.5 flex items-center justify-between gap-2 border-t border-line pt-2">
      <span class="text-[10px] text-faint">Останется на балансе</span>
      <span class="tabular text-[11px] text-muted-foreground">{{ formatMoney(remaining) }}</span>
    </div>
  </div>
</template>
