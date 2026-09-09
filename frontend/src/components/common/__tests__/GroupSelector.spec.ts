import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GroupSelector from '../GroupSelector.vue'

const authState = { isSimpleMode: false }

vi.mock('@/stores', () => ({ useAuthStore: () => authState }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

describe('GroupSelector responsive options', () => {
  it('keeps each group selectable and gives its name to assistive technology', () => {
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue: [],
        groups: [{ id: 1, name: 'A group with a long mobile-friendly name', platform: 'openai', subscription_type: 'standard', rate_multiplier: 1 }],
      },
      global: { stubs: { GroupBadge: true, Icon: true } },
    })
    expect(wrapper.find('.grid').classes()).toEqual(expect.arrayContaining(['grid-cols-1', 'sm:grid-cols-2']))
    expect(wrapper.find('input[type="checkbox"]').attributes('aria-label')).toBe('A group with a long mobile-friendly name')
    expect(wrapper.findAll('label')[1].classes()).toContain('min-w-0')
  })
})

describe('GroupSelector simple-mode binding policy', () => {
  const groups = [
    { id: 1, name: 'Basic', platform: 'anthropic', status: 'active' },
    { id: 2, name: 'Composite', platform: 'composite', status: 'active' },
  ] as never[]
  const mountSelector = (modelValue: number[] = []) => mount(GroupSelector, {
    props: { modelValue, groups },
    global: { stubs: { GroupBadge: { props: ['name'], template: '<span>{{ name }}</span>' }, Icon: true } },
  })
  beforeEach(() => { authState.isSimpleMode = false })

  it('hides composite groups in simple mode and preserves basic groups', () => {
    authState.isSimpleMode = true
    const wrapper = mountSelector()
    expect(wrapper.text()).toContain('Basic')
    expect(wrapper.text()).not.toContain('Composite')
  })
  it('keeps composite groups available in advanced mode', () => {
    expect(mountSelector().text()).toContain('Composite')
  })
  it('cleans hidden historical composite IDs while preserving visible selections', () => {
    authState.isSimpleMode = true
    expect(mountSelector([1, 2]).emitted('update:modelValue')).toEqual([[[1]]])
  })
})
