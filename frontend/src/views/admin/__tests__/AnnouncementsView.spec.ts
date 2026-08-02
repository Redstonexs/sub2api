import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import AnnouncementsView from '../AnnouncementsView.vue'

const {
  listAnnouncements,
  createAnnouncement,
  updateAnnouncement,
  deleteAnnouncement,
  getAllGroups,
  previewAudience,
  sendTestEmail,
  showError,
  showSuccess,
  confirmMock,
} = vi.hoisted(() => ({
  listAnnouncements: vi.fn(),
  createAnnouncement: vi.fn(),
  updateAnnouncement: vi.fn(),
  deleteAnnouncement: vi.fn(),
  getAllGroups: vi.fn(),
  previewAudience: vi.fn(),
  sendTestEmail: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  confirmMock: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    announcements: {
      list: listAnnouncements,
      create: createAnnouncement,
      update: updateAnnouncement,
      delete: deleteAnnouncement,
      previewAudience,
      sendTestEmail,
    },
    groups: { getAll: getAllGroups },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/composables/useConfirm', () => ({
  useConfirm: () => ({ confirm: confirmMock }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  // `t` echoes the key so assertions can name keys directly. extractI18nErrorMessage
  // treats "translation === key" as missing, so `te` reports which keys really exist.
  const t = Object.assign(
    (key: string) => key,
    { te: (key: string) => key.startsWith('admin.announcements.errors.') },
  )
  return { ...actual, useI18n: () => ({ t }) }
})

// Layout shells only provide chrome; stub them so the test focuses on this view.
// The factories are hoisted above the imports, hence the dynamic import of vue.
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

const announcement = {
  id: 7,
  title: 'Scheduled maintenance',
  content: 'body',
  status: 'active',
  notify_mode: 'popup',
  targeting: { any_of: [] },
  starts_at: null as string | null,
  ends_at: null as string | null,
  created_at: '2026-07-24T07:30:00Z',
  updated_at: '2026-07-24T07:30:00Z',
}

function listResponse(items: unknown[]) {
  return { items, total: items.length, page: 1, page_size: 20, pages: 1 }
}

async function mountView() {
  // BaseDialog teleports to body; stubbing Teleport keeps the form inside the
  // wrapper so it can be queried normally.
  const wrapper = mount(AnnouncementsView, {
    attachTo: document.body,
    global: { stubs: { teleport: true } },
  })
  await flushPromises()
  return wrapper
}

describe('AnnouncementsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // AnnouncementPopup (rendered for the admin preview) reads the announcement store.
    setActivePinia(createPinia())
    document.body.innerHTML = ''
    getAllGroups.mockResolvedValue([])
    previewAudience.mockResolvedValue({
      scanned: 100, matched: 42, with_email: 40, unsubscribed: 2,
      deliverable: 38, truncated: false,
    })
    sendTestEmail.mockResolvedValue({ message: 'ok', recipient: 'admin@example.com' })
    listAnnouncements.mockResolvedValue(listResponse([announcement]))
    confirmMock.mockResolvedValue(true)
  })

  it('surfaces the backend error code instead of a generic message', async () => {
    // The API client rejects with a flat {code, reason} object — the old
    // error.response?.data?.detail read was always undefined.
    listAnnouncements.mockRejectedValueOnce({
      status: 400,
      code: 'ANNOUNCEMENT_CONTENT_TOO_LONG',
      reason: 'ANNOUNCEMENT_CONTENT_TOO_LONG',
      message: 'announcement content is too long',
    })
    await mountView()

    expect(showError).toHaveBeenCalledWith('admin.announcements.errors.ANNOUNCEMENT_CONTENT_TOO_LONG')
  })

  it('renders a Markdown editor rather than a bare textarea', async () => {
    const wrapper = await mountView()
    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="markdown-editor"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="md-textarea"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('labels an active announcement whose window has not opened as scheduled', async () => {
    listAnnouncements.mockResolvedValue(listResponse([
      { ...announcement, id: 1, starts_at: new Date(Date.now() + 86_400_000).toISOString() },
      { ...announcement, id: 2, ends_at: new Date(Date.now() - 86_400_000).toISOString() },
      { ...announcement, id: 3 },
      { ...announcement, id: 4, status: 'draft' },
    ]))
    const wrapper = await mountView()

    const text = wrapper.text()
    expect(text).toContain('admin.announcements.lifecycle.scheduled')
    expect(text).toContain('admin.announcements.lifecycle.expired')
    expect(text).toContain('admin.announcements.lifecycle.live')
    wrapper.unmount()
  })

  it('rejects a schedule whose start is not before its end', async () => {
    const wrapper = await mountView()
    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()

    const [startsAt, endsAt] = wrapper.findAll('input[type="datetime-local"]')
    await startsAt.setValue('2026-08-01T10:00')
    await endsAt.setValue('2026-08-01T10:00')

    expect(wrapper.find('[data-testid="announcement-schedule-error"]').text())
      .toBe('admin.announcements.errors.ANNOUNCEMENT_TIME_RANGE_INVALID')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(createAnnouncement).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.announcements.errors.ANNOUNCEMENT_TIME_RANGE_INVALID')
    wrapper.unmount()
  })

  it('duplicates an announcement as a draft', async () => {
    // Duplicating a live email announcement and saving unchanged would fan a
    // second broadcast out to everyone.
    listAnnouncements.mockResolvedValue(listResponse([
      { ...announcement, status: 'active', notify_mode: 'email' },
    ]))
    const wrapper = await mountView()

    await wrapper.find('[data-testid="announcement-duplicate"]').trigger('click')
    await flushPromises()

    const title = wrapper.find<HTMLInputElement>('form input[type="text"]').element
    expect(title.value).toContain('Scheduled maintenance')
    expect(title.value).toContain('admin.announcements.duplicateTitleSuffix')

    createAnnouncement.mockResolvedValue({})
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(createAnnouncement).toHaveBeenCalledTimes(1)
    expect(createAnnouncement.mock.calls[0][0]).toMatchObject({ status: 'draft' })
    wrapper.unmount()
  })

  it('asks before discarding unsaved edits', async () => {
    const wrapper = await mountView()
    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()

    await wrapper.find('form input[type="text"]').setValue('changed')
    confirmMock.mockResolvedValueOnce(false)

    await wrapper.findComponent({ name: 'BaseDialog' }).vm.$emit('close')
    await flushPromises()

    expect(confirmMock).toHaveBeenCalledTimes(1)
    // Cancelled: the dialog stays open with the edit intact.
    expect(wrapper.find<HTMLInputElement>('form input[type="text"]').element.value).toBe('changed')
    wrapper.unmount()
  })

  it('closes without confirming when nothing changed', async () => {
    const wrapper = await mountView()
    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()

    await wrapper.findComponent({ name: 'BaseDialog' }).vm.$emit('close')
    await flushPromises()

    expect(confirmMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

describe('AnnouncementsView email broadcast rails', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
    document.body.innerHTML = ''
    getAllGroups.mockResolvedValue([])
    listAnnouncements.mockResolvedValue(listResponse([announcement]))
    previewAudience.mockResolvedValue({
      scanned: 100, matched: 42, with_email: 40, unsubscribed: 2,
      deliverable: 38, truncated: false,
    })
    sendTestEmail.mockResolvedValue({ message: 'ok', recipient: 'admin@example.com' })
    confirmMock.mockResolvedValue(true)
    createAnnouncement.mockResolvedValue({})
    updateAnnouncement.mockResolvedValue({})
  })

  async function openCreateForm() {
    const wrapper = await mountView()
    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()
    return wrapper
  }

  /**
   * Sets a form Select via its v-model, located by its option values rather than by
   * index — index is ambiguous because the page also renders a status *filter*
   * whose options overlap the form's.
   */
  async function setSelect(wrapper: any, value: string) {
    const selects = wrapper.findAllComponents({ name: 'Select' })
    const target = selects.find((select: any) => {
      const options = (select.props('options') ?? []) as Array<{ value: string }>
      // The filter select carries an extra '' (all statuses) entry.
      return options.some((o) => o.value === value) && !options.some((o) => o.value === '')
    })
    if (!target) throw new Error(`no form Select offers "${value}"`)
    await target.vm.$emit('update:modelValue', value)
    await flushPromises()
  }

  it('shows the deliverable count on demand', async () => {
    const wrapper = await openCreateForm()

    await wrapper.find('[data-testid="announcement-estimate-audience"]').trigger('click')
    await flushPromises()

    expect(previewAudience).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-testid="announcement-audience-result"]').text())
      .toContain('admin.announcements.audienceSummary')
    wrapper.unmount()
  })

  it('confirms before a save that fans an email out to everyone', async () => {
    const wrapper = await openCreateForm()
    await wrapper.find('form input[type="text"]').setValue('Broadcast')
    await setSelect(wrapper, 'active')
    await setSelect(wrapper, 'email')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    // The count backing the dialog is fetched fresh rather than assumed.
    expect(previewAudience).toHaveBeenCalledTimes(1)
    expect(confirmMock).toHaveBeenCalledTimes(1)
    expect(createAnnouncement).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('does not save when the broadcast confirmation is declined', async () => {
    const wrapper = await openCreateForm()
    await wrapper.find('form input[type="text"]').setValue('Broadcast')
    await setSelect(wrapper, 'active')
    await setSelect(wrapper, 'email')
    confirmMock.mockResolvedValueOnce(false)

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(createAnnouncement).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('does not confirm for a draft or a non-email announcement', async () => {
    const wrapper = await openCreateForm()
    await wrapper.find('form input[type="text"]').setValue('Quiet')
    await setSelect(wrapper, 'email') // email, but still a draft

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(confirmMock).not.toHaveBeenCalled()
    expect(createAnnouncement).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('does not re-confirm when an announcement is already published by email', async () => {
    // maybeBroadcastEmail only fires on the transition into active+email, so an
    // edit to an already-broadcasting announcement must not prompt again.
    listAnnouncements.mockResolvedValue(listResponse([
      { ...announcement, status: 'active', notify_mode: 'email' },
    ]))
    const wrapper = await mountView()
    await wrapper.find('[title="common.edit"]').trigger('click')
    await flushPromises()

    await wrapper.find('form input[type="text"]').setValue('Edited title')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(confirmMock).not.toHaveBeenCalled()
    expect(updateAnnouncement).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('offers a test send only when editing a saved announcement', async () => {
    const created = await openCreateForm()
    expect(created.find('[data-testid="announcement-send-test"]').exists()).toBe(false)
    created.unmount()

    const wrapper = await mountView()
    await wrapper.find('[title="common.edit"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-testid="announcement-send-test"]').trigger('click')
    await flushPromises()

    expect(sendTestEmail).toHaveBeenCalledWith(7)
    expect(showSuccess).toHaveBeenCalledWith('admin.announcements.testEmailSent')
    wrapper.unmount()
  })

  it('discards a stale estimate when the targeting changes', async () => {
    const wrapper = await openCreateForm()
    await wrapper.find('[data-testid="announcement-estimate-audience"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="announcement-audience-result"]').exists()).toBe(true)

    // A stale count is worse than none: it is what the publish gate would show.
    const editor = wrapper.findComponent({ name: 'AnnouncementTargetingEditor' })
    await editor.vm.$emit('update:modelValue', { any_of: [{ all_of: [] }] })
    await flushPromises()

    expect(wrapper.find('[data-testid="announcement-audience-result"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
