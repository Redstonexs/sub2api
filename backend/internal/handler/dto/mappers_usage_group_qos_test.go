package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// Legacy rows (no snapshot) expose all four QoS fields as unknown: tier/window
// nil, affected nil, effects empty.
func TestUsageLogFromService_GroupQoSLegacyUnknown(t *testing.T) {
	log := &service.UsageLog{
		ID:        1,
		UserID:    10,
		APIKeyID:  20,
		AccountID: 30,
		RequestID: "req-legacy",
		Model:     "gpt-5",
	}

	userDTO := UsageLogFromService(log)
	adminDTO := UsageLogFromServiceAdmin(log)

	for _, got := range []*UsageLog{userDTO, &adminDTO.UsageLog} {
		require.Nil(t, got.GroupQoSTier, "legacy rows have no tier")
		require.Nil(t, got.GroupQoSWindow, "legacy rows have no window")
		require.Nil(t, got.GroupQoSAffected, "legacy rows are unknown, not unaffected")
		require.Nil(t, got.GroupQoSEffects)
	}
}

// A served request under an active tier with material effects maps to
// affected=true and the named effect strings; the admin DTO inherits the
// user-safe fields.
func TestUsageLogFromService_GroupQoSAffected(t *testing.T) {
	log := &service.UsageLog{
		ID:        1,
		UserID:    10,
		APIKeyID:  20,
		AccountID: 30,
		RequestID: "req-qos",
		Model:     "gpt-5",
		GroupQoSRecord: &service.GroupQoSRecordSnapshot{
			Tier:    2,
			Window:  "weekly",
			Effects: service.GroupQoSEffectModel | service.GroupQoSEffectReasoning,
		},
	}

	userDTO := UsageLogFromService(log)
	require.NotNil(t, userDTO.GroupQoSTier)
	require.Equal(t, 2, *userDTO.GroupQoSTier)
	require.NotNil(t, userDTO.GroupQoSWindow)
	require.Equal(t, "weekly", *userDTO.GroupQoSWindow)
	require.NotNil(t, userDTO.GroupQoSAffected)
	require.True(t, *userDTO.GroupQoSAffected)
	require.Equal(t, []string{"model", "reasoning"}, userDTO.GroupQoSEffects)

	adminDTO := UsageLogFromServiceAdmin(log)
	require.NotNil(t, adminDTO.GroupQoSTier)
	require.Equal(t, 2, *adminDTO.GroupQoSTier)
	require.True(t, *adminDTO.GroupQoSAffected)
	require.Equal(t, []string{"model", "reasoning"}, adminDTO.GroupQoSEffects)
}

// A known request with active QoS but no material effect maps to affected=false
// and no effect names — never an unknown/nil affected.
func TestUsageLogFromService_GroupQoSKnownUnaffected(t *testing.T) {
	log := &service.UsageLog{
		ID:        1,
		UserID:    10,
		APIKeyID:  20,
		AccountID: 30,
		RequestID: "req-qos-mask0",
		Model:     "gpt-5",
		GroupQoSRecord: &service.GroupQoSRecordSnapshot{
			Tier:    1,
			Window:  "daily",
			Effects: 0,
		},
	}

	userDTO := UsageLogFromService(log)
	require.NotNil(t, userDTO.GroupQoSTier)
	require.Equal(t, 1, *userDTO.GroupQoSTier)
	require.Equal(t, "daily", *userDTO.GroupQoSWindow)
	require.NotNil(t, userDTO.GroupQoSAffected)
	require.False(t, *userDTO.GroupQoSAffected, "active tier without effect is known-unaffected")
	require.Nil(t, userDTO.GroupQoSEffects)
}

// The QoS fields are user-safe: neither threshold nor spend is ever exposed.
func TestUsageLogFromService_GroupQoSNeverExposesThresholdOrSpend(t *testing.T) {
	log := &service.UsageLog{
		ID:        1,
		UserID:    10,
		APIKeyID:  20,
		AccountID: 30,
		RequestID: "req-qos-safe",
		Model:     "gpt-5",
		GroupQoSRecord: &service.GroupQoSRecordSnapshot{
			Tier:    3,
			Window:  "monthly",
			Effects: service.GroupQoSEffectRPM,
		},
	}
	userDTO := UsageLogFromService(log)
	payload, err := json.Marshal(userDTO)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "threshold")
	require.NotContains(t, string(payload), "spend")
	require.NotContains(t, string(payload), "usage_usd")
	require.NotContains(t, string(payload), "tier_index")
}

// The API contract serializes the four QoS fields as explicit null for
// legacy/unknown rows (never omitted), matching the documented three-state
// semantics. Both the user and the embedded admin DTO must comply.
func TestUsageLogFromService_GroupQoSLegacySerializesExplicitNull(t *testing.T) {
	log := &service.UsageLog{
		ID:        1,
		UserID:    10,
		APIKeyID:  20,
		AccountID: 30,
		RequestID: "req-legacy-null",
		Model:     "gpt-5",
	}

	var envelope map[string]any
	for _, dto := range []any{UsageLogFromService(log), UsageLogFromServiceAdmin(log)} {
		payload, err := json.Marshal(dto)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(payload, &envelope))

		require.Contains(t, envelope, "group_qos_tier", "field must be present, not omitted")
		require.Nil(t, envelope["group_qos_tier"], "legacy rows serialize explicit null")
		require.Contains(t, envelope, "group_qos_window")
		require.Nil(t, envelope["group_qos_window"])
		require.Contains(t, envelope, "group_qos_affected")
		require.Nil(t, envelope["group_qos_affected"])
		require.Contains(t, envelope, "group_qos_effects")
		require.Nil(t, envelope["group_qos_effects"])
	}

	// An affected row still serializes real values, not null.
	affected := &service.UsageLog{
		ID:        2,
		UserID:    10,
		APIKeyID:  20,
		AccountID: 30,
		RequestID: "req-qos-values",
		Model:     "gpt-5",
		GroupQoSRecord: &service.GroupQoSRecordSnapshot{
			Tier:    2,
			Window:  "weekly",
			Effects: service.GroupQoSEffectModel,
		},
	}
	payload, err := json.Marshal(UsageLogFromService(affected))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(payload, &envelope))
	require.Equal(t, float64(2), envelope["group_qos_tier"])
	require.Equal(t, "weekly", envelope["group_qos_window"])
	require.Equal(t, true, envelope["group_qos_affected"])
	require.Equal(t, []any{"model"}, envelope["group_qos_effects"])
}
