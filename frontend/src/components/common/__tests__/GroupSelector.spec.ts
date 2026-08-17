import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import GroupSelector from '../GroupSelector.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

describe('GroupSelector responsive options', () => {
  it('keeps each group selectable and gives its name to assistive technology', () => {
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue: [],
        groups: [
          {
            id: 1,
            name: 'A group with a long mobile-friendly name',
            platform: 'openai',
            subscription_type: 'standard',
            rate_multiplier: 1,
          },
        ],
      },
      global: {
        stubs: {
          GroupBadge: true,
          Icon: true,
        },
      },
    })

    expect(wrapper.find('.grid').classes()).toEqual(
      expect.arrayContaining(['grid-cols-1', 'sm:grid-cols-2']),
    )
    expect(wrapper.find('input[type="checkbox"]').attributes('aria-label')).toBe(
      'A group with a long mobile-friendly name',
    )
    expect(wrapper.findAll('label')[1].classes()).toContain('min-w-0')
  })
})
