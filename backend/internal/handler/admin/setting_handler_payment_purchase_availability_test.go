//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingHandler_UpdateSettings_PersistsIndependentPurchaseAvailability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Given
	repository := &settingHandlerRepoStub{values: map[string]string{}}
	settingService := service.NewSettingService(repository, &config.Config{
		Default: config.DefaultConfig{UserConcurrency: 5},
	})
	paymentConfigService := service.NewPaymentConfigService(nil, repository, nil)
	handler := NewSettingHandler(settingService, nil, nil, nil, paymentConfigService, nil, nil)
	body := []byte(`{"payment_balance_purchase_enabled":false,"payment_subscription_purchase_enabled":true}`)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")

	// When
	handler.UpdateSettings(context)

	// Then
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "false", repository.values[service.SettingBalancePurchaseEnabled])
	require.Equal(t, "true", repository.values[service.SettingSubscriptionPurchaseEnabled])

	var result response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	data, ok := result.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, data["payment_balance_purchase_enabled"])
	require.Equal(t, true, data["payment_subscription_purchase_enabled"])
}
