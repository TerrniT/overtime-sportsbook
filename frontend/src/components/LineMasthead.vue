<script setup lang="ts">
import { computed } from 'vue'
import { Skeleton } from '@/components/ui/skeleton'
import { useLine } from '@/composables/useLine'
import { formatDateLong, formatInteger, formatTime, pluralRu } from '@/lib/format'

const { snapshot, loading } = useLine()

const matchCount = computed(() => snapshot.value?.matches.length ?? 0)
const outcomeCount = computed(
  () => snapshot.value?.matches.reduce((sum, match) => sum + match.outcomes.length, 0) ?? 0,
)
</script>

<template>
  <section class="pt-7 pb-5 sm:pt-10" aria-labelledby="line-title">
    <div class="flex flex-wrap items-end justify-between gap-5">
      <div class="min-w-0">
        <h1
          id="line-title"
          tabindex="-1"
          class="font-display text-[46px] font-medium uppercase leading-[0.9] tracking-[0.01em] text-foreground sm:text-[58px] xl:text-[66px]"
        >
          Линия
        </h1>
        <p class="mt-3 text-[12px] leading-relaxed text-muted-foreground sm:text-[13px]">
          Футбол · 1X2
        </p>
      </div>

      <div class="flex w-full flex-col items-start gap-3 min-[360px]:flex-row min-[360px]:items-end min-[360px]:justify-between min-[360px]:gap-6 sm:w-auto sm:justify-end sm:gap-8">
        <div class="text-left sm:text-right">
          <p class="font-display text-[13px] uppercase leading-none tracking-[0.12em] text-foreground">
            {{ formatDateLong() }}
          </p>
          <p v-if="snapshot" class="mt-1.5 text-[11px] text-faint">
            {{ formatInteger(matchCount) }}
            {{ pluralRu(matchCount, ['матч', 'матча', 'матчей']) }} ·
            {{ formatInteger(outcomeCount) }}
            {{ pluralRu(outcomeCount, ['исход', 'исхода', 'исходов']) }}
          </p>
          <Skeleton v-else class="mt-1.5 h-3 w-28 rounded-[2px]" />
        </div>

        <div class="w-full border-t border-line pt-2.5 text-left min-[360px]:w-auto min-[360px]:border-t-0 min-[360px]:border-l min-[360px]:pt-0 min-[360px]:pl-4 min-[360px]:text-right sm:pl-6">
          <p class="text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
            Обновлено
          </p>
          <p v-if="snapshot" class="mt-1.5 text-[11px] tabular text-faint">
            {{ formatTime(snapshot.updatedAt) }}
          </p>
          <Skeleton v-else-if="loading" class="mt-1.5 h-3 w-12 rounded-[2px]" />
        </div>
      </div>
    </div>

    <div class="mt-6 flex h-px items-center bg-line" aria-hidden="true">
      <span class="h-px w-16 bg-sodium" />
    </div>
  </section>
</template>
