import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('OpenAI Fast/Flex policy locale keys', () => {
  it('exposes user scope copy at the runtime zh path', () => {
    expect(zh.admin.settings.openaiFastPolicy).toMatchObject({
      userIds: '指定用户 ID',
      userIdsHint: '留空表示对全部 Sub2API 用户生效。指定后仅匹配这些用户的 API Key 请求，且优先于全局规则。',
      userIdPlaceholder: '例如: 1001',
      addUserId: '添加用户 ID',
      removeUserId: '移除用户 ID'
    })
  })

  it('exposes user scope copy at the runtime en path', () => {
    expect(en.admin.settings.openaiFastPolicy).toMatchObject({
      userIds: 'Specific user IDs',
      userIdsHint: 'Leave empty to apply to all Sub2API users. Specified users match requests from their API keys and take precedence over global rules.',
      userIdPlaceholder: 'e.g., 1001',
      addUserId: 'Add user ID',
      removeUserId: 'Remove user ID'
    })
  })

  it('describes target and other-model actions without whitelist terminology', () => {
    expect(zh.admin.settings.openaiFastPolicy).toMatchObject({
      tierAll: '全部 tier 值',
      modelWhitelist: '目标模型',
      fallbackAction: '其他模型处理方式',
      summaryTargetModels: '目标模型',
      summaryOtherModels: '其他模型'
    })
    expect(zh.admin.settings.openaiFastPolicy.modelWhitelistHint).toContain(
      '留空时“处理方式”应用于全部模型'
    )
    expect(zh.admin.settings.openaiFastPolicy.modelWhitelistHint).not.toContain('白名单')

    expect(en.admin.settings.openaiFastPolicy).toMatchObject({
      tierAll: 'All tier values',
      modelWhitelist: 'Target models',
      fallbackAction: 'Other models action',
      summaryTargetModels: 'Target models',
      summaryOtherModels: 'Other models'
    })
    expect(en.admin.settings.openaiFastPolicy.modelWhitelistHint).toContain(
      'Leave empty to apply Action to all models'
    )
    expect(en.admin.settings.openaiFastPolicy.modelWhitelistHint).not.toContain('whitelist')
  })
})
