<template>
  <div class="space-y-6">
    <!-- Email disabled hint - show when email_verify_enabled is off -->
    <div v-if="!emailVerifyEnabled" class="card">
      <div class="p-6">
        <div class="flex items-start gap-3">
          <Icon
            name="mail"
            size="md"
            class="mt-0.5 flex-shrink-0 text-gray-400 dark:text-gray-500"
          />
          <div>
            <h3 class="font-medium text-gray-900 dark:text-white">
              {{ t("admin.settings.emailTabDisabledTitle") }}
            </h3>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.emailTabDisabledHint") }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- SMTP Settings - Only show when email verification is enabled -->
    <div v-if="emailVerifyEnabled" class="card">
      <div
        class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700"
      >
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t("admin.settings.smtp.title") }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.smtp.description") }}
          </p>
        </div>
        <button
          v-if="emailProvider === 'smtp'"
          type="button"
          @click="testSmtpConnection"
          :disabled="testingSmtp || loadFailed"
          class="btn btn-secondary btn-sm"
        >
          <svg
            v-if="testingSmtp"
            class="h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          {{
            testingSmtp
              ? t("admin.settings.smtp.testing")
              : t("admin.settings.smtp.testConnection")
          }}
        </button>
      </div>
      <div class="space-y-6 p-6">
        <!-- Sending method (provider) -->
        <div>
          <label
            class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {{ t("admin.settings.email.provider") }}
          </label>
          <select v-model="emailProvider" class="input">
            <option value="smtp">
              {{ t("admin.settings.email.providerSmtp") }}
            </option>
            <option value="resend">
              {{ t("admin.settings.email.providerResend") }}
            </option>
            <option value="cyberpanel">
              {{ t("admin.settings.email.providerCyberPanel") }}
            </option>
          </select>
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.email.providerHint") }}
          </p>
        </div>

        <!-- API provider credentials (resend / cyberpanel) -->
        <div
          v-if="emailProvider !== 'smtp'"
          class="grid grid-cols-1 gap-6 md:grid-cols-2"
        >
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.email.apiBaseUrl") }}
            </label>
            <input
              v-model="emailApiBaseUrl"
              type="text"
              class="input"
              :placeholder="
                emailProvider === 'cyberpanel'
                  ? t('admin.settings.email.apiBaseUrlPlaceholderCyberPanel')
                  : t('admin.settings.email.apiBaseUrlPlaceholderResend')
              "
            />
            <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.email.apiBaseUrlHint") }}
            </p>
          </div>
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.email.apiKey") }}
            </label>
            <input
              v-model="emailApiKey"
              type="password"
              class="input"
              autocomplete="new-password"
              autocapitalize="off"
              spellcheck="false"
              :placeholder="
                emailApiKeyConfigured
                  ? t('admin.settings.email.apiKeyConfiguredPlaceholder')
                  : t('admin.settings.email.apiKeyPlaceholder')
              "
            />
            <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{
                emailApiKeyConfigured
                  ? t("admin.settings.email.apiKeyConfiguredHint")
                  : t("admin.settings.email.apiKeyHint")
              }}
            </p>
          </div>
        </div>

        <!-- SMTP credentials -->
        <div
          v-if="emailProvider === 'smtp'"
          class="grid grid-cols-1 gap-6 md:grid-cols-2"
        >
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.smtp.host") }}
            </label>
            <input
              v-model="smtpHost"
              type="text"
              class="input"
              :placeholder="t('admin.settings.smtp.hostPlaceholder')"
            />
          </div>
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.smtp.port") }}
            </label>
            <input
              v-model.number="smtpPort"
              type="number"
              min="1"
              max="65535"
              class="input"
              :placeholder="t('admin.settings.smtp.portPlaceholder')"
            />
          </div>
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.smtp.username") }}
            </label>
            <input
              v-model="smtpUsername"
              type="text"
              class="input"
              :placeholder="t('admin.settings.smtp.usernamePlaceholder')"
            />
          </div>
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.smtp.password") }}
            </label>
            <input
              v-model="smtpPassword"
              type="password"
              class="input"
              autocomplete="new-password"
              autocapitalize="off"
              spellcheck="false"
              @keydown="smtpPasswordManuallyEdited = true"
              @paste="smtpPasswordManuallyEdited = true"
              :placeholder="
                smtpPasswordConfigured
                  ? t('admin.settings.smtp.passwordConfiguredPlaceholder')
                  : t('admin.settings.smtp.passwordPlaceholder')
              "
            />
            <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{
                smtpPasswordConfigured
                  ? t("admin.settings.smtp.passwordConfiguredHint")
                  : t("admin.settings.smtp.passwordHint")
              }}
            </p>
          </div>
        </div>

        <!-- Sender identity (shared by all providers) -->
        <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.smtp.fromEmail") }}
            </label>
            <input
              v-model="smtpFromEmail"
              type="email"
              class="input"
              :placeholder="t('admin.settings.smtp.fromEmailPlaceholder')"
            />
          </div>
          <div>
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.smtp.fromName") }}
            </label>
            <input
              v-model="smtpFromName"
              type="text"
              class="input"
              :placeholder="t('admin.settings.smtp.fromNamePlaceholder')"
            />
          </div>
        </div>

        <!-- Use TLS Toggle (SMTP only) -->
        <div
          v-if="emailProvider === 'smtp'"
          class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
        >
          <div>
            <label class="font-medium text-gray-900 dark:text-white">{{
              t("admin.settings.smtp.useTls")
            }}</label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.smtp.useTlsHint") }}
            </p>
          </div>
          <Toggle v-model="smtpUseTls" />
        </div>
      </div>
    </div>

    <!-- Send Test Email - Only show when email verification is enabled -->
    <div v-if="emailVerifyEnabled" class="card">
      <div
        class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
      >
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t("admin.settings.testEmail.title") }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.testEmail.description") }}
        </p>
      </div>
      <div class="p-6">
        <div class="flex items-end gap-4">
          <div class="flex-1">
            <label
              class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.testEmail.recipientEmail") }}
            </label>
            <input
              v-model="testEmailAddress"
              type="email"
              class="input"
              :placeholder="
                t('admin.settings.testEmail.recipientEmailPlaceholder')
              "
            />
          </div>
          <button
            type="button"
            @click="sendTestEmail"
            :disabled="
              sendingTestEmail || !testEmailAddress || loadFailed
            "
            class="btn btn-secondary"
          >
            <svg
              v-if="sendingTestEmail"
              class="h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              sendingTestEmail
                ? t("admin.settings.testEmail.sending")
                : t("admin.settings.testEmail.sendTestEmail")
            }}
          </button>
        </div>
      </div>
    </div>

    <!-- 订阅到期提醒 -->
    <div class="card">
      <div
        class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
      >
        <h3 class="text-base font-medium text-gray-900 dark:text-white">
          {{ t("admin.settings.subscriptionExpiryNotify.title") }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.subscriptionExpiryNotify.description") }}
        </p>
      </div>
      <div class="px-6 py-6">
        <div class="flex items-center justify-between gap-4">
          <div>
            <label
              class="mb-0 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.subscriptionExpiryNotify.enabled") }}
            </label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.subscriptionExpiryNotify.enabledHint") }}
            </p>
          </div>
          <Toggle v-model="subscriptionExpiryNotifyEnabled" />
        </div>
      </div>
    </div>

    <EmailTemplateEditor />

    <!-- Balance Low Notification -->
    <div class="card">
      <div
        class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
      >
        <h3 class="text-base font-medium text-gray-900 dark:text-white">
          {{ t("admin.settings.balanceNotify.title") }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.balanceNotify.description") }}
        </p>
      </div>
      <div class="px-6 py-6 space-y-4">
        <div class="flex items-center justify-between">
          <label
            class="mb-0 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >{{ t("admin.settings.balanceNotify.enabled") }}</label
          >
          <Toggle v-model="balanceLowNotifyEnabled" />
        </div>
        <div v-if="balanceLowNotifyEnabled">
          <label
            class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >{{ t("admin.settings.balanceNotify.threshold") }}</label
          >
          <div class="relative">
            <span
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
              >$</span
            >
            <input
              v-model.number="balanceLowNotifyThreshold"
              type="number"
              min="0"
              step="0.01"
              class="input pl-7"
            />
          </div>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.balanceNotify.thresholdHint") }}
          </p>
        </div>
        <div>
          <label
            class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >{{ t("admin.settings.balanceNotify.rechargeUrl") }}</label
          >
          <input
            v-model="balanceLowNotifyRechargeUrl"
            type="url"
            class="input"
            :placeholder="currentOrigin"
          />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.balanceNotify.rechargeUrlHint") }}
          </p>
        </div>
      </div>
    </div>

    <!-- Account Quota Notification -->
    <div class="card">
      <div
        class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
      >
        <h3 class="text-base font-medium text-gray-900 dark:text-white">
          {{ t("admin.settings.quotaNotify.title") }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.quotaNotify.description") }}
        </p>
      </div>
      <div class="px-6 py-6 space-y-4">
        <div class="flex items-center justify-between">
          <label
            class="mb-0 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >{{ t("admin.settings.quotaNotify.enabled") }}</label
          >
          <Toggle v-model="accountQuotaNotifyEnabled" />
        </div>
        <div v-if="accountQuotaNotifyEnabled">
          <label
            class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
            >{{ t("admin.settings.quotaNotify.emails") }}</label
          >
          <div class="space-y-2">
            <div
              v-for="(entry, index) in accountQuotaNotifyEmails ||
              []"
              :key="index"
              class="flex items-center gap-2"
            >
              <label
                class="relative inline-flex items-center cursor-pointer shrink-0"
              >
                <input
                  type="checkbox"
                  :checked="!entry.disabled"
                  @change="entry.disabled = !entry.disabled"
                  class="sr-only peer"
                />
                <div
                  class="w-9 h-5 bg-gray-200 peer-focus:outline-none rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:after:border-gray-500 peer-checked:bg-primary-600"
                ></div>
              </label>
              <input
                v-model="entry.email"
                type="email"
                class="input flex-1"
                :placeholder="
                  t('admin.settings.quotaNotify.emailPlaceholder')
                "
              />
              <button
                @click="accountQuotaNotifyEmails.splice(index, 1)"
                class="btn btn-secondary px-2"
                type="button"
              >
                <Icon name="x" size="xs" class="h-4 w-4" />
              </button>
            </div>
            <button
              @click="addQuotaNotifyEmail"
              class="btn btn-secondary btn-sm"
              type="button"
            >
              + {{ t("admin.settings.quotaNotify.addEmail") }}
            </button>
          </div>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.quotaNotify.emailsHint") }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api";
import type { NotifyEmailEntry } from "@/types";
import Toggle from "@/components/common/Toggle.vue";
import Icon from "@/components/icons/Icon.vue";
import EmailTemplateEditor from "@/views/admin/settings/EmailTemplateEditor.vue";
import { extractApiErrorMessage } from "@/utils/apiError";
import { useAppStore } from "@/stores";

const { t } = useI18n();
const appStore = useAppStore();

// v-model fields — the parent (SettingsView) owns the form state and global
// persistence; this child is presentation-only and mirrors edits upward.
const emailVerifyEnabled = defineModel<boolean>("emailVerifyEnabled", {
  required: true,
});
const emailProvider = defineModel<string>("emailProvider", { required: true });
const emailApiBaseUrl = defineModel<string>("emailApiBaseUrl", {
  required: true,
});
const emailApiKey = defineModel<string>("emailApiKey", { required: true });
const smtpHost = defineModel<string>("smtpHost", { required: true });
const smtpPort = defineModel<number>("smtpPort", { required: true });
const smtpUsername = defineModel<string>("smtpUsername", { required: true });
const smtpPassword = defineModel<string>("smtpPassword", { required: true });
const smtpFromEmail = defineModel<string>("smtpFromEmail", { required: true });
const smtpFromName = defineModel<string>("smtpFromName", { required: true });
const smtpUseTls = defineModel<boolean>("smtpUseTls", { required: true });
const subscriptionExpiryNotifyEnabled = defineModel<boolean>(
  "subscriptionExpiryNotifyEnabled",
  { required: true },
);
const balanceLowNotifyEnabled = defineModel<boolean>(
  "balanceLowNotifyEnabled",
  { required: true },
);
const balanceLowNotifyThreshold = defineModel<number>(
  "balanceLowNotifyThreshold",
  { required: true },
);
const balanceLowNotifyRechargeUrl = defineModel<string>(
  "balanceLowNotifyRechargeUrl",
  { required: true },
);
const accountQuotaNotifyEnabled = defineModel<boolean>(
  "accountQuotaNotifyEnabled",
  { required: true },
);
const accountQuotaNotifyEmails = defineModel<NotifyEmailEntry[]>(
  "accountQuotaNotifyEmails",
  { required: true },
);

// Read-only props (secrets are never round-tripped through v-model).
defineProps<{
  smtpPasswordConfigured: boolean;
  emailApiKeyConfigured: boolean;
  currentOrigin: string;
  loadFailed: boolean;
}>();

// Email-local test state and actions — owned by this child only.
const testingSmtp = ref(false);
const sendingTestEmail = ref(false);
const testEmailAddress = ref("");
const smtpPasswordManuallyEdited = ref(false);

// The parent clears form.smtp_password after a successful global save (and on
// hydration). When the smtpPassword prop is emptied by the parent, reset the
// local "manually edited" flag so a later test/send request does not reuse a
// stale typed password.
watch(smtpPassword, (value) => {
  if (value === "") {
    smtpPasswordManuallyEdited.value = false;
  }
});

async function testSmtpConnection() {
  testingSmtp.value = true;
  try {
    const smtpPasswordForTest = smtpPasswordManuallyEdited.value
      ? smtpPassword.value
      : "";
    const result = await adminAPI.settings.testSmtpConnection({
      smtp_host: smtpHost.value,
      smtp_port: smtpPort.value,
      smtp_username: smtpUsername.value,
      smtp_password: smtpPasswordForTest,
      smtp_use_tls: smtpUseTls.value,
    });
    // API returns { message: "..." } on success, errors are thrown as exceptions
    appStore.showSuccess(
      result.message || t("admin.settings.smtpConnectionSuccess"),
    );
  } catch (error: unknown) {
    appStore.showError(
      extractApiErrorMessage(error, t("admin.settings.failedToTestSmtp")),
    );
  } finally {
    testingSmtp.value = false;
  }
}

async function sendTestEmail() {
  if (!testEmailAddress.value) {
    appStore.showError(t("admin.settings.testEmail.enterRecipientHint"));
    return;
  }

  sendingTestEmail.value = true;
  try {
    const smtpPasswordForSend = smtpPasswordManuallyEdited.value
      ? smtpPassword.value
      : "";
    const result = await adminAPI.settings.sendTestEmail({
      email: testEmailAddress.value,
      smtp_host: smtpHost.value,
      smtp_port: smtpPort.value,
      smtp_username: smtpUsername.value,
      smtp_password: smtpPasswordForSend,
      smtp_from_email: smtpFromEmail.value,
      smtp_from_name: smtpFromName.value,
      smtp_use_tls: smtpUseTls.value,
    });
    // API returns { message: "..." } on success, errors are thrown as exceptions
    appStore.showSuccess(result.message || t("admin.settings.testEmailSent"));
  } catch (error: unknown) {
    appStore.showError(
      extractApiErrorMessage(error, t("admin.settings.failedToSendTestEmail")),
    );
  } finally {
    sendingTestEmail.value = false;
  }
}

// Quota notify email helpers
function addQuotaNotifyEmail() {
  if (!accountQuotaNotifyEmails.value) {
    accountQuotaNotifyEmails.value = [];
  }
  accountQuotaNotifyEmails.value.push({
    email: "",
    disabled: false,
    verified: true,
  });
}
</script>
