package quotaview

import "github.com/Wei-Shaw/sub2api/internal/service"

// ParseSortWindow 解析分组配额卡片的 sort 查询参数。
// 空串（未提供）默认 5h；仅接受 5h|7d，其余返回 ok=false（调用方应返回 400）。
func ParseSortWindow(raw string) (service.SortWindow, bool) {
	if raw == "" {
		return service.SortWindow5h, true
	}
	switch service.SortWindow(raw) {
	case service.SortWindow5h:
		return service.SortWindow5h, true
	case service.SortWindow7d:
		return service.SortWindow7d, true
	}
	return "", false
}

// SupportsQuotaCard 判断平台是否支持分组配额卡片（仅 Anthropic/OpenAI 平台有 5h/7d 窗口语义）。
func SupportsQuotaCard(platform string) bool {
	switch platform {
	case service.PlatformAnthropic, service.PlatformOpenAI:
		return true
	case service.PlatformGemini, service.PlatformAntigravity, service.PlatformGrok, service.PlatformComposite:
		return false
	default:
		return false
	}
}
