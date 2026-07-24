<template>
  <div v-if="widgetEndpoint && widgetReady" class="cap-widget-wrapper">
    <cap-widget
      ref="widgetRef"
      :key="widgetEndpoint"
      class="cap-widget cap-widget--form-control"
      :style="widgetStyle"
      :data-cap-api-endpoint="widgetEndpoint"
      @solve="onSolve"
      @reset="onReset"
      @error="onError"
    />
  </div>
</template>

<script setup lang="ts">
import type { CapWidget as CapWidgetElement } from 'cap-widget'
import { computed, onMounted, ref } from 'vue'

const props = defineProps<{
  apiEndpoint: string
  siteKey: string
}>()

const emit = defineEmits<{
  (event: 'verify', token: string): void
  (event: 'expire'): void
  (event: 'error'): void
}>()

const widgetRef = ref<CapWidgetElement | null>(null)
const widgetReady = ref(false)
const widgetStyle = {
  '--cap-widget-width': '100%',
  '--cap-widget-height': '64px'
} as const

const widgetEndpoint = computed(() => {
  const apiEndpoint = props.apiEndpoint.trim().replace(/\/+$/, '')
  const siteKey = props.siteKey.trim()
  if (!apiEndpoint || !siteKey) {
    return ''
  }
  return `${apiEndpoint}/${encodeURIComponent(siteKey)}/`
})

function isSolveDetail(value: unknown): value is { token: string } {
  if (typeof value !== 'object' || value === null || !('token' in value)) {
    return false
  }
  return typeof value.token === 'string' && value.token.trim() !== ''
}

function onSolve(event: Event): void {
  const detail = (event as CustomEvent<unknown>).detail
  if (!isSolveDetail(detail)) {
    emit('error')
    return
  }
  emit('verify', detail.token)
}

function onReset(): void {
  emit('expire')
}

function onError(): void {
  emit('error')
}

onMounted(async () => {
  try {
    await import('cap-widget')
    widgetReady.value = true
  } catch {
    emit('error')
  }
})

function reset(): void {
  widgetRef.value?.reset()
}

defineExpose({ reset })
</script>

<style scoped>
.cap-widget-wrapper,
.cap-widget {
  display: block;
  width: 100%;
  min-width: 0;
}

.cap-widget-wrapper {
  min-height: 64px;
}

.cap-widget--form-control {
  max-width: 100%;
  --cap-border-radius: 10px;
  --cap-background: #ffffff;
  --cap-border-color: #e7e3d8;
  --cap-color: #1a1815;
  --cap-checkbox-background: #f8f6f0;
  --cap-checkbox-border: 1px solid #b9b3a7;
  --cap-spinner-color: #b05c40;
  --cap-spinner-background-color: #f3f1ea;
  --cap-focus-ring: #b05c40;
  --cap-troubleshoot-color: #8f4a34;
}

:global(.dark .cap-widget--form-control) {
  --cap-background: #2a2722;
  --cap-border-color: #39352e;
  --cap-color: #f8f6f0;
  --cap-checkbox-background: #211f1a;
  --cap-checkbox-border: 1px solid #605a4e;
  --cap-spinner-color: #b05c40;
  --cap-spinner-background-color: #39352e;
  --cap-focus-ring: #b05c40;
  --cap-troubleshoot-color: #dad5c9;
}
</style>
