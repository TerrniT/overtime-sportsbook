<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { ArrowRight, X } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { useBetSlip } from '@/composables/useBetSlip'
import { formatOdds } from '@/lib/format'
import BetSlip from './BetSlip.vue'

const { active, conflict } = useBetSlip()
const open = ref(false)

watch(active, (next, previous) => {
  if (!next && previous && open.value) open.value = false
})

// A price-change confirmation is its own modal; close the sheet first so the
// two focus traps never stack.
watch(conflict, (next) => {
  if (next && open.value) open.value = false
})

// While the sheet is open, a held error toast must not cover the retry action.
watch(open, (value) => {
  document.documentElement.classList.toggle('slip-sheet-open', value)
})

onBeforeUnmount(() => {
  document.documentElement.classList.remove('slip-sheet-open')
})
</script>

<template>
  <div class="fixed inset-x-0 bottom-0 z-40 border-t border-line bg-[#0b0b0d] lg:hidden">
    <div class="flex items-center gap-3 px-4 pt-2.5 pb-[max(10px,env(safe-area-inset-bottom))]">
      <div class="min-w-0 flex-1">
        <p class="text-[9px] font-medium uppercase tracking-[0.14em] text-faint">
          {{ active ? 'В купоне · 1 исход' : 'Купон пуст' }}
        </p>
        <p v-if="active" class="mt-0.5 truncate text-[11px] text-muted-foreground">
          {{ active.match.homeTeam }} — {{ active.match.awayTeam }} ·
          <span class="text-foreground">{{ active.outcome.label }}</span>
          <span class="tabular ml-1 text-sodium">{{ formatOdds(active.priceMilli) }}</span>
        </p>
        <p v-else class="mt-0.5 text-[11px] text-muted-foreground">Выберите исход в линии</p>
      </div>

      <Button
        type="button"
        class="h-10 shrink-0 rounded-lg bg-sodium px-4 font-display text-[13px] font-medium uppercase tracking-[0.1em] text-ink shadow-none hover:bg-[#ff7c45]"
        @click="open = true"
      >
        Купон
        <ArrowRight class="size-3.5" aria-hidden="true" />
      </Button>
    </div>
  </div>

  <Dialog v-model:open="open">
    <DialogContent
      class="slip-sheet fixed inset-x-0 top-auto! bottom-0 left-0! grid max-h-[88svh] w-full! max-w-none! translate-x-0! translate-y-0! gap-0! overflow-y-auto rounded-t-xl! rounded-b-none! border border-line bg-popover p-0! ring-1 ring-white/10 data-open:slide-in-from-bottom-4 sm:max-w-none!"
      :show-close-button="false"
    >
      <DialogHeader class="sticky top-0 z-30 border-b border-line bg-popover px-4 py-3 pr-12 text-left">
        <DialogTitle
          class="font-display text-[19px] font-medium uppercase leading-none tracking-[0.12em] text-foreground"
        >
          Купон
        </DialogTitle>
        <DialogDescription class="sr-only">
          Выберите сумму и подтвердите ставку.
        </DialogDescription>
        <DialogClose as-child>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            class="absolute top-2 right-2 text-faint hover:bg-[#1b1b1e] hover:text-foreground"
            aria-label="Закрыть купон"
          >
            <X class="size-4" />
          </Button>
        </DialogClose>
      </DialogHeader>

      <BetSlip embedded hide-header />
    </DialogContent>
  </Dialog>
</template>
