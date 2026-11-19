<script setup lang="ts">
import { computed } from 'vue'
import { Info } from 'lucide-vue-next'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useLine } from '@/composables/useLine'
import { formatDateLong, formatInteger, formatMarginBps, formatTime, pluralRu } from '@/lib/format'

const { snapshot, loading } = useLine()

const matchCount = computed(() => snapshot.value?.matches.length ?? 0)
const outcomeCount = computed(
  () => snapshot.value?.matches.reduce((sum, match) => sum + match.outcomes.length, 0) ?? 0,
)
const margin = computed(() => formatMarginBps(snapshot.value?.marginBps ?? 500))

const marginExplanation = computed(
  () =>
    `Две линии (PulseBet ≈ 3% и SharpLine ≈ 6% маржи) де-вигуются, смешиваются 40/60 и получают маржу дома ${margin.value}. Цена округляется вниз до сотых.`,
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
          Линия <span class="text-sodium">дня</span>
        </h1>
        <p class="mt-3 max-w-[54ch] text-[12px] leading-relaxed text-muted-foreground sm:text-[13px]">
          Живая демо-линия футбольных матчей. Цены пересчитываются каждые три секунды, источники
          открыты.
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
          <Tooltip>
            <TooltipTrigger as-child>
              <button
                type="button"
                class="inline-flex items-center gap-1 rounded-[2px] text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:text-sodium focus-visible:outline focus-visible:outline-1 focus-visible:outline-offset-2 focus-visible:outline-sodium"
              >
                маржа дома {{ margin }}
                <Info class="size-3 text-faint" aria-hidden="true" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom" :side-offset="8" class="max-w-[280px] leading-relaxed">
              {{ marginExplanation }}
            </TooltipContent>
          </Tooltip>
          <p v-if="snapshot" class="mt-1.5 text-[11px] tabular text-faint">
            тик #{{ snapshot.seq }} · {{ formatTime(snapshot.updatedAt) }}
          </p>
          <Skeleton v-else-if="loading" class="mt-1.5 h-3 w-20 rounded-[2px]" />
        </div>
      </div>
    </div>

    <div class="mt-6 flex h-px items-center bg-line" aria-hidden="true">
      <span class="h-px w-16 bg-sodium" />
    </div>
  </section>
</template>
