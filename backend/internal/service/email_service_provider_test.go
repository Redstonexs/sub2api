package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEmailProvider(t *testing.T) {
	require.Equal(t, "smtp", NormalizeEmailProvider(""))
	require.Equal(t, "smtp", NormalizeEmailProvider("unknown"))
	require.Equal(t, "smtp", NormalizeEmailProvider("  SMTP "))
	require.Equal(t, "resend", NormalizeEmailProvider("Resend"))
	require.Equal(t, "cyberpanel", NormalizeEmailProvider("CYBERPANEL"))
}

func TestFormatEmailFrom(t *testing.T) {
	require.Equal(t, "no-reply@example.com", formatEmailFrom("no-reply@example.com", ""))
	require.Equal(t, "Acme <no-reply@example.com>", formatEmailFrom("no-reply@example.com", "Acme"))
	// CR/LF are stripped so an injected "Bcc:" can never become its own header line.
	got := formatEmailFrom("no-reply@example.com\r\nBcc: x@y.z", "Acme\r\n")
	require.NotContains(t, got, "\r")
	require.NotContains(t, got, "\n")
}

// captureRequest spins up a test server, runs send, and returns the parsed JSON body + headers.
func captureRequest(t *testing.T, status int, send func(base string) error) (map[string]any, http.Header, string) {
	t.Helper()
	var gotBody map[string]any
	var gotHeader http.Header
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Clone()
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	}))
	defer srv.Close()
	require.NoError(t, send(srv.URL))
	return gotBody, gotHeader, gotPath
}

func TestSendViaResend(t *testing.T) {
	s := &EmailService{}
	cfg := &EmailDeliveryConfig{Provider: EmailProviderResend, From: "no-reply@example.com", FromName: "Acme"}
	body, header, path := captureRequest(t, http.StatusOK, func(base string) error {
		cfg.APIBaseURL = base
		cfg.APIKey = "re_test"
		return s.sendViaResend(context.Background(), cfg, "user@example.com", "Hi", "<p>hi</p>")
	})

	require.Equal(t, "/emails", path)
	require.Equal(t, "Bearer re_test", header.Get("Authorization"))
	require.Equal(t, "application/json", header.Get("Content-Type"))
	require.Equal(t, "Acme <no-reply@example.com>", body["from"])
	require.Equal(t, "Hi", body["subject"])
	require.Equal(t, "<p>hi</p>", body["html"])
	// Resend requires `to` as an array.
	require.Equal(t, []any{"user@example.com"}, body["to"])
}

func TestSendViaCyberPanel(t *testing.T) {
	s := &EmailService{}
	cfg := &EmailDeliveryConfig{Provider: EmailProviderCyberPanel, From: "no-reply@example.com"}
	body, header, path := captureRequest(t, http.StatusOK, func(base string) error {
		cfg.APIBaseURL = base
		cfg.APIKey = "sk_live_test"
		return s.sendViaCyberPanel(context.Background(), cfg, "user@example.com", "Hi", "<p>hi</p>")
	})

	require.Equal(t, "/email/v1/send", path)
	require.Equal(t, "Bearer sk_live_test", header.Get("Authorization"))
	require.Equal(t, "no-reply@example.com", body["from"])
	// CyberPanel takes `to` as a plain string.
	require.Equal(t, "user@example.com", body["to"])
}

func TestPostEmailAPISurfacesErrorBody(t *testing.T) {
	s := &EmailService{}
	cfg := &EmailDeliveryConfig{Provider: EmailProviderResend, From: "no-reply@example.com", APIKey: "re_test"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"domain is not verified"}`))
	}))
	defer srv.Close()
	cfg.APIBaseURL = srv.URL

	err := s.sendViaResend(context.Background(), cfg, "user@example.com", "Hi", "<p>hi</p>")
	require.Error(t, err)
	require.ErrorContains(t, err, "HTTP 422")
	require.ErrorContains(t, err, "domain is not verified")
}
