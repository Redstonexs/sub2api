<template>
  <main ref="storyRoot" data-testid="home-motion-story">
    <section data-testid="home-hero" class="relative overflow-hidden border-b border-gray-800 bg-gray-900 text-gray-100 dark:bg-dark-950">
      <div aria-hidden="true" class="pointer-events-none absolute inset-0 opacity-30">
        <div class="absolute inset-x-0 top-20 border-t border-primary-500/30"></div>
        <div class="absolute inset-x-0 bottom-16 border-t border-success-500/20"></div>
        <div class="absolute bottom-0 left-1/4 top-0 border-l border-gray-700"></div>
        <div class="absolute bottom-0 right-1/4 top-0 border-l border-gray-700"></div>
      </div>

      <div class="relative mx-auto max-w-7xl px-4 py-14 sm:px-6 sm:py-16 lg:px-8 lg:py-20">
        <div class="grid items-center gap-12 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] lg:gap-16">
          <div class="max-w-2xl">
            <p data-motion-reveal class="font-mono text-xs uppercase tracking-[0.2em] text-primary-300">
              <span class="mr-2 inline-block h-2 w-2 rounded-full bg-success-500"></span>{{ t('homeV2.eyebrow') }} / {{ siteName }}
            </p>
            <h1 data-motion-reveal class="mt-5 font-serif text-5xl font-bold leading-[1.02] text-white sm:text-6xl lg:text-7xl">
              {{ t('homeV2.title') }}
            </h1>
            <p data-motion-reveal class="mt-6 max-w-xl text-base leading-7 text-gray-300 sm:text-lg">
              {{ t('homeV2.subtitle') }}
            </p>
            <div data-motion-reveal class="mt-8 flex flex-wrap items-center gap-3">
              <router-link
                data-testid="home-cta-start"
                :to="dashboardOrLogin"
                class="inline-flex items-center gap-2 rounded-lg bg-primary-600 px-5 py-3 text-sm font-semibold text-white transition-colors hover:bg-primary-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-300"
              >
                {{ isAuthenticated ? t('homeV2.ctaDashboard') : t('homeV2.ctaStart') }}
                <Icon name="arrowRight" size="sm" :stroke-width="2" />
              </router-link>
              <a
                v-if="docUrl"
                data-testid="home-cta-docs"
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-2 rounded-lg border border-gray-600 px-5 py-3 text-sm font-semibold text-gray-100 transition-colors hover:border-primary-400 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-300"
              >
                {{ t('homeV2.ctaDocs') }}
                <Icon name="externalLink" size="sm" />
              </a>
            </div>
            <div data-motion-reveal class="mt-10 grid grid-cols-3 border-t border-gray-700 pt-5">
              <div>
                <p class="font-mono text-xs text-primary-300">01</p>
                <p class="mt-1 text-sm font-semibold text-white">{{ t('homeV2.relayRequestEyebrow') }}</p>
              </div>
              <div class="border-l border-gray-700 pl-4">
                <p class="font-mono text-xs text-success-400">02</p>
                <p class="mt-1 text-sm font-semibold text-white">{{ t('homeV2.relayRouteEyebrow') }}</p>
              </div>
              <div class="border-l border-gray-700 pl-4">
                <p class="font-mono text-xs text-info-400">03</p>
                <p class="mt-1 text-sm font-semibold text-white">{{ t('homeV2.relayReceiptEyebrow') }}</p>
              </div>
            </div>
          </div>

          <div data-motion-reveal class="lg:pl-4">
            <HomeRoutingDiagram />
          </div>
        </div>
      </div>
    </section>

    <HomeRelayStage />
    <HomeModelLeaderboard />
    <HomeFeatureStory :dashboard-path="dashboardPath" :is-authenticated="isAuthenticated" />
  </main>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeFeatureStory from './HomeFeatureStory.vue'
import HomeModelLeaderboard from './HomeModelLeaderboard.vue'
import HomeRelayStage from './HomeRelayStage.vue'
import HomeRoutingDiagram from './HomeRoutingDiagram.vue'
import Icon from '@/components/icons/Icon.vue'
import { useHomeMotion } from '@/composables/useHomeMotion'

const props = defineProps<{
  dashboardPath: string
  docUrl: string
  isAuthenticated: boolean
  siteName: string
}>()

const { t } = useI18n()
const storyRoot = ref<HTMLElement | null>(null)
const dashboardOrLogin = computed(() => props.isAuthenticated ? props.dashboardPath : '/login')

useHomeMotion(storyRoot)
</script>
