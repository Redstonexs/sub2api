<template>
  <div
    v-if="platform === 'anthropic' || platform === 'openai'"
    class="space-y-4"
  >
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h4 class="text-sm font-medium text-gray-900 dark:text-white">
        {{ t('dashboard.groupQuotaCard.grantManagement') }}
      </h4>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="space-y-2">
      <div
        v-for="i in 2"
        :key="i"
        class="h-10 animate-pulse rounded-lg bg-gray-200 dark:bg-dark-700"
      />
    </div>

    <!-- Error state -->
    <div
      v-else-if="loadError"
      class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-400"
    >
      {{ loadError }}
    </div>

    <!-- Granted users list -->
    <div v-else class="space-y-3">
      <!-- Empty state -->
      <p
        v-if="grants.length === 0"
        class="text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('dashboard.groupQuotaCard.noGrantedUsers') }}
      </p>

      <!-- Grant rows -->
      <ul v-else class="space-y-2">
        <li
          v-for="grant in grants"
          :key="grant.user_id"
          class="flex items-center justify-between rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="min-w-0 flex-1">
            <span class="text-sm font-medium text-gray-900 dark:text-white">
              {{ grant.username }}
            </span>
            <div class="mt-0.5 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
              <span>{{
                t('dashboard.groupQuotaCard.grantedBy', {
                  name: grant.granted_by,
                })
              }}</span>
              <span>{{
                t('dashboard.groupQuotaCard.grantedAt', {
                  time: formatDateTime(grant.granted_at),
                })
              }}</span>
            </div>
          </div>
          <button
            :disabled="mutating"
            @click="handleRevoke(grant.user_id, grant.username)"
            class="ml-3 flex-shrink-0 rounded-lg px-2.5 py-1.5 text-xs font-medium text-red-600 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-900/20"
          >
            {{ t('dashboard.groupQuotaCard.revoke') }}
          </button>
        </li>
      </ul>

      <!-- Add user form -->
      <div class="flex items-center gap-2 pt-2">
        <input
          v-model="newUserId"
          type="number"
          :placeholder="t('dashboard.groupQuotaCard.addUserById')"
          :disabled="mutating"
          @keyup.enter="handleAdd"
          class="input w-40"
          min="1"
          step="1"
        />
        <button
          :disabled="mutating || !isValidUserId"
          @click="handleAdd"
          class="btn btn-secondary disabled:cursor-not-allowed disabled:opacity-50"
        >
          {{ t('dashboard.groupQuotaCard.add') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useConfirm } from '@/composables/useConfirm'
import { getGroupViewGrants, addGroupViewGrant, removeGroupViewGrant } from '@/api/admin/dashboard'
import { formatDateTime } from '@/utils/format'
import type { GroupViewGrantItem } from '@/types'

const props = defineProps<{
  groupId: number
  platform: string
}>()

const emit = defineEmits<{
  change: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { confirm } = useConfirm()

const grants = ref<GroupViewGrantItem[]>([])
const loading = ref(false)
const loadError = ref('')
const mutating = ref(false)
const newUserId = ref('')

const isValidUserId = computed(() => {
  const n = Number(newUserId.value)
  return Number.isInteger(n) && n > 0
})

async function loadGrants() {
  loading.value = true
  loadError.value = ''
  try {
    grants.value = await getGroupViewGrants(props.groupId)
  } catch (err) {
    loadError.value = (err as { message?: string }).message ?? t('common.unknownError')
  } finally {
    loading.value = false
  }
}

async function handleAdd() {
  if (!isValidUserId.value) return
  const userId = Number(newUserId.value)
  mutating.value = true
  try {
    await addGroupViewGrant(props.groupId, userId)
    appStore.showSuccess(t('dashboard.groupQuotaCard.grantSuccess'))
    newUserId.value = ''
    await loadGrants()
    emit('change')
  } catch {
    appStore.showError(t('dashboard.groupQuotaCard.grantFailed'))
  } finally {
    mutating.value = false
  }
}

async function handleRevoke(userId: number, username: string) {
  const ok = await confirm({
    message: t('dashboard.groupQuotaCard.revoke') + `: ${username}?`,
    danger: true,
  })
  if (!ok) return
  mutating.value = true
  try {
    await removeGroupViewGrant(props.groupId, userId)
    appStore.showSuccess(t('dashboard.groupQuotaCard.revokeSuccess'))
    await loadGrants()
    emit('change')
  } catch {
    appStore.showError(t('dashboard.groupQuotaCard.revokeFailed'))
  } finally {
    mutating.value = false
  }
}

onMounted(loadGrants)

defineExpose({ loadGrants })
</script>
