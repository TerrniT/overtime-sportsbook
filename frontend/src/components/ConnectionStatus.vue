<script setup lang="ts">
import { computed } from 'vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useLine } from '@/composables/useLine'
import { formatTime } from '@/lib/format'

const { phase, stale, snapshot } = useLine()

const state = computed(() => {
  if (stale.value || phase.value === 'offline') {
    return { label: 'Нет связи', dot: 'bg-[#f26d6d]', pulse: false }
  }
  if (phase.value === 'reconnecting') {
    return { label: 'Переподключение…', dot: 'bg-sodium/70', pulse: true }
  }
  if (phase.value === 'live') {
    return { label: 'В линии', dot: 'bg-sodium', pulse: true }
  }
  return { label: 'Подключение…', dot: 'bg-faint', pulse: true }
})

const detail = computed(() => {
  if (!snapshot.value) return 'Ждём первый снимок линии'
  return `Тик #${snapshot.value.seq} · ${formatTime(snapshot.value.updatedAt)} · обновление каждые 3 с`
})
</script>

<template>
  <Tooltip>
    <TooltipTrigger as-child>
      <span
        role="status"
        aria-live="polite"
        tabindex="0"
        class="inline-flex h-6 shrink-0 items-center gap-1.5 rounded-[4px] border border-line bg-panel px-2 text-[9px] font-medium uppercase tracking-[0.14em] text-muted-foreground outline-none transition-colors hover:border-[#3a3a40] focus-visible:border-sodium"
      >
        <span
          class="size-1.5 shrink-0 rounded-full"
          :class="[state.dot, state.pulse ? 'animate-pulse-soft' : '']"
        />
        {{ state.label }}
      </span>
    </TooltipTrigger>
    <TooltipContent side="bottom" :side-offset="8" class="text-[11px] tracking-[0.02em]">
      {{ detail }}
    </TooltipContent>
  </Tooltip>
</template>
