import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const motionMocks = vi.hoisted(() => {
  const contextRevert = vi.fn()
  const conditions = { desktop: false, reducedMotion: true }
  let conditionCleanup: void | (() => void)
  const mediaRevert = vi.fn(() => {
    conditionCleanup?.()
  })
  const mediaAdd = vi.fn((queries: object, callback: (context: { conditions?: Record<string, boolean> }) => void | (() => void)) => {
    void queries
    conditionCleanup = callback({ conditions })
  })
  const matchMedia = vi.fn(() => ({ add: mediaAdd, revert: mediaRevert }))
  const chain = {
    from: vi.fn(),
    set: vi.fn(),
    to: vi.fn()
  }
  const timelineOptions: Array<Record<string, object>> = []
  chain.from.mockReturnValue(chain)
  chain.set.mockReturnValue(chain)
  chain.to.mockReturnValue(chain)
  const context = vi.fn((callback: () => void | (() => void)) => {
    const cleanup = callback()
    return {
      revert: () => {
        cleanup?.()
        contextRevert()
      }
    }
  })

  return {
    conditions,
    context,
    contextRevert,
    matchMedia,
    mediaRevert,
    registerPlugin: vi.fn(),
    set: vi.fn(),
    timeline: vi.fn((options?: Record<string, object>) => {
      timelineOptions.push(options ?? {})
      return chain
    }),
    timelineOptions
  }
})

vi.mock('gsap', () => ({
  gsap: motionMocks
}))

vi.mock('gsap/ScrollTrigger', () => ({
  ScrollTrigger: {}
}))

import { useHomeMotion } from '../useHomeMotion'

describe('useHomeMotion', () => {
  beforeEach(() => {
    motionMocks.conditions.desktop = false
    motionMocks.conditions.reducedMotion = true
    vi.clearAllMocks()
    motionMocks.timelineOptions.length = 0
  })

  it('renders the stable state for reduced motion and reverts GSAP on unmount', () => {
    const Harness = defineComponent({
      setup() {
        const root = ref<HTMLElement | null>(null)
        useHomeMotion(root)
        return { root }
      },
      template: '<main ref="root"><p data-motion-reveal>Visible lifecycle</p></main>'
    })

    const wrapper = mount(Harness)

    expect(motionMocks.context).toHaveBeenCalledTimes(1)
    expect(motionMocks.set).toHaveBeenCalledWith(
      '[data-motion-reveal]',
      { autoAlpha: 1, scale: 1, x: 0, y: 0 }
    )
    expect(motionMocks.timeline).not.toHaveBeenCalled()

    wrapper.unmount()

    expect(motionMocks.mediaRevert).toHaveBeenCalledTimes(1)
    expect(motionMocks.contextRevert).toHaveBeenCalledTimes(1)
  })

  it('pins only the animated relay stage so lifecycle copy does not reappear during the handoff', () => {
    motionMocks.conditions.desktop = true
    motionMocks.conditions.reducedMotion = false

    const Harness = defineComponent({
      setup() {
        const root = ref<HTMLElement | null>(null)
        useHomeMotion(root)
        return { root }
      },
      template: `
        <main ref="root">
          <p data-motion-reveal>Visible lifecycle</p>
          <section data-motion-relay>
            <div data-motion-relay-static class="grid"></div>
            <div data-motion-relay-animated class="hidden">
              <article data-motion-relay-panel></article>
              <article data-motion-relay-panel></article>
              <article data-motion-relay-panel></article>
              <span data-motion-relay-meter></span>
            </div>
          </section>
        </main>
      `
    })

    const wrapper = mount(Harness)
    const staticRelay = wrapper.find('[data-motion-relay-static]')
    const animatedRelay = wrapper.find('[data-motion-relay-animated]')

    expect(motionMocks.timeline).toHaveBeenCalledTimes(2)
    expect(staticRelay.classes()).toContain('hidden')
    expect(staticRelay.classes()).not.toContain('grid')
    expect(animatedRelay.classes()).not.toContain('hidden')
    const scrollTrigger = motionMocks.timelineOptions[1]?.scrollTrigger
    if (!scrollTrigger || !('pin' in scrollTrigger) || !('trigger' in scrollTrigger)) {
      throw new Error('Expected the desktop relay timeline to define a ScrollTrigger')
    }
    expect(scrollTrigger.pin).toBe(animatedRelay.element)
    expect(scrollTrigger.trigger).toBe(animatedRelay.element)

    wrapper.unmount()

    expect(staticRelay.classes()).toContain('grid')
    expect(staticRelay.classes()).not.toContain('hidden')
    expect(animatedRelay.classes()).toContain('hidden')
  })

  it('keeps the mobile relay static without creating a ScrollTrigger pin', () => {
    motionMocks.conditions.desktop = false
    motionMocks.conditions.reducedMotion = false

    const Harness = defineComponent({
      setup() {
        const root = ref<HTMLElement | null>(null)
        useHomeMotion(root)
        return { root }
      },
      template: `
        <main ref="root">
          <p data-motion-reveal>Visible lifecycle</p>
          <section data-motion-relay>
            <div data-motion-relay-static class="grid"></div>
            <div data-motion-relay-animated class="hidden">
              <article data-motion-relay-panel></article>
              <article data-motion-relay-panel></article>
              <article data-motion-relay-panel></article>
              <span data-motion-relay-meter></span>
            </div>
          </section>
        </main>
      `
    })

    const wrapper = mount(Harness)
    const staticRelay = wrapper.find('[data-motion-relay-static]')
    const animatedRelay = wrapper.find('[data-motion-relay-animated]')

    expect(motionMocks.timeline).toHaveBeenCalledTimes(1)
    expect(motionMocks.timelineOptions).toEqual([{ defaults: { ease: 'power3.out' } }])
    expect(staticRelay.classes()).toContain('grid')
    expect(staticRelay.classes()).not.toContain('hidden')
    expect(animatedRelay.classes()).toContain('hidden')
  })
})
