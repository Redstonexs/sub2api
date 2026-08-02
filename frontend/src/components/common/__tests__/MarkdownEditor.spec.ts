import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'

import MarkdownEditor from '../MarkdownEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        // Placeholder strings are inserted into the document, so give them real
        // values rather than raw keys.
        const placeholders: Record<string, string> = {
          'markdownEditor.placeholders.boldText': 'bold text',
          'markdownEditor.placeholders.italicText': 'italic text',
          'markdownEditor.placeholders.strikeText': 'struck text',
          'markdownEditor.placeholders.linkText': 'link text',
          'markdownEditor.placeholders.linkUrl': 'https://example.com',
          'markdownEditor.placeholders.imageAlt': 'image description',
          'markdownEditor.placeholders.imageUrl': 'https://example.com/image.png',
          'markdownEditor.placeholders.code': 'code',
        }
        if (placeholders[key]) return placeholders[key]
        if (params) return `${key}:${JSON.stringify(params)}`
        return key
      },
    }),
  }
})

/** Mounts the editor and keeps `modelValue` in sync with the emitted updates. */
function mountEditor(initial = '', props: Record<string, unknown> = {}) {
  const wrapper = mount(MarkdownEditor, {
    props: {
      modelValue: initial,
      'onUpdate:modelValue': (value: string) => wrapper.setProps({ modelValue: value }),
      ...props,
    },
    attachTo: document.body,
  })
  return wrapper
}

function textarea(wrapper: VueWrapper) {
  return wrapper.find<HTMLTextAreaElement>('[data-testid="md-textarea"]').element
}

/** Places the caret/selection then clicks a toolbar button. */
async function select(wrapper: VueWrapper, start: number, end: number) {
  const el = textarea(wrapper)
  el.setSelectionRange(start, end)
  await wrapper.vm.$nextTick()
}

const value = (wrapper: VueWrapper) => wrapper.props('modelValue') as string

describe('MarkdownEditor toolbar', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('wraps the selection in bold markers', async () => {
    const wrapper = mountEditor('hello world')
    await select(wrapper, 6, 11)
    await wrapper.find('[data-testid="md-tool-bold"]').trigger('click')

    expect(value(wrapper)).toBe('hello **world**')
    wrapper.unmount()
  })

  it('inserts a placeholder when nothing is selected', async () => {
    const wrapper = mountEditor('')
    await select(wrapper, 0, 0)
    await wrapper.find('[data-testid="md-tool-bold"]').trigger('click')

    expect(value(wrapper)).toBe('**bold text**')
    wrapper.unmount()
  })

  it('toggles bold off when the selection is already wrapped', async () => {
    const wrapper = mountEditor('hello **world**')
    // Select just the inner text; the markers sit outside the selection.
    await select(wrapper, 8, 13)
    await wrapper.find('[data-testid="md-tool-bold"]').trigger('click')

    expect(value(wrapper)).toBe('hello world')
    wrapper.unmount()
  })

  it('toggles bold off when the markers are inside the selection', async () => {
    const wrapper = mountEditor('**world**')
    await select(wrapper, 0, 9)
    await wrapper.find('[data-testid="md-tool-bold"]').trigger('click')

    expect(value(wrapper)).toBe('world')
    wrapper.unmount()
  })

  it('applies a heading prefix to every line the selection touches', async () => {
    const wrapper = mountEditor('one\ntwo\nthree')
    await select(wrapper, 1, 6)
    await wrapper.find('[data-testid="md-tool-h2"]').trigger('click')

    expect(value(wrapper)).toBe('## one\n## two\nthree')
    wrapper.unmount()
  })

  it('replaces an existing heading level rather than stacking hashes', async () => {
    const wrapper = mountEditor('# one')
    await select(wrapper, 0, 0)
    await wrapper.find('[data-testid="md-tool-h3"]').trigger('click')

    expect(value(wrapper)).toBe('### one')
    wrapper.unmount()
  })

  it('removes the prefix when every selected line already has it', async () => {
    const wrapper = mountEditor('- one\n- two')
    await select(wrapper, 0, 11)
    await wrapper.find('[data-testid="md-tool-ul"]').trigger('click')

    expect(value(wrapper)).toBe('one\ntwo')
    wrapper.unmount()
  })

  it('numbers ordered list items sequentially', async () => {
    const wrapper = mountEditor('one\ntwo\nthree')
    await select(wrapper, 0, 13)
    await wrapper.find('[data-testid="md-tool-ol"]').trigger('click')

    expect(value(wrapper)).toBe('1. one\n2. two\n3. three')
    wrapper.unmount()
  })

  it('inserts a link with the URL selected for replacement', async () => {
    const wrapper = mountEditor('click here')
    await select(wrapper, 0, 10)
    await wrapper.find('[data-testid="md-tool-link"]').trigger('click')

    expect(value(wrapper)).toBe('[click here](https://example.com)')
    expect(textarea(wrapper).value.slice(
      textarea(wrapper).selectionStart,
      textarea(wrapper).selectionEnd,
    )).toBe('https://example.com')
    wrapper.unmount()
  })

  it('inserts an image with Markdown syntax, not an HTML tag', async () => {
    const wrapper = mountEditor('')
    await select(wrapper, 0, 0)
    await wrapper.find('[data-testid="md-tool-image"]').trigger('click')

    // Literal <img> HTML is dropped in emails; Markdown image syntax survives.
    expect(value(wrapper)).toBe('![image description](https://example.com/image.png)')
    wrapper.unmount()
  })

  it('inserts a table block separated by blank lines', async () => {
    const wrapper = mountEditor('intro')
    await select(wrapper, 5, 5)
    await wrapper.find('[data-testid="md-tool-table"]').trigger('click')

    expect(value(wrapper)).toBe('intro\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n')
    wrapper.unmount()
  })
})

describe('MarkdownEditor keyboard shortcuts', () => {
  it.each([
    ['b', '**bold text**'],
    ['i', '*italic text*'],
  ])('handles Ctrl+%s', async (key, expected) => {
    const wrapper = mountEditor('')
    await select(wrapper, 0, 0)
    await wrapper.find('[data-testid="md-textarea"]').trigger('keydown', { key, ctrlKey: true })
    await wrapper.vm.$nextTick()

    expect(value(wrapper)).toBe(expected)
    wrapper.unmount()
  })

  it('handles Cmd+K for links', async () => {
    const wrapper = mountEditor('')
    await select(wrapper, 0, 0)
    await wrapper.find('[data-testid="md-textarea"]').trigger('keydown', { key: 'k', metaKey: true })
    await wrapper.vm.$nextTick()

    expect(value(wrapper)).toBe('[link text](https://example.com)')
    wrapper.unmount()
  })

  it('ignores plain typing keys', async () => {
    const wrapper = mountEditor('x')
    await select(wrapper, 1, 1)
    await wrapper.find('[data-testid="md-textarea"]').trigger('keydown', { key: 'b' })

    expect(value(wrapper)).toBe('x')
    wrapper.unmount()
  })
})

describe('MarkdownEditor preview', () => {
  it('renders Markdown into the shared markdown-body container', async () => {
    const wrapper = mountEditor('## Heading\n\n**bold**')
    await wrapper.vm.$nextTick()

    const preview = wrapper.find('[data-testid="md-preview"]')
    expect(preview.find('.markdown-body h2').text()).toBe('Heading')
    expect(preview.find('.markdown-body strong').text()).toBe('bold')
    wrapper.unmount()
  })

  it('strips raw HTML in email mode but not in popup mode', async () => {
    const wrapper = mountEditor('<strong>raw</strong>')
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="md-preview"]').html()).toContain('<strong>raw</strong>')

    await wrapper.find('[data-testid="md-preview-mode-email"]').trigger('click')
    expect(wrapper.find('[data-testid="md-preview"]').html()).not.toContain('<strong>raw</strong>')
    expect(wrapper.emitted('update:previewMode')?.[0]).toEqual(['email'])
    wrapper.unmount()
  })

  it('can be hidden and shown again', async () => {
    const wrapper = mountEditor('x')
    expect(wrapper.find('[data-testid="md-preview"]').exists()).toBe(true)

    await wrapper.find('[data-testid="md-toggle-preview"]').trigger('click')
    expect(wrapper.find('[data-testid="md-preview"]').exists()).toBe(false)
    wrapper.unmount()
  })
})

describe('MarkdownEditor counter', () => {
  it('counts code points rather than UTF-16 units', async () => {
    // '𝄞' is a surrogate pair: String.length reports 2, Go's rune count reports 1.
    const wrapper = mountEditor('𝄞')
    const counter = wrapper.find('[data-testid="md-counter"]').text()

    expect(counter).toContain('"count":1')
    expect(counter).toContain('"count":4') // bytes
    wrapper.unmount()
  })

  it('warns near the limit and errors past it', async () => {
    const wrapper = mountEditor('a'.repeat(95), { maxLength: 100 })
    expect(wrapper.find('[data-testid="md-counter"]').classes().join(' ')).toContain('amber')

    await wrapper.setProps({ modelValue: 'a'.repeat(101) })
    expect(wrapper.find('[data-testid="md-counter"]').classes().join(' ')).toContain('red')
    wrapper.unmount()
  })

  it('never truncates input past the limit', async () => {
    const wrapper = mountEditor('', { maxLength: 5 })
    const el = textarea(wrapper)
    el.value = 'far too long to fit'
    await wrapper.find('[data-testid="md-textarea"]').trigger('input')

    // Destroying a paste is worse than showing a validation error.
    expect(value(wrapper)).toBe('far too long to fit')
    wrapper.unmount()
  })
})

describe('MarkdownEditor fullscreen', () => {
  it('toggles a fullscreen layer and leaves it on Escape', async () => {
    const wrapper = mountEditor('x')
    expect(wrapper.find('[data-testid="markdown-editor"]').classes()).not.toContain('fixed')

    await wrapper.find('[data-testid="md-toggle-fullscreen"]').trigger('click')
    expect(wrapper.find('[data-testid="markdown-editor"]').classes()).toContain('fixed')

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="markdown-editor"]').classes()).not.toContain('fixed')
    wrapper.unmount()
  })

  it('does not swallow Escape when not fullscreen', async () => {
    // The editor lives inside BaseDialog, which closes on Escape; only fullscreen
    // may intercept it.
    const wrapper = mountEditor('x')
    const seen = vi.fn()
    document.addEventListener('keydown', seen)

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(seen).toHaveBeenCalledTimes(1)

    document.removeEventListener('keydown', seen)
    wrapper.unmount()
  })
})

describe('MarkdownEditor heading level switching', () => {
  it('toggles a heading off when the level already matches', async () => {
    const wrapper = mountEditor('### one')
    await select(wrapper, 0, 0)
    await wrapper.find('[data-testid="md-tool-h3"]').trigger('click')

    expect(value(wrapper)).toBe('one')
    wrapper.unmount()
  })

  it('converts a bulleted list to a numbered list', async () => {
    const wrapper = mountEditor('- one\n- two')
    await select(wrapper, 0, 11)
    await wrapper.find('[data-testid="md-tool-ol"]').trigger('click')

    expect(value(wrapper)).toBe('1. one\n2. two')
    wrapper.unmount()
  })
})
