//go:build unit

package handler

// 槽位终检与生图跳门回归（handler 半程）：
//   - 槽位获取成功后的利润终检：越线账号释放槽位并要求调用方排除重选，
//     不写响应、不绑定粘连；
//   - openAIResponsesRequiredCapability 的生图意图映射钉死（scheduler 的
//     跳门条件依赖 CapabilityResponses ⇔ 显式生图意图这一耦合）。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type profitCountingConcurrencyCache struct {
	fakeConcurrencyCache
	accountReleases atomic.Int64
}

func (c *profitCountingConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error {
	c.accountReleases.Add(1)
	return nil
}

func profitSlotTestAccount(id int64, rate float64) *service.Account {
	now := time.Now()
	return &service.Account{
		ID:             id,
		Platform:       service.PlatformOpenAI,
		Type:           service.AccountTypeAPIKey,
		Status:         service.StatusActive,
		Schedulable:    true,
		Concurrency:    2,
		RateMultiplier: &rate,
		Extra: map[string]any{
			"upstream_billing_probe": map[string]any{
				"status":      service.UpstreamBillingProbeStatusOK,
				"received_at": now.Add(-time.Minute),
				"fresh_until": now.Add(30 * time.Minute),
				"data": map[string]any{
					"billing_scope":            "token",
					"resolved_rate_multiplier": rate,
					"peak_rate_enabled":        false,
				},
			},
		},
	}
}

func profitSlotTestContext(t *testing.T, gw *service.OpenAIGatewayService, groupID int64, suppress bool) context.Context {
	t.Helper()
	group := &service.Group{
		ID:                   groupID,
		Platform:             service.PlatformOpenAI,
		Status:               service.StatusActive,
		Hydrated:             true,
		RateMultiplier:       1.0,
		SubscriptionType:     service.SubscriptionTypeStandard,
		ProfitControlEnabled: true,
		ProfitMinMargin:      0.5,
	}
	base := context.WithValue(context.Background(), ctxkey.Group, group)
	if suppress {
		base = service.WithOpenAIProfitControlSuppressed(base)
	}
	ctx, pricingAt := gw.WithOpenAIRequestPricingContext(base, &groupID)
	require.False(t, pricingAt.IsZero())
	return ctx
}

func TestAcquireResponsesAccountSlotProfitRecheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &service.OpenAIGatewayService{}
	groupID := int64(50)

	newHandler := func(cache *profitCountingConcurrencyCache) *OpenAIGatewayHandler {
		return &OpenAIGatewayHandler{
			gatewayService:    gw,
			concurrencyHelper: NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatClaude, 0),
		}
	}
	newSelection := func(account *service.Account) *service.AccountSelectionResult {
		return &service.AccountSelectionResult{
			Account:  account,
			Acquired: false,
			WaitPlan: &service.AccountWaitPlan{AccountID: account.ID, MaxConcurrency: 2, Timeout: time.Second, MaxWaiting: 2},
		}
	}

	t.Run("veto releases slot and requests reschedule without writing response", func(t *testing.T) {
		cache := &profitCountingConcurrencyCache{}
		h := newHandler(cache)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(profitSlotTestContext(t, gw, groupID, false))
		streamStarted := false

		release, result := h.acquireResponsesAccountSlot(c, &groupID, "", newSelection(profitSlotTestAccount(1, 0.8)), false, &streamStarted, zap.NewNop())
		require.Equal(t, openAISlotAcquireProfitVetoed, result)
		require.Nil(t, release)
		require.Zero(t, w.Body.Len(), "利润终检否决不得写出任何响应")
		require.Equal(t, int64(1), cache.accountReleases.Load(), "否决后必须立即释放已获取的槽位")
	})

	t.Run("qualifying account acquires normally", func(t *testing.T) {
		cache := &profitCountingConcurrencyCache{}
		h := newHandler(cache)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(profitSlotTestContext(t, gw, groupID, false))
		streamStarted := false

		release, result := h.acquireResponsesAccountSlot(c, &groupID, "", newSelection(profitSlotTestAccount(2, 0.3)), false, &streamStarted, zap.NewNop())
		require.Equal(t, openAISlotAcquireOK, result)
		require.NotNil(t, release)
		release()
	})

	t.Run("image intent suppression keeps official behavior", func(t *testing.T) {
		cache := &profitCountingConcurrencyCache{}
		h := newHandler(cache)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(profitSlotTestContext(t, gw, groupID, true))
		streamStarted := false

		release, result := h.acquireResponsesAccountSlot(c, &groupID, "", newSelection(profitSlotTestAccount(3, 0.8)), false, &streamStarted, zap.NewNop())
		require.Equal(t, openAISlotAcquireOK, result, "生图意图跳门：过贵账号照常获取（图片边界不装门）")
		require.NotNil(t, release)
		release()
	})
}

// scheduler 跳门条件依赖"CapabilityResponses 仅在显式生图意图时被要求"这一
// 映射；后续若扩展该 capability 的用途，本测试失败提示同步收窄跳门条件。
func TestOpenAIResponsesRequiredCapabilityPinsImageIntentMapping(t *testing.T) {
	require.Equal(t, service.OpenAIEndpointCapabilityResponses, openAIResponsesRequiredCapability(true, service.PlatformOpenAI))
	require.Equal(t, service.OpenAIEndpointCapabilityChatCompletions, openAIResponsesRequiredCapability(false, service.PlatformOpenAI))
	require.Equal(t, service.OpenAIEndpointCapabilityChatCompletions, openAIResponsesRequiredCapability(true, service.PlatformGrok))
}

// TestHandleOpenAIProfitVetoExhaustedWrites503 钉死利润否决预算耗尽时 OpenAI 侧
// 错误出口（handleOpenAIProfitVetoExhausted，openai chat/responses/images/
// embeddings/alpha-search 等路径共用的终态写入点）的响应形状与文案覆盖：
// 非流式请求写出 503 + OpenAI 兼容 error 结构，message 走 profitVetoExhaustedText
// 的覆盖语义（error_messages["503"] 优先生效，否则回退默认文案）。
func TestHandleOpenAIProfitVetoExhaustedWrites503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &service.OpenAIGatewayService{}
	groupID := int64(50)

	newHandler := func(cfg *config.Config) *OpenAIGatewayHandler {
		return &OpenAIGatewayHandler{
			gatewayService:    gw,
			cfg:               cfg,
			concurrencyHelper: NewConcurrencyHelper(service.NewConcurrencyService(&fakeConcurrencyCache{}), SSEPingFormatClaude, 0),
		}
	}
	newRequest := func(h *OpenAIGatewayHandler, override bool) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		if override {
			c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil).WithContext(profitSlotTestContext(t, gw, groupID, false))
		} else {
			c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
		}
		c.Set("_test_recorder", w)
		return c
	}
	assertErrorShape := func(t *testing.T, c *gin.Context, wantMessage string) {
		t.Helper()
		wVal, _ := c.Get("_test_recorder")
		w := wVal.(*httptest.ResponseRecorder)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		var resp struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, "api_error", resp.Error.Type)
		require.Equal(t, wantMessage, resp.Error.Message)
	}

	t.Run("fallback message without 503 override", func(t *testing.T) {
		h := newHandler(&config.Config{})
		c := newRequest(h, false)
		h.handleOpenAIProfitVetoExhausted(c, false, zap.NewNop(), maxProfitVetoAttempts)
		assertErrorShape(t, c, profitVetoExhaustedMessage)
	})

	t.Run("custom 503 override wins", func(t *testing.T) {
		h := newHandler(&config.Config{Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": "custom profit control message"}}})
		c := newRequest(h, false)
		h.handleOpenAIProfitVetoExhausted(c, false, zap.NewNop(), maxProfitVetoAttempts)
		assertErrorShape(t, c, "custom profit control message")
	})
}
