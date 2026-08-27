import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'

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

describe('useOnboardingTour driver.js lazy loading', () => {
  beforeEach(() => {
    // useOnboardingTour registers lifecycle hooks; calling it outside a
    // component setup triggers benign Vue warnings. Silence them for clarity.
    vi.spyOn(console, 'warn').mockImplementation(() => {})
  })

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

    const tour = useOnboardingTour({ autoStart: false })
    await tour.startTour()

    // Only starting the tour triggers the dynamic import + driver construction.
    expect(driverModuleLoaded).toBe(true)
    expect(driverFactory).toHaveBeenCalledTimes(1)
    expect(mockDriver.drive).toHaveBeenCalledWith(0)
  })

  it('aborts the tour without an unhandled rejection when driver.js fails to load', async () => {
    vi.doMock('driver.js', () => {
      throw new Error('driver.js chunk failed to load')
    })
    vi.doMock('driver.js/dist/driver.css', () => ({}))

    vi.resetModules()
    const { useOnboardingTour } = await import('../useOnboardingTour')

    const tour = useOnboardingTour({ autoStart: false })
    await expect(tour.startTour()).resolves.toBeUndefined()
  })

  it('aborts the tour without an unhandled rejection when the CSS fails to load', async () => {
    const driverFactory = vi.fn(() => createMockDriver())
    vi.doMock('driver.js', () => ({ driver: driverFactory }))
    vi.doMock('driver.js/dist/driver.css', () => {
      throw new Error('driver.css chunk failed to load')
    })

    vi.resetModules()
    const { useOnboardingTour } = await import('../useOnboardingTour')

    const tour = useOnboardingTour({ autoStart: false })
    await expect(tour.startTour()).resolves.toBeUndefined()
    // CSS failure must prevent driver construction.
    expect(driverFactory).not.toHaveBeenCalled()
  })
})
