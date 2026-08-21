package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// codexProfileAlignment 控制 Codex 出站请求档案对齐。
//
// 真实 Codex 客户端在一次请求里把同一个会话 UUID 同时写进 session_id /
// conversation_id / session-id / thread-id 头、body prompt_cache_key、
// client_metadata.session_id 与 x-codex-turn-metadata；网关历史实现只改写了
// 其中的 header 载体，且用的是 isolateOpenAISessionID 的 16 位十六进制哈希，
// 与 UUID 形态不符，也与 body 侧留存的客户端原值互相矛盾。
//
// 开启后所有载体收敛到同一个 UUID；关闭后逐字回到对齐前的行为，供上游策略
// 变动时回滚（gateway.disable_codex_profile_alignment）。
var codexProfileAlignment = func() *atomic.Bool {
	v := &atomic.Bool{}
	v.Store(true)
	return v
}()

// SetCodexProfileAlignmentEnabled 发布 Codex 出站档案对齐开关。
// 与 SetCodexIdentityEnforcementEnabled 同构：对齐点是所有出站路径共用的纯函数，
// 无法在热路径注入配置，故由持有配置的服务在构造时发布进程级快照。
func SetCodexProfileAlignmentEnabled(enabled bool) {
	codexProfileAlignment.Store(enabled)
}

func codexProfileAlignmentEnabled() bool {
	return codexProfileAlignment.Load()
}

// codexUpstreamSessionID 是所有出站会话标识的唯一构造点：先用 apiKeyID 隔离
// （防跨用户会话碰撞），再套上 UUID 形态（真实 Codex 的 session_id 恒为 UUID）。
//
// isolateOpenAISessionID 本身保持不变——CyberSessionBlockKey 与粘性会话键都依赖
// 它当前的输出，改动会让在途的屏蔽/粘性状态整体失效。
func codexUpstreamSessionID(apiKeyID int64, raw string) string {
	isolated := isolateOpenAISessionID(apiKeyID, raw)
	if isolated == "" || !codexProfileAlignmentEnabled() {
		return isolated
	}
	return generateSessionUUID(isolated)
}

// codexSessionIdentityContextKey 是暂存在 gin context 的出站会话身份键。
const codexSessionIdentityContextKey = "codex_session_identity"

// codexOutboundSessionIdentity 一次入站请求对应的出站会话身份。
// 三个字段的关系镜像 resolveCodexFingerprintIDs：thread 与 session 同值，
// window 为 `{thread}:0`。
type codexOutboundSessionIdentity struct {
	sessionID string
	threadID  string
	windowID  string
}

func newCodexOutboundSessionIdentity(apiKeyID int64, promptCacheKey string) *codexOutboundSessionIdentity {
	if !codexProfileAlignmentEnabled() {
		return nil
	}
	sessionID := codexUpstreamSessionID(apiKeyID, promptCacheKey)
	if sessionID == "" {
		return nil
	}
	return &codexOutboundSessionIdentity{
		sessionID: sessionID,
		threadID:  sessionID,
		windowID:  sessionID + ":0",
	}
}

func stageCodexSessionIdentity(c *gin.Context, id *codexOutboundSessionIdentity) {
	if c != nil {
		c.Set(codexSessionIdentityContextKey, id)
	}
}

func stagedCodexSessionIdentity(c *gin.Context) *codexOutboundSessionIdentity {
	if c == nil {
		return nil
	}
	value, ok := c.Get(codexSessionIdentityContextKey)
	if !ok {
		return nil
	}
	id, ok := value.(*codexOutboundSessionIdentity)
	if !ok {
		return nil
	}
	return id
}

// resolveCodexOutboundSessionIdentity 解析出站会话身份，首次解析后固定在 gin context 上。
//
// 固定是必须的，不是缓存优化：body 侧改写会把 prompt_cache_key 换成派生出的
// sessionID，failover 重入 Forward 时若按新 body 重新派生，就会得到
// isolate(isolate(原值)) 这一层套一层的第二个会话——同一次客户端会话在换号重试
// 后变成两个上游会话。与指纹收敛的 stage 语义不同：会话身份只取决于入站请求
// （apiKeyID + 客户端会话键），与选中哪个账号无关，因此跨 attempt 保持不变即为正确。
func resolveCodexOutboundSessionIdentity(c *gin.Context, apiKeyID int64, promptCacheKey string) *codexOutboundSessionIdentity {
	if existing := stagedCodexSessionIdentity(c); existing != nil {
		return existing
	}
	id := newCodexOutboundSessionIdentity(apiKeyID, promptCacheKey)
	if id != nil {
		stageCodexSessionIdentity(c, id)
	}
	return id
}

// codexSessionIdentityMetadataFields 是需要与出站会话身份保持一致的
// turn metadata 字段。turn_id / sandbox / thread_source 等不在其列：
// 它们与会话身份无关，rewriteCodexTurnMetadataFields 会原样保留。
func (id *codexOutboundSessionIdentity) metadataFields() map[string]any {
	return map[string]any{
		"session_id": id.sessionID,
		"thread_id":  id.threadID,
		"window_id":  id.windowID,
	}
}

// applyCodexSessionIdentityHeaders 把出站会话身份写进所有 header 载体。
//
// session_id / session-id / thread-id 无条件写入：出站身份已经声明为 Codex 客户端
// （User-Agent / originator / version），真实客户端必然携带这三个头，而入站白名单
// 会丢掉客户端自己的值（原样透传会造成跨用户碰撞，故只能由本函数重建）。
// x-codex-window-id 与 x-codex-turn-metadata 只在客户端已经携带时校正——补齐一个
// 原本不存在的载体会改变非 Codex 客户端的请求形态，超出「消除自相矛盾」的范围。
//
// 必须在指纹收敛（applyStagedCodexFingerprintHeaders）之前调用：收敛是显式 opt-in
// 的更强策略，session/full 模式下必须能覆盖本函数的结果。
func applyCodexSessionIdentityHeaders(h http.Header, id *codexOutboundSessionIdentity) {
	if h == nil || id == nil {
		return
	}
	h.Set("session_id", id.sessionID)
	h.Set("session-id", id.sessionID)
	h.Set("thread-id", id.threadID)
	if strings.TrimSpace(h.Get("x-codex-window-id")) != "" {
		h.Set("x-codex-window-id", id.windowID)
	}
	rewriteCodexTurnMetadataFields(h, id.metadataFields())
}

// applyCodexSessionIdentityBody 把出站会话身份写进 body 载体。
// prompt_cache_key 与 header session_id 在真实客户端里恒为同一个值；顺带让
// 「按 apiKeyID 隔离会话」这层保护第一次真正作用到上游缓存键上。
// client_metadata 只校正已存在的 session_id，不凭空创建该对象。
func applyCodexSessionIdentityBody(reqBody map[string]any, id *codexOutboundSessionIdentity) bool {
	if reqBody == nil || id == nil {
		return false
	}
	modified := false
	if key, ok := reqBody["prompt_cache_key"].(string); ok && strings.TrimSpace(key) != "" && key != id.sessionID {
		reqBody["prompt_cache_key"] = id.sessionID
		modified = true
	}
	existing, _ := reqBody["client_metadata"].(map[string]any)
	if existing == nil {
		return modified
	}
	if _, ok := existing["session_id"]; ok {
		existing["session_id"] = id.sessionID
		existing["thread_id"] = id.threadID
		modified = true
	}
	rewriteClientMetadataEmbeddedTurnMetadata(existing, id.metadataFields())
	return modified
}

// applyCodexSessionIdentityBodyRaw 在原始 JSON 字节上完成与
// applyCodexSessionIdentityBody 等价的改写，供透传热路径使用——透传 body 可达数十
// MB，禁止全量 Unmarshal（见 forwardOpenAIPassthrough 的轻量提取注释）。
func applyCodexSessionIdentityBodyRaw(body []byte, id *codexOutboundSessionIdentity) ([]byte, bool, error) {
	if len(body) == 0 || id == nil {
		return body, false, nil
	}
	// 非 JSON 对象的 body 没有会话字段语义，sjson 在这类根上写字段会改写整体结构。
	if !gjson.ParseBytes(body).IsObject() {
		return body, false, nil
	}

	next := body
	modified := false

	if key := gjson.GetBytes(body, "prompt_cache_key"); key.Exists() && key.Type == gjson.String &&
		strings.TrimSpace(key.String()) != "" && key.String() != id.sessionID {
		rewritten, err := sjson.SetBytes(next, "prompt_cache_key", id.sessionID)
		if err != nil {
			return body, false, fmt.Errorf("splice aligned prompt_cache_key: %w", err)
		}
		next = rewritten
		modified = true
	}

	cm := gjson.GetBytes(next, "client_metadata")
	if !cm.IsObject() {
		return next, modified, nil
	}
	existing := map[string]any{}
	if err := json.Unmarshal([]byte(cm.Raw), &existing); err != nil {
		return body, false, fmt.Errorf("decode client_metadata for session identity: %w", err)
	}
	changed := false
	if _, ok := existing["session_id"]; ok {
		existing["session_id"] = id.sessionID
		existing["thread_id"] = id.threadID
		changed = true
	}
	before, _ := existing["x-codex-turn-metadata"].(string)
	rewriteClientMetadataEmbeddedTurnMetadata(existing, id.metadataFields())
	if after, _ := existing["x-codex-turn-metadata"].(string); after != before {
		changed = true
	}
	if !changed {
		return next, modified, nil
	}
	raw, err := json.Marshal(existing)
	if err != nil {
		return body, false, fmt.Errorf("encode aligned client_metadata: %w", err)
	}
	spliced, err := sjson.SetRawBytes(next, "client_metadata", raw)
	if err != nil {
		return body, false, fmt.Errorf("splice aligned client_metadata: %w", err)
	}
	return spliced, true, nil
}

// codexInstallationIDBodyPath 是 client_metadata 里安装标识的 JSON 路径。
const codexInstallationIDBodyPath = "client_metadata.x-codex-installation-id"

// applyCodexInstallationIDHeader 让出站头的安装标识跟随 body 里的同名字段。
//
// applyCodexClientMetadata 会把账号真实的 openai_device_id 补进 body 的
// client_metadata（且绝不覆盖客户端既有值），但历史实现没有对应的头改写——同一个
// 请求于是 body 说设备 A、头说客户端自报的 B。这里统一以 body 为准：body 侧刚由
// 网关按「有真实 device_id 才写、否则保留客户端值」的规则定稿，是权威值。
//
// body 没有该字段时不动头——凭空合成一个账号级恒定安装标识就是指纹收敛，
// 那是显式 opt-in 的策略（见 GetCodexFingerprintMode 与 #5610）。
// 必须在 applyStagedCodexFingerprintHeaders 之前调用，让收敛仍能覆盖。
func applyCodexInstallationIDHeader(h http.Header, account *Account, body []byte) {
	if h == nil || account == nil || !account.IsOpenAIOAuth() || !codexProfileAlignmentEnabled() {
		return
	}
	installationID := strings.TrimSpace(gjson.GetBytes(body, codexInstallationIDBodyPath).String())
	if installationID == "" {
		return
	}
	h.Set("x-codex-installation-id", installationID)
}

// applyCodexClientMetadataRaw 是 applyCodexClientMetadata 的 raw 字节版本，供透传
// 热路径使用——透传 body 可达数十 MB，禁止全量 Unmarshal。语义与 map 版逐点一致：
// 加法式、幂等，仅在账号存在 device_id 且该键缺失时注入，非对象 client_metadata
// 保持原样。
func applyCodexClientMetadataRaw(body []byte, account *Account) ([]byte, bool, error) {
	if len(body) == 0 || account == nil {
		return body, false, nil
	}
	deviceID := strings.TrimSpace(account.GetOpenAIDeviceID())
	if deviceID == "" || !gjson.ParseBytes(body).IsObject() {
		return body, false, nil
	}
	if existing := gjson.GetBytes(body, codexInstallationIDBodyPath); existing.Type == gjson.String &&
		strings.TrimSpace(existing.String()) != "" {
		return body, false, nil
	}
	// 非对象的既有 client_metadata：map 版走 default 分支原样返回，这里对齐该行为，
	// 避免 sjson 把标量整体替换成对象。
	if cm := gjson.GetBytes(body, "client_metadata"); cm.Exists() && !cm.IsObject() {
		return body, false, nil
	}
	next, err := sjson.SetBytes(body, codexInstallationIDBodyPath, deviceID)
	if err != nil {
		return body, false, fmt.Errorf("splice codex installation id: %w", err)
	}
	return next, true, nil
}
