<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      data-testid="home-iframe-override"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else data-testid="home-html-override" v-html="homeContent"></div>
  </div>

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
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
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
const isHomeContentUrl = computed(() => /^https?:\/\//.test(homeContent.value.trim()))
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
