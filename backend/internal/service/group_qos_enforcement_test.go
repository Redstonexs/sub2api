package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func qosRerouteContext(from, to string) context.Context {
	return WithGroupQoSDecision(context.Background(), &GroupQoSDecision{
		ModelMappings: []GroupQoSModelMapping{{From: from, To: to}},
	})
}

// seedChannelCache installs a ready-made cache so ResolveChannelMapping never
// touches a repository.
func seedChannelCache(t *testing.T, cache *channelCache) *ChannelService {
	t.Helper()
	if cache == nil {
		cache = &channelCache{}
	}
	cache.loadedAt = time.Now()
	svc := &ChannelService{}
	svc.cache.Store(cache)
	return svc
}

// The overwhelming majority of groups have no channel bound, which is the code
// path that returns early before channel mapping. QoS must still degrade there —
// if it did not, the feature would silently do nothing for most deployments.
func TestQoSRerouteAppliesWithNoChannelBound(t *testing.T) {
	svc := seedChannelCache(t, nil)
	ctx := qosRerouteContext("gpt-5.6-sol", "gpt-5.6-luna")

	t.Run("ResolveChannelMapping", func(t *testing.T) {
		result := svc.ResolveChannelMapping(ctx, 1, "gpt-5.6-sol")
		require.Equal(t, "gpt-5.6-luna", result.MappedModel)
		require.True(t, result.Mapped)
		require.True(t, result.QoSApplied)
	})

	t.Run("ResolveChannelMappingAndRestrict", func(t *testing.T) {
		groupID := int64(1)
		result, _ := svc.ResolveChannelMappingAndRestrict(ctx, &groupID, "gpt-5.6-sol")
		require.Equal(t, "gpt-5.6-luna", result.MappedModel)
		require.True(t, result.Mapped)
		require.True(t, result.QoSApplied)
	})

	t.Run("nil group id still degrades", func(t *testing.T) {
		result, _ := svc.ResolveChannelMappingAndRestrict(ctx, nil, "gpt-5.6-sol")
		require.Equal(t, "gpt-5.6-luna", result.MappedModel)
		require.True(t, result.QoSApplied)
	})

	t.Run("undegraded request is untouched", func(t *testing.T) {
		result := svc.ResolveChannelMapping(context.Background(), 1, "gpt-5.6-sol")
		require.Equal(t, "gpt-5.6-sol", result.MappedModel)
		require.False(t, result.Mapped)
		require.False(t, result.QoSApplied)
	})
}

// QoS runs before channel mapping: QoS picks the product tier, the channel
// decides how this deployment names/routes it.
func TestQoSRerouteComposesWithChannelMapping(t *testing.T) {
	const groupID = int64(7)
	channel := &Channel{ID: 99, Status: StatusActive, BillingModelSource: BillingModelSourceChannelMapped}
	cache := &channelCache{
		channelByGroupID: map[int64]*Channel{groupID: channel},
		groupPlatform:    map[int64]string{groupID: PlatformOpenAI},
		mappingByGroupModel: map[channelModelKey]string{
			{groupID: groupID, platform: PlatformOpenAI, model: "gpt-5.6-luna"}: "vendor-luna-v2",
		},
	}
	svc := seedChannelCache(t, cache)

	result := svc.ResolveChannelMapping(qosRerouteContext("gpt-5.6-sol", "gpt-5.6-luna"), groupID, "gpt-5.6-sol")
	require.True(t, result.QoSApplied)
	require.True(t, result.Mapped)
	// sol --QoS--> luna --channel--> vendor-luna-v2
	require.Equal(t, "vendor-luna-v2", result.MappedModel)
	require.Equal(t, int64(99), result.ChannelID)

	// The audit trail records the whole rewrite, so a degraded request is
	// diagnosable from the usage log alone.
	require.Equal(t, "gpt-5.6-sol→vendor-luna-v2", result.BuildModelMappingChain("gpt-5.6-sol", ""))
}

func TestQoSMappingChainRecordsDegradation(t *testing.T) {
	result := ChannelMappingResult{MappedModel: "gpt-5.6-luna", Mapped: true, QoSApplied: true}
	require.Equal(t, "gpt-5.6-sol→gpt-5.6-luna", result.BuildModelMappingChain("gpt-5.6-sol", ""))
	require.Equal(t, "gpt-5.6-sol→gpt-5.6-luna→upstream-luna",
		result.BuildModelMappingChain("gpt-5.6-sol", "upstream-luna"))
}

func TestQoSAppliedPropagatesToUsageFields(t *testing.T) {
	result := ChannelMappingResult{MappedModel: "gpt-5.6-luna", Mapped: true, QoSApplied: true}
	fields := result.ToUsageFields("gpt-5.6-sol", "gpt-5.6-luna")
	require.True(t, fields.QoSApplied, "billing must be able to see that a QoS reroute happened")
	require.Equal(t, "gpt-5.6-sol", fields.OriginalModel)
	require.Equal(t, "gpt-5.6-luna", fields.ChannelMappedModel)
}

// The rule both billing paths share. A QoS-degraded request must never fall
// back to the requested model's price: the user received the cheaper model, so
// charging the expensive one would be a downgrade at full price.
func TestShouldBillAtRequestedModelIgnoresQoSDegradedRequests(t *testing.T) {
	// Normal "requested" billing is unaffected.
	require.True(t, shouldBillAtRequestedModel(BillingModelSourceRequested, "gpt-5.6-sol", false))
	// A QoS reroute suppresses the fallback.
	require.False(t, shouldBillAtRequestedModel(BillingModelSourceRequested, "gpt-5.6-sol", true))
	// Other sources are untouched either way.
	require.False(t, shouldBillAtRequestedModel(BillingModelSourceChannelMapped, "gpt-5.6-sol", false))
	require.False(t, shouldBillAtRequestedModel(BillingModelSourceUpstream, "gpt-5.6-sol", true))
	// An empty original model never drives billing.
	require.False(t, shouldBillAtRequestedModel(BillingModelSourceRequested, "", false))
}

func TestTightenRPMLimit(t *testing.T) {
	// 0 means "unlimited" on both sides, so it never tightens the other.
	require.Equal(t, 50, tightenRPMLimit(50, 0))
	require.Equal(t, 20, tightenRPMLimit(0, 20))
	require.Equal(t, 0, tightenRPMLimit(0, 0))
	// The stricter limit wins in both directions — a generous group or per-user
	// override must not let a degraded user escape the squeeze.
	require.Equal(t, 20, tightenRPMLimit(50, 20))
	require.Equal(t, 5, tightenRPMLimit(5, 20))
}

func TestGroupQoSEligible(t *testing.T) {
	require.False(t, GroupQoSEligible(nil))
	require.False(t, GroupQoSEligible(&Group{QoSEnabled: false, QoSTiers: []GroupQoSTier{{Window: "daily"}}}))
	require.False(t, GroupQoSEligible(&Group{QoSEnabled: true}), "enabled with an empty ladder is a no-op")
	require.True(t, GroupQoSEligible(&Group{QoSEnabled: true, QoSTiers: []GroupQoSTier{{Window: "daily"}}}))
}

// A counter that cannot be read must never degrade or block: fail open.
func TestGroupQoSResolveDecisionFailsOpen(t *testing.T) {
	group := &Group{
		ID:         1,
		QoSEnabled: true,
		QoSTiers:   []GroupQoSTier{{Window: "daily", ThresholdUSD: 0, Block: true}},
	}

	t.Run("nil service", func(t *testing.T) {
		var svc *GroupQoSService
		require.Nil(t, svc.ResolveDecision(context.Background(), 1, group))
	})

	t.Run("repository error", func(t *testing.T) {
		svc := NewGroupQoSService(&failingQoSRepo{}, nil)
		require.Nil(t, svc.ResolveDecision(context.Background(), 1, group))
	})

	t.Run("no counter yet means no usage", func(t *testing.T) {
		svc := NewGroupQoSService(&emptyQoSRepo{}, nil)
		// Threshold 0 with zero usage still trips (0 >= 0) — proves the lookup
		// succeeded rather than silently failing open.
		require.NotNil(t, svc.ResolveDecision(context.Background(), 1, group))
	})

	t.Run("disabled group short-circuits", func(t *testing.T) {
		svc := NewGroupQoSService(&failingQoSRepo{}, nil)
		require.Nil(t, svc.ResolveDecision(context.Background(), 1, &Group{ID: 1, QoSTiers: group.QoSTiers}))
	})
}

// Expired windows must not keep a user degraded past their reset.
func TestUserGroupQoSUsageEffectiveUsageAt(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-72 * time.Hour)
	fresh := timezone.StartOfDay(now)

	record := &UserGroupQoSUsageRecord{
		DailyUsageUSD:    100,
		WeeklyUsageUSD:   200,
		MonthlyUsageUSD:  300,
		DailyWindowStart: &stale,
	}
	usage := record.EffectiveUsageAt(now)
	require.Zero(t, usage.DailyUSD, "a rolled-over daily window contributes nothing")
	require.Zero(t, usage.WeeklyUSD, "an uninitialized window contributes nothing")
	require.Zero(t, usage.MonthlyUSD)

	record.DailyWindowStart = &fresh
	require.InDelta(t, 100, record.EffectiveUsageAt(now).DailyUSD, 1e-9)

	require.True(t, (&UserGroupQoSUsageRecord{DailyWindowStart: &stale}).HasExpiredWindowAt(now))
	require.False(t, (&UserGroupQoSUsageRecord{DailyWindowStart: &fresh}).HasExpiredWindowAt(now))

	// A 30-day rolling monthly window.
	old := now.Add(-31 * 24 * time.Hour)
	recent := now.Add(-29 * 24 * time.Hour)
	require.True(t, (&UserGroupQoSUsageRecord{MonthlyWindowStart: &old}).HasExpiredWindowAt(now))
	require.False(t, (&UserGroupQoSUsageRecord{MonthlyWindowStart: &recent}).HasExpiredWindowAt(now))
}

type failingQoSRepo struct{}

func (failingQoSRepo) GetByUserGroup(context.Context, int64, int64) (*UserGroupQoSUsageRecord, error) {
	return nil, context.DeadlineExceeded
}
func (failingQoSRepo) IncrementUsageWithReset(context.Context, int64, int64, float64, time.Time) error {
	return nil
}
func (failingQoSRepo) ResetWindows(context.Context, int64, int64, time.Time) error { return nil }

type emptyQoSRepo struct{}

func (emptyQoSRepo) GetByUserGroup(context.Context, int64, int64) (*UserGroupQoSUsageRecord, error) {
	return nil, nil
}
func (emptyQoSRepo) IncrementUsageWithReset(context.Context, int64, int64, float64, time.Time) error {
	return nil
}
func (emptyQoSRepo) ResetWindows(context.Context, int64, int64, time.Time) error { return nil }
