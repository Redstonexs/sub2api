package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// LoadGatewayErrorMessages 从 DB 读取后台可编辑的 gateway_error_messages 设置，
// 并以 clone-on-write 快照写入 config 的运行时覆盖（模式同 LoadForwardedClientIPSettings）。
//
// 语义：
//   - 设置缺失（ErrSettingNotFound）或值为空白 / "{}" → 视为无 DB 覆盖，清除任何先前的
//     运行时覆盖，使 GatewayErrorMessage 回退到静态 cfg.Gateway.ErrorMessages。
//   - JSON 非法 → 记录日志并按无覆盖处理（与 parseSettings 语义一致）。
//   - 真实 DB 错误 → 返回错误且不触碰现有运行时覆盖，避免一次瞬时失败清掉可用覆盖。
//
// 该方法同时用于 ProvideSettingService 初始化加载与设置刷新后的收敛，幂等且可重复调用。
func (s *SettingService) LoadGatewayErrorMessages(ctx context.Context) error {
	if s == nil || s.cfg == nil || s.settingRepo == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	raw, err := s.settingRepo.GetValue(ctx, SettingKeyGatewayErrorMessages)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			// 键不存在 = 从未设置过或已被删除：清掉覆盖，回退静态配置。
			s.cfg.SetGatewayErrorMessages(nil)
			return nil
		}
		return fmt.Errorf("get gateway error messages setting: %w", err)
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "{}" {
		s.cfg.SetGatewayErrorMessages(nil)
		return nil
	}

	parsed := make(map[string]string)
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		slog.Warn("[Setting] load gateway_error_messages: invalid JSON, falling back to static config", "error", err)
		s.cfg.SetGatewayErrorMessages(nil)
		return nil
	}

	s.cfg.SetGatewayErrorMessages(parsed)
	return nil
}
