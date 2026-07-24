<template>
  <TurnstileWidget
    v-if="provider === 'turnstile' && turnstileSiteKey"
    ref="turnstileRef"
    :site-key="turnstileSiteKey"
    @verify="emit('verify', $event)"
    @expire="emit('expire')"
    @error="emit('error')"
  />
  <CapCaptchaWidget
    v-else-if="provider === 'cap' && capAPIEndpoint && capSiteKey"
    ref="capRef"
    :api-endpoint="capAPIEndpoint"
    :site-key="capSiteKey"
    @verify="emit('verify', $event)"
    @expire="emit('expire')"
    @error="emit('error')"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CaptchaProvider } from '@/types'
import CapCaptchaWidget from './CapCaptchaWidget.vue'
import TurnstileWidget from './TurnstileWidget.vue'

defineProps<{
  provider: CaptchaProvider
  turnstileSiteKey?: string
  capAPIEndpoint?: string
  capSiteKey?: string
}>()

const emit = defineEmits<{
  (event: 'verify', token: string): void
  (event: 'expire'): void
  (event: 'error'): void
}>()

const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const capRef = ref<InstanceType<typeof CapCaptchaWidget> | null>(null)

function reset(): void {
  turnstileRef.value?.reset()
  capRef.value?.reset()
}

defineExpose({ reset })
</script>
