<template>
  <div v-if="widgetEndpoint && widgetReady" class="cap-widget-wrapper">
    <cap-widget
      ref="widgetRef"
      :key="widgetEndpoint"
      class="cap-widget"
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
}
</style>
