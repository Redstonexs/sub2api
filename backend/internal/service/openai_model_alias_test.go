package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeKnownOpenAICodexModel_BareGPT56RoutesToSol(t *testing.T) {
	tests := map[string]string{
		"gpt-5.6":            "gpt-5.6-sol",
		"openai/gpt-5.6":     "gpt-5.6-sol",
		"gpt5.6":             "gpt-5.6-sol",
		"gpt-5.6-high":       "gpt-5.6-sol",
		"gpt-5.6-max":        "gpt-5.6-sol",
		"gpt-5.6-2026-07-09": "gpt-5.6-sol",
		"openai/gpt-5.6-max": "gpt-5.6-sol",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, normalizeKnownOpenAICodexModel(input))
		})
	}
}

func TestUsageBillingModelCandidates_BareGPT56IncludesSol(t *testing.T) {
	require.Equal(t,
		[]string{"gpt-5.6", "gpt-5.6-sol"},
		usageBillingModelCandidates("gpt-5.6"),
	)
	require.Equal(t,
		[]string{"openai/gpt-5.6", "gpt-5.6", "gpt-5.6-sol"},
		usageBillingModelCandidates("openai/gpt-5.6"),
	)
}

func TestNormalizeKnownOpenAICodexModel_PassesThroughUnknownVersions(t *testing.T) {
	// 新版本发布当天必须透传给上游判定，而不是被兜底分支折叠成旧模型。
	for _, model := range []string{"gpt-5.7", "gpt-5.7-codex", "gpt-5.9-sol", "gpt-6", "gpt-6.1-codex"} {
		require.Emptyf(t, normalizeKnownOpenAICodexModel(model),
			"未知版本 %s 必须透传（返回空），否则用户拿到的是旧模型的回答", model)
	}
}

func TestNormalizeKnownOpenAICodexModel_KeepsKnownVersionFolding(t *testing.T) {
	require.Equal(t, "gpt-5.6-sol", normalizeKnownOpenAICodexModel("gpt-5.6"))
	require.Equal(t, "gpt-5.5", normalizeKnownOpenAICodexModel("gpt-5.5"))
	require.Equal(t, "gpt-5.4", normalizeKnownOpenAICodexModel("gpt-5.4"))
	require.Equal(t, "gpt-5.3-codex", normalizeKnownOpenAICodexModel("gpt-5.3"))
	// 裸别名不含版本号，保持原有兜底行为
	require.Equal(t, "gpt-5.3-codex", normalizeKnownOpenAICodexModel("codex"))
	// 已知的旧版本仍然折叠到 gpt-5.4，保持既有行为不回归
	require.Equal(t, "gpt-5.4", normalizeKnownOpenAICodexModel("gpt-5.1"))
	require.Equal(t, "gpt-5.4", normalizeKnownOpenAICodexModel("gpt-5"))
}

func TestOpenAICodexModelVersionToken(t *testing.T) {
	require.Equal(t, "5.6", openAICodexModelVersionToken("gpt-5.6-sol"))
	require.Equal(t, "5.3", openAICodexModelVersionToken("gpt-5.3-codex-spark"))
	require.Equal(t, "5", openAICodexModelVersionToken("gpt-5"))
	require.Equal(t, "6", openAICodexModelVersionToken("gpt-6-turbo"))
	require.Equal(t, "", openAICodexModelVersionToken("codex"))
}
