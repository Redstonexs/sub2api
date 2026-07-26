import { onBeforeUnmount, onMounted, type Ref } from 'vue'
import { gsap } from 'gsap'
import { ScrollTrigger } from 'gsap/ScrollTrigger'

gsap.registerPlugin(ScrollTrigger)

export function useHomeMotion(root: Ref<HTMLElement | null>): void {
  let context: gsap.Context | undefined

  onMounted(() => {
    const element = root.value
    if (!element) {
      return
    }

    context = gsap.context(() => {
      const media = gsap.matchMedia()
      media.add(
        {
          desktop: '(min-width: 1024px)',
          reducedMotion: '(prefers-reduced-motion: reduce)'
        },
        motionContext => {
          if (motionContext.conditions?.reducedMotion) {
            gsap.set('[data-motion-reveal]', { autoAlpha: 1, scale: 1, x: 0, y: 0 })
            return
          }

          const intro = gsap.timeline({ defaults: { ease: 'power3.out' } })
          intro
            .from('[data-motion-reveal]', { autoAlpha: 0, duration: 0.7, stagger: 0.09, y: 24 })
            .from('[data-motion-key]', { autoAlpha: 0, duration: 0.45, scale: 0.92 }, 0.22)
            .from('[data-motion-gateway]', { autoAlpha: 0, duration: 0.5, scale: 0.8 }, 0.34)
            .from('[data-motion-provider]', { autoAlpha: 0, duration: 0.35, stagger: 0.1, x: 22 }, 0.5)
            .to('[data-motion-signal]', { autoAlpha: 1, duration: 0.2 }, 0.64)
            .to('[data-motion-signal]', { duration: 0.55, x: 150 }, 0.72)
            .to('[data-motion-signal]', { duration: 0.5, x: 280, y: -72 }, 1.27)

          if (!motionContext.conditions?.desktop) {
            return
          }

          const staticRelay = element.querySelector<HTMLElement>('[data-motion-relay-static]')
          const animatedRelay = element.querySelector<HTMLElement>('[data-motion-relay-animated]')
          const firstPanel = element.querySelector<HTMLElement>('[data-motion-relay-panel]:nth-child(1)')
          const secondPanel = element.querySelector<HTMLElement>('[data-motion-relay-panel]:nth-child(2)')
          const thirdPanel = element.querySelector<HTMLElement>('[data-motion-relay-panel]:nth-child(3)')
          if (!staticRelay || !animatedRelay || !firstPanel || !secondPanel || !thirdPanel) {
            return
          }

          staticRelay.classList.remove('grid')
          staticRelay.classList.add('hidden')
          animatedRelay.classList.remove('hidden')
          gsap.set('[data-motion-relay-panel]', { autoAlpha: 0, y: 0 })
          gsap.set(firstPanel, { autoAlpha: 1 })
          gsap.set('[data-motion-relay-meter]', { scaleX: 0 })
          gsap.timeline({
            scrollTrigger: {
              anticipatePin: 1,
              end: '+=1100',
              invalidateOnRefresh: true,
              pin: animatedRelay,
              scrub: 1,
              start: 'top top+=72',
              trigger: animatedRelay
            }
          })
            .to('[data-motion-relay-meter]', { duration: 0.3, ease: 'none', scaleX: 1, stagger: 0.12 }, 0.06)
            .set(firstPanel, { autoAlpha: 0 }, 0.34)
            .set(secondPanel, { autoAlpha: 1 }, 0.34)
            .set(secondPanel, { autoAlpha: 0 }, 0.68)
            .set(thirdPanel, { autoAlpha: 1 }, 0.68)

          return () => {
            animatedRelay.classList.add('hidden')
            staticRelay.classList.remove('hidden')
            staticRelay.classList.add('grid')
          }
        }
      )

      return () => media.revert()
    }, element)
  })

  onBeforeUnmount(() => {
    context?.revert()
  })
}
