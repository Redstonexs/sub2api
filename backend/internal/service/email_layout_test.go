//go:build unit

package service

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Every official template goes through emailLayout, so these run across the whole
// registry rather than sampling one: a template added later inherits the checks.
func TestOfficialEmailTemplatesShareResponsiveShell(t *testing.T) {
	for event, byLocale := range notificationEmailOfficialTemplates {
		for locale, tmpl := range byLocale {
			t.Run(event+"/"+locale, func(t *testing.T) {
				html := tmpl.HTML

				require.Contains(t, html, `<meta name="viewport" content="width=device-width,initial-scale=1">`,
					"without a viewport meta, mobile clients render at desktop width and zoom out")
				require.Contains(t, html, "@media only screen and (max-width: 620px)",
					"the shell's mobile overrides are what collapse the 32px padding on a phone")
				require.Contains(t, html, ".e-body img { max-width: 100% !important; height: auto !important; }",
					"this rule is what stops an oversized image being cut off")
				require.Contains(t, html, "max-width:640px", "the card must stay readable on desktop")

				require.LessOrEqual(t, len(html), notificationEmailMaxHTMLLength,
					"an official template that exceeds the limit cannot be saved back by an admin")
			})
		}
	}
}

// The footer, the lang attribute and the font stack all follow the locale the
// template is filed under. Before notificationEmailCard took a locale, every
// Chinese template shipped an English footer.
func TestOfficialEmailTemplatesAreLocalized(t *testing.T) {
	for event, byLocale := range notificationEmailOfficialTemplates {
		t.Run(event, func(t *testing.T) {
			en := byLocale[notificationEmailDefaultLocale].HTML
			zh := byLocale[notificationEmailLocaleChinese].HTML
			require.NotEmpty(t, en)
			require.NotEmpty(t, zh)

			require.Contains(t, en, `<html lang="en"`)
			require.Contains(t, en, "This email was sent automatically by {{site_name}}.")
			require.NotContains(t, en, "此邮件由")

			require.Contains(t, zh, `<html lang="zh-CN"`)
			require.Contains(t, zh, "此邮件由 {{site_name}} 自动发送，请勿直接回复。")
			require.NotContains(t, zh, "This email was sent automatically")
			require.Contains(t, zh, "PingFang SC", "Chinese copy falls back to a serif face without a CJK stack")
		})
	}
}

// The shell carries nested CSS rules, and the notification renderer reads
// "{{name}}" as a placeholder. Two adjacent braces in the stylesheet would be
// parsed as one and rejected by validateNotificationEmailTemplate.
func TestEmailShellCSSDoesNotForgePlaceholders(t *testing.T) {
	rendered := emailLayout{
		Locale:   notificationEmailDefaultLocale,
		Accent:   "#000000",
		Title:    "Title",
		Content:  "<p>body</p>",
		Footer:   "footer",
		ExtraCSS: opsReportMetricsCSS,
	}.render()

	require.NotContains(t, rendered, "{{", "an adjacent '{{' in CSS would be read as a placeholder opener")
	require.Empty(t, notificationEmailPlaceholdersIn(rendered))
}

func TestEmailLayoutOmitsEmptyOptionalBlocks(t *testing.T) {
	// The .e-foot and preheader rules live in the stylesheet unconditionally; it is
	// the markup that is omitted.
	bare := emailLayout{Locale: "en", Accent: "#000000", Title: "T", Content: "<p>c</p>"}.render()
	require.NotContains(t, bare, `<td class="e-foot"`)
	require.NotContains(t, bare, "mso-hide:all")

	full := emailLayout{
		Locale: "en", Accent: "#000000", Title: "T", Content: "<p>c</p>",
		Eyebrow: "Kicker", Subtitle: "Sub", Preheader: "Preview line", Footer: "Bye",
	}.render()
	require.Contains(t, full, "Kicker")
	require.Contains(t, full, "Sub")
	require.Contains(t, full, "Preview line")
	require.Contains(t, full, `<td class="e-foot"`)
}

// A caller's content is inserted verbatim: the single-pass replacer must not
// re-expand a token spelling that happens to appear inside it.
func TestEmailLayoutDoesNotRescanContent(t *testing.T) {
	rendered := emailLayout{
		Locale: "en", Accent: "#123456", Title: "T",
		Content: "<p>__ACCENT__ __FOOTER__ __CONTENT__</p>",
	}.render()

	require.Contains(t, rendered, "<p>__ACCENT__ __FOOTER__ __CONTENT__</p>")
}

func TestEmailFontStackAndFooterFollowLocale(t *testing.T) {
	require.Contains(t, emailFontStack("zh"), "PingFang SC")
	require.NotContains(t, emailFontStack("en"), "PingFang SC")
	// Unknown and empty locales normalize to English rather than erroring.
	require.NotContains(t, emailFontStack("de"), "PingFang SC")

	require.Equal(t, "此邮件由 ACME 自动发送，请勿直接回复。", emailAutoSendFooter("zh", "ACME"))
	require.Equal(t, "This email was sent automatically by ACME. Please do not reply directly.",
		emailAutoSendFooter("", "ACME"))
}

// The label/value tables stack on narrow screens instead of squeezing the value
// into whatever the fixed-width label column leaves behind.
func TestKVTablesUseStackingClasses(t *testing.T) {
	quota := notificationEmailOfficialTemplates[NotificationEmailEventAccountQuotaAlert][notificationEmailDefaultLocale].HTML

	require.Contains(t, quota, `<table class="kv"`)
	require.Contains(t, quota, `<td class="kv-label">`)
	require.Contains(t, quota, `<td class="kv-value" style="overflow-wrap:anywhere;`)
	require.Contains(t, quota, ".e-body .kv td { display: block !important; width: auto !important;")
	require.NotContains(t, quota, "width:128px", "a fixed label column leaves no room for the value on a phone")

	// The wrapping and fixed layout are also inline, so a pathological value still
	// wraps in a client that dropped the stylesheet.
	require.Contains(t, quota, "table-layout:fixed;")
}

// Announcement and ops-report bodies are HTML this package did not author, so
// their tables and lists only get styling through the "rich" wrapper.
func TestUnauthoredContentIsWrappedForStyling(t *testing.T) {
	announcement := notificationEmailOfficialTemplates[NotificationEmailEventAnnouncementBroadcast][notificationEmailDefaultLocale].HTML
	require.Contains(t, announcement, `<div class="rich">{{announcement_content}}</div>`)
	require.Contains(t, announcement, ".e-body .rich table { width: 100%;")
	require.Contains(t, announcement, "table-layout: fixed;")

	report := notificationEmailOfficialTemplates[NotificationEmailEventOpsScheduledReport][notificationEmailDefaultLocale].HTML
	require.Contains(t, report, `<div class="rich" style="display: {{report_detail_display}};">{{report_html}}</div>`)
}

// Both ops-report locales render from one skeleton, so a layout change cannot
// land in only one language.
func TestOpsReportLocalesStayStructurallyIdentical(t *testing.T) {
	tags := regexp.MustCompile(`>[^<]*<`)
	skeleton := func(locale string) string {
		return tags.ReplaceAllString(notificationEmailOpsScheduledReportTemplate(locale), "><")
	}

	en := skeleton(notificationEmailDefaultLocale)
	zh := skeleton(notificationEmailLocaleChinese)
	// The font stack and lang attribute are the only intended differences.
	en = strings.ReplaceAll(en, `lang="en"`, `lang="zh-CN"`)
	en = strings.ReplaceAll(en, emailFontStack("en"), emailFontStack("zh"))

	require.Equal(t, zh, en)
}

func TestHTMLToPlainText(t *testing.T) {
	t.Run("keeps link targets", func(t *testing.T) {
		require.Equal(t, "Reset password (https://x.dev/r)",
			htmlToPlainText(`<p><a class="button" href="https://x.dev/r">Reset password</a></p>`))
	})

	t.Run("collapses a bare link to its url", func(t *testing.T) {
		require.Equal(t, "https://x.dev/r", htmlToPlainText(`<a href="https://x.dev/r">https://x.dev/r</a>`))
	})

	t.Run("drops style and script but keeps image alt text", func(t *testing.T) {
		out := htmlToPlainText(`<style>p { color: red; }</style><script>x()</script><p>Hi <img src="https://c/a.png" alt="chart"></p>`)
		require.Equal(t, "Hi [chart]", out)
	})

	t.Run("keeps inline spacing around markup", func(t *testing.T) {
		require.Equal(t, "expires in 15 minutes", htmlToPlainText(`<p>expires in <strong>15</strong> minutes</p>`))
	})

	t.Run("renders rows without blank lines between them", func(t *testing.T) {
		require.Equal(t, "- one\n- two", htmlToPlainText(`<ul><li>one</li><li>two</li></ul>`))
		require.Equal(t, "EU | OK\nUS | OK",
			htmlToPlainText(`<table><tr><td>EU</td><td>OK</td></tr><tr><td>US</td><td>OK</td></tr></table>`))
	})

	t.Run("skips the hidden preheader", func(t *testing.T) {
		out := htmlToPlainText(`<div style="display:none;max-height:0;">inbox preview</div><p>real body</p>`)
		require.Equal(t, "real body", out)
	})

	t.Run("returns empty when there is no readable text", func(t *testing.T) {
		require.Empty(t, htmlToPlainText(`<html><head><style>p { color: red; }</style></head><body></body></html>`))
	})

	t.Run("renders every official template to non-empty text", func(t *testing.T) {
		for event, byLocale := range notificationEmailOfficialTemplates {
			for locale, tmpl := range byLocale {
				rendered, err := renderNotificationEmail(event, tmpl.Subject, tmpl.HTML,
					notificationEmailSampleVariables(locale), nil)
				require.NoError(t, err)

				plain := htmlToPlainText(rendered.HTML)
				require.NotEmpty(t, plain, "%s/%s produced no text alternative", event, locale)
				require.NotContains(t, plain, "<", "%s/%s leaked markup into the text part", event, locale)
				require.NotContains(t, plain, "e-body", "%s/%s leaked stylesheet text", event, locale)
			}
		}
	})
}

func TestLocalizedNotificationEmailEventLabel(t *testing.T) {
	require.Equal(t, "公告邮件",
		LocalizedNotificationEmailEventLabel(NotificationEmailEventAnnouncementBroadcast, "zh"))
	require.Equal(t, "Announcement broadcast",
		LocalizedNotificationEmailEventLabel(NotificationEmailEventAnnouncementBroadcast, "en"))
	// A non opt-out-able event has no translation and falls back to its English label.
	require.Equal(t, "Password reset",
		LocalizedNotificationEmailEventLabel(NotificationEmailEventAuthPasswordReset, "zh"))
	require.Equal(t, "who.knows", LocalizedNotificationEmailEventLabel("who.knows", "zh"))
}

func TestLegacyEmailBodiesUseTheSharedShell(t *testing.T) {
	notify := &BalanceNotifyService{}
	bodies := map[string]string{
		"balance_low":  notify.buildBalanceLowEmailBody("Alice", 3.14, 10, "MySite", "https://x.dev/pay"),
		"quota_alert":  notify.buildQuotaAlertEmailBody(42, "acct", "openai", "daily", 7, 10, 3, "$3.00", "MySite"),
		"notify_code":  buildNotifyVerifyEmailBody("123456", "MySite"),
		"test_message": BuildTestEmailBody("MySite"),
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			require.Contains(t, body, `<meta name="viewport" content="width=device-width,initial-scale=1">`)
			require.Contains(t, body, "@media only screen and (max-width: 620px)")
			require.NotContains(t, body, "display: flex", "flexbox does not render in most mail clients")
			require.NotContains(t, body, "linear-gradient", "Outlook drops gradients, leaving white text on white")
			require.NotEmpty(t, htmlToPlainText(body))
		})
	}
}

// The previous quota template spliced these into HTML raw.
func TestQuotaAlertEmailEscapesInterpolatedValues(t *testing.T) {
	body := (&BalanceNotifyService{}).buildQuotaAlertEmailBody(
		1, `<script>alert(1)</script>`, "openai", "daily", 1, 2, 1, "$1.00", `A&B`)

	require.NotContains(t, body, "<script>alert(1)</script>")
	require.Contains(t, body, "&lt;script&gt;")
	require.Contains(t, body, "A&amp;B")
}

func TestBalanceLowEmailEscapesUserName(t *testing.T) {
	body := (&BalanceNotifyService{}).buildBalanceLowEmailBody(`<b>x</b>`, 1, 2, "Site", "")

	require.NotContains(t, body, "<b>x</b>")
	require.Contains(t, body, "&lt;b&gt;x&lt;/b&gt;")
}

func TestUnsubscribeConfirmationLabelsAreDistinctPerLocale(t *testing.T) {
	// Guards the pairing the page depends on: a Chinese recipient should see a
	// Chinese event name, not the raw event key.
	for _, event := range []string{
		NotificationEmailEventAnnouncementBroadcast,
		NotificationEmailEventBalanceLow,
		NotificationEmailEventSubscriptionExpiryReminder,
	} {
		zh := LocalizedNotificationEmailEventLabel(event, "zh")
		require.NotEqual(t, event, zh, "optional event %s has no Chinese label", event)
		require.NotEqual(t, LocalizedNotificationEmailEventLabel(event, "en"), zh)
	}
}

func TestEmailLayoutFallsBackToADefaultAccent(t *testing.T) {
	rendered := emailLayout{Locale: "en", Title: "T", Content: "<p>c</p>"}.render()
	require.Contains(t, rendered, "background-color:#3f3f46;")
	require.NotContains(t, rendered, fmt.Sprintf("background-color:%s;", ""))
}
