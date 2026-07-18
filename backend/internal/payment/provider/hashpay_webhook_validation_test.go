//go:build unit

package provider

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHashPayVerifyNotificationRejectsWrongMerchantAndReplay(t *testing.T) {
	// Given
	privateKey, privateKeyPEM := newHashPayTestKey(t)
	provider, err := NewHashPay("hashpay-1", map[string]string{
		"apiBase":    "https://hashpay.example",
		"currency":   "USD",
		"merchantId": "merchant-1",
		"privateKey": privateKeyPEM,
	})
	require.NoError(t, err)

	now := time.Now().Unix()
	rawBody := encryptHashPayCallback(t, &privateKey.PublicKey, map[string]any{
		"timestamp": now,
		"payload": map[string]any{
			"amount":     12.34,
			"currency":   "USD",
			"merchantNo": "sub2_order",
			"orderId":    "hp_123",
			"status":     "paid",
		},
	})

	tests := []struct {
		name    string
		headers map[string]string
		wantErr string
	}{
		{
			name: "wrong merchant",
			headers: map[string]string{
				"x-hashpay-encryption": "RSA-OAEP-256+A256GCM",
				"x-hashpay-merchant":   "merchant-other",
				"x-hashpay-timestamp":  strconv.FormatInt(now, 10),
			},
			wantErr: "merchant",
		},
		{
			name: "replayed callback",
			headers: map[string]string{
				"x-hashpay-encryption": "RSA-OAEP-256+A256GCM",
				"x-hashpay-merchant":   "merchant-1",
				"x-hashpay-timestamp":  strconv.FormatInt(now-301, 10),
			},
			wantErr: "timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := provider.VerifyNotification(context.Background(), rawBody, tt.headers)

			// Then
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
