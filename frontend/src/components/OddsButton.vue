<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ArrowDownRight, ArrowUpRight, Minus } from 'lucide-vue-next'
import { cn } from '@/lib/utils'
import NumberTicker from '@/components/magic/NumberTicker.vue'
import type { Outcome } from '@/api/types'
import { formatOdds, formatOddsValue } from '@/lib/format'

const props = defineProps<{
  outcome: Outcome
  selected: boolean
  disabled?: boolean
  matchLabel: string
}>()

const emit = defineEmits<{
  select: []
}>()

const flashVisible = ref(false)
const flashKey = ref(0)
let flashTimer: ReturnType<typeof setTimeout> | null = null

watch(
  () => props.outcome.priceMilli,
  (next, previous) => {
    if (next === previous) return
    flashKey.value += 1
    flashVisible.value = true
    if (flashTimer) clearTimeout(flashTimer)
    flashTimer = setTimeout(() => {
      flashVisible.value = false
      flashTimer = null
    }, 80)
  },
)

onBeforeUnmount(() => {
  if (flashTimer) clearTimeout(flashTimer)
})

const trendLabel = computed(() => {
  switch (props.outcome.trend) {
    case 'up':
      return 'Цена выросла'
    case 'down':
      return 'Цена упала'
    default:
      return 'Цена без изменений'
  }
})

const ariaLabel = computed(
  () =>
    `${props.matchLabel}, ${props.outcome.label}, коэффициент ${formatOdds(props.outcome.priceMilli)}`,
)
</script>

<template>
  <button
    type="button"
    :disabled="disabled"
    :aria-pressed="selected"
    :aria-label="ariaLabel"
    :title="`${outcome.label} · ${formatOdds(outcome.priceMilli)} · ${trendLabel}`"
    :class="
      cn(
        'group/odds relative flex h-[58px] flex-col justify-between overflow-hidden rounded-lg border px-2.5 py-2 text-left outline-none transition-[background-color,border-color,box-shadow,opacity,transform] duration-200 focus-visible:border-sodium focus-visible:ring-2 focus-visible:ring-sodium/30 sm:h-[62px] lg:px-3',
        selected
          ? 'border-sodium bg-sodium text-ink shadow-[0_0_0_1px_rgba(255,106,42,0.25),0_8px_28px_-12px_rgba(255,106,42,0.55)]'
          : 'border-line bg-[#151517] text-foreground hover:border-[#3d3d43] hover:bg-[#1a1a1d] active:translate-y-px',
        disabled ? 'cursor-not-allowed opacity-40 hover:border-line hover:bg-[#151517]' : '',
      )
    "
    @click="emit('select')"
  >
    <Transition name="price-flash">
      <span
        v-if="flashVisible"
        :key="flashKey"
        aria-hidden="true"
        class="pointer-events-none absolute inset-0 rounded-[inherit]"
        :class="selected ? 'bg-white/20' : 'bg-sodium/[0.09]'"
      />
    </Transition>

    <span class="relative flex items-center justify-between gap-1">
      <span
        class="text-[9px] font-medium uppercase leading-none tracking-[0.16em]"
        :class="selected ? 'text-ink/70' : 'text-muted-foreground'"
      >
        {{ outcome.label }}
      </span>

      <span
        v-if="outcome.trend !== 'flat'"
        :title="trendLabel"
        class="inline-flex items-center"
        :class="selected ? 'text-ink/70' : outcome.trend === 'up' ? 'text-[#d7d0c2]' : 'text-[#8b857b]'"
      >
        <ArrowUpRight v-if="outcome.trend === 'up'" class="size-3" aria-hidden="true" />
        <ArrowDownRight v-else class="size-3" aria-hidden="true" />
        <span class="sr-only">{{ trendLabel }}</span>
      </span>
      <Minus
        v-else
        class="size-2.5 opacity-40"
        :class="selected ? 'text-ink/50' : 'text-faint'"
        aria-hidden="true"
      />
    </span>

    <span class="relative block leading-none">
      <NumberTicker
        :value="outcome.priceMilli / 1000"
        :format="formatOddsValue"
        :duration="420"
        class="font-display text-[25px] font-medium leading-none tracking-[0.01em] sm:text-[27px]"
      />
    </span>
  </button>
</template>
