// Fork-local locale overlay (Redstonexs/sub2api).
// Deep-merged over the upstream locale modules in ./index.ts — keys here are
// either fork-only features (gateway error messages, email provider, announcement
// email broadcast, ops system-log panel, onboarding palette) or fork overrides of
// upstream values. Keep en/fork.ts and zh/fork.ts key-for-key in sync
// (enforced by ../__tests__/localeIntegrity.spec.ts).
export default {
  common: {
    creating: "Creating...",
    sending: "Sending...",
    apply: "Apply",
    clear: "Clear",
    required: "is required",
    tryAgain: "Please try again",
    tableOfContents: "Contents",
    failed: "Failed",
  },
  auth: {
    agreementPrompt: {
      agreeRead: "I have read and agree to",
      mustAgree: "You must agree to the latest terms before logging in.",
      disabledHint: "Until you agree, password entry and quick login stay disabled.",
      viewTerms: "View Terms",
      updateTitle: "Terms Update Notice",
      updateBody: "Our terms of service were updated on {date}. Please read and agree to the following terms before continuing to use the service.",
      recently: "recently",
      relatedDocs: "Related Documents",
      reject: "Reject",
      agreeContinue: "Agree and Continue",
      warnDisabledLogin: "Until you agree to the latest terms, you can't enter credentials or use quick login.",
      warnAgreeBeforeLogin: "Please read and agree to the latest terms before logging in.",
      warnDisabledRegister: "Until you agree to the latest terms, you can't register or use quick login.",
      warnAgreeBeforeRegister: "Please read and agree to the latest terms before registering.",
    },
  },
  admin: {
    users: {
      passwordCopied: "Password copied",
    },
    groups: {
      selectedLabel: "Selected",
      invertSelection: "Invert",
      accountFilterControl: "Account Filter Control",
      oauthOnlyLabel: "Allow OAuth accounts only",
      oauthOnlyEnabledHint: "Enabled — excludes API Key accounts",
      filterNotEnabled: "Disabled",
      privacyOnlyLabel: "Allow only accounts with privacy protection set",
      privacyOnlyEnabledHint: "Enabled — accounts without Privacy set are excluded",
      accountCountLabel: "{count} accounts",
      modelRouting: {
        claudeMaxSimulation: {
          title: "Claude Max usage simulation",
          tooltip: "When enabled, Claude models whose upstream response omits cache-write usage are deterministically mapped to a small input amount plus 1-hour cache creation while preserving the total token count.",
          enabled: "Enabled (simulate 1h cache)",
          disabled: "Disabled",
          hint: "Only adjusts token classification in usage billing logs; no per-request mapping state is persisted.",
        },
      },
    },
    channels: {
      noGroupsSelected: "No groups selected for {platform}. Please select at least one group.",
      emptyModelsInPricing: "{platform} has model pricing entries with no models. Add models or remove those entries.",
    },
    accounts: {
      bulkActions: {
        deleteConfirm: "Delete {count} selected account(s)? This action cannot be undone.",
        resetStatusConfirm: "Reset the error status of {count} selected account(s)?",
        refreshTokenConfirm: "Refresh the token of {count} selected account(s)?",
      },
      fromModel: "Request model",
      toModel: "Actual model",
      messages: {
        accountCreated: "Account created successfully",
      },
      oauth: {
        openai: {
          mobileRefreshTokenAuth: "Manual Mobile RT Input",
          accessTokenAuth: "Manual AT Input",
        },
      },
      gemini: {
        oauthType: {
          personalQuota: "Personal account — enjoys Google One subscription quota",
          recommendedPersonal: "Recommended for individuals",
          noGcp: "No GCP required",
          enterpriseGcp: "Enterprise-grade — requires a GCP project",
          enterpriseGcpHint: "Requires an activated GCP project with a bound credit card",
          enterpriseUsers: "Enterprise users",
          highConcurrency: "High concurrency",
          showAdvancedOAuth: "Show advanced options (self-hosted OAuth Client)",
          hideAdvancedOAuth: "Hide advanced options (self-hosted OAuth Client)",
          changeRegion: "Change region",
        },
      },
    },
    announcements: {
      notifyModeLabels: {
        email: "Email",
      },
      emailUnsubscribed: "Unsubscribed",
      emailSubscribed: "Subscribed",
      emailUnsubscribedLabel: "Email Subscription",
      form: {
        notifyModeHint: "Popup mode shows a popup to users. Email mode also sends the announcement by email to every targeted user.",
      },
    },
    ops: {
      systemLog: {
        title: "System Logs",
        description: "Sorted by newest first; supports filtering, search, and cleanup by condition.",
        queue: "Queue",
        written: "Written",
        dropped: "Dropped",
        failed: "Failed",
        runtimeConfigTitle: "Runtime Log Config (applied live)",
        loading: "Loading...",
        level: "Level",
        stacktraceThreshold: "Stacktrace Threshold",
        samplingInitial: "Sampling Initial",
        samplingThereafter: "Sampling Subsequent",
        retentionDays: "Retention (days)",
        caller: "Caller",
        sampling: "Sampling",
        saving: "Saving...",
        saveAndApply: "Save & Apply",
        rollbackDefault: "Restore Defaults",
        lastWriteError: "Last write error: ",
        timeRange: "Time Range",
        startTimeOptional: "Start Time (optional)",
        endTimeOptional: "End Time (optional)",
        component: "Component",
        componentPlaceholder: "e.g. http.access",
        keyId: "KEY ID",
        platform: "Platform",
        model: "Model",
        keyword: "Keyword",
        keywordPlaceholder: "Message / request_id",
        query: "Search",
        reset: "Reset",
        cleanupCurrent: "Clean by current filter",
        refreshHealth: "Refresh health",
        empty: "No system logs",
        colTime: "Time",
        colDetail: "Log Details",
        all: "All",
        loadFailed: "Failed to load system logs",
        configApplied: "Runtime log config applied",
        saveConfigFailed: "Failed to save log config",
        confirmRollback: "Roll back to the startup config (env/yaml) and apply immediately?",
        rolledBack: "Rolled back to the startup log config",
        rollbackFailed: "Failed to roll back log config",
        confirmCleanup: "Clean up system logs matching the current filter? This action cannot be undone.",
        cleanupDone: "Cleanup complete, deleted {count} logs",
        cleanupFailed: "Failed to clean up system logs",
      },
      runtime: {
        metricThresholds: "Metric Thresholds",
        metricThresholdsHint: "Configure alert thresholds for metrics, values exceeding thresholds will be displayed in red",
        slaMinPercent: "SLA Minimum Percentage",
        slaMinPercentHint: "SLA below this value will be displayed in red (default: 99.5%)",
        ttftP99MaxMs: "TTFT P99 Maximum (ms)",
        ttftP99MaxMsHint: "TTFT P99 above this value will be displayed in red (default: 500ms)",
        requestErrorRateMaxPercent: "Request Error Rate Maximum (%)",
        requestErrorRateMaxPercentHint: "Request error rate above this value will be displayed in red (default: 5%)",
        upstreamErrorRateMaxPercent: "Upstream Error Rate Maximum (%)",
        upstreamErrorRateMaxPercentHint: "Upstream error rate above this value will be displayed in red (default: 5%)",
      },
    },
    settings: {
      email: {
        provider: "Sending Method",
        providerHint: "How outbound email is delivered. API methods (Resend / CyberPanel) send over HTTPS and bypass cloud SMTP port blocks (e.g. DigitalOcean blocking 25/465/587).",
        providerSmtp: "SMTP",
        providerResend: "Resend API",
        providerCyberPanel: "CyberPanel API",
        apiBaseUrl: "API Base URL",
        apiBaseUrlHint: "CyberPanel: your mail server base URL. Resend: optional, defaults to https://api.resend.com.",
        apiBaseUrlPlaceholderResend: "https://api.resend.com (optional)",
        apiBaseUrlPlaceholderCyberPanel: "https://mail.yourdomain.com",
        apiKey: "API Key",
        apiKeyPlaceholder: "Bearer token (e.g. re_... or sk_live_...)",
        apiKeyHint: "Sent as an Authorization: Bearer header.",
        apiKeyConfiguredPlaceholder: "********",
        apiKeyConfiguredHint: "API key configured. Leave empty to keep the current value.",
      },
      gatewayErrorMessages: {
        title: "Gateway Error Messages",
        description: "Customize the error messages shown to users when the gateway returns specific HTTP status codes.",
        codeHeader: "Status code",
        messageHeader: "Message",
        codePlaceholder: "e.g. 429",
        messagePlaceholder: "Message shown to users",
        addRow: "Add message",
        remove: "Remove",
        empty: "No custom messages yet. Use \"Add message\" to override an HTTP status code.",
        hint: "Map an HTTP status code (e.g. 429 or 502) to the message users will see. Add only the codes you want to override; the rest keep their defaults.",
        invalidForm: "Please fix the highlighted gateway error messages before saving.",
        errors: {
          codeEmpty: "Enter an HTTP status code.",
          codeInvalid: "Enter a valid 3-digit HTTP status code (100–599).",
          codeDuplicate: "This status code is already configured.",
          messageEmpty: "Enter a message for this status code.",
        },
      },
      gatewayForwarding: {
        claudeOAuthSystemPromptBlocksPlaceholder: "Leave empty to use the built-in 3 blocks. Supports an array or {'{'}\"blocks\": [...]{'}'}.",
      },
      // Upstream v0.1.151 added the Fast/Flex user-scope UI reading
      // openaiFastPolicy.userIds* but shipped the keys under betaPolicy;
      // overlay them here until upstream relocates them.
      openaiFastPolicy: {
        userIds: "Specific user IDs",
        userIdsHint:
          "Leave empty to apply to all Sub2API users. Specified users match requests from their API keys and take precedence over global rules.",
        userIdPlaceholder: "e.g., 1001",
        addUserId: "Add user ID",
        removeUserId: "Remove user ID",
      },
    },
  },
  onboarding: {
    admin: {
      welcome: {
        description: "<div style=\"line-height: 1.8;\"><p style=\"margin-bottom: 16px;\">Sub2API is a powerful AI service gateway platform that helps you easily manage and distribute AI services.</p><p style=\"margin-bottom: 12px;\"><b>🎯 Core Features:</b></p><ul style=\"margin-left: 20px; margin-bottom: 16px;\"><li>📦 <b>Group Management</b> - Create service tiers (VIP, Free Trial, etc.)</li><li>🔗 <b>Account Pool</b> - Connect multiple upstream AI service accounts</li><li>🔑 <b>Key Distribution</b> - Generate independent API Keys for users</li><li>💰 <b>Billing Control</b> - Flexible rate and quota management</li></ul><p style=\"color: #5D9A51; font-weight: 600;\">Let's complete the initial setup in 3 minutes →</p></div>",
      },
      groupManage: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\"><b>What is a Group?</b></p><p style=\"margin-bottom: 12px;\">Groups are the core concept of Sub2API, like a \"service package\":</p><ul style=\"margin-left: 20px; margin-bottom: 12px; font-size: 13px;\"><li>🎯 Each group can contain multiple upstream accounts</li><li>💰 Each group has independent billing multiplier</li><li>👥 Can be set as public or exclusive</li></ul><p style=\"margin-top: 12px; padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Example:</b> You can create \"VIP Premium\" (high rate) and \"Free Trial\" (low rate) groups</p><p style=\"margin-top: 16px; color: #5D9A51; font-weight: 600;\">👉 Click \"Group Management\" on the left sidebar</p></div>",
      },
      createGroup: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Let's create your first group.</p><p style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📝 Tip:</b> Recommend creating a test group first to familiarize yourself with the process</p><p style=\"color: #5D9A51; font-weight: 600;\">👉 Click the \"Create Group\" button</p></div>",
      },
      groupName: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Give your group an easy-to-identify name.</p><div style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>💡 Naming Suggestions:</b><ul style=\"margin: 8px 0 0 16px;\"><li>\"Test Group\" - For testing</li><li>\"VIP Premium\" - High-quality service</li><li>\"Free Trial\" - Trial version</li></ul></div><p style=\"font-size: 13px; color: #827B6C;\">Click \"Next\" when done</p></div>",
      },
      groupPlatform: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Choose the AI platform this group supports.</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📌 Platform Guide:</b><ul style=\"margin: 8px 0 0 16px;\"><li><b>Anthropic</b> - Claude models</li><li><b>OpenAI</b> - GPT models</li><li><b>Google</b> - Gemini models</li></ul></div><p style=\"font-size: 13px; color: #827B6C;\">One group can only have one platform</p></div>",
      },
      groupMultiplier: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Set the billing multiplier to control user charges.</p><div style=\"padding: 8px 12px; background: #FFF0DB; border-left: 3px solid #E2A846; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚙️ Billing Rules:</b><ul style=\"margin: 8px 0 0 16px;\"><li><b>1.0</b> - Original price (cost price)</li><li><b>1.5</b> - User consumes $1, charged $1.5</li><li><b>2.0</b> - User consumes $1, charged $2</li><li><b>0.8</b> - Subsidy mode (loss-making)</li></ul></div><p style=\"font-size: 13px; color: #827B6C;\">Recommend setting test group to 1.0</p></div>",
      },
      groupExclusive: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Control group visibility and access permissions.</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>🔐 Permission Guide:</b><ul style=\"margin: 8px 0 0 16px;\"><li><b>Off</b> - Public group, visible to all users</li><li><b>On</b> - Exclusive group, only for specified users</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Use Cases:</b> VIP exclusive, internal testing, special customers</p></div>",
      },
      groupSubmit: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Confirm the information and click create to save the group.</p><p style=\"padding: 8px 12px; background: #FFF0DB; border-left: 3px solid #E2A846; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚠️ Note:</b> Platform type cannot be changed after creation, but other settings can be edited anytime</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>📌 Next Step:</b> After creation, we'll add upstream accounts to this group</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 Click \"Create\" button</p></div>",
      },
      accountManage: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\"><b>Great! Group created successfully 🎉</b></p><p style=\"margin-bottom: 12px;\">Now add upstream AI service accounts to enable actual service delivery.</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>🔑 Account Purpose:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Connect to upstream AI services (Claude, GPT, etc.)</li><li>One group can contain multiple accounts (load balancing)</li><li>Supports OAuth and Session Key methods</li></ul></div><p style=\"margin-top: 16px; color: #5D9A51; font-weight: 600;\">👉 Click \"Account Management\" on the left sidebar</p></div>",
      },
      createAccount: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Click the button to start adding your first upstream account.</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Tip:</b> Recommend using OAuth method - more secure and no manual key extraction needed</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 Click \"Add Account\" button</p></div>",
      },
      accountName: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Set an easy-to-identify name for the account.</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Naming Suggestions:</b> \"Claude Main\", \"GPT Backup 1\", \"Test Account\", etc.</p></div>",
      },
      accountPlatform: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Choose the service provider platform for this account.</p><p style=\"padding: 8px 12px; background: #FFF0DB; border-left: 3px solid #E2A846; border-radius: 4px; font-size: 13px;\"><b>⚠️ Important:</b> Platform must match the group you just created</p></div>",
      },
      accountType: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Choose the account authorization method.</p><div style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>✅ Recommended: OAuth Method</b><ul style=\"margin: 8px 0 0 16px;\"><li>No manual key extraction needed</li><li>More secure with auto-refresh support</li><li>Works with Claude Code, ChatGPT OAuth</li></ul></div><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px;\"><b>📌 Session Key Method</b><ul style=\"margin: 8px 0 0 16px;\"><li>Requires manual extraction from browser</li><li>May need periodic updates</li><li>For platforms without OAuth support</li></ul></div></div>",
      },
      accountPriority: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Set the account call priority.</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📊 Priority Rules:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Lower number = higher priority</li><li>System uses low-value accounts first</li><li>Same priority = random selection</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Use Case:</b> Set main account to lower value, backup accounts to higher value</p></div>",
      },
      accountGroups: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\"><b>Key Step!</b> Assign the account to the group you just created.</p><div style=\"padding: 8px 12px; background: #FAE4E1; border-left: 3px solid #D0685B; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚠️ Important Reminder:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Must select at least one group</li><li>Unassigned accounts cannot be used</li><li>One account can be assigned to multiple groups</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Tip:</b> Select the test group you just created</p></div>",
      },
      accountSubmit: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Confirm the information and click save.</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📌 OAuth Flow:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Will redirect to service provider page after clicking save</li><li>Complete login and authorization on provider page</li><li>Auto-return after successful authorization</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>📌 Next Step:</b> After adding account, we'll create an API key</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 Click \"Save\" button</p></div>",
      },
      keyManage: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\"><b>Congratulations! Account setup complete 🎉</b></p><p style=\"margin-bottom: 12px;\">Final step: generate an API Key to test if the service works properly.</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>🔑 API Key Purpose:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Credential for calling AI services</li><li>Each key is bound to one group</li><li>Can set quota and expiration</li><li>Supports independent usage statistics</li></ul></div><p style=\"margin-top: 16px; color: #5D9A51; font-weight: 600;\">👉 Click \"API Keys\" on the left sidebar</p></div>",
      },
      createKey: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Click the button to create your first API Key.</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Tip:</b> Copy and save immediately after creation - key is only shown once</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 Click \"Create Key\" button</p></div>",
      },
      keyName: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Set an easy-to-manage name for the key.</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Naming Suggestions:</b> \"Test Key\", \"Production\", \"Mobile\", etc.</p></div>",
      },
      keyGroup: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Select the group you just configured.</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📌 Group Determines:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Which accounts this key can use</li><li>What billing multiplier applies</li><li>Whether it's an exclusive key</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Tip:</b> Select the test group you just created</p></div>",
      },
      keySubmit: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">System will generate a complete API Key after clicking create.</p><div style=\"padding: 8px 12px; background: #FAE4E1; border-left: 3px solid #D0685B; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚠️ Important Reminder:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Key is only shown once, copy immediately</li><li>Need to regenerate if lost</li><li>Keep it safe, don't share with others</li></ul></div><div style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>🚀 Next Steps:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Copy the generated sk-xxx key</li><li>Use in any OpenAI-compatible client</li><li>Start experiencing AI services!</li></ul></div><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 Click \"Create\" button</p></div>",
      },
    },
    user: {
      welcome: {
        description: "<div style=\"line-height: 1.8;\"><p style=\"margin-bottom: 16px;\">Hello! Welcome to the Sub2API AI service platform.</p><p style=\"margin-bottom: 12px;\"><b>🎯 Quick Start:</b></p><ul style=\"margin-left: 20px; margin-bottom: 16px;\"><li>🔑 Create API Key</li><li>📋 Copy key to your application</li><li>🚀 Start using AI services</li></ul><p style=\"color: #5D9A51; font-weight: 600;\">Just 1 minute, let's get started →</p></div>",
      },
      keyManage: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Manage all your API access keys here.</p><p style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px;\"><b>📌 What is an API Key?</b><br/>An API key is your credential for accessing AI services, like a key that allows your application to call AI capabilities.</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 Click to enter key page</p></div>",
      },
      createKey: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Click the button to create your first API key.</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Tip:</b> Key is only shown once after creation, make sure to copy and save</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 Click \"Create Key\"</p></div>",
      },
      keyName: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Give your key an easy-to-identify name.</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 Examples:</b> \"My First Key\", \"For Testing\", etc.</p></div>",
      },
      keyGroup: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Select the service group assigned by the administrator.</p><p style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px;\"><b>📌 Group Info:</b><br/>Different groups may have different service quality and billing rates, choose according to your needs.</p></div>",
      },
      keySubmit: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">Click to confirm and create your API key.</p><div style=\"padding: 8px 12px; background: #FAE4E1; border-left: 3px solid #D0685B; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚠️ Important:</b><ul style=\"margin: 8px 0 0 16px;\"><li>Copy the key (sk-xxx) immediately after creation</li><li>Key is only shown once, need to regenerate if lost</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>🚀 How to Use:</b><br/>Configure the key in any OpenAI-compatible client (like ChatBox, OpenCat, etc.) and start using!</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 Click \"Create\" button</p></div>",
      },
    },
  },
}
