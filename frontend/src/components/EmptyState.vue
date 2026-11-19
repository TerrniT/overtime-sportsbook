<script setup lang="ts">
import { Ticket } from 'lucide-vue-next'

withDefaults(
  defineProps<{
    title: string
    description?: string
    compact?: boolean
  }>(),
  {
    compact: false,
  },
)
</script>

<template>
  <div class="flex flex-col items-center justify-center px-4 text-center" :class="compact ? 'py-7' : 'py-11'">
    <span
      class="grid place-items-center rounded-[6px] border border-line bg-[#141416]"
      :class="compact ? 'size-8' : 'size-10'"
      aria-hidden="true"
    >
      <Ticket class="text-muted-foreground" :class="compact ? 'size-3.5' : 'size-4'" />
    </span>
    <p
      class="uppercase tracking-[0.08em] text-foreground"
      :class="compact ? 'mt-3 font-display text-[15px]' : 'mt-4 font-display text-[17px]'"
    >
      {{ title }}
    </p>
    <p v-if="description" class="mt-1.5 max-w-[32ch] text-[12px] leading-relaxed text-muted-foreground">
      {{ description }}
    </p>
  </div>
</template>
