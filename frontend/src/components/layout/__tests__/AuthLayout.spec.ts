import { describe, expect, it, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import AuthLayout from '../AuthLayout.vue'

const fetchPublicSettings = vi.fn()

const appStoreState: {
  cachedPublicSettings: { site_subtitle?: string } | null
  siteLogo: string
  siteName: string
  publicSettingsLoaded: boolean
  fetchPublicSettings: typeof fetchPublicSettings
} = {
  cachedPublicSettings: null,
  siteName: 'Sub2API',
  siteLogo: '',
  publicSettingsLoaded: true,
  fetchPublicSettings,
}

const messages: Record<string, string> = {
  'disclaimer.independentService':
    '{siteName} is an independent API gateway service and is not affiliated with or endorsed by Anthropic, OpenAI, Google, xAI, or any AI model provider.',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
      locale: { value: 'en' },
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => appStoreState,
}))

function mountAuthLayout() {
  return mount(AuthLayout)
}

describe('AuthLayout', () => {
  beforeEach(() => {
    appStoreState.cachedPublicSettings = null
    appStoreState.siteName = 'Sub2API'
    appStoreState.siteLogo = ''
    appStoreState.publicSettingsLoaded = true
    fetchPublicSettings.mockReset()
  })

  it('renders the independent-service disclaimer in the footer area', async () => {
    const wrapper = mountAuthLayout()
    await flushPromises()
    await nextTick()

    const text = wrapper.text()
    expect(text).toContain('independent API gateway')
    expect(text).toContain('not affiliated')

    wrapper.unmount()
  })
})
