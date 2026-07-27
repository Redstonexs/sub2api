import { describe, expect, it } from 'vitest'
import src from '../CustomPageView.vue?raw'

describe('CustomPageView iframe security attributes', () => {
  it('iframe tag contains sandbox="allow-scripts allow-same-origin allow-forms allow-popups"', () => {
    // Extract the iframe opening tag region from raw source
    const iframeMatch = src.match(/<iframe[\s\S]*?>/)
    expect(iframeMatch).not.toBeNull()
    const iframeTag = iframeMatch![0]
    expect(iframeTag).toContain('sandbox="allow-scripts allow-same-origin allow-forms allow-popups"')
  })

  it('iframe tag contains referrerpolicy="no-referrer"', () => {
    const iframeMatch = src.match(/<iframe[\s\S]*?>/)
    expect(iframeMatch).not.toBeNull()
    const iframeTag = iframeMatch![0]
    expect(iframeTag).toContain('referrerpolicy="no-referrer"')
  })
})
