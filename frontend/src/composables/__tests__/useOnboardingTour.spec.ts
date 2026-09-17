import { describe, expect, it, vi, afterEach } from 'vitest'
import { defineComponent } from 'vue'
import { enableAutoUnmount, mount } from '@vue/test-utils'

enableAutoUnmount(afterEach)

// Static dependencies are mocked so the composable can be exercised in isolation.
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { id: 'test-user', role: 'admin' },
    isSimpleMode: false
  })
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    getDriverInstance: () => null,
    setDriverInstance: vi.fn(),
    isDriverActive: () => false,
    setControlMethods: vi.fn(),
    clearControlMethods: vi.fn()
  })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/components/Guide/steps', () => ({
  getAdminSteps: () => [],
  getUserSteps: () => []
}))

function createMockDriver() {
  return {
    destroy: vi.fn(),
    isActive: vi.fn(() => true),
    getActiveIndex: vi.fn(() => 0),
    getActiveElement: vi.fn(() => null),
    moveNext: vi.fn(),
    movePrevious: vi.fn(),
    drive: vi.fn()
  }
}

function mountTour(useOnboardingTour: typeof import('../useOnboardingTour')['useOnboardingTour']) {
  return mount(defineComponent({
    setup: () => useOnboardingTour({ autoStart: false }),
    render: () => null
  })).vm
}

describe('useOnboardingTour driver.js lazy loading', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unmock('driver.js')
    vi.unmock('driver.js/dist/driver.css')
    vi.resetModules()
  })

  it('does not load driver.js until startTour is invoked', async () => {
    let driverModuleLoaded = false
    const mockDriver = createMockDriver()
    const driverFactory = vi.fn(() => mockDriver)

    vi.doMock('driver.js', () => {
      driverModuleLoaded = true
      return { driver: driverFactory }
    })
    vi.doMock('driver.js/dist/driver.css', () => ({}))

    vi.resetModules()
    const { useOnboardingTour } = await import('../useOnboardingTour')

    // Importing the composable must not pull driver.js into the bundle.
    expect(driverModuleLoaded).toBe(false)
    expect(driverFactory).not.toHaveBeenCalled()

    const tour = mountTour(useOnboardingTour)
    await tour.startTour()

    // Only starting the tour triggers the dynamic import + driver construction.
    expect(driverModuleLoaded).toBe(true)
    expect(driverFactory).toHaveBeenCalledTimes(1)
    expect(mockDriver.drive).toHaveBeenCalledWith(0)
  })

  it('aborts the tour without an unhandled rejection when driver.js fails to load', async () => {
    const errorLog = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.doMock('driver.js', () => {
      throw new Error('driver.js chunk failed to load')
    })
    vi.doMock('driver.js/dist/driver.css', () => ({}))

    vi.resetModules()
    const { useOnboardingTour } = await import('../useOnboardingTour')

    const tour = mountTour(useOnboardingTour)
    await expect(tour.startTour()).resolves.toBeUndefined()
    expect(errorLog).toHaveBeenCalledTimes(1)
    expect(errorLog).toHaveBeenCalledWith(
      'Onboarding: failed to load driver.js, tour aborted:', expect.any(Error)
    )
  })

  it('aborts the tour without an unhandled rejection when the CSS fails to load', async () => {
    const errorLog = vi.spyOn(console, 'error').mockImplementation(() => {})
    const driverFactory = vi.fn(() => createMockDriver())
    vi.doMock('driver.js', () => ({ driver: driverFactory }))
    vi.doMock('driver.js/dist/driver.css', () => {
      throw new Error('driver.css chunk failed to load')
    })

    vi.resetModules()
    const { useOnboardingTour } = await import('../useOnboardingTour')

    const tour = mountTour(useOnboardingTour)
    await expect(tour.startTour()).resolves.toBeUndefined()
    expect(errorLog).toHaveBeenCalledTimes(1)
    expect(errorLog).toHaveBeenCalledWith(
      'Onboarding: failed to load driver.js, tour aborted:', expect.any(Error)
    )
    // CSS failure must prevent driver construction.
    expect(driverFactory).not.toHaveBeenCalled()
  })
})
