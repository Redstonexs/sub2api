<template>
  <form class="space-y-3" @submit.prevent="handleSubmit">
    <input
      v-model="email"
      :data-testid="`${testIdPrefix}-create-account-email`"
      type="email"
      class="input w-full"
      :placeholder="t('auth.emailPlaceholder')"
      :disabled="isSubmitting || isSendingCode"
    />
    <input
      v-model="password"
      :data-testid="`${testIdPrefix}-create-account-password`"
      type="password"
      class="input w-full"
      :placeholder="t('auth.passwordPlaceholder')"
      :disabled="isSubmitting"
    />
    <div v-if="emailVerifyEnabled && captchaProvider !== 'none'" class="space-y-2">
      <CaptchaWidget
        ref="captchaRef"
        :provider="captchaProvider"
        :turnstile-site-key="turnstileSiteKey"
        :cap-a-p-i-endpoint="capAPIEndpoint"
        :cap-site-key="capSiteKey"
        @verify="onCaptchaVerify"
        @expire="onCaptchaExpire"
        @error="onCaptchaError"
      />
    </div>
    <div v-if="emailVerifyEnabled" class="flex gap-3">
      <input
        v-model="verifyCode"
        :data-testid="`${testIdPrefix}-create-account-verify-code`"
        type="text"
        inputmode="numeric"
        maxlength="6"
        class="input min-w-0 flex-1"
        placeholder="123456"
        :disabled="isSubmitting"
      />
      <button
        :data-testid="`${testIdPrefix}-create-account-send-code`"
        type="button"
        class="btn btn-secondary shrink-0"
        :disabled="isSubmitting || isSendingCode || countdown > 0 || !email.trim() || (captchaProvider !== 'none' && !captchaToken)"
        @click="handleSendCode"
      >
        {{
          isSendingCode
            ? t('auth.sendingCode')
            : countdown > 0
              ? t('auth.resendCountdown', { countdown })
              : t('auth.sendCode')
        }}
      </button>
    </div>
    <p v-if="emailVerifyEnabled && sendCodeSuccess" class="text-sm text-green-600 dark:text-green-400">
      {{ t('auth.codeSentSuccess') }}
    </p>
    <p v-else-if="emailVerifyEnabled" class="text-xs text-gray-500 dark:text-dark-400">
      {{ t('auth.verificationCodeHint') }}
    </p>
    <input
      v-if="invitationCodeEnabled"
      v-model="invitationCode"
      :data-testid="`${testIdPrefix}-create-account-invitation-code`"
      type="text"
      class="input w-full"
      :placeholder="t('auth.invitationCodePlaceholder')"
      :disabled="isSubmitting"
    />
    <button
      :data-testid="`${testIdPrefix}-create-account-submit`"
      type="button"
      class="btn btn-primary w-full"
      :disabled="isSubmitting || !email.trim() || password.length < 6 || (invitationCodeEnabled && !invitationCode.trim())"
      @click="handleSubmit"
    >
      {{ isSubmitting ? t('common.processing') : t('auth.createAccount') }}
    </button>
    <button
      type="button"
      class="btn btn-secondary w-full"
      :disabled="isSubmitting"
      @click="emitSwitchToBind"
    >
      {{ t('auth.alreadyHaveAccount') }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CaptchaWidget from '@/components/CaptchaWidget.vue'
import { getPublicSettings, sendPendingOAuthVerifyCode } from '@/api/auth'
import { useAppStore } from '@/stores'
import type { CaptchaProvider } from '@/types'
import { captchaPayload, resolveCaptchaProvider } from '@/utils/captcha'

export type PendingOAuthCreateAccountPayload = {
  email: string
  password: string
  verifyCode: string
  invitationCode?: string
}

const props = defineProps<{
  initialEmail: string
  testIdPrefix: string
  isSubmitting: boolean
  errorMessage?: string
}>()

const emit = defineEmits<{
  submit: [payload: PendingOAuthCreateAccountPayload]
  switchToBind: [email: string]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const email = ref('')
const password = ref('')
const verifyCode = ref('')
const invitationCode = ref('')
const isSendingCode = ref(false)
const sendCodeError = ref('')
const sendCodeSuccess = ref(false)
const countdown = ref(0)
const invitationCodeEnabled = ref(false)
const emailVerifyEnabled = ref(true)
const captchaProvider = ref<CaptchaProvider>('none')
const turnstileSiteKey = ref('')
const capAPIEndpoint = ref('')
const capSiteKey = ref('')
const captchaToken = ref('')
const captchaRef = ref<InstanceType<typeof CaptchaWidget> | null>(null)

let countdownTimer: ReturnType<typeof setInterval> | null = null

watch(
  () => props.initialEmail,
  value => {
    email.value = value || ''
  },
  { immediate: true }
)

watch(sendCodeError, value => {
  if (value) {
    appStore.showError(value)
  }
})

watch(
  () => props.errorMessage,
  value => {
    if (value) {
      appStore.showError(value)
    }
  }
)

function clearCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function startCountdown(seconds: number) {
  clearCountdown()
  countdown.value = Math.max(0, seconds)

  if (countdown.value <= 0) {
    return
  }

  countdownTimer = setInterval(() => {
    if (countdown.value <= 1) {
      countdown.value = 0
      clearCountdown()
      return
    }

    countdown.value -= 1
  }, 1000)
}

function getRequestErrorMessage(error: unknown, fallback: string): string {
  const err = error as { message?: string; response?: { data?: { detail?: string; message?: string } } }
  return err.response?.data?.detail || err.response?.data?.message || err.message || fallback
}

function resetCaptcha() {
  captchaToken.value = ''
  captchaRef.value?.reset()
}

function onCaptchaVerify(token: string) {
  captchaToken.value = token
  sendCodeError.value = ''
}

function onCaptchaExpire() {
  captchaToken.value = ''
  sendCodeError.value = t('auth.turnstileExpired')
}

function onCaptchaError() {
  captchaToken.value = ''
  sendCodeError.value = t('auth.turnstileFailed')
}

async function handleSendCode() {
  const trimmedEmail = email.value.trim()
  if (!trimmedEmail) {
    return
  }

  if (captchaProvider.value !== 'none' && !captchaToken.value) {
    sendCodeError.value = t('auth.completeVerification')
    return
  }

  isSendingCode.value = true
  sendCodeError.value = ''
  sendCodeSuccess.value = false

  try {
    const response = await sendPendingOAuthVerifyCode({
      email: trimmedEmail,
      ...captchaPayload(captchaProvider.value, captchaToken.value)
    })
    sendCodeSuccess.value = true
    startCountdown(response.countdown)
    if (captchaProvider.value !== 'none') {
      resetCaptcha()
    }
  } catch (error: unknown) {
    sendCodeError.value = getRequestErrorMessage(error, t('auth.sendCodeFailed'))
  } finally {
    isSendingCode.value = false
  }
}

function handleSubmit() {
  const trimmedEmail = email.value.trim()
  if (!trimmedEmail || password.value.length < 6) {
    return
  }

  emit('submit', {
    email: trimmedEmail,
    password: password.value,
    verifyCode: emailVerifyEnabled.value ? verifyCode.value.trim() : '',
    invitationCode: invitationCode.value.trim() || undefined
  })
}

function emitSwitchToBind() {
  emit('switchToBind', email.value.trim())
}

onMounted(async () => {
  try {
    const settings = await getPublicSettings()
    invitationCodeEnabled.value = settings.invitation_code_enabled === true
    emailVerifyEnabled.value = settings.email_verify_enabled !== false
    captchaProvider.value = resolveCaptchaProvider(settings)
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    capAPIEndpoint.value = settings.cap_api_endpoint || ''
    capSiteKey.value = settings.cap_site_key || ''
  } catch {
    invitationCodeEnabled.value = false
    emailVerifyEnabled.value = true
    captchaProvider.value = 'none'
    turnstileSiteKey.value = ''
    capAPIEndpoint.value = ''
    capSiteKey.value = ''
  }
})

onUnmounted(() => {
  clearCountdown()
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
