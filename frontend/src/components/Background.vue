<script setup lang="ts">
import GridPattern from './magic/GridPattern.vue'
</script>

<template>
  <div aria-hidden="true" class="pointer-events-none fixed inset-0 z-0 overflow-hidden">
    <GridPattern
      :width="64"
      :height="64"
      class="text-white/[0.05] [mask-image:radial-gradient(ellipse_75%_60%_at_50%_0%,black_10%,transparent_100%)]"
    />
    <!-- One sodium wash, like a floodlight above the stand. Nothing else glows. -->
    <div
      class="absolute -top-48 right-[-12%] h-[480px] w-[640px] rounded-full bg-sodium/[0.07] blur-[120px]"
    />
    <div class="absolute inset-x-0 top-0 h-px bg-white/[0.06]" />
  </div>
</template>
