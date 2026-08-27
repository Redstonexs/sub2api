<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { onMounted, onBeforeUnmount, watch } from 'vue'
import Toast from '@/components/common/Toast.vue'
import ConfirmDialogHost from '@/components/common/ConfirmDialogHost.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import AdminComplianceDialog from '@/components/admin/AdminComplianceDialog.vue'
import { resolveRouteDocumentTitle } from '@/router/title'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import { useAppStore, useAuthStore, useSubscriptionStore, useAnnouncementStore, useAdminComplianceStore, useAdminSettingsStore } from '@/stores'
import { getSetupStatus } from '@/api/setup'
import { updateFavicon } from '@/utils/branding'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()
const announcementStore = useAnnouncementStore()
const adminComplianceStore = useAdminComplianceStore()
const adminSettingsStore = useAdminSettingsStore()

function updateDocumentTitle() {
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  document.title = resolveRouteDocumentTitle(route, appStore.siteName, customMenuItems)
}

// Watch for site settings changes and update favicon/title
watch(
  () => appStore.siteLogo,
  (newLogo) => {
    if (newLogo) {
      updateFavicon(newLogo)
    }
  },
  { immediate: true }
)

watch(
  [
    () => route.fullPath,
    () => route.meta.title,
    () => route.meta.titleKey,
    () => appStore.siteName,
    () => appStore.cachedPublicSettings?.custom_menu_items,
    () => authStore.isAdmin,
    () => adminSettingsStore.customMenuItems,
  ],
  updateDocumentTitle,
  { deep: true }
)

// Announcements: periodic refresh for long-lived authenticated sessions.
// The store throttles fetches to 20 minutes, so poll at the same cadence.
// This replaces the removed per-route afterEach check without re-adding it:
// a continuously visible, non-navigating session still gets fresh data.
const ANNOUNCEMENT_REFRESH_MS = 20 * 60 * 1000
let announcementRefreshTimer: ReturnType<typeof setInterval> | null = null
let pendingLoginAnnouncement: ReturnType<typeof setTimeout> | null = null

function startAnnouncementRefresh() {
  if (announcementRefreshTimer) return // never create duplicate timers
  announcementRefreshTimer = setInterval(() => {
    announcementStore.fetchAnnouncements()
  }, ANNOUNCEMENT_REFRESH_MS)
}

function stopAnnouncementRefresh() {
  if (announcementRefreshTimer) {
    clearInterval(announcementRefreshTimer)
    announcementRefreshTimer = null
  }
}

// A stale delayed login callback must never start the recurring timer: clear it
// on logout/unmount so only a live authenticated session can own the timer.
function clearPendingLoginAnnouncement() {
  if (pendingLoginAnnouncement) {
    clearTimeout(pendingLoginAnnouncement)
    pendingLoginAnnouncement = null
  }
}

// Watch for authentication state and manage subscription data + announcements
function onVisibilityChange() {
  if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
}

function onAdminComplianceRequired(event: Event) {
  const detail = (event as CustomEvent<Record<string, string>>).detail || {}
  adminComplianceStore.requireAcknowledgement(detail)
}

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated, oldValue) => {
    if (isAuthenticated) {
      if (authStore.isAdmin) {
        adminComplianceStore.fetchStatus().catch((error) => {
          console.error('Failed to fetch admin compliance status:', error)
        })
      }

      // User logged in: preload subscriptions and start polling
      subscriptionStore.fetchActiveSubscriptions().catch((error) => {
        console.error('Failed to preload subscriptions:', error)
      })
      subscriptionStore.startPolling()

      // Announcements: new login vs page refresh restore
      if (oldValue === false) {
        // New login: delay 3s then force fetch. The recurring timer starts only
        // after that forced fetch has run, so its first tick is not throttled
        // by the store's 20-minute window (which would push the next real
        // refresh out to ~40 minutes).
        pendingLoginAnnouncement = setTimeout(() => {
          pendingLoginAnnouncement = null
          announcementStore.fetchAnnouncements(true)
          startAnnouncementRefresh()
        }, 3000)
      } else {
        // Page refresh restore (oldValue was undefined)
        announcementStore.fetchAnnouncements()
        startAnnouncementRefresh()
      }

      // Register visibility change listener
      document.addEventListener('visibilitychange', onVisibilityChange)
    } else {
      // User logged out: clear data and stop polling
      clearPendingLoginAnnouncement()
      subscriptionStore.clear()
      announcementStore.reset()
      adminComplianceStore.reset()
      document.removeEventListener('visibilitychange', onVisibilityChange)
      stopAnnouncementRefresh()
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  clearPendingLoginAnnouncement()
  stopAnnouncementRefresh()
  document.removeEventListener('visibilitychange', onVisibilityChange)
  window.removeEventListener('admin-compliance-required', onAdminComplianceRequired)
})

onMounted(async () => {
  window.addEventListener('admin-compliance-required', onAdminComplianceRequired)

  // Check if setup is needed
  try {
    const status = await getSetupStatus()
    if (status.needs_setup && route.path !== '/setup') {
      router.replace('/setup')
      return
    }
  } catch {
    // If setup endpoint fails, assume normal mode and continue
  }

  // Load public settings into appStore (will be cached for other components)
  await appStore.fetchPublicSettings()

  // Re-resolve document title now that site settings are available
  updateDocumentTitle()
})
</script>

<template>
  <NavigationProgress />
  <RouterView />
  <Toast />
  <ConfirmDialogHost />
  <AnnouncementPopup />
  <AdminComplianceDialog />
</template>
