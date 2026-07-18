import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import Select from '../Select.vue'

const messages: Record<string, string> = {
  'common.selectOption': '请选择一个选项'
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => messages[key] ?? key,
    locale: ref('zh-CN')
  })
}))

const options = [
  { value: 'stripe', label: 'Stripe' },
  { value: 'hashpay', label: 'HashPay' }
]

const mountSelect = (props: Record<string, unknown> = {}) =>
  mount(Select, {
    props: {
      modelValue: null,
      options,
      ...props
    },
    global: {
      stubs: {
        Icon: true
      }
    }
  })

describe('Select accessibility naming', () => {
  it('uses the localized fallback when no accessible name is provided', () => {
    const wrapper = mountSelect()

    expect(wrapper.get('button.select-trigger').attributes('aria-label')).toBe('请选择一个选项')
    expect(wrapper.get('button.select-trigger').attributes('aria-labelledby')).toBeUndefined()
  })

  it('binds caller-provided id and aria-label to the trigger', () => {
    const wrapper = mountSelect({
      id: 'payment-provider-key',
      ariaLabel: 'Payment provider key'
    })

    const trigger = wrapper.get('button.select-trigger')
    expect(trigger.attributes('id')).toBe('payment-provider-key')
    expect(trigger.attributes('aria-label')).toBe('Payment provider key')
  })

  it('uses aria-labelledby without falling back to a generic label', () => {
    const wrapper = mountSelect({ ariaLabelledby: 'payment-provider-key-label' })

    const trigger = wrapper.get('button.select-trigger')
    expect(trigger.attributes('aria-labelledby')).toBe('payment-provider-key-label')
    expect(trigger.attributes('aria-label')).toBeUndefined()
  })

  it('binds aria-describedby to the trigger for caller-provided hints', () => {
    const wrapper = mountSelect({ ariaDescribedby: 'payment-provider-key-hint' })

    expect(wrapper.get('button.select-trigger').attributes('aria-describedby')).toBe('payment-provider-key-hint')
  })
})
