import { describe, expect, it } from 'vitest'
import source from '../VersionBadge.vue?raw'

describe('VersionBadge rollback command surface', () => {
  it('does not expose a copyable curl one-liner in the component source', () => {
    expect(source).not.toContain('curl -sSL')
  })

  it('links to the GitHub release tag for manual rollback instructions', () => {
    expect(source).toContain('releases/tag/')
  })
})
