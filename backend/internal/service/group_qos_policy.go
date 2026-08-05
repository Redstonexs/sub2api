package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type GroupQoSModelMapping = domain.GroupQoSModelMapping
type GroupQoSTier = domain.GroupQoSTier

const (
	GroupQoSMetricList    = domain.GroupQoSMetricList
	GroupQoSMetricCharged = domain.GroupQoSMetricCharged

	GroupQoSWindowDaily   = domain.GroupQoSWindowDaily
	GroupQoSWindowWeekly  = domain.GroupQoSWindowWeekly
	GroupQoSWindowMonthly = domain.GroupQoSWindowMonthly
)

const (
	maxGroupQoSTiers         = 16
	maxGroupQoSModelMappings = 64
	maxGroupQoSModelNameLen  = 200
)

// GroupQoSUsage is a requester's accumulated cost in one group, per rolling
// window. It mirrors the windows behind the group daily/weekly/monthly limits.
type GroupQoSUsage struct {
	DailyUSD   float64
	WeeklyUSD  float64
	MonthlyUSD float64
}

// UsageForWindow returns the accumulated cost for one window name. An unknown
// window yields 0 so a malformed tier can never trip.
func (u GroupQoSUsage) UsageForWindow(window string) float64 {
	switch window {
	case GroupQoSWindowDaily:
		return u.DailyUSD
	case GroupQoSWindowWeekly:
		return u.WeeklyUSD
	case GroupQoSWindowMonthly:
		return u.MonthlyUSD
	default:
		return 0
	}
}

// GroupQoSDecision is the single tier in effect for one request. A nil decision
// means no degradation applies.
type GroupQoSDecision struct {
	TierIndex    int
	Window       string
	ThresholdUSD float64
	UsageUSD     float64

	ModelMappings      []GroupQoSModelMapping
	MaxReasoningEffort string
	RPMLimit           *int
	Block              bool
}

type groupQoSDecisionContextKey struct{}

// WithGroupQoSDecision binds a resolved decision to a request. The mapping slice
// is copied so retries and asynchronous forwarding cannot observe later
// mutations of the group snapshot.
func WithGroupQoSDecision(ctx context.Context, decision *GroupQoSDecision) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if decision == nil {
		return ctx
	}
	bound := *decision
	bound.ModelMappings = append([]GroupQoSModelMapping(nil), decision.ModelMappings...)
	if decision.RPMLimit != nil {
		limit := *decision.RPMLimit
		bound.RPMLimit = &limit
	}
	return context.WithValue(ctx, groupQoSDecisionContextKey{}, &bound)
}

// GroupQoSDecisionFromContext returns the decision bound to a request, or nil
// when the request is not degraded.
func GroupQoSDecisionFromContext(ctx context.Context) *GroupQoSDecision {
	if ctx == nil {
		return nil
	}
	decision, _ := ctx.Value(groupQoSDecisionContextKey{}).(*GroupQoSDecision)
	return decision
}

// NormalizeGroupQoSMetric canonicalizes the per-group threshold metric. Anything
// unrecognized falls back to list cost, the safer default: it counts
// undiscounted consumption, so a deep group discount cannot mask abuse.
func NormalizeGroupQoSMetric(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case GroupQoSMetricCharged:
		return GroupQoSMetricCharged
	default:
		return GroupQoSMetricList
	}
}

func normalizeGroupQoSWindow(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case GroupQoSWindowDaily:
		return GroupQoSWindowDaily, true
	case GroupQoSWindowWeekly:
		return GroupQoSWindowWeekly, true
	case GroupQoSWindowMonthly:
		return GroupQoSWindowMonthly, true
	default:
		return "", false
	}
}

// NormalizeGroupQoSTiers validates and canonicalizes a group's degradation
// ladder. Tiers are stored in ascending severity order; within one window the
// thresholds must strictly increase so that "highest matching index wins" is
// unambiguous.
func NormalizeGroupQoSTiers(platform string, raw []GroupQoSTier) ([]GroupQoSTier, error) {
	if len(raw) > maxGroupQoSTiers {
		return nil, fmt.Errorf("qos tiers cannot exceed %d entries", maxGroupQoSTiers)
	}

	normalized := make([]GroupQoSTier, 0, len(raw))
	lastThresholdByWindow := make(map[string]float64, 3)

	for i, tier := range raw {
		window, ok := normalizeGroupQoSWindow(tier.Window)
		if !ok {
			return nil, fmt.Errorf("qos tier %d has unknown window %q; allowed values: %s, %s, %s",
				i+1, tier.Window, GroupQoSWindowDaily, GroupQoSWindowWeekly, GroupQoSWindowMonthly)
		}
		if math.IsNaN(tier.ThresholdUSD) || math.IsInf(tier.ThresholdUSD, 0) || tier.ThresholdUSD < 0 {
			return nil, fmt.Errorf("qos tier %d threshold must be a non-negative number", i+1)
		}
		if previous, seen := lastThresholdByWindow[window]; seen && tier.ThresholdUSD <= previous {
			return nil, fmt.Errorf(
				"qos tier %d threshold %.4f must be greater than the previous %s tier threshold %.4f",
				i+1, tier.ThresholdUSD, window, previous)
		}
		lastThresholdByWindow[window] = tier.ThresholdUSD

		mappings, err := normalizeGroupQoSModelMappings(tier.ModelMappings)
		if err != nil {
			return nil, fmt.Errorf("qos tier %d: %w", i+1, err)
		}

		effort := ""
		if strings.TrimSpace(tier.MaxReasoningEffort) != "" {
			effort, err = normalizeMaxReasoningEffortForPlatform(platform, tier.MaxReasoningEffort)
			if err != nil {
				return nil, fmt.Errorf("qos tier %d reasoning effort: %w", i+1, err)
			}
		}

		var rpmLimit *int
		if tier.RPMLimit != nil {
			if *tier.RPMLimit < 0 {
				return nil, fmt.Errorf("qos tier %d rpm limit cannot be negative", i+1)
			}
			limit := *tier.RPMLimit
			rpmLimit = &limit
		}

		if len(mappings) == 0 && effort == "" && rpmLimit == nil && !tier.Block {
			return nil, fmt.Errorf("qos tier %d must apply at least one action", i+1)
		}

		normalized = append(normalized, GroupQoSTier{
			Window:             window,
			ThresholdUSD:       tier.ThresholdUSD,
			ModelMappings:      mappings,
			MaxReasoningEffort: effort,
			RPMLimit:           rpmLimit,
			Block:              tier.Block,
		})
	}

	return normalized, nil
}

func normalizeGroupQoSModelMappings(raw []GroupQoSModelMapping) ([]GroupQoSModelMapping, error) {
	if len(raw) > maxGroupQoSModelMappings {
		return nil, fmt.Errorf("model mappings cannot exceed %d entries", maxGroupQoSModelMappings)
	}

	normalized := make([]GroupQoSModelMapping, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for i, mapping := range raw {
		from := strings.ToLower(strings.TrimSpace(mapping.From))
		to := strings.TrimSpace(mapping.To)
		if from == "" || to == "" {
			return nil, fmt.Errorf("model mapping %d requires both a source and a target model", i+1)
		}
		if len(from) > maxGroupQoSModelNameLen || len(to) > maxGroupQoSModelNameLen {
			return nil, fmt.Errorf("model mapping %d names cannot exceed %d characters", i+1, maxGroupQoSModelNameLen)
		}
		// A wildcard target would forward a literal "*" upstream.
		if strings.Contains(to, "*") {
			return nil, fmt.Errorf("model mapping %d target %q cannot contain a wildcard", i+1, mapping.To)
		}
		if _, exists := seen[from]; exists {
			return nil, fmt.Errorf("duplicate model mapping source %q", mapping.From)
		}
		seen[from] = struct{}{}
		normalized = append(normalized, GroupQoSModelMapping{From: from, To: to})
	}
	return normalized, nil
}

// ResolveGroupQoSTier picks the tier in effect for a requester. The highest
// matching index wins, so an admin-ordered ladder degrades progressively.
// Returns nil when no tier is reached.
func ResolveGroupQoSTier(usage GroupQoSUsage, tiers []GroupQoSTier) *GroupQoSDecision {
	var decision *GroupQoSDecision
	for i, tier := range tiers {
		spent := usage.UsageForWindow(tier.Window)
		if spent < tier.ThresholdUSD {
			continue
		}
		decision = &GroupQoSDecision{
			TierIndex:          i,
			Window:             tier.Window,
			ThresholdUSD:       tier.ThresholdUSD,
			UsageUSD:           spent,
			ModelMappings:      tier.ModelMappings,
			MaxReasoningEffort: tier.MaxReasoningEffort,
			RPMLimit:           tier.RPMLimit,
			Block:              tier.Block,
		}
	}
	return decision
}

// ApplyGroupQoSModelMapping rewrites a requested model under the active tier.
// Exact matches win over wildcards; wildcards are tried in configured order.
// Returns the input unchanged when no mapping applies.
func ApplyGroupQoSModelMapping(ctx context.Context, model string) (string, bool) {
	decision := GroupQoSDecisionFromContext(ctx)
	if decision == nil || len(decision.ModelMappings) == 0 {
		return model, false
	}

	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return model, false
	}
	lower := strings.ToLower(trimmed)

	for _, mapping := range decision.ModelMappings {
		if prefix, isWildcard := splitWildcardSuffix(mapping.From); !isWildcard && prefix == lower {
			return mapping.To, mapping.To != trimmed
		}
	}
	for _, mapping := range decision.ModelMappings {
		prefix, isWildcard := splitWildcardSuffix(mapping.From)
		if isWildcard && strings.HasPrefix(lower, prefix) {
			return mapping.To, mapping.To != trimmed
		}
	}
	return model, false
}

// EffectiveMaxReasoningEffort returns the stricter of a group's standing
// ceiling and the active QoS tier's ceiling. Either may be empty (no ceiling).
func EffectiveMaxReasoningEffort(groupCeiling string, decision *GroupQoSDecision) string {
	if decision == nil || decision.MaxReasoningEffort == "" {
		return groupCeiling
	}
	if groupCeiling == "" {
		return decision.MaxReasoningEffort
	}
	groupRank, groupOK := reasoningEffortRank(groupCeiling)
	qosRank, qosOK := reasoningEffortRank(decision.MaxReasoningEffort)
	if !groupOK {
		return decision.MaxReasoningEffort
	}
	if !qosOK {
		return groupCeiling
	}
	if qosRank < groupRank {
		return decision.MaxReasoningEffort
	}
	return groupCeiling
}
