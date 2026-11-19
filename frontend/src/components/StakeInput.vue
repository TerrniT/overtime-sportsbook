<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatMoneyWhole, formatStakeInput, parseStakeToMinor } from '@/lib/format'

const props = defineProps<{
  modelValue: string
  balanceMinor: number | null
  effectiveMaxMinor: number
  error: string | null
  disabled?: boolean
  inputId?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  blur: []
}>()

const id = computed(() => props.inputId ?? 'stake')
const errorId = computed(() => `${id.value}-error`)

const model = computed({
  get: () => props.modelValue,
  set: (value: string) => emit('update:modelValue', value),
})

const chips = [
  { label: '100 ₽', minor: 10_000 },
  { label: '500 ₽', minor: 50_000 },
  { label: '1 000 ₽', minor: 100_000 },
]

function chipActive(minor: number): boolean {
  return parseStakeToMinor(props.modelValue) === minor
}

function applyChip(minor: number): void {
  emit('update:modelValue', formatStakeInput(minor))
  emit('blur')
}

function applyMax(): void {
  if (props.effectiveMaxMinor > 0) applyChip(props.effectiveMaxMinor)
}

const maxDisabled = computed(() => props.disabled || props.effectiveMaxMinor <= 0)
</script>

<template>
  <div>
    <div class="flex items-baseline justify-between gap-3">
      <label :for="id" class="text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
        Сумма ставки
      </label>
      <span v-if="balanceMinor !== null" class="tabular text-[10px] text-faint">
        доступно {{ formatMoneyWhole(balanceMinor) }}
      </span>
    </div>

    <div class="relative mt-2">
      <Input
        :id="id"
        v-model="model"
        type="text"
        inputmode="decimal"
        autocomplete="off"
        placeholder="0"
        :disabled="disabled"
        :aria-invalid="Boolean(error)"
        :aria-describedby="error ? errorId : undefined"
        :class="
          cn(
            'h-12 rounded-lg border-input bg-[#0c0c0e] pr-10 font-display text-[23px] tracking-[0.02em] text-foreground tabular shadow-none transition-colors placeholder:text-[#5f5b54] focus-visible:border-sodium focus-visible:ring-2 focus-visible:ring-sodium/25',
            error ? 'border-[#a14a4a]' : '',
          )
        "
        @blur="emit('blur')"
      />
      <span
        class="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 font-display text-[16px] text-muted-foreground"
        aria-hidden="true"
      >
        ₽
      </span>
    </div>

    <p v-if="error" :id="errorId" class="mt-1.5 text-[11px] leading-relaxed text-[#f1a1a1]" role="alert">
      {{ error }}
    </p>

    <div class="mt-2.5 flex flex-wrap gap-1.5">
      <Button
        v-for="chip in chips"
        :key="chip.minor"
        type="button"
        variant="outline"
        size="sm"
        :disabled="disabled"
        :class="
          cn(
            'h-7 rounded-[6px] border-line bg-transparent px-2.5 text-[11px] font-medium text-muted-foreground tabular shadow-none hover:border-[#3d3d43] hover:bg-[#17171a] hover:text-foreground',
            chipActive(chip.minor) ? 'border-sodium/60 bg-sodium/[0.08] text-foreground' : '',
          )
        "
        @click="applyChip(chip.minor)"
      >
        {{ chip.label }}
      </Button>
      <Button
        type="button"
        variant="outline"
        size="sm"
        :disabled="maxDisabled"
        :class="
          cn(
            'h-7 rounded-[6px] border-line bg-transparent px-2.5 text-[11px] font-medium text-muted-foreground shadow-none hover:border-[#3d3d43] hover:bg-[#17171a] hover:text-foreground',
            chipActive(effectiveMaxMinor) ? 'border-sodium/60 bg-sodium/[0.08] text-foreground' : '',
          )
        "
        @click="applyMax"
      >
        Макс
      </Button>
    </div>
  </div>
</template>
