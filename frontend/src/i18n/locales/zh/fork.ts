// Fork-local locale overlay (Redstonexs/sub2api).
// Deep-merged over the upstream locale modules in ./index.ts — keys here are
// either fork-only features (gateway error messages, email provider, announcement
// email broadcast, ops system-log panel, onboarding palette) or fork overrides of
// upstream values. Keep en/fork.ts and zh/fork.ts key-for-key in sync
// (enforced by ../__tests__/localeIntegrity.spec.ts).
export default {
  common: {
    creating: "创建中...",
    sending: "发送中...",
    apply: "应用",
    clear: "清除",
    required: "不能为空",
    tryAgain: "请重试",
    tableOfContents: "目录",
    failed: "失败",
  },
  auth: {
    agreementPrompt: {
      agreeRead: "我已阅读并同意",
      mustAgree: "继续登录前需要先同意最新条款。",
      disabledHint: "未同意前，账号密码输入和快捷登录会保持禁用。",
      viewTerms: "查看条款",
      updateTitle: "条款更新通知",
      updateBody: "我们的服务条款已于 {date} 更新。在继续使用服务之前，请仔细阅读并同意以下条款。",
      recently: "近期",
      relatedDocs: "相关文档",
      reject: "拒绝",
      agreeContinue: "同意并继续",
      warnDisabledLogin: "未同意最新条款前，无法输入账号密码或使用快捷登录。",
      warnAgreeBeforeLogin: "请先阅读并同意最新条款后再登录。",
      warnDisabledRegister: "未同意最新条款前，无法注册或使用快捷登录。",
      warnAgreeBeforeRegister: "请先阅读并同意最新条款后再注册。",
    },
  },
  admin: {
    dashboard: {
      newUsersToday: "今日新增用户",
      active: "活跃",
      ok: "正常",
      err: "异常",
      create: "创建",
      userUsageTrend: "用户使用趋势（Top 12）",
    },
    users: {
      passwordCopied: "密码已复制",
    },
    groups: {
      selectedLabel: "已选",
      invertSelection: "反选",
      accountFilterControl: "账号过滤控制",
      oauthOnlyLabel: "仅允许 OAuth 账号",
      oauthOnlyEnabledHint: "已启用 — 排除 API Key 类型账号",
      filterNotEnabled: "未启用",
      privacyOnlyLabel: "仅允许隐私保护已设置的账号",
      privacyOnlyEnabledHint: "已启用 — Privacy 未设置的账号将被排除",
      accountCountLabel: "{count} 个账号",
      claudeMaxSimulation: {
        title: "Claude Max 用量模拟",
        tooltip: "启用后，对于上游未返回缓存写入用量的 Claude 模型，系统会在保持总 Token 数不变的前提下，将 Token 确定性地映射为少量输入加 1 小时缓存创建。",
        enabled: "已启用（模拟 1h 缓存）",
        disabled: "已禁用",
        hint: "仅调整用量计费日志中的 Token 分类，不会持久化任何按请求的映射状态。",
      },
    },
    channels: {
      noGroupsSelected: "{platform} 未选择任何分组，请至少选择一个分组",
      emptyModelsInPricing: "{platform} 存在未填写模型的定价条目，请补充模型或删除该条目",
    },
    accounts: {
      bulkActions: {
        deleteConfirm: "确定删除选中的 {count} 个账号吗？此操作不可撤销。",
        resetStatusConfirm: "确定重置选中的 {count} 个账号的错误状态吗？",
        refreshTokenConfirm: "确定刷新选中的 {count} 个账号的令牌吗？",
      },
      fromModel: "请求模型",
      toModel: "实际模型",
      messages: {
        accountCreated: "账号创建成功",
      },
      oauth: {
        openai: {
          mobileRefreshTokenAuth: "手动输入移动端 RT",
          accessTokenAuth: "手动输入 AT",
        },
      },
      gemini: {
        oauthType: {
          personalQuota: "个人账号，享受 Google One 订阅配额",
          recommendedPersonal: "推荐个人用户",
          noGcp: "无需 GCP",
          enterpriseGcp: "企业级，需要 GCP 项目",
          enterpriseGcpHint: "需要激活 GCP 项目并绑定信用卡",
          enterpriseUsers: "企业用户",
          highConcurrency: "高并发",
          showAdvancedOAuth: "显示高级选项（自建 OAuth Client）",
          hideAdvancedOAuth: "隐藏高级选项（自建 OAuth Client）",
          changeRegion: "修改归属地",
        },
      },
    },
    announcements: {
      notifyModeLabels: {
        email: "邮件",
      },
      emailUnsubscribed: "已退订",
      emailSubscribed: "已订阅",
      emailUnsubscribedLabel: "邮件订阅",
      form: {
        notifyModeHint: "弹窗模式会自动弹出通知给用户；邮件模式还会把公告通过邮件发送给所有符合条件的用户。",
      },
    },
    ops: {
      systemLog: {
        title: "系统日志",
        description: "默认按最新时间倒序，支持筛选搜索与按条件清理。",
        queue: "队列",
        written: "写入",
        dropped: "丢弃",
        failed: "失败",
        runtimeConfigTitle: "运行时日志配置（实时生效）",
        loading: "加载中...",
        level: "级别",
        stacktraceThreshold: "堆栈阈值",
        samplingInitial: "采样初始",
        samplingThereafter: "采样后续",
        retentionDays: "保留天数",
        caller: "调用方",
        sampling: "采样",
        saving: "保存中...",
        saveAndApply: "保存并生效",
        rollbackDefault: "回滚默认值",
        lastWriteError: "最近写入错误：",
        timeRange: "时间范围",
        startTimeOptional: "开始时间（可选）",
        endTimeOptional: "结束时间（可选）",
        component: "组件",
        componentPlaceholder: "如 http.access",
        keyId: "KEY ID",
        platform: "平台",
        model: "模型",
        keyword: "关键词",
        keywordPlaceholder: "消息/request_id",
        query: "查询",
        reset: "重置",
        cleanupCurrent: "按当前筛选清理",
        refreshHealth: "刷新健康指标",
        empty: "暂无系统日志",
        colTime: "时间",
        colDetail: "日志详细信息",
        all: "全部",
        loadFailed: "系统日志加载失败",
        configApplied: "日志运行时配置已生效",
        saveConfigFailed: "保存日志配置失败",
        confirmRollback: "确认回滚为启动配置（env/yaml）并立即生效？",
        rolledBack: "已回滚到启动日志配置",
        rollbackFailed: "回滚日志配置失败",
        confirmCleanup: "确认按当前筛选条件清理系统日志？该操作不可撤销。",
        cleanupDone: "清理完成，删除 {count} 条日志",
        cleanupFailed: "清理系统日志失败",
      },
      runtime: {
        metricThresholds: "指标阈值配置",
        metricThresholdsHint: "配置各项指标的告警阈值，超出阈值时将以红色显示",
        slaMinPercent: "SLA最低百分比",
        slaMinPercentHint: "SLA低于此值时显示为红色（默认：99.5%）",
        ttftP99MaxMs: "TTFT P99最大值（毫秒）",
        ttftP99MaxMsHint: "TTFT P99高于此值时显示为红色（默认：500ms）",
        requestErrorRateMaxPercent: "请求错误率最大值（%）",
        requestErrorRateMaxPercentHint: "请求错误率高于此值时显示为红色（默认：5%）",
        upstreamErrorRateMaxPercent: "上游错误率最大值（%）",
        upstreamErrorRateMaxPercentHint: "上游错误率高于此值时显示为红色（默认：5%）",
      },
    },
    settings: {
      email: {
        provider: "发送方式",
        providerHint: "邮件的发送渠道。API 方式（Resend / CyberPanel）通过 HTTPS 发送，可绕过云厂商对 SMTP 端口的封锁（如 DigitalOcean 封锁 25/465/587）。",
        providerSmtp: "SMTP",
        providerResend: "Resend API",
        providerCyberPanel: "CyberPanel API",
        apiBaseUrl: "API 地址",
        apiBaseUrlHint: "CyberPanel：填写你的邮件服务器地址。Resend：可选，默认 https://api.resend.com。",
        apiBaseUrlPlaceholderResend: "https://api.resend.com（可选）",
        apiBaseUrlPlaceholderCyberPanel: "https://mail.yourdomain.com",
        apiKey: "API Key",
        apiKeyPlaceholder: "Bearer 令牌（如 re_... 或 sk_live_...）",
        apiKeyHint: "以 Authorization: Bearer 请求头发送。",
        apiKeyConfiguredPlaceholder: "********",
        apiKeyConfiguredHint: "已配置 API Key，留空则保持当前值不变。",
      },
      gatewayErrorMessages: {
        title: "网关错误提示",
        description: "自定义网关在返回特定 HTTP 状态码时展示给用户的错误提示。",
        codeHeader: "状态码",
        messageHeader: "提示内容",
        codePlaceholder: "如 429",
        messagePlaceholder: "展示给用户的提示",
        addRow: "添加提示",
        remove: "删除",
        empty: "暂无自定义提示。点击“添加提示”可覆盖某个 HTTP 状态码。",
        hint: "将 HTTP 状态码（如 429 或 502）映射到展示给用户的提示。仅需添加要覆盖的状态码，其余保持默认。",
        invalidForm: "请先修正高亮的网关错误提示，然后再保存。",
        errors: {
          codeEmpty: "请输入 HTTP 状态码。",
          codeInvalid: "请输入有效的 3 位 HTTP 状态码（100–599）。",
          codeDuplicate: "该状态码已配置。",
          messageEmpty: "请为该状态码输入提示内容。",
        },
      },
      gatewayForwarding: {
        claudeOAuthSystemPromptBlocksPlaceholder: "留空时使用内置 3 个 blocks。支持数组或 {'{'}\"blocks\": [...]{'}'}。",
      },
    },
  },
  payment: {
    admin: {
      allowUserRefund: "允许用户退款",
    },
  },
  onboarding: {
    admin: {
      welcome: {
        description: "<div style=\"line-height: 1.8;\"><p style=\"margin-bottom: 16px;\">Sub2API 是一个强大的 AI 服务中转平台，让您轻松管理和分发 AI 服务。</p><p style=\"margin-bottom: 12px;\"><b>🎯 核心功能：</b></p><ul style=\"margin-left: 20px; margin-bottom: 16px;\"><li>📦 <b>分组管理</b> - 创建不同的服务套餐（VIP、免费试用等）</li><li>🔗 <b>账号池</b> - 连接多个上游 AI 服务商账号</li><li>🔑 <b>密钥分发</b> - 为用户生成独立的 API Key</li><li>💰 <b>计费管理</b> - 灵活的费率和配额控制</li></ul><p style=\"color: #5D9A51; font-weight: 600;\">接下来，我们将用 3 分钟带您完成首次配置 →</p></div>",
      },
      groupManage: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\"><b>什么是分组？</b></p><p style=\"margin-bottom: 12px;\">分组是 Sub2API 的核心概念，它就像一个\"服务套餐\"：</p><ul style=\"margin-left: 20px; margin-bottom: 12px; font-size: 13px;\"><li>🎯 每个分组可以包含多个上游账号</li><li>💰 每个分组有独立的计费倍率</li><li>👥 可以设置为公开或专属分组</li></ul><p style=\"margin-top: 12px; padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 示例：</b>您可以创建\"VIP专线\"（高倍率）和\"免费试用\"（低倍率）两个分组</p><p style=\"margin-top: 16px; color: #5D9A51; font-weight: 600;\">👉 点击左侧的\"分组管理\"开始</p></div>",
      },
      createGroup: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">现在让我们创建第一个分组。</p><p style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📝 提示：</b>建议先创建一个测试分组，熟悉流程后再创建正式分组</p><p style=\"color: #5D9A51; font-weight: 600;\">👉 点击\"创建分组\"按钮</p></div>",
      },
      groupName: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">为您的分组起一个易于识别的名称。</p><div style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>💡 命名建议：</b><ul style=\"margin: 8px 0 0 16px;\"><li>\"测试分组\" - 用于测试</li><li>\"VIP专线\" - 高质量服务</li><li>\"免费试用\" - 体验版</li></ul></div><p style=\"font-size: 13px; color: #827B6C;\">填写完成后点击\"下一步\"继续</p></div>",
      },
      groupPlatform: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">选择该分组支持的 AI 平台。</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📌 平台说明：</b><ul style=\"margin: 8px 0 0 16px;\"><li><b>Anthropic</b> - Claude 系列模型</li><li><b>OpenAI</b> - GPT 系列模型</li><li><b>Google</b> - Gemini 系列模型</li></ul></div><p style=\"font-size: 13px; color: #827B6C;\">一个分组只能选择一个平台</p></div>",
      },
      groupMultiplier: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">设置该分组的计费倍率，控制用户的实际扣费。</p><div style=\"padding: 8px 12px; background: #FFF0DB; border-left: 3px solid #E2A846; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚙️ 计费规则：</b><ul style=\"margin: 8px 0 0 16px;\"><li><b>1.0</b> - 原价计费（成本价）</li><li><b>1.5</b> - 用户消耗 $1，扣除 $1.5</li><li><b>2.0</b> - 用户消耗 $1，扣除 $2</li><li><b>0.8</b> - 补贴模式（亏本运营）</li></ul></div><p style=\"font-size: 13px; color: #827B6C;\">建议测试分组设置为 1.0</p></div>",
      },
      groupExclusive: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">控制分组的可见性和访问权限。</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>🔐 权限说明：</b><ul style=\"margin: 8px 0 0 16px;\"><li><b>关闭</b> - 公开分组，所有用户可见</li><li><b>开启</b> - 专属分组，仅指定用户可见</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 使用场景：</b>VIP 用户专属、内部测试、特殊客户等</p></div>",
      },
      groupSubmit: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">确认信息无误后，点击创建按钮保存分组。</p><p style=\"padding: 8px 12px; background: #FFF0DB; border-left: 3px solid #E2A846; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚠️ 注意：</b>分组创建后，平台类型不可修改，其他信息可以随时编辑</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>📌 下一步：</b>创建成功后，我们将添加上游账号到这个分组</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 点击\"创建\"按钮</p></div>",
      },
      accountManage: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\"><b>太棒了！分组已创建成功 🎉</b></p><p style=\"margin-bottom: 12px;\">现在需要添加上游 AI 服务商的账号，让分组能够实际提供服务。</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>🔑 账号的作用：</b><ul style=\"margin: 8px 0 0 16px;\"><li>连接到上游 AI 服务（Claude、GPT 等）</li><li>一个分组可以包含多个账号（负载均衡）</li><li>支持 OAuth 和 Session Key 两种方式</li></ul></div><p style=\"margin-top: 16px; color: #5D9A51; font-weight: 600;\">👉 点击左侧的\"账号管理\"</p></div>",
      },
      createAccount: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">点击按钮开始添加您的第一个上游账号。</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 提示：</b>建议使用 OAuth 方式，更安全且无需手动提取密钥</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 点击\"添加账号\"按钮</p></div>",
      },
      accountName: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">为账号设置一个便于识别的名称。</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 命名建议：</b>\"Claude主账号\"、\"GPT备用1\"、\"测试账号\" 等</p></div>",
      },
      accountPlatform: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">选择该账号对应的服务商平台。</p><p style=\"padding: 8px 12px; background: #FFF0DB; border-left: 3px solid #E2A846; border-radius: 4px; font-size: 13px;\"><b>⚠️ 重要：</b>平台必须与刚才创建的分组平台一致</p></div>",
      },
      accountType: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">选择账号的授权方式。</p><div style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>✅ 推荐：OAuth 方式</b><ul style=\"margin: 8px 0 0 16px;\"><li>无需手动提取密钥</li><li>更安全，支持自动刷新</li><li>适用于 Claude Code、ChatGPT OAuth</li></ul></div><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px;\"><b>📌 Session Key 方式</b><ul style=\"margin: 8px 0 0 16px;\"><li>需要手动从浏览器提取</li><li>可能需要定期更新</li><li>适用于不支持 OAuth 的平台</li></ul></div></div>",
      },
      accountPriority: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">设置账号的调用优先级。</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📊 优先级规则：</b><ul style=\"margin: 8px 0 0 16px;\"><li>数字越小，优先级越高</li><li>系统优先使用低数值账号</li><li>相同优先级则随机选择</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 使用场景：</b>主账号设置低数值，备用账号设置高数值</p></div>",
      },
      accountGroups: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\"><b>关键步骤！</b>将账号分配到刚才创建的分组。</p><div style=\"padding: 8px 12px; background: #FAE4E1; border-left: 3px solid #D0685B; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚠️ 重要提醒：</b><ul style=\"margin: 8px 0 0 16px;\"><li>必须勾选至少一个分组</li><li>未分配分组的账号无法使用</li><li>一个账号可以分配给多个分组</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 提示：</b>请勾选刚才创建的测试分组</p></div>",
      },
      accountSubmit: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">确认信息无误后，点击保存按钮。</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📌 OAuth 授权流程：</b><ul style=\"margin: 8px 0 0 16px;\"><li>点击保存后会跳转到服务商页面</li><li>在服务商页面完成登录授权</li><li>授权成功后自动返回</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>📌 下一步：</b>账号添加成功后，我们将创建 API 密钥</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 点击\"保存\"按钮</p></div>",
      },
      keyManage: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\"><b>恭喜！账号配置完成 🎉</b></p><p style=\"margin-bottom: 12px;\">最后一步，生成 API Key 来测试服务是否正常工作。</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>🔑 API Key 的作用：</b><ul style=\"margin: 8px 0 0 16px;\"><li>用于调用 AI 服务的凭证</li><li>每个 Key 绑定一个分组</li><li>可以设置配额和有效期</li><li>支持独立的使用统计</li></ul></div><p style=\"margin-top: 16px; color: #5D9A51; font-weight: 600;\">👉 点击左侧的\"API 密钥\"</p></div>",
      },
      createKey: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">点击按钮创建您的第一个 API Key。</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 提示：</b>创建后请立即复制保存，密钥只显示一次</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 点击\"创建密钥\"按钮</p></div>",
      },
      keyName: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">为密钥设置一个便于管理的名称。</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 命名建议：</b>\"测试密钥\"、\"生产环境\"、\"移动端\" 等</p></div>",
      },
      keyGroup: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">选择刚才配置好的分组。</p><div style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>📌 分组决定：</b><ul style=\"margin: 8px 0 0 16px;\"><li>该密钥可以使用哪些账号</li><li>计费倍率是多少</li><li>是否为专属密钥</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 提示：</b>选择刚才创建的测试分组</p></div>",
      },
      keySubmit: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">点击创建后，系统会生成完整的 API Key。</p><div style=\"padding: 8px 12px; background: #FAE4E1; border-left: 3px solid #D0685B; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚠️ 重要提醒：</b><ul style=\"margin: 8px 0 0 16px;\"><li>密钥只显示一次，请立即复制</li><li>丢失后需要重新生成</li><li>妥善保管，不要泄露给他人</li></ul></div><div style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>🚀 下一步：</b><ul style=\"margin: 8px 0 0 16px;\"><li>复制生成的 sk-xxx 密钥</li><li>在支持 OpenAI 接口的客户端中使用</li><li>开始体验 AI 服务！</li></ul></div><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 点击\"创建\"按钮</p></div>",
      },
    },
    user: {
      welcome: {
        description: "<div style=\"line-height: 1.8;\"><p style=\"margin-bottom: 16px;\">您好！欢迎来到 Sub2API AI 服务平台。</p><p style=\"margin-bottom: 12px;\"><b>🎯 快速开始：</b></p><ul style=\"margin-left: 20px; margin-bottom: 16px;\"><li>🔑 创建 API 密钥</li><li>📋 复制密钥到您的应用</li><li>🚀 开始使用 AI 服务</li></ul><p style=\"color: #5D9A51; font-weight: 600;\">只需 1 分钟，让我们开始吧 →</p></div>",
      },
      keyManage: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">在这里管理您的所有 API 访问密钥。</p><p style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px;\"><b>📌 什么是 API 密钥？</b><br/>API 密钥是您访问 AI 服务的凭证，就像一把钥匙，让您的应用能够调用 AI 能力。</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 点击进入密钥页面</p></div>",
      },
      createKey: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">点击按钮创建您的第一个 API 密钥。</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 提示：</b>创建后密钥只显示一次，请务必复制保存</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 点击\"创建密钥\"</p></div>",
      },
      keyName: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">为密钥起一个便于识别的名称。</p><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>💡 示例：</b>\"我的第一个密钥\"、\"测试用\" 等</p></div>",
      },
      keyGroup: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">选择管理员为您分配的服务分组。</p><p style=\"padding: 8px 12px; background: #F1F6FD; border-left: 3px solid #5688CF; border-radius: 4px; font-size: 13px;\"><b>📌 分组说明：</b><br/>不同分组可能有不同的服务质量和计费标准，请根据需要选择。</p></div>",
      },
      keySubmit: {
        description: "<div style=\"line-height: 1.7;\"><p style=\"margin-bottom: 12px;\">点击确认创建您的 API 密钥。</p><div style=\"padding: 8px 12px; background: #FAE4E1; border-left: 3px solid #D0685B; border-radius: 4px; font-size: 13px; margin-bottom: 12px;\"><b>⚠️ 重要：</b><ul style=\"margin: 8px 0 0 16px;\"><li>创建后请立即复制密钥（sk-xxx）</li><li>密钥只显示一次，丢失需重新生成</li></ul></div><p style=\"padding: 8px 12px; background: #F4FCF2; border-left: 3px solid #5D9A51; border-radius: 4px; font-size: 13px;\"><b>🚀 如何使用：</b><br/>将密钥配置到支持 OpenAI 接口的任何客户端（如 ChatBox、OpenCat 等），即可开始使用！</p><p style=\"margin-top: 12px; color: #5D9A51; font-weight: 600;\">👉 点击\"创建\"按钮</p></div>",
      },
    },
  },
}
