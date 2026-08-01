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

// model 取自请求体、可能再经渠道映射，属于客户端可控输入。一旦它带上会改变上游
// URL 结构的字符，必须在构造上游请求时直接报错，而不是转义后照发。
func TestGeminiChatCompletionsUpstreamRequestRejectsModelTurningIntoQuery(t *testing.T) {
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
	require.Error(t, err)
	require.Nil(t, request)

	// 合规 model 仍照常构造，且不会产生查询串或 fragment。
	buildRequest, _ = svc.buildGeminiChatCompletionsUpstreamRequestFunc(
		account,
		"gemini-2.5-pro",
		[]byte(`{}`),
		false,
		false,
	)
	request, _, err = buildRequest(context.Background())
	require.NoError(t, err)
	require.Empty(t, request.URL.RawQuery)
	require.Empty(t, request.URL.Fragment)
	require.Contains(t, request.URL.EscapedPath(), "gemini-2.5-pro")
}

// 上游 URL 构造走的是"只校验、不改写"的护栏（见 upstream_path_guard.go）：
// 畸形的 model/action 直接报错，而不是转义成合规片段后照发——后者会让上游收到
// 与客户端意图不同的请求，并掩盖调用方的错误。
func TestBuildGeminiAIStudioModelActionURLRejectsDynamicPathSegments(t *testing.T) {
	t.Parallel()

	_, err := buildGeminiAIStudioModelActionURL(
		"https://generativelanguage.googleapis.com",
		"gemini-2.5-pro/../other?alt=attacker#fragment",
		"generateContent",
		true,
	)
	require.Error(t, err)

	// action 是闭集允许清单，任何附带查询串的写法都必须拒绝。
	_, err = buildGeminiAIStudioModelActionURL(
		"https://generativelanguage.googleapis.com",
		"gemini-2.5-pro",
		"generateContent?x=1",
		true,
	)
	require.Error(t, err)

	// 合规输入保持原样转发，流式追加 alt=sse。
	built, err := buildGeminiAIStudioModelActionURL(
		"https://generativelanguage.googleapis.com",
		"gemini-2.5-pro",
		"generateContent",
		true,
	)
	require.NoError(t, err)
	parsed, err := url.Parse(built)
	require.NoError(t, err)
	require.Equal(t, "alt=sse", parsed.RawQuery)
	require.Empty(t, parsed.Fragment)
	require.Equal(t, "/v1beta/models/gemini-2.5-pro:generateContent", parsed.EscapedPath())
}

// GET 转发的子路径同样过护栏：畸形片段在发出请求前就被拒，上游一次都不会被调用。
func TestGeminiAIStudioGetRejectsModelPathTurningIntoQuery(t *testing.T) {
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
	require.Error(t, err)
	require.Zero(t, httpStub.calls)

	// 合规路径仍照常转发，且不产生查询串。
	_, err = svc.ForwardAIStudioGET(
		context.Background(),
		account,
		"/v1beta/models/gemini-2.5-pro",
	)
	require.NoError(t, err)
	require.NotNil(t, httpStub.lastReq)
	require.Empty(t, httpStub.lastReq.URL.RawQuery)
	require.Contains(t, httpStub.lastReq.URL.EscapedPath(), "gemini-2.5-pro")
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
