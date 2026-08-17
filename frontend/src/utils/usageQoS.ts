import type { UsageLog, UsageQoSEffect, UsageQoSWindow } from '@/types'

export type UsageQoSState = 'unknown' | 'active' | 'affected'

export interface UsageQoSPresentation {
  state: UsageQoSState
  tier: number | null
  window: UsageQoSWindow | null
  effects: UsageQoSEffect[]
}

const EFFECT_ORDER: UsageQoSEffect[] = ['model', 'reasoning', 'rpm']

const isValidTier = (value: number | null | undefined): value is number =>
  typeof value === 'number' && Number.isInteger(value) && value >= 1

export const getUsageQoSPresentation = (row: Partial<Pick<UsageLog, 'group_qos_tier' | 'group_qos_window' | 'group_qos_affected' | 'group_qos_effects'>>): UsageQoSPresentation => {
  const group_qos_tier = row.group_qos_tier ?? null
  const group_qos_window = row.group_qos_window ?? null
  const group_qos_affected = row.group_qos_affected ?? null
  const group_qos_effects = row.group_qos_effects ?? null
  const hasSnapshot = group_qos_affected !== null || group_qos_tier !== null || group_qos_window !== null
  const effects = EFFECT_ORDER.filter((effect) => group_qos_effects?.includes(effect))

  return {
    state: group_qos_affected === true
      ? 'affected'
      : group_qos_affected === false && hasSnapshot
        ? 'active'
        : 'unknown',
    tier: isValidTier(group_qos_tier) ? group_qos_tier : null,
    window: group_qos_window,
    effects,
  }
}
