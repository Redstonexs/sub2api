//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseGeminiModelActionRejectsUnsafeModelSegment(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		".:generateContent",
		"..:generateContent",
		"../v1/users:generateContent",
		"gemini-2.5-pro?alt=sse:generateContent",
		"gemini-2.5-pro#fragment:generateContent",
		"gemini-2.5-pro%2Fv1%2Fusers:generateContent",
		"gemini-2.5-pro\\v1\\users:generateContent",
		" gemini-2.5-pro:generateContent",
	} {
		_, _, err := parseGeminiModelAction(path)
		require.Error(t, err, path)
	}
}

func TestParseGeminiModelActionAcceptsSafeModelSegment(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		path   string
		model  string
		action string
	}{
		{"gemini-2.5-pro:generateContent", "gemini-2.5-pro", "generateContent"},
		{"gemini_2.5-pro/streamGenerateContent", "gemini_2.5-pro", "streamGenerateContent"},
		{"custom.Model-42:countTokens", "custom.Model-42", "countTokens"},
	} {
		model, action, err := parseGeminiModelAction(tc.path)
		require.NoError(t, err, tc.path)
		require.Equal(t, tc.model, model)
		require.Equal(t, tc.action, action)
	}
}

func TestGeminiV1BetaModelsRejectsEncodedUnsafeModelPathBeforeBodyRead(t *testing.T) {
	groupID := int64(1)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
			ID:      1,
			GroupID: &groupID,
			Group: &service.Group{
				ID:       groupID,
				Platform: service.PlatformGemini,
			},
		})
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
	})
	router.POST("/v1beta/models/*modelAction", (&GatewayHandler{}).GeminiV1BetaModels)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	response, err := server.Client().Post(
		server.URL+"/v1beta/models/gemini-2.5-pro%3Falt%3Dsse:generateContent",
		"application/json",
		strings.NewReader(""),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestGeminiV1BetaGetModelRejectsEncodedUnsafeModelPath(t *testing.T) {
	groupID := int64(1)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
			ID:      1,
			GroupID: &groupID,
			Group: &service.Group{
				ID:       groupID,
				Platform: service.PlatformGemini,
			},
		})
	})
	router.GET("/v1beta/models/:model", (&GatewayHandler{}).GeminiV1BetaGetModel)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/v1beta/models/gemini-2.5-pro%3Falt%3Dattacker")
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}
