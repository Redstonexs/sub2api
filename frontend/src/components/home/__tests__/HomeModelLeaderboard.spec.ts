import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import HomeModelLeaderboard from '../HomeModelLeaderboard.vue'
import {
  LEADERBOARD_AS_OF,
  LEADERBOARD_EVALUATIONS,
  LEADERBOARD_SOURCE_URL,
  MODEL_LEADERBOARD,
} from '@/constants/modelLeaderboard'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${Object.values(params).join(',')}` : key,
    }),
  }
})

describe('HomeModelLeaderboard', () => {
  it('renders one row per snapshot entry, in snapshot order', () => {
    const wrapper = mount(HomeModelLeaderboard)
    const rows = wrapper.findAll('[data-testid="home-leaderboard-row"]')

    expect(rows).toHaveLength(MODEL_LEADERBOARD.length)
    MODEL_LEADERBOARD.forEach((entry, index) => {
      expect(rows[index].text()).toContain(entry.model)
      expect(rows[index].text()).toContain(entry.creator)
      expect(rows[index].text()).toContain(String(entry.score))
    })
  })

  it('leads with Claude and surfaces the top OpenAI model', () => {
    const wrapper = mount(HomeModelLeaderboard)
    const stats = wrapper.findAll('dd')

    expect(MODEL_LEADERBOARD[0].creator).toBe('Anthropic')
    expect(stats[0].text()).toContain(MODEL_LEADERBOARD[0].model)

    const topOpenAi = MODEL_LEADERBOARD.find(entry => entry.accent === 'gpt')
    expect(topOpenAi).toBeDefined()
    expect(stats[2].text()).toContain(topOpenAi!.model)
    expect(wrapper.text()).toContain(String(LEADERBOARD_EVALUATIONS.length))
  })

  it('scales each bar to the 0-100 index rather than to the leader', () => {
    const wrapper = mount(HomeModelLeaderboard)
    const bars = wrapper.findAll('[data-testid="home-leaderboard-row"] span[style]')

    MODEL_LEADERBOARD.forEach((entry, index) => {
      expect(bars[index].attributes('style')).toContain(`width: ${entry.score}%`)
    })
  })

  it('attributes the snapshot to the live Artificial Analysis leaderboard', () => {
    const wrapper = mount(HomeModelLeaderboard)
    const source = wrapper.get('[data-testid="home-leaderboard-source"]')

    expect(source.attributes('href')).toBe(LEADERBOARD_SOURCE_URL)
    expect(source.attributes('rel')).toBe('noopener noreferrer')
    expect(source.attributes('target')).toBe('_blank')
    expect(wrapper.text()).toContain(`homeV2.leaderboardSnapshot:${LEADERBOARD_AS_OF}`)
  })
})
