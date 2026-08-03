//go:build unit

package handler

// Gate 3 利润否决选号耗尽修复的聚焦测试：
//   - SelectionExhaustedByProfitVeto（FailoverState）与
//     openAISelectionExhaustedByProfitVeto（OpenAI 侧独立计数）两个判定的真值表；
//   - 小候选池（1/2 账号）整池被利润门否决、且未触发 maxProfitVetoAttempts 时，
//     handler 选号循环在 select 报错处必须写出协议正确的 503（profitVetoExhaustedText
//     文案，支持 error_messages["503"] 覆盖），而不是通用 502；
//   - 混合（否决 + 真实 failover/资格失败）与零否决场景保持既有通用行为。
//
// 说明：gateway/OpenAI 两套调度器都会在候选过滤阶段预筛利润不合格账号，现有
// fixture（repo stub + 静态倍率）无法确定性复现「调度器放行 → 槽位终检否决」的
// 时间差路径，因此这里用「真实状态机（FailoverState/RecordProfitVeto/判定）+
// 真实 handler 写入器」驱动与各选号循环新增分支逐字同构的决策形状，覆盖响应
// 路由与文案，而不是整个调度循环。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// 判定真值表
// ---------------------------------------------------------------------------

func TestSelectionExhaustedByProfitVetoTruthTable(t *testing.T) {
	t.Run("zero veto with empty exclusions → false", func(t *testing.T) {
		fs := NewFailoverState(10, false)
		require.False(t, fs.SelectionExhaustedByProfitVeto())
	})

	t.Run("zero veto with real failover exclusions → false", func(t *testing.T) {
		fs := NewFailoverState(10, false)
		fs.FailedAccountIDs[1] = struct{}{}
		fs.FailedAccountIDs[2] = struct{}{}
		require.False(t, fs.SelectionExhaustedByProfitVeto())
	})

	t.Run("one veto only → true", func(t *testing.T) {
		fs := NewFailoverState(10, false)
		require.Equal(t, FailoverContinue, fs.RecordProfitVeto(1))
		require.True(t, fs.SelectionExhaustedByProfitVeto())
	})

	t.Run("two vetoes only → true", func(t *testing.T) {
		fs := NewFailoverState(10, false)
		fs.RecordProfitVeto(1)
		fs.RecordProfitVeto(2)
		require.True(t, fs.SelectionExhaustedByProfitVeto())
	})

	t.Run("one veto plus real failover exclusion → false", func(t *testing.T) {
		fs := NewFailoverState(10, false)
		fs.RecordProfitVeto(1)
		// 账号 2 因真实上游失败被排除，不来自利润门否决。
		fs.FailedAccountIDs[2] = struct{}{}
		require.False(t, fs.SelectionExhaustedByProfitVeto())
	})

	t.Run("veto record without exclusion list entry → false", func(t *testing.T) {
		// 防御边界：计数已递增但排除集被外部清空（如 503 退避分支）时不得误判。
		fs := NewFailoverState(10, false)
		fs.RecordProfitVeto(7)
		fs.FailedAccountIDs = make(map[int64]struct{})
		require.False(t, fs.SelectionExhaustedByProfitVeto())
	})
}

func TestOpenAISelectionExhaustedByProfitVetoTruthTable(t *testing.T) {
	t.Run("empty sets → false", func(t *testing.T) {
		require.False(t, openAISelectionExhaustedByProfitVeto(
			map[int64]struct{}{}, map[int64]struct{}{}))
	})

	t.Run("one veto only → true", func(t *testing.T) {
		require.True(t, openAISelectionExhaustedByProfitVeto(
			map[int64]struct{}{1: {}}, map[int64]struct{}{1: {}}))
	})

	t.Run("two vetoes only → true", func(t *testing.T) {
		require.True(t, openAISelectionExhaustedByProfitVeto(
			map[int64]struct{}{1: {}, 2: {}}, map[int64]struct{}{1: {}, 2: {}}))
	})

	t.Run("veto plus real failover exclusion → false", func(t *testing.T) {
		require.False(t, openAISelectionExhaustedByProfitVeto(
			map[int64]struct{}{1: {}, 2: {}}, map[int64]struct{}{1: {}}))
	})

	t.Run("failover exclusions only → false", func(t *testing.T) {
		require.False(t, openAISelectionExhaustedByProfitVeto(
			map[int64]struct{}{1: {}, 2: {}}, map[int64]struct{}{}))
	})

	t.Run("vetoed set extra entry still pure when every exclusion is vetoed → true", func(t *testing.T) {
		// 防御边界：否决集比排除集多一个已放回池中的账号（RecordProfitVeto 之后
		// 又因别的原因被解除排除）时，现有排除仍全部来自否决。
		require.True(t, openAISelectionExhaustedByProfitVeto(
			map[int64]struct{}{1: {}}, map[int64]struct{}{1: {}, 2: {}}))
	})

	t.Run("non-empty vetoed but empty exclusions → false", func(t *testing.T) {
		require.False(t, openAISelectionExhaustedByProfitVeto(
			map[int64]struct{}{}, map[int64]struct{}{1: {}}))
	})
}

// recordOpenAIProfitVetoTracked 把否决账号同时登记进独立否决集：真值表之上的
// 记账契约——排除集条目必须与否决集条目一一对应（真实 failover 不进入否决集）。
func TestOpenAIProfitVetoTrackedRecordsVetoedSetOnly(t *testing.T) {
	failed := make(map[int64]struct{})
	vetoed := make(map[int64]struct{})
	count := 0

	require.True(t, recordOpenAIProfitVetoTracked(failed, vetoed, 1, &count))
	require.Contains(t, failed, int64(1))
	require.Contains(t, vetoed, int64(1))

	// 真实 failover 失败只写排除集，不写否决集。
	failed[2] = struct{}{}
	require.NotContains(t, vetoed, int64(2))
	require.False(t, openAISelectionExhaustedByProfitVeto(failed, vetoed))
}

// ---------------------------------------------------------------------------
// FailoverState 系五个选号循环的小池路由（新增分支决策形状）
// ---------------------------------------------------------------------------

// gatewaySelectionExhaustionDecision 复刻五个 FailoverState 选号循环在 select
// 报错分支的决策形状（gateway_handler.go ×2 / gateway_handler_chat_completions.go
// / gateway_handler_responses.go / gemini_v1beta_handler.go 同构）：
//  1. 排除列表全由利润门否决造成 → writeVeto503 写入协议正确 503；
//  2. 否则走 HandleSelectionExhausted 通用处理（503 退避 / 通用 502）。
//
// 返回写出的状态码与 error message；未写出响应时状态码为 0。
func gatewaySelectionExhaustionDecision(
	h *GatewayHandler,
	c *gin.Context,
	fs *FailoverState,
	writeVeto503 func(c *gin.Context, message string),
) (int, string) {
	if fs.SelectionExhaustedByProfitVeto() {
		writeVeto503(c, profitVetoExhaustedText(h.cfg))
		return recorderStatusAndMessage(c)
	}
	switch fs.HandleSelectionExhausted(c.Request.Context()) {
	case FailoverContinue:
		return 0, ""
	case FailoverCanceled:
		return http.StatusRequestTimeout, "" // 499 语义：见 failoverClientGone
	default:
		if fs.LastFailoverErr != nil {
			h.handleFailoverExhausted(c, fs.LastFailoverErr, service.PlatformOpenAI, false)
		} else {
			h.handleFailoverExhaustedSimple(c, 502, false)
		}
		return recorderStatusAndMessage(c)
	}
}

func newGatewaySelectionTestContext(t *testing.T, h *GatewayHandler) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("_test_recorder", w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	return c
}

func recorderStatusAndMessage(c *gin.Context) (int, string) {
	wVal, _ := c.Get("_test_recorder")
	rec := wVal.(*httptest.ResponseRecorder)
	if rec.Code == 0 && rec.Body.Len() == 0 {
		return 0, ""
	}
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body.Error.Message
}

// TestGatewaySelectionExhaustionByProfitVetoSmallPool 钉死修复的核心语义：1/2 账号
// 的小候选池在 maxProfitVetoAttempts 之前被整池利润否决时，select 报错必须路由到
// 协议正确 503（profitVetoExhaustedText 文案、支持自定义覆盖），而不是通用 502。
func TestGatewaySelectionExhaustionByProfitVetoSmallPool(t *testing.T) {
	writeStreamingVeto503 := func(h *GatewayHandler) func(c *gin.Context, message string) {
		return func(c *gin.Context, message string) {
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", message, false)
		}
	}

	t.Run("one-account pool all vetoed → 503 profit message", func(t *testing.T) {
		fs := NewFailoverState(10, false)
		require.Equal(t, FailoverContinue, fs.RecordProfitVeto(1), "小池否决未达上限，循环应继续")
		h := &GatewayHandler{cfg: &config.Config{}}
		c := newGatewaySelectionTestContext(t, h)

		status, msg := gatewaySelectionExhaustionDecision(h, c, fs, writeStreamingVeto503(h))
		require.Equal(t, http.StatusServiceUnavailable, status)
		require.Equal(t, profitVetoExhaustedMessage, msg)
	})

	t.Run("two-account pool all vetoed → 503 profit message", func(t *testing.T) {
		fs := NewFailoverState(10, false)
		fs.RecordProfitVeto(1)
		fs.RecordProfitVeto(2)
		h := &GatewayHandler{cfg: &config.Config{}}
		c := newGatewaySelectionTestContext(t, h)

		status, msg := gatewaySelectionExhaustionDecision(h, c, fs, writeStreamingVeto503(h))
		require.Equal(t, http.StatusServiceUnavailable, status)
		require.Equal(t, profitVetoExhaustedMessage, msg)
	})

	t.Run("custom 503 override wins on small-pool veto exhaustion", func(t *testing.T) {
		const override503 = "custom profit control message"
		fs := NewFailoverState(10, false)
		fs.RecordProfitVeto(9)
		h := &GatewayHandler{cfg: &config.Config{
			Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": override503}},
		}}
		c := newGatewaySelectionTestContext(t, h)

		status, msg := gatewaySelectionExhaustionDecision(h, c, fs, writeStreamingVeto503(h))
		require.Equal(t, http.StatusServiceUnavailable, status)
		require.Equal(t, override503, msg)
	})

	t.Run("mixed veto + real failover exclusion keeps generic 502", func(t *testing.T) {
		fs := NewFailoverState(10, false)
		fs.RecordProfitVeto(1)
		// 账号 2 因真实上游失败被排除：不再是纯利润否决耗尽。
		fs.FailedAccountIDs[2] = struct{}{}
		h := &GatewayHandler{cfg: &config.Config{}}
		c := newGatewaySelectionTestContext(t, h)

		status, msg := gatewaySelectionExhaustionDecision(h, c, fs, writeStreamingVeto503(h))
		require.Equal(t, http.StatusBadGateway, status)
		require.NotEqual(t, profitVetoExhaustedMessage, msg)
	})

	t.Run("zero veto ordinary exhaustion keeps generic 502", func(t *testing.T) {
		fs := NewFailoverState(3, false)
		fs.FailedAccountIDs[100] = struct{}{} // 无任何利润否决
		h := &GatewayHandler{cfg: &config.Config{}}
		c := newGatewaySelectionTestContext(t, h)

		status, _ := gatewaySelectionExhaustionDecision(h, c, fs, writeStreamingVeto503(h))
		require.Equal(t, http.StatusBadGateway, status)
	})

	t.Run("zero veto with 503 backoff state keeps backoff retry", func(t *testing.T) {
		fs := NewFailoverState(3, false)
		fs.LastFailoverErr = newTestFailoverErr(http.StatusServiceUnavailable, false, false)
		fs.SwitchCount = 1
		fs.FailedAccountIDs[100] = struct{}{} // 真实 503，无利润否决
		h := &GatewayHandler{cfg: &config.Config{}}
		c := newGatewaySelectionTestContext(t, h)

		status, _ := gatewaySelectionExhaustionDecision(h, c, fs, writeStreamingVeto503(h))
		require.Equal(t, 0, status, "退避分支不应写出响应（调用方继续循环）")
		require.Empty(t, fs.FailedAccountIDs, "无利润否决的 503 退避仍应清空排除列表")
	})
}

// TestSelectionExhaustedByProfitVetoProtocolWriters 钉死五个 FailoverState 循环
// 新增分支各自使用的协议正确 503 写入器都输出 503 + profitVetoExhaustedText
// 文案（覆盖自定义 error_messages["503"] 语义由 profitVetoExhaustedText 统一保证）。
func TestSelectionExhaustedByProfitVetoProtocolWriters(t *testing.T) {
	tests := []struct {
		name    string
		write   func(h *GatewayHandler, c *gin.Context, message string)
		readMsg func(c *gin.Context) string
	}{
		{
			name: "gateway streaming-aware (gateway_handler.go ×2)",
			write: func(h *GatewayHandler, c *gin.Context, message string) {
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", message, false)
			},
		},
		{
			name: "chat completions (gateway_handler_chat_completions.go)",
			write: func(h *GatewayHandler, c *gin.Context, message string) {
				h.chatCompletionsErrorResponse(c, http.StatusServiceUnavailable, "api_error", message)
			},
		},
		{
			name: "responses (gateway_handler_responses.go)",
			write: func(h *GatewayHandler, c *gin.Context, message string) {
				h.responsesErrorResponse(c, http.StatusServiceUnavailable, "api_error", message)
			},
		},
		{
			name: "gemini google error (gemini_v1beta_handler.go)",
			write: func(h *GatewayHandler, c *gin.Context, message string) {
				googleError(c, http.StatusServiceUnavailable, message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &GatewayHandler{cfg: &config.Config{
				Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": "custom profit control message"}},
			}}
			c := newGatewaySelectionTestContext(t, h)
			tt.write(h, c, profitVetoExhaustedText(h.cfg))

			status, msg := recorderStatusAndMessage(c)
			require.Equal(t, http.StatusServiceUnavailable, status)
			require.Equal(t, "custom profit control message", msg)
		})
	}
}

// ---------------------------------------------------------------------------
// OpenAI 侧四个调度循环的小池路由（recordOpenAIProfitVetoTracked + 判定 + 出口）
// ---------------------------------------------------------------------------

type openAIProfitVetoRouteResult struct {
	outcome     string // "forwarded" | "veto_503" | "generic_502"
	forwardedID int64
	iterations  int
	status      int
	message     string
}

// runOpenAIProfitVetoRoute 复刻 OpenAI 侧四个调度循环（openai_gateway_handler.go
// Responses/Messages、openai_chat_completions.go、grok_media.go 同构）在
// 「选号 → 槽位终检否决 → 排除重选 → 选号报错」处的决策形状：否决走
// recordOpenAIProfitVetoTracked（同时登记否决集），选号报错时先检查纯否决耗尽
// （openAISelectionExhaustedByProfitVeto）→ handleOpenAIProfitVetoExhausted 503，
// 否则回落通用 502。
func newOpenAIProfitVetoTestContext(t *testing.T) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("_test_recorder", w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	return c
}

func runOpenAIProfitVetoRoute(
	t *testing.T,
	h *OpenAIGatewayHandler,
	pool []int64,
	vetoed map[int64]bool,
	maxIterations int,
) openAIProfitVetoRouteResult {
	t.Helper()
	res := openAIProfitVetoRouteResult{}
	failed := make(map[int64]struct{})
	vetoedSet := make(map[int64]struct{})
	count := 0

	for res.iterations = 1; res.iterations <= maxIterations; res.iterations++ {
		var picked int64
		for _, id := range pool {
			if _, excluded := failed[id]; !excluded {
				picked = id
				break
			}
		}
		if picked == 0 {
			// 选号报错分支：先检查纯利润否决耗尽。
			if openAISelectionExhaustedByProfitVeto(failed, vetoedSet) {
				c := newOpenAIProfitVetoTestContext(t)
				h.handleOpenAIProfitVetoExhausted(c, false, zap.NewNop(), count)
				res.outcome = "veto_503"
				res.status, res.message = recorderStatusAndMessage(c)
				return res
			}
			// 通用选号耗尽：没有失败错误信息时按 502 处理。
			c := newOpenAIProfitVetoTestContext(t)
			h.handleFailoverExhaustedSimple(c, http.StatusBadGateway, false)
			res.outcome = "generic_502"
			res.status, res.message = recorderStatusAndMessage(c)
			return res
		}
		if vetoed[picked] {
			if !recordOpenAIProfitVetoTracked(failed, vetoedSet, picked, &count) {
				c := newOpenAIProfitVetoTestContext(t)
				h.handleOpenAIProfitVetoExhausted(c, false, zap.NewNop(), count)
				res.outcome = "veto_503"
				res.status, res.message = recorderStatusAndMessage(c)
				return res
			}
			continue
		}
		res.outcome = "forwarded"
		res.forwardedID = picked
		return res
	}
	res.outcome = "budget_exceeded"
	return res
}

func TestOpenAISelectionExhaustionByProfitVetoSmallPool(t *testing.T) {
	t.Run("one-account pool all vetoed → 503 profit message", func(t *testing.T) {
		h := &OpenAIGatewayHandler{cfg: &config.Config{}}
		res := runOpenAIProfitVetoRoute(t, h, []int64{1}, map[int64]bool{1: true}, 20)
		require.Equal(t, "veto_503", res.outcome)
		require.Equal(t, http.StatusServiceUnavailable, res.status)
		require.Equal(t, profitVetoExhaustedMessage, res.message)
	})

	t.Run("two-account pool all vetoed → 503 profit message", func(t *testing.T) {
		h := &OpenAIGatewayHandler{cfg: &config.Config{}}
		res := runOpenAIProfitVetoRoute(t, h, []int64{1, 2}, map[int64]bool{1: true, 2: true}, 20)
		require.Equal(t, "veto_503", res.outcome)
		require.Equal(t, http.StatusServiceUnavailable, res.status)
		require.Equal(t, profitVetoExhaustedMessage, res.message)
	})

	t.Run("custom 503 override wins on small-pool veto exhaustion", func(t *testing.T) {
		const override503 = "custom profit control message"
		h := &OpenAIGatewayHandler{cfg: &config.Config{
			Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": override503}},
		}}
		res := runOpenAIProfitVetoRoute(t, h, []int64{3}, map[int64]bool{3: true}, 20)
		require.Equal(t, "veto_503", res.outcome)
		require.Equal(t, http.StatusServiceUnavailable, res.status)
		require.Equal(t, override503, res.message)
	})

	t.Run("healthy account forwarded despite one vetoed candidate", func(t *testing.T) {
		h := &OpenAIGatewayHandler{cfg: &config.Config{}}
		res := runOpenAIProfitVetoRoute(t, h, []int64{1, 2}, map[int64]bool{1: true}, 20)
		require.Equal(t, "forwarded", res.outcome)
		require.Equal(t, int64(2), res.forwardedID)
	})

	t.Run("veto plus real failover exclusion keeps generic 502", func(t *testing.T) {
		// 账号 2 以真实 failover 失败身份加入排除集（不走否决记账）：判定不得
		// 误判为纯否决耗尽，回落通用 502。
		h := &OpenAIGatewayHandler{cfg: &config.Config{}}
		failed := map[int64]struct{}{1: {}, 2: {}}
		vetoedSet := map[int64]struct{}{1: {}}
		require.False(t, openAISelectionExhaustedByProfitVeto(failed, vetoedSet))

		c := newOpenAIProfitVetoTestContext(t)
		h.handleFailoverExhaustedSimple(c, http.StatusBadGateway, false)
		status, _ := recorderStatusAndMessage(c)
		require.Equal(t, http.StatusBadGateway, status)
	})
}
