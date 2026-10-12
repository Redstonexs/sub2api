package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyErrorResponse(t *testing.T) {
	tests := []struct {
		cfg         *config.Config
		name        string
		err         error
		slotType    string
		wantStatus  int
		wantType    string
		wantCode    string
		wantMessage string
	}{
		{
			name:        "true concurrency timeout remains rate limit",
			cfg:         nil,
			err:         &ConcurrencyError{SlotType: "account", IsTimeout: true},
			slotType:    "user",
			wantStatus:  http.StatusTooManyRequests,
			wantType:    "rate_limit_error",
			wantCode:    gatewayConcurrencyLimitCode,
			wantMessage: "Concurrency limit exceeded for account, please retry later",
		},
		{
			name:        "full local wait queue has gateway code",
			cfg:         nil,
			err:         &WaitQueueFullError{SlotType: "account"},
			slotType:    "account",
			wantStatus:  http.StatusTooManyRequests,
			wantType:    "rate_limit_error",
			wantCode:    gatewayQueueFullCode,
			wantMessage: "Too many pending requests, please retry later",
		},
		{
			name:        "client cancellation is not classified as concurrency limit",
			cfg:         nil,
			err:         context.Canceled,
			slotType:    "user",
			wantStatus:  statusClientClosedRequest,
			wantType:    "api_error",
			wantMessage: "context canceled",
		},
		{
			name:        "deadline exceeded is service unavailable",
			cfg:         nil,
			err:         context.DeadlineExceeded,
			slotType:    "user",
			wantStatus:  http.StatusServiceUnavailable,
			wantType:    "api_error",
			wantMessage: "Service temporarily unavailable, please retry later",
		},
		{
			name:        "redis acquire error is service unavailable",
			cfg:         nil,
			err:         errors.New("redis unavailable"),
			slotType:    "user",
			wantStatus:  http.StatusServiceUnavailable,
			wantType:    "api_error",
			wantMessage: "Service temporarily unavailable, please retry later",
		},
		{
			name: "custom 429 message overrides wait queue full",
			cfg: &config.Config{
				Gateway: config.GatewayConfig{
					ErrorMessages: map[string]string{
						"429": "Custom 429 message",
						"503": "Custom 503 message",
					},
				},
			},
			err:         &WaitQueueFullError{},
			slotType:    "user",
			wantStatus:  http.StatusTooManyRequests,
			wantType:    "rate_limit_error",
			wantCode:    gatewayQueueFullCode,
			wantMessage: "Custom 429 message",
		},
		{
			name: "custom 429 message overrides account concurrency error",
			cfg: &config.Config{
				Gateway: config.GatewayConfig{
					ErrorMessages: map[string]string{
						"429": "Custom 429 message",
						"503": "Custom 503 message",
					},
				},
			},
			err:         &ConcurrencyError{SlotType: "account"},
			slotType:    "user",
			wantStatus:  http.StatusTooManyRequests,
			wantType:    "rate_limit_error",
			wantCode:    gatewayConcurrencyLimitCode,
			wantMessage: "Custom 429 message",
		},
		{
			name: "custom 503 message overrides generic redis error",
			cfg: &config.Config{
				Gateway: config.GatewayConfig{
					ErrorMessages: map[string]string{
						"429": "Custom 429 message",
						"503": "Custom 503 message",
					},
				},
			},
			err:         errors.New("redis unavailable"),
			slotType:    "user",
			wantStatus:  http.StatusServiceUnavailable,
			wantType:    "api_error",
			wantMessage: "Custom 503 message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, errType, code, message := concurrencyErrorResponse(tt.cfg, tt.err, tt.slotType)
			require.Equal(t, tt.wantStatus, status)
			require.Equal(t, tt.wantType, errType)
			require.Equal(t, tt.wantCode, code)
			require.Equal(t, tt.wantMessage, message)
		})
	}
}

func TestGoogleConcurrencyError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantGStatus string
		wantMessage string
	}{
		{
			name:        "client cancellation is 499",
			err:         context.Canceled,
			wantStatus:  statusClientClosedRequest,
			wantGStatus: "CANCELLED",
			wantMessage: "context canceled",
		},
		{
			name:        "full wait queue is 429",
			err:         &WaitQueueFullError{SlotType: "user"},
			wantStatus:  http.StatusTooManyRequests,
			wantGStatus: "RESOURCE_EXHAUSTED",
			wantMessage: "Too many pending requests, please retry later",
		},
		{
			name:        "slot wait timeout is 429",
			err:         &ConcurrencyError{SlotType: "user", IsTimeout: true},
			wantStatus:  http.StatusTooManyRequests,
			wantGStatus: "RESOURCE_EXHAUSTED",
			wantMessage: "Concurrency limit exceeded for user, please retry later",
		},
		{
			name:        "acquire backend error is 503",
			err:         errors.New("redis unavailable"),
			wantStatus:  http.StatusServiceUnavailable,
			wantGStatus: "INTERNAL",
			wantMessage: "Service temporarily unavailable, please retry later",
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			// cfg 为 nil 时沿用内置文案，与其它入口的默认行为一致。
			(&GatewayHandler{}).googleConcurrencyError(c, tt.err, "user")

			require.Equal(t, tt.wantStatus, rec.Code)
			var body struct {
				Error struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Status  string `json:"status"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tt.wantStatus, body.Error.Code)
			require.Equal(t, tt.wantGStatus, body.Error.Status)
			require.Equal(t, tt.wantMessage, body.Error.Message)
		})
	}
}
