import { describe, expect, it } from 'vitest'
import { marked } from 'marked'

import {
  ANNOUNCEMENT_ALLOWED_ATTR,
  ANNOUNCEMENT_ALLOWED_TAGS,
  ANNOUNCEMENT_IMAGE_STYLE,
  renderAnnouncementEmailPreview,
  renderAnnouncementMarkdown,
  sanitizeAnnouncementHtml,
} from '../markdown'

describe('renderAnnouncementMarkdown', () => {
  it('renders GFM with soft line breaks', () => {
    const html = renderAnnouncementMarkdown('# Heading\n\nline one\nline two')
    expect(html).toContain('<h1>Heading</h1>')
    expect(html).toContain('<br>')
  })

  it('renders tables and strikethrough', () => {
    const html = renderAnnouncementMarkdown('~~gone~~\n\n| a | b |\n|---|---|\n| 1 | 2 |')
    expect(html).toContain('<del>gone</del>')
    expect(html).toContain('<table>')
    expect(html).toContain('<td>1</td>')
  })

  it('returns an empty string for blank input', () => {
    expect(renderAnnouncementMarkdown('')).toBe('')
    expect(renderAnnouncementMarkdown(null)).toBe('')
    expect(renderAnnouncementMarkdown(undefined)).toBe('')
  })
})

describe('sanitizeAnnouncementHtml', () => {
  it('strips scripts and event handlers', () => {
    const html = sanitizeAnnouncementHtml(
      '<script>window.__xss = true</script><p onclick="alert(1)">text</p>',
    )
    expect(html).not.toContain('<script')
    expect(html).not.toContain('onclick')
    expect(html).toContain('text')
  })

  it('strips javascript: links but keeps http, mailto and relative ones', () => {
    expect(sanitizeAnnouncementHtml('<a href="javascript:alert(1)">x</a>')).not.toContain('javascript:')
    expect(sanitizeAnnouncementHtml('<a href="https://example.com">x</a>')).toContain('href="https://example.com"')
    expect(sanitizeAnnouncementHtml('<a href="mailto:a@example.com">x</a>')).toContain('mailto:a@example.com')
    expect(sanitizeAnnouncementHtml('<a href="/docs">x</a>')).toContain('href="/docs"')
  })

  it('unwraps disallowed containers but keeps their permitted children', () => {
    // Locks DOMPurify KEEP_CONTENT at its default; AnnouncementPopup.spec.ts relies
    // on <div> wrappers being unwrapped rather than dropped whole.
    const html = sanitizeAnnouncementHtml('<div><h3>kept</h3><ul><li>item</li></ul></div>')
    expect(html).not.toContain('<div')
    expect(html).toContain('<h3>kept</h3>')
    expect(html).toContain('<li>item</li>')
  })

  it('drops task-list checkboxes, matching the backend allowlist', () => {
    const html = renderAnnouncementMarkdown('- [ ] todo\n- [x] done')
    expect(html).not.toContain('<input')
    expect(html).toContain('todo')
  })

  it('drops class and title attributes, matching the backend allowlist', () => {
    const html = renderAnnouncementMarkdown('```go\nx := 1\n```')
    expect(html).toContain('<pre>')
    expect(html).not.toContain('class=')
  })
})

describe('image handling', () => {
  it('keeps absolute http(s) images', () => {
    const html = renderAnnouncementMarkdown('![logo](https://cdn.example.com/logo.png)')
    expect(html).toContain('src="https://cdn.example.com/logo.png"')
    expect(html).toContain('alt="logo"')
  })

  it.each([
    ['data URI', '![x](data:image/png;base64,iVBORw0KGgo=)'],
    ['relative path', '![x](/uploads/local.png)'],
    ['protocol relative', '![x](//evil.example.com/tracker.gif)'],
    ['javascript', '![x](javascript:alert&#40;1&#41;)'],
  ])('removes the whole <img> for a %s source', (_name, source) => {
    // The backend removes the node rather than leaving a src-less <img>, which mail
    // clients draw as a broken-image placeholder.
    expect(renderAnnouncementMarkdown(source)).not.toContain('<img')
  })

  it('drops an out-of-charset width but keeps the image', () => {
    const html = sanitizeAnnouncementHtml(
      '<img src="https://cdn.example.com/a.png" width="100%" height="expression(1)">',
    )
    expect(html).toContain('src="https://cdn.example.com/a.png"')
    expect(html).toContain('width="100%"')
    expect(html).not.toContain('height=')
  })

  it('makes every image fluid, matching the backend', () => {
    // Externally hosted images arrive at their natural size; without this a wide
    // asset overflows the email card and is cut off on phones.
    expect(renderAnnouncementMarkdown('![logo](https://cdn.example.com/logo.png)'))
      .toContain(`style="${ANNOUNCEMENT_IMAGE_STYLE}"`)
    expect(renderAnnouncementEmailPreview('![logo](https://cdn.example.com/logo.png)'))
      .toContain(`style="${ANNOUNCEMENT_IMAGE_STYLE}"`)
  })

  it('replaces an author-supplied style rather than trusting it', () => {
    // `style` is not in the allowlist; the hook sets it after sanitization, so an
    // author cannot smuggle CSS in through an image. Mirrors the backend policy,
    // which admits only the one literal.
    const html = sanitizeAnnouncementHtml(
      '<img src="https://cdn.example.com/a.png" style="position:fixed;top:0">',
    )
    expect(html).toContain(`style="${ANNOUNCEMENT_IMAGE_STYLE}"`)
    expect(html).not.toContain('position:fixed')
  })
})

describe('renderAnnouncementEmailPreview', () => {
  it('drops raw HTML, mirroring goldmark built without WithUnsafe', () => {
    const source = '<strong>raw</strong>\n\n**markdown**'
    const email = renderAnnouncementEmailPreview(source)

    expect(email).not.toContain('<strong>raw</strong>')
    expect(email).toContain('<strong>markdown</strong>')
  })

  it('drops a literal <img> tag but keeps Markdown image syntax', () => {
    expect(renderAnnouncementEmailPreview('<img src="https://cdn.example.com/a.png">')).not.toContain('<img')
    expect(renderAnnouncementEmailPreview('![a](https://cdn.example.com/a.png)')).toContain('<img')
  })

  it('differs from the in-app render exactly on raw HTML', () => {
    const source = '<strong>raw</strong>'
    expect(renderAnnouncementMarkdown(source)).toContain('<strong>raw</strong>')
    expect(renderAnnouncementEmailPreview(source)).not.toContain('<strong>')
  })
})

describe('allowlist mirrors the backend policy', () => {
  // Kept in sync with newSafeHTMLPolicy() in backend/internal/pkg/mdhtml/mdhtml.go.
  it('lists exactly the backend elements', () => {
    expect([...ANNOUNCEMENT_ALLOWED_TAGS].sort()).toEqual([
      'a', 'blockquote', 'br', 'code', 'del', 'em', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
      'hr', 'img', 'li', 'ol', 'p', 'pre', 's', 'strong', 'table', 'tbody', 'td',
      'th', 'thead', 'tr', 'ul',
    ])
  })

  it('lists exactly the backend attributes', () => {
    expect([...ANNOUNCEMENT_ALLOWED_ATTR].sort()).toEqual([
      'alt', 'height', 'href', 'src', 'width',
    ])
  })
})

describe('global marked singleton isolation', () => {
  it('does not mutate the shared marked instance', () => {
    // The announcement components used to call marked.setOptions({breaks:true}) at
    // module scope, which silently turned on soft line breaks for Model Plaza and
    // custom pages. Importing this module must not do that.
    renderAnnouncementMarkdown('a\nb')
    renderAnnouncementEmailPreview('a\nb')

    expect(marked.parse('a\nb') as string).not.toContain('<br>')
  })
})
