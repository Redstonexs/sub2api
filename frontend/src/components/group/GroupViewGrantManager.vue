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
      <span
        v-if="!loading && !loadError && grants.length > 0"
        class="text-xs text-gray-500 dark:text-gray-400"
      >
        {{ t('dashboard.groupQuotaCard.grantCount', { count: grants.length }) }}
      </span>
    </div>

    <!-- User search picker -->
    <div ref="pickerRef" class="relative">
      <div class="relative">
        <Icon
          name="search"
          size="sm"
          class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
        />
        <input
          ref="searchInputRef"
          v-model="searchQuery"
          type="text"
          role="combobox"
          autocomplete="off"
          aria-autocomplete="list"
          :aria-expanded="showDropdown"
          :disabled="mutating"
          :placeholder="t('dashboard.groupQuotaCard.searchUserPlaceholder')"
          class="input w-full pl-9"
          @input="scheduleSearch"
          @focus="openDropdown"
          @keydown.down.prevent="moveHighlight(1)"
          @keydown.up.prevent="moveHighlight(-1)"
          @keydown.enter.prevent="pickHighlighted"
          @keydown.esc="closeDropdown"
        />
      </div>

      <!-- Results dropdown. Teleported and fixed-positioned because this panel is
           usually rendered inside a dialog whose body is an overflow-y-auto scroll
           container — an in-tree absolute dropdown would be clipped by it. -->
      <Teleport to="body">
      <div
        v-if="showDropdown"
        ref="dropdownRef"
        role="listbox"
        :style="dropdownStyle"
        class="z-[60] overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-600 dark:bg-dark-700"
      >
        <div
          v-if="searchLoading"
          class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
        >
          {{ t('common.loading') }}
        </div>
        <div
          v-else-if="searchError"
          class="px-4 py-3 text-sm text-red-600 dark:text-red-400"
        >
          {{ searchError }}
        </div>
        <div
          v-else-if="options.length === 0"
          class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
        >
          {{ t('dashboard.groupQuotaCard.searchUserEmpty') }}
        </div>
        <template v-else>
          <button
            v-for="(option, index) in options"
            :key="option.kind === 'user' ? `u-${option.user.user_id}` : `raw-${option.userId}`"
            :ref="(el) => setOptionRef(el, index)"
            type="button"
            role="option"
            :aria-selected="index === highlightIndex"
            :disabled="mutating || (option.kind === 'user' && option.user.granted)"
            class="flex w-full items-center justify-between gap-3 px-4 py-2 text-left text-sm disabled:cursor-not-allowed"
            :class="[
              index === highlightIndex ? 'bg-gray-100 dark:bg-dark-600' : '',
              option.kind === 'user' && option.user.granted
                ? 'opacity-60'
                : 'hover:bg-gray-100 dark:hover:bg-dark-600',
            ]"
            @mouseenter="highlightIndex = index"
            @click="selectOption(option)"
          >
            <!-- Matched user -->
            <template v-if="option.kind === 'user'">
              <span class="flex min-w-0 flex-col">
                <span class="flex items-center gap-1.5">
                  <span class="truncate font-medium text-gray-900 dark:text-white">
                    {{ option.user.username }}
                  </span>
                  <span
                    v-if="option.user.role === 'admin'"
                    class="shrink-0 rounded bg-purple-100 px-1.5 py-0.5 text-[10px] font-medium uppercase text-purple-700 dark:bg-purple-900/30 dark:text-purple-300"
                  >
                    {{ t('dashboard.groupQuotaCard.roleAdmin') }}
                  </span>
                  <span
                    v-if="option.user.status !== 'active'"
                    class="shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium uppercase text-gray-600 dark:bg-dark-600 dark:text-gray-300"
                  >
                    {{ t('dashboard.groupQuotaCard.userDisabled') }}
                  </span>
                </span>
                <span class="truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ option.user.email }}
                </span>
              </span>
              <span class="shrink-0 text-xs text-gray-400">
                <template v-if="option.user.granted">
                  {{ t('dashboard.groupQuotaCard.alreadyGranted') }}
                </template>
                <template v-else>#{{ option.user.user_id }}</template>
              </span>
            </template>

            <!-- Raw user-ID fallback: keeps the original "grant by ID" path usable
                 when the admin knows the ID but the account has no searchable name -->
            <template v-else>
              <span class="truncate text-gray-700 dark:text-gray-200">
                {{ t('dashboard.groupQuotaCard.grantRawUserId', { id: option.userId }) }}
              </span>
              <span class="shrink-0 text-xs text-gray-400">#{{ option.userId }}</span>
            </template>
          </button>
        </template>
      </div>
      </Teleport>
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

      <template v-else>
        <!-- Filter over the existing grants, only once the list is long enough to need it -->
        <input
          v-if="grants.length > GRANT_FILTER_THRESHOLD"
          v-model="grantFilter"
          type="text"
          autocomplete="off"
          :placeholder="t('dashboard.groupQuotaCard.filterGrantedUsers')"
          class="input input-sm w-full"
        />

        <p
          v-if="visibleGrants.length === 0"
          class="text-sm text-gray-500 dark:text-gray-400"
        >
          {{ t('dashboard.groupQuotaCard.noMatchingGrants') }}
        </p>

        <!-- Grant rows -->
        <ul v-else class="space-y-2">
          <li
            v-for="grant in visibleGrants"
            :key="grant.user_id"
            class="flex items-center justify-between rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="min-w-0 flex-1">
              <span class="text-sm font-medium text-gray-900 dark:text-white">
                {{ grant.username }}
              </span>
              <span class="ml-2 text-xs text-gray-400">#{{ grant.user_id }}</span>
              <div class="mt-0.5 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                <span v-if="grant.email" class="truncate">{{ grant.email }}</span>
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
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useConfirm } from '@/composables/useConfirm'
import {
  getGroupViewGrants,
  searchGroupViewGrantCandidates,
  addGroupViewGrant,
  removeGroupViewGrant,
} from '@/api/admin/dashboard'
import { formatDateTime } from '@/utils/format'
import Icon from '@/components/icons/Icon.vue'
import type { GroupViewGrantItem, GroupViewGrantCandidate } from '@/types'

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

/** Show the grant-list filter box only once scanning by eye stops being practical. */
const GRANT_FILTER_THRESHOLD = 5
const SEARCH_DEBOUNCE_MS = 300

/** A dropdown row: either a matched user, or the "grant this raw ID" escape hatch. */
type PickerOption =
  | { kind: 'user'; user: GroupViewGrantCandidate }
  | { kind: 'rawId'; userId: number }

const grants = ref<GroupViewGrantItem[]>([])
const loading = ref(false)
const loadError = ref('')
const mutating = ref(false)
const grantFilter = ref('')

const pickerRef = ref<HTMLElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const dropdownStyle = ref<Record<string, string>>({})
const optionRefs = ref<(HTMLElement | null)[]>([])
const searchQuery = ref('')
const candidates = ref<GroupViewGrantCandidate[]>([])
const searchLoading = ref(false)
const searchError = ref('')
const showDropdown = ref(false)
const highlightIndex = ref(0)

let searchTimer: ReturnType<typeof setTimeout> | null = null
// Bumped on every new/cancelled search so late responses from stale queries are dropped.
let searchSequence = 0

const visibleGrants = computed(() => {
  const needle = grantFilter.value.trim().toLowerCase()
  if (!needle) return grants.value
  return grants.value.filter(
    (g) =>
      g.username.toLowerCase().includes(needle) ||
      (g.email ?? '').toLowerCase().includes(needle) ||
      String(g.user_id).includes(needle),
  )
})

const options = computed<PickerOption[]>(() => {
  const rows: PickerOption[] = candidates.value.map((user) => ({ kind: 'user', user }))
  // A purely numeric query that matched no account by ID still gets a direct grant row,
  // so the pre-search "add by user ID" workflow keeps working.
  const rawId = Number(searchQuery.value.trim())
  if (
    /^\d+$/.test(searchQuery.value.trim()) &&
    Number.isSafeInteger(rawId) &&
    rawId > 0 &&
    !candidates.value.some((u) => u.user_id === rawId)
  ) {
    rows.push({ kind: 'rawId', userId: rawId })
  }
  return rows
})

function setOptionRef(el: unknown, index: number): void {
  optionRefs.value[index] = (el as HTMLElement | null) ?? null
}

function cancelPendingSearch(): void {
  if (searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
  searchSequence += 1
}

async function runSearch(keyword: string): Promise<void> {
  cancelPendingSearch()
  const sequence = searchSequence
  searchLoading.value = true
  searchError.value = ''
  try {
    const results = await searchGroupViewGrantCandidates(props.groupId, keyword)
    if (sequence !== searchSequence) return
    candidates.value = results
    highlightIndex.value = 0
  } catch (err) {
    if (sequence !== searchSequence) return
    candidates.value = []
    searchError.value =
      (err as { message?: string }).message ?? t('common.unknownError')
  } finally {
    if (sequence === searchSequence) searchLoading.value = false
  }
}

function scheduleSearch(): void {
  cancelPendingSearch()
  showDropdown.value = true
  positionDropdown()
  const keyword = searchQuery.value.trim()
  const sequence = searchSequence
  searchTimer = setTimeout(() => {
    if (sequence !== searchSequence) return
    void runSearch(keyword)
  }, SEARCH_DEBOUNCE_MS)
}

/** Anchor the teleported dropdown to the search input, flipping above it when short on room. */
function positionDropdown(): void {
  const input = searchInputRef.value
  if (!input) return
  const rect = input.getBoundingClientRect()
  const gap = 4
  const below = window.innerHeight - rect.bottom - gap
  const above = rect.top - gap
  const flip = below < 160 && above > below
  const maxHeight = Math.max(96, Math.min(256, flip ? above : below))
  dropdownStyle.value = {
    position: 'fixed',
    left: `${rect.left}px`,
    width: `${rect.width}px`,
    maxHeight: `${maxHeight}px`,
    ...(flip
      ? { bottom: `${window.innerHeight - rect.top + gap}px` }
      : { top: `${rect.bottom + gap}px` }),
  }
}

function openDropdown(): void {
  showDropdown.value = true
  positionDropdown()
  // Empty-query results double as a "recent users" list, worth fetching once on focus.
  if (candidates.value.length === 0 && !searchLoading.value) {
    void runSearch(searchQuery.value.trim())
  }
}

function closeDropdown(): void {
  cancelPendingSearch()
  showDropdown.value = false
  searchLoading.value = false
}

async function moveHighlight(delta: number): Promise<void> {
  if (!showDropdown.value) {
    openDropdown()
    return
  }
  const count = options.value.length
  if (count === 0) return
  highlightIndex.value = (highlightIndex.value + delta + count) % count
  await nextTick()
  const el = optionRefs.value[highlightIndex.value]
  // Guarded: jsdom (and older engines) leave scrollIntoView undefined.
  if (typeof el?.scrollIntoView === 'function') {
    el.scrollIntoView({ block: 'nearest' })
  }
}

function pickHighlighted(): void {
  if (!showDropdown.value) return
  const option = options.value[highlightIndex.value]
  if (option) void selectOption(option)
}

async function selectOption(option: PickerOption): Promise<void> {
  if (option.kind === 'user') {
    if (option.user.granted) return
    await grantAccess(option.user.user_id)
    return
  }
  await grantAccess(option.userId)
}

async function loadGrants(): Promise<void> {
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

async function grantAccess(userId: number): Promise<void> {
  if (!Number.isSafeInteger(userId) || userId <= 0) return
  mutating.value = true
  try {
    await addGroupViewGrant(props.groupId, userId)
    appStore.showSuccess(t('dashboard.groupQuotaCard.grantSuccess'))
    // Flag the row in place instead of closing: admins usually authorize several
    // people in one sitting, and re-typing the search each time is the annoying part.
    candidates.value = candidates.value.map((u) =>
      u.user_id === userId ? { ...u, granted: true } : u,
    )
    await loadGrants()
    emit('change')
  } catch (err) {
    appStore.showError(
      (err as { message?: string }).message ?? t('dashboard.groupQuotaCard.grantFailed'),
    )
  } finally {
    mutating.value = false
  }
}

async function handleRevoke(userId: number, username: string): Promise<void> {
  const ok = await confirm({
    message: t('dashboard.groupQuotaCard.revoke') + `: ${username}?`,
    danger: true,
  })
  if (!ok) return
  mutating.value = true
  try {
    await removeGroupViewGrant(props.groupId, userId)
    appStore.showSuccess(t('dashboard.groupQuotaCard.revokeSuccess'))
    candidates.value = candidates.value.map((u) =>
      u.user_id === userId ? { ...u, granted: false } : u,
    )
    await loadGrants()
    emit('change')
  } catch {
    appStore.showError(t('dashboard.groupQuotaCard.revokeFailed'))
  } finally {
    mutating.value = false
  }
}

function handleDocumentClick(event: MouseEvent): void {
  const target = event.target as Node | null
  if (!target) return
  // The dropdown lives outside pickerRef (teleported), so it needs its own check —
  // otherwise clicking a result would close the list mid-grant.
  if (pickerRef.value?.contains(target) || dropdownRef.value?.contains(target)) return
  closeDropdown()
}

function handleViewportChange(): void {
  if (showDropdown.value) positionDropdown()
}

// Keep the option-ref array in step with the rendered rows so stale entries
// never get scrolled into view after results shrink.
watch(options, (next) => {
  optionRefs.value.length = next.length
  if (highlightIndex.value >= next.length) highlightIndex.value = 0
})

onMounted(() => {
  void loadGrants()
  document.addEventListener('click', handleDocumentClick)
  // Capture phase: the dialog body scrolls, and scroll events on it don't bubble.
  window.addEventListener('scroll', handleViewportChange, true)
  window.addEventListener('resize', handleViewportChange)
})

onUnmounted(() => {
  cancelPendingSearch()
  document.removeEventListener('click', handleDocumentClick)
  window.removeEventListener('scroll', handleViewportChange, true)
  window.removeEventListener('resize', handleViewportChange)
})

defineExpose({ loadGrants })
</script>
