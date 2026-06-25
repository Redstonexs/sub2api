package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
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
		wantMessage string
	}{
		{
			name:        "true concurrency timeout remains rate limit",
			cfg:         nil,
			err:         &ConcurrencyError{SlotType: "account", IsTimeout: true},
			slotType:    "user",
			wantStatus:  http.StatusTooManyRequests,
			wantType:    "rate_limit_error",
			wantMessage: "Concurrency limit exceeded for account, please retry later",
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
			status, errType, message := concurrencyErrorResponse(tt.cfg, tt.err, tt.slotType)
			require.Equal(t, tt.wantStatus, status)
			require.Equal(t, tt.wantType, errType)
			require.Equal(t, tt.wantMessage, message)
		})
	}
}
