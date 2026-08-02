import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GroupQuotaCard from '../GroupQuotaCard.vue'
import UsageProgressBar from '@/components/account/UsageProgressBar.vue'
import Select from '@/components/common/Select.vue'
import type { GroupQuotaCard as GroupQuotaCardType, ViewableGroup, AdminGroup } from '@/types'

// Mock i18n — return keys so assertions check for 'dashboard.groupQuotaCard.title' etc.
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

// Mock admin API modules
const mockAdminGetAll = vi.fn()
const mockAdminGetGroupQuotaCard = vi.fn()
vi.mock('@/api/admin/groups', () => ({
  groupsAPI: {
    getAll: (...args: unknown[]) => mockAdminGetAll(...args)
  }
}))
vi.mock('@/api/admin/dashboard', () => ({
  dashboardAPI: {
    getGroupQuotaCard: (...args: unknown[]) => mockAdminGetGroupQuotaCard(...args)
  }
}))

// Mock user API module
const mockGetMyViewableGroups = vi.fn()
const mockUserGetGroupQuotaCard = vi.fn()
vi.mock('@/api/user', () => ({
  userAPI: {
    getMyViewableGroups: (...args: unknown[]) => mockGetMyViewableGroups(...args),
    getGroupQuotaCard: (...args: unknown[]) => mockUserGetGroupQuotaCard(...args)
  }
}))

const SAMPLE_GROUPS: ViewableGroup[] = [
  { group_id: 1, group_name: 'Claude Group', platform: 'anthropic' },
  { group_id: 2, group_name: 'OpenAI Group', platform: 'openai' }
]

const SAMPLE_ADMIN_GROUPS: AdminGroup[] = [
  { id: 1, name: 'Claude Group', platform: 'anthropic', status: 'active', subscription_type: 'standard', rate_multiplier: 1, is_exclusive: false, description: null, model_routing: null, model_routing_enabled: false, mcp_xml_inject: false, allow_image_generation: false, allow_batch_image_generation: false, image_rate_independent: false, image_rate_multiplier: 1, batch_image_discount_multiplier: 1, batch_image_hold_multiplier: 1, video_rate_independent: false, video_rate_multiplier: 1, allow_messages_dispatch: false, allow_live: false, require_oauth_only: false, require_privacy_set: false, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 2, name: 'OpenAI Group', platform: 'openai', status: 'active', subscription_type: 'standard', rate_multiplier: 1, is_exclusive: false, description: null, model_routing: null, model_routing_enabled: false, mcp_xml_inject: false, allow_image_generation: false, allow_batch_image_generation: false, image_rate_independent: false, image_rate_multiplier: 1, batch_image_discount_multiplier: 1, batch_image_hold_multiplier: 1, video_rate_independent: false, video_rate_multiplier: 1, allow_messages_dispatch: false, allow_live: false, require_oauth_only: false, require_privacy_set: false, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }
]

const SAMPLE_CARD: GroupQuotaCardType = {
  group_id: 1,
  group_name: 'Claude Group',
  platform: 'anthropic',
  total_remaining_5h: 62.5,
  total_remaining_7d: 120.0,
  accounts: [
    {
      account_id: 10,
      display_name: 'account-alpha',
      five_hour: { utilization: 80, resets_at: '2026-03-17T05:00:00Z' },
      seven_day: { utilization: 45, resets_at: '2026-03-24T00:00:00Z' }
    },
    {
      account_id: null,
      display_name: '账号1',
      five_hour: { utilization: 60, resets_at: '2026-03-17T05:00:00Z' },
      seven_day: null
    }
  ]
}

describe('GroupQuotaCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ========== RENDER TESTS ==========

  it('renders totals from card data — bar utilization = 100 - remaining', async () => {
    // Empty accounts so only the two totals bars render — clean assertion target.
    const totalsCard: GroupQuotaCardType = {
      group_id: 1, group_name: 'Test', platform: 'anthropic',
      total_remaining_5h: 62.5, total_remaining_7d: 30, accounts: []
    }
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockResolvedValue(totalsCard)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    const bars = wrapper.findAllComponents(UsageProgressBar)
    // Two totals bars only (no accounts → no per-account bars).
    expect(bars).toHaveLength(2)
    // 5h: total_remaining 62.5 → utilization = 100 - 62.5 = 37.5
    expect(bars[0].props('utilization')).toBe(37.5)
    // 7d: total_remaining 30 → utilization = 100 - 30 = 70
    expect(bars[1].props('utilization')).toBe(70)
    // Labels present
    expect(wrapper.text()).toContain('dashboard.groupQuotaCard.totalRemaining5h')
    expect(wrapper.text()).toContain('dashboard.groupQuotaCard.totalRemaining7d')
  })

  it('totals show noData "—" when total_remaining_* are null', async () => {
    const nullCard: GroupQuotaCardType = {
      group_id: 1, group_name: 'Test', platform: 'anthropic',
      total_remaining_5h: null, total_remaining_7d: null, accounts: []
    }
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockResolvedValue(nullCard)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    // Both totals null → noData marker rendered for each; no UsageProgressBar.
    expect(wrapper.findAllComponents(UsageProgressBar)).toHaveLength(0)
    // noData key appears twice (one per null total).
    const text = wrapper.text()
    const noDataCount = text.split('dashboard.groupQuotaCard.noData').length - 1
    expect(noDataCount).toBe(2)
  })

  it('per-account rows render display_name + both window bars; null window → noData', async () => {
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockResolvedValue(SAMPLE_CARD)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    // display_names present (server-provided, rendered verbatim)
    expect(wrapper.text()).toContain('account-alpha')
    expect(wrapper.text()).toContain('账号1')
    // 账号1 has seven_day null → noData marker rendered for that window
    expect(wrapper.text()).toContain('dashboard.groupQuotaCard.noData')
  })

  it('renders accounts in server-provided order — no client-side re-sort', async () => {
    // Feed unsorted accounts; verify DOM order matches input order
    const unsortedCard: GroupQuotaCardType = {
      group_id: 1, group_name: 'Test', platform: 'openai',
      total_remaining_5h: 50, total_remaining_7d: 100,
      accounts: [
        { account_id: 3, display_name: 'Zebra', five_hour: { utilization: 10, resets_at: null }, seven_day: { utilization: 20, resets_at: null } },
        { account_id: 1, display_name: 'Alpha', five_hour: { utilization: 90, resets_at: null }, seven_day: { utilization: 80, resets_at: null } },
        { account_id: 2, display_name: 'Middle', five_hour: { utilization: 50, resets_at: null }, seven_day: { utilization: 60, resets_at: null } }
      ]
    }
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockResolvedValue(unsortedCard)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    const text = wrapper.text()
    const zebraIdx = text.indexOf('Zebra')
    const alphaIdx = text.indexOf('Alpha')
    const middleIdx = text.indexOf('Middle')
    expect(zebraIdx).toBeLessThan(alphaIdx)
    expect(alphaIdx).toBeLessThan(middleIdx)
  })

  it('sort toggle click switches active button and re-calls API with sortBy 7d', async () => {
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockResolvedValue(SAMPLE_CARD)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    // Initial call with default sortBy '5h'
    expect(mockUserGetGroupQuotaCard).toHaveBeenCalledTimes(1)
    expect(mockUserGetGroupQuotaCard).toHaveBeenCalledWith(1, '5h')

    // 5h button active initially, 7d inactive
    expect(wrapper.get('[data-testid="sort-5h"]').classes()).toContain('bg-primary-50')
    expect(wrapper.get('[data-testid="sort-7d"]').classes()).not.toContain('bg-primary-50')

    // Click the 7d sort button
    await wrapper.get('[data-testid="sort-7d"]').trigger('click')
    await flushPromises()

    expect(mockUserGetGroupQuotaCard).toHaveBeenCalledTimes(2)
    expect(mockUserGetGroupQuotaCard).toHaveBeenLastCalledWith(1, '7d')
    // Active state switched to 7d
    expect(wrapper.get('[data-testid="sort-7d"]').classes()).toContain('bg-primary-50')
    expect(wrapper.get('[data-testid="sort-5h"]').classes()).not.toContain('bg-primary-50')
  })

  it('changes the selected group through the shared selector', async () => {
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockResolvedValue(SAMPLE_CARD)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    await wrapper.findComponent(Select).vm.$emit('update:modelValue', 2)
    await flushPromises()

    expect(mockUserGetGroupQuotaCard).toHaveBeenLastCalledWith(2, '5h')
    expect(wrapper.findComponent(Select).props('modelValue')).toBe(2)
    expect(mockAdminGetAll).not.toHaveBeenCalled()
    expect(mockAdminGetGroupQuotaCard).not.toHaveBeenCalled()
  })

  it('keeps the controls and cached usage visible while a new quota window loads', async () => {
    let resolveRefresh: (value: GroupQuotaCardType) => void = () => {
      throw new Error('Refresh resolver was not initialized')
    }
    const pendingRefresh = new Promise<GroupQuotaCardType>((resolve) => {
      resolveRefresh = resolve
    })
    const refreshedCard: GroupQuotaCardType = {
      ...SAMPLE_CARD,
      total_remaining_5h: 40,
      accounts: [{ ...SAMPLE_CARD.accounts[0], display_name: 'account-bravo' }]
    }

    // Given cached quota data and a pending refresh.
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard
      .mockResolvedValueOnce(SAMPLE_CARD)
      .mockImplementationOnce(() => pendingRefresh)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    // When the user changes the displayed quota window.
    await wrapper.get('[data-testid="sort-7d"]').trigger('click')

    // Then controls and the cached data stay available behind a local loading status.
    expect(wrapper.get('[data-testid="sort-5h"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sort-7d"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="group-quota-content"]').classes()).not.toContain('group-quota-data-leave-active')
    expect(wrapper.text()).toContain('account-alpha')
    expect(wrapper.get('[data-testid="group-quota-data-viewport"]').attributes('aria-busy')).toBe('true')
    expect(wrapper.get('[data-testid="group-quota-data-loading"]').exists()).toBe(true)

    resolveRefresh(refreshedCard)
    await flushPromises()

    await vi.waitFor(() => {
      expect(wrapper.find('[data-testid="group-quota-data-loading"]').exists()).toBe(false)
    })
    expect(wrapper.text()).toContain('account-bravo')
  })

  // ========== MODE TESTS ==========

  it('user mode: calls userAPI.getGroupQuotaCard + userAPI.getMyViewableGroups; admin API NOT called', async () => {
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockResolvedValue(SAMPLE_CARD)

    mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    expect(mockGetMyViewableGroups).toHaveBeenCalled()
    expect(mockUserGetGroupQuotaCard).toHaveBeenCalled()
    expect(mockAdminGetAll).not.toHaveBeenCalled()
    expect(mockAdminGetGroupQuotaCard).not.toHaveBeenCalled()
  })

  it('admin mode: calls admin groups.getAll + dashboardAPI.getGroupQuotaCard; user API NOT called', async () => {
    mockAdminGetAll.mockResolvedValue(SAMPLE_ADMIN_GROUPS)
    mockAdminGetGroupQuotaCard.mockResolvedValue(SAMPLE_CARD)

    mount(GroupQuotaCard, { props: { isAdmin: true } })
    await flushPromises()

    expect(mockAdminGetAll).toHaveBeenCalled()
    expect(mockAdminGetGroupQuotaCard).toHaveBeenCalled()
    expect(mockGetMyViewableGroups).not.toHaveBeenCalled()
    expect(mockUserGetGroupQuotaCard).not.toHaveBeenCalled()
  })

  it('admin mode: filters groups to anthropic|openai only (gemini excluded)', async () => {
    const mixedGroups: AdminGroup[] = [
      ...SAMPLE_ADMIN_GROUPS,
      { id: 3, name: 'Gemini Group', platform: 'gemini', status: 'active', subscription_type: 'standard', rate_multiplier: 1, is_exclusive: false, description: null, model_routing: null, model_routing_enabled: false, mcp_xml_inject: false, allow_image_generation: false, allow_batch_image_generation: false, image_rate_independent: false, image_rate_multiplier: 1, batch_image_discount_multiplier: 1, batch_image_hold_multiplier: 1, video_rate_independent: false, video_rate_multiplier: 1, allow_messages_dispatch: false, allow_live: false, require_oauth_only: false, require_privacy_set: false, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }
    ]
    mockAdminGetAll.mockResolvedValue(mixedGroups)
    mockAdminGetGroupQuotaCard.mockResolvedValue(SAMPLE_CARD)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: true } })
    await flushPromises()

    expect(mockAdminGetAll).toHaveBeenCalled()
    // Gemini is filtered out before options are passed to the shared selector.
    expect(wrapper.findComponent(Select).props('options')).toEqual([
      { value: 1, label: 'Claude Group' },
      { value: 2, label: 'OpenAI Group' }
    ])
  })

  // ========== EMPTY STATE TESTS ==========

  it('user with no viewable groups → quota card is absent', async () => {
    mockGetMyViewableGroups.mockResolvedValue([])

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    expect(wrapper.find('.group-quota-card').exists()).toBe(false)
    expect(mockUserGetGroupQuotaCard).not.toHaveBeenCalled()
  })

  it('anonymized display: renders server display_name verbatim without client renumbering', async () => {
    const anonymizedCard: GroupQuotaCardType = {
      group_id: 1, group_name: 'Test', platform: 'anthropic',
      total_remaining_5h: 50, total_remaining_7d: 80,
      accounts: [
        { account_id: null, display_name: '账号1', five_hour: { utilization: 50, resets_at: null }, seven_day: { utilization: 30, resets_at: null } },
        { account_id: null, display_name: '账号2', five_hour: { utilization: 70, resets_at: null }, seven_day: { utilization: 50, resets_at: null } }
      ]
    }
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockResolvedValue(anonymizedCard)

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    expect(wrapper.text()).toContain('账号1')
    expect(wrapper.text()).toContain('账号2')
  })

  // ========== ERROR STATE ==========

  it('error state: shows loadFailed message with retry button', async () => {
    mockGetMyViewableGroups.mockResolvedValue(SAMPLE_GROUPS)
    mockUserGetGroupQuotaCard.mockRejectedValue(new Error('Network error'))

    const wrapper = mount(GroupQuotaCard, { props: { isAdmin: false } })
    await flushPromises()

    expect(wrapper.text()).toContain('dashboard.groupQuotaCard.loadFailed')
    // Retry button present
    const retryBtn = wrapper.findAll('button').find(b => b.text().includes('Retry'))
    expect(retryBtn).toBeTruthy()
  })
})
