package provider

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

const hashPayHTTPTimeout = 15 * time.Second

type hashPayConfig struct {
	apiBase    string
	currency   string
	merchantID string
	privateKey *rsa.PrivateKey
}

// HashPay implements the hosted crypto checkout provider API.
type HashPay struct {
	instanceID string
	config     hashPayConfig
	httpClient *http.Client
}

// NewHashPay builds a HashPay provider from a configured merchant identity.
func NewHashPay(instanceID string, config map[string]string) (*HashPay, error) {
	cfg, err := newHashPayConfig(config)
	if err != nil {
		return nil, err
	}
	return &HashPay{
		instanceID: instanceID,
		config:     cfg,
		httpClient: &http.Client{Timeout: hashPayHTTPTimeout},
	}, nil
}

func newHashPayConfig(raw map[string]string) (hashPayConfig, error) {
	apiBase, err := normalizeHashPayAPIBase(raw["apiBase"])
	if err != nil {
		return hashPayConfig{}, err
	}
	merchantID := strings.TrimSpace(raw["merchantId"])
	if merchantID == "" {
		return hashPayConfig{}, fmt.Errorf("hashpay config missing required key: merchantId")
	}
	privateKey, err := parseHashPayPrivateKey(raw["privateKey"])
	if err != nil {
		return hashPayConfig{}, err
	}
	currency, err := payment.NormalizePaymentCurrency(raw["currency"])
	if err != nil {
		return hashPayConfig{}, fmt.Errorf("hashpay config currency: %w", err)
	}
	return hashPayConfig{
		apiBase:    apiBase,
		currency:   currency,
		merchantID: merchantID,
		privateKey: privateKey,
	}, nil
}

func normalizeHashPayAPIBase(raw string) (string, error) {
	base := strings.TrimSpace(raw)
	if base == "" {
		return "", fmt.Errorf("hashpay config missing required key: apiBase")
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("hashpay apiBase must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("hashpay apiBase must not contain credentials, query, or fragment")
	}
	parsed.RawPath = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.Path = strings.TrimSuffix(parsed.Path, "/api")
	return strings.TrimRight(parsed.String(), "/"), nil
}

func parseHashPayPrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, rest := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil || len(bytes.TrimSpace(rest)) != 0 {
		return nil, fmt.Errorf("hashpay privateKey must be a single PEM private key")
	}

	var privateKey *rsa.PrivateKey
	var err error
	switch block.Type {
	case "PRIVATE KEY":
		parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			err = parseErr
			break
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("hashpay privateKey must be RSA")
		}
		privateKey = rsaKey
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("hashpay privateKey has unsupported PEM type %s", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("parse hashpay privateKey: %w", err)
	}
	if err := privateKey.Validate(); err != nil {
		return nil, fmt.Errorf("validate hashpay privateKey: %w", err)
	}
	if privateKey.N.BitLen() < 2048 {
		return nil, fmt.Errorf("hashpay privateKey must be at least 2048 bits")
	}
	return privateKey, nil
}

func (h *HashPay) Name() string        { return "HashPay" }
func (h *HashPay) ProviderKey() string { return payment.TypeHashPay }
func (h *HashPay) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeHashPay}
}

func (h *HashPay) MerchantIdentityMetadata() map[string]string {
	if h == nil {
		return nil
	}
	return map[string]string{
		"currency":    h.config.currency,
		"merchant_id": h.config.merchantID,
	}
}

func (h *HashPay) Refund(_ context.Context, _ payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, fmt.Errorf("hashpay refund is not supported")
}

func (h *HashPay) endpoint(path string) string {
	return h.config.apiBase + path
}
