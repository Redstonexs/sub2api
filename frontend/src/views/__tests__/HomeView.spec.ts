import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import { nextTick } from 'vue'

import HomeView from '../HomeView.vue'
import type { PublicSettings, User } from '@/types'

const fetchPublicSettings = vi.fn()
const checkAuth = vi.fn()

const appStoreState = {
  cachedPublicSettings: null as PublicSettings | null,
  siteName: 'Sub2API',
  siteLogo: '',
  docUrl: '',
  publicSettingsLoaded: true,
  fetchPublicSettings,
  showInfo: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
}

const authStoreState = {
  isAuthenticated: false,
  isAdmin: false,
  user: null as User | null,
  checkAuth,
}

const messages: Record<string, string> = {
  'homeV2.nav.locale': 'Locale',
  'homeV2.nav.theme': 'Theme',
  'homeV2.nav.login': 'Login',
  'homeV2.nav.dashboard': 'Dashboard',
  'homeV2.hero.title': 'Sub2API',
  'homeV2.hero.subtitle': 'AI API Gateway',
  'homeV2.cta.start': 'Get Started',
  'homeV2.footer.copyright': 'All rights reserved',
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
  useAuthStore: () => authStoreState,
}))

function mountHomeView() {
  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
        LocaleSwitcher: true,
        Icon: true,
      },
    },
  })
}

async function mountHomeViewWithRouter() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div>Home</div>' } },
      { path: '/home', redirect: '/' },
      { path: '/login', component: { template: '<div>Login</div>' } },
      { path: '/dashboard', component: { template: '<div>Dashboard</div>' } },
      { path: '/admin/dashboard', component: { template: '<div>Admin</div>' } },
    ],
  })
  await router.push('/home')
  expect(router.currentRoute.value.path).toBe('/')
  await router.isReady()

  const wrapper = mount(HomeView, {
    global: {
      plugins: [router],
      stubs: {
        LocaleSwitcher: true,
        Icon: true,
      },
    },
  })

  await flushPromises()
  await nextTick()

  return { wrapper, router }
}

function expectLinkHref(
  wrapper: ReturnType<typeof mount>,
  selector: string,
  expected: string
): void {
  const el = wrapper.find(selector)
  expect(el.exists()).toBe(true)
  const link = el.find('a')
  const href = link.exists() ? link.attributes('href') : el.attributes('href')
  expect(href).toBe(expected)
}

describe('HomeView', () => {
  beforeEach(() => {
    appStoreState.cachedPublicSettings = null
    appStoreState.siteName = 'Sub2API'
    appStoreState.siteLogo = ''
    appStoreState.docUrl = ''
    appStoreState.publicSettingsLoaded = true
    fetchPublicSettings.mockReset()

    authStoreState.isAuthenticated = false
    authStoreState.isAdmin = false
    authStoreState.user = null
    checkAuth.mockReset()

    localStorage.clear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('S1 default page structure', () => {
    it('renders hero with non-empty text', async () => {
      const wrapper = mountHomeView()
      await flushPromises()
      await nextTick()

      const hero = wrapper.find('[data-testid="home-hero"]')
      expect(hero.exists()).toBe(true)
      expect(hero.text().trim().length).toBeGreaterThan(0)

      wrapper.unmount()
    })

    it('renders routing diagram', async () => {
      const wrapper = mountHomeView()
      await flushPromises()
      await nextTick()

      expect(wrapper.find('[data-testid="home-routing-diagram"]').exists()).toBe(true)

      wrapper.unmount()
    })

    it('renders nav with locale, theme, and auth controls', async () => {
      const wrapper = mountHomeView()
      await flushPromises()
      await nextTick()

      const nav = wrapper.find('[data-testid="home-nav"]')
      expect(nav.exists()).toBe(true)
      expect(nav.find('[data-testid="home-nav-locale"]').exists()).toBe(true)
      expect(nav.find('[data-testid="home-nav-theme"]').exists()).toBe(true)
      expect(nav.find('[data-testid="home-nav-auth"]').exists()).toBe(true)

      wrapper.unmount()
    })

    it('renders quickstart, providers, CTA band, and footer', async () => {
      const wrapper = mountHomeView()
      await flushPromises()
      await nextTick()

      expect(wrapper.find('[data-testid="home-quickstart"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="home-providers"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="home-cta-band"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="home-footer"]').exists()).toBe(true)

      wrapper.unmount()
    })
  })

  describe('M1 responsive routing diagram', () => {
    it('renders a simplified mobile variant and keeps the full variant desktop-only', async () => {
      const wrapper = mountHomeView()
      await flushPromises()
      await nextTick()

      const diagram = wrapper.find('[data-testid="home-routing-diagram"]')
      expect(diagram.exists()).toBe(true)

      const desktop = wrapper.find('[data-testid="home-routing-diagram-desktop"]')
      expect(desktop.exists()).toBe(true)
      expect(desktop.classes()).toContain('hidden')
      expect(desktop.classes()).toContain('lg:block')
      expect(desktop.find('svg').exists()).toBe(true)

      const mobile = wrapper.find('[data-testid="home-routing-diagram-mobile"]')
      expect(mobile.exists()).toBe(true)
      expect(mobile.classes()).toContain('lg:hidden')
      expect(mobile.find('svg').exists()).toBe(true)

      wrapper.unmount()
    })
  })

  describe('S2 home_content admin override', () => {
    it('renders iframe override and hides hero when home_content is a URL', async () => {
      appStoreState.cachedPublicSettings = {
        home_content: 'https://example.com/custom',
      } as unknown as PublicSettings

      const wrapper = mountHomeView()
      await flushPromises()
      await nextTick()

      const iframe = wrapper.find('[data-testid="home-iframe-override"]')
      expect(iframe.exists()).toBe(true)
      expect(iframe.attributes('src')).toBe('https://example.com/custom')
      expect(wrapper.find('[data-testid="home-hero"]').exists()).toBe(false)

      wrapper.unmount()
    })

    it('renders HTML override and hides hero when home_content is HTML', async () => {
      appStoreState.cachedPublicSettings = {
        home_content: '<div class="custom">Hello Custom</div>',
      } as unknown as PublicSettings

      const wrapper = mountHomeView()
      await flushPromises()
      await nextTick()

      const htmlOverride = wrapper.find('[data-testid="home-html-override"]')
      expect(htmlOverride.exists()).toBe(true)
      expect(htmlOverride.html()).toContain('Hello Custom')
      expect(wrapper.find('[data-testid="home-hero"]').exists()).toBe(false)

      wrapper.unmount()
    })
  })

  describe('S3 auth-aware CTA', () => {
    it('guest: nav auth points to /login and CTA start points to /login', async () => {
      authStoreState.isAuthenticated = false
      authStoreState.isAdmin = false
      authStoreState.user = null

      const { wrapper } = await mountHomeViewWithRouter()

      expectLinkHref(wrapper, '[data-testid="home-nav-auth"]', '/login')
      expectLinkHref(wrapper, '[data-testid="home-cta-start"]', '/login')

      wrapper.unmount()
    })

    it('authenticated non-admin: nav auth points to /dashboard', async () => {
      authStoreState.isAuthenticated = true
      authStoreState.isAdmin = false
      authStoreState.user = {
        email: 'user@example.com',
        role: 'user',
      } as unknown as User

      const { wrapper } = await mountHomeViewWithRouter()

      expectLinkHref(wrapper, '[data-testid="home-nav-auth"]', '/dashboard')

      wrapper.unmount()
    })

    it('authenticated admin: nav auth points to /admin/dashboard', async () => {
      authStoreState.isAuthenticated = true
      authStoreState.isAdmin = true
      authStoreState.user = {
        email: 'admin@example.com',
        role: 'admin',
      } as unknown as User

      const { wrapper } = await mountHomeViewWithRouter()

      expectLinkHref(wrapper, '[data-testid="home-nav-auth"]', '/admin/dashboard')

      wrapper.unmount()
    })
  })
})
