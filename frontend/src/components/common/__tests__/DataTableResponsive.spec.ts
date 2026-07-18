import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import DataTable from '../DataTable.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('DataTable tablet layout', () => {
  const mountAtViewport = (width: number) => {
    const matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query === '(min-width: 1536px)' && width >= 1536,
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
          data: [{ id: 1, name: 'Tablet-safe row' }],
        },
      }),
    }
  }

  it('uses readable cards at 1535px before sticky columns can overlap table cells', () => {
    const { matchMedia, wrapper } = mountAtViewport(1535)

    expect(matchMedia).toHaveBeenCalledWith('(min-width: 1536px)')
    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.text()).toContain('Tablet-safe row')
  })

  it('restores the table at 1536px when the application shell has enough room', () => {
    const { wrapper } = mountAtViewport(1536)

    expect(wrapper.find('table').exists()).toBe(true)
  })
})
