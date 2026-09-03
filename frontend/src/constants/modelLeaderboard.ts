/**
 * Snapshot of the Artificial Analysis Intelligence Index rendered on the public
 * homepage (`components/home/HomeModelLeaderboard.vue`).
 *
 * Curated by hand from https://artificialanalysis.ai/leaderboards/models. The
 * published board lists every reasoning-effort configuration as its own row —
 * Claude Fable 5.1 alone occupies five of the top twenty — so this snapshot
 * keeps only the highest-scoring configuration per model family and preserves
 * the source ordering. The homepage states that reduction in its footnote.
 *
 * This is a static snapshot, never live data: refresh `LEADERBOARD_AS_OF`
 * together with the rows, because the section renders that date next to the
 * source link.
 */

/** Live leaderboard the snapshot is taken from. */
export const LEADERBOARD_SOURCE_URL = 'https://artificialanalysis.ai/leaderboards/models'

/** ISO date the rows below were last verified against the source. */
export const LEADERBOARD_AS_OF = '2026-09-03'

/** Evaluations aggregated into the Intelligence Index, in the source's order. */
export const LEADERBOARD_EVALUATIONS = [
  'GDPval-AA v2',
  '\u{1D70F}³-Banking',
  'Terminal-Bench v2.1',
  'SciCode',
  "Humanity's Last Exam",
  'GPQA Diamond',
  'CritPt',
  'AA-Omniscience',
  'AA-LCR',
] as const

/** Palette bucket for a row; drives the static Tailwind classes in the section. */
export type LeaderboardAccent = 'claude' | 'gpt' | 'gemini' | 'grok' | 'neutral'

export interface LeaderboardEntry {
  /** Position within this de-duplicated snapshot, not the raw source rank. */
  rank: number
  model: string
  creator: string
  /** Intelligence Index, normalised 0-100 by the source. */
  score: number
  accent: LeaderboardAccent
}

export const MODEL_LEADERBOARD: readonly LeaderboardEntry[] = [
  { rank: 1, model: 'Claude Fable 5.1', creator: 'Anthropic', score: 66, accent: 'claude' },
  { rank: 2, model: 'Claude Opus 5', creator: 'Anthropic', score: 63, accent: 'claude' },
  { rank: 3, model: 'Muse Spark 1.3', creator: 'Meta', score: 62, accent: 'neutral' },
  { rank: 4, model: 'Claude Fable 5', creator: 'Anthropic', score: 62, accent: 'claude' },
  { rank: 5, model: 'GPT-5.6 Sol', creator: 'OpenAI', score: 61, accent: 'gpt' },
  { rank: 6, model: 'Grok 4.6', creator: 'SpaceXAI', score: 61, accent: 'grok' },
  { rank: 7, model: 'Kimi K3', creator: 'Moonshot AI', score: 60, accent: 'neutral' },
  { rank: 8, model: 'GLM-5.3', creator: 'Z AI', score: 60, accent: 'neutral' },
  { rank: 9, model: 'Gemini 3.8 Flash', creator: 'Google', score: 59, accent: 'gemini' },
  { rank: 10, model: 'Qwen3.8 Max', creator: 'Alibaba', score: 58, accent: 'neutral' },
]
