<template>
  <section class="group-quota-card card overflow-hidden" :aria-label="tk('title')">
    <GroupQuotaCardControls
      :groups-loading="groupsLoading"
      :group-options="groupOptions"
      :selected-group-id="selectedGroupId"
      :sort-by="sortBy"
      :title="tk('title')"
      :select-group-label="tk('selectGroup')"
      :sort5h-label="tk('sortBy5h')"
      :sort7d-label="tk('sortBy7d')"
      @group-change="onGroupChange"
      @sort-change="setSortBy"
    />

    <div data-testid="group-quota-data-viewport" class="relative min-h-40 p-4" :aria-busy="isInitialLoad || isRefreshing">
      <div v-if="isInitialLoad" class="grid grid-cols-2 gap-3" aria-hidden="true">
        <div v-for="index in 2" :key="index" class="h-14 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-700" />
      </div>

      <div v-else-if="error && !cardData" class="py-4">
        <p class="text-sm text-danger-600 dark:text-danger-400">{{ tk('loadFailed') }}</p>
        <button class="btn btn-secondary mt-2 text-xs" @click="retry">Retry</button>
      </div>

      <div v-else-if="showNoViewableGroups" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ tk('noViewableGroups') }}
      </div>

      <Transition name="group-quota-data" mode="out-in">
        <div v-if="cardData" :key="cardContentKey" data-testid="group-quota-content" class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div v-for="window in totalWindows" :key="window.label" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/50">
              <div class="mb-1 text-[10px] font-medium text-gray-500 dark:text-gray-400">
                {{ tk(window.totalKey) }}
              </div>
              <UsageProgressBar
                v-if="window.remaining != null"
                :label="window.label"
                :utilization="100 - window.remaining"
                :resets-at="null"
                :color="window.color"
              />
              <div v-else class="text-xs text-gray-400 dark:text-gray-500">
                {{ tk('noData') }}
              </div>
            </div>
          </div>

          <div>
            <div class="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-300">
              {{ tk('perAccountUsage') }}
            </div>
            <div v-if="cardData.accounts.length === 0" class="py-4 text-center text-xs text-gray-400 dark:text-gray-500">
              {{ tk('noAccounts') }}
            </div>
            <div v-else class="group-quota-card__account-list space-y-2 pr-1">
              <div
                v-for="acct in cardData.accounts"
                :key="acct.account_id ?? acct.display_name"
                class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/50"
              >
                <div class="mb-1.5 truncate text-xs font-medium text-gray-900 dark:text-white">
                  {{ acct.display_name }}
                </div>
                <div class="space-y-1">
                  <UsageProgressBar
                    v-if="acct.five_hour"
                    label="5h"
                    :utilization="acct.five_hour.utilization"
                    :resets-at="acct.five_hour.resets_at"
                    color="amber"
                  />
                  <div v-else class="pl-[33px] text-[10px] text-gray-400 dark:text-gray-500">
                    {{ tk('noData') }}
                  </div>
                  <UsageProgressBar
                    v-if="acct.seven_day"
                    label="7d"
                    :utilization="acct.seven_day.utilization"
                    :resets-at="acct.seven_day.resets_at"
                    color="emerald"
                  />
                  <div v-else class="pl-[33px] text-[10px] text-gray-400 dark:text-gray-500">
                    {{ tk('noData') }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>

      <p v-if="error && cardData" class="mt-3 text-xs text-danger-600 dark:text-danger-400" role="alert">
        {{ tk('loadFailed') }}
      </p>

      <Transition name="group-quota-loading">
        <div v-if="isRefreshing" data-testid="group-quota-data-loading" class="pointer-events-none absolute inset-x-3 top-3 z-10 flex justify-center">
          <span class="flex items-center gap-2 rounded-full border border-gray-200 bg-white/95 px-3 py-1.5 text-xs font-medium text-gray-600 shadow-card dark:border-dark-600 dark:bg-dark-800/95 dark:text-gray-300">
            <span class="h-3 w-3 animate-spin rounded-full border-2 border-primary-200 border-t-primary-600" />
            {{ tk('loading') }}
          </span>
        </div>
      </Transition>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import UsageProgressBar from '@/components/account/UsageProgressBar.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import GroupQuotaCardControls from './GroupQuotaCardControls.vue'
import { groupsAPI } from '@/api/admin/groups'
import { dashboardAPI } from '@/api/admin/dashboard'
import { userAPI } from '@/api/user'
import type { GroupQuotaCard, ViewableGroup, AdminGroup } from '@/types'

const props = defineProps<{
  isAdmin: boolean
}>()

const { t } = useI18n()

// i18n prefix helper
const prefix = computed(() =>
  props.isAdmin ? 'admin.dashboard.groupQuotaCard' : 'dashboard.groupQuotaCard'
)
const tk = (key: string) => t(`${prefix.value}.${key}`)

// State
const availableGroups = ref<ViewableGroup[]>([])
const selectedGroupId = ref<number | null>(null)
const sortBy = ref<'5h' | '7d'>('5h')
const cardData = ref<GroupQuotaCard | null>(null)
const loading = ref(false)
const groupsLoading = ref(true)
const error = ref<string | null>(null)
let cardRequestId = 0
const cardContentVersion = ref(0)

const groupOptions = computed<SelectOption[]>(() =>
  availableGroups.value.map((group) => ({ value: group.group_id, label: group.group_name }))
)
const isInitialLoad = computed(() => groupsLoading.value || (loading.value && cardData.value === null))
const isRefreshing = computed(() => loading.value && cardData.value !== null)
const showNoViewableGroups = computed(() => !props.isAdmin && !groupsLoading.value && availableGroups.value.length === 0)
const cardContentKey = computed(() => `${cardData.value?.group_id ?? 'empty'}-${cardContentVersion.value}`)
const totalWindows = computed(() => [
  { label: '5h', totalKey: 'totalRemaining5h', remaining: cardData.value?.total_remaining_5h, color: 'amber' as const },
  { label: '7d', totalKey: 'totalRemaining7d', remaining: cardData.value?.total_remaining_7d, color: 'emerald' as const }
])

async function loadGroups() {
  groupsLoading.value = true
  error.value = null
  try {
    if (props.isAdmin) {
      const all = await groupsAPI.getAll()
      availableGroups.value = all
        .filter((g: AdminGroup) => g.platform === 'anthropic' || g.platform === 'openai')
        .map((g: AdminGroup) => ({
          group_id: g.id,
          group_name: g.name,
          platform: g.platform
        }))
    } else {
      availableGroups.value = await userAPI.getMyViewableGroups()
    }

    if (availableGroups.value.length > 0) {
      selectedGroupId.value = availableGroups.value[0].group_id
      await loadCard()
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    groupsLoading.value = false
  }
}

async function loadCard() {
  if (!selectedGroupId.value) return
  const requestId = ++cardRequestId
  loading.value = true
  error.value = null
  try {
    let nextCardData: GroupQuotaCard
    if (props.isAdmin) {
      nextCardData = await dashboardAPI.getGroupQuotaCard(selectedGroupId.value, sortBy.value)
    } else {
      nextCardData = await userAPI.getGroupQuotaCard(selectedGroupId.value, sortBy.value)
    }
    if (requestId === cardRequestId) {
      cardData.value = nextCardData
      cardContentVersion.value += 1
    }
  } catch (cause) {
    if (requestId === cardRequestId) {
      error.value = cause instanceof Error ? cause.message : String(cause)
    }
  } finally {
    if (requestId === cardRequestId) loading.value = false
  }
}

function onGroupChange(value: string | number | boolean | null) {
  if (typeof value !== 'number' || value === selectedGroupId.value) return
  selectedGroupId.value = value
  loadCard()
}

function setSortBy(value: '5h' | '7d') {
  if (sortBy.value === value) return
  sortBy.value = value
  loadCard()
}

function retry() {
  if (selectedGroupId.value) {
    loadCard()
  } else {
    loadGroups()
  }
}

onMounted(loadGroups)
</script>

<style scoped>
.group-quota-card__account-list {
  max-block-size: 14rem;
  overflow-y: auto;
}

.group-quota-data-enter-active,
.group-quota-data-leave-active,
.group-quota-loading-enter-active,
.group-quota-loading-leave-active {
  transition: opacity 200ms ease-out, transform 200ms ease-out;
}

.group-quota-data-enter-from,
.group-quota-data-leave-to {
  opacity: 0;
  transform: translateY(0.25rem);
}

.group-quota-loading-enter-from,
.group-quota-loading-leave-to {
  opacity: 0;
  transform: translateY(-0.25rem);
}

@media (prefers-reduced-motion: reduce) {
  .group-quota-data-enter-active,
  .group-quota-data-leave-active,
  .group-quota-loading-enter-active,
  .group-quota-loading-leave-active {
    transition: none;
  }
}
</style>
