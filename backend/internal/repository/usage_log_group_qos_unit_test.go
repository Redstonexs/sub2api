//go:build unit

package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// newQoSUsageLog builds a UsageLog carrying a known QoS snapshot.
func newQoSUsageLog(record *service.GroupQoSRecordSnapshot) *service.UsageLog {
	return &service.UsageLog{
		UserID:         1,
		APIKeyID:       2,
		AccountID:      3,
		RequestID:      "req-group-qos",
		Model:          "gpt-5",
		InputTokens:    10,
		OutputTokens:   5,
		TotalCost:      1.0,
		ActualCost:     1.0,
		GroupQoSRecord: record,
		CreatedAt:      time.Now().UTC(),
	}
}

// TestPrepareUsageLogInsert_GroupQoSSnapshotArgWiring pins the three group QoS
// columns to the arg slice / arg-type table so every INSERT column list stays in
// sync. They sit between session_id and created_at.
func TestPrepareUsageLogInsert_GroupQoSSnapshotArgWiring(t *testing.T) {
	prepared := prepareUsageLogInsert(newQoSUsageLog(&service.GroupQoSRecordSnapshot{
		Tier:    2,
		Window:  "weekly",
		Effects: service.GroupQoSEffectModel | service.GroupQoSEffectRPM,
	}))

	require.Len(t, prepared.args, len(usageLogInsertArgTypes))

	tierArg := prepared.args[len(prepared.args)-4]
	require.Equal(t, int16(2), tierArg, "group_qos_tier arg should be the int16 tier")

	windowArg := prepared.args[len(prepared.args)-3]
	ws, ok := windowArg.(*string)
	require.True(t, ok, "group_qos_window arg should be *string, got %T", windowArg)
	require.NotNil(t, ws)
	require.Equal(t, "weekly", *ws)

	maskArg := prepared.args[len(prepared.args)-2]
	require.Equal(t, int16(service.GroupQoSEffectModel|service.GroupQoSEffectRPM), maskArg,
		"group_qos_effect_mask arg should carry the effect bits")

	require.Equal(t, "smallint", usageLogInsertArgTypes[len(usageLogInsertArgTypes)-4])
	require.Equal(t, "text", usageLogInsertArgTypes[len(usageLogInsertArgTypes)-3])
	require.Equal(t, "smallint", usageLogInsertArgTypes[len(usageLogInsertArgTypes)-2])
}

// TestPrepareUsageLogInsert_GroupQoSAllNullWithoutSnapshot proves that a request
// without an active QoS tier persists all three columns as SQL NULL (never 0 or
// an empty string).
func TestPrepareUsageLogInsert_GroupQoSAllNullWithoutSnapshot(t *testing.T) {
	prepared := prepareUsageLogInsert(newQoSUsageLog(nil))

	require.Nil(t, prepared.args[len(prepared.args)-4], "group_qos_tier must be NULL")
	require.Nil(t, prepared.args[len(prepared.args)-3], "group_qos_window must be NULL")
	require.Nil(t, prepared.args[len(prepared.args)-2], "group_qos_effect_mask must be NULL")
}

// TestScanUsageLog_GroupQoSSnapshotRoundTrip proves the SELECT side reconstructs
// the snapshot (tier/window/effects) from the three columns.
func TestScanUsageLog_GroupQoSSnapshotRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	log, err := scanUsageLog(usageLogScannerStub{values: []any{
		int64(1),  // id
		int64(10), // user_id
		int64(20), // api_key_id
		int64(30), // account_id
		sql.NullString{Valid: true, String: "req-qos-scan"},
		"gpt-5",          // model
		sql.NullString{}, // requested_model
		sql.NullString{}, // upstream_model
		sql.NullString{}, // upstream_response_model
		sql.NullBool{},   // upstream_model_mismatch
		sql.NullInt64{},  // group_id
		sql.NullInt64{},  // subscription_id
		1, 2, 3, 4, 5, 6, // tokens
		0, 0.0, // image_output_tokens, image_output_cost
		0, 0.0, // image_input_tokens, image_input_cost
		0.1, 0.2, 0.3, 0.4, 1.0, 0.9, // costs
		1.0,               // rate_multiplier
		sql.NullFloat64{}, // account_rate_multiplier
		int16(service.BillingTypeBalance),
		int16(service.RequestTypeSync),
		false, // stream
		false, // openai_ws_mode
		sql.NullInt64{},
		sql.NullInt64{},
		sql.NullString{},
		sql.NullString{},
		0,
		sql.NullString{},
		sql.NullString{}, // image_input_size
		sql.NullString{}, // image_output_size
		sql.NullString{}, // image_size_source
		sql.NullString{}, // image_size_breakdown
		0,                // video_count
		sql.NullString{}, // video_resolution
		sql.NullInt64{},  // video_duration_seconds
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		false,
		false,
		sql.NullInt64{},                      // channel_id
		sql.NullString{},                     // model_mapping_chain
		sql.NullString{},                     // billing_tier
		sql.NullString{},                     // billing_mode
		sql.NullFloat64{},                    // account_stats_cost
		sql.NullString{},                     // session_id
		sql.NullInt64{Valid: true, Int64: 3}, // group_qos_tier
		sql.NullString{Valid: true, String: "monthly"},                                                          // group_qos_window
		sql.NullInt64{Valid: true, Int64: int64(service.GroupQoSEffectModel | service.GroupQoSEffectReasoning)}, // group_qos_effect_mask
		now,
	}})
	require.NoError(t, err)
	require.NotNil(t, log.GroupQoSRecord)
	require.Equal(t, 3, log.GroupQoSRecord.Tier)
	require.Equal(t, "monthly", log.GroupQoSRecord.Window)
	require.Equal(t, service.GroupQoSEffectModel|service.GroupQoSEffectReasoning, log.GroupQoSRecord.Effects)
	require.True(t, log.GroupQoSRecord.Affected())
}

// TestScanUsageLog_GroupQoSZeroMaskSurvives proves mask=0 (active tier, no
// material effect) survives the round trip as a known-unaffected snapshot.
func TestScanUsageLog_GroupQoSZeroMaskSurvives(t *testing.T) {
	now := time.Now().UTC()
	log, err := scanUsageLog(usageLogScannerStub{values: []any{
		int64(2),  // id
		int64(11), // user_id
		int64(21), // api_key_id
		int64(31), // account_id
		sql.NullString{Valid: true, String: "req-qos-zero"},
		"gpt-5",
		sql.NullString{}, // requested_model
		sql.NullString{}, // upstream_model
		sql.NullString{}, // upstream_response_model
		sql.NullBool{},   // upstream_model_mismatch
		sql.NullInt64{},  // group_id
		sql.NullInt64{},  // subscription_id
		1, 2, 3, 4, 5, 6,
		0, 0.0, // image_output_tokens, image_output_cost
		0, 0.0, // image_input_tokens, image_input_cost
		0.1, 0.2, 0.3, 0.4, 1.0, 0.9,
		1.0,
		sql.NullFloat64{},
		int16(service.BillingTypeBalance),
		int16(service.RequestTypeSync),
		false, false,
		sql.NullInt64{},
		sql.NullInt64{},
		sql.NullString{},
		sql.NullString{},
		0,
		sql.NullString{},
		sql.NullString{}, // image_input_size
		sql.NullString{}, // image_output_size
		sql.NullString{}, // image_size_source
		sql.NullString{}, // image_size_breakdown
		0,                // video_count
		sql.NullString{}, // video_resolution
		sql.NullInt64{},  // video_duration_seconds
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		false,
		false,
		sql.NullInt64{},                      // channel_id
		sql.NullString{},                     // model_mapping_chain
		sql.NullString{},                     // billing_tier
		sql.NullString{},                     // billing_mode
		sql.NullFloat64{},                    // account_stats_cost
		sql.NullString{},                     // session_id
		sql.NullInt64{Valid: true, Int64: 1}, // group_qos_tier
		sql.NullString{Valid: true, String: "daily"}, // group_qos_window
		sql.NullInt64{Valid: true, Int64: 0},         // group_qos_effect_mask
		now,
	}})
	require.NoError(t, err)
	require.NotNil(t, log.GroupQoSRecord)
	require.Equal(t, 1, log.GroupQoSRecord.Tier)
	require.Equal(t, "daily", log.GroupQoSRecord.Window)
	require.Zero(t, log.GroupQoSRecord.Effects)
	require.False(t, log.GroupQoSRecord.Affected())
}

// TestScanUsageLog_GroupQoSAllNullIsLegacy proves legacy rows (all three columns
// NULL) yield a nil snapshot, keeping the tri-state DTO semantics.
func TestScanUsageLog_GroupQoSAllNullIsLegacy(t *testing.T) {
	now := time.Now().UTC()
	log, err := scanUsageLog(usageLogScannerStub{values: []any{
		int64(3),
		int64(12),
		int64(22),
		int64(32),
		sql.NullString{Valid: true, String: "req-qos-legacy"},
		"gpt-5",
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		sql.NullBool{},
		sql.NullInt64{},
		sql.NullInt64{},
		1, 2, 3, 4, 5, 6,
		0, 0.0,
		0, 0.0,
		0.1, 0.2, 0.3, 0.4, 1.0, 0.9,
		1.0,
		sql.NullFloat64{},
		int16(service.BillingTypeBalance),
		int16(service.RequestTypeSync),
		false, false,
		sql.NullInt64{},
		sql.NullInt64{},
		sql.NullString{},
		sql.NullString{},
		0,
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		0,
		sql.NullString{},
		sql.NullInt64{},
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		false,
		false,
		sql.NullInt64{},
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		sql.NullFloat64{},
		sql.NullString{},
		sql.NullInt64{},  // group_qos_tier NULL
		sql.NullString{}, // group_qos_window NULL
		sql.NullInt64{},  // group_qos_effect_mask NULL
		now,
	}})
	require.NoError(t, err)
	require.Nil(t, log.GroupQoSRecord, "all-NULL QoS columns must map to a nil snapshot")
}
