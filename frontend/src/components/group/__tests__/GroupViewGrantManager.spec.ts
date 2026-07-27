import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

const apiMocks = vi.hoisted(() => ({
  getGroupViewGrants: vi.fn(),
  addGroupViewGrant: vi.fn(),
  removeGroupViewGrant: vi.fn(),
}))

const confirmMock = vi.hoisted(() => vi.fn())

vi.mock('@/api/admin/dashboard', () => ({
  getGroupViewGrants: apiMocks.getGroupViewGrants,
  addGroupViewGrant: apiMocks.addGroupViewGrant,
  removeGroupViewGrant: apiMocks.removeGroupViewGrant,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
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
    granted_by: 'admin',
    granted_at: '2026-07-01T10:00:00Z',
  },
  {
    user_id: 2,
    username: 'bob',
    granted_by: 'admin',
    granted_at: '2026-07-02T12:00:00Z',
  },
]

function mountComponent(
  overrides: { groupId?: number; platform?: string } = {},
) {
  return mount(GroupViewGrantManager, {
    props: {
      groupId: overrides.groupId ?? 42,
      platform: overrides.platform ?? 'anthropic',
    },
  })
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  apiMocks.getGroupViewGrants.mockResolvedValue([])
  apiMocks.addGroupViewGrant.mockResolvedValue(undefined)
  apiMocks.removeGroupViewGrant.mockResolvedValue(undefined)
  confirmMock.mockReset()
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
    expect(html).toContain('dashboard.groupQuotaCard.grantedBy')
    expect(html).toContain('dashboard.groupQuotaCard.grantedAt')
  })

  it('shows noGrantedUsers when list is empty', async () => {
    apiMocks.getGroupViewGrants.mockResolvedValueOnce([])
    const w = mountComponent()
    await flushPromises()

    expect(w.html()).toContain('dashboard.groupQuotaCard.noGrantedUsers')
  })

  it('add: calls API with correct args, refetches, clears input', async () => {
    apiMocks.getGroupViewGrants
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce(mockGrants)
    const w = mountComponent()
    await flushPromises()

    const input = w.find('input[type=number]')
    await input.setValue('7')

    const buttons = w.findAll('button')
    const addBtn = buttons.find((b) => b.text() === 'dashboard.groupQuotaCard.add')
    expect(addBtn).toBeTruthy()
    await addBtn?.trigger('click')
    await flushPromises()

    expect(apiMocks.addGroupViewGrant).toHaveBeenCalledWith(42, 7)
    expect(apiMocks.getGroupViewGrants).toHaveBeenCalledTimes(2)
  })

  it('add: invalid input (0) does not call API', async () => {
    apiMocks.getGroupViewGrants.mockResolvedValueOnce([])
    const w = mountComponent()
    await flushPromises()

    const input = w.find('input[type=number]')
    await input.setValue('0')

    const buttons = w.findAll('button')
    const addBtn = buttons.find((b) => b.text() === 'dashboard.groupQuotaCard.add')
    await addBtn?.trigger('click')
    await flushPromises()

    expect(apiMocks.addGroupViewGrant).not.toHaveBeenCalled()
  })

  it('add: invalid input (negative) does not call API', async () => {
    apiMocks.getGroupViewGrants.mockResolvedValueOnce([])
    const w = mountComponent()
    await flushPromises()

    const input = w.find('input[type=number]')
    await input.setValue('-5')

    const buttons = w.findAll('button')
    const addBtn = buttons.find((b) => b.text() === 'dashboard.groupQuotaCard.add')
    await addBtn?.trigger('click')
    await flushPromises()

    expect(apiMocks.addGroupViewGrant).not.toHaveBeenCalled()
  })

  it('add: non-numeric input does not call API', async () => {
    apiMocks.getGroupViewGrants.mockResolvedValueOnce([])
    const w = mountComponent()
    await flushPromises()

    const input = w.find('input[type=number]')
    await input.setValue('abc')

    const buttons = w.findAll('button')
    const addBtn = buttons.find((b) => b.text() === 'dashboard.groupQuotaCard.add')
    await addBtn?.trigger('click')
    await flushPromises()

    expect(apiMocks.addGroupViewGrant).not.toHaveBeenCalled()
  })

  it('revoke: confirm true → calls API → refetches list', async () => {
    confirmMock.mockResolvedValueOnce(true)
    apiMocks.getGroupViewGrants
      .mockResolvedValueOnce(mockGrants)
      .mockResolvedValueOnce([mockGrants[0]])
    const w = mountComponent()
    await flushPromises()

    const revokeBtns = w.findAll('button').filter((b) =>
      b.text() === 'dashboard.groupQuotaCard.revoke',
    )
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

    const revokeBtns = w.findAll('button').filter((b) =>
      b.text() === 'dashboard.groupQuotaCard.revoke',
    )
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
    apiMocks.getGroupViewGrants.mockRejectedValueOnce(
      new Error('network error'),
    )
    const w = mountComponent()
    await flushPromises()

    expect(w.html()).toContain('network error')
  })
})
