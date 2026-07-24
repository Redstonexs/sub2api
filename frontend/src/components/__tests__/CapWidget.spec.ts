import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CapCaptchaWidget from '../CapCaptchaWidget.vue'

vi.mock('cap-widget', () => ({}))

describe('CapCaptchaWidget', () => {
  it('renders the official custom element and emits a solved token', async () => {
    const wrapper = mount(CapCaptchaWidget, {
      props: {
        apiEndpoint: 'https://cap.example.com',
        siteKey: 'public-site-key'
      }
    })

    await flushPromises()

    const widget = wrapper.get('cap-widget')
    expect(widget.attributes('data-cap-api-endpoint')).toBe('https://cap.example.com/public-site-key/')

    await HTMLElement.prototype.dispatchEvent.call(widget.element, new CustomEvent('solve', {
      bubbles: true,
      composed: true,
      detail: { token: 'cap-token' }
    }))
    expect(wrapper.emitted('verify')).toEqual([['cap-token']])
  })
})
