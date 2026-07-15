import { describe, expect, it } from 'vitest'

import { sanitizeSvg } from '../sanitize'

describe('sanitizeSvg', () => {
  it('removes executable SVG event attributes', () => {
    const sanitized = sanitizeSvg('<svg><circle onload="alert(1)" /></svg>')

    expect(sanitized).not.toContain('onload')
    expect(sanitized).not.toContain('alert(1)')
  })
})
