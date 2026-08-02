//go:build unit

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayErrorMessage_LiveOverrideTakesPrecedenceOverStatic(t *testing.T) {
	cfg := &Config{Gateway: GatewayConfig{ErrorMessages: map[string]string{
		"503": "static 503",
		"429": "static 429",
	}}}
	require.Equal(t, "static 429", GatewayErrorMessage(cfg, 429, "fallback"), "static config applies before any live override")

	cfg.SetGatewayErrorMessages(map[string]string{"429": "live 429"})

	require.Equal(t, "live 429", GatewayErrorMessage(cfg, 429, "fallback"), "live override wins for codes it covers")
	require.Equal(t, "static 503", GatewayErrorMessage(cfg, 503, "fallback"), "static config still wins for codes the live map does not cover")
	require.Equal(t, "fallback", GatewayErrorMessage(cfg, 500, "fallback"), "unconfigured code still falls back to the default")
}

func TestGatewayErrorMessage_LiveOverrideCloneOnWrite(t *testing.T) {
	cfg := &Config{Gateway: GatewayConfig{ErrorMessages: map[string]string{"503": "static 503"}}}

	input := map[string]string{"429": "live 429"}
	cfg.SetGatewayErrorMessages(input)
	input["429"] = "mutated 429"
	input["502"] = "added later"

	require.Equal(t, "live 429", GatewayErrorMessage(cfg, 429, "fallback"), "mutating the source map after Set must not affect the published snapshot")

	got := cfg.GatewayErrorMessagesLive()
	got["429"] = "mutated 429"
	require.Equal(t, "live 429", GatewayErrorMessage(cfg, 429, "fallback"), "mutating the getter result must not affect the published snapshot")
	require.Equal(t, map[string]string{"429": "live 429"}, cfg.GatewayErrorMessagesLive(), "getter returns a fresh clone each time")
}

func TestGatewayErrorMessage_LiveOverrideClearedByNilAndEmpty(t *testing.T) {
	cfg := &Config{Gateway: GatewayConfig{ErrorMessages: map[string]string{
		"503": "static 503",
		"429": "static 429",
	}}}

	cfg.SetGatewayErrorMessages(map[string]string{"429": "live 429"})
	require.Equal(t, "live 429", GatewayErrorMessage(cfg, 429, "fallback"))

	cfg.SetGatewayErrorMessages(nil)
	require.Equal(t, "static 429", GatewayErrorMessage(cfg, 429, "fallback"), "nil clears the live override and static config resumes")

	cfg.SetGatewayErrorMessages(map[string]string{"429": "live 429"})
	cfg.SetGatewayErrorMessages(map[string]string{})
	require.Equal(t, "static 429", GatewayErrorMessage(cfg, 429, "fallback"), "empty map clears the live override and static config resumes")
}

func TestGatewayErrorMessage_BlankLiveValueFallsBack(t *testing.T) {
	cfg := &Config{Gateway: GatewayConfig{ErrorMessages: map[string]string{"503": "static 503"}}}
	cfg.SetGatewayErrorMessages(map[string]string{"429": "   ", "502": ""})
	require.Equal(t, "fallback", GatewayErrorMessage(cfg, 429, "fallback"), "blank live value falls back")
	require.Equal(t, "fallback", GatewayErrorMessage(cfg, 502, "fallback"), "empty live value falls back")
	require.Equal(t, "static 503", GatewayErrorMessage(cfg, 503, "fallback"))
}

func TestGatewayErrorMessage_LiveGetterNilWhenNoOverride(t *testing.T) {
	require.Nil(t, (*Config)(nil).GatewayErrorMessagesLive(), "nil config has no live override")
	require.Nil(t, (&Config{}).GatewayErrorMessagesLive(), "no override means nil")
	require.Equal(t, "fallback", GatewayErrorMessage(nil, 429, "fallback"), "nil cfg preserves fallback semantics")
}
