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
})
