//go:build unit

package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestHashPayCreatePaymentSignsExactPayload(t *testing.T) {
	// Given
	privateKey, privateKeyPEM := newHashPayTestKey(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/merchant/new", r.URL.Path)
		require.Equal(t, "merchant-1", r.Header.Get("X-Merchant-Id"))

		timestamp, err := strconv.ParseInt(r.Header.Get("X-Timestamp"), 10, 64)
		require.NoError(t, err)
		require.WithinDuration(t, time.Now(), time.Unix(timestamp, 0), 5*time.Second)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, `{"merchantNo":"sub2_order","amount":12.34,"currency":"USD","description":"Subscription","return_url":"https://app.example.com/payment/result"}`, string(body))
		verifyHashPayRequestSignature(t, &privateKey.PublicKey, r.Method, r.URL.RequestURI(), r.Header.Get("X-Timestamp"), body, r.Header.Get("X-Signature"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"checkoutUrl":"https://hashpay.example/pay/hp_123","order":{"id":"hp_123","status":"pending","amount":12.34,"currency":"USD"},"reused":false}`))
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

	// When
	response, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		Amount:    "12.34",
		OrderID:   "sub2_order",
		ReturnURL: "https://app.example.com/payment/result",
		Subject:   "Subscription",
	})

	// Then
	require.NoError(t, err)
	require.Equal(t, "hp_123", response.TradeNo)
	require.Equal(t, "https://hashpay.example/pay/hp_123", response.PayURL)
	require.Equal(t, "USD", response.Currency)
}

func TestHashPayQueryOrderSignsEmptyBodyAndMapsPaidStatus(t *testing.T) {
	// Given
	privateKey, privateKeyPEM := newHashPayTestKey(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/order/hp_paid", r.URL.Path)
		verifyHashPayRequestSignature(t, &privateKey.PublicKey, r.Method, r.URL.RequestURI(), r.Header.Get("X-Timestamp"), nil, r.Header.Get("X-Signature"))

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"hp_paid","merchantNo":"sub2_order","amount":9.5,"currency":"USD","status":"paid"}`))
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

	// When
	response, err := provider.QueryOrder(context.Background(), "hp_paid")

	// Then
	require.NoError(t, err)
	require.Equal(t, "hp_paid", response.TradeNo)
	require.Equal(t, payment.ProviderStatusPaid, response.Status)
	require.InDelta(t, 9.5, response.Amount, 0.0001)
	require.Equal(t, "USD", response.Metadata["currency"])
	require.Equal(t, "merchant-1", response.Metadata["merchant_id"])
}

func TestHashPayQueryOrderRejectsUnexpectedMerchantAPIResponses(t *testing.T) {
	_, privateKeyPEM := newHashPayTestKey(t)

	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "response ID belongs to another order",
			body:    `{"id":"hp_other","merchantNo":"sub2_order","amount":9.5,"currency":"USD","status":"paid"}`,
			wantErr: "does not match requested order",
		},
		{
			name:    "response has invalid currency",
			body:    `{"id":"hp_paid","merchantNo":"sub2_order","amount":9.5,"currency":"not-a-currency","status":"paid"}`,
			wantErr: "invalid currency",
		},
		{
			name:    "response omits currency",
			body:    `{"id":"hp_paid","merchantNo":"sub2_order","amount":9.5,"currency":"","status":"paid"}`,
			wantErr: "missing currency",
		},
		{
			name:    "response has undocumented paid-like status",
			body:    `{"id":"hp_paid","merchantNo":"sub2_order","amount":9.5,"currency":"USD","status":"completed"}`,
			wantErr: "unsupported status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/order/hp_paid", r.URL.Path)
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

			_, err = provider.QueryOrder(context.Background(), "hp_paid")
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
