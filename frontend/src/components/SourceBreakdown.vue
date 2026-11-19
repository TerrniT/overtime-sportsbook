<script setup lang="ts">
import { computed } from 'vue'
import { Info } from 'lucide-vue-next'
import type { Match } from '@/api/types'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatMarginBps, formatOdds } from '@/lib/format'

const props = defineProps<{
  match: Match
  marginBps: number
}>()

const margin = computed(() => formatMarginBps(props.marginBps))

const explanation = computed(
  () =>
    `PulseBet и SharpLine дают независимые цены. Сначала снимаем их маржу (де-виг), затем смешиваем 40/60 и добавляем маржу дома ${margin.value}. Наша цена округлена вниз до сотых.`,
)

function sourcePrice(outcomeIndex: number, sourceId: string): string {
  const outcome = props.match.outcomes[outcomeIndex]
  const source = outcome?.sources.find((item) => item.id === sourceId)
  return source ? formatOdds(source.priceMilli) : '—'
}
</script>

<template>
  <div class="overflow-hidden rounded-lg border border-line bg-[#0c0c0e]">
    <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-b border-line px-3 py-2">
      <span class="text-[9px] font-medium uppercase tracking-[0.16em] text-muted-foreground">
        Как собрана цена
      </span>
      <span class="tabular text-[10px] text-faint">де-виг 40/60 · маржа {{ margin }}</span>
    </div>

    <table class="w-full border-collapse text-left">
      <caption class="sr-only">Источники цены для {{ match.homeTeam }} — {{ match.awayTeam }}</caption>
      <thead>
        <tr class="text-[8px] uppercase tracking-[0.14em] text-faint">
          <th scope="col" class="px-3 py-1.5 font-medium">Исход</th>
          <th scope="col" class="px-2 py-1.5 font-medium">PulseBet</th>
          <th scope="col" class="px-2 py-1.5 font-medium">SharpLine</th>
          <th scope="col" class="px-2 py-1.5 text-right font-medium">
            <span class="inline-flex items-center gap-1">
              Наша линия
              <Tooltip>
                <TooltipTrigger as-child>
                  <button
                    type="button"
                    class="inline-flex rounded-[2px] text-faint outline-none hover:text-foreground focus-visible:text-sodium focus-visible:outline focus-visible:outline-1 focus-visible:outline-offset-2 focus-visible:outline-sodium"
                    aria-label="Как считается наша линия"
                  >
                    <Info class="size-2.5" aria-hidden="true" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top" :side-offset="6" class="max-w-[280px] leading-relaxed">
                  {{ explanation }}
                </TooltipContent>
              </Tooltip>
            </span>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(outcome, index) in match.outcomes"
          :key="outcome.key"
          class="border-t border-line/70"
        >
          <th scope="row" class="px-3 py-2 text-[11px] font-medium text-muted-foreground">
            {{ outcome.label }}
          </th>
          <td class="tabular px-2 py-2 font-display text-[14px] tracking-[0.02em] text-muted-foreground">
            {{ sourcePrice(index, 'pulse') }}
          </td>
          <td class="tabular px-2 py-2 font-display text-[14px] tracking-[0.02em] text-muted-foreground">
            {{ sourcePrice(index, 'sharp') }}
          </td>
          <td class="tabular px-2 py-2 text-right font-display text-[15px] tracking-[0.02em] text-foreground">
            {{ formatOdds(outcome.priceMilli) }}
          </td>
        </tr>
      </tbody>
    </table>

    <p class="border-t border-line/70 px-3 py-2 text-[10px] leading-relaxed text-faint">
      Источники симулированы. Наша линия — их консенсус плюс маржа дома, без обещания прибыли.
    </p>
  </div>
</template>
