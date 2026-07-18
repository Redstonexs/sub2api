//go:build unit

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestHashPayVerifyNotificationDecryptsAndConfirmsPaidOrder(t *testing.T) {
	// Given
	privateKey, privateKeyPEM := newHashPayTestKey(t)
	var queryCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryCalled.Store(true)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/order/hp_123", r.URL.Path)
		verifyHashPayRequestSignature(t, &privateKey.PublicKey, r.Method, r.URL.RequestURI(), r.Header.Get("X-Timestamp"), nil, r.Header.Get("X-Signature"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"hp_123","merchantNo":"sub2_order","amount":12.34,"currency":"USD","status":"paid"}`))
	}))
	defer server.Close()

	provider, err := NewHashPay("hashpay-1", map[string]string{
		"apiBase":    server.URL,
		"currency":   "USD",
		"merchantId": "merchant-1",
		"privateKey": privateKeyPEM,
	})
	require.NoError(t, err)
	provider.httpClient = server.Client()

	now := time.Now().Unix()
	rawBody := encryptHashPayCallback(t, &privateKey.PublicKey, map[string]any{
		"timestamp": now,
		"payload": map[string]any{
			"amount":     12.34,
			"currency":   "USD",
			"merchantNo": "sub2_order",
			"orderId":    "hp_123",
			"payment":    map[string]string{"network": "TRC20"},
			"status":     "paid",
		},
	})
	headers := map[string]string{
		"x-hashpay-encryption": "RSA-OAEP-256+A256GCM",
		"x-hashpay-merchant":   "merchant-1",
		"x-hashpay-timestamp":  strconv.FormatInt(now, 10),
	}

	// When
	notification, err := provider.VerifyNotification(context.Background(), rawBody, headers)

	// Then
	require.NoError(t, err)
	require.NotNil(t, notification)
	require.Equal(t, "hp_123", notification.TradeNo)
	require.Equal(t, "sub2_order", notification.OrderID)
	require.InDelta(t, 12.34, notification.Amount, 0.0001)
	require.Equal(t, payment.NotificationStatusSuccess, notification.Status)
	require.Equal(t, "USD", notification.Metadata["currency"])
	require.Equal(t, "merchant-1", notification.Metadata["merchant_id"])
	require.Equal(t, "sub2_order", notification.Metadata["merchant_no"])
	require.True(t, queryCalled.Load(), "HashPay callback must be confirmed by the signed merchant API")
}

func TestHashPayVerifyNotificationRejectsCallbackThatDisagreesWithMerchantAPI(t *testing.T) {
	privateKey, privateKeyPEM := newHashPayTestKey(t)
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
	headers := map[string]string{
		"x-hashpay-encryption": "RSA-OAEP-256+A256GCM",
		"x-hashpay-merchant":   "merchant-1",
		"x-hashpay-timestamp":  strconv.FormatInt(now, 10),
	}

	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "merchant order does not match",
			body:    `{"id":"hp_123","merchantNo":"sub2_other","amount":12.34,"currency":"USD","status":"paid"}`,
			wantErr: "merchantNo mismatch",
		},
		{
			name:    "amount does not match",
			body:    `{"id":"hp_123","merchantNo":"sub2_order","amount":99.99,"currency":"USD","status":"paid"}`,
			wantErr: "amount mismatch",
		},
		{
			name:    "currency does not match",
			body:    `{"id":"hp_123","merchantNo":"sub2_order","amount":12.34,"currency":"EUR","status":"paid"}`,
			wantErr: "currency mismatch",
		},
		{
			name:    "merchant order is not paid",
			body:    `{"id":"hp_123","merchantNo":"sub2_order","amount":12.34,"currency":"USD","status":"pending"}`,
			wantErr: "status is not paid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/order/hp_123", r.URL.Path)
				w.Header().Set("content-type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			provider, err := NewHashPay("hashpay-1", map[string]string{
				"apiBase":    server.URL,
				"currency":   "USD",
				"merchantId": "merchant-1",
				"privateKey": privateKeyPEM,
			})
			require.NoError(t, err)
			provider.httpClient = server.Client()

			_, err = provider.VerifyNotification(context.Background(), rawBody, headers)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestHashPayVerifyNotificationRejectsMissingCallbackCurrency(t *testing.T) {
	privateKey, privateKeyPEM := newHashPayTestKey(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/order/hp_123", r.URL.Path)
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"hp_123","merchantNo":"sub2_order","amount":12.34,"currency":"CNY","status":"paid"}`))
	}))
	defer server.Close()

	provider, err := NewHashPay("hashpay-1", map[string]string{
		"apiBase":    server.URL,
		"currency":   "CNY",
		"merchantId": "merchant-1",
		"privateKey": privateKeyPEM,
	})
	require.NoError(t, err)
	provider.httpClient = server.Client()

	now := time.Now().Unix()
	rawBody := encryptHashPayCallback(t, &privateKey.PublicKey, map[string]any{
		"timestamp": now,
		"payload": map[string]any{
			"amount":     12.34,
			"currency":   "",
			"merchantNo": "sub2_order",
			"orderId":    "hp_123",
			"status":     "paid",
		},
	})

	_, err = provider.VerifyNotification(context.Background(), rawBody, map[string]string{
		"x-hashpay-encryption": "RSA-OAEP-256+A256GCM",
		"x-hashpay-merchant":   "merchant-1",
		"x-hashpay-timestamp":  strconv.FormatInt(now, 10),
	})
	require.ErrorContains(t, err, "missing currency")
}
