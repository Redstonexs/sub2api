<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-64">
            <input
              v-model="params.search"
              type="text"
              :placeholder="t('common.search')"
              class="input"
              data-testid="announcement-archive-search"
              @input="debouncedReload"
            />
          </div>

          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input
              v-model="params.unread_only"
              type="checkbox"
              data-testid="announcement-archive-unread-only"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              @change="reload"
            />
            <span>{{ t('announcements.unreadOnly') }}</span>
          </label>

          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh')"
              @click="load"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="items" :loading="loading">
          <template #cell-title="{ row }">
            <button
              type="button"
              class="min-w-0 text-left"
              :data-testid="`announcement-archive-open-${row.id}`"
              @click="selected = row"
            >
              <div class="flex items-center gap-2">
                <span
                  v-if="!row.read_at"
                  class="h-2 w-2 shrink-0 rounded-full bg-primary-500"
                  :aria-label="t('announcements.unread')"
                ></span>
                <span class="truncate font-medium text-gray-900 dark:text-white">{{ row.title }}</span>
              </div>
              <p class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ excerpt(row.content) }}</p>
            </button>
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
          </template>

          <template #cell-read_at="{ row }">
            <span v-if="row.read_at" class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('announcements.read') }} · {{ formatDateTime(row.read_at) }}
            </span>
            <span v-else class="badge badge-primary">{{ t('announcements.unread') }}</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #empty>
            <EmptyState :title="t('empty.noData')" :description="t('announcements.description')" />
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

    <!-- Reuses the delivery popup in preview mode so an archived notice reads exactly
         as it did when it was live, and opening it does not mark it read. -->
    <AnnouncementPopup
      v-if="selected"
      :announcement="selected"
      preview
      @close="selected = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { announcementsAPI } from '@/api'
import { useTableLoader } from '@/composables/useTableLoader'
import { formatDateTime } from '@/utils/format'
import type { UserAnnouncement } from '@/types'
import type { Column } from '@/components/common/types'

import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const selected = ref<UserAnnouncement | null>(null)

const {
  items, loading, params, pagination,
  load, reload, debouncedReload, handlePageChange, handlePageSizeChange,
} = useTableLoader<UserAnnouncement, { unread_only: boolean; search: string }>({
  fetchFn: announcementsAPI.listArchive,
  initialParams: { unread_only: false, search: '' },
})

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('announcements.title') },
  { key: 'severity', label: t('admin.announcements.columns.severity') },
  { key: 'read_at', label: t('announcements.readAt') },
  { key: 'created_at', label: t('admin.announcements.columns.createdAt') },
])

/** First non-empty line of the body with Markdown markers stripped. */
function excerpt(content: string): string {
  const line = content
    .split('\n')
    .map((l) => l.replace(/^[#>\-*\s]+/, '').trim())
    .find((l) => l.length > 0) ?? ''
  return line.length > 140 ? `${line.slice(0, 140)}…` : line
}

onMounted(load)
</script>
