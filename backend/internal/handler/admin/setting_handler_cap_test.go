//go:build unit

package admin

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSettingHandler_UpdateSettings_PersistsCAPConfiguration(t *testing.T) {
	handler, repository := newStepUpSwitchTestHandler(t, map[string]string{})

	recorder := doUpdateSettings(t, handler, map[string]any{
		"captcha_provider": "cap",
		"cap_api_endpoint": "https://cap.example.com",
		"cap_site_key":     "public-site-key",
		"cap_secret_key":   "server-secret",
	}, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "cap", repository.values[service.SettingKeyCaptchaProvider])
	require.Equal(t, "https://cap.example.com", repository.values[service.SettingKeyCapAPIEndpoint])
	require.Equal(t, "public-site-key", repository.values[service.SettingKeyCapSiteKey])
	require.Equal(t, "server-secret", repository.values[service.SettingKeyCapSecretKey])
	require.NotContains(t, recorder.Body.String(), "server-secret")
}
