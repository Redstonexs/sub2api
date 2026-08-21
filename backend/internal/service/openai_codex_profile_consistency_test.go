//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 出站请求档案一致性：真实 Codex 客户端在一次请求里只有一个会话身份，且恒为 UUID。
// 本文件是该不变量的回归护栏——网关历史上按入站端点不同发出两种 session_id 形态
// （桥接路径 UUID、主路径 16 位十六进制），且 header 改了、body 没改。

const (
	testCodexClientSession = "0199b1f2-4c3d-7a1e-9f80-2b6d5e4a1c33"
	testCodexTurnMetadata  = `{"installation_id":"inst-client","session_id":"` + testCodexClientSession +
		`","thread_id":"` + testCodexClientSession + `","turn_id":"turn-client","window_id":"` +
		testCodexClientSession + `:0","sandbox":"workspace-write","thread_source":"tui"}`
)

func codexProfileTestBody(promptCacheKey string) []byte {
	return []byte(`{"model":"gpt-5.5","stream":true,"prompt_cache_key":"` + promptCacheKey +
		`","client_metadata":{"session_id":"` + testCodexClientSession +
		`","thread_id":"` + testCodexClientSession +
		`","x-codex-turn-metadata":` + mustJSONString(testCodexTurnMetadata) +
		`},"input":[{"type":"message","role":"user","content":"hello"}]}`)
}

func codexProfileTestContext(t *testing.T, body []byte) *gin.Context {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("User-Agent", "codex-tui/0.146.0 (Ubuntu 22.4.0; x86_64) xterm-256color")
	c.Request.Header.Set("originator", "codex-tui")
	c.Request.Header.Set("session_id", testCodexClientSession)
	c.Request.Header.Set("conversation_id", testCodexClientSession)
	c.Request.Header.Set("x-codex-window-id", testCodexClientSession+":0")
	c.Request.Header.Set("x-codex-turn-metadata", testCodexTurnMetadata)
	return c
}

func codexProfileTestAccount() *Account {
	return &Account{
		ID:          123,
		Name:        "acc",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
		Extra:       map[string]any{"openai_device_id": "device-from-account"},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func codexProfileTestUpstream() *httpUpstreamRecorder {
	return &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"stop"}}`)),
	}}
}

// forwardCodexProfileRequest 跑一次完整 Forward（上游固定 400 立即返回），
// 返回出站请求与出站 body，用于检查 header 与 body 两侧的载体是否同源。
func forwardCodexProfileRequest(t *testing.T, promptCacheKey string) (*http.Request, []byte) {
	t.Helper()
	body := codexProfileTestBody(promptCacheKey)
	c := codexProfileTestContext(t, body)
	upstream := codexProfileTestUpstream()
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}

	_, err := svc.Forward(context.Background(), c, codexProfileTestAccount(), body)
	require.Error(t, err, "stub upstream returns 400 so Forward reports failure")
	require.NotNil(t, upstream.lastReq)
	return upstream.lastReq, upstream.lastBody
}

// 一次请求里的所有会话载体必须收敛到同一个 UUID。
func TestCodexProfile_AllSessionCarriersAgree(t *testing.T) {
	req, outBody := forwardCodexProfileRequest(t, testCodexClientSession)

	sessionID := req.Header.Get("session_id")
	require.NotEmpty(t, sessionID)
	_, parseErr := uuid.Parse(sessionID)
	require.NoError(t, parseErr, "session_id must be UUID-shaped like the official client, got %q", sessionID)
	require.NotEqual(t, testCodexClientSession, sessionID, "客户端原值必须被 apiKey 维度隔离")

	// header 侧载体
	require.Equal(t, sessionID, req.Header.Get("conversation_id"))
	require.Equal(t, sessionID, req.Header.Get("session-id"))
	require.Equal(t, sessionID, req.Header.Get("thread-id"))
	require.Equal(t, sessionID+":0", req.Header.Get("x-codex-window-id"))

	headerMeta := req.Header.Get("x-codex-turn-metadata")
	require.Equal(t, sessionID, gjson.Get(headerMeta, "session_id").String())
	require.Equal(t, sessionID, gjson.Get(headerMeta, "thread_id").String())
	require.Equal(t, sessionID+":0", gjson.Get(headerMeta, "window_id").String())
	// 与会话身份无关的字段必须原样保留。
	require.Equal(t, "workspace-write", gjson.Get(headerMeta, "sandbox").String())
	require.Equal(t, "tui", gjson.Get(headerMeta, "thread_source").String())
	require.Equal(t, "turn-client", gjson.Get(headerMeta, "turn_id").String())

	// body 侧载体
	require.Equal(t, sessionID, gjson.GetBytes(outBody, "prompt_cache_key").String())
	require.Equal(t, sessionID, gjson.GetBytes(outBody, "client_metadata.session_id").String())
	require.Equal(t, sessionID, gjson.GetBytes(outBody, "client_metadata.thread_id").String())
	bodyMeta := gjson.GetBytes(outBody, "client_metadata.x-codex-turn-metadata").String()
	require.Equal(t, sessionID, gjson.Get(bodyMeta, "session_id").String())
	require.Equal(t, sessionID, gjson.Get(bodyMeta, "thread_id").String())
}

// 安装标识：出站头必须跟随 body 里 applyCodexClientMetadata 定稿的值。
func TestCodexProfile_InstallationIDHeaderFollowsBody(t *testing.T) {
	t.Run("客户端未声明时用账号 device_id，两侧一致", func(t *testing.T) {
		req, outBody := forwardCodexProfileRequest(t, testCodexClientSession)
		bodyValue := gjson.GetBytes(outBody, "client_metadata.x-codex-installation-id").String()
		require.Equal(t, "device-from-account", bodyValue)
		require.Equal(t, bodyValue, req.Header.Get("x-codex-installation-id"))
	})

	t.Run("账号无 device_id 时不伪造", func(t *testing.T) {
		body := codexProfileTestBody(testCodexClientSession)
		c := codexProfileTestContext(t, body)
		upstream := codexProfileTestUpstream()
		svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
		account := codexProfileTestAccount()
		account.Extra = map[string]any{}

		_, err := svc.Forward(context.Background(), c, account, body)
		require.Error(t, err)
		require.NotNil(t, upstream.lastReq)
		require.False(t, gjson.GetBytes(upstream.lastBody, "client_metadata.x-codex-installation-id").Exists())
		require.Empty(t, upstream.lastReq.Header.Get("x-codex-installation-id"))
	})
}

// /backend-api/codex 的所有出站路径都不再声明 responses=experimental。
func TestCodexProfile_NoLegacyResponsesBeta(t *testing.T) {
	req, _ := forwardCodexProfileRequest(t, testCodexClientSession)
	require.Empty(t, req.Header.Get("OpenAI-Beta"))
}

// 会话隔离仍然生效，且绝不跨用户/跨会话塌缩——这是 #5610 的护栏：
// 收敛（把多个用户压到一个账号级会话）是显式 opt-in 的独立策略，对齐不得代劳。
func TestCodexProfile_IsolationWithoutCollapse(t *testing.T) {
	t.Run("同一会话键在不同 apiKey 下互不相同", func(t *testing.T) {
		a := codexUpstreamSessionID(1, testCodexClientSession)
		b := codexUpstreamSessionID(2, testCodexClientSession)
		require.NotEmpty(t, a)
		require.NotEqual(t, a, b)
	})

	t.Run("同一 apiKey 下不同会话键互不相同", func(t *testing.T) {
		a := codexUpstreamSessionID(1, "session-a")
		b := codexUpstreamSessionID(1, "session-b")
		require.NotEqual(t, a, b)
	})

	t.Run("同输入恒等", func(t *testing.T) {
		require.Equal(t,
			codexUpstreamSessionID(7, testCodexClientSession),
			codexUpstreamSessionID(7, testCodexClientSession))
	})

	t.Run("空输入不合成身份", func(t *testing.T) {
		require.Empty(t, codexUpstreamSessionID(1, ""))
		require.Nil(t, newCodexOutboundSessionIdentity(1, "   "))
	})
}

// 身份在 gin context 上固定：failover 重入不得因 body 已被改写而派生出第二个会话。
func TestCodexProfile_IdentityStableAcrossReentry(t *testing.T) {
	c := codexProfileTestContext(t, codexProfileTestBody(testCodexClientSession))

	first := resolveCodexOutboundSessionIdentity(c, 5, testCodexClientSession)
	require.NotNil(t, first)
	// 第二次用「已被改写过的」会话键重入——若按新值重新派生就会 isolate 两层。
	second := resolveCodexOutboundSessionIdentity(c, 5, first.sessionID)
	require.Equal(t, first.sessionID, second.sessionID)
}

// 指纹收敛是更强的 opt-in 策略，必须能覆盖对齐结果。
func TestCodexProfile_ConvergenceOverridesAlignment(t *testing.T) {
	body := codexProfileTestBody(testCodexClientSession)
	c := codexProfileTestContext(t, body)
	upstream := codexProfileTestUpstream()
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := codexProfileTestAccount()
	account.Extra = map[string]any{
		"openai_device_id":           "device-from-account",
		codexFingerprintModeExtraKey: string(codexFingerprintSession),
		codexFingerprintSeedExtraKey: "6f1d2c3b-4a59-4e87-9b0d-1c2e3f4a5b6c",
	}

	_, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	require.NotNil(t, upstream.lastReq)

	converged := resolveConvergedSessionID("6f1d2c3b-4a59-4e87-9b0d-1c2e3f4a5b6c")
	require.NotEmpty(t, converged)
	require.Equal(t, converged, upstream.lastReq.Header.Get("session_id"))
	require.Equal(t, converged, gjson.GetBytes(upstream.lastBody, "prompt_cache_key").String())
	require.Equal(t, converged, gjson.GetBytes(upstream.lastBody, "client_metadata.session_id").String())
}

// 关闭开关后逐字回到对齐前的行为，供上游策略变动时回滚。
func TestCodexProfile_KillSwitchRevertsToLegacyShape(t *testing.T) {
	SetCodexProfileAlignmentEnabled(false)
	t.Cleanup(func() { SetCodexProfileAlignmentEnabled(true) })

	req, outBody := forwardCodexProfileRequest(t, testCodexClientSession)

	require.Equal(t, isolateOpenAISessionID(0, testCodexClientSession), req.Header.Get("session_id"))
	require.Empty(t, req.Header.Get("session-id"))
	require.Empty(t, req.Header.Get("thread-id"))
	// body 侧保留客户端原值。
	require.Equal(t, testCodexClientSession, gjson.GetBytes(outBody, "prompt_cache_key").String())
	require.Equal(t, testCodexClientSession, gjson.GetBytes(outBody, "client_metadata.session_id").String())
}

// map 版与 raw 字节版（透传热路径）必须逐点等价，否则两条路径的档案会漂移。
func TestCodexProfile_RawAndMapBodyRewritesMatch(t *testing.T) {
	id := newCodexOutboundSessionIdentity(9, testCodexClientSession)
	require.NotNil(t, id)

	body := codexProfileTestBody(testCodexClientSession)
	rawOut, rawChanged, err := applyCodexSessionIdentityBodyRaw(body, id)
	require.NoError(t, err)
	require.True(t, rawChanged)

	decoded := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.True(t, applyCodexSessionIdentityBody(decoded, id))
	mapEncoded, err := json.Marshal(decoded)
	require.NoError(t, err)

	for _, path := range []string{
		"prompt_cache_key",
		"client_metadata.session_id",
		"client_metadata.thread_id",
		"client_metadata.x-codex-turn-metadata",
	} {
		require.Equal(t,
			gjson.GetBytes(mapEncoded, path).String(),
			gjson.GetBytes(rawOut, path).String(),
			"path %s must match between map and raw rewrites", path)
	}

	t.Run("非对象 body 原样返回", func(t *testing.T) {
		out, changed, err := applyCodexSessionIdentityBodyRaw([]byte(`[1,2,3]`), id)
		require.NoError(t, err)
		require.False(t, changed)
		require.Equal(t, `[1,2,3]`, string(out))
	})
}
