package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// passiveBatchStatsLogRepo implements UsageLogRepository (via embedding) plus
// the optional heterogeneous account-window batch reader port, recording calls
// so tests can assert exactly-one-batch behavior and the requested starts.
type passiveBatchStatsLogRepo struct {
	UsageLogRepository
	calls      int
	requests   []usagestats.AccountWindowStatsRequest
	statsByKey map[usagestats.AccountWindowStatsKey]*usagestats.AccountStats
	err        error
}

func (r *passiveBatchStatsLogRepo) GetAccountWindowStatsBatchByWindows(ctx context.Context, requests []usagestats.AccountWindowStatsRequest) (map[usagestats.AccountWindowStatsKey]*usagestats.AccountStats, error) {
	r.calls++
	r.requests = append(r.requests, requests...)
	if r.err != nil {
		return nil, r.err
	}
	return r.statsByKey, nil
}

func TestAccountUsageService_GetPassiveUsageBatch_AttachesWindowStats(t *testing.T) {
	t.Parallel()

	now := time.Now()
	anthropicSessionStart := now.Add(-3 * time.Hour).Truncate(time.Second)
	anthropicSessionEnd := now.Add(2 * time.Hour).Truncate(time.Second)
	anthropic7dReset := now.Add(1 * time.Hour).Truncate(time.Second)
	openai5hReset := now.Add(30 * time.Minute).Truncate(time.Second)
	openai7dReset := now.Add(2 * time.Hour).Truncate(time.Second)

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:                 8001,
				Platform:           PlatformAnthropic,
				Type:               AccountTypeOAuth,
				SessionWindowStart: &anthropicSessionStart,
				SessionWindowEnd:   &anthropicSessionEnd,
				Extra: map[string]any{
					"passive_usage_7d_utilization": 0.5,
					"passive_usage_7d_reset":       float64(anthropic7dReset.Unix()),
				},
			},
			{
				ID:       8002,
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					"codex_5h_used_percent": 18.0,
					"codex_5h_reset_at":     openai5hReset.Format(time.RFC3339),
					"codex_7d_used_percent": 34.0,
					"codex_7d_reset_at":     openai7dReset.Format(time.RFC3339),
				},
			},
		},
	}

	statsByKey := map[usagestats.AccountWindowStatsKey]*usagestats.AccountStats{
		{AccountID: 8001, WindowKey: "5h"}: {Requests: 10, Tokens: 100, Cost: 1.5, StandardCost: 1.0, UserCost: 0.8},
		{AccountID: 8001, WindowKey: "7d"}: {Requests: 50, Tokens: 500, Cost: 7.5, StandardCost: 5.0, UserCost: 4.0},
		{AccountID: 8002, WindowKey: "5h"}: {Requests: 20, Tokens: 200, Cost: 2.5, StandardCost: 2.0, UserCost: 1.8},
		{AccountID: 8002, WindowKey: "7d"}: {Requests: 60, Tokens: 600, Cost: 8.5, StandardCost: 6.0, UserCost: 5.0},
	}
	logRepo := &passiveBatchStatsLogRepo{statsByKey: statsByKey}

	svc := &AccountUsageService{
		accountRepo:  repo,
		usageLogRepo: logRepo,
		cache:        NewUsageCache(),
	}

	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{8001, 8002})
	if err != nil {
		t.Fatalf("GetPassiveUsageBatch() error = %v", err)
	}

	// No active upstream behavior: both accounts are passive.
	if result[8001] == nil || result[8001].Source != "passive" {
		t.Fatalf("expected anthropic passive usage, got %#v", result[8001])
	}
	if result[8002] == nil || result[8002].Source != "passive" {
		t.Fatalf("expected openai passive usage, got %#v", result[8002])
	}

	// Exactly one batch call.
	if logRepo.calls != 1 {
		t.Fatalf("expected exactly one batch call, got %d", logRepo.calls)
	}

	// Values attached per window.
	assertWindowStats(t, result[8001].FiveHour, statsByKey[usagestats.AccountWindowStatsKey{AccountID: 8001, WindowKey: "5h"}], "anthropic 5h")
	assertWindowStats(t, result[8001].SevenDay, statsByKey[usagestats.AccountWindowStatsKey{AccountID: 8001, WindowKey: "7d"}], "anthropic 7d")
	assertWindowStats(t, result[8002].FiveHour, statsByKey[usagestats.AccountWindowStatsKey{AccountID: 8002, WindowKey: "5h"}], "openai 5h")
	assertWindowStats(t, result[8002].SevenDay, statsByKey[usagestats.AccountWindowStatsKey{AccountID: 8002, WindowKey: "7d"}], "openai 7d")

	// Expected starts.
	got := make(map[usagestats.AccountWindowStatsKey]time.Time, len(logRepo.requests))
	for _, req := range logRepo.requests {
		got[usagestats.AccountWindowStatsKey{AccountID: req.AccountID, WindowKey: req.WindowKey}] = req.StartTime
	}
	expectedStarts := map[usagestats.AccountWindowStatsKey]time.Time{
		{AccountID: 8001, WindowKey: "5h"}: anthropicSessionStart, // current session-window start
		{AccountID: 8001, WindowKey: "7d"}: anthropic7dReset.Add(-7 * 24 * time.Hour),
		{AccountID: 8002, WindowKey: "5h"}: openai5hReset.Add(-5 * time.Hour),
		{AccountID: 8002, WindowKey: "7d"}: openai7dReset.Add(-7 * 24 * time.Hour),
	}
	for key, want := range expectedStarts {
		gotStart, ok := got[key]
		if !ok {
			t.Fatalf("missing request for %+v", key)
		}
		if !gotStart.Equal(want) {
			t.Errorf("start for %+v = %v, want %v", key, gotStart, want)
		}
	}
}

func TestAccountUsageService_GetPassiveUsageBatch_BatchFailurePreservesQuota(t *testing.T) {
	t.Parallel()

	now := time.Now()
	sessionStart := now.Add(-3 * time.Hour).Truncate(time.Second)
	sessionEnd := now.Add(2 * time.Hour).Truncate(time.Second)

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:                 8003,
				Platform:           PlatformAnthropic,
				Type:               AccountTypeOAuth,
				SessionWindowStart: &sessionStart,
				SessionWindowEnd:   &sessionEnd,
				Extra: map[string]any{
					"passive_usage_7d_utilization": 0.5,
					"passive_usage_7d_reset":       float64(now.Add(time.Hour).Unix()),
				},
			},
		},
	}
	logRepo := &passiveBatchStatsLogRepo{err: errors.New("boom")}

	svc := &AccountUsageService{
		accountRepo:  repo,
		usageLogRepo: logRepo,
		cache:        NewUsageCache(),
	}

	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{8003})
	if err != nil {
		t.Fatalf("GetPassiveUsageBatch() error = %v", err)
	}

	// Quota data preserved, no stats attached, exactly one batch call attempted.
	info := result[8003]
	if info == nil || info.FiveHour == nil || info.SevenDay == nil {
		t.Fatalf("expected quota data preserved, got %#v", info)
	}
	if info.FiveHour.WindowStats != nil {
		t.Errorf("expected no 5h window stats on batch failure, got %#v", info.FiveHour.WindowStats)
	}
	if info.SevenDay.WindowStats != nil {
		t.Errorf("expected no 7d window stats on batch failure, got %#v", info.SevenDay.WindowStats)
	}
	if logRepo.calls != 1 {
		t.Errorf("expected exactly one batch call, got %d", logRepo.calls)
	}
}

func TestAccountUsageService_GetPassiveUsageBatch_NoBatchPort(t *testing.T) {
	t.Parallel()

	now := time.Now()
	sessionStart := now.Add(-3 * time.Hour).Truncate(time.Second)
	sessionEnd := now.Add(2 * time.Hour).Truncate(time.Second)

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:                 8004,
				Platform:           PlatformAnthropic,
				Type:               AccountTypeOAuth,
				SessionWindowStart: &sessionStart,
				SessionWindowEnd:   &sessionEnd,
				Extra: map[string]any{
					"passive_usage_7d_utilization": 0.5,
					"passive_usage_7d_reset":       float64(now.Add(time.Hour).Unix()),
				},
			},
		},
	}
	// usageBatchLogRepoStub does NOT implement the heterogeneous batch port.
	svc := &AccountUsageService{
		accountRepo:  repo,
		usageLogRepo: &usageBatchLogRepoStub{},
		cache:        NewUsageCache(),
	}

	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{8004})
	if err != nil {
		t.Fatalf("GetPassiveUsageBatch() error = %v", err)
	}
	info := result[8004]
	if info == nil || info.FiveHour == nil {
		t.Fatalf("expected quota data preserved, got %#v", info)
	}
	if info.FiveHour.WindowStats != nil {
		t.Errorf("expected no window stats without batch port, got %#v", info.FiveHour.WindowStats)
	}
}

func assertWindowStats(t *testing.T, progress *UsageProgress, want *usagestats.AccountStats, label string) {
	t.Helper()
	if progress == nil {
		t.Fatalf("%s: progress is nil", label)
	}
	if progress.WindowStats == nil {
		t.Fatalf("%s: WindowStats is nil", label)
	}
	ws := progress.WindowStats
	if ws.Requests != want.Requests || ws.Tokens != want.Tokens ||
		ws.Cost != want.Cost || ws.StandardCost != want.StandardCost || ws.UserCost != want.UserCost {
		t.Errorf("%s: WindowStats = %+v, want %+v", label, ws, want)
	}
}
