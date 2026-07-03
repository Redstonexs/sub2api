/**
 * Shared chart theme — single source of truth for chart colors.
 *
 * The categorical series palette is warm-tempered to match the app palette and
 * validated (lightness band, chroma floor, adjacent-pair CVD separation,
 * surface contrast) on both surfaces: light #FFFFFF and dark #2A2722.
 * Worst adjacent CVD pair ΔE 22.9 (deutan). If you tune any hex, re-validate
 * both modes before shipping.
 *
 * Rules baked in here:
 * - Assign series colors in fixed order (never cycle hues); overflow → CHART_OTHER.
 * - Chart text/grid always come from chartChrome(), not from series colors.
 */

/** Fixed-order categorical palette (max 8 distinct series). */
export const CHART_SERIES = [
  '#CC785C', // terracotta (brand)
  '#4A7EC9', // denim
  '#C08A21', // ochre
  '#0D9488', // teal
  '#9A5BA8', // plum
  '#67923D', // olive
  '#C25E7E', // rose
  '#7A6BD1' // iris
] as const

/** Bucket color for "Other" / overflow series. */
export const CHART_OTHER = '#ABA493'

/**
 * Token-usage role colors. Hue semantics preserved from the legacy charts
 * (input was blue, output green, cache creation amber, cache read cyan,
 * cache hit rate purple) so long-time users keep their bearings.
 */
export const CHART_ROLES = {
  input: '#4A7EC9',
  output: '#67923D',
  cacheCreation: '#C08A21',
  cacheRead: '#0D9488',
  cacheHitRate: '#9A5BA8'
} as const

/** Fixed-order series color with "Other" overflow. */
export function seriesColor(index: number): string {
  return index < CHART_SERIES.length ? CHART_SERIES[index] : CHART_OTHER
}

/** `#RRGGBB` + alpha (0–1) → `#RRGGBBAA`. */
export function withAlpha(hex: string, alpha: number): string {
  const a = Math.round(Math.min(1, Math.max(0, alpha)) * 255)
    .toString(16)
    .padStart(2, '0')
  return `${hex}${a}`
}

export interface ChartChrome {
  /** Axis tick / legend label ink. */
  text: string
  /** Secondary ink (axis titles, captions). */
  muted: string
  /** Gridline color (recessive). */
  grid: string
  /** Tooltip / chart border color. */
  border: string
}

/** Warm-stone chart chrome for the current theme. */
export function chartChrome(isDark: boolean): ChartChrome {
  return isDark
    ? { text: '#D5CFC0', muted: '#ABA493', grid: '#39352E', border: '#45413A' }
    : { text: '#45413A', muted: '#827B6C', grid: '#E7E3D8', border: '#D5CFC0' }
}

/** Chart.js tooltip colors for the current theme. */
export function chartTooltipStyle(isDark: boolean) {
  return isDark
    ? { backgroundColor: '#2A2722', titleColor: '#F3F1EA', bodyColor: '#D5CFC0' }
    : { backgroundColor: '#FFFFFF', titleColor: '#1A1815', bodyColor: '#605A4E' }
}
