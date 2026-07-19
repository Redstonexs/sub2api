import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import PaymentView from '../PaymentView.vue'

const routeState = vi.hoisted(() => ({
  path: '/purchase',
  query: {},
}))

const getCheckoutInfo = vi.hoisted(() => vi.fn())

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({
      push: vi.fn(),
      replace: vi.fn().mockResolvedValue(undefined),
      resolve: vi.fn(() => ({ href: '/payment/stripe?mock=1' })),
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { username: 'demo-user', balance: 0 },
    refreshUser: vi.fn(),
  }),
}))

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => ({
    createOrder: vi.fn(),
  }),
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({
    activeSubscriptions: [],
    fetchActiveSubscriptions: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showInfo: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getCheckoutInfo,
  },
}))

vi.mock('@/utils/device', () => ({
  isMobileDevice: () => true,
}))

function checkoutInfoFixture(overrides: Record<string, unknown> = {}) {
  return {
    data: {
      methods: {},
      global_min: 0,
      global_max: 0,
      plans: [],
      balance_disabled: false,
      balance_recharge_multiplier: 1,
      subscription_usd_to_cny_rate: 0,
      recharge_fee_rate: 0,
      help_text: '',
      help_image_url: '',
      stripe_publishable_key: '',
      ...overrides,
    },
  }
}

async function mountPaymentView(checkout: Record<string, unknown>) {
  getCheckoutInfo.mockResolvedValue(checkoutInfoFixture(checkout))

  const wrapper = shallowMount(PaymentView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        SubscriptionPlanCard: {
          props: ['plan'],
          emits: ['select'],
          template: '<button class="subscription-plan-card" @click="$emit(\'select\', plan)">{{ plan.name }}</button>',
        },
        Teleport: true,
        Transition: false,
      },
    },
  })

  await flushPromises()
  await flushPromises()
  return wrapper
}

describe('PaymentView purchase availability', () => {
  beforeEach(() => {
    routeState.query = {}
    getCheckoutInfo.mockReset()
  })

  it('shows only sale-enabled subscriptions when balance purchasing is closed', async () => {
    // Given
    const salePlan = {
      id: 7,
      group_id: 3,
      name: 'Starter',
      description: '',
      price: 128,
      original_price: 0,
      validity_days: 30,
      validity_unit: 'day',
      rate_multiplier: 1,
      daily_limit_usd: null,
      weekly_limit_usd: null,
      monthly_limit_usd: null,
      features: [],
      group_platform: 'openai',
      sort_order: 1,
      group_name: 'OpenAI',
    }

    // When
    const wrapper = await mountPaymentView({
      balance_purchase_enabled: false,
      subscription_purchase_enabled: true,
      plans: [salePlan],
    })

    // Then
    expect(wrapper.text()).not.toContain('payment.tabTopUp')
    expect(wrapper.findAll('.subscription-plan-card')).toHaveLength(1)
    expect(wrapper.text()).toContain('Starter')
  })

  it('shows an unavailable state when both purchase types are closed', async () => {
    // Given / When
    const wrapper = await mountPaymentView({
      balance_purchase_enabled: false,
      subscription_purchase_enabled: false,
    })

    // Then
    expect(wrapper.text()).toContain('payment.purchaseUnavailable')
    expect(wrapper.findAll('button')).toHaveLength(0)
  })

  it('keeps only balance top-up available when subscriptions are closed', async () => {
    // Given / When
    const wrapper = await mountPaymentView({
      balance_purchase_enabled: true,
      subscription_purchase_enabled: false,
    })

    // Then
    expect(wrapper.text()).toContain('payment.rechargeAccount')
    expect(wrapper.text()).not.toContain('payment.tabSubscribe')
    expect(wrapper.text()).not.toContain('payment.noPlans')
    expect(wrapper.findAll('.subscription-plan-card')).toHaveLength(0)
  })

  it('preserves the legacy balance-disabled guard when balance purchasing is otherwise open', async () => {
    // Given / When
    const wrapper = await mountPaymentView({
      balance_disabled: true,
      balance_purchase_enabled: true,
      subscription_purchase_enabled: true,
    })

    // Then
    expect(wrapper.text()).not.toContain('payment.tabTopUp')
    expect(wrapper.text()).toContain('payment.noPlans')
  })

  it('removes tabpanel semantics while a subscription confirmation replaces the tabs', async () => {
    // Given
    const wrapper = await mountPaymentView({
      balance_purchase_enabled: true,
      subscription_purchase_enabled: true,
      plans: [{
        id: 7,
        group_id: 3,
        name: 'Starter',
        description: '',
        price: 128,
        original_price: 0,
        validity_days: 30,
        validity_unit: 'day',
        rate_multiplier: 1,
        daily_limit_usd: null,
        weekly_limit_usd: null,
        monthly_limit_usd: null,
        features: [],
        group_platform: 'openai',
        sort_order: 1,
        group_name: 'OpenAI',
      }],
    })

    // When
    await wrapper.get('#purchase-tab-subscription').trigger('click')
    await wrapper.get('.subscription-plan-card').trigger('click')

    // Then
    expect(wrapper.find('[role="tablist"]').exists()).toBe(false)
    expect(wrapper.find('#payment-purchase-panel').exists()).toBe(false)
    expect(wrapper.find('[role="tabpanel"]').exists()).toBe(false)
  })
})
