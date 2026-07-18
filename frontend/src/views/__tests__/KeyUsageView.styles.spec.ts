import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const currentDir = dirname(fileURLToPath(import.meta.url))
const keyUsageViewSource = readFileSync(resolve(currentDir, '../KeyUsageView.vue'), 'utf8')

describe('KeyUsageView skeleton theme', () => {
  it('keeps the dark shimmer scoped to skeleton elements instead of the document dark root', () => {
    expect(keyUsageViewSource).toContain('.dark .skeleton {')
    expect(keyUsageViewSource).not.toContain(':global(.dark) .skeleton {')
  })
})
