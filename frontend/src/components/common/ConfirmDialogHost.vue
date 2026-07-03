<template>
  <ConfirmDialog
    :show="state.show"
    :title="state.title || t('common.confirm')"
    :message="state.message"
    :confirm-text="state.confirmText"
    :cancel-text="state.cancelText"
    :danger="state.danger"
    :z-index="60"
    @confirm="resolveConfirm(true)"
    @cancel="resolveConfirm(false)"
  />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ConfirmDialog from './ConfirmDialog.vue'
import { useConfirmHost } from '@/composables/useConfirm'

const { t } = useI18n()
const { state, resolveConfirm } = useConfirmHost()
</script>
