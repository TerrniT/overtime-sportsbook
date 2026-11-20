<script setup lang="ts">
import type { Match } from '@/api/types'
import { formatMatchLabel, formatOdds } from '@/lib/format'

const props = defineProps<{
  match: Match
}>()

function sourcePrice(outcomeIndex: number, sourceId: string): string {
  const outcome = props.match.outcomes[outcomeIndex]
  const source = outcome?.sources.find((item) => item.id === sourceId)
  return source ? formatOdds(source.priceMilli) : '—'
}
</script>

<template>
  <div class="overflow-hidden rounded-lg border border-line bg-[#0c0c0e]">
    <div class="border-b border-line px-3 py-2">
      <span class="text-[9px] font-medium uppercase tracking-[0.16em] text-muted-foreground">
        Цены источников
      </span>
    </div>

    <table class="w-full border-collapse text-left">
      <caption class="sr-only">Цены источников для {{ formatMatchLabel(match) }}</caption>
      <thead>
        <tr class="text-[8px] uppercase tracking-[0.14em] text-faint">
          <th scope="col" class="px-3 py-1.5 font-medium">Исход</th>
          <th scope="col" class="px-2 py-1.5 font-medium">PulseBet</th>
          <th scope="col" class="px-2 py-1.5 font-medium">SharpLine</th>
          <th scope="col" class="px-2 py-1.5 text-right font-medium">Наша цена</th>
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
      Источники цен симулированы.
    </p>
  </div>
</template>
