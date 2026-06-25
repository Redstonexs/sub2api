package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const statusClientClosedRequest = 499

func concurrencyErrorResponse(cfg *config.Config, err error, slotType string) (int, string, string) {
	var waitQueueFullErr *WaitQueueFullError
	if errors.As(err, &waitQueueFullErr) {
		return http.StatusTooManyRequests, "rate_limit_error",
			config.GatewayErrorMessage(cfg, http.StatusTooManyRequests, "Too many pending requests, please retry later")
	}

	var concurrencyErr *ConcurrencyError
	if errors.As(err, &concurrencyErr) {
		if concurrencyErr.SlotType != "" {
			slotType = concurrencyErr.SlotType
		}
		return http.StatusTooManyRequests, "rate_limit_error",
			config.GatewayErrorMessage(cfg, http.StatusTooManyRequests, fmt.Sprintf("Concurrency limit exceeded for %s, please retry later", slotType))
	}

	if errors.Is(err, context.Canceled) {
		return statusClientClosedRequest, "api_error", "context canceled"
	}

	return http.StatusServiceUnavailable, "api_error",
		config.GatewayErrorMessage(cfg, http.StatusServiceUnavailable, "Service temporarily unavailable, please retry later")
}
