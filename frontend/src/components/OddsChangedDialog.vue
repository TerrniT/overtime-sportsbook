<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, Loader2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { useBetSlip } from '@/composables/useBetSlip'
import { useLine } from '@/composables/useLine'
import { payoutMinor } from '@/lib/bet'
import { formatMoney, formatOdds } from '@/lib/format'

const { conflict, resolvingConflict, stakeMinor, acceptNewPrice, cancelConflict } = useBetSlip()
const { stale } = useLine()

const open = computed(() => Boolean(conflict.value))

const newPayout = computed(() => {
  if (!conflict.value || stakeMinor.value === null) return null
  return payoutMinor(stakeMinor.value, conflict.value.currentPriceMilli)
})

function onOpenChange(value: boolean): void {
  if (!value) cancelConflict()
}
</script>

<template>
  <Dialog :open="open" @update:open="onOpenChange">
    <DialogContent
      v-if="conflict"
      :show-close-button="!resolvingConflict"
      class="gap-0 rounded-xl border border-line bg-popover p-0 ring-1 ring-white/10 sm:max-w-[440px]"
      @interact-outside.prevent
    >
      <DialogHeader class="border-b border-line px-5 py-4">
        <DialogTitle
          class="font-display text-[23px] font-medium uppercase leading-none tracking-[0.02em] text-foreground"
        >
          Коэффициент изменился
        </DialogTitle>
        <DialogDescription class="mt-2 text-[12px] leading-relaxed text-muted-foreground">
          Пока вы заполняли купон, цена обновилась. Ставка по старой цене не принята, деньги не
          списаны.
        </DialogDescription>
      </DialogHeader>

      <div class="px-5 py-4">
        <div class="grid grid-cols-[1fr_auto_1fr] items-center gap-3">
          <div>
            <p class="text-[9px] uppercase tracking-[0.14em] text-faint">Вы видели</p>
            <p
              class="mt-1 font-display text-[30px] leading-none tracking-[0.02em] text-[#6f6a62] line-through decoration-[#4a4742] tabular"
            >
              {{ formatOdds(conflict.expectedPriceMilli) }}
            </p>
          </div>
          <ArrowRight class="mt-4 size-4 text-faint" aria-hidden="true" />
          <div class="text-right">
            <p class="text-[9px] uppercase tracking-[0.14em] text-faint">Сейчас</p>
            <p
              class="mt-1 font-display text-[30px] leading-none tracking-[0.02em] text-sodium tabular"
            >
              {{ formatOdds(conflict.currentPriceMilli) }}
            </p>
          </div>
        </div>

        <div class="mt-4 flex items-center justify-between gap-3 rounded-[7px] border border-line bg-[#0d0d0f] px-3 py-2.5">
          <div>
            <p class="text-[9px] uppercase tracking-[0.14em] text-faint">Новый потенциальный выигрыш</p>
            <p class="mt-0.5 text-[10px] text-muted-foreground">
              при той же сумме ставки
            </p>
          </div>
          <p class="font-display text-[19px] leading-none tracking-[0.02em] text-foreground tabular">
            {{ newPayout !== null ? formatMoney(newPayout) : '—' }}
          </p>
        </div>
      </div>

      <DialogFooter
        class="flex flex-col-reverse gap-2 border-t border-line px-5 py-4 sm:flex-row sm:justify-end"
      >
        <Button
          type="button"
          variant="outline"
          :disabled="resolvingConflict"
          class="h-10 border-line bg-transparent font-display text-[13px] uppercase tracking-[0.08em] text-muted-foreground hover:bg-[#1b1b1e] hover:text-foreground"
          @click="cancelConflict"
        >
          Отменить
        </Button>
        <Button
          type="button"
          :disabled="resolvingConflict || stale"
          class="h-10 bg-sodium font-display text-[13px] font-medium uppercase tracking-[0.08em] text-ink hover:bg-[#ff7c45] disabled:bg-[#241a14] disabled:text-[#a4836f] disabled:opacity-100"
          @click="acceptNewPrice"
        >
          <Loader2 v-if="resolvingConflict" class="size-4 animate-spin" aria-hidden="true" />
          Принять {{ formatOdds(conflict.currentPriceMilli) }}
        </Button>
      </DialogFooter>

      <p v-if="stale" class="border-t border-line px-5 py-2.5 text-[10px] leading-relaxed text-[#e8c0a8]" role="status">
        Линия устарела. Подтверждение недоступно, пока линия не обновится.
      </p>
    </DialogContent>
  </Dialog>
</template>
