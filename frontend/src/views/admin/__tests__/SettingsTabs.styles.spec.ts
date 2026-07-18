import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const currentDir = dirname(fileURLToPath(import.meta.url))
const settingsViewSource = readFileSync(resolve(currentDir, '../SettingsView.vue'), 'utf8')

describe('SettingsView tab rail', () => {
  it('keeps labels readable by scrolling constrained tabs instead of squeezing each one equally', () => {
    expect(settingsViewSource).toContain('@apply flex min-w-max items-center gap-1;')
    expect(settingsViewSource).not.toContain('@apply min-w-full;')
    expect(settingsViewSource).not.toContain('flex-1 basis-0 overflow-hidden px-2 text-[13px]')
  })
})
