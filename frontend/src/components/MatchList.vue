<script setup lang="ts">
import { WifiOff } from 'lucide-vue-next'
import { Skeleton } from '@/components/ui/skeleton'
import { useLine } from '@/composables/useLine'
import { formatMarginBps } from '@/lib/format'
import EmptyState from './EmptyState.vue'
import ErrorPanel from './ErrorPanel.vue'
import MatchCard from './MatchCard.vue'

const { snapshot, loading, loadError, stale, refresh } = useLine()
</script>

<template>
  <section aria-labelledby="matches-heading">
    <h2 id="matches-heading" class="sr-only">Линия матчей</h2>

    <div
      v-if="stale && snapshot"
      class="mb-2.5 flex items-start gap-2.5 rounded-[8px] border border-[#452a2a] bg-[#180f10] px-3 py-2.5"
      role="status"
    >
      <WifiOff class="mt-0.5 size-3.5 shrink-0 text-[#f26d6d]" aria-hidden="true" />
      <p class="text-[11px] leading-relaxed text-[#c7a9a9]">
        Нет свежих цен больше 9 секунд. Ставки приостановлены: коэффициенты на экране могли
        устареть.
      </p>
    </div>

    <div
      v-if="loading && !snapshot"
      class="space-y-2.5 lg:space-y-0 lg:overflow-hidden lg:rounded-[10px] lg:border lg:border-line lg:bg-panel"
      aria-hidden="true"
    >
      <div
        v-for="index in 5"
        :key="index"
        class="rounded-[10px] border border-line bg-panel px-3.5 py-4 lg:grid lg:grid-cols-[64px_minmax(0,1fr)_288px] xl:grid-cols-[64px_minmax(0,1fr)_330px] lg:items-center lg:gap-5 lg:rounded-none lg:border-0 lg:border-b lg:border-line lg:px-5 lg:last:border-b-0"
      >
        <Skeleton class="h-4 w-10 rounded-[2px]" />
        <div class="mt-3 space-y-2 lg:mt-0">
          <Skeleton class="h-4 w-48 max-w-full rounded-[2px]" />
          <Skeleton class="h-2.5 w-28 rounded-[2px]" />
        </div>
        <div class="mt-3 grid grid-cols-3 gap-2 lg:mt-0">
          <Skeleton v-for="cell in 3" :key="cell" class="h-[58px] rounded-lg sm:h-[62px]" />
        </div>
      </div>
      <span class="sr-only">Загрузка линии</span>
    </div>

    <ErrorPanel
      v-else-if="loadError && !snapshot"
      title="Линия не загрузилась"
      :description="loadError"
      @retry="refresh"
    />

    <div
      v-else-if="snapshot"
      class="lg:overflow-hidden lg:rounded-[10px] lg:border lg:border-line lg:bg-panel"
    >
      <div
        class="hidden grid-cols-[64px_minmax(0,1fr)_330px] items-center gap-5 border-b border-line px-5 py-2 lg:grid"
      >
        <span class="text-[8px] font-medium uppercase tracking-[0.16em] text-faint">Начало</span>
        <span class="text-[8px] font-medium uppercase tracking-[0.16em] text-faint">Матч</span>
        <span class="text-right text-[8px] font-medium uppercase tracking-[0.16em] text-faint">
          Коэффициент 1X2
        </span>
      </div>

      <EmptyState
        v-if="snapshot.matches.length === 0"
        title="Линия пуста"
        description="Сервер вернул пустой список матчей. Загляните чуть позже."
      />

      <TransitionGroup
        v-else
        tag="div"
        name="row"
        class="space-y-2.5 lg:space-y-0 lg:divide-y lg:divide-line"
      >
        <MatchCard v-for="match in snapshot.matches" :key="match.id" :match="match" />
      </TransitionGroup>

      <div class="hidden items-center justify-between border-t border-line px-5 py-2.5 lg:flex">
        <span class="text-[9px] uppercase tracking-[0.14em] text-faint">
          Демо-линия · симуляция
        </span>
        <span class="text-[9px] uppercase tracking-[0.14em] text-faint">
          Тик #{{ snapshot.seq }} · маржа {{ formatMarginBps(snapshot.marginBps) }}
        </span>
      </div>
    </div>
  </section>
</template>
