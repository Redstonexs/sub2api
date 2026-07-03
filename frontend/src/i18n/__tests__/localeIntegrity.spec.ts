import { describe, expect, it } from 'vitest'
import { baseCompile } from '@intlify/message-compiler'

import en from '../locales/en'
import zh from '../locales/zh'

// Recursively flattens a locale object into dot-joined leaf paths.
type LocaleNode = string | number | boolean | LocaleNode[] | { [key: string]: LocaleNode }

function flatten(node: LocaleNode, prefix: string, out: Map<string, LocaleNode>): Map<string, LocaleNode> {
  if (Array.isArray(node)) {
    node.forEach((item, i) => flatten(item, `${prefix}[${i}]`, out))
  } else if (node !== null && typeof node === 'object') {
    for (const [key, value] of Object.entries(node)) {
      flatten(value, prefix ? `${prefix}.${key}` : key, out)
    }
  } else {
    out.set(prefix, node)
  }
  return out
}

const enLeaves = flatten(en as LocaleNode, '', new Map())
const zhLeaves = flatten(zh as LocaleNode, '', new Map())

describe('locale integrity', () => {
  it('en and zh define exactly the same keys', () => {
    const enOnly = [...enLeaves.keys()].filter((k) => !zhLeaves.has(k)).sort()
    const zhOnly = [...zhLeaves.keys()].filter((k) => !enLeaves.has(k)).sort()
    expect(enOnly, 'keys present in en.ts but missing from zh.ts').toEqual([])
    expect(zhOnly, 'keys present in zh.ts but missing from en.ts').toEqual([])
  })

  it.each([
    ['en', enLeaves],
    ['zh', zhLeaves]
  ])('every %s message compiles (unescaped { } blank the page at render time)', (_locale, leaves) => {
    const failures: string[] = []
    for (const [path, value] of leaves) {
      if (typeof value !== 'string') {
        failures.push(`${path}: non-string leaf (${typeof value})`)
        continue
      }
      try {
        baseCompile(value)
      } catch (error) {
        failures.push(`${path}: ${(error as Error).message}`)
      }
    }
    // Literal braces must be escaped as {'{'} and {'}'} in locale strings.
    expect(failures, failures.join('\n')).toEqual([])
  })
})
