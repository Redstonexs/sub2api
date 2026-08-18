import en from '@/i18n/locales/en/landing'
import zh from '@/i18n/locales/zh/landing'
import { describe, expect, it } from 'vitest'

describe('setup locale parity', () => {
  it('keeps setup translation keys mirrored', () => {
    expect(Object.keys(en.setup).sort()).toEqual(Object.keys(zh.setup).sort())
    expect(Object.keys(en.setup.bootstrapToken).sort()).toEqual(Object.keys(zh.setup.bootstrapToken).sort())
  })
})
