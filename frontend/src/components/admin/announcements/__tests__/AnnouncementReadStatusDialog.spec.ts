import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AnnouncementReadStatusDialog from '../AnnouncementReadStatusDialog.vue'

const { getReadStatus, showError } = vi.hoisted(() => ({
  getReadStatus: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    announcements: {
      getReadStatus,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20,
}))

const BaseDialogStub = {
  props: ['show', 'title', 'width'],
  emits: ['close'],
  template: '<div><slot /><slot name="footer" /></div>',
}

describe('AnnouncementReadStatusDialog', () => {
  beforeEach(() => {
    getReadStatus.mockReset()
    showError.mockReset()
    vi.useFakeTimers()
  })

  it('closes by aborting active requests and clearing debounced reloads', async () => {
    let activeSignal: AbortSignal | undefined
    getReadStatus.mockImplementation(async (...args: any[]) => {
      activeSignal = args[4]?.signal
      return new Promise(() => {})
    })

    const wrapper = mount(AnnouncementReadStatusDialog, {
      props: {
        show: false,
        announcementId: 1,
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          DataTable: true,
          Pagination: true,
          Icon: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(getReadStatus).toHaveBeenCalledTimes(1)
    expect(activeSignal?.aborted).toBe(false)

    const setupState = (wrapper.vm as any).$?.setupState
    setupState.search = 'alice'
    setupState.handleSearch()

    setupState.handleClose()
    await flushPromises()

    expect(activeSignal?.aborted).toBe(true)
    expect(wrapper.emitted('close')).toHaveLength(1)

    vi.advanceTimersByTime(350)
    await flushPromises()

    expect(getReadStatus).toHaveBeenCalledTimes(1)
  })

  it('renders email unsubscribe status badge for each user', async () => {
    getReadStatus.mockResolvedValue({
      items: [
        { user_id: 1, email: 'alice@example.com', username: 'alice', balance: 10, eligible: true, announcement_email_unsubscribed: true, read_at: null },
        { user_id: 2, email: 'bob@example.com', username: 'bob', balance: 5, eligible: false, announcement_email_unsubscribed: false, read_at: null },
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const DataTableStub = {
      props: ['columns', 'data', 'loading', 'serverSideSort', 'defaultSortKey', 'defaultSortOrder'],
      emits: ['sort'],
      template: `
        <div class="data-table-stub">
          <div v-for="(row, i) in data" :key="i" class="row">
            <div v-for="col in columns" :key="col.key" class="cell" :data-col="col.key">
              <slot :name="'cell-' + col.key" :row="row" :value="row[col.key]">{{ row[col.key] }}</slot>
            </div>
          </div>
        </div>
      `,
    }

    const wrapper = mount(AnnouncementReadStatusDialog, {
      props: {
        show: false,
        announcementId: 1,
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          DataTable: DataTableStub,
          Pagination: true,
          Icon: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    const html = wrapper.html()
    expect(html).toContain('admin.announcements.emailUnsubscribed')
    expect(html).toContain('admin.announcements.emailSubscribed')
  })
})
