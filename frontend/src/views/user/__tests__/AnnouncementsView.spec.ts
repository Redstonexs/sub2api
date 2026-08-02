import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import AnnouncementsView from '../AnnouncementsView.vue'

const { listArchive } = vi.hoisted(() => ({ listArchive: vi.fn() }))

vi.mock('@/api', () => ({
  announcementsAPI: { listArchive },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

async function passthroughModule(name: string) {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      name,
      setup: (_props, { slots }) => () => h('div', Object.values(slots).map((slot) => slot?.())),
    }),
  }
}

vi.mock('@/components/layout/AppLayout.vue', () => passthroughModule('AppLayout'))
vi.mock('@/components/layout/TablePageLayout.vue', () => passthroughModule('TablePageLayout'))

function archived(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    title: 'Maintenance window',
    content: '## Heading\n\nWe will be down briefly.',
    notify_mode: 'silent',
    severity: 'warning',
    show_banner: false,
    created_at: '2026-07-24T07:30:00Z',
    updated_at: '2026-07-24T07:30:00Z',
    ...overrides,
  }
}

async function mountView() {
  const wrapper = mount(AnnouncementsView, {
    attachTo: document.body,
    global: { stubs: { teleport: true } },
  })
  await flushPromises()
  return wrapper
}

describe('user AnnouncementsView (archive)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // AnnouncementPopup reads the announcement store.
    setActivePinia(createPinia())
    document.body.innerHTML = ''
    listArchive.mockResolvedValue({ items: [archived()], total: 1, page: 1, page_size: 20, pages: 1 })
  })

  it('loads the archive on mount', async () => {
    const wrapper = await mountView()

    expect(listArchive).toHaveBeenCalledTimes(1)
    expect(listArchive.mock.calls[0][2]).toMatchObject({ unread_only: false, search: '' })
    expect(wrapper.text()).toContain('Maintenance window')
    wrapper.unmount()
  })

  it('shows a Markdown-stripped excerpt rather than raw syntax', async () => {
    const wrapper = await mountView()

    expect(wrapper.text()).toContain('Heading')
    expect(wrapper.text()).not.toContain('## Heading')
    wrapper.unmount()
  })

  it('reloads with unread_only when the filter is toggled', async () => {
    const wrapper = await mountView()

    await wrapper.find('[data-testid="announcement-archive-unread-only"]').setValue(true)
    await flushPromises()

    expect(listArchive).toHaveBeenCalledTimes(2)
    expect(listArchive.mock.calls[1][2]).toMatchObject({ unread_only: true })
    wrapper.unmount()
  })

  it('opens the detail popup without marking the announcement read', async () => {
    const wrapper = await mountView()

    await wrapper.find('[data-testid="announcement-archive-open-1"]').trigger('click')
    await flushPromises()

    const popup = wrapper.findComponent({ name: 'AnnouncementPopup' })
    expect(popup.exists()).toBe(true)
    // preview mode: reading the archive must not consume the unread state.
    expect(popup.props('preview')).toBe(true)
    wrapper.unmount()
  })

  it('marks an unread row distinctly from a read one', async () => {
    listArchive.mockResolvedValue({
      items: [archived({ id: 1 }), archived({ id: 2, read_at: '2026-07-25T09:00:00Z' })],
      total: 2, page: 1, page_size: 20, pages: 1,
    })
    const wrapper = await mountView()

    const text = wrapper.text()
    expect(text).toContain('announcements.unread')
    expect(text).toContain('announcements.read')
    wrapper.unmount()
  })
})
