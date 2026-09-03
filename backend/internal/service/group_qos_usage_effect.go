package service

import (
	"bytes"
	"context"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// GroupQoSEffectMask is a bit field describing which group QoS degradation
// actions materially changed a served request.
//
// It is deliberately separate from ChannelUsageFields.QoSApplied: that boolean
// keeps its existing meaning (the billing model was rerouted, so the requested
// model's price must not be charged). A bit here means the request actually
// changed behavior — not merely that a tier was active.
type GroupQoSEffectMask int16

const (
	// GroupQoSEffectModel marks a request whose requested model was rerouted to
	// a cheaper model by the active QoS tier.
	GroupQoSEffectModel GroupQoSEffectMask = 1 << iota
	// GroupQoSEffectReasoning marks a request whose reasoning effort was reduced
	// beyond the group's standing ceiling by the active QoS tier.
	GroupQoSEffectReasoning
	// GroupQoSEffectRPM marks a request whose RPM budget was tightened below the
	// pre-QoS group-layer limit by the active QoS tier.
	GroupQoSEffectRPM
)

// groupQoSEffectNames is the stable, user-facing name order for the bits.
var groupQoSEffectNames = []struct {
	mask GroupQoSEffectMask
	name string
}{
	{GroupQoSEffectModel, "model"},
	{GroupQoSEffectReasoning, "reasoning"},
	{GroupQoSEffectRPM, "rpm"},
}

// Has reports whether the mask contains effect.
func (m GroupQoSEffectMask) Has(effect GroupQoSEffectMask) bool {
	return m&effect != 0
}

// Names returns the user-facing effect names present in the mask, in a stable
// order (model, reasoning, rpm). An empty mask yields nil.
func (m GroupQoSEffectMask) Names() []string {
	names := make([]string, 0, 3)
	for _, entry := range groupQoSEffectNames {
		if m.Has(entry.mask) {
			names = append(names, entry.name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

// GroupQoSRecordSnapshot is the durable, admission-time QoS snapshot carried by
// a served request's usage record. It preserves the difference between the
// active policy and the request's actual impact: tier/window identify the tier
// that was in effect, Effects records only the material changes it caused.
type GroupQoSRecordSnapshot struct {
	// Tier is the 1-based tier number in effect (matches the
	// X-Sub2API-QoS-Tier response header).
	Tier int
	// Window is the canonical rolling window that tripped the tier
	// (daily/weekly/monthly).
	Window string
	// Effects is 0 for a request that was served under an active tier without
	// any material change.
	Effects GroupQoSEffectMask
}

// Affected reports whether the snapshot carries at least one material effect.
// A nil snapshot (no active QoS) is never affected.
func (s *GroupQoSRecordSnapshot) Affected() bool {
	return s != nil && s.Effects != 0
}

// GroupQoSRecordSnapshotFromDecision seeds the durable snapshot from a frozen
// admission-time decision. A nil decision (no tier reached, QoS disabled, or
// fail-open resolution) yields nil — the usage row then persists all-NULL.
func GroupQoSRecordSnapshotFromDecision(decision *GroupQoSDecision) *GroupQoSRecordSnapshot {
	if decision == nil {
		return nil
	}
	return &GroupQoSRecordSnapshot{
		Tier:    decision.TierIndex + 1,
		Window:  decision.Window,
		Effects: 0,
	}
}

// groupQoSRecordSnapshotContextKey is the request-scoped key for the snapshot
// accumulator.
type groupQoSRecordSnapshotContextKey struct{}

// groupQoSRecordAccumulator is the request-scoped, concurrency-safe holder of
// the QoS record snapshot. It is created once per request (in the QoS
// middleware, when a tier is in effect) and consulted when the usage input
// snapshot is built; the usage worker pool never sees the request context, so
// the snapshot must be frozen into the input before the task is submitted.
//
// tier/window are the immutable admission-time values. turnEffects is the
// effect mask of the current WS turn (or, for plain HTTP requests that never
// call BeginGroupQoSTurn, of the whole request). WS turns reset it per turn so
// one turn's QoS effect cannot leak into a later unaffected turn.
type groupQoSRecordAccumulator struct {
	mu sync.Mutex
	// tier/window are the admission-time snapshot values (1-based tier and the
	// canonical rolling window that tripped it).
	tier   int
	window string
	// groupReasoningCeiling is the group's standing reasoning ceiling at
	// admission; groupReasoningMappings its standing exact mappings. They are
	// needed to tell whether a QoS reasoning cap actually changed the final
	// policy outcome compared with applying the standing policy alone.
	groupReasoningCeiling  string
	groupReasoningMappings []ReasoningEffortMapping
	// turnEffects is the current turn's effect mask. HTTP requests (no turn
	// boundary) accumulate the whole request here.
	turnEffects GroupQoSEffectMask
}

// BindGroupQoSRecordSnapshot binds the request-scoped QoS record accumulator
// seeded from the admission-time snapshot. It is a no-op for a nil snapshot.
func BindGroupQoSRecordSnapshot(ctx context.Context, snap *GroupQoSRecordSnapshot, groupReasoningCeiling string, groupReasoningMappings []ReasoningEffortMapping) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if snap == nil {
		return ctx
	}
	acc := &groupQoSRecordAccumulator{
		tier:                   snap.Tier,
		window:                 snap.Window,
		groupReasoningCeiling:  groupReasoningCeiling,
		groupReasoningMappings: append([]ReasoningEffortMapping(nil), groupReasoningMappings...),
	}
	return context.WithValue(ctx, groupQoSRecordSnapshotContextKey{}, acc)
}

// BeginGroupQoSTurn starts a new WS turn's effect window on the bound
// accumulator. Turn 1 keeps whatever accumulated since admission (e.g. an
// admission-time RPM tightening); later turns start clean so a QoS effect from
// one turn never leaks into a later unaffected turn. No-op without a bound
// accumulator or for plain HTTP requests.
func BeginGroupQoSTurn(ctx context.Context, turn int) {
	acc, _ := ctx.Value(groupQoSRecordSnapshotContextKey{}).(*groupQoSRecordAccumulator)
	if acc == nil {
		return
	}
	acc.mu.Lock()
	if turn > 1 {
		acc.turnEffects = 0
	}
	acc.mu.Unlock()
}

// GroupQoSRecordSnapshotFromContext returns a copy of the request's QoS record
// snapshot, or nil when the request had no active tier. The returned snapshot
// is safe to mutate: it never aliases the shared accumulator.
func GroupQoSRecordSnapshotFromContext(ctx context.Context) *GroupQoSRecordSnapshot {
	acc, _ := ctx.Value(groupQoSRecordSnapshotContextKey{}).(*groupQoSRecordAccumulator)
	if acc == nil {
		return nil
	}
	acc.mu.Lock()
	defer acc.mu.Unlock()
	return &GroupQoSRecordSnapshot{
		Tier:    acc.tier,
		Window:  acc.window,
		Effects: acc.turnEffects,
	}
}

// MarkGroupQoSRecordEffect ORs a material effect into the request's snapshot.
// A request without a bound accumulator (no active tier) is untouched.
func MarkGroupQoSRecordEffect(ctx context.Context, effect GroupQoSEffectMask) {
	acc, _ := ctx.Value(groupQoSRecordSnapshotContextKey{}).(*groupQoSRecordAccumulator)
	if acc == nil {
		return
	}
	acc.mu.Lock()
	acc.turnEffects |= effect
	acc.mu.Unlock()
}

// MarkGroupQoSReasoningEffect records a QoS reasoning reduction when the active
// QoS ceiling changed the final policy outcome for body compared with applying
// the group's standing ceiling and mappings alone. body is the pre-policy
// request body (the exact bytes the combined policy was applied to).
func MarkGroupQoSReasoningEffect(ctx context.Context, body []byte) {
	acc, _ := ctx.Value(groupQoSRecordSnapshotContextKey{}).(*groupQoSRecordAccumulator)
	if acc == nil {
		return
	}
	acc.mu.Lock()
	defer acc.mu.Unlock()
	if !GroupQoSReasoningEffectActual(body, acc.groupReasoningCeiling, acc.groupReasoningMappings, GroupQoSDecisionFromContext(ctx)) {
		return
	}
	acc.turnEffects |= GroupQoSEffectReasoning
}

// ApplyOpenAIReasoningEffortPolicyWithGroupQoS applies the standing group
// policy first, preserving its configured deny/downgrade behavior, and then
// applies the active QoS ceiling as downgrade-only. QoS effects are recorded
// only after that second policy has successfully changed the request.
func ApplyOpenAIReasoningEffortPolicyWithGroupQoS(ctx context.Context, body []byte, maxEffort string, mappings []ReasoningEffortMapping, overLimit string) ([]byte, bool, error) {
	standing, standingChanged, err := ApplyOpenAIReasoningEffortPolicy(body, maxEffort, mappings, overLimit)
	if err != nil {
		return body, false, err
	}
	decision := GroupQoSDecisionFromContext(ctx)
	if decision == nil || strings.TrimSpace(decision.MaxReasoningEffort) == "" {
		return standing, standingChanged, nil
	}
	qosBody, qosChanged, err := ApplyOpenAIReasoningEffortPolicy(
		standing, decision.MaxReasoningEffort, nil, ReasoningEffortOverLimitDowngrade,
	)
	if err != nil {
		return body, false, err
	}
	if qosChanged {
		MarkGroupQoSReasoningEffect(ctx, body)
	}
	return qosBody, standingChanged || qosChanged, nil
}

// ApplyOpenAIReasoningEffortPolicyWithGroupQoSForModel is the model-scoped
// variant used by compatibility bridges. The policy sees the client-requested
// model while the returned body retains its already mapped upstream model.
func ApplyOpenAIReasoningEffortPolicyWithGroupQoSForModel(ctx context.Context, body []byte, requestModel, maxEffort string, mappings []ReasoningEffortMapping, overLimit string) ([]byte, bool, error) {
	requestModel = strings.TrimSpace(requestModel)
	if requestModel == "" {
		return ApplyOpenAIReasoningEffortPolicyWithGroupQoS(ctx, body, maxEffort, mappings, overLimit)
	}

	originalModel := gjson.GetBytes(body, "model")
	policyBody, err := sjson.SetBytes(body, "model", requestModel)
	if err != nil {
		return body, false, err
	}
	updated, changed, err := ApplyOpenAIReasoningEffortPolicyWithGroupQoS(ctx, policyBody, maxEffort, mappings, overLimit)
	if err != nil {
		return body, false, err
	}
	if originalModel.Exists() {
		updated, err = sjson.SetBytes(updated, "model", originalModel.String())
	} else {
		updated, err = sjson.DeleteBytes(updated, "model")
	}
	if err != nil {
		return body, false, err
	}
	return updated, changed, nil
}

// ApplyOpenAIReasoningEffortPolicyFromContextWithGroupQoSForModel applies the
// bound standing policy and the active QoS ceiling using the client model for
// scoped mappings.
func ApplyOpenAIReasoningEffortPolicyFromContextWithGroupQoSForModel(ctx context.Context, body []byte, requestModel string) ([]byte, bool, error) {
	if ctx == nil {
		return body, false, nil
	}
	policy, ok := ctx.Value(openAIReasoningEffortPolicyContextKey{}).(openAIReasoningEffortPolicy)
	if !ok {
		return body, false, nil
	}
	return ApplyOpenAIReasoningEffortPolicyWithGroupQoSForModel(
		ctx, body, requestModel, policy.maxEffort, policy.mappings, policy.overLimit,
	)
}

func applyOpenAIReasoningEffortPolicyWithGroupQoS(body []byte, requestModel string, hooks *OpenAIWSIngressHooks, ctx context.Context) ([]byte, error) {
	if hooks == nil {
		return body, nil
	}
	updated, _, err := ApplyOpenAIReasoningEffortPolicyWithGroupQoSForModel(
		ctx, body, requestModel, hooks.MaxReasoningEffort, hooks.ReasoningEffortMappings, hooks.MaxReasoningEffortOverLimit,
	)
	return updated, err
}

// GroupQoSRPMTightened reports whether a QoS tier's RPM limit is positive and
// strictly stricter than the pre-QoS group-layer limit (override ?? group
// rpm_limit; 0 means unlimited on both sides). A redundant cap — equal to or
// looser than the group-layer value — never counts as an effect.
func GroupQoSRPMTightened(decision *GroupQoSDecision, preQoSLimit int) bool {
	if decision == nil || decision.RPMLimit == nil {
		return false
	}
	qos := *decision.RPMLimit
	if qos <= 0 {
		return false
	}
	return preQoSLimit <= 0 || qos < preQoSLimit
}

// GroupQoSRPMMaterial reports whether a served request's QoS RPM limit was the
// material limiter: the QoS limit is positive, it strictly tightened the
// group-layer limit, and it is not shadowed by a stricter-or-equal user-level
// global cap (user.RPMLimit; 0 = no cap). A request rejected or fail-opened at
// the Redis increment never reaches this predicate.
func GroupQoSRPMMaterial(qosLimit *int, preQoSLimit, userRPMLimit int) bool {
	if qosLimit == nil || *qosLimit <= 0 {
		return false
	}
	if preQoSLimit > 0 && *qosLimit >= preQoSLimit {
		return false
	}
	if userRPMLimit > 0 && userRPMLimit <= *qosLimit {
		return false
	}
	return true
}

// GroupQoSReasoningEffectActual reports whether the QoS ceiling changes the
// final reasoning policy outcome for body compared with applying the standing
// group ceiling and mappings alone. A merely active tier, a redundant cap, or
// a rewrite already produced by the standing policy alone never counts.
func GroupQoSReasoningEffectActual(body []byte, groupCeiling string, groupMappings []ReasoningEffortMapping, decision *GroupQoSDecision) bool {
	if decision == nil || len(body) == 0 {
		return false
	}
	effective := EffectiveMaxReasoningEffort(groupCeiling, decision)
	combined, combinedChanged, combinedErr := ApplyOpenAIReasoningEffortPolicy(body, effective, groupMappings, ReasoningEffortOverLimitDowngrade)
	if combinedErr != nil {
		return false
	}
	if !combinedChanged {
		return false
	}
	standing, _, standingErr := ApplyOpenAIReasoningEffortPolicy(body, groupCeiling, groupMappings, ReasoningEffortOverLimitDowngrade)
	if standingErr != nil {
		return false
	}
	return !bytes.Equal(combined, standing)
}

// GroupQoSRecordFromUsageInput merges the admission-time snapshot carried by the
// usage input with the model effect derived from the existing QoSApplied flag.
// QoSApplied keeps its billing semantics unchanged; it is only reused here as
// the authoritative signal that the requested model was actually rerouted.
//
// A nil snapshot (no active tier / fail-open) stays nil so the persisted
// columns remain NULL.
func GroupQoSRecordFromUsageInput(snap *GroupQoSRecordSnapshot, qosApplied bool) *GroupQoSRecordSnapshot {
	if snap == nil {
		return nil
	}
	merged := *snap
	if qosApplied {
		merged.Effects |= GroupQoSEffectModel
	}
	return &merged
}
