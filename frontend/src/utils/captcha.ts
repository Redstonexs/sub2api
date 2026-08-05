import type { CaptchaProvider } from '@/types'

export interface CaptchaPublicSettings {
  captcha_provider?: CaptchaProvider
  cap_api_endpoint?: string
  cap_site_key?: string
  turnstile_enabled?: boolean
  turnstile_site_key?: string
}

export function resolveCaptchaProvider(
  settings: CaptchaPublicSettings | null | undefined
): CaptchaProvider {
  if (settings?.captcha_provider === 'cap') {
    return hasCapConfiguration(settings) ? 'cap' : 'none'
  }
  if (settings?.captcha_provider === 'turnstile') {
    return hasTurnstileConfiguration(settings) ? 'turnstile' : 'none'
  }
  if (settings?.captcha_provider === 'none') {
    return 'none'
  }
  return hasTurnstileConfiguration(settings) ? 'turnstile' : 'none'
}

function hasCapConfiguration(settings: CaptchaPublicSettings): boolean {
  return Boolean(
    settings.cap_api_endpoint?.trim() && settings.cap_site_key?.trim()
  )
}

function hasTurnstileConfiguration(settings: CaptchaPublicSettings | null | undefined): boolean {
  return Boolean(settings?.turnstile_enabled && settings.turnstile_site_key?.trim())
}
