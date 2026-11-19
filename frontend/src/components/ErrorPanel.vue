<script setup lang="ts">
import { WifiOff } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

defineProps<{
  title?: string
  description?: string
}>()

const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="rounded-[10px] border border-[#3d2929] bg-[#150f10] px-5 py-9 text-center sm:px-8 sm:py-12">
    <span
      class="mx-auto grid size-10 place-items-center rounded-[6px] border border-[#4a2c2c] bg-[#1d1213]"
      aria-hidden="true"
    >
      <WifiOff class="size-4 text-[#f26d6d]" />
    </span>
    <h2 class="mt-4 font-display text-[20px] uppercase tracking-[0.04em] text-foreground">
      {{ title ?? 'Связь с линией потеряна' }}
    </h2>
    <p class="mx-auto mt-1.5 max-w-[42ch] text-[12px] leading-relaxed text-[#b9a7a7]">
      {{ description ?? 'Бэкенд не ответил. Повторите запрос — данные появятся, как только связь вернётся.' }}
    </p>
    <Button variant="outline" size="sm" class="mt-5 border-[#4a2c2c] hover:bg-[#221517]" @click="emit('retry')">
      Повторить
    </Button>
  </div>
</template>
