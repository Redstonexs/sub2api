package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type capVerifier struct {
	httpClient *http.Client
}

func NewCapVerifier() service.CapVerifier {
	sharedClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:           10 * time.Second,
		AllowPrivateHosts: true,
	})
	if err != nil {
		sharedClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &capVerifier{httpClient: sharedClient}
}

func (v *capVerifier) VerifyToken(ctx context.Context, apiEndpoint, siteKey, secretKey, token string) (*service.CapVerifyResponse, error) {
	endpoint, err := url.Parse(strings.TrimSpace(apiEndpoint))
	if err != nil {
		return nil, fmt.Errorf("parse CAP API endpoint: %w", err)
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return nil, fmt.Errorf("CAP API endpoint uses unsupported scheme %q", endpoint.Scheme)
	}
	if endpoint.Host == "" {
		return nil, fmt.Errorf("CAP API endpoint is missing host")
	}
	if endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, fmt.Errorf("CAP API endpoint must not include a query or fragment")
	}

	payload, err := json.Marshal(struct {
		Secret   string `json:"secret"`
		Response string `json:"response"`
	}{
		Secret:   secretKey,
		Response: token,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal CAP siteverify request: %w", err)
	}

	verifyURL := strings.TrimRight(endpoint.String(), "/") + "/" + url.PathEscape(siteKey) + "/siteverify"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create CAP siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send CAP siteverify request: %w", err)
	}
	defer resp.Body.Close()

	var result service.CapVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode CAP siteverify response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &service.CapVerifyResponse{Success: false}, nil
	}
	return &result, nil
}
