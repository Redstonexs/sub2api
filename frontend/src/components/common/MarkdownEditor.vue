<template>
  <div
    :class="[
      'flex flex-col overflow-hidden rounded-xl border border-gray-300 bg-white dark:border-dark-600 dark:bg-dark-800',
      fullscreen ? 'fixed inset-0 z-[130] rounded-none' : 'relative',
    ]"
    data-testid="markdown-editor"
  >
    <!-- Toolbar -->
    <div
      class="flex flex-wrap items-center gap-0.5 border-b border-gray-200 bg-gray-50/80 px-2 py-1.5 dark:border-dark-700 dark:bg-dark-900/40"
    >
      <template v-for="(item, index) in toolbar" :key="item.key ?? `sep-${index}`">
        <span v-if="item.separator" class="mx-1 h-5 w-px bg-gray-200 dark:bg-dark-600" />
        <button
          v-else
          type="button"
          :disabled="disabled"
          :title="item.title"
          :aria-label="item.title"
          :data-testid="`md-tool-${item.key}`"
          class="flex h-7 min-w-7 items-center justify-center rounded-md px-1.5 text-sm text-gray-600 transition-colors hover:bg-gray-200 hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white"
          @click="item.run"
        >
          <Icon v-if="item.icon" :name="item.icon" size="sm" />
          <span v-else :class="item.labelClass">{{ item.label }}</span>
        </button>
      </template>

      <div class="ml-auto flex items-center gap-1">
        <button
          type="button"
          :title="t('markdownEditor.togglePreview')"
          :aria-pressed="previewVisible"
          data-testid="md-toggle-preview"
          :class="[
            'flex h-7 items-center gap-1 rounded-md px-2 text-xs font-medium transition-colors',
            previewVisible
              ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
              : 'text-gray-600 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-dark-700',
          ]"
          @click="previewVisible = !previewVisible"
        >
          <Icon name="eye" size="sm" />
          <span class="hidden sm:inline">{{ t('markdownEditor.preview') }}</span>
        </button>
        <button
          type="button"
          :title="fullscreen ? t('markdownEditor.exitFullscreen') : t('markdownEditor.fullscreen')"
          data-testid="md-toggle-fullscreen"
          class="flex h-7 w-7 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-dark-700"
          @click="fullscreen = !fullscreen"
        >
          <Icon :name="fullscreen ? 'arrowDown' : 'arrowsUpDown'" size="sm" />
        </button>
      </div>
    </div>

    <!-- Preview-mode tabs -->
    <div
      v-if="previewVisible"
      class="flex items-center gap-1 border-b border-gray-200 bg-white px-2 py-1 dark:border-dark-700 dark:bg-dark-800"
    >
      <span class="mr-1 text-xs text-gray-500 dark:text-dark-400">{{ t('markdownEditor.previewAs') }}</span>
      <button
        v-for="mode in previewModes"
        :key="mode"
        type="button"
        :data-testid="`md-preview-mode-${mode}`"
        :class="[
          'rounded-md px-2 py-0.5 text-xs font-medium transition-colors',
          activePreviewMode === mode
            ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
            : 'text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-700',
        ]"
        @click="setPreviewMode(mode)"
      >
        {{ t(`markdownEditor.previewModes.${mode}`) }}
      </button>
    </div>

    <!-- Editor / preview panes -->
    <div :class="['flex min-h-0 flex-1', previewVisible ? 'divide-x divide-gray-200 dark:divide-dark-700' : '']">
      <div :class="previewVisible ? 'w-1/2 min-w-0' : 'w-full'">
        <textarea
          ref="textareaRef"
          :value="modelValue"
          :rows="fullscreen ? undefined : rows"
          :placeholder="placeholder"
          :disabled="disabled"
          spellcheck="false"
          data-testid="md-textarea"
          class="block h-full w-full resize-none border-0 bg-transparent px-3 py-2.5 font-mono text-sm leading-relaxed text-gray-900 outline-none placeholder:text-gray-400 focus:ring-0 disabled:cursor-not-allowed disabled:opacity-60 dark:text-gray-100 dark:placeholder:text-dark-500"
          :style="fullscreen ? undefined : { height: `${rows * 1.6}rem` }"
          @input="onInput"
          @keydown="onKeydown"
          @scroll="syncScroll('editor')"
        ></textarea>
      </div>

      <div
        v-if="previewVisible"
        ref="previewRef"
        :class="[
          'w-1/2 min-w-0 overflow-y-auto px-4 py-3',
          activePreviewMode === 'email' ? 'bg-[#f4f5f7]' : 'bg-white dark:bg-dark-800',
        ]"
        :style="fullscreen ? undefined : { height: `${rows * 1.6}rem` }"
        data-testid="md-preview"
        @scroll="syncScroll('preview')"
      >
        <p
          v-if="activePreviewMode === 'email'"
          class="mb-3 rounded-md bg-amber-50 px-2 py-1.5 text-[11px] leading-snug text-amber-800"
        >
          {{ t('markdownEditor.emailPreviewNote') }}
        </p>
        <div :class="previewSurfaceClass">
          <div v-if="renderedPreview" class="markdown-body" v-html="renderedPreview"></div>
          <p v-else class="text-sm text-gray-400 dark:text-dark-500">
            {{ t('markdownEditor.previewEmpty') }}
          </p>
        </div>
      </div>
    </div>

    <!-- Status bar -->
    <div
      class="flex items-center justify-between gap-3 border-t border-gray-200 bg-gray-50/80 px-3 py-1.5 text-[11px] dark:border-dark-700 dark:bg-dark-900/40"
    >
      <span class="hidden text-gray-400 dark:text-dark-500 sm:inline">
        {{ t('markdownEditor.shortcutHint') }}
      </span>
      <span :class="['ml-auto tabular-nums', counterClass]" data-testid="md-counter">
        {{ t('markdownEditor.characters', { count: characterCount, max: maxLength }) }}
        <span class="text-gray-400 dark:text-dark-500"> · {{ t('markdownEditor.bytes', { count: byteCount }) }}</span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { renderAnnouncementEmailPreview, renderAnnouncementMarkdown } from '@/utils/markdown'
import '@/styles/announcement-markdown.css'

/**
 * A dependency-free Markdown editor: toolbar, keyboard shortcuts, live split
 * preview, fullscreen and a character/byte counter.
 *
 * The preview deliberately reuses `@/utils/markdown`, the same renderer the
 * announcement popup, bell and banner use, so the admin sees exactly what a user
 * will see. `previewMode: 'email'` switches to the email renderer, which strips
 * raw HTML the way the backend's goldmark pipeline does.
 *
 * Editing is implemented by splicing `modelValue` and restoring the selection,
 * not `document.execCommand` (deprecated and inconsistent). One consequence worth
 * knowing: toolbar edits are not part of the textarea's native undo stack, so
 * Ctrl/Cmd+Z will not step back through them.
 */
type PreviewMode = 'popup' | 'bell' | 'email'

const props = withDefaults(defineProps<{
  modelValue: string
  rows?: number
  maxLength?: number
  placeholder?: string
  disabled?: boolean
  showPreview?: boolean
  previewMode?: PreviewMode
}>(), {
  rows: 14,
  maxLength: 20000,
  placeholder: '',
  disabled: false,
  showPreview: true,
  previewMode: 'popup',
})

const emit = defineEmits<{
  'update:modelValue': [string]
  'update:previewMode': [PreviewMode]
}>()

const { t } = useI18n()

const textareaRef = ref<HTMLTextAreaElement | null>(null)
const previewRef = ref<HTMLElement | null>(null)
const previewVisible = ref(props.showPreview)
const fullscreen = ref(false)
const internalPreviewMode = ref<PreviewMode>(props.previewMode)

const previewModes: PreviewMode[] = ['popup', 'bell', 'email']
const activePreviewMode = computed(() => internalPreviewMode.value)

watch(() => props.previewMode, (mode) => { internalPreviewMode.value = mode })
watch(() => props.showPreview, (show) => { previewVisible.value = show })

function setPreviewMode(mode: PreviewMode) {
  internalPreviewMode.value = mode
  emit('update:previewMode', mode)
}

// ===== Counters =====
// Spread-counting matches Go's utf8.RuneCountInString, which is what the backend
// validates against; `String.length` would count UTF-16 units and reject valid
// content early (or accept content the backend then rejects).
const characterCount = computed(() => [...(props.modelValue ?? '')].length)
const byteCount = computed(() => new TextEncoder().encode(props.modelValue ?? '').length)

const counterClass = computed(() => {
  if (characterCount.value > props.maxLength) return 'font-semibold text-red-600 dark:text-red-400'
  if (characterCount.value >= props.maxLength * 0.9) return 'font-medium text-amber-600 dark:text-amber-400'
  return 'text-gray-500 dark:text-dark-400'
})

// ===== Preview =====
const renderedPreview = computed(() => (
  activePreviewMode.value === 'email'
    ? renderAnnouncementEmailPreview(props.modelValue)
    : renderAnnouncementMarkdown(props.modelValue)
))

const previewSurfaceClass = computed(() => {
  switch (activePreviewMode.value) {
    // Approximates the width of each delivery surface so line wrapping in the
    // preview resembles what the reader actually gets.
    case 'bell':
      return 'mx-auto max-w-[420px]'
    case 'email':
      return 'mx-auto max-w-[600px] rounded-lg bg-white p-5 text-gray-800 shadow-sm [&_.markdown-body]:text-gray-800'
    default:
      return 'mx-auto max-w-[600px]'
  }
})

// ===== Editing primitives =====
function onInput(event: Event) {
  // Never truncate: silently destroying a paste is worse than a validation error.
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
}

/** Splices `modelValue` and restores the caret once Vue has flushed the change. */
async function replaceRange(start: number, end: number, text: string, selectionStart: number, selectionEnd: number) {
  const value = props.modelValue ?? ''
  emit('update:modelValue', value.slice(0, start) + text + value.slice(end))
  await nextTick()
  const el = textareaRef.value
  if (!el) return
  el.focus()
  el.setSelectionRange(selectionStart, selectionEnd)
}

/** Wraps the selection (or inserts `placeholder`), toggling the markers off if already applied. */
async function applyWrap(before: string, after: string, placeholder: string) {
  const el = textareaRef.value
  if (!el || props.disabled) return

  const value = props.modelValue ?? ''
  const { selectionStart: start, selectionEnd: end } = el
  const selected = value.slice(start, end)

  const wrappedOutside =
    start >= before.length &&
    value.slice(start - before.length, start) === before &&
    value.slice(end, end + after.length) === after
  if (wrappedOutside) {
    await replaceRange(start - before.length, end + after.length, selected, start - before.length, end - before.length)
    return
  }
  if (selected.startsWith(before) && selected.endsWith(after) && selected.length >= before.length + after.length) {
    const inner = selected.slice(before.length, selected.length - after.length)
    await replaceRange(start, end, inner, start, start + inner.length)
    return
  }

  const body = selected || placeholder
  await replaceRange(start, end, before + body + after, start + before.length, start + before.length + body.length)
}

/**
 * Adds or removes a per-line prefix across every line the selection touches.
 *
 * `pattern` matches any prefix of the same family, so applying H3 to an existing
 * H1 swaps the level instead of stacking hashes. Removal requires the *exact*
 * prefix on every line — otherwise clicking H3 on an H1 would just delete the
 * heading.
 */
async function applyLinePrefix(prefix: string | ((index: number) => string), pattern: RegExp) {
  const el = textareaRef.value
  if (!el || props.disabled) return

  const value = props.modelValue ?? ''
  const lineStart = value.lastIndexOf('\n', el.selectionStart - 1) + 1
  let lineEnd = value.indexOf('\n', el.selectionEnd)
  if (lineEnd === -1) lineEnd = value.length

  const prefixFor = (index: number) => (typeof prefix === 'function' ? prefix(index) : prefix)
  const lines = value.slice(lineStart, lineEnd).split('\n')
  const remove = lines.every((line, index) => line.startsWith(prefixFor(index)))
  const next = lines
    .map((line, index) => {
      const stripped = line.replace(pattern, '')
      return remove ? stripped : prefixFor(index) + stripped
    })
    .join('\n')

  await replaceRange(lineStart, lineEnd, next, lineStart, lineStart + next.length)
}

/** Inserts a block on its own line(s), keeping surrounding blank-line spacing sane. */
async function insertBlock(text: string) {
  const el = textareaRef.value
  if (!el || props.disabled) return

  const value = props.modelValue ?? ''
  const start = el.selectionStart
  const end = el.selectionEnd
  const before = start > 0 && !value.slice(0, start).endsWith('\n') ? '\n\n' : ''
  const after = end < value.length && !value.slice(end).startsWith('\n') ? '\n\n' : '\n'
  const payload = before + text + after
  const caret = start + payload.length
  await replaceRange(start, end, payload, caret, caret)
}

async function insertLink() {
  const el = textareaRef.value
  if (!el || props.disabled) return
  const value = props.modelValue ?? ''
  const selected = value.slice(el.selectionStart, el.selectionEnd)
  const label = selected || t('markdownEditor.placeholders.linkText')
  const url = t('markdownEditor.placeholders.linkUrl')
  const start = el.selectionStart
  // Leave the URL selected — it is the part the admin has to replace.
  await replaceRange(
    start,
    el.selectionEnd,
    `[${label}](${url})`,
    start + label.length + 3,
    start + label.length + 3 + url.length,
  )
}

async function insertImage() {
  const el = textareaRef.value
  if (!el || props.disabled) return
  const alt = t('markdownEditor.placeholders.imageAlt')
  const url = t('markdownEditor.placeholders.imageUrl')
  const start = el.selectionStart
  await replaceRange(
    start,
    el.selectionEnd,
    `![${alt}](${url})`,
    start + alt.length + 4,
    start + alt.length + 4 + url.length,
  )
}

// ===== Toolbar =====
const TABLE_SNIPPET = ['| A | B |', '| --- | --- |', '| 1 | 2 |'].join('\n')
// One pattern for both list buttons, so switching a bulleted list to a numbered
// one replaces the marker instead of prefixing it.
const LIST_ITEM_PREFIX = /^(?:[-*+]|\d+\.)\s/

interface ToolbarItem {
  key?: string
  title?: string
  label?: string
  labelClass?: string
  icon?: 'link' | 'terminal'
  separator?: boolean
  run?: () => void
}

const toolbar = computed<ToolbarItem[]>(() => [
  {
    key: 'bold',
    title: `${t('markdownEditor.toolbar.bold')} (Ctrl/⌘+B)`,
    label: 'B',
    labelClass: 'font-bold',
    run: () => applyWrap('**', '**', t('markdownEditor.placeholders.boldText')),
  },
  {
    key: 'italic',
    title: `${t('markdownEditor.toolbar.italic')} (Ctrl/⌘+I)`,
    label: 'I',
    labelClass: 'font-serif italic',
    run: () => applyWrap('*', '*', t('markdownEditor.placeholders.italicText')),
  },
  {
    key: 'strike',
    title: t('markdownEditor.toolbar.strikethrough'),
    label: 'S',
    labelClass: 'line-through',
    run: () => applyWrap('~~', '~~', t('markdownEditor.placeholders.strikeText')),
  },
  { separator: true },
  {
    key: 'h1',
    title: t('markdownEditor.toolbar.heading1'),
    label: 'H1',
    labelClass: 'text-xs font-semibold',
    run: () => applyLinePrefix('# ', /^#{1,6}\s/),
  },
  {
    key: 'h2',
    title: t('markdownEditor.toolbar.heading2'),
    label: 'H2',
    labelClass: 'text-xs font-semibold',
    run: () => applyLinePrefix('## ', /^#{1,6}\s/),
  },
  {
    key: 'h3',
    title: t('markdownEditor.toolbar.heading3'),
    label: 'H3',
    labelClass: 'text-xs font-semibold',
    run: () => applyLinePrefix('### ', /^#{1,6}\s/),
  },
  { separator: true },
  {
    key: 'quote',
    title: t('markdownEditor.toolbar.quote'),
    label: '❝',
    run: () => applyLinePrefix('> ', /^>\s?/),
  },
  {
    key: 'ul',
    title: t('markdownEditor.toolbar.bulletList'),
    label: '•—',
    labelClass: 'text-xs tracking-tighter',
    run: () => applyLinePrefix('- ', LIST_ITEM_PREFIX),
  },
  {
    key: 'ol',
    title: t('markdownEditor.toolbar.numberedList'),
    label: '1.',
    labelClass: 'text-xs',
    run: () => applyLinePrefix((index) => `${index + 1}. `, LIST_ITEM_PREFIX),
  },
  { separator: true },
  {
    key: 'link',
    title: `${t('markdownEditor.toolbar.link')} (Ctrl/⌘+K)`,
    icon: 'link',
    run: insertLink,
  },
  {
    key: 'image',
    title: t('markdownEditor.toolbar.image'),
    label: '▣',
    run: insertImage,
  },
  {
    key: 'code',
    title: t('markdownEditor.toolbar.inlineCode'),
    icon: 'terminal',
    run: () => applyWrap('`', '`', t('markdownEditor.placeholders.code')),
  },
  {
    key: 'codeblock',
    title: t('markdownEditor.toolbar.codeBlock'),
    label: '```',
    labelClass: 'text-[10px] font-mono',
    run: () => insertBlock('```\n' + t('markdownEditor.placeholders.code') + '\n```'),
  },
  { separator: true },
  {
    key: 'table',
    title: t('markdownEditor.toolbar.table'),
    label: '⊞',
    run: () => insertBlock(TABLE_SNIPPET),
  },
  {
    key: 'hr',
    title: t('markdownEditor.toolbar.divider'),
    label: '―',
    run: () => insertBlock('---'),
  },
])

// ===== Keyboard =====
function onKeydown(event: KeyboardEvent) {
  if (!(event.metaKey || event.ctrlKey) || event.altKey) return
  switch (event.key.toLowerCase()) {
    case 'b':
      event.preventDefault()
      void applyWrap('**', '**', t('markdownEditor.placeholders.boldText'))
      break
    case 'i':
      event.preventDefault()
      void applyWrap('*', '*', t('markdownEditor.placeholders.italicText'))
      break
    case 'k':
      event.preventDefault()
      void insertLink()
      break
  }
}

// Escape must exit fullscreen rather than close the surrounding dialog. BaseDialog
// listens on document in the bubble phase, so intercepting in the capture phase
// keeps the dialog open.
function onDocumentKeydownCapture(event: KeyboardEvent) {
  if (event.key === 'Escape' && fullscreen.value) {
    event.stopPropagation()
    event.preventDefault()
    fullscreen.value = false
  }
}

watch(fullscreen, (isFullscreen) => {
  if (typeof document === 'undefined') return
  if (isFullscreen) {
    document.addEventListener('keydown', onDocumentKeydownCapture, true)
  } else {
    document.removeEventListener('keydown', onDocumentKeydownCapture, true)
  }
})

onBeforeUnmount(() => {
  if (typeof document !== 'undefined') {
    document.removeEventListener('keydown', onDocumentKeydownCapture, true)
  }
})

// ===== Scroll sync =====
// Ratio-based and bidirectional; `syncing` stops the programmatic scroll on one
// pane from bouncing back through the other pane's scroll handler.
let syncing = false
function syncScroll(source: 'editor' | 'preview') {
  if (syncing || !previewVisible.value) return
  const src = source === 'editor' ? textareaRef.value : previewRef.value
  const dst = source === 'editor' ? previewRef.value : textareaRef.value
  if (!src || !dst) return

  syncing = true
  const scrollable = src.scrollHeight - src.clientHeight
  dst.scrollTop = scrollable > 0 ? (src.scrollTop / scrollable) * (dst.scrollHeight - dst.clientHeight) : 0
  const release = () => { syncing = false }
  if (typeof requestAnimationFrame === 'function') requestAnimationFrame(release)
  else release()
}

defineExpose({
  focus: () => textareaRef.value?.focus(),
  insert: (text: string) => {
    const el = textareaRef.value
    if (!el) return
    const caret = el.selectionStart + text.length
    return replaceRange(el.selectionStart, el.selectionEnd, text, caret, caret)
  },
})
</script>
