<template>
  <header class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
    <h3 class="pt-1 text-sm font-semibold text-gray-900 dark:text-white">
      {{ title }}
    </h3>
    <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
      <div v-if="groupsLoading" class="h-9 w-full animate-pulse rounded-lg bg-gray-100 dark:bg-dark-700 sm:w-48" />
      <Select
        v-else-if="groupOptions.length > 0"
        id="group-quota-selector"
        :model-value="selectedGroupId"
        :options="groupOptions"
        :placeholder="selectGroupLabel"
        :aria-label="selectGroupLabel"
        class="w-full sm:w-48"
        @update:model-value="emit('group-change', $event)"
      />
      <div class="flex overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600" role="group" :aria-label="title">
        <button
          data-testid="sort-5h"
          :aria-pressed="sortBy === '5h'"
          :class="[
            'px-2.5 py-2 text-xs font-medium transition-colors',
            sortBy === '5h'
              ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
              : 'bg-white text-gray-600 hover:bg-gray-50 dark:bg-dark-800 dark:text-gray-400 dark:hover:bg-dark-700'
          ]"
          @click="emit('sort-change', '5h')"
        >
          {{ sort5hLabel }}
        </button>
        <button
          data-testid="sort-7d"
          :aria-pressed="sortBy === '7d'"
          :class="[
            'border-l border-gray-200 px-2.5 py-2 text-xs font-medium transition-colors dark:border-dark-600',
            sortBy === '7d'
              ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
              : 'bg-white text-gray-600 hover:bg-gray-50 dark:bg-dark-800 dark:text-gray-400 dark:hover:bg-dark-700'
          ]"
          @click="emit('sort-change', '7d')"
        >
          {{ sort7dLabel }}
        </button>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import Select from '@/components/common/Select.vue'
import type { SelectOption } from '@/components/common/Select.vue'

defineProps<{
  groupsLoading: boolean
  groupOptions: SelectOption[]
  selectedGroupId: number | null
  sortBy: '5h' | '7d'
  title: string
  selectGroupLabel: string
  sort5hLabel: string
  sort7dLabel: string
}>()

const emit = defineEmits<{
  'group-change': [value: string | number | boolean | null]
  'sort-change': [value: '5h' | '7d']
}>()
</script>
