//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type antigravityOAuthHandlerEncryptorStub struct{}

func (antigravityOAuthHandlerEncryptorStub) Encrypt(string) (string, error) {
	return "synthetic-ciphertext", nil
}

func (antigravityOAuthHandlerEncryptorStub) Decrypt(value string) (string, error) {
	if value != "synthetic-ciphertext" {
		return "", service.ErrAntigravityOAuthCredentialsInvalid
	}
	return "synthetic-secret", nil
}

func newAntigravityOAuthCredentialHandlerTest(t *testing.T, values map[string]string) (*SettingHandler, *settingHandlerRepoStub) {
	t.Helper()
	repo := &settingHandlerRepoStub{values: values}
	settingService := service.NewSettingService(repo, &config.Config{})
	settingService.SetSecretEncryptor(antigravityOAuthHandlerEncryptorStub{}, true)
	return NewSettingHandler(settingService, nil, nil, nil, nil, nil, nil), repo
}

func antigravityOAuthHandlerRequest(t *testing.T, method string, handler gin.HandlerFunc, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/credentials", handler)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, "/credentials", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAntigravityOAuthCredentialsHandlerGetRedactsSecretAndCiphertext(t *testing.T) {
	handler, _ := newAntigravityOAuthCredentialHandlerTest(t, map[string]string{
		service.SettingKeyAntigravityOAuthClientID:               "synthetic-client-id",
		service.SettingKeyAntigravityOAuthClientSecretCiphertext: "v1:synthetic-ciphertext",
	})

	recorder := antigravityOAuthHandlerRequest(t, http.MethodGet, handler.GetAntigravityOAuthCredentials, nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "synthetic-secret")
	require.NotContains(t, recorder.Body.String(), "synthetic-ciphertext")

	var envelope struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	_, hasSecret := envelope.Data["client_secret"]
	require.False(t, hasSecret)
	require.Equal(t, "synthetic-client-id", envelope.Data["client_id"])
	require.Equal(t, true, envelope.Data["client_secret_configured"])
	require.Equal(t, "settings", envelope.Data["source"])
}

func TestAntigravityOAuthCredentialsHandlerPutPreservesOmittedAndEmptySecret(t *testing.T) {
	handler, repo := newAntigravityOAuthCredentialHandlerTest(t, map[string]string{
		service.SettingKeyAntigravityOAuthClientID:               "synthetic-client-id",
		service.SettingKeyAntigravityOAuthClientSecretCiphertext: "v1:synthetic-ciphertext",
	})

	for _, body := range []string{
		`{"client_id":"synthetic-client-id"}`,
		`{"client_id":"synthetic-client-id","client_secret":""}`,
	} {
		recorder := antigravityOAuthHandlerRequest(t, http.MethodPut, handler.UpdateAntigravityOAuthCredentials, []byte(body))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "v1:synthetic-ciphertext", repo.values[service.SettingKeyAntigravityOAuthClientSecretCiphertext])
		require.NotContains(t, recorder.Body.String(), "synthetic-secret")
		require.NotContains(t, recorder.Body.String(), "synthetic-ciphertext")
	}
}

func TestAntigravityOAuthCredentialsHandlerDeleteReturnsStableStatus(t *testing.T) {
	handler, repo := newAntigravityOAuthCredentialHandlerTest(t, map[string]string{
		service.SettingKeyAntigravityOAuthClientID:               "synthetic-client-id",
		service.SettingKeyAntigravityOAuthClientSecretCiphertext: "v1:synthetic-ciphertext",
	})

	recorder := antigravityOAuthHandlerRequest(t, http.MethodDelete, handler.DeleteAntigravityOAuthCredentials, nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "", repo.values[service.SettingKeyAntigravityOAuthClientID])
	require.Equal(t, "", repo.values[service.SettingKeyAntigravityOAuthClientSecretCiphertext])

	var envelope struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, "cleared", envelope.Data.Status)
	require.NotContains(t, recorder.Body.String(), "synthetic")
}
