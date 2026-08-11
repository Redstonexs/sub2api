package service

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// These are the built-in fallback bodies for the content-moderation notification
// emails, used when the templated path in NotificationEmailService fails to
// render. They go through the shared responsive shell in email_layout.go with the
// same accent colours as their official counterparts, so a recipient cannot tell
// which path produced the mail — and so the fallback is not the one message that
// overflows on a phone.

// contentModerationEmailUser renders the recipient label for a moderation log.
func contentModerationEmailUser(log *ContentModerationLog) string {
	name := strings.TrimSpace(log.UserEmail)
	if name == "" && log.UserID != nil {
		name = fmt.Sprintf("UID %d", *log.UserID)
	}
	return html.EscapeString(name)
}

// contentModerationBannedBanner is the "account is currently banned" callout.
// Inline-styled: it is the one element whose colour carries the message, so it
// has to survive clients that drop <style>.
const contentModerationBannedBanner = `
<p style="margin:24px 0 0;padding:16px 18px;border-radius:10px;background-color:#ef4444;color:#ffffff;font-size:16px;font-weight:700;line-height:1.6;text-align:center;">账户当前处于封禁状态，所有 API 请求将被拒绝</p>`

// contentModerationDetailRows renders the shared trigger-detail table under its
// own heading.
func contentModerationDetailRows(heading, timeLabel, timestamp string, log *ContentModerationLog, threshold int) string {
	return "\n<h2>" + heading + "</h2>" + emailKVTable(
		emailKVRow(timeLabel, html.EscapeString(timestamp))+
			emailKVRow("触发来源", "内容审核")+
			emailKVRow("所属分组", html.EscapeString(defaultContentModerationString(log.GroupName, "-")))+
			emailKVRow("命中类别 / 分数", fmt.Sprintf("%s / %.3f",
				html.EscapeString(defaultContentModerationString(log.HighestCategory, "-")), log.HighestScore))+
			emailKVRow("累计触发次数", fmt.Sprintf("%d 次（阈值 %d）", log.ViolationCount, threshold)))
}

func contentModerationBanThreshold(cfg *ContentModerationConfig) int {
	if cfg == nil || cfg.BanThreshold <= 0 {
		return defaultContentModerationBanThreshold
	}
	return cfg.BanThreshold
}

func buildContentModerationViolationEmailBody(siteName string, log *ContentModerationLog, cfg *ContentModerationConfig) string {
	if log == nil {
		return ""
	}
	content := `
<p>尊敬的用户 <strong>` + contentModerationEmailUser(log) + `</strong>，您的 API 请求在内容审计中触发平台风控策略。详情如下。</p>` +
		contentModerationDetailRows("触发详情", "触发时间", time.Now().Format("2006-01-02 15:04:05"), log, contentModerationBanThreshold(cfg))
	if log.AutoBanned {
		content += contentModerationBannedBanner
	}
	return emailLayout{
		Locale:  notificationEmailLocaleChinese,
		Accent:  "#ef4444",
		Eyebrow: "Risk control / 风控提醒",
		Title:   "账户触发内容审计规则",
		Content: content,
		Footer:  emailAutoSendFooter(notificationEmailLocaleChinese, html.EscapeString(siteName)),
	}.render()
}

func buildContentModerationAccountDisabledEmailBody(siteName string, log *ContentModerationLog, cfg *ContentModerationConfig) string {
	if log == nil {
		return ""
	}
	content := `
<p>尊敬的用户 <strong>` + contentModerationEmailUser(log) + `</strong>，您的账户在计数周期内多次触发平台风控策略，系统已自动禁用该账户。详情如下。</p>` +
		contentModerationDetailRows("封禁详情", "封禁时间", time.Now().Format("2006-01-02 15:04:05"), log, contentModerationBanThreshold(cfg)) +
		contentModerationBannedBanner + `
<p style="margin-top:24px;">如需申诉或恢复账号，请联系平台管理员处理。</p>`
	return emailLayout{
		Locale:  notificationEmailLocaleChinese,
		Accent:  "#b91c1c",
		Eyebrow: "Risk control / 账户封禁",
		Title:   "账户已被自动禁用",
		Content: content,
		Footer:  emailAutoSendFooter(notificationEmailLocaleChinese, html.EscapeString(siteName)),
	}.render()
}

func defaultContentModerationString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

// buildCyberPolicyNoticeEmailBody 是 cyber_policy 通知邮件的内置兜底正文，
// 当 notification email 模板渲染失败时使用（与 sendViolationEmail 的兜底同理）。
func buildCyberPolicyNoticeEmailBody(siteName string, log *ContentModerationLog) string {
	if log == nil {
		return ""
	}
	content := `
<p>尊敬的用户 <strong>` + contentModerationEmailUser(log) + `</strong>，您的请求被上游网络安全策略（cyber policy）拦截。</p>` +
		emailKVTable(
			emailKVRow("触发时间", html.EscapeString(log.CreatedAt.Format("2006-01-02 15:04:05")))+
				emailKVRow("模型", html.EscapeString(defaultContentModerationString(log.Model, "-")))+
				// The upstream blob can be one unbroken token, so it needs break-all.
				emailKVRowUnbroken("上游说明", html.EscapeString(defaultContentModerationString(log.Error, "-")))) + `
<p>如认为系误判，可调整请求措辞后重试，或申请获得授权的安全访问权限。</p>`
	return emailLayout{
		Locale:  notificationEmailLocaleChinese,
		Accent:  "#ef4444",
		Eyebrow: "Risk control / 网络安全策略",
		Title:   "请求被网络安全策略拦截",
		Content: content,
		Footer:  emailAutoSendFooter(notificationEmailLocaleChinese, html.EscapeString(siteName)),
	}.render()
}
