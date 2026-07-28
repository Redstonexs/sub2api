import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, DOMWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

const apiMocks = vi.hoisted(() => ({
  getGroupViewGrants: vi.fn(),
  searchGroupViewGrantCandidates: vi.fn(),
  addGroupViewGrant: vi.fn(),
  removeGroupViewGrant: vi.fn(),
}))

const confirmMock = vi.hoisted(() => vi.fn())
const toastMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/dashboard', () => ({
  getGroupViewGrants: apiMocks.getGroupViewGrants,
  searchGroupViewGrantCandidates: apiMocks.searchGroupViewGrantCandidates,
  addGroupViewGrant: apiMocks.addGroupViewGrant,
  removeGroupViewGrant: apiMocks.removeGroupViewGrant,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => toastMocks,
}))

vi.mock('@/composables/useConfirm', () => ({
  useConfirm: () => ({
    confirm: confirmMock,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) => {
        if (params) {
          return key.replace(/\{(\w+)\}/g, (_, k) => params[k] ?? '')
        }
        return key
      },
    }),
  }
})

vi.mock('@/utils/format', () => ({
  formatDateTime: (d: string) => d,
}))

import GroupViewGrantManager from '../GroupViewGrantManager.vue'

const mockGrants = [
  {
    user_id: 1,
    username: 'alice',
    email: 'alice@test.com',
    granted_by: 'admin',
    granted_at: '2026-07-01T10:00:00Z',
  },
  {
    user_id: 2,
    username: 'bob',
    email: 'bob@test.com',
    granted_by: 'admin',
    granted_at: '2026-07-02T12:00:00Z',
  },
]

const mockCandidates = [
  {
    user_id: 7,
    username: 'carol',
    email: 'carol@test.com',
    role: 'user',
    status: 'active',
    granted: false,
  },
  {
    user_id: 1,
    username: 'alice',
    email: 'alice@test.com',
    role: 'admin',
    status: 'active',
    granted: true,
  },
]

let wrapper: ReturnType<typeof mount> | null = null

function mountComponent(overrides: { groupId?: number; platform?: string } = {}) {
  wrapper = mount(GroupViewGrantManager, {
    props: {
      groupId: overrides.groupId ?? 42,
      platform: overrides.platform ?? 'anthropic',
    },
    attachTo: document.body,
  })
  return wrapper
}

/** Focus the search box and let the on-focus "recent users" fetch settle. */
async function openPicker(w: ReturnType<typeof mountComponent>) {
  await w.find('input[role=combobox]').trigger('focus')
  await flushPromises()
}

/** Type into the search box and fast-forward past the debounce. */
async function search(w: ReturnType<typeof mountComponent>, query: string) {
  const input = w.find('input[role=combobox]')
  await input.setValue(query)
  await input.trigger('input')
  await vi.advanceTimersByTimeAsync(300)
  await flushPromises()
}

/** The dropdown is teleported to <body>, so it lives outside the component subtree. */
function dropdownHtml(): string {
  return document.body.innerHTML
}

function optionButtons() {
  return Array.from(
    document.body.querySelectorAll<HTMLElement>('button[role=option]'),
  ).map((el) => new DOMWrapper(el))
}

beforeEach(() => {
  vi.useFakeTimers()
  setActivePinia(createPinia())
  vi.clearAllMocks()
  apiMocks.getGroupViewGrants.mockResolvedValue([])
  apiMocks.searchGroupViewGrantCandidates.mockResolvedValue([])
  apiMocks.addGroupViewGrant.mockResolvedValue(undefined)
  apiMocks.removeGroupViewGrant.mockResolvedValue(undefined)
  confirmMock.mockReset()
})

afterEach(() => {
  // Unmount so teleported dropdown nodes don't leak into the next test's body query.
  wrapper?.unmount()
  wrapper = null
  document.body.innerHTML = ''
  vi.useRealTimers()
})

describe('GroupViewGrantManager', () => {
  it('renders granted users list from mock', async () => {
    apiMocks.getGroupViewGrants.mockResolvedValueOnce(mockGrants)
    const w = mountComponent()
    await flushPromises()

    expect(apiMocks.getGroupViewGrants).toHaveBeenCalledWith(42)
    const html = w.html()
    expect(html).toContain('alice')
    expect(html).toContain('bob')
    expect(html).toContain('alice@test.com')
    expect(html).toContain('dashboard.groupQuotaCard.grantedBy')
    expect(html).toContain('dashboard.groupQuotaCard.grantedAt')
  })

  it('shows noGrantedUsers when list is empty', async () => {
    apiMocks.getGroupViewGrants.mockResolvedValueOnce([])
    const w = mountComponent()
    await flushPromises()

    expect(w.html()).toContain('dashboard.groupQuotaCard.noGrantedUsers')
  })

  it('search: focus fetches recent users and renders them', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValueOnce(mockCandidates)
    const w = mountComponent()
    await flushPromises()
    await openPicker(w)

    expect(apiMocks.searchGroupViewGrantCandidates).toHaveBeenCalledWith(42, '')
    const html = dropdownHtml()
    expect(html).toContain('carol')
    expect(html).toContain('carol@test.com')
  })

  it('search: debounces keystrokes into a single query', async () => {
    const w = mountComponent()
    await flushPromises()

    const input = w.find('input[role=combobox]')
    for (const q of ['c', 'ca', 'car']) {
      await input.setValue(q)
      await input.trigger('input')
      await vi.advanceTimersByTimeAsync(100)
    }
    await vi.advanceTimersByTimeAsync(300)
    await flushPromises()

    expect(apiMocks.searchGroupViewGrantCandidates).toHaveBeenCalledTimes(1)
    expect(apiMocks.searchGroupViewGrantCandidates).toHaveBeenCalledWith(42, 'car')
  })

  it('search: clicking a result grants access and refetches grants', async () => {
    apiMocks.getGroupViewGrants
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce(mockGrants)
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValue(mockCandidates)
    const w = mountComponent()
    await flushPromises()
    await search(w, 'car')

    await optionButtons()[0].trigger('click')
    await flushPromises()

    expect(apiMocks.addGroupViewGrant).toHaveBeenCalledWith(42, 7)
    expect(apiMocks.getGroupViewGrants).toHaveBeenCalledTimes(2)
    expect(w.emitted('change')).toBeTruthy()
  })

  it('search: dropdown survives clicking a result but closes on an outside click', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValue(mockCandidates)
    const w = mountComponent()
    await flushPromises()
    await search(w, 'car')

    // The dropdown is teleported outside the picker, so the outside-click guard has
    // to recognize it explicitly — otherwise picking a result would close the list.
    document.body.querySelector<HTMLElement>('button[role=option]')?.click()
    await flushPromises()
    expect(optionButtons().length).toBeGreaterThan(0)

    document.body.click()
    await flushPromises()
    expect(optionButtons()).toHaveLength(0)
  })

  it('search: already-granted candidates are disabled and cannot be re-granted', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValue(mockCandidates)
    const w = mountComponent()
    await flushPromises()
    await search(w, 'a')

    const grantedOption = optionButtons()[1]
    expect(grantedOption.text()).toContain('dashboard.groupQuotaCard.alreadyGranted')
    expect(grantedOption.attributes('disabled')).toBeDefined()

    await grantedOption.trigger('click')
    await flushPromises()
    expect(apiMocks.addGroupViewGrant).not.toHaveBeenCalled()
  })

  it('search: keyboard navigation grants the highlighted user', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValue(mockCandidates)
    const w = mountComponent()
    await flushPromises()
    await search(w, 'c')

    const input = w.find('input[role=combobox]')
    // Move off index 0 (carol) onto index 1 (alice, already granted), then back.
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'ArrowUp' })
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()

    expect(apiMocks.addGroupViewGrant).toHaveBeenCalledWith(42, 7)
  })

  it('search: numeric query with no ID match offers a direct grant-by-ID row', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValue([])
    const w = mountComponent()
    await flushPromises()
    await search(w, '99')

    const options = optionButtons()
    expect(options).toHaveLength(1)
    expect(options[0].text()).toContain('dashboard.groupQuotaCard.grantRawUserId')

    await options[0].trigger('click')
    await flushPromises()
    expect(apiMocks.addGroupViewGrant).toHaveBeenCalledWith(42, 99)
  })

  it('search: numeric query matching a candidate ID does not duplicate a raw-ID row', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValue([mockCandidates[0]])
    const w = mountComponent()
    await flushPromises()
    await search(w, '7')

    const options = optionButtons()
    expect(options).toHaveLength(1)
    expect(options[0].text()).not.toContain('dashboard.groupQuotaCard.grantRawUserId')
  })

  it('search: non-numeric query with no matches shows the empty state', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValue([])
    const w = mountComponent()
    await flushPromises()
    await search(w, 'zzz')

    expect(optionButtons()).toHaveLength(0)
    expect(dropdownHtml()).toContain('dashboard.groupQuotaCard.searchUserEmpty')
  })

  it('search: API failure surfaces an error inside the dropdown', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockRejectedValueOnce(
      new Error('search boom'),
    )
    const w = mountComponent()
    await flushPromises()
    await search(w, 'x')

    expect(dropdownHtml()).toContain('search boom')
  })

  it('grant failure surfaces the API error message', async () => {
    apiMocks.searchGroupViewGrantCandidates.mockResolvedValue(mockCandidates)
    apiMocks.addGroupViewGrant.mockRejectedValueOnce(new Error('user not found'))
    const w = mountComponent()
    await flushPromises()
    await search(w, 'car')

    await optionButtons()[0].trigger('click')
    await flushPromises()

    expect(toastMocks.showError).toHaveBeenCalledWith('user not found')
  })

  it('grant list filter narrows rows once the list is long enough', async () => {
    const many = Array.from({ length: 6 }, (_, i) => ({
      user_id: i + 1,
      username: `user${i + 1}`,
      email: `user${i + 1}@test.com`,
      granted_by: 'admin',
      granted_at: '2026-07-01T10:00:00Z',
    }))
    apiMocks.getGroupViewGrants.mockResolvedValueOnce(many)
    const w = mountComponent()
    await flushPromises()

    const filter = w.find('input:not([role=combobox])')
    expect(filter.exists()).toBe(true)

    await filter.setValue('user3')
    await flushPromises()

    const rows = w.findAll('li')
    expect(rows).toHaveLength(1)
    expect(rows[0].text()).toContain('user3')
  })

  it('grant list filter is hidden while the list is short', async () => {
    apiMocks.getGroupViewGrants.mockResolvedValueOnce(mockGrants)
    const w = mountComponent()
    await flushPromises()

    expect(w.find('input:not([role=combobox])').exists()).toBe(false)
  })

  it('revoke: confirm true → calls API → refetches list', async () => {
    confirmMock.mockResolvedValueOnce(true)
    apiMocks.getGroupViewGrants
      .mockResolvedValueOnce(mockGrants)
      .mockResolvedValueOnce([mockGrants[0]])
    const w = mountComponent()
    await flushPromises()

    const revokeBtns = w
      .findAll('button')
      .filter((b) => b.text() === 'dashboard.groupQuotaCard.revoke')
    expect(revokeBtns.length).toBe(2)
    await revokeBtns[0].trigger('click')
    await flushPromises()

    expect(confirmMock).toHaveBeenCalled()
    expect(apiMocks.removeGroupViewGrant).toHaveBeenCalledWith(42, 1)
    expect(apiMocks.getGroupViewGrants).toHaveBeenCalledTimes(2)
  })

  it('revoke: confirm false → does not call API', async () => {
    confirmMock.mockResolvedValueOnce(false)
    apiMocks.getGroupViewGrants.mockResolvedValueOnce(mockGrants)
    const w = mountComponent()
    await flushPromises()

    const revokeBtns = w
      .findAll('button')
      .filter((b) => b.text() === 'dashboard.groupQuotaCard.revoke')
    await revokeBtns[0].trigger('click')
    await flushPromises()

    expect(confirmMock).toHaveBeenCalled()
    expect(apiMocks.removeGroupViewGrant).not.toHaveBeenCalled()
  })

  it('platform gate: gemini → renders nothing', async () => {
    const w = mountComponent({ platform: 'gemini' })
    const html = w.html()
    expect(html).not.toContain('alice')
    expect(html).not.toContain('dashboard.groupQuotaCard')
  })

  it('platform gate: empty grants shows noGrantedUsers', async () => {
    apiMocks.getGroupViewGrants.mockResolvedValueOnce([])
    const w = mountComponent({ platform: 'openai' })
    await flushPromises()

    expect(w.html()).toContain('dashboard.groupQuotaCard.noGrantedUsers')
  })

  it('API error on load surfaces error message', async () => {
    apiMocks.getGroupViewGrants.mockRejectedValueOnce(new Error('network error'))
    const w = mountComponent()
    await flushPromises()

    expect(w.html()).toContain('network error')
  })
})
