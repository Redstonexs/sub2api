package handler

import (
	"html"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingHandler 公开设置处理器（无需认证）
type SettingHandler struct {
	settingService           *service.SettingService
	notificationEmailService *service.NotificationEmailService
	version                  string
}

// NewSettingHandler 创建公开设置处理器
func NewSettingHandler(settingService *service.SettingService, version string) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
		version:        version,
	}
}

// SetNotificationEmailService attaches the public notification email service without
// changing the constructor signature used by existing tests.
func (h *SettingHandler) SetNotificationEmailService(notificationEmailService *service.NotificationEmailService) {
	h.notificationEmailService = notificationEmailService
}

// GetPublicSettings 获取公开设置
// GET /api/v1/settings/public
func (h *SettingHandler) GetPublicSettings(c *gin.Context) {
	settings, err := h.settingService.GetPublicSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.PublicSettings{
		RegistrationEnabled:                 settings.RegistrationEnabled,
		EmailVerifyEnabled:                  settings.EmailVerifyEnabled,
		ForceEmailOnThirdPartySignup:        settings.ForceEmailOnThirdPartySignup,
		RegistrationEmailSuffixWhitelist:    settings.RegistrationEmailSuffixWhitelist,
		RegistrationEmailDomainQuotaEnabled: settings.RegistrationEmailDomainQuotaEnabled,
		PromoCodeEnabled:                    settings.PromoCodeEnabled,
		PasswordResetEnabled:                settings.PasswordResetEnabled,
		InvitationCodeEnabled:               settings.InvitationCodeEnabled,
		TotpEnabled:                         settings.TotpEnabled,
		PasskeyEnabled:                      settings.PasskeyEnabled,
		LoginAgreementEnabled:               settings.LoginAgreementEnabled,
		LoginAgreementMode:                  settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:             settings.LoginAgreementUpdatedAt,
		LoginAgreementRevision:              settings.LoginAgreementRevision,
		LoginAgreementDocuments:             publicLoginAgreementDocumentsToDTO(settings.LoginAgreementDocuments),
		TurnstileEnabled:                    settings.TurnstileEnabled,
		TurnstileSiteKey:                    settings.TurnstileSiteKey,
		CaptchaProvider:                     string(settings.CaptchaProvider),
		CapAPIEndpoint:                      settings.CapAPIEndpoint,
		CapSiteKey:                          settings.CapSiteKey,
		TencentCaptchaEnabled:               settings.TencentCaptchaEnabled,
		TencentCaptchaAppID:                 settings.TencentCaptchaAppID,
		TencentCaptchaRegion:                settings.TencentCaptchaRegion,
		AliyunCaptchaEnabled:                settings.AliyunCaptchaEnabled,
		AliyunCaptchaSceneID:                settings.AliyunCaptchaSceneID,
		AliyunCaptchaPrefix:                 settings.AliyunCaptchaPrefix,
		AliyunCaptchaRegion:                 settings.AliyunCaptchaRegion,
		SiteName:                            settings.SiteName,
		SiteLogo:                            settings.SiteLogo,
		SiteSubtitle:                        settings.SiteSubtitle,
		APIBaseURL:                          settings.APIBaseURL,
		ContactInfo:                         settings.ContactInfo,
		DocURL:                              settings.DocURL,
		HomeContent:                         settings.HomeContent,
		CompactHomeEnabled:                  settings.CompactHomeEnabled,
		HideCcsImportButton:                 settings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:         settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:             settings.PurchaseSubscriptionURL,
		TableDefaultPageSize:                settings.TableDefaultPageSize,
		TablePageSizeOptions:                settings.TablePageSizeOptions,
		CustomMenuItems:                     dto.ParseUserVisibleMenuItems(settings.CustomMenuItems),
		CustomEndpoints:                     dto.ParseCustomEndpoints(settings.CustomEndpoints),
		DingTalkOAuthEnabled:                settings.DingTalkOAuthEnabled,
		LinuxDoOAuthEnabled:                 settings.LinuxDoOAuthEnabled,
		WeChatOAuthEnabled:                  settings.WeChatOAuthEnabled,
		WeChatOAuthOpenEnabled:              settings.WeChatOAuthOpenEnabled,
		WeChatOAuthMPEnabled:                settings.WeChatOAuthMPEnabled,
		WeChatOAuthMobileEnabled:            settings.WeChatOAuthMobileEnabled,
		OIDCOAuthEnabled:                    settings.OIDCOAuthEnabled,
		OIDCOAuthProviderName:               settings.OIDCOAuthProviderName,
		GitHubOAuthEnabled:                  settings.GitHubOAuthEnabled,
		GoogleOAuthEnabled:                  settings.GoogleOAuthEnabled,
		BackendModeEnabled:                  settings.BackendModeEnabled,
		PaymentEnabled:                      settings.PaymentEnabled,
		Version:                             h.version,
		ServerTimezone:                      timezone.Name(),
		ServerUTCOffset:                     timezone.UTCOffset(),
		BalanceLowNotifyEnabled:             settings.BalanceLowNotifyEnabled,
		AccountQuotaNotifyEnabled:           settings.AccountQuotaNotifyEnabled,
		BalanceLowNotifyThreshold:           settings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:         settings.BalanceLowNotifyRechargeURL,

		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorMode:                   settings.ChannelMonitorMode,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,
		ChannelMonitorHideThroughput:         settings.ChannelMonitorHideThroughput,

		AvailableChannelsEnabled: settings.AvailableChannelsEnabled,

		ModelPlazaEnabled:     settings.ModelPlazaEnabled,
		ModelPlazaRequireAuth: settings.ModelPlazaRequireAuth,

		AffiliateEnabled: settings.AffiliateEnabled,

		RiskControlEnabled: settings.RiskControlEnabled,

		AllowUserViewErrorRequests: settings.AllowUserViewErrorRequests,
	})
}

// UnsubscribeNotificationEmail handles optional notification email opt-outs.
// GET /api/v1/settings/email-unsubscribe?token=...
func (h *SettingHandler) UnsubscribeNotificationEmail(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		response.BadRequest(c, "token is required")
		return
	}
	result, err := h.notificationEmailService.Unsubscribe(c.Request.Context(), token)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	// This page is reached by tapping a link inside an email, so it is opened on a
	// phone more often than not, and by whoever received that email — which is why
	// it follows the recipient's stored locale rather than defaulting to English.
	locale := h.notificationEmailService.ResolveRecipientLocale(c.Request.Context(), 0, result.Email)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(unsubscribeConfirmationPage(locale, result)))
}

// unsubscribeConfirmationPage renders the opt-out confirmation shown after an
// unsubscribe link is followed.
func unsubscribeConfirmationPage(locale string, result service.NotificationEmailUnsubscribeResult) string {
	email := html.EscapeString(result.Email)
	event := html.EscapeString(service.LocalizedNotificationEmailEventLabel(result.Event, locale))

	lang, title, heading, body := "en", "Unsubscribed", "Unsubscribed",
		"<strong>"+email+"</strong> will no longer receive <strong>"+event+"</strong> emails."
	note := "Changed your mind? You can turn these emails back on from your notification settings."
	if service.IsChineseEmailLocale(locale) {
		lang, title, heading, body = "zh-CN", "已退订", "已退订",
			"<strong>"+email+"</strong> 将不再收到<strong>"+event+"</strong>。"
		note = "如需恢复接收，可在通知设置中重新开启。"
	}

	return `<!doctype html>
<html lang="` + lang + `">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + title + `</title>
</head>
<body style="margin:0;padding:24px 16px;background-color:#f4f4f5;color:#18181b;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC','Microsoft YaHei',Arial,sans-serif;line-height:1.7;">
<div style="max-width:520px;margin:0 auto;padding:28px 24px;background-color:#ffffff;border-radius:12px;">
<h1 style="margin:0 0 12px;font-size:22px;line-height:1.3;">` + heading + `</h1>
<p style="margin:0 0 12px;font-size:15px;word-break:break-word;">` + body + `</p>
<p style="margin:0;color:#71717a;font-size:13px;">` + note + `</p>
</div>
</body>
</html>`
}

func publicLoginAgreementDocumentsToDTO(items []service.LoginAgreementDocument) []dto.LoginAgreementDocument {
	result := make([]dto.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		result = append(result, dto.LoginAgreementDocument{
			ID:        item.ID,
			Title:     item.Title,
			ContentMD: item.ContentMD,
		})
	}
	return result
}
