import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it, vi } from 'vitest'
import SetupWizardView from '@/views/setup/SetupWizardView.vue'
import messages from '@/i18n/locales/en'

const { testDatabase, testRedis, install } = vi.hoisted(() => ({
  testDatabase: vi.fn().mockResolvedValue(undefined),
  testRedis: vi.fn().mockResolvedValue(undefined),
  install: vi.fn()
}))

vi.mock('@/api/setup', () => ({
  testDatabase,
  testRedis,
  install,
  setBootstrapToken: vi.fn(),
  clearBootstrapToken: vi.fn()
}))

describe('SetupWizardView bootstrap unlock', () => {
  it('gates connection tests and installation on a nonempty token', async () => {
    const wrapper = mount(SetupWizardView, {
      global: {
        plugins: [createI18n({ legacy: false, locale: 'en', messages: { en: messages } })],
        stubs: { Icon: true, Select: true, Toggle: true }
      }
    })

    const token = wrapper.find('#bootstrap-token')
    const buttons = () => wrapper.findAll('button')
    expect(buttons()[0].attributes('disabled')).toBeDefined()

    await token.setValue('bootstrap-secret')
    expect(buttons()[0].attributes('disabled')).toBeUndefined()
    await buttons()[0].trigger('click')
    await buttons()[1].trigger('click')
    await buttons()[0].trigger('click')
    await buttons()[1].trigger('click')

    const adminInputs = wrapper.findAll('input')
    await adminInputs[1].setValue('admin@example.com')
    await adminInputs[2].setValue('password')
    await adminInputs[3].setValue('password')
    await buttons()[1].trigger('click')

    await token.setValue('')
    expect(buttons()[0].attributes('disabled')).toBeDefined()
    expect(install).not.toHaveBeenCalled()
    expect(testDatabase).toHaveBeenCalledOnce()
    expect(testRedis).toHaveBeenCalledOnce()
  })
})
