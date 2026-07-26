<template>
  <section data-motion-relay class="border-b border-gray-200 bg-gray-100 py-16 dark:border-dark-800 dark:bg-dark-900 sm:py-20">
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <div data-motion-reveal class="max-w-2xl">
        <p class="font-mono text-xs uppercase tracking-[0.2em] text-primary-700 dark:text-primary-300">{{ t('homeV2.relayKicker') }}</p>
        <h2 class="mt-4 font-serif text-3xl font-bold text-gray-900 dark:text-white sm:text-4xl">{{ t('homeV2.relayTitle') }}</h2>
        <p class="mt-4 text-base leading-7 text-gray-600 dark:text-gray-300">{{ t('homeV2.relayDescription') }}</p>
      </div>

      <div data-motion-relay-static class="mt-10 grid gap-4 lg:grid-cols-3">
        <article v-for="stage in stages" :key="stage.number" class="border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-950">
          <p class="font-mono text-xs text-primary-700 dark:text-primary-300">{{ stage.number }} / {{ t(stage.eyebrow) }}</p>
          <h3 class="mt-5 text-xl font-semibold text-gray-900 dark:text-white">{{ t(stage.title) }}</h3>
          <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t(stage.description) }}</p>
          <code class="mt-6 block border-l-2 border-primary-500 bg-gray-100 px-3 py-2 font-mono text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-200">{{ stage.detail }}</code>
        </article>
      </div>

      <div data-motion-relay-animated class="hidden overflow-hidden border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-950">
        <div data-motion-relay-track class="relative min-h-[390px]">
          <article v-for="stage in stages" :key="`motion-${stage.number}`" data-motion-relay-panel class="absolute inset-0 p-8 lg:p-12">
            <p class="font-mono text-xs text-primary-700 dark:text-primary-300">{{ stage.number }} / {{ t(stage.eyebrow) }}</p>
            <h3 class="mt-8 max-w-xl font-serif text-4xl font-bold text-gray-900 dark:text-white">{{ t(stage.title) }}</h3>
            <p class="mt-5 max-w-xl text-base leading-7 text-gray-600 dark:text-gray-300">{{ t(stage.description) }}</p>
            <code class="mt-10 block max-w-xl border-l-2 border-primary-500 bg-gray-100 px-4 py-3 font-mono text-sm text-gray-700 dark:bg-dark-900 dark:text-gray-200">{{ stage.detail }}</code>
            <span data-motion-relay-meter class="mt-12 block h-1 origin-left bg-primary-500"></span>
          </article>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const stages = [
  {
    number: '01',
    eyebrow: 'homeV2.relayRequestEyebrow',
    title: 'homeV2.relayRequestTitle',
    description: 'homeV2.relayRequestDescription',
    detail: 'POST /v1/messages'
  },
  {
    number: '02',
    eyebrow: 'homeV2.relayRouteEyebrow',
    title: 'homeV2.relayRouteTitle',
    description: 'homeV2.relayRouteDescription',
    detail: 'health -> capacity -> sticky session'
  },
  {
    number: '03',
    eyebrow: 'homeV2.relayReceiptEyebrow',
    title: 'homeV2.relayReceiptTitle',
    description: 'homeV2.relayReceiptDescription',
    detail: 'usage -> quota -> balance'
  }
]
</script>
