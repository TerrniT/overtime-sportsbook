<script setup lang="ts">
import { RotateCw } from 'lucide-vue-next'
import NumberTicker from '@/components/magic/NumberTicker.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { useSession } from '@/composables/useSession'
import { formatMoneyValue, formatMoneyWhole } from '@/lib/format'

const { player, loading, fetchSession } = useSession()
</script>

<template>
  <div class="flex flex-col items-end leading-none">
    <span class="max-[359px]:hidden text-[9px] font-medium uppercase tracking-[0.18em] text-faint">Демо-баланс</span>

    <template v-if="player">
      <NumberTicker
        :value="player.balanceMinor / 100"
        :format="formatMoneyValue"
        :duration="800"
        class="mt-1 font-display text-[17px] leading-none tracking-[0.03em] text-foreground min-[360px]:text-[19px]"
      />
      <span class="sr-only" aria-live="polite">{{ formatMoneyWhole(player.balanceMinor) }}</span>
    </template>

    <Skeleton v-else-if="loading" class="mt-1.5 h-5 w-24 rounded-[4px]" />

    <button
      v-else
      type="button"
      class="mt-1 flex items-center gap-1 text-[11px] font-medium text-sodium hover:underline"
      @click="fetchSession"
    >
      <RotateCw class="size-3" aria-hidden="true" />
      Повторить
    </button>
  </div>
</template>
