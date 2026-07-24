//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type capVerifierSpy struct {
	called    int
	endpoint  string
	siteKey   string
	secretKey string
	lastToken string
	result    *CapVerifyResponse
	err       error
}

func (s *capVerifierSpy) VerifyToken(_ context.Context, endpoint, siteKey, secretKey, token string) (*CapVerifyResponse, error) {
	s.called++
	s.endpoint = endpoint
	s.siteKey = siteKey
	s.secretKey = secretKey
	s.lastToken = token
	if s.err != nil {
		return nil, s.err
	}
	if s.result != nil {
		return s.result, nil
	}
	return &CapVerifyResponse{Success: true}, nil
}

func newAuthServiceForCapTest(settings map[string]string, capVerifier *capVerifierSpy, turnstileVerifier *turnstileVerifierSpy) *AuthService {
	authService := newAuthServiceForRegisterTurnstileTest(settings, turnstileVerifier)
	authService.capService = NewCapService(authService.settingService, capVerifier)
	return authService
}

func TestAuthService_VerifyCaptcha_usesCAPWhenProviderSelected(t *testing.T) {
	// Given
	capSpy := &capVerifierSpy{}
	turnstileSpy := &turnstileVerifierSpy{}
	authService := newAuthServiceForCapTest(map[string]string{
		SettingKeyCaptchaProvider: "cap",
		SettingKeyCapAPIEndpoint:  "https://cap.example.com",
		SettingKeyCapSiteKey:      "site-key",
		SettingKeyCapSecretKey:    "site-secret",
	}, capSpy, turnstileSpy)

	// When
	err := authService.VerifyCaptcha(context.Background(), "cap-token", "turnstile-token", "127.0.0.1")

	// Then
	require.NoError(t, err)
	require.Equal(t, 1, capSpy.called)
	require.Equal(t, "https://cap.example.com", capSpy.endpoint)
	require.Equal(t, "site-key", capSpy.siteKey)
	require.Equal(t, "site-secret", capSpy.secretKey)
	require.Equal(t, "cap-token", capSpy.lastToken)
	require.Equal(t, 0, turnstileSpy.called)
}

func TestAuthService_VerifyCaptcha_rejectsCAPWhenStandaloneRejectsToken(t *testing.T) {
	// Given
	capSpy := &capVerifierSpy{result: &CapVerifyResponse{Success: false}}
	turnstileSpy := &turnstileVerifierSpy{}
	authService := newAuthServiceForCapTest(map[string]string{
		SettingKeyCaptchaProvider: "cap",
		SettingKeyCapAPIEndpoint:  "https://cap.example.com",
		SettingKeyCapSiteKey:      "site-key",
		SettingKeyCapSecretKey:    "site-secret",
	}, capSpy, turnstileSpy)

	// When
	err := authService.VerifyCaptcha(context.Background(), "replayed-cap-token", "", "127.0.0.1")

	// Then
	require.ErrorIs(t, err, ErrCapVerificationFailed)
	require.Equal(t, 1, capSpy.called)
	require.Equal(t, 0, turnstileSpy.called)
}

func TestAuthService_VerifyCaptcha_usesTurnstileWhenLegacyProviderIsConfigured(t *testing.T) {
	// Given
	capSpy := &capVerifierSpy{}
	turnstileSpy := &turnstileVerifierSpy{}
	authService := newAuthServiceForCapTest(map[string]string{
		SettingKeyTurnstileEnabled:   "true",
		SettingKeyTurnstileSecretKey: "turnstile-secret",
	}, capSpy, turnstileSpy)

	// When
	err := authService.VerifyCaptcha(context.Background(), "", "turnstile-token", "127.0.0.1")

	// Then
	require.NoError(t, err)
	require.Equal(t, 0, capSpy.called)
	require.Equal(t, 1, turnstileSpy.called)
	require.Equal(t, "turnstile-token", turnstileSpy.lastToken)
}

func TestAuthService_VerifyCaptchaForRegister_skipsDuplicateCAPWhenEmailVerificationCodeExists(t *testing.T) {
	// Given
	capSpy := &capVerifierSpy{}
	turnstileSpy := &turnstileVerifierSpy{}
	authService := newAuthServiceForCapTest(map[string]string{
		SettingKeyCaptchaProvider:    "cap",
		SettingKeyCapAPIEndpoint:     "https://cap.example.com",
		SettingKeyCapSiteKey:         "site-key",
		SettingKeyCapSecretKey:       "site-secret",
		SettingKeyEmailVerifyEnabled: "true",
	}, capSpy, turnstileSpy)

	// When
	err := authService.VerifyCaptchaForRegister(context.Background(), "", "", "127.0.0.1", "123456")

	// Then
	require.NoError(t, err)
	require.Equal(t, 0, capSpy.called)
	require.Equal(t, 0, turnstileSpy.called)
}
