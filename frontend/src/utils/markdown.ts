/**
 * Announcement Markdown rendering.
 *
 * This module is the single place where the frontend's Markdown pipeline is
 * reconciled with the backend's. Announcement content is rendered twice — in the
 * browser (popup, bell, banner, admin preview) by `marked` + DOMPurify, and into
 * broadcast emails by goldmark + bluemonday
 * (backend/internal/pkg/mdhtml/mdhtml.go). If the two drift, an admin previews
 * one thing and recipients receive another.
 *
 * Two differences have to be modelled explicitly:
 *
 *  1. *Sanitizer.* The allowlists below mirror `newSafeHTMLPolicy()`. Change both
 *     together.
 *  2. *Parser.* goldmark is built without `WithUnsafe()`, so raw HTML embedded in
 *     the Markdown never reaches bluemonday at all — it is dropped during
 *     rendering. `marked` passes raw HTML straight through to DOMPurify. So
 *     `<strong>x</strong>` written literally renders in-app but disappears in
 *     email, while `**x**` works in both. `renderAnnouncementEmailPreview` models
 *     the email side by disabling the raw-HTML renderer.
 *
 * Everything here uses scoped `Marked` and DOMPurify instances. The shared
 * `marked` singleton is deliberately left untouched: other pages (Model Plaza,
 * custom pages, legal documents) parse Markdown with it, and mutating it via
 * `setOptions` — as the announcement components used to — silently changed how
 * those pages rendered.
 */
import { Marked } from 'marked'
import DOMPurify from 'dompurify'

/**
 * Elements kept by the sanitizer.
 *
 * Mirrors `policy.AllowElements(...)` in backend/internal/pkg/mdhtml/mdhtml.go.
 * Notably absent: `input`, so GFM task lists (`- [ ] todo`) render as plain text
 * on both sides rather than widening the email sanitizer to accept form controls.
 */
export const ANNOUNCEMENT_ALLOWED_TAGS: readonly string[] = [
  'p', 'br',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
  'strong', 'em', 's', 'del', 'code', 'pre', 'blockquote',
  'ul', 'ol', 'li',
  'table', 'thead', 'tbody', 'tr', 'th', 'td',
  'hr', 'a', 'img',
]

/**
 * Attributes kept by the sanitizer. Mirrors the `AllowAttrs(...)` calls in
 * `newSafeHTMLPolicy()`. `class`, `id` and `title` are absent because bluemonday
 * drops them too — which is why fenced code blocks lose `class="language-go"`.
 */
export const ANNOUNCEMENT_ALLOWED_ATTR: readonly string[] = [
  'href', 'src', 'alt', 'width', 'height',
]

/**
 * Stamped on every surviving image, mirroring `responsiveImageStyle` in
 * backend/internal/pkg/mdhtml/mdhtml.go. Externally hosted images arrive at
 * their natural size, so without it a wide screenshot overflows the email card
 * and is cut off on phones.
 *
 * `style` is deliberately absent from `ANNOUNCEMENT_ALLOWED_ATTR`: the hook
 * below *sets* it after sanitization, so — exactly as on the backend, where the
 * bluemonday policy admits this one literal and nothing else — an author cannot
 * smuggle their own CSS in through an image.
 */
export const ANNOUNCEMENT_IMAGE_STYLE = 'max-width:100%;height:auto'

/**
 * Images must be absolute http(s) URLs. There is no upload endpoint, and a
 * relative or protocol-relative `src` resolves against the mail client's base
 * rather than ours. Mirrors `absoluteHTTPURL` on the backend.
 */
const ABSOLUTE_HTTP_URL = /^https?:\/\//i

/**
 * `alt` charset, mirroring `bluemonday.Paragraph`. An alt that fails this loses
 * the attribute (the image itself survives), exactly as bluemonday behaves.
 */
const ALT_TEXT = /^[\p{L}\p{N}\s\-_',[\]!./\\()]*$/u

/** `width`/`height` units, mirroring `bluemonday.NumberOrPercent`. */
const NUMBER_OR_PERCENT = /^[0-9]+%?$/

/**
 * Permitted URI schemes for links.
 *
 * The leading alternatives cover the schemes bluemonday's `AllowStandardURLs()`
 * permits (http, https, mailto); the trailing two are DOMPurify's own way of
 * spelling "no scheme at all", matching `AllowRelativeURLs(true)`. This is
 * DOMPurify's default regexp minus ftp/tel/callto/sms/cid/xmpp/matrix.
 */
const ANNOUNCEMENT_URI_REGEXP = /^(?:https?:|mailto:|[^a-z]|[a-z+.\-]+(?:[^a-z+.\-:]|$))/i

/** Renders in-app: raw HTML is passed to the sanitizer, matching `marked`. */
const browserMarked = new Marked({ gfm: true, breaks: true })

/**
 * Renders what a broadcast email will contain: the raw-HTML renderers are
 * neutralised so embedded HTML disappears before sanitization, mirroring goldmark
 * built without `WithUnsafe()`.
 */
const emailMarked = new Marked({ gfm: true, breaks: true })
emailMarked.use({ renderer: { html: () => '' } })

/**
 * A DOMPurify instance private to announcements.
 *
 * Hooks in DOMPurify are registered per-instance and the default export is a
 * shared one, so adding the image hook to it would silently change every other
 * `DOMPurify.sanitize()` call in the app.
 */
const announcementPurifier = DOMPurify()

announcementPurifier.addHook('afterSanitizeAttributes', (node) => {
  if (node.nodeName !== 'IMG') return
  const element = node as Element

  // The backend drops the whole node rather than leaving a src-less <img>, which
  // mail clients draw as a broken-image placeholder. Match that.
  if (!ABSOLUTE_HTTP_URL.test(element.getAttribute('src') ?? '')) {
    element.remove()
    return
  }
  const alt = element.getAttribute('alt')
  if (alt !== null && !ALT_TEXT.test(alt)) element.removeAttribute('alt')
  for (const attr of ['width', 'height']) {
    const value = element.getAttribute(attr)
    if (value !== null && !NUMBER_OR_PERCENT.test(value)) element.removeAttribute(attr)
  }
  element.setAttribute('style', ANNOUNCEMENT_IMAGE_STYLE)
})

/**
 * Sanitizes an HTML fragment down to the announcement allowlist.
 *
 * `KEEP_CONTENT` is left at its default (`true`), so a disallowed wrapper such as
 * `<div>` is unwrapped and its permitted children survive — the same shape
 * bluemonday produces.
 */
export function sanitizeAnnouncementHtml(html: string): string {
  return announcementPurifier.sanitize(html, {
    ALLOWED_TAGS: [...ANNOUNCEMENT_ALLOWED_TAGS],
    ALLOWED_ATTR: [...ANNOUNCEMENT_ALLOWED_ATTR],
    ALLOWED_URI_REGEXP: ANNOUNCEMENT_URI_REGEXP,
  })
}

/** Renders announcement Markdown for display inside the app. */
export function renderAnnouncementMarkdown(source: string | null | undefined): string {
  if (!source) return ''
  return sanitizeAnnouncementHtml(browserMarked.parse(source) as string)
}

/**
 * Renders announcement Markdown the way a broadcast email will render it: same
 * sanitizer, but with raw HTML stripped first to match goldmark.
 */
export function renderAnnouncementEmailPreview(source: string | null | undefined): string {
  if (!source) return ''
  return sanitizeAnnouncementHtml(emailMarked.parse(source) as string)
}
