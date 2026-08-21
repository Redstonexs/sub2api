//go:build integration

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLog_UpstreamModelMismatchFilterAndPartialIndex(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	repo := newUsageLogRepositoryWithSQL(client, tx)

	user := mustCreateUser(t, client, &service.User{Email: "model-audit@test.com"})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: "sk-model-audit", Name: "model-audit"})
	account := mustCreateAccount(t, client, &service.Account{Name: "model-audit-account"})
	now := time.Now().UTC()
	responseModel := "gpt-5.4"
	for _, mismatch := range []bool{true, false} {
		mismatchValue := mismatch
		_, err := repo.Create(ctx, &service.UsageLog{
			UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID,
			Model: "gpt-5.5", InputTokens: 1, OutputTokens: 1,
			UpstreamResponseModel: &responseModel, UpstreamModelMismatch: &mismatchValue,
			CreatedAt: now,
		})
		require.NoError(t, err)
	}

	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	trueValue := true
	stats, err := repo.GetStatsWithFilters(ctx, usagestats.UsageLogFilters{
		UserID: user.ID, StartTime: &start, EndTime: &end, UpstreamModelMismatch: &trueValue,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalRequests)
	require.Equal(t, []usagestats.EndpointStat{{
		Endpoint: "unknown", Requests: 1, TotalTokens: 2,
	}}, stats.Endpoints)
	require.Equal(t, []usagestats.EndpointStat{{
		Endpoint: "unknown", Requests: 1, TotalTokens: 2,
	}}, stats.UpstreamEndpoints)
	require.Equal(t, []usagestats.EndpointStat{{
		Endpoint: "unknown -> unknown", Requests: 1, TotalTokens: 2,
	}}, stats.EndpointPaths)

	trend, err := repo.GetUsageTrendWithUsageFilters(ctx, start, end, "hour", usagestats.UsageLogFilters{
		UserID: user.ID, UpstreamModelMismatch: &trueValue,
	})
	require.NoError(t, err)
	require.Len(t, trend, 1)
	require.Equal(t, int64(1), trend[0].Requests)

	_, err = tx.ExecContext(ctx, "SET LOCAL enable_seqscan = off")
	require.NoError(t, err)
	assertPlanUsesIndex := func(query, indexName string, args ...any) {
		rows, queryErr := tx.QueryContext(ctx, query, args...)
		require.NoError(t, queryErr)
		var planLines []string
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			planLines = append(planLines, line)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		require.Contains(t, strings.Join(planLines, "\n"), indexName)
	}
	assertPlanUsesIndex(`
EXPLAIN (COSTS OFF)
SELECT id
FROM usage_logs
WHERE upstream_model_mismatch IS TRUE
ORDER BY created_at DESC, id DESC
LIMIT 100
`, usageLogsUpstreamModelMismatchIndex)
	assertPlanUsesIndex(`
EXPLAIN (COSTS OFF)
SELECT id
FROM usage_logs
WHERE COALESCE(NULLIF(TRIM(requested_model), ''), model) = $1
  AND created_at >= $2 AND created_at < $3
ORDER BY created_at DESC, id DESC
LIMIT 100
`, usageLogsEffectiveRequestedModelIndex, "gpt-5.5", start, end)
	assertPlanUsesIndex(`
EXPLAIN (COSTS OFF)
SELECT id
FROM usage_logs
WHERE COALESCE(NULLIF(TRIM(upstream_model), ''), model) = $1
  AND created_at >= $2 AND created_at < $3
ORDER BY created_at DESC, id DESC
LIMIT 100
`, usageLogsEffectiveUpstreamModelIndex, "gpt-5.5", start, end)
}

func TestUsageLog_GetStatsWithFilters_AggregatesAndEndpoints(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	repo := newUsageLogRepositoryWithSQL(client, tx)

	user := mustCreateUser(t, client, &service.User{Email: "stats@test.com"})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: "sk-stats-1", Name: "k"})
	account := mustCreateAccount(t, client, &service.Account{Name: "acc-stats"})

	now := time.Now().UTC()
	inboundEndpoint := "/v1/messages"
	upstreamEndpoint := "/v1/responses"
	for i := 0; i < 3; i++ {
		_, err := repo.Create(ctx, &service.UsageLog{
			UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID,
			Model: "claude-3", InputTokens: 2, OutputTokens: 3,
			CacheCreationTokens: 4, CacheReadTokens: 5,
			TotalCost: 0.5, ActualCost: 0.4, CreatedAt: now,
			InboundEndpoint: &inboundEndpoint, UpstreamEndpoint: &upstreamEndpoint,
		})
		require.NoError(t, err)
	}

	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)
	// 按本测试创建的 user 维度过滤:集成库为共享实例,其它用 testEntClient 的兄弟测试会留下
	// 已提交的 usage_log 行(含零 token 的失败请求),不限定 user 会把它们计入 TotalRequests。
	stats, err := repo.GetStatsWithFilters(ctx, usagestats.UsageLogFilters{UserID: user.ID, StartTime: &start, EndTime: &end})
	require.NoError(t, err)
	require.Equal(t, int64(3), stats.TotalRequests)
	require.Equal(t, int64(6), stats.TotalInputTokens)
	require.Equal(t, int64(9), stats.TotalOutputTokens)
	require.Equal(t, int64(27), stats.TotalCacheTokens)
	require.Equal(t, int64(12), stats.TotalCacheCreationTokens)
	require.Equal(t, int64(15), stats.TotalCacheReadTokens)
	require.InDelta(t, 1.2, stats.TotalActualCost, 1e-9)
	require.NotEmpty(t, stats.Endpoints)
	require.NotEmpty(t, stats.UpstreamEndpoints)
	require.NotEmpty(t, stats.EndpointPaths)
}

func TestUsageLog_GetAccountWindowStatsBatchByWindows(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	repo := newUsageLogRepositoryWithSQL(client, tx)

	user := mustCreateUser(t, client, &service.User{Email: "batch-windows@test.com"})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: "sk-batch-windows", Name: "k"})
	accountA := mustCreateAccount(t, client, &service.Account{Name: "acc-batch-windows-a"})
	accountB := mustCreateAccount(t, client, &service.Account{Name: "acc-batch-windows-b"})

	now := time.Now().UTC()
	// accountA: two logs inside the 5h window, one log before it (excluded).
	_, err := repo.Create(ctx, &service.UsageLog{
		UserID: user.ID, APIKeyID: apiKey.ID, AccountID: accountA.ID,
		Model: "claude-3", InputTokens: 2, OutputTokens: 3,
		CacheCreationTokens: 4, CacheReadTokens: 5,
		TotalCost: 0.5, ActualCost: 0.4, CreatedAt: now,
	})
	require.NoError(t, err)
	_, err = repo.Create(ctx, &service.UsageLog{
		UserID: user.ID, APIKeyID: apiKey.ID, AccountID: accountA.ID,
		Model: "claude-3", InputTokens: 10, OutputTokens: 20,
		CacheCreationTokens: 0, CacheReadTokens: 0,
		TotalCost: 1.0, ActualCost: 0.8, CreatedAt: now.Add(-time.Hour),
	})
	require.NoError(t, err)
	_, err = repo.Create(ctx, &service.UsageLog{
		UserID: user.ID, APIKeyID: apiKey.ID, AccountID: accountA.ID,
		Model: "claude-3", InputTokens: 100, OutputTokens: 100,
		CacheCreationTokens: 0, CacheReadTokens: 0,
		TotalCost: 9.0, ActualCost: 7.0, CreatedAt: now.Add(-10 * time.Hour),
	})
	require.NoError(t, err)

	// accountB: no logs at all (zero-log window must return zero-valued stats).
	_ = accountB

	requests := []usagestats.AccountWindowStatsRequest{
		{AccountID: accountA.ID, WindowKey: "5h", StartTime: now.Add(-5 * time.Hour)},
		{AccountID: accountA.ID, WindowKey: "7d", StartTime: now.Add(-7 * 24 * time.Hour)},
		{AccountID: accountB.ID, WindowKey: "5h", StartTime: now.Add(-5 * time.Hour)},
	}
	got, err := repo.GetAccountWindowStatsBatchByWindows(ctx, requests)
	require.NoError(t, err)

	// accountA 5h: two logs (now, now-1h); the now-10h log is excluded.
	a5h := got[usagestats.AccountWindowStatsKey{AccountID: accountA.ID, WindowKey: "5h"}]
	require.NotNil(t, a5h)
	require.Equal(t, int64(2), a5h.Requests)
	require.Equal(t, int64(2+3+4+5+10+20), a5h.Tokens)
	require.InDelta(t, 1.5, a5h.Cost, 1e-9)
	require.InDelta(t, 1.5, a5h.StandardCost, 1e-9)
	require.InDelta(t, 1.2, a5h.UserCost, 1e-9)

	// accountA 7d: all three logs.
	a7d := got[usagestats.AccountWindowStatsKey{AccountID: accountA.ID, WindowKey: "7d"}]
	require.NotNil(t, a7d)
	require.Equal(t, int64(3), a7d.Requests)
	require.Equal(t, int64(2+3+4+5+10+20+100+100), a7d.Tokens)
	require.InDelta(t, 10.5, a7d.Cost, 1e-9)
	require.InDelta(t, 10.5, a7d.StandardCost, 1e-9)
	require.InDelta(t, 8.2, a7d.UserCost, 1e-9)

	// accountB 5h: zero-log window returns zero-valued stats.
	b5h := got[usagestats.AccountWindowStatsKey{AccountID: accountB.ID, WindowKey: "5h"}]
	require.NotNil(t, b5h)
	require.Equal(t, int64(0), b5h.Requests)
	require.Equal(t, int64(0), b5h.Tokens)
	require.InDelta(t, 0.0, b5h.Cost, 1e-9)
	require.InDelta(t, 0.0, b5h.StandardCost, 1e-9)
	require.InDelta(t, 0.0, b5h.UserCost, 1e-9)

	// Cross-check against the single-account query for the same window.
	single, err := repo.GetAccountWindowStats(ctx, accountA.ID, now.Add(-5*time.Hour))
	require.NoError(t, err)
	require.Equal(t, single.Requests, a5h.Requests)
	require.Equal(t, single.Tokens, a5h.Tokens)
	require.InDelta(t, single.Cost, a5h.Cost, 1e-9)
	require.InDelta(t, single.StandardCost, a5h.StandardCost, 1e-9)
	require.InDelta(t, single.UserCost, a5h.UserCost, 1e-9)
}
