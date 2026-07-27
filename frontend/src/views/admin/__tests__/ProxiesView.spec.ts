import { describe, expect, it } from 'vitest'
import src from '../ProxiesView.vue?raw'

describe('ProxiesView flag rendering', () => {
  it('does not depend on the flag CDN', () => {
    expect(src).not.toMatch(/unpkg\.com/)
  })

  it('renders the country code as a local text badge', () => {
    expect(src).toContain('{{ row.country_code')
  })
})
