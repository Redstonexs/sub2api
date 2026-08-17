import { describe, expect, it } from 'vitest'

import { getUsageQoSPresentation } from '../usageQoS'

describe('getUsageQoSPresentation', () => {
  it('keeps missing and legacy snapshots unknown', () => {
    expect(getUsageQoSPresentation({
      group_qos_tier: null,
      group_qos_window: null,
      group_qos_affected: null,
      group_qos_effects: null,
    })).toMatchObject({ state: 'unknown', tier: null, window: null, effects: [] })
  })

  it('normalizes omitted QoS fields from mixed-version responses', () => {
    expect(getUsageQoSPresentation({})).toEqual({
      state: 'unknown',
      tier: null,
      window: null,
      effects: [],
    })
  })

  it('distinguishes an active tier that did not alter the request', () => {
    expect(getUsageQoSPresentation({
      group_qos_tier: 2,
      group_qos_window: 'weekly',
      group_qos_affected: false,
      group_qos_effects: [],
    })).toEqual({ state: 'active', tier: 2, window: 'weekly', effects: [] })
  })

  it('presents affected effects in a stable display order', () => {
    expect(getUsageQoSPresentation({
      group_qos_tier: 1,
      group_qos_window: 'daily',
      group_qos_affected: true,
      group_qos_effects: ['rpm', 'model'],
    }).effects).toEqual(['model', 'rpm'])
  })
})
