package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrCapVerificationFailed = infraerrors.BadRequest("CAP_VERIFICATION_FAILED", "cap verification failed")
	ErrCapNotConfigured      = infraerrors.ServiceUnavailable("CAP_NOT_CONFIGURED", "cap not configured")
	ErrCaptchaNotConfigured  = infraerrors.ServiceUnavailable("CAPTCHA_NOT_CONFIGURED", "captcha not configured")
)

// CaptchaProvider identifies the configured human-verification service.
type CaptchaProvider string

const (
	CaptchaProviderNone      CaptchaProvider = "none"
	CaptchaProviderTurnstile CaptchaProvider = "turnstile"
	CaptchaProviderCAP       CaptchaProvider = "cap"
)

// ParseCaptchaProvider returns the supported provider and whether the input was valid.
func ParseCaptchaProvider(raw string) (CaptchaProvider, bool) {
	switch CaptchaProvider(strings.ToLower(strings.TrimSpace(raw))) {
	case CaptchaProviderNone:
		return CaptchaProviderNone, true
	case CaptchaProviderTurnstile:
		return CaptchaProviderTurnstile, true
	case CaptchaProviderCAP:
		return CaptchaProviderCAP, true
	default:
		return CaptchaProviderNone, false
	}
}

func captchaProviderFromSettings(settings map[string]string) CaptchaProvider {
	if provider, valid := ParseCaptchaProvider(settings[SettingKeyCaptchaProvider]); valid && strings.TrimSpace(settings[SettingKeyCaptchaProvider]) != "" {
		return provider
	}
	if settings[SettingKeyTurnstileEnabled] == "true" {
		return CaptchaProviderTurnstile
	}
	return CaptchaProviderNone
}

// CapService verifies one-time proof-of-work tokens against Cap Standalone.
type CapService struct {
	settingService *SettingService
	verifier       CapVerifier
}

func NewCapService(settingService *SettingService, verifier CapVerifier) *CapService {
	return &CapService{
		settingService: settingService,
		verifier:       verifier,
	}
}

func (s *CapService) VerifyToken(ctx context.Context, token string) error {
	if s == nil || s.settingService == nil || s.verifier == nil {
		return ErrCapNotConfigured
	}

	apiEndpoint := s.settingService.GetCapAPIEndpoint(ctx)
	siteKey := s.settingService.GetCapSiteKey(ctx)
	secretKey := s.settingService.GetCapSecretKey(ctx)
	if apiEndpoint == "" || siteKey == "" || secretKey == "" {
		logger.LegacyPrintf("service.cap", "%s", "[CAP] Standalone endpoint, site key, or secret key is not configured")
		return ErrCapNotConfigured
	}
	if strings.TrimSpace(token) == "" {
		logger.LegacyPrintf("service.cap", "%s", "[CAP] Token is empty")
		return ErrCapVerificationFailed
	}

	result, err := s.verifier.VerifyToken(ctx, apiEndpoint, siteKey, secretKey, token)
	if err != nil {
		logger.LegacyPrintf("service.cap", "[CAP] Siteverify request failed: %v", err)
		return fmt.Errorf("verify CAP token: %w", err)
	}
	if result == nil || !result.Success {
		logger.LegacyPrintf("service.cap", "%s", "[CAP] Verification failed")
		return ErrCapVerificationFailed
	}

	return nil
}
