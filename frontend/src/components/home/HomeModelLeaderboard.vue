<template>
  <section
    data-testid="home-leaderboard"
    class="border-b border-gray-200 bg-white py-16 dark:border-dark-800 dark:bg-dark-950 sm:py-20"
  >
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="flex flex-col justify-between gap-6 lg:flex-row lg:items-end">
        <div data-motion-reveal class="max-w-2xl">
          <p class="font-mono text-xs uppercase tracking-[0.2em] text-primary-700 dark:text-primary-300">
            <Icon name="trophy" size="xs" class="mr-2 inline-block align-[-2px]" />{{ t('homeV2.leaderboardKicker') }}
          </p>
          <h2 class="mt-4 font-serif text-3xl font-bold text-gray-900 dark:text-white sm:text-4xl">
            {{ t('homeV2.leaderboardTitle') }}
          </h2>
          <p class="mt-4 text-base leading-7 text-gray-600 dark:text-gray-300">
            {{ t('homeV2.leaderboardDescription') }}
          </p>
        </div>

        <a
          data-testid="home-leaderboard-source"
          :href="sourceUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex w-fit shrink-0 items-center gap-2 border border-gray-300 px-4 py-2.5 font-mono text-xs text-gray-600 transition-colors hover:border-primary-500 hover:text-primary-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-dark-700 dark:text-gray-300 dark:hover:border-primary-400 dark:hover:text-primary-300"
        >
          {{ t('homeV2.leaderboardViewLive') }}
          <Icon name="externalLink" size="xs" />
        </a>
      </div>

      <dl data-motion-reveal class="mt-10 grid gap-px border border-gray-200 bg-gray-200 dark:border-dark-800 dark:bg-dark-800 sm:grid-cols-3">
        <div class="bg-gray-50 p-5 dark:bg-dark-900">
          <dt class="font-mono text-xs uppercase tracking-[0.15em] text-gray-500 dark:text-gray-400">
            {{ t('homeV2.leaderboardStatLeaderLabel') }}
          </dt>
          <dd class="mt-3 text-xl font-semibold text-gray-900 dark:text-white">{{ leader.model }}</dd>
          <dd class="mt-1 font-mono text-sm text-primary-700 dark:text-primary-300">
            {{ leader.score }} {{ t('homeV2.leaderboardScoreUnit') }}
          </dd>
        </div>
        <div class="bg-gray-50 p-5 dark:bg-dark-900">
          <dt class="font-mono text-xs uppercase tracking-[0.15em] text-gray-500 dark:text-gray-400">
            {{ t('homeV2.leaderboardStatOpenAiLabel') }}
          </dt>
          <dd class="mt-3 text-xl font-semibold text-gray-900 dark:text-white">{{ topOpenAi.model }}</dd>
          <dd class="mt-1 font-mono text-sm text-success-700 dark:text-success-400">
            {{ topOpenAi.score }} {{ t('homeV2.leaderboardScoreUnit') }}
          </dd>
        </div>
        <div class="bg-gray-50 p-5 dark:bg-dark-900">
          <dt class="font-mono text-xs uppercase tracking-[0.15em] text-gray-500 dark:text-gray-400">
            {{ t('homeV2.leaderboardStatEvalsLabel') }}
          </dt>
          <dd class="mt-3 text-xl font-semibold text-gray-900 dark:text-white">{{ evaluationCount }}</dd>
          <dd class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">{{ evaluationNames }}</dd>
        </div>
      </dl>

      <ol data-motion-reveal class="mt-8 border border-gray-200 dark:border-dark-800">
        <!-- Narrow: rank / model / score on one row, meter spanning the row below.
             sm and up: all four cells share a single row. -->
        <li
          v-for="entry in entries"
          :key="entry.model"
          data-testid="home-leaderboard-row"
          class="grid grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-x-4 gap-y-3 border-b border-gray-200 p-4 last:border-b-0 sm:grid-cols-[2rem_minmax(0,1fr)_14rem_auto] sm:gap-x-6 sm:px-6 dark:border-dark-800"
        >
          <span class="font-mono text-xs text-gray-500 dark:text-gray-400">
            {{ String(entry.rank).padStart(2, '0') }}
          </span>

          <span class="min-w-0">
            <span class="block truncate text-base font-semibold text-gray-900 dark:text-white">{{ entry.model }}</span>
            <span class="mt-0.5 block font-mono text-xs" :class="ACCENT_TEXT[entry.accent]">{{ entry.creator }}</span>
          </span>

          <span class="text-right font-mono text-sm font-semibold text-gray-900 sm:col-start-4 dark:text-white">
            {{ entry.score }}
          </span>

          <span
            aria-hidden="true"
            class="col-span-2 col-start-2 row-start-2 h-1.5 bg-gray-100 sm:col-span-1 sm:col-start-3 sm:row-start-1 dark:bg-dark-800"
          >
            <span class="block h-full" :class="ACCENT_BAR[entry.accent]" :style="{ width: `${entry.score}%` }"></span>
          </span>
        </li>
      </ol>

      <p class="mt-5 max-w-3xl font-mono text-xs leading-5 text-gray-500 dark:text-gray-400">
        {{ t('homeV2.leaderboardScoreLabel') }} · {{ t('homeV2.leaderboardSnapshot', { date: asOf }) }} ·
        {{ t('homeV2.leaderboardNote') }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  LEADERBOARD_AS_OF,
  LEADERBOARD_EVALUATIONS,
  LEADERBOARD_SOURCE_URL,
  MODEL_LEADERBOARD,
  type LeaderboardAccent,
} from '@/constants/modelLeaderboard'

// Tailwind needs whole class names in the source, so keep these as full literals.
const ACCENT_TEXT: Record<LeaderboardAccent, string> = {
  claude: 'text-primary-700 dark:text-primary-300',
  gpt: 'text-success-700 dark:text-success-400',
  gemini: 'text-info-700 dark:text-info-400',
  grok: 'text-warning-700 dark:text-warning-400',
  neutral: 'text-gray-600 dark:text-gray-400',
}

const ACCENT_BAR: Record<LeaderboardAccent, string> = {
  claude: 'bg-primary-600 dark:bg-primary-400',
  gpt: 'bg-success-600 dark:bg-success-400',
  gemini: 'bg-info-600 dark:bg-info-400',
  grok: 'bg-warning-600 dark:bg-warning-400',
  neutral: 'bg-gray-400 dark:bg-gray-500',
}

const { t } = useI18n()

const entries = MODEL_LEADERBOARD
const sourceUrl = LEADERBOARD_SOURCE_URL
const asOf = LEADERBOARD_AS_OF
const evaluationCount = LEADERBOARD_EVALUATIONS.length
const evaluationNames = LEADERBOARD_EVALUATIONS.join(' · ')

const leader = computed(() => entries[0])
const topOpenAi = computed(() => entries.find(entry => entry.accent === 'gpt') ?? entries[0])
</script>
