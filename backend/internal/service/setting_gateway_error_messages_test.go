package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type mockGatewayErrorMessagesSettingRepo struct {
	SettingRepository
}

func (r *mockGatewayErrorMessagesSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func TestSettingKeyGatewayErrorMessages(t *testing.T) {
	require.Equal(t, "gateway_error_messages", SettingKeyGatewayErrorMessages)
}

func TestBuildSystemSettingsUpdates_GatewayErrorMessages(t *testing.T) {
	cfg := &config.Config{}
	svc := NewSettingService(&mockGatewayErrorMessagesSettingRepo{}, cfg)
	updates, err := svc.buildSystemSettingsUpdates(context.Background(), &SystemSettings{
		GatewayErrorMessages: map[string]string{
			"429": "Too many requests",
			"502": "Upstream unavailable",
		},
	})
	require.NoError(t, err)
	require.Equal(t, `{"429":"Too many requests","502":"Upstream unavailable"}`, updates[SettingKeyGatewayErrorMessages])
}

func TestParseSettings_GatewayErrorMessages(t *testing.T) {
	cfg := &config.Config{}
	svc := NewSettingService(&mockGatewayErrorMessagesSettingRepo{}, cfg)

	settings := svc.parseSettings(map[string]string{
		SettingKeyGatewayErrorMessages: `{"429":"Please retry later","502":"Upstream unavailable"}`,
	})

	require.Equal(t, map[string]string{
		"429": "Please retry later",
		"502": "Upstream unavailable",
	}, settings.GatewayErrorMessages)
}

func TestParseSettings_GatewayErrorMessages_InvalidJSON(t *testing.T) {
	cfg := &config.Config{}
	svc := NewSettingService(&mockGatewayErrorMessagesSettingRepo{}, cfg)

	settings := svc.parseSettings(map[string]string{
		SettingKeyGatewayErrorMessages: `not json`,
	})

	require.Nil(t, settings.GatewayErrorMessages)
}
