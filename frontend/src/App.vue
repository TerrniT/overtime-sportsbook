<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { Toaster } from '@yusufceng1z/ding-vue'
import { TooltipProvider } from '@/components/ui/tooltip'
import Background from '@/components/Background.vue'
import AppHeader from '@/components/AppHeader.vue'
import LineMasthead from '@/components/LineMasthead.vue'
import MatchList from '@/components/MatchList.vue'
import BetsHistory from '@/components/BetsHistory.vue'
import BetSlip from '@/components/BetSlip.vue'
import MobileSlipBar from '@/components/MobileSlipBar.vue'
import OddsChangedDialog from '@/components/OddsChangedDialog.vue'
import { useBets } from '@/composables/useBets'
import { useLine } from '@/composables/useLine'
import { useSession } from '@/composables/useSession'

const { fetchSession } = useSession()
const { start, stop } = useLine()
const { fetchBets } = useBets()

onMounted(() => {
  void fetchSession()
  void fetchBets()
  start()
})

onBeforeUnmount(stop)
</script>

<template>
  <TooltipProvider :delay-duration="150">
    <a
      href="#line-title"
      class="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-50 focus:rounded-[4px] focus:bg-sodium focus:px-3 focus:py-2 focus:font-display focus:text-[13px] focus:uppercase focus:tracking-[0.08em] focus:text-ink"
    >
      К линии
    </a>

    <Background />

    <div class="relative z-10 flex min-h-svh flex-col">
      <AppHeader />

      <main class="mx-auto w-full max-w-[1440px] flex-1 px-4 pb-28 sm:px-6 lg:px-8 lg:pb-14">
        <LineMasthead />

        <div class="grid grid-cols-1 gap-8 lg:grid-cols-12 lg:gap-6 xl:gap-8">
          <div class="min-w-0 lg:col-span-8">
            <MatchList />
            <BetsHistory class="mt-8" />
          </div>

          <aside class="hidden lg:col-span-4 lg:block" aria-label="Купон">
            <div class="sticky top-[84px] ml-auto w-full max-w-[400px]">
              <BetSlip />
            </div>
          </aside>
        </div>

        <footer
          class="mt-10 flex flex-wrap items-center justify-between gap-x-6 gap-y-2 border-t border-line pt-4 pb-2 text-[10px] leading-relaxed text-faint"
        >
          <p>Все матчи, коэффициенты и баланс симулированы. Реальных ставок и выплат нет.</p>
          <p class="uppercase tracking-[0.14em]">Овертайм · демо-линия</p>
        </footer>
      </main>
    </div>

    <MobileSlipBar />
    <OddsChangedDialog />
    <Toaster theme="dark" position="bottom-right" />
  </TooltipProvider>
</template>
