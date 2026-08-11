//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The site name no longer appears in the header — these bodies now render through
// the shared shell, which puts the purpose in the <h1> and the site name in the
// footer, matching the official templates. These tests still guard the thing that
// matters: every interpolated value reaches the HTML escaped.

func TestBuildVerifyCodeEmailBody_EscapesSiteName(t *testing.T) {
	svc := &EmailService{}

	t.Run("escapes_script_injection", func(t *testing.T) {
		body := svc.buildVerifyCodeEmailBody("en", "123456", `</h1><script>alert(1)</script><h1>`)

		assert.NotContains(t, body, "<script>")
		assert.Contains(t, body, "&lt;script&gt;")
	})

	t.Run("escapes_html_entities", func(t *testing.T) {
		body := svc.buildVerifyCodeEmailBody("en", "123456", `A&B<C>"D`)

		assert.Contains(t, body, "A&amp;B&lt;C&gt;&#34;D")
	})

	t.Run("normal_site_name_unchanged", func(t *testing.T) {
		body := svc.buildVerifyCodeEmailBody("en", "654321", "My Site")

		assert.Contains(t, body, "My Site")
		assert.Contains(t, body, "654321")
		assert.Contains(t, body, "<h1")
	})

	t.Run("follows_recipient_locale", func(t *testing.T) {
		en := svc.buildVerifyCodeEmailBody("en", "654321", "My Site")
		zh := svc.buildVerifyCodeEmailBody("zh", "654321", "My Site")

		assert.Contains(t, en, `<html lang="en"`)
		assert.Contains(t, en, "Email verification code")
		assert.Contains(t, en, "This email was sent automatically by My Site.")

		assert.Contains(t, zh, `<html lang="zh-CN"`)
		assert.Contains(t, zh, "邮箱验证码")
		assert.Contains(t, zh, "此邮件由 My Site 自动发送")
	})
}

func TestBuildPasswordResetEmailBody_EscapesSiteName(t *testing.T) {
	svc := &EmailService{}

	t.Run("escapes_html_tags_in_site_name", func(t *testing.T) {
		body := svc.buildPasswordResetEmailBody("zh", "https://example.com/reset?token=abc", `</h1><img src=x onerror=alert(1)>`)

		assert.NotContains(t, body, "<img src=x")
		assert.True(t, strings.Contains(body, "&lt;img"))
	})

	t.Run("escapes_html_entities", func(t *testing.T) {
		body := svc.buildPasswordResetEmailBody("zh", "https://example.com/reset", `A&B<C>`)

		assert.Contains(t, body, "A&amp;B&lt;C&gt;")
	})

	t.Run("normal_site_name_and_url_unchanged", func(t *testing.T) {
		resetURL := "https://example.com/reset?token=xyz"
		body := svc.buildPasswordResetEmailBody("zh", resetURL, "Sub2API")

		assert.Contains(t, body, "Sub2API")
		assert.Contains(t, body, resetURL)
	})

	t.Run("escapes_ampersand_in_reset_url", func(t *testing.T) {
		resetURL := "https://example.com/reset?a=1&b=2"
		body := svc.buildPasswordResetEmailBody("zh", resetURL, "Site")

		assert.NotContains(t, body, `href="https://example.com/reset?a=1&b=2"`)
		assert.Contains(t, body, `href="https://example.com/reset?a=1&amp;b=2"`)
	})

	t.Run("follows_recipient_locale", func(t *testing.T) {
		en := svc.buildPasswordResetEmailBody("en", "https://example.com/reset", "Site")
		zh := svc.buildPasswordResetEmailBody("zh", "https://example.com/reset", "Site")

		assert.Contains(t, en, "Reset password")
		assert.Contains(t, en, "This email was sent automatically by Site.")
		assert.Contains(t, zh, "重置密码")
		assert.Contains(t, zh, "此邮件由 Site 自动发送")
	})
}

func TestLegacyEmailSubjectFollowsLocale(t *testing.T) {
	assert.Equal(t, "[Site] Password reset request",
		legacyEmailSubject("en", "Site", "Password reset request", "密码重置请求"))
	assert.Equal(t, "[Site] 密码重置请求",
		legacyEmailSubject("zh", "Site", "Password reset request", "密码重置请求"))
	// An unknown or empty locale falls back to English, matching normalizeNotificationLocale.
	assert.Equal(t, "[Site] Password reset request",
		legacyEmailSubject("", "Site", "Password reset request", "密码重置请求"))
}
