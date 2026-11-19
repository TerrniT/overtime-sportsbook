<script setup lang="ts">
// Ported from Magic UI NumberTicker (https://magicui.design/docs/components/number-ticker).
// No React or motion dependency: a small rAF ease-out, reduced-motion aware.
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    value: number
    duration?: number
    format?: (value: number) => string
  }>(),
  {
    duration: 650,
  },
)

const display = ref(props.value)
let frame: number | null = null

const prefersReducedMotion = (): boolean =>
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

function cancel(): void {
  if (frame !== null) {
    cancelAnimationFrame(frame)
    frame = null
  }
}

function animateTo(target: number): void {
  cancel()

  if (prefersReducedMotion() || props.duration <= 0) {
    display.value = target
    return
  }

  const from = display.value
  const delta = target - from
  if (delta === 0) return

  const started = performance.now()
  const tick = (now: number) => {
    const progress = Math.min(1, (now - started) / props.duration)
    const eased = 1 - Math.pow(1 - progress, 3)
    display.value = from + delta * eased

    if (progress < 1) {
      frame = requestAnimationFrame(tick)
    } else {
      display.value = target
      frame = null
    }
  }

  frame = requestAnimationFrame(tick)
}

onMounted(() => {
  display.value = props.value
})

watch(
  () => props.value,
  (next) => animateTo(next),
)

onBeforeUnmount(cancel)

const output = computed(() =>
  props.format ? props.format(display.value) : Math.round(display.value).toLocaleString('ru-RU'),
)
</script>

<template>
  <span class="tabular"><slot :value="output">{{ output }}</slot></span>
</template>
