<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronDown } from 'lucide-vue-next'
import type { Match } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { useBetSlip } from '@/composables/useBetSlip'
import { useLine } from '@/composables/useLine'
import { formatKickoffDay, formatKickoffFull, formatMatchLabel, formatTime } from '@/lib/format'
import OddsButton from './OddsButton.vue'
import SourceBreakdown from './SourceBreakdown.vue'

const props = defineProps<{ match: Match }>()

const { active, inFlight, hasUnresolvedAttempt, select } = useBetSlip()
const { stale } = useLine()

const expanded = ref(false)
const sourcesId = computed(() => `sources-${props.match.id}`)
const matchLabel = computed(() => formatMatchLabel(props.match))
const rowSelected = computed(() => active.value?.match.id === props.match.id)

function isSelected(outcomeKey: string): boolean {
  return active.value?.match.id === props.match.id && active.value.outcome.key === outcomeKey
}

const oddsDisabled = computed(() => stale.value || inFlight.value || hasUnresolvedAttempt.value)
</script>

<template>
  <article
    class="relative rounded-[10px] border border-line bg-panel px-3.5 py-3.5 lg:grid lg:grid-cols-[64px_minmax(0,1fr)_288px] xl:grid-cols-[64px_minmax(0,1fr)_330px] lg:items-center lg:gap-5 lg:rounded-none lg:border-0 lg:bg-transparent lg:px-5 lg:py-4"
    :class="rowSelected ? 'bg-sodium/[0.025]' : ''"
  >
    <div class="flex items-center justify-between gap-3 lg:block">
      <div class="flex items-baseline gap-2 lg:block">
        <span
          class="font-display text-[19px] leading-none tracking-[0.04em] text-foreground tabular"
          :title="formatKickoffFull(match.startsAt)"
        >
          {{ formatTime(match.startsAt) }}
        </span>
        <span class="text-[10px] uppercase tracking-[0.1em] text-faint lg:mt-1 lg:block">
          {{ formatKickoffDay(match.startsAt) }}
        </span>
      </div>
      <Badge
        variant="outline"
        :title="match.league"
        class="h-[18px] max-w-[140px] truncate rounded-[3px] border-line bg-transparent px-1.5 text-[8px] font-medium uppercase leading-none tracking-[0.08em] text-muted-foreground lg:mt-2 lg:max-w-[64px] lg:px-0.5 lg:tracking-[0.04em]"
      >
        {{ match.league }}
      </Badge>
    </div>

    <div class="mt-2 min-w-0 lg:mt-0">
      <h3
        class="text-balance break-words font-display text-[17px] font-medium uppercase leading-[1.2] tracking-[0.02em] text-foreground sm:text-[18px] lg:text-[19px]"
      >
        {{ match.homeTeam }}
        <span class="mx-0.5 text-faint" aria-hidden="true">—</span>
        {{ match.awayTeam }}
      </h3>

      <button
        type="button"
        class="mt-1.5 -my-1 inline-flex min-h-6 items-center gap-1 py-1 text-[10px] text-faint outline-none transition-colors hover:text-muted-foreground focus-visible:text-sodium"
        :aria-expanded="expanded"
        :aria-controls="sourcesId"
        @click="expanded = !expanded"
      >
        <span>Цены источников</span>
        <ChevronDown
          class="size-3 transition-transform duration-200"
          :class="expanded ? 'rotate-180' : ''"
          aria-hidden="true"
        />
      </button>
    </div>

    <div class="mt-3 grid grid-cols-3 gap-2 lg:mt-0">
      <OddsButton
        v-for="outcome in match.outcomes"
        :key="outcome.key"
        :outcome="outcome"
        :selected="isSelected(outcome.key)"
        :disabled="oddsDisabled"
        :match-label="matchLabel"
        @select="select(match, outcome)"
      />
    </div>

    <Transition name="sources">
      <div v-if="expanded" :id="sourcesId" class="mt-3 lg:col-span-3">
        <SourceBreakdown :match="match" />
      </div>
    </Transition>
  </article>
</template>
