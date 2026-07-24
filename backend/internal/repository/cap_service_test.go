//go:build unit

package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapVerifier_sendsStandaloneSiteverifyJSON_whenTokenProvided(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, http.MethodPost, request.Method)
		require.Equal(t, "/site-key/siteverify", request.URL.Path)
		require.Contains(t, request.Header.Get("Content-Type"), "application/json")

		var body struct {
			Secret   string `json:"secret"`
			Response string `json:"response"`
		}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		require.Equal(t, "site-secret", body.Secret)
		require.Equal(t, "cap-token", body.Response)

		writer.Header().Set("Content-Type", "application/json")
		_, err := writer.Write([]byte(`{"success":true}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	verifier, ok := NewCapVerifier().(*capVerifier)
	require.True(t, ok)
	verifier.httpClient = server.Client()

	// When
	result, err := verifier.VerifyToken(context.Background(), server.URL, "site-key", "site-secret", "cap-token")

	// Then
	require.NoError(t, err)
	require.True(t, result.Success)
}

func TestCapVerifier_returnsUnsuccessfulResult_whenStandaloneRejectsToken(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/site-key/siteverify", request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		_, err := writer.Write([]byte(`{"success":false}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	verifier, ok := NewCapVerifier().(*capVerifier)
	require.True(t, ok)
	verifier.httpClient = server.Client()

	// When
	result, err := verifier.VerifyToken(context.Background(), server.URL, "site-key", "site-secret", "expired-cap-token")

	// Then
	require.NoError(t, err)
	require.False(t, result.Success)
}
