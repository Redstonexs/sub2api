<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      data-testid="home-iframe-override"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
      referrerpolicy="no-referrer"
      allowfullscreen
    ></iframe>
    <div v-else data-testid="home-html-override" v-html="sanitizedHomeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative min-h-screen bg-gray-50 text-gray-900 blueprint-bg dark:bg-dark-950 dark:text-gray-200"
  >
    <header
      data-testid="home-nav"
      class="sticky top-0 z-40 border-b border-gray-200/80 bg-gray-50/90 backdrop-blur dark:border-dark-800 dark:bg-dark-950/90"
    >
      <nav class="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
        <div class="flex min-w-0 items-center gap-2.5">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white ring-1 ring-primary-500/40 dark:bg-dark-900">
            <img :src="siteLogo || '/logo.svg'" alt="" class="h-full w-full object-contain" />
          </div>
          <span class="truncate text-sm font-semibold">{{ siteName }}</span>
        </div>

        <div class="flex items-center gap-1.5 sm:gap-2">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-lg p-2.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-gray-200"
            :aria-label="t('homeV2.navDocs')"
          >
            <Icon name="book" size="sm" />
          </a>

          <span data-testid="home-nav-locale"><LocaleSwitcher /></span>

          <button
            data-testid="home-nav-theme"
            class="rounded-lg p-2.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-gray-200"
            :aria-label="t(isDark ? 'homeV2.themeToLight' : 'homeV2.themeToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="sm" />
            <Icon v-else name="moon" size="sm" />
          </button>

          <router-link
            data-testid="home-nav-auth"
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex items-center gap-2 rounded-lg bg-primary-600 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-dark-950 sm:px-4"
          >
            <template v-if="isAuthenticated">
              <span class="flex h-5 w-5 items-center justify-center rounded-full bg-white/20 text-xs font-bold">
                {{ userInitial }}
              </span>
              <span class="hidden sm:inline">{{ t('homeV2.navDashboard') }}</span>
            </template>
            <span v-else>{{ t('homeV2.navLogin') }}</span>
          </router-link>
        </div>
      </nav>
    </header>

    <HomeMotionStory
      :dashboard-path="dashboardPath"
      :doc-url="docUrl"
      :is-authenticated="isAuthenticated"
      :site-name="siteName"
    />

    <footer data-testid="home-footer" class="border-t border-gray-200 dark:border-dark-800">
      <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 px-4 py-8 sm:flex-row sm:px-6 lg:px-8">
        <p class="font-mono text-xs text-gray-500 dark:text-gray-400">
          &copy; {{ currentYear }} {{ siteName }} {{ t('homeV2.footerRights') }}
        </p>
        <div class="flex items-center gap-5">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="font-mono text-xs text-gray-500 transition-colors hover:text-primary-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:text-primary-300"
          >
            {{ t('homeV2.footerDocs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="font-mono text-xs text-gray-500 transition-colors hover:text-primary-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:text-primary-300"
          >
            {{ t('homeV2.footerGithub') }}
          </a>
        </div>
      </div>
      <div class="mx-auto max-w-7xl px-4 pb-6 sm:px-6 lg:px-8">
        <p class="text-center font-mono text-xs text-gray-400 dark:text-gray-500">
          {{ t('disclaimer.independentService', { siteName }) }}
        </p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import DOMPurify from 'dompurify'
import HomeMotionStory from '@/components/home/HomeMotionStory.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useTheme } from '@/composables/useTheme'
import { useAuthStore, useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const isHomeContentUrl = computed(() => /^https?:\/\//.test(homeContent.value.trim()))
const sanitizedHomeContent = computed(() =>
  DOMPurify.sanitize(homeContent.value, {
    ADD_TAGS: ['iframe'],
    ADD_ATTR: ['src', 'sandbox', 'allowfullscreen', 'referrerpolicy', 'allow', 'name', 'loading', 'frameborder'],
  })
)

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => authStore.user?.email?.charAt(0).toUpperCase() || '')
const currentYear = computed(() => new Date().getFullYear())
const githubUrl = 'https://github.com/Redstonexs/sub2api'
const { isDark, toggleTheme } = useTheme()

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.cachedPublicSettings) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.blueprint-bg {
  background-image: radial-gradient(circle, rgba(176, 92, 64, 0.06) 1px, transparent 1px);
  background-size: 24px 24px;
}

:deep(.dark) .blueprint-bg {
  background-image: radial-gradient(circle, rgba(142, 211, 129, 0.08) 1px, transparent 1px);
}
</style>
