package service

import "strings"

// Shared HTML chrome for every email this gateway sends.
//
// Mail clients are not browsers. Outlook on Windows renders through Word, which
// ignores max-width, media queries, flexbox and border-radius; a few webmail
// clients drop <style> blocks entirely; and iOS/Android inflate small text
// unless told not to. So the shell below pairs *inline* styles — which survive
// everywhere and carry the base appearance — with one <style> block that adds
// the mobile overrides, each marked !important so it can beat the inline
// declaration it is overriding.
//
// Two rules when editing emailShellTemplate:
//
//  1. Never let two '{' or two '}' end up adjacent. Notification templates are
//     rendered by notification_email_service.go, which treats "{{name}}" as a
//     placeholder, so a nested CSS rule closed as "}}" parses as one. Keeping a
//     newline between nested braces (as below) is enough.
//  2. Do not feed the rendered shell to fmt.Sprintf. It contains bare '%' in
//     width="100%" attributes. Format the content fragment first, then wrap it.
const (
	emailCanvasColor     = "#f4f4f5"
	emailSurfaceColor    = "#ffffff"
	emailTextColor       = "#18181b"
	emailMutedColor      = "#71717a"
	emailBorderColor     = "#e4e4e7"
	emailFooterColor     = "#8b8b93"
	emailFooterBackColor = "#fafafa"
)

// emailLayout is one email's chrome. Title, Content and Footer are raw HTML:
// callers are responsible for escaping any user-controlled value they splice in
// (notification templates do it in renderNotificationEmailString, legacy
// builders call html.EscapeString).
type emailLayout struct {
	// Locale selects the footer wording, the html lang attribute and whether the
	// font stack carries CJK faces. Anything other than "zh" is treated as "en".
	Locale string
	// Accent is the header background and the colour of links and buttons.
	Accent string
	// Title is the header heading, also used as the document <title>. Plain text.
	Title string
	// Eyebrow is an optional kicker rendered above Title, Subtitle an optional
	// line below it. Both are omitted when empty.
	Eyebrow  string
	Subtitle string
	// Preheader is the inbox preview line. Left empty the block is omitted and
	// clients fall back to the start of the visible body, which is usually fine.
	Preheader string
	// Content is the body HTML. Helper classes available to it: "button", "muted",
	// "code", and "kv"/"kv-label"/"kv-value" for label/value tables.
	Content string
	// Footer is the closing note. Empty renders no footer row.
	Footer string
	// ExtraCSS is appended to the shell's <style> block for templates that need
	// layout the helper classes do not cover (the ops report's metric grid). Put
	// mobile overrides in their own @media block here; the shell's own media
	// block is closed before this lands.
	ExtraCSS string
}

func (l emailLayout) render() string {
	accent := strings.TrimSpace(l.Accent)
	if accent == "" {
		accent = "#3f3f46"
	}
	lang := "en"
	if isChineseEmailLocale(l.Locale) {
		lang = "zh-CN"
	}

	preheader := ""
	if text := strings.TrimSpace(l.Preheader); text != "" {
		preheader = `<div style="display:none;max-height:0;overflow:hidden;mso-hide:all;font-size:1px;line-height:1px;color:` +
			emailCanvasColor + `;opacity:0;">` + text + "</div>\n"
	}
	footer := ""
	if text := strings.TrimSpace(l.Footer); text != "" {
		footer = `<tr><td class="e-foot" style="padding:18px 32px;background-color:` + emailFooterBackColor +
			`;border-top:1px solid ` + emailBorderColor + `;color:` + emailFooterColor +
			`;font-size:12px;line-height:1.6;">` + text + "</td></tr>\n        "
	}

	extraCSS := ""
	if css := strings.TrimSpace(l.ExtraCSS); css != "" {
		extraCSS = "\n" + css
	}
	eyebrow := ""
	if text := strings.TrimSpace(l.Eyebrow); text != "" {
		eyebrow = `<p style="margin:0 0 8px;color:#ffffff;opacity:0.75;font-size:12px;font-weight:700;line-height:1.4;text-transform:uppercase;">` +
			text + "</p>\n          "
	}
	subtitle := ""
	if text := strings.TrimSpace(l.Subtitle); text != "" {
		subtitle = "\n          " +
			`<p style="margin:8px 0 0;color:#ffffff;opacity:0.85;font-size:14px;line-height:1.5;">` + text + `</p>`
	}

	// Replace runs a single left-to-right pass and never rescans inserted text, so
	// content that happens to contain a token spelling is not re-expanded.
	return strings.NewReplacer(
		"__LANG__", lang,
		"__FONT__", emailFontStack(l.Locale),
		"__ACCENT__", accent,
		"__TITLE__", l.Title,
		"__EYEBROW__", eyebrow,
		"__SUBTITLE__", subtitle,
		"__PREHEADER__", preheader,
		"__CONTENT__", l.Content,
		"__FOOTER__", footer,
		"__EXTRA_CSS__", extraCSS,
	).Replace(emailShellTemplate)
}

// emailFontStack keeps Chinese copy off the serif fallback Windows picks when a
// stack names no CJK face. Single-quoted: the stack is spliced into a
// style="..." attribute.
func emailFontStack(locale string) string {
	if isChineseEmailLocale(locale) {
		return "-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC','Hiragino Sans GB','Microsoft YaHei',Arial,sans-serif"
	}
	return "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif"
}

func isChineseEmailLocale(locale string) bool {
	return normalizeNotificationLocale(locale) == notificationEmailLocaleChinese
}

// emailAutoSendFooter is the "sent automatically, do not reply" closing line.
// site may be a literal (already escaped) or the {{site_name}} placeholder.
func emailAutoSendFooter(locale, site string) string {
	if isChineseEmailLocale(locale) {
		return "此邮件由 " + site + " 自动发送，请勿直接回复。"
	}
	return "This email was sent automatically by " + site + ". Please do not reply directly."
}

// emailKVRow renders one label/value row for a "kv" table. Both sides are raw
// HTML; callers escape. The classes are what the mobile media query keys off to
// stack the two cells instead of squeezing the value into a sliver.
//
// The wrapping declarations are inline rather than left to the stylesheet: a
// value can be arbitrary upstream text, and one unbroken 4000-character token is
// exactly the case that still has to wrap in a client that dropped <style>.
func emailKVRow(label, value string) string {
	return emailKVRowWrapped(label, value, "word-break:break-word;")
}

// emailKVRowUnbroken is for values that can arrive as a single unbroken token —
// an upstream error blob, an opaque ID. break-all splits mid-token, which reads
// badly for ordinary prose and is why it is not the default.
func emailKVRowUnbroken(label, value string) string {
	return emailKVRowWrapped(label, value, "word-break:break-all;white-space:pre-wrap;")
}

func emailKVRowWrapped(label, value, wrap string) string {
	return `<tr><td class="kv-label">` + label +
		`</td><td class="kv-value" style="overflow-wrap:anywhere;` + wrap + `">` + value + `</td></tr>`
}

// emailKVTable wraps pre-rendered emailKVRow output. table-layout:fixed is what
// keeps a long value inside the column rather than widening the table past the
// card; the class rule repeats it for clients that apply the stylesheet.
func emailKVTable(rows string) string {
	return `<table class="kv" role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"` +
		` style="width:100%;border-collapse:collapse;table-layout:fixed;">` + rows + `</table>`
}

const emailShellTemplate = `<!DOCTYPE html>
<html lang="__LANG__" dir="ltr" xmlns:o="urn:schemas-microsoft-com:office:office">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="x-apple-disable-message-reformatting">
<meta name="format-detection" content="telephone=no,date=no,address=no,email=no">
<meta name="color-scheme" content="light">
<meta name="supported-color-schemes" content="light">
<title>__TITLE__</title>
<!--[if mso]>
<noscript><xml><o:OfficeDocumentSettings><o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript>
<![endif]-->
<style>
  body { margin: 0 !important; padding: 0 !important; width: 100% !important; -webkit-text-size-adjust: 100%; -ms-text-size-adjust: 100%; }
  table { border-collapse: collapse; mso-table-lspace: 0pt; mso-table-rspace: 0pt; }
  img { border: 0; outline: none; text-decoration: none; -ms-interpolation-mode: bicubic; }
  a { color: __ACCENT__; }
  .e-body p { margin: 0 0 14px; }
  .e-body p:last-child { margin-bottom: 0; }
  .e-body h2 { margin: 24px 0 10px; font-size: 17px; line-height: 1.4; font-weight: 600; color: ` + emailTextColor + `; }
  .e-body .muted { color: ` + emailMutedColor + `; font-size: 13px; line-height: 1.6; }
  .e-body .button { display: inline-block; margin: 6px 0; padding: 12px 24px; background-color: __ACCENT__; border-radius: 8px; color: #ffffff !important; font-size: 15px; font-weight: 600; text-decoration: none; }
  .e-body .code { display: block; margin: 20px 0; padding: 18px 12px; background-color: ` + emailCanvasColor + `; border: 1px solid ` + emailBorderColor + `; border-radius: 10px; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 30px; font-weight: 700; letter-spacing: 8px; text-indent: 8px; text-align: center; color: ` + emailTextColor + `; }
  .e-body .kv { width: 100%; margin: 4px 0 18px; border-collapse: collapse; table-layout: fixed; }
  .e-body .kv td { padding: 9px 0; border-bottom: 1px solid #efeff1; font-size: 14px; line-height: 1.5; vertical-align: top; }
  .e-body .kv tr:last-child td { border-bottom: 0; }
  .e-body .kv-label { width: 40%; padding-right: 12px; color: ` + emailMutedColor + `; }
  .e-body .kv-value { color: ` + emailTextColor + `; overflow-wrap: anywhere; word-break: break-word; }
  /* Announcement images are hosted elsewhere and arrive at their natural size;
     without this they overflow the card on every phone. */
  .e-body img { max-width: 100% !important; height: auto !important; }
  .e-body table { max-width: 100% !important; }
  .e-body pre { max-width: 100%; overflow-x: auto; white-space: pre-wrap; word-break: break-word; }
  /* "rich" wraps HTML this file did not author — Markdown-rendered announcement
     bodies and ops report detail. Their tables and lists carry no classes of
     their own, so without these they arrive with browser-default styling, and
     an unconstrained table runs off the side of the card. Scoped to .rich so
     these element selectors cannot fight the .kv and .metrics rules above. */
  .e-body .rich h1, .e-body .rich h2 { margin: 22px 0 10px; font-size: 18px; line-height: 1.4; font-weight: 600; }
  .e-body .rich h3, .e-body .rich h4, .e-body .rich h5, .e-body .rich h6 { margin: 18px 0 8px; font-size: 15px; line-height: 1.4; font-weight: 600; }
  .e-body .rich ul, .e-body .rich ol { margin: 0 0 14px; padding-left: 22px; }
  .e-body .rich li { margin: 0 0 6px; }
  .e-body .rich blockquote { margin: 0 0 14px; padding: 6px 0 6px 14px; border-left: 3px solid ` + emailBorderColor + `; color: ` + emailMutedColor + `; }
  .e-body .rich code { padding: 2px 5px; background-color: ` + emailCanvasColor + `; border-radius: 4px; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 13px; }
  .e-body .rich pre { margin: 0 0 14px; padding: 12px 14px; background-color: ` + emailCanvasColor + `; border: 1px solid ` + emailBorderColor + `; border-radius: 8px; font-size: 13px; }
  .e-body .rich pre code { padding: 0; background-color: transparent; }
  .e-body .rich hr { margin: 20px 0; border: 0; border-top: 1px solid ` + emailBorderColor + `; }
  .e-body .rich table { width: 100%; margin: 4px 0 18px; border-collapse: collapse; table-layout: fixed; }
  .e-body .rich th, .e-body .rich td { padding: 8px 10px; border: 1px solid ` + emailBorderColor + `; font-size: 13px; line-height: 1.5; text-align: left; vertical-align: top; overflow-wrap: anywhere; word-break: break-word; }
  .e-body .rich th { background-color: ` + emailCanvasColor + `; font-weight: 600; }
  .e-body .rich img { margin: 6px 0; }
  @media only screen and (max-width: 620px) {
    .e-gutter { padding: 0 !important; }
    .e-card { border-radius: 0 !important; }
    .e-head { padding: 24px 20px !important; }
    .e-head h1 { font-size: 20px !important; }
    .e-body { padding: 24px 20px !important; font-size: 15px !important; }
    .e-foot { padding: 16px 20px !important; }
    .e-body .code { font-size: 24px !important; letter-spacing: 4px !important; text-indent: 4px !important; padding: 14px 8px !important; }
    .e-body .kv td { display: block !important; width: auto !important; padding-right: 0 !important; }
    .e-body .kv .kv-label { padding: 10px 0 0 !important; border-bottom: 0 !important; font-size: 12px !important; }
    .e-body .kv .kv-value { padding: 2px 0 10px !important; }
  }__EXTRA_CSS__
</style>
</head>
<body style="margin:0;padding:0;width:100%;background-color:` + emailCanvasColor + `;color:` + emailTextColor + `;font-family:__FONT__;">
__PREHEADER__<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;background-color:` + emailCanvasColor + `;">
  <tr>
    <td class="e-gutter" align="center" style="padding:24px 12px;">
      <!--[if mso]><table role="presentation" width="640" cellpadding="0" cellspacing="0" border="0"><tr><td><![endif]-->
      <table role="presentation" class="e-card" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:640px;margin:0 auto;background-color:` + emailSurfaceColor + `;border-radius:12px;overflow:hidden;">
        <tr><td class="e-head" style="padding:28px 32px;background-color:__ACCENT__;">
          __EYEBROW__<h1 style="margin:0;color:#ffffff;font-size:24px;line-height:1.3;font-weight:700;">__TITLE__</h1>__SUBTITLE__
        </td></tr>
        <tr><td class="e-body" style="padding:32px;color:` + emailTextColor + `;font-size:15px;line-height:1.7;word-break:break-word;">__CONTENT__</td></tr>
        __FOOTER__</table>
      <!--[if mso]></td></tr></table><![endif]-->
    </td>
  </tr>
</table>
</body>
</html>`
