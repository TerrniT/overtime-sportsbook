<script setup lang="ts">
import BalanceChip from './BalanceChip.vue'
import ConnectionStatus from './ConnectionStatus.vue'
</script>

<template>
  <header class="sticky top-0 z-40 border-b border-line bg-[#0a0a0b]">
    <div
      class="mx-auto flex h-14 w-full max-w-[1440px] items-center gap-2 px-4 sm:h-16 sm:gap-3 sm:px-6 lg:px-8"
    >
      <a
        href="/"
        class="flex shrink-0 items-center gap-2.5 outline-none focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-sodium"
        aria-label="Овертайм — на главную"
      >
        <span
          class="grid size-6 place-items-center rounded-[4px] bg-sodium font-display text-[18px] font-semibold leading-none text-ink"
          aria-hidden="true"
        >
          О
        </span>
        <span
          class="font-display text-[15px] font-semibold uppercase leading-none tracking-[0.18em] text-foreground min-[360px]:text-[16px] sm:text-[18px]"
        >
          Овертайм
        </span>
      </a>

      <div class="ml-auto flex min-w-0 items-center gap-2 sm:gap-4">
        <ConnectionStatus />
        <span class="hidden h-8 w-px bg-line sm:block" aria-hidden="true" />
        <BalanceChip />
        <span class="hidden h-8 w-px bg-line md:block" aria-hidden="true" />
        <div class="hidden items-center gap-2 md:flex">
          <span
            class="grid size-7 place-items-center rounded-[4px] border border-line bg-panel font-display text-[12px] uppercase leading-none text-muted-foreground"
            aria-hidden="true"
          >
            И
          </span>
          <span class="text-[10px] uppercase tracking-[0.14em] text-muted-foreground">Игрок</span>
        </div>
      </div>
    </div>
  </header>
</template>
