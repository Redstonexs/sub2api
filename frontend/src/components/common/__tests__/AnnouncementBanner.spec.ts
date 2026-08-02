import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import AnnouncementBanner from '../AnnouncementBanner.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import type { UserAnnouncement } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

function announcement(overrides: Partial<UserAnnouncement> = {}): UserAnnouncement {
  return {
    id: 1,
    title: 'Maintenance window',
    content: 'We will be down briefly.',
    notify_mode: 'silent',
    severity: 'info',
    show_banner: true,
    created_at: '2026-07-24T07:30:00Z',
    updated_at: '2026-07-24T07:30:00Z',
    ...overrides,
  }
}

function mountBanner() {
  return mount(AnnouncementBanner, { global: { stubs: { teleport: true } } })
}

describe('AnnouncementBanner', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('renders nothing when no announcement is flagged for a banner', () => {
    const store = useAnnouncementStore()
    store.announcements = [announcement({ show_banner: false })]

    expect(mountBanner().find('[data-testid="announcement-banner"]').exists()).toBe(false)
  })

  it('renders the flagged announcement with a one-line excerpt', () => {
    const store = useAnnouncementStore()
    store.announcements = [announcement({ content: '# Heading\n\nFirst body line.' })]

    const wrapper = mountBanner()
    expect(wrapper.find('[data-testid="announcement-banner"]').text()).toContain('Maintenance window')
    // The excerpt takes the first non-empty line with Markdown markers stripped.
    expect(wrapper.text()).toContain('Heading')
  })

  it('shows only the highest-severity banner and counts the rest', () => {
    const store = useAnnouncementStore()
    store.announcements = [
      announcement({ id: 1, title: 'Info one', severity: 'info' }),
      announcement({ id: 2, title: 'Critical one', severity: 'critical' }),
      announcement({ id: 3, title: 'Warning one', severity: 'warning' }),
    ]

    const wrapper = mountBanner()
    const text = wrapper.find('[data-testid="announcement-banner"]').text()
    expect(text).toContain('Critical one')
    expect(text).not.toContain('Warning one')
    expect(text).toContain('+2')
  })

  it('dismisses per device without writing a read receipt', async () => {
    const store = useAnnouncementStore()
    store.announcements = [announcement()]
    const markAsRead = vi.spyOn(store, 'markAsRead')

    const wrapper = mountBanner()
    await wrapper.find('[data-testid="announcement-banner-dismiss"]').trigger('click')

    expect(wrapper.find('[data-testid="announcement-banner"]').exists()).toBe(false)
    // A read receipt would poison the admin read-status analytics and hide the
    // announcement from the bell.
    expect(markAsRead).not.toHaveBeenCalled()
    expect(store.announcements[0].read_at).toBeUndefined()
    expect(localStorage.getItem('announcement_banner_dismissed')).toContain('"1"')
  })

  it('re-surfaces a dismissed banner after the announcement is edited', async () => {
    const store = useAnnouncementStore()
    store.announcements = [announcement()]

    const first = mountBanner()
    await first.find('[data-testid="announcement-banner-dismiss"]').trigger('click')
    expect(first.find('[data-testid="announcement-banner"]').exists()).toBe(false)

    // Dismissals are keyed by id -> updated_at.
    store.announcements = [announcement({ updated_at: '2026-07-25T09:00:00Z' })]
    expect(mountBanner().find('[data-testid="announcement-banner"]').exists()).toBe(true)
  })

  it('keeps dismissals across a store reset', () => {
    const store = useAnnouncementStore()
    store.announcements = [announcement()]
    store.dismissBanner(1)

    // reset() clears session state; a device preference is not session state.
    store.reset()
    store.announcements = [announcement()]

    expect(mountBanner().find('[data-testid="announcement-banner"]').exists()).toBe(false)
  })

  it('applies a severity-specific tone', () => {
    const store = useAnnouncementStore()
    store.announcements = [announcement({ severity: 'critical' })]

    const classes = mountBanner().find('[data-testid="announcement-banner"]').classes().join(' ')
    expect(classes).toContain('red')
  })
})

describe('announcement store banner state', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('counts every unread announcement, not just the first 20', () => {
    const store = useAnnouncementStore()
    store.announcements = Array.from({ length: 25 }, (_, i) => announcement({ id: i + 1 }))

    // The store used to slice(0, 20), which silently capped unreadCount.
    expect(store.unreadCount).toBe(25)
  })

  it('ignores a corrupt dismissal payload', () => {
    localStorage.setItem('announcement_banner_dismissed', 'not json')
    setActivePinia(createPinia())

    const store = useAnnouncementStore()
    store.announcements = [announcement()]
    expect(store.bannerAnnouncements).toHaveLength(1)
  })
})
