import { describe, expect, it } from 'vitest'
import { resolveCaptchaProvider } from '@/utils/captcha'

describe('captcha utilities', () => {
  it('selects CAP only when its public configuration is complete', () => {
    expect(resolveCaptchaProvider({
      captcha_provider: 'cap',
      cap_api_endpoint: 'https://cap.example.com',
      cap_site_key: 'public-key'
    })).toBe('cap')

    expect(resolveCaptchaProvider({
      captcha_provider: 'cap',
      cap_api_endpoint: 'https://cap.example.com'
    })).toBe('none')
  })

  it('keeps legacy Turnstile configuration working when no provider is set', () => {
    expect(resolveCaptchaProvider({
      turnstile_enabled: true,
      turnstile_site_key: 'turnstile-key'
    })).toBe('turnstile')
  })
})
