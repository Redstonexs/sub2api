package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupQoSMetric(t *testing.T) {
	require.Equal(t, GroupQoSMetricList, NormalizeGroupQoSMetric(""))
	require.Equal(t, GroupQoSMetricList, NormalizeGroupQoSMetric("list"))
	require.Equal(t, GroupQoSMetricCharged, NormalizeGroupQoSMetric(" CHARGED "))
	// Unknown values fall back to list cost, the safer default.
	require.Equal(t, GroupQoSMetricList, NormalizeGroupQoSMetric("bogus"))
}

func TestNormalizeGroupQoSTiers(t *testing.T) {
	t.Run("canonicalizes windows and mapping sources", func(t *testing.T) {
		tiers, err := NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{
				Window:        " DAILY ",
				ThresholdUSD:  50,
				ModelMappings: []GroupQoSModelMapping{{From: " GPT-5.6-Sol* ", To: "gpt-5.6-terra"}},
			},
		})
		require.NoError(t, err)
		require.Len(t, tiers, 1)
		require.Equal(t, GroupQoSWindowDaily, tiers[0].Window)
		require.Equal(t, "gpt-5.6-sol*", tiers[0].ModelMappings[0].From)
		require.Equal(t, "gpt-5.6-terra", tiers[0].ModelMappings[0].To)
	})

	t.Run("rejects non-ascending thresholds within one window", func(t *testing.T) {
		_, err := NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: 100, Block: true},
			{Window: "daily", ThresholdUSD: 50, Block: true},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be greater than")
	})

	t.Run("allows independent ladders per window", func(t *testing.T) {
		_, err := NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: 100, Block: true},
			{Window: "weekly", ThresholdUSD: 50, Block: true},
		})
		require.NoError(t, err)
	})

	t.Run("rejects unknown window", func(t *testing.T) {
		_, err := NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{Window: "hourly", ThresholdUSD: 1, Block: true},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown window")
	})

	t.Run("rejects a tier with no action", func(t *testing.T) {
		_, err := NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: 10},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one action")
	})

	t.Run("rejects negative threshold and rpm", func(t *testing.T) {
		_, err := NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: -1, Block: true},
		})
		require.Error(t, err)

		_, err = NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: 1, RPMLimit: intPtr(-5)},
		})
		require.Error(t, err)
	})

	t.Run("rejects duplicate and wildcard-target mappings", func(t *testing.T) {
		_, err := NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: 1, ModelMappings: []GroupQoSModelMapping{
				{From: "a", To: "b"}, {From: "A", To: "c"},
			}},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate")

		_, err = NormalizeGroupQoSTiers(PlatformOpenAI, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: 1, ModelMappings: []GroupQoSModelMapping{
				{From: "a", To: "b*"},
			}},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "wildcard")
	})

	t.Run("rejects reasoning effort on a platform that has none", func(t *testing.T) {
		_, err := NormalizeGroupQoSTiers(PlatformAnthropic, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: 1, MaxReasoningEffort: "low"},
		})
		require.Error(t, err)
	})

	t.Run("allows non-effort actions on any platform", func(t *testing.T) {
		tiers, err := NormalizeGroupQoSTiers(PlatformAnthropic, []GroupQoSTier{
			{Window: "daily", ThresholdUSD: 1, ModelMappings: []GroupQoSModelMapping{
				{From: "claude-opus-*", To: "claude-sonnet-4"},
			}},
		})
		require.NoError(t, err)
		require.Len(t, tiers, 1)
	})
}

func TestResolveGroupQoSTier(t *testing.T) {
	ladder := []GroupQoSTier{
		{Window: "daily", ThresholdUSD: 50, ModelMappings: []GroupQoSModelMapping{{From: "sol", To: "terra"}}},
		{Window: "daily", ThresholdUSD: 100, ModelMappings: []GroupQoSModelMapping{{From: "sol", To: "luna"}}},
		{Window: "daily", ThresholdUSD: 200, Block: true},
	}

	t.Run("below the first threshold yields no decision", func(t *testing.T) {
		require.Nil(t, ResolveGroupQoSTier(GroupQoSUsage{DailyUSD: 49.99}, ladder))
	})

	t.Run("matches at exactly the threshold", func(t *testing.T) {
		d := ResolveGroupQoSTier(GroupQoSUsage{DailyUSD: 50}, ladder)
		require.NotNil(t, d)
		require.Equal(t, 0, d.TierIndex)
		require.Equal(t, "terra", d.ModelMappings[0].To)
	})

	t.Run("highest matching index wins", func(t *testing.T) {
		d := ResolveGroupQoSTier(GroupQoSUsage{DailyUSD: 150}, ladder)
		require.NotNil(t, d)
		require.Equal(t, 1, d.TierIndex)
		require.Equal(t, "luna", d.ModelMappings[0].To)
		require.False(t, d.Block)

		d = ResolveGroupQoSTier(GroupQoSUsage{DailyUSD: 500}, ladder)
		require.NotNil(t, d)
		require.Equal(t, 2, d.TierIndex)
		require.True(t, d.Block)
	})

	t.Run("empty ladder yields no decision", func(t *testing.T) {
		require.Nil(t, ResolveGroupQoSTier(GroupQoSUsage{DailyUSD: 1e9}, nil))
	})

	t.Run("windows are independent", func(t *testing.T) {
		weekly := []GroupQoSTier{{Window: "weekly", ThresholdUSD: 100, Block: true}}
		require.Nil(t, ResolveGroupQoSTier(GroupQoSUsage{DailyUSD: 500}, weekly))
		require.NotNil(t, ResolveGroupQoSTier(GroupQoSUsage{WeeklyUSD: 100}, weekly))
	})

	t.Run("unknown window never trips", func(t *testing.T) {
		bad := []GroupQoSTier{{Window: "hourly", ThresholdUSD: 1, Block: true}}
		require.Nil(t, ResolveGroupQoSTier(GroupQoSUsage{DailyUSD: 1e9}, bad))
	})
}

func TestApplyGroupQoSModelMapping(t *testing.T) {
	bind := func(mappings []GroupQoSModelMapping) context.Context {
		return WithGroupQoSDecision(context.Background(), &GroupQoSDecision{ModelMappings: mappings})
	}

	t.Run("unbound context leaves the model alone", func(t *testing.T) {
		got, changed := ApplyGroupQoSModelMapping(context.Background(), "gpt-5.6-sol")
		require.Equal(t, "gpt-5.6-sol", got)
		require.False(t, changed)
	})

	t.Run("exact match", func(t *testing.T) {
		ctx := bind([]GroupQoSModelMapping{{From: "gpt-5.6-sol", To: "gpt-5.6-luna"}})
		got, changed := ApplyGroupQoSModelMapping(ctx, "GPT-5.6-Sol")
		require.True(t, changed)
		require.Equal(t, "gpt-5.6-luna", got)
	})

	t.Run("wildcard prefix match", func(t *testing.T) {
		ctx := bind([]GroupQoSModelMapping{{From: "gpt-5.6-sol*", To: "gpt-5.6-luna"}})
		got, changed := ApplyGroupQoSModelMapping(ctx, "gpt-5.6-sol-high")
		require.True(t, changed)
		require.Equal(t, "gpt-5.6-luna", got)
	})

	t.Run("exact match beats wildcard regardless of order", func(t *testing.T) {
		ctx := bind([]GroupQoSModelMapping{
			{From: "gpt-5.6-sol*", To: "wildcard-target"},
			{From: "gpt-5.6-sol-high", To: "exact-target"},
		})
		got, _ := ApplyGroupQoSModelMapping(ctx, "gpt-5.6-sol-high")
		require.Equal(t, "exact-target", got)
	})

	t.Run("no match leaves the model alone", func(t *testing.T) {
		ctx := bind([]GroupQoSModelMapping{{From: "gpt-5.6-sol", To: "gpt-5.6-luna"}})
		got, changed := ApplyGroupQoSModelMapping(ctx, "claude-opus-4")
		require.Equal(t, "claude-opus-4", got)
		require.False(t, changed)
	})

	t.Run("self-mapping reports no change", func(t *testing.T) {
		ctx := bind([]GroupQoSModelMapping{{From: "sol", To: "sol"}})
		_, changed := ApplyGroupQoSModelMapping(ctx, "sol")
		require.False(t, changed)
	})
}

func TestWithGroupQoSDecisionCopiesMappings(t *testing.T) {
	mappings := []GroupQoSModelMapping{{From: "sol", To: "luna"}}
	limit := 20
	ctx := WithGroupQoSDecision(context.Background(), &GroupQoSDecision{
		ModelMappings: mappings,
		RPMLimit:      &limit,
	})

	// Mutating the caller's slice and pointer must not affect the bound decision.
	mappings[0].To = "mutated"
	limit = 999

	bound := GroupQoSDecisionFromContext(ctx)
	require.NotNil(t, bound)
	require.Equal(t, "luna", bound.ModelMappings[0].To)
	require.Equal(t, 20, *bound.RPMLimit)
}

func TestEffectiveMaxReasoningEffort(t *testing.T) {
	require.Equal(t, "high", EffectiveMaxReasoningEffort("high", nil))
	require.Equal(t, "high", EffectiveMaxReasoningEffort("high", &GroupQoSDecision{}))
	// QoS-only ceiling.
	require.Equal(t, "low", EffectiveMaxReasoningEffort("", &GroupQoSDecision{MaxReasoningEffort: "low"}))
	// The stricter ceiling wins in both directions.
	require.Equal(t, "low", EffectiveMaxReasoningEffort("high", &GroupQoSDecision{MaxReasoningEffort: "low"}))
	require.Equal(t, "low", EffectiveMaxReasoningEffort("low", &GroupQoSDecision{MaxReasoningEffort: "high"}))
}
