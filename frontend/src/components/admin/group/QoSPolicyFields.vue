<template>
  <div class="space-y-4">
    <div class="flex items-start justify-between gap-4">
      <div class="min-w-0">
        <label :for="`${idPrefix}-qos-enabled`" class="input-label mb-0">
          {{ t("admin.groups.form.qosEnabled") }}
        </label>
        <p class="input-hint">{{ t("admin.groups.form.qosEnabledHint") }}</p>
      </div>
      <Toggle
        :id="`${idPrefix}-qos-enabled`"
        :model-value="enabled"
        :aria-label="t('admin.groups.form.qosEnabled')"
        @update:model-value="emit('update:enabled', $event)"
      />
    </div>

    <div v-if="enabled" class="space-y-4">
      <div>
        <label :for="`${idPrefix}-qos-metric`" class="input-label">
          {{ t("admin.groups.form.qosMetric") }}
        </label>
        <Select
          :id="`${idPrefix}-qos-metric`"
          :model-value="metric"
          :options="metricOptions"
          :aria-label="t('admin.groups.form.qosMetric')"
          :searchable="false"
          @update:model-value="emit('update:metric', normalizeGroupQoSMetric(asString($event)))"
        />
        <p class="input-hint">{{ t("admin.groups.form.qosMetricHint") }}</p>
      </div>

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between gap-3">
          <div class="min-w-0">
            <label class="input-label mb-0">
              {{ t("admin.groups.form.qosTiers") }}
            </label>
            <p class="input-hint">{{ t("admin.groups.form.qosTiersHint") }}</p>
          </div>
          <button
            type="button"
            class="inline-flex min-h-11 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-primary-600 transition-colors hover:bg-primary-50 hover:text-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:text-primary-400 dark:hover:bg-primary-900/20 dark:hover:text-primary-300"
            @click="addTier"
          >
            <Icon name="plus" size="sm" />
            {{ t("admin.groups.form.addQosTier") }}
          </button>
        </div>

        <p v-if="tiers.length === 0" class="input-hint">
          {{ t("admin.groups.form.qosTiersEmpty") }}
        </p>

        <div v-else class="space-y-3">
          <div
            v-for="(tier, tierIndex) in tiers"
            :key="tier.id"
            class="rounded-lg border border-gray-200 bg-gray-50/40 p-3 dark:border-dark-600 dark:bg-dark-800/40"
          >
            <div class="mb-3 flex items-center justify-between gap-3">
              <span class="text-sm font-medium text-gray-700 dark:text-dark-200">
                {{ t("admin.groups.form.qosTierLabel", { index: tierIndex + 1 }) }}
              </span>
              <button
                type="button"
                class="flex h-11 w-11 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 focus:outline-none focus:ring-2 focus:ring-red-500/30 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('admin.groups.form.removeQosTier')"
                :aria-label="t('admin.groups.form.removeQosTier')"
                @click="removeTier(tier.id)"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>

            <div class="grid gap-3 md:grid-cols-2">
              <div>
                <label :for="`${idPrefix}-${tier.id}-window`" class="input-label">
                  {{ t("admin.groups.form.qosWindow") }}
                </label>
                <Select
                  :id="`${idPrefix}-${tier.id}-window`"
                  :model-value="tier.window"
                  :options="windowOptions"
                  :aria-label="t('admin.groups.form.qosWindow')"
                  :searchable="false"
                  @update:model-value="updateTier(tier.id, 'window', asString($event))"
                />
              </div>
              <div>
                <label :for="`${idPrefix}-${tier.id}-threshold`" class="input-label">
                  {{ t("admin.groups.form.qosThreshold") }}
                </label>
                <input
                  :id="`${idPrefix}-${tier.id}-threshold`"
                  :value="tier.thresholdUsd"
                  type="number"
                  min="0"
                  step="0.01"
                  class="input"
                  :class="{ 'input-error': showValidation && !!errors.tiers[tier.id] }"
                  @input="updateTier(tier.id, 'thresholdUsd', ($event.target as HTMLInputElement).value)"
                />
              </div>
            </div>

            <p
              v-if="showValidation && errors.tiers[tier.id]"
              class="mt-1 text-xs text-red-600 dark:text-red-400"
              role="alert"
            >
              {{ t(`admin.groups.form.qosError.${errors.tiers[tier.id]}`) }}
            </p>

            <div class="mt-3 grid gap-3 md:grid-cols-2">
              <div v-if="supportsReasoningEffort">
                <label :for="`${idPrefix}-${tier.id}-effort`" class="input-label">
                  {{ t("admin.groups.form.qosMaxReasoningEffort") }}
                </label>
                <Select
                  :id="`${idPrefix}-${tier.id}-effort`"
                  :model-value="tier.maxReasoningEffort"
                  :options="reasoningEffortOptions"
                  :placeholder="t('admin.groups.form.qosNoChange')"
                  :aria-label="t('admin.groups.form.qosMaxReasoningEffort')"
                  :searchable="false"
                  clearable
                  @update:model-value="updateTier(tier.id, 'maxReasoningEffort', asString($event))"
                />
              </div>
              <div>
                <label :for="`${idPrefix}-${tier.id}-rpm`" class="input-label">
                  {{ t("admin.groups.form.qosRpmLimit") }}
                </label>
                <input
                  :id="`${idPrefix}-${tier.id}-rpm`"
                  :value="tier.rpmLimit"
                  type="number"
                  min="0"
                  step="1"
                  class="input"
                  :placeholder="t('admin.groups.form.qosNoChange')"
                  @input="updateTier(tier.id, 'rpmLimit', ($event.target as HTMLInputElement).value)"
                />
              </div>
            </div>

            <div class="mt-3 flex items-start justify-between gap-4">
              <div class="min-w-0">
                <label :for="`${idPrefix}-${tier.id}-block`" class="input-label mb-0">
                  {{ t("admin.groups.form.qosBlock") }}
                </label>
                <p class="input-hint">{{ t("admin.groups.form.qosBlockHint") }}</p>
              </div>
              <Toggle
                :id="`${idPrefix}-${tier.id}-block`"
                :model-value="tier.block"
                :aria-label="t('admin.groups.form.qosBlock')"
                @update:model-value="updateTier(tier.id, 'block', $event)"
              />
            </div>

            <div class="mt-3 border-t border-gray-200 pt-3 dark:border-dark-600">
              <div class="mb-2 flex items-center justify-between gap-3">
                <label class="input-label mb-0">
                  {{ t("admin.groups.form.qosModelMappings") }}
                </label>
                <button
                  type="button"
                  class="inline-flex min-h-11 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-primary-600 transition-colors hover:bg-primary-50 hover:text-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:text-primary-400 dark:hover:bg-primary-900/20 dark:hover:text-primary-300"
                  @click="addMapping(tier.id)"
                >
                  <Icon name="plus" size="sm" />
                  {{ t("admin.groups.form.addQosModelMapping") }}
                </button>
              </div>

              <div v-if="tier.modelMappings.length > 0" class="space-y-2">
                <div
                  v-for="mapping in tier.modelMappings"
                  :key="mapping.id"
                  class="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_auto] md:items-start"
                >
                  <div>
                    <input
                      :value="mapping.from"
                      type="text"
                      class="input"
                      :class="{ 'input-error': showValidation && !!errors.mappings[mapping.id]?.from }"
                      :placeholder="t('admin.groups.form.qosModelFromPlaceholder')"
                      :aria-label="t('admin.groups.form.qosModelFrom')"
                      @input="updateMapping(tier.id, mapping.id, 'from', ($event.target as HTMLInputElement).value)"
                    />
                    <p
                      v-if="showValidation && errors.mappings[mapping.id]?.from"
                      class="mt-1 text-xs text-red-600 dark:text-red-400"
                      role="alert"
                    >
                      {{ t(`admin.groups.form.qosError.${errors.mappings[mapping.id]?.from}`) }}
                    </p>
                  </div>

                  <div class="hidden pt-3 text-gray-400 md:block dark:text-dark-400">
                    <Icon name="arrowRight" size="sm" />
                  </div>

                  <div>
                    <input
                      :value="mapping.to"
                      type="text"
                      class="input"
                      :class="{ 'input-error': showValidation && !!errors.mappings[mapping.id]?.to }"
                      :placeholder="t('admin.groups.form.qosModelToPlaceholder')"
                      :aria-label="t('admin.groups.form.qosModelTo')"
                      @input="updateMapping(tier.id, mapping.id, 'to', ($event.target as HTMLInputElement).value)"
                    />
                    <p
                      v-if="showValidation && errors.mappings[mapping.id]?.to"
                      class="mt-1 text-xs text-red-600 dark:text-red-400"
                      role="alert"
                    >
                      {{ t(`admin.groups.form.qosError.${errors.mappings[mapping.id]?.to}`) }}
                    </p>
                  </div>

                  <button
                    type="button"
                    class="flex h-11 w-11 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 focus:outline-none focus:ring-2 focus:ring-red-500/30 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                    :title="t('admin.groups.form.removeQosModelMapping')"
                    :aria-label="t('admin.groups.form.removeQosModelMapping')"
                    @click="removeMapping(tier.id, mapping.id)"
                  >
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { GroupPlatform, GroupQoSMetric } from "@/types";
import Icon from "@/components/icons/Icon.vue";
import Select from "@/components/common/Select.vue";
import Toggle from "@/components/common/Toggle.vue";
import { reasoningEffortOptionsForPlatform, supportsReasoningEffortPolicyPlatform } from "@/views/admin/groupsReasoningEffort";
import {
  createGroupQoSModelMappingRow,
  createGroupQoSTierRow,
  groupQoSMetrics,
  groupQoSWindows,
  hasGroupQoSErrors,
  normalizeGroupQoSMetric,
  validateGroupQoSTiers,
  type GroupQoSTierRow,
} from "@/views/admin/groupsQoS";

const props = defineProps<{
  idPrefix: string;
  platform: GroupPlatform;
  enabled: boolean;
  metric: GroupQoSMetric;
  tiers: GroupQoSTierRow[];
}>();

const emit = defineEmits<{
  (event: "update:enabled", value: boolean): void;
  (event: "update:metric", value: GroupQoSMetric): void;
  (event: "update:tiers", value: GroupQoSTierRow[]): void;
}>();

const { t } = useI18n();
const showValidation = ref(false);

const supportsReasoningEffort = computed(() =>
  supportsReasoningEffortPolicyPlatform(props.platform),
);
const reasoningEffortOptions = computed(() =>
  reasoningEffortOptionsForPlatform(props.platform),
);
const windowOptions = computed(() =>
  groupQoSWindows.map((value) => ({
    value,
    label: t(`admin.groups.form.qosWindowOption.${value}`),
  })),
);
const metricOptions = computed(() =>
  groupQoSMetrics.map((value) => ({
    value,
    label: t(`admin.groups.form.qosMetricOption.${value}`),
  })),
);
const errors = computed(() => validateGroupQoSTiers(props.tiers, props.platform));

const asString = (value: string | number | boolean | null): string =>
  value == null ? "" : String(value);

const updateTier = (
  id: string,
  field: keyof GroupQoSTierRow,
  value: string | boolean,
) => {
  emit(
    "update:tiers",
    props.tiers.map((row) => (row.id === id ? { ...row, [field]: value } : row)),
  );
};

const addTier = () => {
  emit("update:tiers", [...props.tiers, createGroupQoSTierRow()]);
};

const removeTier = (id: string) => {
  emit(
    "update:tiers",
    props.tiers.filter((row) => row.id !== id),
  );
};

const addMapping = (tierID: string) => {
  emit(
    "update:tiers",
    props.tiers.map((row) =>
      row.id === tierID
        ? {
            ...row,
            modelMappings: [
              ...row.modelMappings,
              createGroupQoSModelMappingRow(),
            ],
          }
        : row,
    ),
  );
};

const removeMapping = (tierID: string, mappingID: string) => {
  emit(
    "update:tiers",
    props.tiers.map((row) =>
      row.id === tierID
        ? {
            ...row,
            modelMappings: row.modelMappings.filter((m) => m.id !== mappingID),
          }
        : row,
    ),
  );
};

const updateMapping = (
  tierID: string,
  mappingID: string,
  field: "from" | "to",
  value: string,
) => {
  emit(
    "update:tiers",
    props.tiers.map((row) =>
      row.id === tierID
        ? {
            ...row,
            modelMappings: row.modelMappings.map((m) =>
              m.id === mappingID ? { ...m, [field]: value } : m,
            ),
          }
        : row,
    ),
  );
};

const validate = (): boolean => {
  showValidation.value = true;
  // A disabled ladder never blocks a save — an admin may leave a half-built
  // ladder parked behind the toggle.
  if (!props.enabled) return true;
  return !hasGroupQoSErrors(errors.value);
};

const resetValidation = () => {
  showValidation.value = false;
};

defineExpose({ validate, resetValidation });
</script>
