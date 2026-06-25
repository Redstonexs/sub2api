package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrEmailNotConfigured    = infraerrors.ServiceUnavailable("EMAIL_NOT_CONFIGURED", "email service not configured")
	ErrInvalidVerifyCode     = infraerrors.BadRequest("INVALID_VERIFY_CODE", "invalid or expired verification code")
	ErrVerifyCodeTooFrequent = infraerrors.TooManyRequests("VERIFY_CODE_TOO_FREQUENT", "please wait before requesting a new code")
	ErrVerifyCodeMaxAttempts = infraerrors.TooManyRequests("VERIFY_CODE_MAX_ATTEMPTS", "too many failed attempts, please request a new code")

	// Password reset errors
	ErrInvalidResetToken = infraerrors.BadRequest("INVALID_RESET_TOKEN", "invalid or expired password reset token")
)

// EmailCache defines cache operations for email service
type EmailCache interface {
	GetVerificationCode(ctx context.Context, email string) (*VerificationCodeData, error)
	SetVerificationCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error
	DeleteVerificationCode(ctx context.Context, email string) error

	// Notify email verification code methods
	GetNotifyVerifyCode(ctx context.Context, email string) (*VerificationCodeData, error)
	SetNotifyVerifyCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error
	DeleteNotifyVerifyCode(ctx context.Context, email string) error

	// Password reset token methods
	GetPasswordResetToken(ctx context.Context, email string) (*PasswordResetTokenData, error)
	SetPasswordResetToken(ctx context.Context, email string, data *PasswordResetTokenData, ttl time.Duration) error
	DeletePasswordResetToken(ctx context.Context, email string) error

	// Password reset email cooldown methods
	// Returns true if in cooldown period (email was sent recently)
	IsPasswordResetEmailInCooldown(ctx context.Context, email string) bool
	SetPasswordResetEmailCooldown(ctx context.Context, email string, ttl time.Duration) error

	// Notify code rate limiting per user
	IncrNotifyCodeUserRate(ctx context.Context, userID int64, window time.Duration) (int64, error)
	GetNotifyCodeUserRate(ctx context.Context, userID int64) (int64, error)
}

// VerificationCodeData represents verification code data
type VerificationCodeData struct {
	Code      string
	Attempts  int
	CreatedAt time.Time
	ExpiresAt time.Time // absolute expiry; used to preserve remaining TTL when updating attempts
}

// PasswordResetTokenData represents password reset token data
type PasswordResetTokenData struct {
	Token     string
	CreatedAt time.Time
}

const (
	verifyCodeTTL         = 15 * time.Minute
	verifyCodeCooldown    = 1 * time.Minute
	maxVerifyCodeAttempts = 5

	// Password reset token settings
	passwordResetTokenTTL = 30 * time.Minute

	// Password reset email cooldown (prevent email bombing)
	passwordResetEmailCooldown = 30 * time.Second
)

// SMTPConfig SMTP配置
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
}

// EmailProvider 邮件发送渠道。API 渠道(resend/cyberpanel)通过 HTTPS(443) 发送，
// 可绕过 DigitalOcean 等云厂商对 SMTP 端口(25/465/587)的默认封锁。
type EmailProvider string

const (
	EmailProviderSMTP       EmailProvider = "smtp"
	EmailProviderResend     EmailProvider = "resend"
	EmailProviderCyberPanel EmailProvider = "cyberpanel"
)

// defaultResendBaseURL 是 Resend 官方 API 地址。
const defaultResendBaseURL = "https://api.resend.com"

// NormalizeEmailProvider 归一化渠道取值，未知/空值回退到 smtp。
func NormalizeEmailProvider(v string) string {
	switch EmailProvider(strings.ToLower(strings.TrimSpace(v))) {
	case EmailProviderResend:
		return string(EmailProviderResend)
	case EmailProviderCyberPanel:
		return string(EmailProviderCyberPanel)
	default:
		return string(EmailProviderSMTP)
	}
}

// EmailDeliveryConfig 是解析后的发送渠道配置（发件人身份 + 渠道凭据）。
type EmailDeliveryConfig struct {
	Provider EmailProvider
	From     string
	FromName string

	// API 渠道（resend / cyberpanel）
	APIBaseURL string
	APIKey     string

	// SMTP 渠道
	SMTP *SMTPConfig
}

// EmailService 邮件服务
type EmailService struct {
	settingRepo              SettingRepository
	cache                    EmailCache
	notificationEmailService *NotificationEmailService
}

// NewEmailService 创建邮件服务实例
func NewEmailService(settingRepo SettingRepository, cache EmailCache) *EmailService {
	return &EmailService{
		settingRepo: settingRepo,
		cache:       cache,
	}
}

func (s *EmailService) SetNotificationEmailService(notificationEmailService *NotificationEmailService) {
	s.notificationEmailService = notificationEmailService
}

func firstEmailLocale(locales []string) string {
	if len(locales) == 0 {
		return ""
	}
	return strings.TrimSpace(locales[0])
}

func emailRecipientName(email string) string {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return ""
	}
	if at := strings.Index(trimmed, "@"); at > 0 {
		return trimmed[:at]
	}
	return trimmed
}

// GetSMTPConfig 从数据库获取SMTP配置
func (s *EmailService) GetSMTPConfig(ctx context.Context) (*SMTPConfig, error) {
	keys := []string{
		SettingKeySMTPHost,
		SettingKeySMTPPort,
		SettingKeySMTPUsername,
		SettingKeySMTPPassword,
		SettingKeySMTPFrom,
		SettingKeySMTPFromName,
		SettingKeySMTPUseTLS,
	}

	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("get smtp settings: %w", err)
	}

	host := strings.TrimSpace(settings[SettingKeySMTPHost])
	if host == "" {
		return nil, ErrEmailNotConfigured
	}

	port := 587 // 默认端口
	if portStr := settings[SettingKeySMTPPort]; portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	useTLS := settings[SettingKeySMTPUseTLS] == "true"

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: strings.TrimSpace(settings[SettingKeySMTPUsername]),
		Password: strings.TrimSpace(settings[SettingKeySMTPPassword]),
		From:     strings.TrimSpace(settings[SettingKeySMTPFrom]),
		FromName: strings.TrimSpace(settings[SettingKeySMTPFromName]),
		UseTLS:   useTLS,
	}, nil
}

// ResolveProvider 返回当前配置的发送渠道（默认 smtp）。
func (s *EmailService) ResolveProvider(ctx context.Context) EmailProvider {
	v, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyEmailProvider})
	if err != nil {
		return EmailProviderSMTP
	}
	return EmailProvider(NormalizeEmailProvider(v[SettingKeyEmailProvider]))
}

// GetEmailDeliveryConfig 解析当前发送渠道及其凭据。
func (s *EmailService) GetEmailDeliveryConfig(ctx context.Context) (*EmailDeliveryConfig, error) {
	settings, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyEmailProvider,
		SettingKeyEmailAPIBaseURL,
		SettingKeyEmailAPIKey,
		SettingKeySMTPFrom,
		SettingKeySMTPFromName,
	})
	if err != nil {
		return nil, fmt.Errorf("get email settings: %w", err)
	}

	cfg := &EmailDeliveryConfig{
		Provider: EmailProvider(NormalizeEmailProvider(settings[SettingKeyEmailProvider])),
		From:     strings.TrimSpace(settings[SettingKeySMTPFrom]),
		FromName: strings.TrimSpace(settings[SettingKeySMTPFromName]),
	}

	switch cfg.Provider {
	case EmailProviderResend, EmailProviderCyberPanel:
		cfg.APIKey = strings.TrimSpace(settings[SettingKeyEmailAPIKey])
		cfg.APIBaseURL = strings.TrimRight(strings.TrimSpace(settings[SettingKeyEmailAPIBaseURL]), "/")
		if cfg.APIKey == "" || cfg.From == "" {
			return nil, ErrEmailNotConfigured
		}
		if cfg.APIBaseURL == "" {
			if cfg.Provider == EmailProviderResend {
				cfg.APIBaseURL = defaultResendBaseURL
			} else {
				// CyberPanel 必须提供自有域名的 API 地址。
				return nil, ErrEmailNotConfigured
			}
		}
	default:
		smtpCfg, err := s.GetSMTPConfig(ctx)
		if err != nil {
			return nil, err
		}
		cfg.SMTP = smtpCfg
		if cfg.From == "" {
			cfg.From = smtpCfg.From
		}
		if cfg.FromName == "" {
			cfg.FromName = smtpCfg.FromName
		}
	}
	return cfg, nil
}

// SendEmail 发送邮件（使用数据库中保存的配置，按渠道自动路由）。
func (s *EmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	cfg, err := s.GetEmailDeliveryConfig(ctx)
	if err != nil {
		return err
	}

	switch cfg.Provider {
	case EmailProviderResend:
		return s.sendViaResend(ctx, cfg, to, subject, body)
	case EmailProviderCyberPanel:
		return s.sendViaCyberPanel(ctx, cfg, to, subject, body)
	default:
		return s.SendEmailWithConfig(cfg.SMTP, to, subject, body)
	}
}

const smtpDialTimeout = 10 * time.Second
const smtpIOTimeout = 20 * time.Second

// SendEmailWithConfig 使用指定配置发送邮件
func (s *EmailService) SendEmailWithConfig(config *SMTPConfig, to, subject, body string) error {
	// Sanitize all SMTP header fields to prevent header injection (CR/LF removal).
	to = sanitizeEmailHeader(to)
	subject = sanitizeEmailHeader(subject)

	from := sanitizeEmailHeader(config.From)
	if config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", sanitizeEmailHeader(config.FromName), sanitizeEmailHeader(config.From))
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	auth := smtpAuth(config)

	client, err := dialSMTPClient(config)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	return smtpDeliver(client, auth, config.From, to, []byte(msg))
}

// smtpSecurity describes how the SMTP transport is encrypted.
type smtpSecurity int

const (
	// smtpSecuritySTARTTLS connects in cleartext and upgrades to TLS via the
	// STARTTLS command when the server advertises it (submission ports 587/25/2525).
	smtpSecuritySTARTTLS smtpSecurity = iota
	// smtpSecurityImplicitTLS performs the TLS handshake immediately on connect,
	// i.e. SMTPS (port 465).
	smtpSecurityImplicitTLS
)

func (m smtpSecurity) String() string {
	if m == smtpSecurityImplicitTLS {
		return "implicit TLS"
	}
	return "STARTTLS"
}

// resolveSMTPSecurity decides the transport security for a config.
//
// The well-known submission ports have unambiguous, standardized semantics
// (RFC 8314 / RFC 6409), so they are honored regardless of the UseTLS flag:
//
//	465              -> implicit TLS (never speaks cleartext)
//	25 / 587 / 2525  -> STARTTLS     (never speaks implicit TLS)
//
// Pinning these prevents the most common misconfiguration — e.g. enabling
// "Use TLS" on port 587 — which otherwise makes the client attempt a TLS
// handshake against a server waiting to send a plaintext greeting, hanging
// until the deadline fires and surfacing as an "i/o timeout". For any other
// (non-standard) port, the explicit UseTLS flag is honored.
func resolveSMTPSecurity(config *SMTPConfig) smtpSecurity {
	switch config.Port {
	case 465:
		return smtpSecurityImplicitTLS
	case 25, 587, 2525:
		return smtpSecuritySTARTTLS
	default:
		if config.UseTLS {
			return smtpSecurityImplicitTLS
		}
		return smtpSecuritySTARTTLS
	}
}

// smtpAuth builds the AUTH mechanism, or nil when no username is configured so
// that unauthenticated relays remain supported.
func smtpAuth(config *SMTPConfig) smtp.Auth {
	if strings.TrimSpace(config.Username) == "" {
		return nil
	}
	return smtp.PlainAuth("", config.Username, config.Password, config.Host)
}

// dialSMTPClient establishes an SMTP session with dial + I/O timeouts and, on
// the STARTTLS transport, upgrades the connection before returning. The caller
// owns the returned client and must Close it.
func dialSMTPClient(config *SMTPConfig) (*smtp.Client, error) {
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	security := resolveSMTPSecurity(config)
	if config.UseTLS != (security == smtpSecurityImplicitTLS) {
		slog.Warn("smtp: overriding UseTLS to match the standard semantics of the configured port",
			"port", config.Port, "use_tls", config.UseTLS, "resolved", security.String())
	}

	dialer := &net.Dialer{Timeout: smtpDialTimeout}

	var conn net.Conn
	var err error
	if security == smtpSecurityImplicitTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName: config.Host,
			MinVersion: tls.VersionTLS12, // 强制 TLS 1.2+，避免协议降级。
		})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return nil, smtpDialError(addr, security, err)
	}
	_ = conn.SetDeadline(time.Now().Add(smtpIOTimeout))

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("new smtp client for %s: %w", addr, err)
	}

	if security == smtpSecuritySTARTTLS {
		if err := startTLS(client, config.Host); err != nil {
			_ = client.Close()
			return nil, err
		}
	}
	return client, nil
}

// startTLS upgrades a cleartext SMTP connection when the server advertises the
// STARTTLS extension (the EHLO required to populate the capability list is sent
// implicitly by Extension). If STARTTLS is offered, the handshake must succeed —
// we fail loudly rather than silently continuing in cleartext.
func startTLS(client *smtp.Client, host string) error {
	ok, _ := client.Extension("STARTTLS")
	if !ok {
		return nil
	}
	if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("starttls upgrade with %s failed: %w", host, err)
	}
	return nil
}

// smtpDeliver runs the AUTH/MAIL/RCPT/DATA exchange over an established client.
func smtpDeliver(client *smtp.Client, auth smtp.Auth, from, to string, msg []byte) error {
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from %q: %w", from, err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to %q: %w", to, err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write message body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("complete message: %w", err)
	}
	// The message is committed once the Data writer closes; some servers emit a
	// non-standard reply to QUIT, so its error is not treated as a failure.
	_ = client.Quit()
	return nil
}

// smtpDialError annotates connection failures with actionable guidance. A
// timeout during connect (while ICMP/ping to the host still succeeds) almost
// always means the hosting provider blocks outbound SMTP: DigitalOcean — and
// most clouds (AWS, GCP, Oracle, Vultr...) — filter ports 25/465/587 by default
// to curb spam, and only the TCP SMTP ports are blocked, so ping is unaffected.
func smtpDialError(addr string, security smtpSecurity, err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("timed out establishing a %s connection to %s: the host is reachable but the SMTP port never accepted the connection. "+
			"This is typically the hosting provider blocking outbound SMTP (DigitalOcean and most clouds block 25/465/587 by default; ping/ICMP is unaffected). "+
			"Fixes: ask the provider to unblock SMTP, send via an email API/relay over port 443 or 2525, or confirm the port matches the encryption mode (587=STARTTLS, 465=TLS): %w",
			security, addr, err)
	}
	return fmt.Errorf("connect to %s (%s) failed: %w", addr, security, err)
}

const emailAPITimeout = 20 * time.Second

// emailHTTPClient 用于 API 渠道发送，带整体请求超时。
var emailHTTPClient = &http.Client{Timeout: emailAPITimeout}

// formatEmailFrom 组装符合 RFC 5322 的发件人字段（"Name <addr>" 或 "addr"）。
func formatEmailFrom(from, fromName string) string {
	from = sanitizeEmailHeader(from)
	if name := sanitizeEmailHeader(fromName); name != "" {
		return fmt.Sprintf("%s <%s>", name, from)
	}
	return from
}

// sendViaResend 通过 Resend API 发送：POST {base}/emails，收件人 to 为数组。
func (s *EmailService) sendViaResend(ctx context.Context, cfg *EmailDeliveryConfig, to, subject, body string) error {
	payload := map[string]any{
		"from":    formatEmailFrom(cfg.From, cfg.FromName),
		"to":      []string{sanitizeEmailHeader(to)},
		"subject": sanitizeEmailHeader(subject),
		"html":    body,
	}
	return s.postEmailAPI(ctx, string(EmailProviderResend), cfg.APIBaseURL+"/emails", cfg.APIKey, payload)
}

// sendViaCyberPanel 通过 CyberPanel API 发送：POST {base}/email/v1/send，收件人 to 为字符串。
func (s *EmailService) sendViaCyberPanel(ctx context.Context, cfg *EmailDeliveryConfig, to, subject, body string) error {
	payload := map[string]any{
		"from":    formatEmailFrom(cfg.From, cfg.FromName),
		"to":      sanitizeEmailHeader(to),
		"subject": sanitizeEmailHeader(subject),
		"html":    body,
	}
	return s.postEmailAPI(ctx, string(EmailProviderCyberPanel), cfg.APIBaseURL+"/email/v1/send", cfg.APIKey, payload)
}

// postEmailAPI 执行带 Bearer 鉴权的 JSON POST，并对非 2xx 响应给出可诊断的错误。
func (s *EmailService) postEmailAPI(ctx context.Context, provider, endpoint, apiKey string, payload map[string]any) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: marshal payload: %w", provider, err)
	}

	ctx, cancel := context.WithTimeout(ctx, emailAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("%s: build request: %w", provider, err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := emailHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: request to %s failed: %w", provider, endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s: send failed (HTTP %d): %s", provider, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// GenerateVerifyCode 生成6位数字验证码
func (s *EmailService) GenerateVerifyCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}
	return string(code), nil
}

// SendVerifyCode 发送验证码邮件
func (s *EmailService) SendVerifyCode(ctx context.Context, email, siteName string, locale ...string) error {
	// 检查是否在冷却期内
	existing, err := s.cache.GetVerificationCode(ctx, email)
	if err == nil && existing != nil {
		if time.Since(existing.CreatedAt) < verifyCodeCooldown {
			return ErrVerifyCodeTooFrequent
		}
	}

	// 生成验证码
	code, err := s.GenerateVerifyCode()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	// 保存验证码到 Redis
	data := &VerificationCodeData{
		Code:      code,
		Attempts:  0,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(verifyCodeTTL),
	}
	if err := s.cache.SetVerificationCode(ctx, email, data, verifyCodeTTL); err != nil {
		return fmt.Errorf("save verify code: %w", err)
	}

	if s.notificationEmailService != nil {
		err := s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
			Event:          NotificationEmailEventAuthVerifyCode,
			Locale:         firstEmailLocale(locale),
			RecipientEmail: email,
			RecipientName:  emailRecipientName(email),
			Variables: map[string]string{
				"verification_code":  code,
				"expires_in_minutes": strconv.Itoa(int(verifyCodeTTL / time.Minute)),
			},
		})
		if err == nil {
			return nil
		}
		if !shouldFallbackNotificationEmail(err) {
			return err
		}
		slog.Warn("failed to send templated verification email, falling back to legacy template", "recipient_hash", notificationEmailHash(email), "error", err)
	}

	// 构建邮件内容
	subject := fmt.Sprintf("[%s] Email Verification Code", siteName)
	body := s.buildVerifyCodeEmailBody(code, siteName)

	// 发送邮件
	if err := s.SendEmail(ctx, email, subject, body); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

// VerifyCode 验证验证码
func (s *EmailService) VerifyCode(ctx context.Context, email, code string) error {
	data, err := s.cache.GetVerificationCode(ctx, email)
	if err != nil || data == nil {
		return ErrInvalidVerifyCode
	}

	// 检查是否已达到最大尝试次数
	if data.Attempts >= maxVerifyCodeAttempts {
		return ErrVerifyCodeMaxAttempts
	}

	// 验证码不匹配 (constant-time comparison to prevent timing attacks)
	if subtle.ConstantTimeCompare([]byte(data.Code), []byte(code)) != 1 {
		data.Attempts++
		remaining := time.Until(data.ExpiresAt)
		if remaining <= 0 {
			return ErrInvalidVerifyCode
		}
		if err := s.cache.SetVerificationCode(ctx, email, data, remaining); err != nil {
			slog.Error("failed to update verification attempt count", "email", email, "error", err)
		}
		if data.Attempts >= maxVerifyCodeAttempts {
			return ErrVerifyCodeMaxAttempts
		}
		return ErrInvalidVerifyCode
	}

	// 验证成功，删除验证码
	if err := s.cache.DeleteVerificationCode(ctx, email); err != nil {
		slog.Error("failed to delete verification code after success", "email", email, "error", err)
	}
	return nil
}

// buildVerifyCodeEmailBody 构建验证码邮件HTML内容
func (s *EmailService) buildVerifyCodeEmailBody(code, siteName string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 40px 30px; text-align: center; }
        .code { font-size: 36px; font-weight: bold; letter-spacing: 8px; color: #333; background-color: #f8f9fa; padding: 20px 30px; border-radius: 8px; display: inline-block; margin: 20px 0; font-family: monospace; }
        .info { color: #666; font-size: 14px; line-height: 1.6; margin-top: 20px; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s</h1>
        </div>
        <div class="content">
            <p style="font-size: 18px; color: #333;">Your verification code is:</p>
            <div class="code">%s</div>
            <div class="info">
                <p>This code will expire in <strong>15 minutes</strong>.</p>
                <p>If you did not request this code, please ignore this email.</p>
            </div>
        </div>
        <div class="footer">
            <p>This is an automated message, please do not reply.</p>
        </div>
    </div>
</body>
</html>
`, siteName, code)
}

// TestSMTPConnectionWithConfig 使用指定配置测试SMTP连接。
// 复用与真实发送一致的连接逻辑（超时、端口语义、STARTTLS 升级、可选鉴权），
// 这样“测试连接”的结果才能真实反映发送时的行为。
func (s *EmailService) TestSMTPConnectionWithConfig(config *SMTPConfig) error {
	client, err := dialSMTPClient(config)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if auth := smtpAuth(config); auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp authentication failed: %w", err)
			}
		}
	}

	return client.Quit()
}

// GeneratePasswordResetToken generates a secure 32-byte random token (64 hex characters)
func (s *EmailService) GeneratePasswordResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SendPasswordResetEmail sends a password reset email with a reset link
func (s *EmailService) SendPasswordResetEmail(ctx context.Context, email, siteName, resetURL string, locale ...string) error {
	var token string
	var needSaveToken bool

	// Check if token already exists
	existing, err := s.cache.GetPasswordResetToken(ctx, email)
	if err == nil && existing != nil {
		// Token exists, reuse it (allows resending email without generating new token)
		token = existing.Token
		needSaveToken = false
	} else {
		// Generate new token
		token, err = s.GeneratePasswordResetToken()
		if err != nil {
			return fmt.Errorf("generate token: %w", err)
		}
		needSaveToken = true
	}

	// Save token to Redis (only if new token generated)
	if needSaveToken {
		data := &PasswordResetTokenData{
			Token:     token,
			CreatedAt: time.Now(),
		}
		if err := s.cache.SetPasswordResetToken(ctx, email, data, passwordResetTokenTTL); err != nil {
			return fmt.Errorf("save reset token: %w", err)
		}
	}

	// Build full reset URL with URL-encoded token and email
	fullResetURL := fmt.Sprintf("%s?email=%s&token=%s", resetURL, url.QueryEscape(email), url.QueryEscape(token))

	if s.notificationEmailService != nil {
		err := s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
			Event:          NotificationEmailEventAuthPasswordReset,
			Locale:         firstEmailLocale(locale),
			RecipientEmail: email,
			RecipientName:  emailRecipientName(email),
			Variables: map[string]string{
				"reset_url":          fullResetURL,
				"expires_in_minutes": strconv.Itoa(int(passwordResetTokenTTL / time.Minute)),
			},
		})
		if err == nil {
			return nil
		}
		if !shouldFallbackNotificationEmail(err) {
			return err
		}
		slog.Warn("failed to send templated password reset email, falling back to legacy template", "recipient_hash", notificationEmailHash(email), "error", err)
	}

	// Build email content
	subject := fmt.Sprintf("[%s] 密码重置请求", siteName)
	body := s.buildPasswordResetEmailBody(fullResetURL, siteName)

	// Send email
	if err := s.SendEmail(ctx, email, subject, body); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

// SendPasswordResetEmailWithCooldown sends password reset email with cooldown check (called by queue worker)
// This method wraps SendPasswordResetEmail with email cooldown to prevent email bombing
func (s *EmailService) SendPasswordResetEmailWithCooldown(ctx context.Context, email, siteName, resetURL string, locale ...string) error {
	// Check email cooldown to prevent email bombing
	if s.cache.IsPasswordResetEmailInCooldown(ctx, email) {
		slog.Info("password reset email skipped due to cooldown", "email", email)
		return nil // Silent success to prevent revealing cooldown to attackers
	}

	// Send email using core method
	if err := s.SendPasswordResetEmail(ctx, email, siteName, resetURL, firstEmailLocale(locale)); err != nil {
		return err
	}

	// Set cooldown marker (Redis TTL handles expiration)
	if err := s.cache.SetPasswordResetEmailCooldown(ctx, email, passwordResetEmailCooldown); err != nil {
		slog.Error("failed to set password reset cooldown", "email", email, "error", err)
	}

	return nil
}

// VerifyPasswordResetToken verifies the password reset token without consuming it
func (s *EmailService) VerifyPasswordResetToken(ctx context.Context, email, token string) error {
	data, err := s.cache.GetPasswordResetToken(ctx, email)
	if err != nil || data == nil {
		return ErrInvalidResetToken
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(data.Token), []byte(token)) != 1 {
		return ErrInvalidResetToken
	}

	return nil
}

// ConsumePasswordResetToken verifies and deletes the token (one-time use)
func (s *EmailService) ConsumePasswordResetToken(ctx context.Context, email, token string) error {
	// Verify first
	if err := s.VerifyPasswordResetToken(ctx, email, token); err != nil {
		return err
	}

	// Delete after verification (one-time use)
	if err := s.cache.DeletePasswordResetToken(ctx, email); err != nil {
		slog.Error("failed to delete password reset token after consumption", "email", email, "error", err)
	}
	return nil
}

// buildPasswordResetEmailBody builds the HTML content for password reset email
func (s *EmailService) buildPasswordResetEmailBody(resetURL, siteName string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 40px 30px; text-align: center; }
        .button { display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 14px 32px; text-decoration: none; border-radius: 8px; font-size: 16px; font-weight: 600; margin: 20px 0; }
        .button:hover { opacity: 0.9; }
        .info { color: #666; font-size: 14px; line-height: 1.6; margin-top: 20px; }
        .link-fallback { color: #666; font-size: 12px; word-break: break-all; margin-top: 20px; padding: 15px; background-color: #f8f9fa; border-radius: 4px; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
        .warning { color: #e74c3c; font-weight: 500; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s</h1>
        </div>
        <div class="content">
            <p style="font-size: 18px; color: #333;">密码重置请求</p>
            <p style="color: #666;">您已请求重置密码。请点击下方按钮设置新密码：</p>
            <a href="%s" class="button">重置密码</a>
            <div class="info">
                <p>此链接将在 <strong>30 分钟</strong>后失效。</p>
                <p class="warning">如果您没有请求重置密码，请忽略此邮件。您的密码将保持不变。</p>
            </div>
            <div class="link-fallback">
                <p>如果按钮无法点击，请复制以下链接到浏览器中打开：</p>
                <p>%s</p>
            </div>
        </div>
        <div class="footer">
            <p>这是一封自动发送的邮件，请勿回复。</p>
        </div>
    </div>
</body>
</html>
`, siteName, resetURL, resetURL)
}
