package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// The effect bits are independent named constants, distinct from the
// QoSApplied billing flag (which is tested separately in channel tests).
func TestGroupQoSEffectMaskNames(t *testing.T) {
	require.Equal(t, []string(nil), GroupQoSEffectMask(0).Names())
	require.Equal(t, []string{"model"}, GroupQoSEffectModel.Names())
	require.Equal(t, []string{"reasoning"}, GroupQoSEffectReasoning.Names())
	require.Equal(t, []string{"rpm"}, GroupQoSEffectRPM.Names())
	require.Equal(t, []string{"model", "reasoning", "rpm"},
		(GroupQoSEffectModel | GroupQoSEffectReasoning | GroupQoSEffectRPM).Names())
	require.True(t, (GroupQoSEffectModel | GroupQoSEffectRPM).Has(GroupQoSEffectModel))
	require.True(t, (GroupQoSEffectModel | GroupQoSEffectRPM).Has(GroupQoSEffectRPM))
	require.False(t, (GroupQoSEffectModel | GroupQoSEffectRPM).Has(GroupQoSEffectReasoning))
}

// RPM is affected only when the QoS limit is positive AND strictly stricter
// than the pre-QoS group-layer limit (0 = unlimited on both sides).
func TestGroupQoSRPMTightened(t *testing.T) {
	qos := func(v int) *GroupQoSDecision { return &GroupQoSDecision{RPMLimit: &v} }

	require.False(t, GroupQoSRPMTightened(nil, 50), "no decision")
	require.False(t, GroupQoSRPMTightened(&GroupQoSDecision{}, 50), "decision without rpm limit")
	require.False(t, GroupQoSRPMTightened(qos(0), 50), "qos rpm 0 never tightens")
	require.False(t, GroupQoSRPMTightened(qos(50), 50), "equal cap is redundant")
	require.False(t, GroupQoSRPMTightened(qos(60), 50), "looser cap is redundant")
	require.True(t, GroupQoSRPMTightened(qos(20), 50), "qos strictly stricter than group layer")
	require.True(t, GroupQoSRPMTightened(qos(20), 0), "qos imposes a limit where none existed")
}

// GroupQoSRPMMaterial additionally requires the tightened QoS limit to actually
// be the binding constraint: a stricter-or-equal user-level global cap shadows
// it (the user cap would limit the request first, so the QoS cap has no
// material limiting effect).
func TestGroupQoSRPMMaterial(t *testing.T) {
	limit := func(v int) *int { return &v }

	require.False(t, GroupQoSRPMMaterial(nil, 50, 0), "no qos limit")
	require.False(t, GroupQoSRPMMaterial(limit(0), 50, 0), "qos rpm 0 never material")
	require.False(t, GroupQoSRPMMaterial(limit(50), 50, 0), "equal to group layer: not tightened")
	require.False(t, GroupQoSRPMMaterial(limit(60), 50, 0), "looser than group layer: not tightened")
	require.True(t, GroupQoSRPMMaterial(limit(20), 50, 0), "strictly tightened, no user cap")
	require.True(t, GroupQoSRPMMaterial(limit(20), 0, 0), "imposes a limit where none existed")
	// User-level global cap interactions.
	require.False(t, GroupQoSRPMMaterial(limit(20), 50, 20), "equal user cap shadows the qos cap")
	require.False(t, GroupQoSRPMMaterial(limit(20), 50, 10), "stricter user cap limits first")
	require.False(t, GroupQoSRPMMaterial(limit(20), 0, 20), "user cap equal to qos cap: qos adds nothing")
	require.True(t, GroupQoSRPMMaterial(limit(20), 0, 100), "user cap looser: qos cap binds")
	require.True(t, GroupQoSRPMMaterial(limit(20), 50, 0), "no user cap: qos cap binds")
}

// GroupQoSReasoningEffectActual compares the final policy outcome: the QoS
// ceiling only counts when it changes the result compared with applying the
// standing group ceiling + mappings alone. "changed" from the QoS-aware
// combined policy alone is never sufficient.
func TestGroupQoSReasoningEffectActual(t *testing.T) {
	highEffortBody := []byte(`{"model":"gpt-5","reasoning":{"effort":"high"}}`)
	lowEffortBody := []byte(`{"model":"gpt-5","reasoning":{"effort":"low"}}`)
	noEffortBody := []byte(`{"model":"gpt-5"}`)

	low := &GroupQoSDecision{MaxReasoningEffort: "low"}
	medium := &GroupQoSDecision{MaxReasoningEffort: "medium"}
	turbo := &GroupQoSDecision{MaxReasoningEffort: "turbo"}

	require.False(t, GroupQoSReasoningEffectActual(highEffortBody, "high", nil, nil), "no decision")
	require.False(t, GroupQoSReasoningEffectActual(highEffortBody, "high", nil, &GroupQoSDecision{}), "tier without reasoning cap")
	require.False(t, GroupQoSReasoningEffectActual(noEffortBody, "high", nil, low), "no effort field to rewrite")

	// QoS ceiling strictly stricter than standing, and the request is actually
	// capped below it -> material.
	require.True(t, GroupQoSReasoningEffectActual(highEffortBody, "high", nil, low),
		"qos low caps a high request while standing high leaves it alone")
	require.True(t, GroupQoSReasoningEffectActual(highEffortBody, "", nil, low),
		"no standing ceiling: any qos cap on a high request is a reduction")

	// The request is already within the QoS ceiling -> combined == standing.
	require.False(t, GroupQoSReasoningEffectActual(lowEffortBody, "high", nil, low),
		"request already at/below qos ceiling: no material change")
	require.False(t, GroupQoSReasoningEffectActual(lowEffortBody, "", nil, low),
		"already below the cap")

	// Standing ceiling alone already caps the request the same way -> redundant.
	require.False(t, GroupQoSReasoningEffectActual(highEffortBody, "medium", nil, medium),
		"equal cap is redundant: combined == standing")
	require.False(t, GroupQoSReasoningEffectActual(highEffortBody, "low", nil, medium),
		"qos cap looser than standing: standing already caps harder")

	// Standing ceiling caps at medium, QoS caps at low: outcomes differ.
	require.True(t, GroupQoSReasoningEffectActual(highEffortBody, "medium", nil, low),
		"qos caps below what the standing ceiling would leave")

	// Unrecognized QoS value never counts.
	require.False(t, GroupQoSReasoningEffectActual(highEffortBody, "", nil, turbo),
		"unrecognized value never counts")

	// Mappings are shared by both policies: a mapping already lowering the
	// effort within the QoS ceiling makes the QoS cap non-material.
	mappings := []ReasoningEffortMapping{{From: "high", To: "low"}}
	require.False(t, GroupQoSReasoningEffectActual(highEffortBody, "", mappings, low),
		"standing mappings already rewrite high->low; qos cap changes nothing")
	// A mapping lowering only to medium, while QoS caps at low: material.
	mappingsMedium := []ReasoningEffortMapping{{From: "high", To: "medium"}}
	require.True(t, GroupQoSReasoningEffectActual(highEffortBody, "", mappingsMedium, low),
		"qos cap tightens beyond the standing mapping result")
	// Mapping with a standing ceiling that caps at the same final value: redundant.
	require.False(t, GroupQoSReasoningEffectActual(highEffortBody, "low", mappingsMedium, low),
		"standing ceiling already caps to low; qos cap is redundant")
}

// A decision seeds tier (1-based, matching the response header) and window with
// an empty effect mask; a nil decision means no active QoS.
func TestGroupQoSRecordSnapshotFromDecision(t *testing.T) {
	require.Nil(t, GroupQoSRecordSnapshotFromDecision(nil))

	snap := GroupQoSRecordSnapshotFromDecision(&GroupQoSDecision{
		TierIndex: 1,
		Window:    "weekly",
	})
	require.NotNil(t, snap)
	require.Equal(t, 2, snap.Tier, "tier is 1-based")
	require.Equal(t, "weekly", snap.Window)
	require.Zero(t, snap.Effects, "a mere active tier has no material effect yet")
	require.False(t, snap.Affected())

	// A hard-block tier never yields a served effect.
	blockSnap := GroupQoSRecordSnapshotFromDecision(&GroupQoSDecision{TierIndex: 0, Block: true})
	require.NotNil(t, blockSnap)
	require.Zero(t, blockSnap.Effects)
}

// The model effect merges the existing QoSApplied billing flag without changing
// its semantics; a nil snapshot (no active tier / fail-open) stays nil.
func TestGroupQoSRecordFromUsageInput(t *testing.T) {
	require.Nil(t, GroupQoSRecordFromUsageInput(nil, true), "fail-open with a stray flag stays NULL")

	snap := &GroupQoSRecordSnapshot{Tier: 1, Window: "daily", Effects: GroupQoSEffectReasoning}

	unaffected := GroupQoSRecordFromUsageInput(snap, false)
	require.NotNil(t, unaffected)
	require.Equal(t, GroupQoSEffectReasoning, unaffected.Effects, "no model reroute, no model bit")

	affected := GroupQoSRecordFromUsageInput(snap, true)
	require.NotNil(t, affected)
	require.Equal(t, GroupQoSEffectReasoning|GroupQoSEffectModel, affected.Effects)
	require.True(t, affected.Affected())

	// The merge must not mutate the input snapshot.
	require.Equal(t, GroupQoSEffectReasoning, snap.Effects)
}

// The request-scoped accumulator is the stable input snapshot source: marks are
// concurrency-safe and reads return defensive copies.
func TestGroupQoSRecordSnapshotContext(t *testing.T) {
	ctx := BindGroupQoSRecordSnapshot(context.Background(),
		&GroupQoSRecordSnapshot{Tier: 1, Window: "daily"}, "high", nil)

	require.Nil(t, GroupQoSRecordSnapshotFromContext(context.Background()), "no accumulator -> nil")

	got := GroupQoSRecordSnapshotFromContext(ctx)
	require.NotNil(t, got)
	require.Equal(t, 1, got.Tier)
	require.Equal(t, "daily", got.Window)
	require.Zero(t, got.Effects)

	// Mutating the returned copy must not leak into the accumulator.
	got.Effects |= GroupQoSEffectModel
	require.Zero(t, GroupQoSRecordSnapshotFromContext(ctx).Effects)

	// Concurrent marks are safe and cumulative.
	var wg sync.WaitGroup
	for _, effect := range []GroupQoSEffectMask{GroupQoSEffectModel, GroupQoSEffectReasoning, GroupQoSEffectRPM} {
		wg.Add(1)
		go func(e GroupQoSEffectMask) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				MarkGroupQoSRecordEffect(ctx, e)
			}
		}(effect)
	}
	wg.Wait()
	require.Equal(t, GroupQoSEffectModel|GroupQoSEffectReasoning|GroupQoSEffectRPM,
		GroupQoSRecordSnapshotFromContext(ctx).Effects)

	require.Nil(t, BindGroupQoSRecordSnapshot(context.Background(), nil, "high", nil).Value(groupQoSRecordSnapshotContextKey{}),
		"nil snapshot must not bind an accumulator")
}

// BeginGroupQoSTurn gives each WS turn its own frozen snapshot/effect mask:
// turn 1 keeps admission-time effects (e.g. an RPM tightening), later turns
// start clean so one turn's QoS effect never leaks into an unaffected turn.
func TestGroupQoSRecordSnapshotPerTurn(t *testing.T) {
	ctx := BindGroupQoSRecordSnapshot(context.Background(),
		&GroupQoSRecordSnapshot{Tier: 2, Window: "weekly"}, "high", nil)

	// Admission-time RPM tightening lands before the first turn begins.
	MarkGroupQoSRecordEffect(ctx, GroupQoSEffectRPM)

	// Turn 1 keeps the admission effects.
	BeginGroupQoSTurn(ctx, 1)
	MarkGroupQoSRecordEffect(ctx, GroupQoSEffectModel)
	turn1 := GroupQoSRecordSnapshotFromContext(ctx)
	require.Equal(t, 2, turn1.Tier)
	require.Equal(t, "weekly", turn1.Window)
	require.Equal(t, GroupQoSEffectRPM|GroupQoSEffectModel, turn1.Effects)

	// Turn 2 starts clean: the previous turn's effects must not leak.
	BeginGroupQoSTurn(ctx, 2)
	turn2Start := GroupQoSRecordSnapshotFromContext(ctx)
	require.Equal(t, 2, turn2Start.Tier, "admission tier/window retained")
	require.Zero(t, turn2Start.Effects, "no leakage from turn 1")
	MarkGroupQoSRecordEffect(ctx, GroupQoSEffectReasoning)
	require.Equal(t, GroupQoSEffectReasoning, GroupQoSRecordSnapshotFromContext(ctx).Effects)

	// Turn 3 unaffected again.
	BeginGroupQoSTurn(ctx, 3)
	require.Zero(t, GroupQoSRecordSnapshotFromContext(ctx).Effects)
}

// Multi-turn regression mirroring the WS flow: a reasoning rewrite on turn 1
// must not leak into turn 2, and an unaffected turn 3 stays clean while an
// actually-rewritten later turn is marked on its own snapshot only.
func TestGroupQoSReasoningEffectIsPerTurn(t *testing.T) {
	highBody := []byte(`{"model":"gpt-5","reasoning":{"effort":"high"}}`)
	ctx := BindGroupQoSRecordSnapshot(context.Background(),
		&GroupQoSRecordSnapshot{Tier: 2, Window: "weekly"}, "high", nil)
	ctx = WithGroupQoSDecision(ctx, &GroupQoSDecision{TierIndex: 1, MaxReasoningEffort: "low"})

	// Turn 1: rewritten -> marked on turn 1's snapshot.
	BeginGroupQoSTurn(ctx, 1)
	MarkGroupQoSReasoningEffect(ctx, highBody)
	require.Equal(t, GroupQoSEffectReasoning, GroupQoSRecordSnapshotFromContext(ctx).Effects)

	// Turn 2: the same rewrite applies to this turn too and is marked on turn
	// 2's own snapshot (turn 1's mark is not carried forward by BeginGroupQoSTurn).
	BeginGroupQoSTurn(ctx, 2)
	MarkGroupQoSReasoningEffect(ctx, highBody)
	require.Equal(t, GroupQoSEffectReasoning, GroupQoSRecordSnapshotFromContext(ctx).Effects,
		"turn 2 genuinely applies the rewrite too (same body)")

	// Turn 3: a body that is NOT rewritten by the policy -> clean snapshot.
	BeginGroupQoSTurn(ctx, 3)
	lowBody := []byte(`{"model":"gpt-5","reasoning":{"effort":"low"}}`)
	MarkGroupQoSReasoningEffect(ctx, lowBody)
	require.Zero(t, GroupQoSRecordSnapshotFromContext(ctx).Effects,
		"unaffected turn stays clean: no leakage from earlier turns")

	// Turn 4: rewritten again -> marked on turn 4's own snapshot only.
	BeginGroupQoSTurn(ctx, 4)
	MarkGroupQoSReasoningEffect(ctx, highBody)
	require.Equal(t, GroupQoSEffectReasoning, GroupQoSRecordSnapshotFromContext(ctx).Effects)
}

// MarkGroupQoSReasoningEffect records a QoS reasoning reduction only when the
// QoS ceiling changes the final policy outcome for the body compared with the
// standing group ceiling + mappings alone.
func TestMarkGroupQoSReasoningEffect(t *testing.T) {
	highBody := []byte(`{"model":"gpt-5","reasoning":{"effort":"high"}}`)

	// QoS strictly stricter than the standing ceiling and the rewrite is real.
	ctx := BindGroupQoSRecordSnapshot(context.Background(),
		&GroupQoSRecordSnapshot{Tier: 2, Window: "weekly"}, "high", nil)
	ctx = WithGroupQoSDecision(ctx, &GroupQoSDecision{TierIndex: 1, MaxReasoningEffort: "low"})
	MarkGroupQoSReasoningEffect(ctx, highBody)
	require.Equal(t, GroupQoSEffectReasoning, GroupQoSRecordSnapshotFromContext(ctx).Effects)

	// The standing ceiling alone already caps the request: redundant, never marked.
	redundantCtx := BindGroupQoSRecordSnapshot(context.Background(),
		&GroupQoSRecordSnapshot{Tier: 1, Window: "daily"}, "low", nil)
	redundantCtx = WithGroupQoSDecision(redundantCtx, &GroupQoSDecision{TierIndex: 0, MaxReasoningEffort: "low"})
	MarkGroupQoSReasoningEffect(redundantCtx, highBody)
	require.Zero(t, GroupQoSRecordSnapshotFromContext(redundantCtx).Effects,
		"an equal cap is not a reduction beyond the standing ceiling")

	// Standing mappings alone already produce the QoS outcome: not material.
	mappedCtx := BindGroupQoSRecordSnapshot(context.Background(),
		&GroupQoSRecordSnapshot{Tier: 1, Window: "daily"}, "", []ReasoningEffortMapping{{From: "high", To: "low"}})
	mappedCtx = WithGroupQoSDecision(mappedCtx, &GroupQoSDecision{TierIndex: 0, MaxReasoningEffort: "low"})
	MarkGroupQoSReasoningEffect(mappedCtx, highBody)
	require.Zero(t, GroupQoSRecordSnapshotFromContext(mappedCtx).Effects,
		"the standing mapping already rewrites high->low; the qos cap changes nothing")

	// No decision bound -> never marked.
	noDecisionCtx := BindGroupQoSRecordSnapshot(context.Background(),
		&GroupQoSRecordSnapshot{Tier: 1, Window: "daily"}, "high", nil)
	MarkGroupQoSReasoningEffect(noDecisionCtx, highBody)
	require.Zero(t, GroupQoSRecordSnapshotFromContext(noDecisionCtx).Effects)
}
