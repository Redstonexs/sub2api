//go:build unit

package service

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGeminiChatCompletionsUpstreamRequestDoesNotTurnModelIntoQuery(t *testing.T) {
	svc := &GeminiMessagesCompatService{cfg: &config.Config{}}
	account := &Account{
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "test-api-key",
		},
	}

	buildRequest, _ := svc.buildGeminiChatCompletionsUpstreamRequestFunc(
		account,
		"gemini-2.5-pro?alt=attacker",
		[]byte(`{}`),
		false,
		false,
	)
	request, _, err := buildRequest(context.Background())
	require.NoError(t, err)
	require.Empty(t, request.URL.RawQuery)
	require.Empty(t, request.URL.Fragment)
	require.Contains(t, request.URL.EscapedPath(), "%3F")
}

func TestBuildGeminiAIStudioURLEscapesDynamicPathSegments(t *testing.T) {
	t.Parallel()

	built := buildGeminiAIStudioURL(
		"https://generativelanguage.googleapis.com",
		"gemini-2.5-pro/../other?alt=attacker#fragment",
		"generateContent?x=1",
		true,
	)
	parsed, err := url.Parse(built)
	require.NoError(t, err)
	require.Equal(t, "alt=sse", parsed.RawQuery)
	require.Empty(t, parsed.Fragment)
	require.Contains(t, parsed.EscapedPath(), "%2F")
	require.Contains(t, parsed.EscapedPath(), "%3F")
	require.Contains(t, parsed.EscapedPath(), "%23")
}

func TestGeminiAIStudioGetDoesNotTurnModelPathIntoQuery(t *testing.T) {
	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: http.NoBody},
	}
	svc := &GeminiMessagesCompatService{httpUpstream: httpStub, cfg: &config.Config{}}
	account := &Account{
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "test-api-key",
		},
	}

	_, err := svc.ForwardAIStudioGET(
		context.Background(),
		account,
		"/v1beta/models/gemini-2.5-pro?alt=attacker",
	)
	require.NoError(t, err)
	require.NotNil(t, httpStub.lastReq)
	require.Empty(t, httpStub.lastReq.URL.RawQuery)
	require.Contains(t, httpStub.lastReq.URL.EscapedPath(), "%3F")
}

func TestGeminiAIStudioGetRejectsReservedDotModelSegments(t *testing.T) {
	for _, model := range []string{".", ".."} {
		httpStub := &geminiCompatHTTPUpstreamStub{
			response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: http.NoBody},
		}
		svc := &GeminiMessagesCompatService{httpUpstream: httpStub, cfg: &config.Config{}}
		account := &Account{
			Type: AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key": "test-api-key",
			},
		}

		_, err := svc.ForwardAIStudioGET(context.Background(), account, "/v1beta/models/"+model)
		require.Error(t, err, model)
		require.Zero(t, httpStub.calls, model)
	}
}
