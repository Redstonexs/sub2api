<template>
  <div data-testid="home-routing-diagram" class="w-full border border-gray-700 bg-dark-900 p-4 shadow-2xl shadow-black/20 sm:p-6">
    <div data-testid="home-routing-diagram-desktop" class="relative hidden min-h-[330px] lg:block">
      <svg viewBox="0 0 560 330" fill="none" class="absolute inset-0 h-full w-full" aria-hidden="true">
        <path d="M122 165H236" class="stroke-gray-600" stroke-width="2" />
        <path data-motion-route-line d="M322 165C376 165 390 65 448 65" class="stroke-primary-400" stroke-width="2" />
        <path data-motion-route-line d="M322 165H448" class="stroke-success-400" stroke-width="2" />
        <path data-motion-route-line d="M322 165C376 165 390 265 448 265" class="stroke-info-400" stroke-width="2" />
      </svg>

      <div data-motion-key class="absolute left-0 top-1/2 w-32 -translate-y-1/2 border border-gray-600 bg-gray-800 p-3 text-center">
        <Icon name="key" size="sm" class="mx-auto text-primary-300" />
        <p class="mt-2 font-mono text-[10px] uppercase tracking-[0.16em] text-gray-400">API key</p>
        <p class="mt-1 text-xs font-semibold text-white">{{ t('homeV2.diagramLabelKey') }}</p>
      </div>

      <div data-motion-gateway class="absolute left-[42%] top-1/2 flex h-24 w-24 -translate-x-1/2 -translate-y-1/2 flex-col items-center justify-center border border-primary-400 bg-gray-900 text-center shadow-lg shadow-primary-500/10">
        <Icon name="sync" size="md" class="text-primary-300" />
        <p class="mt-2 text-xs font-bold text-white">{{ t('homeV2.diagramLabelGateway') }}</p>
      </div>

      <div data-motion-provider class="absolute right-0 top-6 flex w-28 items-center gap-2 border border-gray-700 bg-gray-800 px-3 py-2">
        <span class="font-mono text-xs font-bold text-primary-300">C</span><span class="text-xs text-gray-200">Claude</span>
      </div>
      <div data-motion-provider class="absolute right-0 top-1/2 flex w-28 -translate-y-1/2 items-center gap-2 border border-gray-700 bg-gray-800 px-3 py-2">
        <span class="font-mono text-xs font-bold text-success-400">G</span><span class="text-xs text-gray-200">GPT</span>
      </div>
      <div data-motion-provider class="absolute bottom-6 right-0 flex w-28 items-center gap-2 border border-gray-700 bg-gray-800 px-3 py-2">
        <span class="font-mono text-xs font-bold text-info-400">G</span><span class="text-xs text-gray-200">Gemini</span>
      </div>
      <span data-motion-signal aria-hidden="true" class="absolute left-[23%] top-1/2 h-3 w-3 -translate-y-1/2 rounded-full bg-primary-300 shadow-lg shadow-primary-400/70"></span>
    </div>

    <div data-testid="home-routing-diagram-mobile" class="lg:hidden">
      <svg viewBox="0 0 340 108" fill="none" class="h-auto w-full" aria-hidden="true">
        <path d="M96 54H141" class="stroke-gray-600" stroke-width="2" />
        <path d="M199 54H244" class="stroke-primary-400" stroke-width="2" />
      </svg>
      <div class="-mt-[88px] grid grid-cols-[1fr_auto_1fr_auto_1fr] items-center gap-2 pb-7">
        <div class="border border-gray-600 bg-gray-800 px-2 py-3 text-center">
          <p class="font-mono text-[10px] text-primary-300">KEY</p>
          <p class="mt-1 text-xs font-semibold text-white">{{ t('homeV2.diagramLabelKey') }}</p>
        </div>
        <Icon name="arrowRight" size="sm" class="text-gray-500" />
        <div class="border border-primary-400 bg-gray-900 px-2 py-3 text-center">
          <p class="font-mono text-[10px] text-primary-300">ROUTE</p>
          <p class="mt-1 text-xs font-semibold text-white">{{ t('homeV2.diagramLabelGateway') }}</p>
        </div>
        <Icon name="arrowRight" size="sm" class="text-gray-500" />
        <div class="border border-gray-600 bg-gray-800 px-2 py-3 text-center">
          <p class="font-mono text-[10px] text-success-400">MODEL</p>
          <p class="mt-1 text-xs font-semibold text-white">{{ t('homeV2.diagramLabelProviders') }}</p>
        </div>
      </div>
    </div>

    <p class="mt-3 border-t border-gray-800 pt-3 text-center font-mono text-xs uppercase tracking-[0.16em] text-gray-400">
      <span class="mr-2 inline-block h-2 w-2 rounded-full bg-success-500"></span>{{ t('homeV2.diagramStatus') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
</script>
