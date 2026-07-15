package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDebugEnvBool(t *testing.T) {
	t.Run("empty is false", func(t *testing.T) {
		if parseDebugEnvBool("") {
			t.Fatalf("expected false for empty string")
		}
	})

	t.Run("true-like values", func(t *testing.T) {
		for _, value := range []string{"1", "true", "TRUE", "yes", "on"} {
			t.Run(value, func(t *testing.T) {
				if !parseDebugEnvBool(value) {
					t.Fatalf("expected true for %q", value)
				}
			})
		}
	})

	t.Run("false-like values", func(t *testing.T) {
		for _, value := range []string{"0", "false", "off", "debug"} {
			t.Run(value, func(t *testing.T) {
				if parseDebugEnvBool(value) {
					t.Fatalf("expected false for %q", value)
				}
			})
		}
	})
}

func TestInitDebugGatewayBodyFileCreatesOwnerOnlyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway-debug.log")
	svc := &GatewayService{}
	svc.initDebugGatewayBodyFile(path)

	file := svc.debugGatewayBodyFile.Load()
	if file == nil {
		t.Fatal("debug gateway file was not initialized")
	}
	t.Cleanup(func() { _ = file.Close() })

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat debug gateway file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("debug gateway file mode = %04o, want 0600", info.Mode().Perm())
	}
}
