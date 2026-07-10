type MessageTree = Record<string, unknown>

function isTree(value: unknown): value is MessageTree {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

// Deep-merges the fork locale overlay onto the upstream locale modules.
// Leaf values in `patch` win; nested objects merge recursively.
export function deepMerge<T extends MessageTree>(target: T, patch: MessageTree): T {
  const out: MessageTree = { ...target }
  for (const [key, value] of Object.entries(patch)) {
    const current = out[key]
    out[key] = isTree(value) && isTree(current) ? deepMerge(current, value) : value
  }
  return out as T
}
