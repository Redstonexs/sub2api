import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { announcementsAPI } from '@/api'
import type { UserAnnouncement } from '@/types'

const THROTTLE_MS = 20 * 60 * 1000 // 20 minutes

const BANNER_DISMISSAL_KEY = 'announcement_banner_dismissed'

/** critical sorts first, then warning, then info. */
const SEVERITY_RANK: Record<string, number> = { critical: 3, warning: 2, info: 1 }

type DismissalMap = Record<string, string>

/**
 * Banner dismissals live in localStorage rather than the announcement_reads table.
 *
 * That table drives the admin read-status analytics and the bell's unread count;
 * writing a read receipt when someone closes a banner would poison delivery stats
 * and hide the announcement from the bell. Dismissal is a per-device preference.
 *
 * Keyed by announcement id -> its updated_at, so editing an announcement
 * re-surfaces a banner that was already dismissed.
 */
function readDismissals(): DismissalMap {
  try {
    const raw = localStorage.getItem(BANNER_DISMISSAL_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' ? (parsed as DismissalMap) : {}
  } catch {
    return {}
  }
}

function writeDismissals(map: DismissalMap) {
  try {
    localStorage.setItem(BANNER_DISMISSAL_KEY, JSON.stringify(map))
  } catch {
    // Private browsing / quota — dismissal simply does not persist.
  }
}

export const useAnnouncementStore = defineStore('announcements', () => {
  // State
  const announcements = ref<UserAnnouncement[]>([])
  const loading = ref(false)
  const lastFetchTime = ref(0)
  const popupQueue = ref<UserAnnouncement[]>([])
  const currentPopup = ref<UserAnnouncement | null>(null)
  // In-memory mirror of the persisted dismissals so the banner reacts immediately.
  const dismissedBanners = ref<DismissalMap>(readDismissals())

  // Session-scoped dedup set — not reactive, used as plain lookup only
  let shownPopupIds = new Set<number>()

  // Getters
  const unreadCount = computed(() =>
    announcements.value.filter((a) => !a.read_at).length
  )

  /** Banner-flagged announcements that this device has not dismissed, worst first. */
  const bannerAnnouncements = computed(() =>
    announcements.value
      .filter((a) => a.show_banner && dismissedBanners.value[a.id] !== a.updated_at)
      .sort((a, b) => (SEVERITY_RANK[b.severity] ?? 1) - (SEVERITY_RANK[a.severity] ?? 1) || b.id - a.id)
  )

  function dismissBanner(id: number) {
    const ann = announcements.value.find((a) => a.id === id)
    if (!ann) return
    dismissedBanners.value = { ...dismissedBanners.value, [id]: ann.updated_at }
    writeDismissals(dismissedBanners.value)
  }

  /** Drops dismissals for announcements the backend no longer returns. */
  function pruneDismissals() {
    const live = new Set(announcements.value.map((a) => String(a.id)))
    const next: DismissalMap = {}
    for (const [id, updatedAt] of Object.entries(dismissedBanners.value)) {
      if (live.has(id)) next[id] = updatedAt
    }
    if (Object.keys(next).length !== Object.keys(dismissedBanners.value).length) {
      dismissedBanners.value = next
      writeDismissals(next)
    }
  }

  // Actions
  async function fetchAnnouncements(force = false) {
    const now = Date.now()
    if (!force && lastFetchTime.value > 0 && now - lastFetchTime.value < THROTTLE_MS) {
      return
    }

    // Set immediately to prevent concurrent duplicate requests
    lastFetchTime.value = now

    try {
      loading.value = true
      // No client-side cap: slicing to 20 here silently broke unreadCount. The
      // backend already bounds the active set (announcementActiveScanLimit).
      announcements.value = await announcementsAPI.list(false)
      pruneDismissals()
      enqueueNewPopups()
    } catch (err: any) {
      // Revert throttle timestamp on failure so retry is allowed
      lastFetchTime.value = 0
      console.error('Failed to fetch announcements:', err)
    } finally {
      loading.value = false
    }
  }

  function enqueueNewPopups() {
    const newPopups = announcements.value.filter(
      (a) => a.notify_mode === 'popup' && !a.read_at && !shownPopupIds.has(a.id)
    )
    if (newPopups.length === 0) return

    for (const p of newPopups) {
      if (!popupQueue.value.some((q) => q.id === p.id)) {
        popupQueue.value.push(p)
      }
    }

    if (!currentPopup.value) {
      showNextPopup()
    }
  }

  function showNextPopup() {
    if (popupQueue.value.length === 0) {
      currentPopup.value = null
      return
    }
    currentPopup.value = popupQueue.value.shift()!
    shownPopupIds.add(currentPopup.value.id)
  }

  async function dismissPopup() {
    if (!currentPopup.value) return
    const id = currentPopup.value.id
    currentPopup.value = null

    // Mark as read (fire-and-forget, UI already updated)
    markAsRead(id)

    // Show next popup after a short delay
    if (popupQueue.value.length > 0) {
      setTimeout(() => showNextPopup(), 300)
    }
  }

  async function markAsRead(id: number) {
    try {
      await announcementsAPI.markRead(id)
      const ann = announcements.value.find((a) => a.id === id)
      if (ann) {
        ann.read_at = new Date().toISOString()
      }
    } catch (err: any) {
      console.error('Failed to mark announcement as read:', err)
    }
  }

  async function markAllAsRead() {
    const unread = announcements.value.filter((a) => !a.read_at)
    if (unread.length === 0) return

    try {
      loading.value = true
      await Promise.all(unread.map((a) => announcementsAPI.markRead(a.id)))
      announcements.value.forEach((a) => {
        if (!a.read_at) {
          a.read_at = new Date().toISOString()
        }
      })
    } catch (err: any) {
      console.error('Failed to mark all as read:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  function reset() {
    // dismissedBanners is intentionally left alone: it is a device preference
    // persisted in localStorage, not session state.
    announcements.value = []
    lastFetchTime.value = 0
    shownPopupIds = new Set()
    popupQueue.value = []
    currentPopup.value = null
    loading.value = false
  }

  return {
    // State
    announcements,
    loading,
    currentPopup,
    // Getters
    unreadCount,
    bannerAnnouncements,
    // Actions
    fetchAnnouncements,
    dismissBanner,
    dismissPopup,
    markAsRead,
    markAllAsRead,
    reset,
  }
})
