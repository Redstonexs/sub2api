<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <!-- Left: Search + Filters -->
          <div class="flex-1 sm:max-w-64">
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('admin.announcements.searchAnnouncements')"
              class="input"
              @input="handleSearch"
            />
          </div>
          <Select
            v-model="filters.status"
            :options="statusFilterOptions"
            class="w-40"
            @change="handleStatusChange"
          />

          <!-- Right: Action buttons -->
          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              @click="loadAnnouncements"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button @click="openCreateDialog" class="btn btn-primary">
              <Icon name="plus" size="md" class="mr-1" />
              {{ t('admin.announcements.createAnnouncement') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="announcements"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-title="{ value, row }">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium text-gray-900 dark:text-white">{{ value }}</span>
              </div>
              <div class="mt-1 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                <span>#{{ row.id }}</span>
                <span class="text-gray-300 dark:text-dark-700">·</span>
                <span>{{ formatDateTime(row.created_at) }}</span>
              </div>
            </div>
          </template>

          <template #cell-status="{ value, row }">
            <div class="flex flex-wrap items-center gap-1">
              <span
                :class="[
                  'badge',
                  value === 'active'
                    ? 'badge-success'
                    : value === 'draft'
                      ? 'badge-gray'
                      : 'badge-warning'
                ]"
              >
                {{ statusLabel(value) }}
              </span>
              <!-- "Active" only means published; a schedule can still make it invisible. -->
              <span
                v-if="lifecycleOf(row)"
                :class="['badge', lifecycleOf(row) === 'live' ? 'badge-primary' : 'badge-gray']"
              >
                {{ t(`admin.announcements.lifecycle.${lifecycleOf(row)}`) }}
              </span>
            </div>
          </template>

          <template #cell-severity="{ row }">
            <span
              :class="[
                'badge',
                row.severity === 'critical'
                  ? 'badge-danger'
                  : row.severity === 'warning'
                    ? 'badge-warning'
                    : 'badge-primary'
              ]"
            >
              {{ t(`admin.announcements.severityLabels.${row.severity || 'info'}`) }}
            </span>
            <span v-if="row.show_banner" class="badge badge-gray ml-1">
              {{ t('admin.announcements.form.showBanner') }}
            </span>
          </template>

          <template #cell-notify_mode="{ row }">
            <span
              :class="[
                'badge',
                row.notify_mode === 'popup'
                  ? 'badge-warning'
                  : row.notify_mode === 'email'
                    ? 'badge-primary'
                    : 'badge-gray'
              ]"
            >
              {{ t(`admin.announcements.notifyModeLabels.${row.notify_mode || 'silent'}`) }}
            </span>
          </template>

          <template #cell-targeting="{ row }">
            <span class="text-sm text-gray-600 dark:text-gray-300">
              {{ targetingSummary(row.targeting) }}
            </span>
          </template>

          <template #cell-timeRange="{ row }">
            <div class="text-sm text-gray-600 dark:text-gray-300">
              <div>
                <span class="font-medium">{{ t('admin.announcements.form.startsAt') }}:</span>
                <span class="ml-1">{{ row.starts_at ? formatDateTime(row.starts_at) : t('admin.announcements.timeImmediate') }}</span>
              </div>
              <div class="mt-0.5">
                <span class="font-medium">{{ t('admin.announcements.form.endsAt') }}:</span>
                <span class="ml-1">{{ row.ends_at ? formatDateTime(row.ends_at) : t('admin.announcements.timeNever') }}</span>
              </div>
            </div>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center space-x-1">
              <button
                @click="openPreview(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                :title="t('admin.announcements.preview')"
              >
                <Icon name="eye" size="sm" />
              </button>
              <button
                @click="openReadStatus(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                :title="t('admin.announcements.readStatus')"
              >
                <Icon name="chartBar" size="sm" />
              </button>
              <button
                @click="openEditDialog(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('common.edit')"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                @click="openDuplicateDialog(row)"
                data-testid="announcement-duplicate"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('admin.announcements.duplicate')"
              >
                <Icon name="copy" size="sm" />
              </button>
              <button
                @click="handleDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('common.delete')"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('empty.noData')"
              :description="t('admin.announcements.failedToLoad')"
              :action-text="t('admin.announcements.createAnnouncement')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create/Edit Dialog -->
    <BaseDialog
      :show="showEditDialog"
      :title="isEditing ? t('admin.announcements.editAnnouncement') : t('admin.announcements.createAnnouncement')"
      width="extra-wide"
      @close="requestCloseEdit"
    >
      <form id="announcement-form" @submit.prevent="handleSave" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.announcements.form.title') }}</label>
          <input v-model="form.title" type="text" class="input" required />
        </div>

        <div>
          <label class="input-label">{{ t('admin.announcements.form.content') }}</label>
          <MarkdownEditor
            v-model="form.content"
            :max-length="ANNOUNCEMENT_CONTENT_MAX_CHARS"
            :placeholder="t('admin.announcements.form.contentPlaceholder')"
          />
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.announcements.form.status') }}</label>
            <Select v-model="form.status" :options="statusOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.notifyMode') }}</label>
            <Select v-model="form.notify_mode" :options="notifyModeOptions" />
            <p class="input-hint">{{ t('admin.announcements.form.notifyModeHint') }}</p>

            <div class="mt-2 flex flex-wrap items-center gap-2">
              <button
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="estimatingAudience"
                data-testid="announcement-estimate-audience"
                @click="estimateAudience"
              >
                {{ estimatingAudience ? t('common.loading') : t('admin.announcements.estimateAudience') }}
              </button>
              <span
                v-if="audienceStats"
                class="text-sm text-gray-600 dark:text-gray-300"
                data-testid="announcement-audience-result"
              >
                {{ audienceSummary }}
              </span>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.announcements.form.severity') }}</label>
            <Select v-model="form.severity" :options="severityOptions" data-testid="announcement-severity" />
            <p class="input-hint">{{ t('admin.announcements.form.severityHint') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.showBanner') }}</label>
            <label class="mt-2 flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input
                v-model="form.show_banner"
                type="checkbox"
                data-testid="announcement-show-banner"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span>{{ t('admin.announcements.form.showBannerLabel') }}</span>
            </label>
            <p class="input-hint">{{ t('admin.announcements.form.showBannerHint') }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.announcements.form.startsAt') }}</label>
            <input v-model="form.starts_at_str" type="datetime-local" class="input" />
            <p class="input-hint">{{ t('admin.announcements.form.startsAtHint') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.endsAt') }}</label>
            <input v-model="form.ends_at_str" type="datetime-local" class="input" />
            <p class="input-hint">{{ t('admin.announcements.form.endsAtHint') }}</p>
          </div>
        </div>
        <p class="input-hint -mt-2">{{ t('admin.announcements.form.scheduleHint') }}</p>
        <p v-if="scheduleError" class="text-sm text-red-600 dark:text-red-400" data-testid="announcement-schedule-error">
          {{ scheduleError }}
        </p>

        <AnnouncementTargetingEditor
          v-model="form.targeting"
          :groups="subscriptionGroups"
        />
      </form>

      <template #footer>
        <div class="flex flex-wrap justify-end gap-3">
          <!-- Edit mode only: a test send needs a persisted announcement id. -->
          <button
            v-if="isEditing"
            type="button"
            class="btn btn-secondary mr-auto"
            :disabled="sendingTestEmail"
            data-testid="announcement-send-test"
            @click="sendTestEmail"
          >
            {{ sendingTestEmail ? t('common.sending') : t('admin.announcements.sendTestEmail') }}
          </button>
          <button type="button" @click="requestCloseEdit" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="announcement-form" :disabled="saving" class="btn btn-primary">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.announcements.deleteAnnouncement')"
      :message="t('admin.announcements.deleteConfirm')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Read Status Dialog -->
    <AnnouncementReadStatusDialog
      :show="showReadStatusDialog"
      :announcement-id="readStatusAnnouncementId"
      @close="showReadStatusDialog = false"
    />

    <AnnouncementPopup
      :announcement="previewAnnouncement"
      preview
      @close="previewAnnouncement = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { adminAPI } from '@/api/admin'
import { formatDateTime, formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { useConfirm } from '@/composables/useConfirm'
import type {
  AdminGroup,
  Announcement,
  AnnouncementAudienceStats,
  AnnouncementSeverity,
  AnnouncementTargeting,
} from '@/types'
import type { Column } from '@/components/common/types'

import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import MarkdownEditor from '@/components/common/MarkdownEditor.vue'
import Icon from '@/components/icons/Icon.vue'

import AnnouncementTargetingEditor from '@/components/admin/announcements/AnnouncementTargetingEditor.vue'
import AnnouncementReadStatusDialog from '@/components/admin/announcements/AnnouncementReadStatusDialog.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'

const { t } = useI18n()
const appStore = useAppStore()
const { confirm } = useConfirm()

/** Mirrors announcementMaxContentRunes in backend/internal/service/announcement_service.go. */
const ANNOUNCEMENT_CONTENT_MAX_CHARS = 20000
/** Mirrors the targeting limits enforced by domain.AnnouncementTargeting.NormalizeAndValidate. */
const TARGETING_MAX_GROUPS = 50
const TARGETING_MAX_CONDITIONS = 50

const announcements = ref<Announcement[]>([])
const loading = ref(false)

/**
 * Resolves a backend error to a localized message.
 *
 * The API client rejects with a *flat* `{status, code, reason, metadata}` object,
 * so the `error.response?.data?.detail` this view used to read was always
 * undefined and every backend error code was discarded.
 */
function announcementError(error: unknown, fallbackKey: string): string {
  return extractI18nErrorMessage(error, t, 'admin.announcements.errors', t(fallbackKey))
}

const filters = reactive({
  status: '',
})
const searchQuery = ref('')

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const statusFilterOptions = computed(() => [
  { value: '', label: t('admin.announcements.allStatus') },
  { value: 'draft', label: t('admin.announcements.statusLabels.draft') },
  { value: 'active', label: t('admin.announcements.statusLabels.active') },
  { value: 'archived', label: t('admin.announcements.statusLabels.archived') }
])

const statusOptions = computed(() => [
  { value: 'draft', label: t('admin.announcements.statusLabels.draft') },
  { value: 'active', label: t('admin.announcements.statusLabels.active') },
  { value: 'archived', label: t('admin.announcements.statusLabels.archived') }
])

const notifyModeOptions = computed(() => [
  { value: 'silent', label: t('admin.announcements.notifyModeLabels.silent') },
  { value: 'popup', label: t('admin.announcements.notifyModeLabels.popup') },
  { value: 'email', label: t('admin.announcements.notifyModeLabels.email') }
])

const severityOptions = computed(() => [
  { value: 'info', label: t('admin.announcements.severityLabels.info') },
  { value: 'warning', label: t('admin.announcements.severityLabels.warning') },
  { value: 'critical', label: t('admin.announcements.severityLabels.critical') }
])

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('admin.announcements.columns.title'), sortable: true },
  { key: 'status', label: t('admin.announcements.columns.status'), sortable: true },
  { key: 'notify_mode', label: t('admin.announcements.columns.notifyMode'), sortable: true },
  { key: 'severity', label: t('admin.announcements.columns.severity'), sortable: true },
  { key: 'targeting', label: t('admin.announcements.columns.targeting') },
  { key: 'timeRange', label: t('admin.announcements.columns.timeRange') },
  { key: 'created_at', label: t('admin.announcements.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.announcements.columns.actions') }
])

const statusLabel = (status: string) => {
  if (status === 'draft') return t('admin.announcements.statusLabels.draft')
  if (status === 'active') return t('admin.announcements.statusLabels.active')
  if (status === 'archived') return t('admin.announcements.statusLabels.archived')
  return status
}

const targetingSummary = (targeting: AnnouncementTargeting) => {
  const anyOf = targeting?.any_of ?? []
  if (!anyOf || anyOf.length === 0) return t('admin.announcements.targetingSummaryAll')
  return t('admin.announcements.targetingSummaryCustom', { groups: anyOf.length })
}

// ===== CRUD / list =====
let currentController: AbortController | null = null

async function loadAnnouncements() {
  currentController?.abort()
  const requestController = new AbortController()
  currentController = requestController
  const { signal } = requestController

  try {
    loading.value = true
    const res = await adminAPI.announcements.list(pagination.page, pagination.page_size, {
      status: filters.status || undefined,
      search: searchQuery.value || undefined,
      sort_by: sortState.sort_by,
      sort_order: sortState.sort_order
    }, { signal })

    if (signal.aborted || currentController !== requestController) return

    announcements.value = res.items
    pagination.total = res.total
    pagination.pages = res.pages
    pagination.page = res.page
    pagination.page_size = res.page_size
  } catch (error: any) {
    if (
      signal.aborted ||
      currentController !== requestController ||
      error?.name === 'AbortError' ||
      error?.code === 'ERR_CANCELED'
    ) {
      return
    }
    console.error('Error loading announcements:', error)
    appStore.showError(announcementError(error, 'admin.announcements.failedToLoad'))
  } finally {
    if (currentController === requestController) {
      loading.value = false
      currentController = null
    }
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadAnnouncements()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadAnnouncements()
}

function handleStatusChange() {
  pagination.page = 1
  loadAnnouncements()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadAnnouncements()
}

let searchDebounceTimer: number | null = null
function handleSearch() {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  searchDebounceTimer = window.setTimeout(() => {
    pagination.page = 1
    loadAnnouncements()
  }, 300)
}

// ===== Create/Edit dialog =====
const showEditDialog = ref(false)
const saving = ref(false)
const editingAnnouncement = ref<Announcement | null>(null)

const isEditing = computed(() => !!editingAnnouncement.value)

const form = reactive({
  title: '',
  content: '',
  status: 'draft',
  notify_mode: 'silent',
  severity: 'info' as AnnouncementSeverity,
  show_banner: false,
  starts_at_str: '',
  ends_at_str: '',
  targeting: { any_of: [] } as AnnouncementTargeting
})

const subscriptionGroups = ref<AdminGroup[]>([])

async function loadSubscriptionGroups() {
  try {
    const all = await adminAPI.groups.getAll()
    subscriptionGroups.value = (all || []).filter((g) => g.subscription_type === 'subscription')
  } catch (error: any) {
    console.error('Error loading groups:', error)
    // not fatal
  }
}

function resetForm() {
  form.title = ''
  form.content = ''
  form.status = 'draft'
  form.notify_mode = 'silent'
  form.severity = 'info'
  form.show_banner = false
  form.starts_at_str = ''
  form.ends_at_str = ''
  form.targeting = { any_of: [] }
}

function fillFormFromAnnouncement(a: Announcement) {
  form.title = a.title
  form.content = a.content
  form.status = a.status
  form.notify_mode = a.notify_mode || 'silent'
  form.severity = a.severity || 'info'
  form.show_banner = a.show_banner ?? false

  // Backend returns RFC3339 strings
  form.starts_at_str = a.starts_at ? formatDateTimeLocalInput(Math.floor(new Date(a.starts_at).getTime() / 1000)) : ''
  form.ends_at_str = a.ends_at ? formatDateTimeLocalInput(Math.floor(new Date(a.ends_at).getTime() / 1000)) : ''

  form.targeting = a.targeting ?? { any_of: [] }
}

/** JSON snapshot of the form as opened, so we can detect unsaved edits on close. */
const pristineForm = ref('')

function snapshotForm() {
  pristineForm.value = JSON.stringify(form)
  audienceStats.value = null
}

const isDirty = computed(() => JSON.stringify(form) !== pristineForm.value)

function openCreateDialog() {
  editingAnnouncement.value = null
  resetForm()
  snapshotForm()
  showEditDialog.value = true
}

function openEditDialog(row: Announcement) {
  editingAnnouncement.value = row
  fillFormFromAnnouncement(row)
  snapshotForm()
  showEditDialog.value = true
}

/**
 * Opens the create dialog pre-filled from an existing announcement, always as a
 * draft: duplicating a live email announcement and saving it unchanged would fan
 * a second broadcast out to every recipient.
 */
function openDuplicateDialog(row: Announcement) {
  editingAnnouncement.value = null
  fillFormFromAnnouncement(row)
  form.title = `${row.title} ${t('admin.announcements.duplicateTitleSuffix')}`
  form.status = 'draft'
  snapshotForm()
  showEditDialog.value = true
}

function closeEdit() {
  showEditDialog.value = false
  editingAnnouncement.value = null
}

async function requestCloseEdit() {
  if (isDirty.value) {
    const discard = await confirm({
      title: t('admin.announcements.unsavedChangesTitle'),
      message: t('admin.announcements.unsavedChangesMessage'),
      confirmText: t('admin.announcements.discard'),
      cancelText: t('common.cancel'),
      danger: true,
    })
    if (!discard) return
  }
  closeEdit()
}

const contentTooLong = computed(() => [...form.content].length > ANNOUNCEMENT_CONTENT_MAX_CHARS)

// ===== Email broadcast safety rails =====
const audienceStats = ref<AnnouncementAudienceStats | null>(null)
const estimatingAudience = ref(false)
const sendingTestEmail = ref(false)

const audienceSummary = computed(() => {
  const stats = audienceStats.value
  if (!stats) return ''
  const key = stats.truncated
    ? 'admin.announcements.audienceSummaryTruncated'
    : 'admin.announcements.audienceSummary'
  return t(key, {
    deliverable: stats.deliverable,
    matched: stats.matched,
    unsubscribed: stats.unsubscribed,
  })
})

// Targeting edits invalidate a previous estimate; a stale count is worse than none
// because it is what the publish gate shows.
watch(() => JSON.stringify(form.targeting), () => { audienceStats.value = null })

async function estimateAudience(): Promise<AnnouncementAudienceStats | null> {
  estimatingAudience.value = true
  try {
    const stats = await adminAPI.announcements.previewAudience(form.targeting)
    audienceStats.value = stats
    return stats
  } catch (error: any) {
    console.error('Failed to estimate announcement audience:', error)
    appStore.showError(announcementError(error, 'admin.announcements.failedToEstimateAudience'))
    return null
  } finally {
    estimatingAudience.value = false
  }
}

async function sendTestEmail() {
  if (!editingAnnouncement.value) return
  sendingTestEmail.value = true
  try {
    const { recipient } = await adminAPI.announcements.sendTestEmail(editingAnnouncement.value.id)
    appStore.showSuccess(t('admin.announcements.testEmailSent', { email: recipient }))
  } catch (error: any) {
    console.error('Failed to send announcement test email:', error)
    appStore.showError(announcementError(error, 'admin.announcements.failedToSendTestEmail'))
  } finally {
    sendingTestEmail.value = false
  }
}

/**
 * True when saving would flip this announcement into active+email, i.e. fan a
 * broadcast out to every matched user. That currently happens on one ordinary Save
 * click, so it gets an explicit confirmation showing the recipient count.
 */
const willBroadcast = computed(() => {
  const becomingActiveEmail = form.status === 'active' && form.notify_mode === 'email'
  if (!becomingActiveEmail) return false
  const original = editingAnnouncement.value
  if (!original) return true
  // maybeBroadcastEmail only fires on the transition into active+email.
  return !(original.status === 'active' && original.notify_mode === 'email')
})

async function confirmBroadcast(): Promise<boolean> {
  // Estimate fresh rather than trusting a stale count from earlier in the session.
  const stats = audienceStats.value ?? await estimateAudience()
  const message = stats
    ? t('admin.announcements.broadcastConfirmMessage', {
        deliverable: stats.deliverable,
        suffix: stats.truncated ? t('admin.announcements.broadcastConfirmTruncated') : '',
      })
    : t('admin.announcements.broadcastConfirmUnknown')

  return confirm({
    title: t('admin.announcements.broadcastConfirmTitle'),
    message,
    confirmText: t('admin.announcements.broadcastConfirmAction'),
    cancelText: t('common.cancel'),
    danger: true,
  })
}

const scheduleError = computed(() => {
  const startsAt = parseDateTimeLocalInput(form.starts_at_str)
  const endsAt = parseDateTimeLocalInput(form.ends_at_str)
  if (startsAt !== null && endsAt !== null && startsAt >= endsAt) {
    return t('admin.announcements.errors.ANNOUNCEMENT_TIME_RANGE_INVALID')
  }
  return ''
})

/**
 * Whether an *active* announcement is currently visible to users. Draft and
 * archived rows return null — their status badge already says everything.
 */
function lifecycleOf(row: Announcement): 'scheduled' | 'live' | 'expired' | null {
  if (row.status !== 'active') return null
  const now = Date.now()
  if (row.starts_at && new Date(row.starts_at).getTime() > now) return 'scheduled'
  if (row.ends_at && new Date(row.ends_at).getTime() <= now) return 'expired'
  return 'live'
}

function buildCreatePayload() {
  const startsAt = parseDateTimeLocalInput(form.starts_at_str)
  const endsAt = parseDateTimeLocalInput(form.ends_at_str)

  return {
    title: form.title,
    content: form.content,
    status: form.status as any,
    notify_mode: form.notify_mode as any,
    severity: form.severity,
    show_banner: form.show_banner,
    targeting: form.targeting,
    starts_at: startsAt ?? undefined,
    ends_at: endsAt ?? undefined
  }
}

function buildUpdatePayload(original: Announcement) {
  const payload: any = {}

  if (form.title !== original.title) payload.title = form.title
  if (form.content !== original.content) payload.content = form.content
  if (form.status !== original.status) payload.status = form.status
  if (form.notify_mode !== (original.notify_mode || 'silent')) payload.notify_mode = form.notify_mode
  if (form.severity !== (original.severity || 'info')) payload.severity = form.severity
  if (form.show_banner !== (original.show_banner ?? false)) payload.show_banner = form.show_banner

  // starts_at / ends_at: distinguish unchanged vs clear(0) vs set
  const originalStarts = original.starts_at ? Math.floor(new Date(original.starts_at).getTime() / 1000) : null
  const originalEnds = original.ends_at ? Math.floor(new Date(original.ends_at).getTime() / 1000) : null

  const newStarts = parseDateTimeLocalInput(form.starts_at_str)
  const newEnds = parseDateTimeLocalInput(form.ends_at_str)

  if (newStarts !== originalStarts) {
    payload.starts_at = newStarts === null ? 0 : newStarts
  }
  if (newEnds !== originalEnds) {
    payload.ends_at = newEnds === null ? 0 : newEnds
  }

  // targeting: do shallow compare by JSON
  if (JSON.stringify(form.targeting ?? {}) !== JSON.stringify(original.targeting ?? {})) {
    payload.targeting = form.targeting
  }

  return payload
}

async function handleSave() {
  // Frontend validation for targeting (to avoid ANNOUNCEMENT_INVALID_TARGET)
  const anyOf = form.targeting?.any_of ?? []
  if (anyOf.length > TARGETING_MAX_GROUPS) {
    appStore.showError(t('admin.announcements.errors.TARGETING_TOO_MANY_GROUPS'))
    return
  }
  for (const g of anyOf) {
    const allOf = g?.all_of ?? []
    if (allOf.length > TARGETING_MAX_CONDITIONS) {
      appStore.showError(t('admin.announcements.errors.TARGETING_TOO_MANY_CONDITIONS'))
      return
    }
  }
  if (scheduleError.value) {
    appStore.showError(scheduleError.value)
    return
  }
  if (contentTooLong.value) {
    appStore.showError(t('admin.announcements.errors.ANNOUNCEMENT_CONTENT_TOO_LONG'))
    return
  }

  if (willBroadcast.value && !(await confirmBroadcast())) return

  saving.value = true
  try {
    if (!editingAnnouncement.value) {
      const payload = buildCreatePayload()
      await adminAPI.announcements.create(payload)
      appStore.showSuccess(t('common.success'))
      showEditDialog.value = false
      await loadAnnouncements()
      return
    }

    const original = editingAnnouncement.value
    const payload = buildUpdatePayload(original)
    await adminAPI.announcements.update(original.id, payload)
    appStore.showSuccess(t('common.success'))
    showEditDialog.value = false
    editingAnnouncement.value = null
    await loadAnnouncements()
  } catch (error: any) {
    console.error('Failed to save announcement:', error)
    appStore.showError(announcementError(
      error,
      editingAnnouncement.value ? 'admin.announcements.failedToUpdate' : 'admin.announcements.failedToCreate',
    ))
  } finally {
    saving.value = false
  }
}

// ===== Delete =====
const showDeleteDialog = ref(false)
const deletingAnnouncement = ref<Announcement | null>(null)

function handleDelete(row: Announcement) {
  deletingAnnouncement.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingAnnouncement.value) return

  try {
    await adminAPI.announcements.delete(deletingAnnouncement.value.id)
    appStore.showSuccess(t('common.success'))
    showDeleteDialog.value = false
    deletingAnnouncement.value = null
    await loadAnnouncements()
  } catch (error: any) {
    console.error('Failed to delete announcement:', error)
    appStore.showError(announcementError(error, 'admin.announcements.failedToDelete'))
  }
}

// ===== Read status =====
const showReadStatusDialog = ref(false)
const readStatusAnnouncementId = ref<number | null>(null)
const previewAnnouncement = ref<Announcement | null>(null)

function openPreview(row: Announcement) {
  previewAnnouncement.value = row
}

function openReadStatus(row: Announcement) {
  readStatusAnnouncementId.value = row.id
  showReadStatusDialog.value = true
}

onMounted(async () => {
  await loadSubscriptionGroups()
  await loadAnnouncements()
})

onUnmounted(() => {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  currentController?.abort()
})
</script>
