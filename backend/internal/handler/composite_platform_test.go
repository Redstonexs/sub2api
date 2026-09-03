package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCompositeTargetPlatformAllowedResolvesKnownAllowedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/embeddings", nil)
	apiKey := &service.APIKey{Group: &service.Group{Platform: service.PlatformComposite}}

	require.True(t, compositeTargetPlatformAllowed(c, apiKey, "text-embedding-3-large", service.PlatformOpenAI))
	platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
	require.True(t, ok)
	require.Equal(t, service.PlatformOpenAI, platform)
}

func TestOpenAICompatibleTextTargetAllowsCompositeProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	providers := []struct {
		model    string
		platform string
	}{
		{model: "grok-4.3", platform: service.PlatformGrok},
		{model: "kimi-k2-thinking", platform: service.PlatformKimi},
		{model: "k3", platform: service.PlatformKimi},
		{model: "glm-5.2", platform: service.PlatformZhipu},
		{model: "deepseek-v3.2", platform: service.PlatformDeepseek},
	}
	for _, path := range []string{"/v1/messages", "/v1/chat/completions", "/v1/responses", "/v1/responses/input_tokens", "/v1/messages/count_tokens"} {
		for _, provider := range providers {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", path, nil)
			apiKey := &service.APIKey{Group: &service.Group{Platform: service.PlatformComposite}}

			require.True(t, openAICompatibleTextTargetAllowed(c, apiKey, provider.model), "path=%s model=%s", path, provider.model)
			platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
			require.True(t, ok, "path=%s model=%s", path, provider.model)
			require.Equal(t, provider.platform, platform, "path=%s model=%s", path, provider.model)
		}
	}
}

// WS ingress 对 CN 账号既过不了 transport 过滤、HTTP 桥也没有 Responses 转换，
// 放行只会把明确的策略拒绝换成 "no available account"，因此 WS 白名单保持 openai+grok。
func TestResponsesWebSocketCompositePlatformGuardKeepsOpenAIAndGrokOnly(t *testing.T) {
	require.True(t, isResponsesWebSocketCompositePlatform(service.PlatformOpenAI))
	require.True(t, isResponsesWebSocketCompositePlatform(service.PlatformGrok))
	for _, platform := range []string{
		service.PlatformKimi, service.PlatformZhipu, service.PlatformDeepseek,
		service.PlatformAnthropic, service.PlatformGemini,
	} {
		require.False(t, isResponsesWebSocketCompositePlatform(platform), "platform=%s", platform)
	}
}

func TestCompositeTargetPlatformAllowedRejectsWrongOrUnknownModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name  string
		model string
	}{
		{name: "wrong provider", model: "claude-sonnet-4-5"},
		{name: "unknown provider", model: "llama-4-maverick"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/embeddings", nil)
			apiKey := &service.APIKey{Group: &service.Group{Platform: service.PlatformComposite}}

			require.False(t, compositeTargetPlatformAllowed(c, apiKey, tc.model, service.PlatformOpenAI))
		})
	}
}

func TestCompositeTargetPlatformResolvedRejectsUnknownModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	apiKey := &service.APIKey{Group: &service.Group{Platform: service.PlatformComposite}}

	require.False(t, compositeTargetPlatformResolved(c, apiKey, "llama-4-maverick"))
	_, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
	require.False(t, ok)
}

func TestCompositeTargetPlatformResolvedAllowsConcreteGroupWithoutResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	apiKey := &service.APIKey{Group: &service.Group{Platform: service.PlatformAnthropic}}

	require.True(t, compositeTargetPlatformResolved(c, apiKey, "llama-4-maverick"))
}

func TestOpenAIReasoningEffortPolicyForCompositeTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{
		Platform:           service.PlatformComposite,
		MaxReasoningEffort: "medium",
		ReasoningEffortMappings: []service.ReasoningEffortMapping{
			{From: "max", To: "xhigh"},
		},
	}
	apiKey := &service.APIKey{Group: group}
	body := []byte(`{"reasoning":{"effort":"max"}}`)

	openAICtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	openAICtx.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	openAICtx.Request = openAICtx.Request.WithContext(service.WithResolvedTargetPlatform(openAICtx.Request.Context(), service.PlatformOpenAI))
	got, changed, err := applyOpenAIReasoningEffortPolicyForRequest(openAICtx, apiKey, body)
	require.NoError(t, err)
	require.True(t, changed)
	require.JSONEq(t, `{"reasoning":{"effort":"medium"}}`, string(got))
	requested := service.RequestedReasoningEffortFromContext(openAICtx.Request.Context())
	require.NotNil(t, requested)
	require.Equal(t, "max", *requested)

	bindOpenAIReasoningEffortPolicyForMessagesRequest(openAICtx, apiKey, []byte(`{"output_config":{"effort":"max"}}`))
	bound, changed, err := service.ApplyOpenAIReasoningEffortPolicyFromContext(openAICtx.Request.Context(), body)
	require.NoError(t, err)
	require.True(t, changed)
	require.JSONEq(t, `{"reasoning":{"effort":"medium"}}`, string(bound))

	omittedCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	omittedCtx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	omittedCtx.Request = omittedCtx.Request.WithContext(service.WithResolvedTargetPlatform(omittedCtx.Request.Context(), service.PlatformOpenAI))
	bindOpenAIReasoningEffortPolicyForMessagesRequest(omittedCtx, apiKey, []byte(`{"model":"gpt-5"}`))
	omitted, changed, err := service.ApplyOpenAIReasoningEffortPolicyFromContext(omittedCtx.Request.Context(), body)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, body, omitted)

	denyGroup := *group
	denyGroup.MaxReasoningEffortOverLimit = service.ReasoningEffortOverLimitDeny
	denyAPIKey := &service.APIKey{Group: &denyGroup}
	denyCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	denyCtx.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	denyCtx.Request = denyCtx.Request.WithContext(service.WithResolvedTargetPlatform(denyCtx.Request.Context(), service.PlatformOpenAI))
	_, _, err = applyOpenAIReasoningEffortPolicyForRequest(denyCtx, denyAPIKey, body)
	require.Error(t, err)
	var overLimit *service.ReasoningEffortOverLimitError
	require.ErrorAs(t, err, &overLimit)

	grokCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	grokCtx.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	grokCtx.Request = grokCtx.Request.WithContext(service.WithResolvedTargetPlatform(grokCtx.Request.Context(), service.PlatformGrok))
	got, changed, err = applyOpenAIReasoningEffortPolicyForRequest(grokCtx, apiKey, body)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, body, got)
}

func TestClientRequestedModelUsesCompositePublicModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request = c.Request.WithContext(service.WithCompositeRouteDecision(c.Request.Context(), service.CompositeRouteDecision{
		Matched:        true,
		Source:         service.CompositeRouteSourceExplicit,
		PublicModel:    "public-alias",
		TargetPlatform: service.PlatformOpenAI,
		UpstreamModel:  "gpt-5",
	}))

	input := buildContentModerationInput(c, nil, middleware2.AuthSubject{UserID: 42}, service.ContentModerationProtocolOpenAIChat, "gpt-5", nil)
	require.Equal(t, "public-alias", input.Model)
	require.Equal(t, service.PlatformOpenAI, input.Provider)

	fields := clientRequestedUsageFields(c, service.ChannelMappingResult{MappedModel: "gpt-5"}, "gpt-5", "gpt-5")
	require.Equal(t, "public-alias", fields.OriginalModel)
	require.Equal(t, "public-alias", fields.ChannelMappedModel)
	require.Equal(t, "public-alias\u2192gpt-5", fields.ModelMappingChain)
}

// clientRequestedUsageFields freezes the admission-time QoS snapshot into the
// usage fields synchronously. The value survives the request lifecycle, so
// async usage workers (web search, HTTP, WS turns) never need the gin context.
func TestClientRequestedUsageFieldsFreezesGroupQoSRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/web_search", nil)
	reqCtx := service.BindGroupQoSRecordSnapshot(
		c.Request.Context(),
		&service.GroupQoSRecordSnapshot{Tier: 2, Window: "weekly"},
		"high", nil,
	)
	reqCtx = service.WithGroupQoSDecision(reqCtx, &service.GroupQoSDecision{TierIndex: 1, MaxReasoningEffort: "low"})
	service.MarkGroupQoSRecordEffect(reqCtx, service.GroupQoSEffectRPM)
	c.Request = c.Request.WithContext(reqCtx)

	// No mapping on the web-search path: the requested model is used as-is.
	fields := clientRequestedUsageFields(c, service.ChannelMappingResult{}, "grok-web-search", "grok-web-search")
	require.NotNil(t, fields.GroupQoSRecord, "admission snapshot must be attached")
	require.Equal(t, 2, fields.GroupQoSRecord.Tier)
	require.Equal(t, "weekly", fields.GroupQoSRecord.Window)
	require.Equal(t, service.GroupQoSEffectRPM, fields.GroupQoSRecord.Effects)

	// The snapshot is frozen: later request-scoped marks must not leak into the
	// already-captured value (this is what protects the async worker).
	service.MarkGroupQoSRecordEffect(reqCtx, service.GroupQoSEffectReasoning)
	require.Equal(t, service.GroupQoSEffectRPM, fields.GroupQoSRecord.Effects,
		"the frozen usage fields must not observe later context mutations")

	// No accumulator bound -> nil snapshot (undegraded / fail-open stays NULL).
	plainCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	plainCtx.Request = httptest.NewRequest("POST", "/v1/web_search", nil)
	plainFields := clientRequestedUsageFields(plainCtx, service.ChannelMappingResult{}, "grok-web-search", "grok-web-search")
	require.Nil(t, plainFields.GroupQoSRecord)
}

// withGroupQoSRecordFromContext feeds the WS AfterTurn callback, which runs on
// the passthrough relay goroutine and can outlive the Gin handler goroutine.
//
// Regression: once the handler returns, the gin pool recycles the *gin.Context
// and repoints c.Request to an unrelated request. The QoS snapshot must be read
// from the request-scoped context captured before the relay started — never
// from the original *gin.Context / c.Request.
func TestWithGroupQoSRecordFromContextAfterGinRequestRecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)

	// Admission-time QoS binding mirrors the auth middleware.
	reqCtx := service.BindGroupQoSRecordSnapshot(
		c.Request.Context(),
		&service.GroupQoSRecordSnapshot{Tier: 3, Window: "monthly"},
		"high", nil,
	)
	reqCtx = service.WithGroupQoSDecision(reqCtx, &service.GroupQoSDecision{TierIndex: 2, MaxReasoningEffort: "low"})
	service.MarkGroupQoSRecordEffect(reqCtx, service.GroupQoSEffectRPM)
	c.Request = c.Request.WithContext(reqCtx)

	// The handler freezes the request-scoped context before starting the relay
	// (openai_gateway_handler.go: wsRelayCtx := ctx at hooks construction).
	wsRelayCtx := c.Request.Context()

	// The request goroutine finishes while the relay is still alive: the gin
	// pool repoints the recycled c.Request to an unrelated request that carries
	// a *different* QoS tier. The buggy path (withGroupQoSRecord(c, ...)) would
	// silently attach the recycled request's snapshot instead of the original.
	recycledCtx := service.BindGroupQoSRecordSnapshot(
		context.Background(),
		&service.GroupQoSRecordSnapshot{Tier: 1, Window: "daily"},
		"", nil,
	)
	service.MarkGroupQoSRecordEffect(recycledCtx, service.GroupQoSEffectReasoning)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil).WithContext(recycledCtx)

	// AfterTurn (relay goroutine): freeze the snapshot from the captured context.
	fields := withGroupQoSRecordFromContext(wsRelayCtx, service.ChannelMappingResult{
		MappedModel:        "gpt-5",
		BillingModelSource: service.BillingModelSourceRequested,
	}.ToUsageFields("gpt-5", "gpt-5"))
	require.NotNil(t, fields.GroupQoSRecord, "admission snapshot must be attached from the captured ctx")
	require.Equal(t, 3, fields.GroupQoSRecord.Tier, "must come from the original request, not the recycled one")
	require.Equal(t, "monthly", fields.GroupQoSRecord.Window)
	require.Equal(t, service.GroupQoSEffectRPM, fields.GroupQoSRecord.Effects,
		"effects marked on the recycled request must never be observed")

	// Per-turn isolation: turn 2 starts clean — turn 1's effects do not leak,
	// while tier/window keep the admission-time values.
	service.BeginGroupQoSTurn(wsRelayCtx, 2)
	turn2Fields := withGroupQoSRecordFromContext(wsRelayCtx, service.ChannelMappingResult{
		MappedModel:        "gpt-5",
		BillingModelSource: service.BillingModelSourceRequested,
	}.ToUsageFields("gpt-5", "gpt-5"))
	require.Equal(t, 3, turn2Fields.GroupQoSRecord.Tier, "admission tier/window retained")
	require.Zero(t, turn2Fields.GroupQoSRecord.Effects, "no leakage from turn 1 into turn 2")

	// The gin-based wrapper stays a no-op when the request is gone entirely.
	recycled, _ := gin.CreateTestContext(httptest.NewRecorder())
	recycled.Request = nil
	noRequestFields := withGroupQoSRecord(recycled, service.ChannelUsageFields{OriginalModel: "gpt-5"})
	require.Nil(t, noRequestFields.GroupQoSRecord, "nil request -> snapshot left untouched")
}
