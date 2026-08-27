package service

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
)

func TestAntigravityOAuthService_GenerateAuthURLUsesConfiguredClientID(t *testing.T) {
	t.Setenv(antigravity.AntigravityOAuthClientIDEnv, "  synthetic-service-client-id  ")
	t.Setenv(antigravity.AntigravityOAuthClientSecretEnv, "  synthetic-service-client-secret  ")

	service := NewAntigravityOAuthService(nil)
	defer service.Stop()

	result, err := service.GenerateAuthURL(context.Background(), nil)
	if err != nil {
		t.Fatalf("GenerateAuthURL() failed: %v", err)
	}
	parsed, err := url.Parse(result.AuthURL)
	if err != nil {
		t.Fatalf("generated auth URL is invalid: %v", err)
	}
	if got := parsed.Query().Get("client_id"); got != "synthetic-service-client-id" {
		t.Fatalf("generated auth URL client_id = %q", got)
	}
}

func TestAntigravityOAuthService_GenerateAuthURLFailsBeforeSessionStorage(t *testing.T) {
	t.Setenv(antigravity.AntigravityOAuthClientIDEnv, "")
	t.Setenv(antigravity.AntigravityOAuthClientSecretEnv, "")

	service := NewAntigravityOAuthService(nil)
	defer service.Stop()

	result, err := service.GenerateAuthURL(context.Background(), nil)
	if err == nil || result != nil {
		t.Fatalf("GenerateAuthURL() should fail without credentials: result=%v err=%v", result, err)
	}
	if !strings.Contains(err.Error(), antigravity.AntigravityOAuthClientIDEnv) ||
		!strings.Contains(err.Error(), antigravity.AntigravityOAuthClientSecretEnv) {
		t.Fatalf("configuration error should name both environment variables: %v", err)
	}
}

func TestResolveDefaultTierID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		loadRaw map[string]any
		want    string
	}{
		{
			name:    "nil loadRaw",
			loadRaw: nil,
			want:    "",
		},
		{
			name: "missing allowedTiers",
			loadRaw: map[string]any{
				"paidTier": map[string]any{"id": "g1-pro-tier"},
			},
			want: "",
		},
		{
			name:    "empty allowedTiers",
			loadRaw: map[string]any{"allowedTiers": []any{}},
			want:    "",
		},
		{
			name: "tier missing id field",
			loadRaw: map[string]any{
				"allowedTiers": []any{
					map[string]any{"isDefault": true},
				},
			},
			want: "",
		},
		{
			name: "allowedTiers but no default",
			loadRaw: map[string]any{
				"allowedTiers": []any{
					map[string]any{"id": "free-tier", "isDefault": false},
					map[string]any{"id": "standard-tier", "isDefault": false},
				},
			},
			want: "",
		},
		{
			name: "default tier found",
			loadRaw: map[string]any{
				"allowedTiers": []any{
					map[string]any{"id": "free-tier", "isDefault": true},
					map[string]any{"id": "standard-tier", "isDefault": false},
				},
			},
			want: "free-tier",
		},
		{
			name: "default tier id with spaces",
			loadRaw: map[string]any{
				"allowedTiers": []any{
					map[string]any{"id": "  standard-tier  ", "isDefault": true},
				},
			},
			want: "standard-tier",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveDefaultTierID(tc.loadRaw)
			if got != tc.want {
				t.Fatalf("resolveDefaultTierID() = %q, want %q", got, tc.want)
			}
		})
	}
}
