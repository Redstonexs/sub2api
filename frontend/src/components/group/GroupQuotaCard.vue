<template>
  <div class="card p-4">
    <!-- Loading state -->
    <div v-if="loading" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ tk('loading') }}
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="py-4">
      <p class="text-sm text-red-600 dark:text-red-400">{{ tk('loadFailed') }}</p>
      <button class="btn btn-secondary mt-2 text-xs" @click="retry">Retry</button>
    </div>

    <!-- No viewable groups (user mode) -->
    <div v-else-if="!isAdmin && availableGroups.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ tk('noViewableGroups') }}
    </div>

    <!-- Main content -->
    <template v-else>
      <!-- Header row: title + group selector + sort buttons -->
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ tk('title') }}
        </h3>
        <div class="flex items-center gap-2">
          <!-- Group selector -->
          <select
            v-if="availableGroups.length > 0"
            :value="selectedGroupId"
            @change="onGroupChange"
            class="input text-xs py-1.5 pr-8"
          >
            <option disabled value="">{{ tk('selectGroup') }}</option>
            <option v-for="g in availableGroups" :key="g.group_id" :value="g.group_id">
              {{ g.group_name }}
            </option>
          </select>
          <!-- Sort toggle buttons -->
          <div class="flex rounded-lg border border-gray-200 dark:border-dark-600 overflow-hidden">
            <button
              data-testid="sort-5h"
              :class="[
                'px-2.5 py-1 text-xs font-medium transition-colors',
                sortBy === '5h'
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                  : 'bg-white text-gray-600 hover:bg-gray-50 dark:bg-dark-800 dark:text-gray-400 dark:hover:bg-dark-700'
              ]"
              @click="setSortBy('5h')"
            >
              {{ tk('sortBy5h') }}
            </button>
            <button
              data-testid="sort-7d"
              :class="[
                'px-2.5 py-1 text-xs font-medium transition-colors border-l border-gray-200 dark:border-dark-600',
                sortBy === '7d'
                  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
                  : 'bg-white text-gray-600 hover:bg-gray-50 dark:bg-dark-800 dark:text-gray-400 dark:hover:bg-dark-700'
              ]"
              @click="setSortBy('7d')"
            >
              {{ tk('sortBy7d') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Totals row -->
      <div class="mb-3 grid grid-cols-2 gap-3">
        <!-- 5h total remaining -->
        <div class="rounded-lg bg-gray-50 dark:bg-dark-700/50 p-2.5">
          <div class="mb-1 text-[10px] font-medium text-gray-500 dark:text-gray-400">
            {{ tk('totalRemaining5h') }}
          </div>
          <template v-if="cardData?.total_remaining_5h != null">
            <UsageProgressBar
              label="5h"
              :utilization="100 - cardData.total_remaining_5h"
              :resets-at="null"
              color="amber"
            />
          </template>
          <div v-else class="text-xs text-gray-400 dark:text-gray-500">
            {{ tk('noData') }}
          </div>
        </div>
        <!-- 7d total remaining -->
        <div class="rounded-lg bg-gray-50 dark:bg-dark-700/50 p-2.5">
          <div class="mb-1 text-[10px] font-medium text-gray-500 dark:text-gray-400">
            {{ tk('totalRemaining7d') }}
          </div>
          <template v-if="cardData?.total_remaining_7d != null">
            <UsageProgressBar
              label="7d"
              :utilization="100 - cardData.total_remaining_7d"
              :resets-at="null"
              color="emerald"
            />
          </template>
          <div v-else class="text-xs text-gray-400 dark:text-gray-500">
            {{ tk('noData') }}
          </div>
        </div>
      </div>

      <!-- Per-account section -->
      <div v-if="cardData">
        <div class="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-300">
          {{ tk('perAccountUsage') }}
        </div>

        <!-- No accounts -->
        <div v-if="cardData.accounts.length === 0" class="py-4 text-center text-xs text-gray-400 dark:text-gray-500">
          {{ tk('noAccounts') }}
        </div>

        <!-- Account rows -->
        <div v-else class="space-y-2">
          <div
            v-for="acct in cardData.accounts"
            :key="acct.account_id ?? acct.display_name"
            class="rounded-lg bg-gray-50 dark:bg-dark-700/50 p-2.5"
          >
            <div class="mb-1.5 text-xs font-medium text-gray-900 dark:text-white">
              {{ acct.display_name }}
            </div>
            <div class="space-y-1">
              <!-- 5h window -->
              <template v-if="acct.five_hour">
                <UsageProgressBar
                  label="5h"
                  :utilization="acct.five_hour.utilization"
                  :resets-at="acct.five_hour.resets_at"
                  color="amber"
                />
              </template>
              <div v-else class="text-[10px] text-gray-400 dark:text-gray-500 pl-[33px]">
                {{ tk('noData') }}
              </div>
              <!-- 7d window -->
              <template v-if="acct.seven_day">
                <UsageProgressBar
                  label="7d"
                  :utilization="acct.seven_day.utilization"
                  :resets-at="acct.seven_day.resets_at"
                  color="emerald"
                />
              </template>
              <div v-else class="text-[10px] text-gray-400 dark:text-gray-500 pl-[33px]">
                {{ tk('noData') }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import UsageProgressBar from '@/components/account/UsageProgressBar.vue'
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
const error = ref<string | null>(null)

// Load groups on mount
async function loadGroups() {
  try {
    if (props.isAdmin) {
      const all = await groupsAPI.getAll()
      // Filter to anthropic|openai platforms
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

    // Auto-select first group
    if (availableGroups.value.length > 0) {
      selectedGroupId.value = availableGroups.value[0].group_id
      await loadCard()
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

// Load card data for selected group
async function loadCard() {
  if (!selectedGroupId.value) return
  loading.value = true
  error.value = null
  try {
    if (props.isAdmin) {
      cardData.value = await dashboardAPI.getGroupQuotaCard(selectedGroupId.value, sortBy.value)
    } else {
      cardData.value = await userAPI.getGroupQuotaCard(selectedGroupId.value, sortBy.value)
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
    cardData.value = null
  } finally {
    loading.value = false
  }
}

function onGroupChange(event: Event) {
  const el = event.currentTarget
  if (!(el instanceof HTMLSelectElement)) return
  const val = el.value
  selectedGroupId.value = val ? Number(val) : null
  if (selectedGroupId.value) {
    loadCard()
  }
}

function setSortBy(value: '5h' | '7d') {
  if (sortBy.value !== value) {
    sortBy.value = value
    if (selectedGroupId.value) {
      loadCard()
    }
  }
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
