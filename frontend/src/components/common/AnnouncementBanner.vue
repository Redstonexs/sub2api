<template>
  <div v-if="banner" :class="['border-b px-4 py-2.5 md:px-6 lg:px-8', toneClasses.container]" data-testid="announcement-banner">
    <div class="mx-auto flex max-w-7xl items-start gap-3">
      <Icon :name="toneClasses.icon" size="sm" :class="['mt-0.5 shrink-0', toneClasses.iconColor]" />

      <button
        type="button"
        class="min-w-0 flex-1 text-left"
        data-testid="announcement-banner-open"
        @click="showDetail = true"
      >
        <span :class="['font-semibold', toneClasses.title]">{{ banner.title }}</span>
        <span v-if="excerpt" class="ml-2 text-sm opacity-80">{{ excerpt }}</span>
      </button>

      <span
        v-if="remainingCount > 0"
        class="shrink-0 rounded-full bg-black/10 px-2 py-0.5 text-xs font-medium dark:bg-white/10"
      >
        +{{ remainingCount }}
      </span>

      <button
        type="button"
        :aria-label="t('common.close')"
        :title="t('common.close')"
        data-testid="announcement-banner-dismiss"
        class="-mr-1 shrink-0 rounded-lg p-1 opacity-60 transition-opacity hover:opacity-100"
        @click="dismiss"
      >
        <Icon name="x" size="sm" />
      </button>
    </div>

    <AnnouncementPopup
      v-if="showDetail"
      :announcement="banner"
      preview
      @close="showDetail = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import Icon from '@/components/icons/Icon.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import { useAnnouncementStore } from '@/stores/announcements'

/**
 * A persistent top-of-page banner for announcements flagged `show_banner`.
 *
 * Only the single highest-severity undismissed banner is rendered; the rest are
 * summarised as a "+N" chip so a burst of announcements cannot push the page down.
 * Dismissal is per-device (localStorage) rather than a read receipt — see the store
 * for why.
 *
 * Detail opens the existing AnnouncementPopup in `preview` mode, which shows the
 * content without marking it read; the bell remains the place to do that.
 */
const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const { bannerAnnouncements } = storeToRefs(announcementStore)

const showDetail = ref(false)

const banner = computed(() => bannerAnnouncements.value[0] ?? null)
const remainingCount = computed(() => Math.max(0, bannerAnnouncements.value.length - 1))

/** First non-empty line of the body, stripped of Markdown noise. */
const excerpt = computed(() => {
  const raw = banner.value?.content ?? ''
  const line = raw
    .split('\n')
    .map((l) => l.replace(/^[#>\-*\s]+/, '').trim())
    .find((l) => l.length > 0) ?? ''
  return line.length > 120 ? `${line.slice(0, 120)}…` : line
})

const TONES = {
  critical: {
    container: 'border-red-200 bg-red-50 text-red-900 dark:border-red-900/40 dark:bg-red-950/40 dark:text-red-100',
    icon: 'exclamationCircle',
    iconColor: 'text-red-600 dark:text-red-400',
    title: 'text-red-900 dark:text-red-100',
  },
  warning: {
    container: 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/40 dark:text-amber-100',
    icon: 'exclamationTriangle',
    iconColor: 'text-amber-600 dark:text-amber-400',
    title: 'text-amber-900 dark:text-amber-100',
  },
  info: {
    container: 'border-primary-200 bg-primary-50 text-primary-900 dark:border-primary-900/40 dark:bg-primary-950/40 dark:text-primary-100',
    icon: 'infoCircle',
    iconColor: 'text-primary-600 dark:text-primary-400',
    title: 'text-primary-900 dark:text-primary-100',
  },
} as const

const toneClasses = computed(() => TONES[banner.value?.severity ?? 'info'] ?? TONES.info)

function dismiss() {
  if (banner.value) announcementStore.dismissBanner(banner.value.id)
}
</script>
