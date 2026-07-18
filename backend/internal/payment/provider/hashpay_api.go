package provider

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const (
	maxHashPayResponseSize = 1 << 20
	maxHashPayErrorSummary = 512
)

type hashPayCreateOrderRequest struct {
	MerchantNo  string      `json:"merchantNo"`
	Amount      json.Number `json:"amount"`
	Currency    string      `json:"currency"`
	Description string      `json:"description,omitempty"`
	ReturnURL   string      `json:"return_url,omitempty"`
}

type hashPayCreateOrderResponse struct {
	CheckoutURL string `json:"checkoutUrl"`
	Order       struct {
		ID string `json:"id"`
	} `json:"order"`
}

type hashPayOrderResponse struct {
	Amount     json.Number `json:"amount"`
	Currency   string      `json:"currency"`
	ID         string      `json:"id"`
	MerchantNo string      `json:"merchantNo"`
	Status     string      `json:"status"`
}

func (h *HashPay) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || !amount.IsPositive() {
		return nil, fmt.Errorf("hashpay create payment: invalid amount %s", req.Amount)
	}
	payload, err := json.Marshal(hashPayCreateOrderRequest{
		MerchantNo:  req.OrderID,
		Amount:      json.Number(amount.String()),
		Currency:    h.config.currency,
		Description: strings.TrimSpace(req.Subject),
		ReturnURL:   strings.TrimSpace(req.ReturnURL),
	})
	if err != nil {
		return nil, fmt.Errorf("hashpay encode create payment: %w", err)
	}
	body, err := h.doSigned(ctx, http.MethodPost, "/api/merchant/new", payload)
	if err != nil {
		return nil, fmt.Errorf("hashpay create payment: %w", err)
	}
	var response hashPayCreateOrderResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("hashpay decode create payment: %w", err)
	}
	if strings.TrimSpace(response.CheckoutURL) == "" || strings.TrimSpace(response.Order.ID) == "" {
		return nil, fmt.Errorf("hashpay create payment: missing checkout URL or order ID")
	}
	return &payment.CreatePaymentResponse{
		TradeNo:  response.Order.ID,
		PayURL:   response.CheckoutURL,
		Currency: h.config.currency,
	}, nil
}

func (h *HashPay) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	response, err := h.queryOrder(ctx, tradeNo)
	if err != nil {
		return nil, err
	}
	amount, err := response.Amount.Float64()
	if err != nil || amount <= 0 {
		return nil, fmt.Errorf("hashpay query order: invalid amount")
	}
	currency, err := normalizeHashPayReportedCurrency(response.Currency)
	if err != nil {
		return nil, fmt.Errorf("hashpay query order: %w", err)
	}
	status, err := hashPayProviderStatus(response.Status)
	if err != nil {
		return nil, err
	}
	return &payment.QueryOrderResponse{
		TradeNo: response.ID,
		Status:  status,
		Amount:  amount,
		Metadata: map[string]string{
			"currency":    currency,
			"merchant_id": h.config.merchantID,
			"merchant_no": response.MerchantNo,
			"status":      strings.ToLower(strings.TrimSpace(response.Status)),
		},
	}, nil
}

func (h *HashPay) queryOrder(ctx context.Context, tradeNo string) (hashPayOrderResponse, error) {
	orderID := strings.TrimSpace(tradeNo)
	if orderID == "" {
		return hashPayOrderResponse{}, fmt.Errorf("hashpay query order: missing order ID")
	}
	body, err := h.doSigned(ctx, http.MethodGet, "/api/order/"+url.PathEscape(orderID), nil)
	if err != nil {
		return hashPayOrderResponse{}, fmt.Errorf("hashpay query order: %w", err)
	}
	var response hashPayOrderResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return hashPayOrderResponse{}, fmt.Errorf("hashpay decode query order: %w", err)
	}
	response.ID = strings.TrimSpace(response.ID)
	response.MerchantNo = strings.TrimSpace(response.MerchantNo)
	if response.ID == "" {
		return hashPayOrderResponse{}, fmt.Errorf("hashpay query order: missing order ID")
	}
	if response.ID != orderID {
		return hashPayOrderResponse{}, fmt.Errorf("hashpay query order: response order ID %q does not match requested order %q", response.ID, orderID)
	}
	if response.MerchantNo == "" {
		return hashPayOrderResponse{}, fmt.Errorf("hashpay query order: missing merchantNo")
	}
	if _, err := parseHashPayAmount(response.Amount); err != nil {
		return hashPayOrderResponse{}, fmt.Errorf("hashpay query order: %w", err)
	}
	if _, err := normalizeHashPayReportedCurrency(response.Currency); err != nil {
		return hashPayOrderResponse{}, fmt.Errorf("hashpay query order: %w", err)
	}
	if _, err := hashPayProviderStatus(response.Status); err != nil {
		return hashPayOrderResponse{}, err
	}
	return response, nil
}

func parseHashPayAmount(raw json.Number) (decimal.Decimal, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(raw.String()))
	if err != nil || !amount.IsPositive() {
		return decimal.Zero, fmt.Errorf("invalid amount")
	}
	return amount, nil
}

func normalizeHashPayReportedCurrency(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("missing currency")
	}
	currency, err := payment.NormalizePaymentCurrency(raw)
	if err != nil {
		return "", fmt.Errorf("invalid currency: %w", err)
	}
	return currency, nil
}

func (h *HashPay) doSigned(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if h == nil || h.config.privateKey == nil {
		return nil, fmt.Errorf("hashpay provider is not configured")
	}
	endpoint, err := url.Parse(h.endpoint(path))
	if err != nil {
		return nil, fmt.Errorf("parse hashpay endpoint: %w", err)
	}
	signedPath := endpoint.EscapedPath()
	if signedPath == "" {
		signedPath = "/"
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature, err := h.sign(method, signedPath, timestamp, body)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create hashpay request: %w", err)
	}
	request.Header.Set("X-Merchant-Id", h.config.merchantID)
	request.Header.Set("X-Signature", signature)
	request.Header.Set("X-Timestamp", timestamp)
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}

	client := h.httpClient
	if client == nil {
		client = &http.Client{Timeout: hashPayHTTPTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send hashpay request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxHashPayResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read hashpay response: %w", err)
	}
	if len(responseBody) > maxHashPayResponseSize {
		return nil, fmt.Errorf("hashpay response exceeds %d bytes", maxHashPayResponseSize)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, hashPayAPIError(response.StatusCode, responseBody)
	}
	return responseBody, nil
}

func (h *HashPay) sign(method, path, timestamp string, body []byte) (string, error) {
	payload := strings.Join([]string{method, path, timestamp, string(body)}, "\n")
	digest := sha256.Sum256([]byte(payload))
	signature, err := rsa.SignPKCS1v15(rand.Reader, h.config.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign hashpay request: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func hashPayProviderStatus(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "paid":
		return payment.ProviderStatusPaid, nil
	case "pending":
		return payment.ProviderStatusPending, nil
	case "expired", "invalid":
		return payment.ProviderStatusFailed, nil
	default:
		return "", fmt.Errorf("hashpay query order: unsupported status %q", raw)
	}
}

func hashPayAPIError(status int, body []byte) error {
	var payload struct {
		Error struct {
			Key string `json:"key"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Error.Key) != "" {
		return fmt.Errorf("hashpay API request failed with status %d: %s", status, payload.Error.Key)
	}
	summary := strings.TrimSpace(string(body))
	if len(summary) > maxHashPayErrorSummary {
		summary = summary[:maxHashPayErrorSummary]
	}
	if summary == "" {
		return fmt.Errorf("hashpay API request failed with status %d", status)
	}
	return fmt.Errorf("hashpay API request failed with status %d: %s", status, summary)
}
