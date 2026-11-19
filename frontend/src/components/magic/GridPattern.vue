<script setup lang="ts">
// Ported from Magic UI GridPattern (https://magicui.design/docs/components/grid-pattern).
import { useId } from 'vue'

withDefaults(
  defineProps<{
    width?: number
    height?: number
    x?: number
    y?: number
  }>(),
  {
    width: 64,
    height: 64,
    x: -1,
    y: -1,
  },
)

const id = useId()
</script>

<template>
  <svg aria-hidden="true" class="pointer-events-none absolute inset-0 h-full w-full">
    <defs>
      <pattern
        :id="id"
        :width="width"
        :height="height"
        patternUnits="userSpaceOnUse"
        :x="x"
        :y="y"
      >
        <path :d="`M ${width} 0 L 0 0 0 ${height}`" fill="none" stroke="currentColor" />
      </pattern>
    </defs>
    <rect width="100%" height="100%" :fill="`url(#${id})`" />
  </svg>
</template>
