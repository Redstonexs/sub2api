import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ProviderCard from '@/components/payment/ProviderCard.vue'
import type { ProviderInstance } from '@/types/payment'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function providerFactory(overrides: Partial<ProviderInstance> = {}): ProviderInstance {
  return {
    id: 1,
    provider_key: 'hashpay',
    name: 'HashPay',
    config: {},
    supported_types: ['hashpay'],
    enabled: true,
    payment_mode: '',
    refund_enabled: false,
    allow_user_refund: false,
    limits: '',
    sort_order: 0,
    ...overrides,
  }
}

function mountCard(provider = providerFactory()) {
  return mount(ProviderCard, {
    props: {
      provider,
      enabled: true,
      availableTypes: [{ value: 'hashpay', label: 'HashPay' }],
    },
    global: {
      stubs: {
        Icon: true,
        ToggleSwitch: {
          props: ['label', 'checked'],
          template: '<button :data-label="label" :data-checked="checked" />',
        },
      },
    },
  })
}

describe('ProviderCard HashPay refund safety', () => {
  it('does not expose refund controls for HashPay even if a legacy record enables them', () => {
    const wrapper = mountCard(providerFactory({ refund_enabled: true, allow_user_refund: true }))

    expect(wrapper.find('[data-label="admin.settings.payment.refundEnabled"]').exists()).toBe(false)
    expect(wrapper.find('[data-label="admin.settings.payment.allowUserRefund"]').exists()).toBe(false)
  })
})
