<template>
  <div>
    <section data-motion-reveal class="mx-auto max-w-7xl px-4 py-16 sm:px-6 sm:py-20 lg:px-8">
      <div class="flex flex-col justify-between gap-6 border-b border-gray-200 pb-8 dark:border-dark-800 sm:flex-row sm:items-end">
        <div class="max-w-2xl">
          <p class="font-mono text-xs uppercase tracking-[0.2em] text-primary-700 dark:text-primary-300">{{ t('homeV2.capabilityKicker') }}</p>
          <h2 class="mt-4 font-serif text-3xl font-bold text-gray-900 dark:text-white sm:text-4xl">{{ t('homeV2.capabilityTitle') }}</h2>
        </div>
        <p class="max-w-sm text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('homeV2.capabilityDescription') }}</p>
      </div>

      <div class="mt-8 grid border border-gray-200 dark:border-dark-800 md:grid-cols-2">
        <article class="border-b border-gray-200 p-6 dark:border-dark-800 md:border-r">
          <Icon name="server" size="md" class="text-primary-600 dark:text-primary-300" />
          <h3 class="mt-6 text-xl font-semibold text-gray-900 dark:text-white">{{ t('homeV2.featGatewayTitle') }}</h3>
          <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('homeV2.featGatewayDesc') }}</p>
        </article>
        <article class="border-b border-gray-200 p-6 dark:border-dark-800">
          <Icon name="cpu" size="md" class="text-success-600 dark:text-success-400" />
          <h3 class="mt-6 text-xl font-semibold text-gray-900 dark:text-white">{{ t('homeV2.featSchedulingTitle') }}</h3>
          <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('homeV2.featSchedulingDesc') }}</p>
        </article>
        <article class="border-b border-gray-200 p-6 dark:border-dark-800 md:border-b-0 md:border-r">
          <Icon name="chart" size="md" class="text-info-600 dark:text-info-400" />
          <h3 class="mt-6 text-xl font-semibold text-gray-900 dark:text-white">{{ t('homeV2.featBillingTitle') }}</h3>
          <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('homeV2.featBillingDesc') }}</p>
        </article>
        <article class="p-6">
          <Icon name="shield" size="md" class="text-primary-600 dark:text-primary-300" />
          <h3 class="mt-6 text-xl font-semibold text-gray-900 dark:text-white">{{ t('homeV2.featStickyTitle') }}</h3>
          <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('homeV2.featStickyDesc') }}</p>
        </article>
      </div>
    </section>

    <section data-testid="home-quickstart" data-motion-reveal class="border-y border-gray-200 bg-gray-900 py-16 text-gray-100 dark:border-dark-800 dark:bg-dark-950 sm:py-20">
      <div class="mx-auto grid max-w-7xl gap-10 px-4 sm:px-6 lg:grid-cols-[0.75fr_1.25fr] lg:px-8">
        <div>
          <p class="font-mono text-xs uppercase tracking-[0.2em] text-primary-300">{{ t('homeV2.quickstartTitle') }}</p>
          <h2 class="mt-4 font-serif text-3xl font-bold text-white sm:text-4xl">{{ t('homeV2.quickstartStep1') }}</h2>
          <p class="mt-4 max-w-md text-sm leading-6 text-gray-300">{{ t('homeV2.quickstartDescription') }}</p>
        </div>
        <ol class="border border-gray-700">
          <li class="border-b border-gray-700 p-5">
            <span class="font-mono text-xs text-primary-300">01</span>
            <p class="mt-2 text-sm font-medium text-white">{{ t('homeV2.quickstartStep1') }}</p>
          </li>
          <li class="border-b border-gray-700 p-5">
            <span class="font-mono text-xs text-primary-300">02</span>
            <p class="mt-2 text-sm font-medium text-white">{{ t('homeV2.quickstartStep2') }}</p>
            <code class="mt-4 block overflow-x-auto border-l-2 border-success-400 bg-dark-900 px-3 py-3 font-mono text-xs text-gray-200">ANTHROPIC_BASE_URL=https://your-gateway.example.com</code>
          </li>
          <li class="p-5">
            <span class="font-mono text-xs text-primary-300">03</span>
            <p class="mt-2 text-sm font-medium text-white">{{ t('homeV2.quickstartStep3') }}</p>
            <code class="mt-4 block overflow-x-auto border-l-2 border-info-400 bg-dark-900 px-3 py-3 font-mono text-xs text-gray-200">POST $ANTHROPIC_BASE_URL/v1/messages</code>
          </li>
        </ol>
      </div>
    </section>

    <section data-testid="home-providers" data-motion-reveal class="mx-auto max-w-7xl px-4 py-16 sm:px-6 sm:py-20 lg:px-8">
      <div class="grid gap-8 border-y border-gray-200 py-8 dark:border-dark-800 lg:grid-cols-[0.65fr_1.35fr] lg:items-center">
        <div>
          <p class="font-mono text-xs uppercase tracking-[0.2em] text-primary-700 dark:text-primary-300">{{ t('homeV2.providersTitle') }}</p>
          <p class="mt-4 max-w-sm text-base leading-7 text-gray-600 dark:text-gray-300">{{ t('homeV2.providersDesc') }}</p>
        </div>
        <div class="grid grid-cols-2 gap-px border border-gray-200 bg-gray-200 dark:border-dark-800 dark:bg-dark-800 sm:grid-cols-4">
          <div class="bg-gray-50 p-4 dark:bg-dark-950"><span class="font-mono text-sm text-primary-700 dark:text-primary-300">C / Claude</span></div>
          <div class="bg-gray-50 p-4 dark:bg-dark-950"><span class="font-mono text-sm text-success-700 dark:text-success-400">G / GPT</span></div>
          <div class="bg-gray-50 p-4 dark:bg-dark-950"><span class="font-mono text-sm text-info-700 dark:text-info-400">G / Gemini</span></div>
          <div class="bg-gray-50 p-4 dark:bg-dark-950"><span class="font-mono text-sm text-gray-600 dark:text-gray-300">+ / {{ t('homeV2.providerMore') }}</span></div>
        </div>
      </div>
    </section>

    <section data-testid="home-cta-band" data-motion-reveal class="border-y border-primary-800 bg-primary-700 py-16 text-white sm:py-20">
      <div class="mx-auto flex max-w-7xl flex-col gap-6 px-4 sm:px-6 lg:flex-row lg:items-end lg:justify-between lg:px-8">
        <div class="max-w-2xl">
          <p class="font-mono text-xs uppercase tracking-[0.2em] text-primary-100">{{ t('homeV2.ctaKicker') }}</p>
          <h2 class="mt-4 font-serif text-3xl font-bold sm:text-4xl">{{ t('homeV2.ctaBandTitle') }}</h2>
          <p class="mt-4 text-base leading-7 text-primary-100">{{ t('homeV2.ctaBandDesc') }}</p>
        </div>
        <router-link
          :to="dashboardOrLogin"
          class="inline-flex w-fit items-center gap-2 rounded-lg bg-white px-5 py-3 text-sm font-semibold text-primary-800 transition-colors hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white"
        >
          {{ t('homeV2.ctaBandBtn') }}
          <Icon name="arrowRight" size="sm" :stroke-width="2" />
        </router-link>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  dashboardPath: string
  isAuthenticated: boolean
}>()

const { t } = useI18n()
const dashboardOrLogin = computed(() => props.isAuthenticated ? props.dashboardPath : '/login')
</script>
