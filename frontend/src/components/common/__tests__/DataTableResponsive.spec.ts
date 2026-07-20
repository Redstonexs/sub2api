import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import DataTable from '../DataTable.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('DataTable responsive layout', () => {
  const mountAtViewport = (width: number) => {
    const matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query === '(min-width: 768px)' && width >= 768,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
    Object.defineProperty(window, 'matchMedia', { configurable: true, value: matchMedia })

    return {
      matchMedia,
      wrapper: mount(DataTable, {
        props: {
          columns: [
            { key: 'name', label: 'Name' },
            { key: 'actions', label: 'Actions' },
          ],
          data: [{ id: 1, name: 'Responsive row' }],
        },
      }),
    }
  }

  it('uses stacked cards below 768px where table cells cannot fit', () => {
    const { matchMedia, wrapper } = mountAtViewport(767)

    expect(matchMedia).toHaveBeenCalledWith('(min-width: 768px)')
    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.text()).toContain('Responsive row')
  })

  it('restores the horizontal table at 768px and above', () => {
    const { wrapper } = mountAtViewport(768)

    expect(wrapper.find('table').exists()).toBe(true)
  })

  it('keeps the horizontal table on constrained desktop widths like 1280px', () => {
    const { wrapper } = mountAtViewport(1280)

    expect(wrapper.find('table').exists()).toBe(true)
  })
})
