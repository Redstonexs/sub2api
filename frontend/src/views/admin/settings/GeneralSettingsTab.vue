<template>
  <!-- Site Settings -->
  <div class="card">
    <div
      class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
    >
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t("admin.settings.site.title") }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t("admin.settings.site.description") }}
      </p>
    </div>
    <div class="space-y-6 p-6">
      <!-- Backend Mode -->
      <div
        class="flex items-center justify-between rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
      >
        <div>
          <h3 class="text-sm font-medium text-gray-900 dark:text-white">
            {{ t("admin.settings.site.backendMode") }}
          </h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.site.backendModeDescription") }}
          </p>
        </div>
        <Toggle
          :model-value="backendModeEnabled"
          @update:model-value="emit('update:backendModeEnabled', $event)"
        />
      </div>

      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div>
          <label
            class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {{ t("admin.settings.site.siteName") }}
          </label>
          <input
            :value="siteName"
            type="text"
            class="input"
            data-testid="site-name-input"
            :placeholder="t('admin.settings.site.siteNamePlaceholder')"
            @input="emit('update:siteName', ($event.target as HTMLInputElement).value)"
          />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.site.siteNameHint") }}
          </p>
        </div>
        <div>
          <label
            class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {{ t("admin.settings.site.siteSubtitle") }}
          </label>
          <input
            :value="siteSubtitle"
            type="text"
            class="input"
            :placeholder="
              t('admin.settings.site.siteSubtitlePlaceholder')
            "
            @input="emit('update:siteSubtitle', ($event.target as HTMLInputElement).value)"
          />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.site.siteSubtitleHint") }}
          </p>
        </div>
      </div>

      <!-- API Base URL -->
      <div>
        <label
          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {{ t("admin.settings.site.apiBaseUrl") }}
        </label>
        <input
          :value="apiBaseUrl"
          type="text"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.site.apiBaseUrlPlaceholder')"
          @input="emit('update:apiBaseUrl', ($event.target as HTMLInputElement).value)"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.site.apiBaseUrlHint") }}
        </p>
      </div>

      <!-- Global Table Preferences -->
      <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
        <h3 class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t("admin.settings.site.tablePreferencesTitle") }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.site.tablePreferencesDescription") }}
        </p>
        <div class="mt-4 grid grid-cols-1 gap-6 md:grid-cols-2">
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.site.tableDefaultPageSize") }}
            </label>
            <input
              :value="tableDefaultPageSize"
              type="number"
              min="5"
              max="1000"
              step="1"
              class="input w-40"
              data-testid="table-default-page-size-input"
              @input="emit('update:tableDefaultPageSize', Number(($event.target as HTMLInputElement).value))"
            />
            <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.site.tableDefaultPageSizeHint") }}
            </p>
          </div>
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.site.tablePageSizeOptions") }}
            </label>
            <input
              :value="tablePageSizeOptionsInput"
              type="text"
              class="input font-mono text-sm"
              data-testid="page-size-options-input"
              :placeholder="
                t('admin.settings.site.tablePageSizeOptionsPlaceholder')
              "
              @input="emit('update:tablePageSizeOptionsInput', ($event.target as HTMLInputElement).value)"
            />
            <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.site.tablePageSizeOptionsHint") }}
            </p>
          </div>
        </div>
      </div>

      <!-- Custom Endpoints -->
      <div>
        <label
          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {{ t("admin.settings.site.customEndpoints.title") }}
        </label>
        <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.site.customEndpoints.description") }}
        </p>

        <div class="space-y-3">
          <div
            v-for="(ep, index) in localCustomEndpoints"
            :key="index"
            class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
          >
            <div class="mb-3 flex items-center justify-between">
              <span
                class="text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                {{
                  t("admin.settings.site.customEndpoints.itemLabel", {
                    n: index + 1,
                  })
                }}
              </span>
              <button
                type="button"
                class="rounded p-1 text-red-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                @click="removeEndpoint(index)"
              >
                <svg
                  class="h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </button>
            </div>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label
                  class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                >
                  {{ t("admin.settings.site.customEndpoints.name") }}
                </label>
                <input
                  :value="ep.name"
                  type="text"
                  class="input text-sm"
                  :placeholder="
                    t(
                      'admin.settings.site.customEndpoints.namePlaceholder',
                    )
                  "
                  @input="updateEndpointField(index, 'name', ($event.target as HTMLInputElement).value)"
                />
              </div>
              <div>
                <label
                  class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                >
                  {{
                    t("admin.settings.site.customEndpoints.endpointUrl")
                  }}
                </label>
                <input
                  :value="ep.endpoint"
                  type="url"
                  class="input font-mono text-sm"
                  :placeholder="
                    t(
                      'admin.settings.site.customEndpoints.endpointUrlPlaceholder',
                    )
                  "
                  @input="updateEndpointField(index, 'endpoint', ($event.target as HTMLInputElement).value)"
                />
              </div>
              <div class="sm:col-span-2">
                <label
                  class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                >
                  {{
                    t(
                      "admin.settings.site.customEndpoints.descriptionLabel",
                    )
                  }}
                </label>
                <input
                  :value="ep.description"
                  type="text"
                  class="input text-sm"
                  :placeholder="
                    t(
                      'admin.settings.site.customEndpoints.descriptionPlaceholder',
                    )
                  "
                  @input="updateEndpointField(index, 'description', ($event.target as HTMLInputElement).value)"
                />
              </div>
            </div>
          </div>
        </div>

        <button
          type="button"
          class="mt-3 flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 px-4 py-2.5 text-sm text-gray-500 transition-colors hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:text-gray-400 dark:hover:border-primary-500 dark:hover:text-primary-400"
          @click="addEndpoint"
        >
          <svg
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M12 4v16m8-8H4"
            />
          </svg>
          {{ t("admin.settings.site.customEndpoints.add") }}
        </button>
      </div>

      <!-- Contact Info -->
      <div>
        <label
          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {{ t("admin.settings.site.contactInfo") }}
        </label>
        <input
          :value="contactInfo"
          type="text"
          class="input"
          :placeholder="t('admin.settings.site.contactInfoPlaceholder')"
          @input="emit('update:contactInfo', ($event.target as HTMLInputElement).value)"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.site.contactInfoHint") }}
        </p>
      </div>

      <!-- Doc URL -->
      <div>
        <label
          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {{ t("admin.settings.site.docUrl") }}
        </label>
        <input
          :value="docUrl"
          type="url"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.site.docUrlPlaceholder')"
          @input="emit('update:docUrl', ($event.target as HTMLInputElement).value)"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.site.docUrlHint") }}
        </p>
      </div>

      <!-- Site Logo Upload -->
      <div>
        <label
          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {{ t("admin.settings.site.siteLogo") }}
        </label>
        <ImageUpload
          :model-value="siteLogo"
          mode="image"
          :upload-label="t('admin.settings.site.uploadImage')"
          :remove-label="t('admin.settings.site.remove')"
          :hint="t('admin.settings.site.logoHint')"
          :max-size="300 * 1024"
          @update:model-value="emit('update:siteLogo', $event)"
        />
      </div>

      <!-- Home Content -->
      <div>
        <label
          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {{ t("admin.settings.site.homeContent") }}
        </label>
        <textarea
          :value="homeContent"
          rows="6"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.site.homeContentPlaceholder')"
          @input="emit('update:homeContent', ($event.target as HTMLTextAreaElement).value)"
        ></textarea>
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.site.homeContentHint") }}
        </p>
        <!-- iframe CSP Warning -->
        <p class="mt-2 text-xs text-amber-600 dark:text-amber-400">
          {{ t("admin.settings.site.homeContentIframeWarning") }}
        </p>
      </div>

      <!-- Compact Home Page -->
      <div class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700">
        <div>
          <label class="font-medium text-gray-900 dark:text-white">{{
            t("admin.settings.site.compactHome")
          }}</label>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.site.compactHomeHint") }}
          </p>
        </div>
        <Toggle
          :model-value="compactHomeEnabled"
          data-testid="compact-home-toggle"
          @update:model-value="emit('update:compactHomeEnabled', $event)"
        />
      </div>

      <!-- Hide CCS Import Button -->
      <div
        class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
      >
        <div>
          <label class="font-medium text-gray-900 dark:text-white">{{
            t("admin.settings.site.hideCcsImportButton")
          }}</label>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.site.hideCcsImportButtonHint") }}
          </p>
        </div>
        <Toggle
          :model-value="hideCcsImportButton"
          @update:model-value="emit('update:hideCcsImportButton', $event)"
        />
      </div>
    </div>
  </div>

  <!-- Custom Menu Items -->
  <div class="card">
    <div
      class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
    >
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t("admin.settings.customMenu.title") }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t("admin.settings.customMenu.description") }}
      </p>
    </div>
    <div class="space-y-4 p-6">
      <!-- Existing menu items -->
      <div
        v-for="(item, index) in localCustomMenuItems"
        :key="item.id || index"
        class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
      >
        <div class="mb-3 flex items-center justify-between">
          <span
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {{
              t("admin.settings.customMenu.itemLabel", { n: index + 1 })
            }}
          </span>
          <div class="flex items-center gap-2">
            <!-- Move up -->
            <button
              v-if="index > 0"
              type="button"
              class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
              :title="t('admin.settings.customMenu.moveUp')"
              @click="moveMenuItem(index, -1)"
            >
              <svg
                class="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M5 15l7-7 7 7"
                />
              </svg>
            </button>
            <!-- Move down -->
            <button
              v-if="index < localCustomMenuItems.length - 1"
              type="button"
              class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
              :title="t('admin.settings.customMenu.moveDown')"
              @click="moveMenuItem(index, 1)"
            >
              <svg
                class="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M19 9l-7 7-7-7"
                />
              </svg>
            </button>
            <!-- Delete -->
            <button
              type="button"
              class="rounded p-1 text-red-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
              :title="t('admin.settings.customMenu.remove')"
              @click="removeMenuItem(index)"
            >
              <svg
                class="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <!-- Label -->
          <div>
            <label
              class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
            >
              {{ t("admin.settings.customMenu.name") }}
            </label>
            <input
              :value="item.label"
              type="text"
              class="input text-sm"
              :placeholder="
                t('admin.settings.customMenu.namePlaceholder')
              "
              @input="updateMenuItemField(index, 'label', ($event.target as HTMLInputElement).value)"
            />
          </div>

          <!-- Visibility -->
          <div>
            <label
              class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
            >
              {{ t("admin.settings.customMenu.visibility") }}
            </label>
            <select
              :value="item.visibility"
              class="input text-sm"
              @change="updateMenuItemField(index, 'visibility', ($event.target as HTMLSelectElement).value)"
            >
              <option value="user">
                {{ t("admin.settings.customMenu.visibilityUser") }}
              </option>
              <option value="admin">
                {{ t("admin.settings.customMenu.visibilityAdmin") }}
              </option>
            </select>
          </div>

          <!-- URL (full width) -->
          <div class="sm:col-span-2 lg:col-span-3">
            <label
              class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
            >
              {{ t("admin.settings.customMenu.url") }}
            </label>
            <input
              :value="item.url"
              type="url"
              class="input font-mono text-sm"
              :placeholder="
                t('admin.settings.customMenu.urlPlaceholder')
              "
              @input="updateMenuItemField(index, 'url', ($event.target as HTMLInputElement).value)"
            />
          </div>

          <!-- SVG Icon (full width) -->
          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
            >
              {{ t("admin.settings.customMenu.iconSvg") }}
            </label>
            <ImageUpload
              :model-value="item.icon_svg"
              mode="svg"
              size="sm"
              :upload-label="t('admin.settings.customMenu.uploadSvg')"
              :remove-label="t('admin.settings.customMenu.removeSvg')"
              @update:model-value="(v: string) => updateMenuItemField(index, 'icon_svg', v)"
            />
          </div>
        </div>
      </div>

      <!-- Add button -->
      <button
        type="button"
        class="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 py-3 text-sm text-gray-500 transition-colors hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:text-gray-400 dark:hover:border-primary-500 dark:hover:text-primary-400"
        @click="addMenuItem"
      >
        <svg
          class="h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M12 4v16m8-8H4"
          />
        </svg>
        {{ t("admin.settings.customMenu.add") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { CustomEndpoint, CustomMenuItem } from "@/types";
import Toggle from "@/components/common/Toggle.vue";
import ImageUpload from "@/components/common/ImageUpload.vue";

const props = defineProps<{
  backendModeEnabled: boolean;
  siteName: string;
  siteSubtitle: string;
  apiBaseUrl: string;
  tableDefaultPageSize: number;
  tablePageSizeOptionsInput: string;
  customEndpoints: CustomEndpoint[];
  contactInfo: string;
  docUrl: string;
  siteLogo: string;
  homeContent: string;
  compactHomeEnabled: boolean;
  hideCcsImportButton: boolean;
  customMenuItems: CustomMenuItem[];
}>();

const emit = defineEmits<{
  (e: "update:backendModeEnabled", value: boolean): void;
  (e: "update:siteName", value: string): void;
  (e: "update:siteSubtitle", value: string): void;
  (e: "update:apiBaseUrl", value: string): void;
  (e: "update:tableDefaultPageSize", value: number): void;
  (e: "update:tablePageSizeOptionsInput", value: string): void;
  (e: "update:customEndpoints", value: CustomEndpoint[]): void;
  (e: "update:contactInfo", value: string): void;
  (e: "update:docUrl", value: string): void;
  (e: "update:siteLogo", value: string): void;
  (e: "update:homeContent", value: string): void;
  (e: "update:compactHomeEnabled", value: boolean): void;
  (e: "update:hideCcsImportButton", value: boolean): void;
  (e: "update:customMenuItems", value: CustomMenuItem[]): void;
}>();

const { t } = useI18n();

// Presentation-only: keep local deep copies of the array props and emit
// complete arrays upward so the parent keeps ownership of the form state.
const localCustomEndpoints = ref<CustomEndpoint[]>(
  props.customEndpoints.map((ep) => ({ ...ep })),
);
const localCustomMenuItems = ref<CustomMenuItem[]>(
  props.customMenuItems.map((item) => ({ ...item })),
);

watch(
  () => props.customEndpoints,
  (endpoints) => {
    localCustomEndpoints.value = endpoints.map((ep) => ({ ...ep }));
  },
);

watch(
  () => props.customMenuItems,
  (items) => {
    localCustomMenuItems.value = items.map((item) => ({ ...item }));
  },
);

function emitCustomEndpoints(): void {
  emit(
    "update:customEndpoints",
    localCustomEndpoints.value.map((ep) => ({ ...ep })),
  );
}

function emitCustomMenuItems(): void {
  emit(
    "update:customMenuItems",
    localCustomMenuItems.value.map((item) => ({ ...item })),
  );
}

function updateEndpointField(
  index: number,
  field: "name" | "endpoint" | "description",
  value: string,
): void {
  const endpoint = localCustomEndpoints.value[index];
  if (!endpoint) {
    return;
  }
  endpoint[field] = value;
  emitCustomEndpoints();
}

// Custom endpoint management
function addEndpoint(): void {
  localCustomEndpoints.value.push({ name: "", endpoint: "", description: "" });
  emitCustomEndpoints();
}

function removeEndpoint(index: number): void {
  localCustomEndpoints.value.splice(index, 1);
  emitCustomEndpoints();
}

function updateMenuItemField(
  index: number,
  field: "label" | "icon_svg" | "url" | "visibility",
  value: string,
): void {
  const item = localCustomMenuItems.value[index];
  if (!item) {
    return;
  }
  if (field === "visibility") {
    item.visibility = value as "user" | "admin";
  } else {
    item[field] = value;
  }
  emitCustomMenuItems();
}

// Custom menu item management
function addMenuItem(): void {
  localCustomMenuItems.value.push({
    id: "",
    label: "",
    icon_svg: "",
    url: "",
    visibility: "user",
    sort_order: localCustomMenuItems.value.length,
  });
  emitCustomMenuItems();
}

function removeMenuItem(index: number): void {
  localCustomMenuItems.value.splice(index, 1);
  // Re-index sort_order
  localCustomMenuItems.value.forEach((item, i) => {
    item.sort_order = i;
  });
  emitCustomMenuItems();
}

function moveMenuItem(index: number, direction: -1 | 1): void {
  const targetIndex = index + direction;
  if (targetIndex < 0 || targetIndex >= localCustomMenuItems.value.length) {
    return;
  }
  const items = localCustomMenuItems.value;
  const temp = items[index];
  items[index] = items[targetIndex];
  items[targetIndex] = temp;
  // Re-index sort_order
  items.forEach((item, i) => {
    item.sort_order = i;
  });
  emitCustomMenuItems();
}
</script>