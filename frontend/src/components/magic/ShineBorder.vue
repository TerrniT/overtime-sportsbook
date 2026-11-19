<script setup lang="ts">
// Ported from Magic UI ShineBorder (https://magicui.design/docs/components/shine-border).
// Rendered as an absolute overlay; the parent supplies position/radius.
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    active?: boolean
    borderWidth?: number
    duration?: number
    color?: string[]
  }>(),
  {
    active: true,
    borderWidth: 1,
    duration: 8,
    color: () => ['#ff6a2a', '#ffb38a', '#ff6a2a'],
  },
)

const style = computed<Record<string, string>>(() => ({
  '--border-width': `${props.borderWidth}px`,
  '--duration': `${props.duration}s`,
  backgroundImage: `radial-gradient(transparent, transparent, ${props.color.join(',')}, transparent, transparent)`,
  backgroundSize: '300% 300%',
  mask: 'linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)',
  maskComposite: 'exclude',
  WebkitMaskComposite: 'xor',
  padding: `${props.borderWidth}px`,
}))
</script>

<template>
  <div
    v-if="active"
    aria-hidden="true"
    class="shine-border pointer-events-none absolute inset-0 z-20 overflow-hidden rounded-[inherit]"
    :style="style"
  />
</template>

<style scoped>
.shine-border {
  animation: shine var(--duration) infinite linear;
}

@keyframes shine {
  0% {
    background-position: 0% 0%;
  }
  50% {
    background-position: 100% 100%;
  }
  100% {
    background-position: 0% 0%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .shine-border {
    animation: none;
  }
}
</style>
