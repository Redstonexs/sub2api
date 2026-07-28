import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'

import Select from '../Select.vue'

const messages: Record<string, string> = {
  'common.selectOption': '请选择一个选项'
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
      locale: ref('zh-CN')
    })
  }
})

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

const originalInnerWidth = window.innerWidth
let unmountWrapper: (() => void) | undefined

const setViewportWidth = (width: number) => {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: width,
  })
}

const mockTriggerRect = (left: number, width: number) => {
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: left,
    y: 20,
    top: 20,
    right: left + width,
    bottom: 60,
    left,
    width,
    height: 40,
    toJSON: () => ({}),
  })
}

const openSelect = async () => {
  const wrapper = mount(Select, {
    props: {
      modelValue: null,
      options: [
        {
          value: 'example',
          label: 'very-long-unbroken-option-value-that-must-not-overflow',
        },
      ],
    },
  })
  unmountWrapper = () => wrapper.unmount()

  await wrapper.get('button').trigger('click')
  await nextTick()

  return document.body.querySelector<HTMLElement>('.select-dropdown-portal')
}

afterEach(() => {
  unmountWrapper?.()
  unmountWrapper = undefined
  document.body.innerHTML = ''
  setViewportWidth(originalInnerWidth)
  vi.restoreAllMocks()
})

describe('Select dropdown viewport constraints', () => {
  it('preserves the existing 200px minimum width when space is available', async () => {
    setViewportWidth(1024)
    mockTriggerRect(20, 80)

    const dropdown = await openSelect()

    expect(dropdown).not.toBeNull()
    expect(dropdown?.style.left).toBe('20px')
    expect(dropdown?.style.minWidth).toBe('200px')
    expect(dropdown?.style.maxWidth).toBe('996px')
  })

  it('shrinks the minimum width to fit near the right viewport edge', async () => {
    setViewportWidth(320)
    mockTriggerRect(220, 80)

    const dropdown = await openSelect()

    expect(dropdown).not.toBeNull()
    expect(dropdown?.style.left).toBe('220px')
    expect(dropdown?.style.minWidth).toBe('92px')
    expect(dropdown?.style.maxWidth).toBe('92px')
  })

  it('clamps a trigger left of the viewport to the safe padding', async () => {
    setViewportWidth(320)
    mockTriggerRect(-20, 80)

    const dropdown = await openSelect()

    expect(dropdown).not.toBeNull()
    expect(dropdown?.style.left).toBe('8px')
    expect(dropdown?.style.minWidth).toBe('200px')
    expect(dropdown?.style.maxWidth).toBe('304px')
  })

  it('clamps an offscreen-right trigger position to the viewport boundary', async () => {
    setViewportWidth(320)
    mockTriggerRect(400, 80)

    const dropdown = await openSelect()

    expect(dropdown).not.toBeNull()
    expect(dropdown?.style.left).toBe('312px')
    expect(dropdown?.style.minWidth).toBe('0px')
    expect(dropdown?.style.maxWidth).toBe('0px')
  })
})
