package provider

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

const (
	hashPayCallbackAlgorithm = "RSA-OAEP-256+A256GCM"
	hashPayCallbackTolerance = 5 * time.Minute
)

type hashPayCallbackEnvelope struct {
	Alg  string `json:"alg"`
	Data string `json:"data"`
	IV   string `json:"iv"`
	Key  string `json:"key"`
}

type hashPayCallbackPayload struct {
	Amount     json.Number `json:"amount"`
	Currency   string      `json:"currency"`
	MerchantNo string      `json:"merchantNo"`
	OrderID    string      `json:"orderId"`
	Status     string      `json:"status"`
}

type hashPayCallbackPlaintext struct {
	Payload   hashPayCallbackPayload `json:"payload"`
	Timestamp int64                  `json:"timestamp"`
}

func (h *HashPay) VerifyNotification(ctx context.Context, rawBody string, headers map[string]string) (*payment.PaymentNotification, error) {
	if h == nil || h.config.privateKey == nil {
		return nil, fmt.Errorf("hashpay provider is not configured")
	}
	merchantID := hashPayHeader(headers, "x-hashpay-merchant")
	if merchantID == "" {
		return nil, fmt.Errorf("hashpay notification missing merchant header")
	}
	if !strings.EqualFold(merchantID, h.config.merchantID) {
		return nil, fmt.Errorf("hashpay notification merchant mismatch")
	}
	if algorithm := hashPayHeader(headers, "x-hashpay-encryption"); algorithm != hashPayCallbackAlgorithm {
		return nil, fmt.Errorf("hashpay notification has unsupported encryption algorithm")
	}
	headerTimestamp, err := parseHashPayTimestamp(hashPayHeader(headers, "x-hashpay-timestamp"), time.Now())
	if err != nil {
		return nil, err
	}

	var envelope hashPayCallbackEnvelope
	if err := json.Unmarshal([]byte(rawBody), &envelope); err != nil {
		return nil, fmt.Errorf("decode hashpay notification envelope: %w", err)
	}
	if envelope.Alg != hashPayCallbackAlgorithm {
		return nil, fmt.Errorf("hashpay notification envelope has unsupported encryption algorithm")
	}
	plaintext, err := h.decryptCallbackEnvelope(envelope)
	if err != nil {
		return nil, err
	}
	var message hashPayCallbackPlaintext
	if err := json.Unmarshal(plaintext, &message); err != nil {
		return nil, fmt.Errorf("decode hashpay notification payload: %w", err)
	}
	if message.Timestamp != headerTimestamp {
		return nil, fmt.Errorf("hashpay notification timestamp mismatch")
	}
	if _, err := parseHashPayTimestamp(strconv.FormatInt(message.Timestamp, 10), time.Now()); err != nil {
		return nil, err
	}
	callbackAmount, err := parseHashPayAmount(message.Payload.Amount)
	if err != nil {
		return nil, fmt.Errorf("hashpay notification has invalid amount")
	}
	merchantNo := strings.TrimSpace(message.Payload.MerchantNo)
	orderID := strings.TrimSpace(message.Payload.OrderID)
	if merchantNo == "" || orderID == "" {
		return nil, fmt.Errorf("hashpay notification missing merchantNo or orderId")
	}
	callbackCurrency, err := normalizeHashPayReportedCurrency(message.Payload.Currency)
	if err != nil {
		return nil, fmt.Errorf("hashpay notification has invalid currency: %w", err)
	}
	status := strings.ToLower(strings.TrimSpace(message.Payload.Status))
	notificationStatus, err := hashPayNotificationStatus(status)
	if err != nil {
		return nil, err
	}
	confirmedOrder, err := h.queryOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("hashpay notification confirm order: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(confirmedOrder.Status), "paid") {
		return nil, fmt.Errorf("hashpay notification confirmed order status is not paid")
	}
	if confirmedOrder.MerchantNo != merchantNo {
		return nil, fmt.Errorf("hashpay notification merchantNo mismatch")
	}
	confirmedAmount, err := parseHashPayAmount(confirmedOrder.Amount)
	if err != nil {
		return nil, fmt.Errorf("hashpay notification confirmed order has invalid amount")
	}
	if !confirmedAmount.Equal(callbackAmount) {
		return nil, fmt.Errorf("hashpay notification amount mismatch")
	}
	confirmedCurrency, err := normalizeHashPayReportedCurrency(confirmedOrder.Currency)
	if err != nil {
		return nil, fmt.Errorf("hashpay notification confirmed order has invalid currency: %w", err)
	}
	if confirmedCurrency != callbackCurrency {
		return nil, fmt.Errorf("hashpay notification currency mismatch")
	}
	amount := confirmedAmount.InexactFloat64()
	return &payment.PaymentNotification{
		TradeNo: confirmedOrder.ID,
		OrderID: confirmedOrder.MerchantNo,
		Amount:  amount,
		Status:  notificationStatus,
		RawData: rawBody,
		Metadata: map[string]string{
			"currency":    confirmedCurrency,
			"merchant_id": h.config.merchantID,
			"merchant_no": confirmedOrder.MerchantNo,
			"status":      status,
		},
	}, nil
}

func (h *HashPay) decryptCallbackEnvelope(envelope hashPayCallbackEnvelope) ([]byte, error) {
	encryptedKey, err := base64.StdEncoding.DecodeString(envelope.Key)
	if err != nil {
		return nil, fmt.Errorf("decode hashpay encrypted content key: %w", err)
	}
	contentKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, h.config.privateKey, encryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt hashpay content key: %w", err)
	}
	if len(contentKey) != 32 {
		return nil, fmt.Errorf("hashpay content key has invalid length")
	}
	iv, err := base64.StdEncoding.DecodeString(envelope.IV)
	if err != nil {
		return nil, fmt.Errorf("decode hashpay nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("decode hashpay ciphertext: %w", err)
	}
	block, err := aes.NewCipher(contentKey)
	if err != nil {
		return nil, fmt.Errorf("create hashpay AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create hashpay AES-GCM: %w", err)
	}
	if len(iv) != gcm.NonceSize() {
		return nil, fmt.Errorf("hashpay nonce has invalid length")
	}
	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt hashpay callback payload: %w", err)
	}
	return plaintext, nil
}

func hashPayHeader(headers map[string]string, key string) string {
	for header, value := range headers {
		if strings.EqualFold(header, key) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseHashPayTimestamp(raw string, now time.Time) (int64, error) {
	timestamp, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("hashpay notification has invalid timestamp")
	}
	delta := now.Unix() - timestamp
	if delta < 0 {
		delta = -delta
	}
	if delta > int64(hashPayCallbackTolerance/time.Second) {
		return 0, fmt.Errorf("hashpay notification timestamp is outside tolerance")
	}
	return timestamp, nil
}

func hashPayNotificationStatus(status string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "paid":
		return payment.NotificationStatusSuccess, nil
	default:
		return "", fmt.Errorf("hashpay notification has unsupported status %s", status)
	}
}
