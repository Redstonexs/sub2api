package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestGenerateBootstrapToken(t *testing.T) {
	t.Parallel()

	token1, err := generateBootstrapToken()
	if err != nil {
		t.Fatalf("generateBootstrapToken() error = %v", err)
	}
	if len(token1) != bootstrapTokenHexLen {
		t.Fatalf("token length = %d, want %d", len(token1), bootstrapTokenHexLen)
	}

	// Second call must produce a different token.
	token2, err := generateBootstrapToken()
	if err != nil {
		t.Fatalf("generateBootstrapToken() error = %v", err)
	}
	if token1 == token2 {
		t.Fatal("two generated tokens are identical")
	}
}

func TestConstantTimeCompare(t *testing.T) {
	t.Parallel()

	t.Run("equal strings", func(t *testing.T) {
		if !constantTimeCompare("abc123", "abc123") {
			t.Fatal("constantTimeCompare should return true for equal strings")
		}
	})

	t.Run("different strings", func(t *testing.T) {
		if constantTimeCompare("abc123", "xyz789") {
			t.Fatal("constantTimeCompare should return false for different strings")
		}
	})

	t.Run("different lengths", func(t *testing.T) {
		if constantTimeCompare("short", "longer") {
			t.Fatal("constantTimeCompare should return false for different lengths")
		}
	})

	t.Run("empty strings", func(t *testing.T) {
		if !constantTimeCompare("", "") {
			t.Fatal("constantTimeCompare should return true for empty strings")
		}
	})
}

func TestValidateBootstrapTokenFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific file permission tests")
	}

	t.Run("valid token file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token")
		token := strings.Repeat("ab", 32) // 64 hex chars
		if err := os.WriteFile(path, []byte(token+"\n"), 0400); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		if err := validateBootstrapTokenFile(path); err != nil {
			t.Fatalf("validateBootstrapTokenFile() error = %v", err)
		}
	})

	t.Run("symlink fails", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target")
		link := filepath.Join(dir, "link")
		if err := os.WriteFile(target, []byte(strings.Repeat("ab", 32)+"\n"), 0400); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("Symlink error = %v", err)
		}
		if err := validateBootstrapTokenFile(link); err == nil {
			t.Fatal("validateBootstrapTokenFile should reject symlink")
		}
	})

	t.Run("file with group permissions fails", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token")
		token := strings.Repeat("ab", 32)
		if err := os.WriteFile(path, []byte(token+"\n"), 0440); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		if err := validateBootstrapTokenFile(path); err == nil {
			t.Fatal("validateBootstrapTokenFile should reject file with group perms")
		}
	})

	t.Run("file with other permissions fails", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token")
		token := strings.Repeat("ab", 32)
		if err := os.WriteFile(path, []byte(token+"\n"), 0404); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		if err := validateBootstrapTokenFile(path); err == nil {
			t.Fatal("validateBootstrapTokenFile should reject file with other perms")
		}
	})

	t.Run("wrong owner fails", func(t *testing.T) {
		// Only meaningful when running as non-root (root can chown to any UID).
		if os.Geteuid() == 0 {
			t.Skip("test requires non-root execution")
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "token")
		token := strings.Repeat("ab", 32)
		if err := os.WriteFile(path, []byte(token+"\n"), 0400); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		// Change owner to nobody (65534) — will fail when not root.
		if err := os.Chown(path, 65534, -1); err != nil {
			t.Skipf("cannot change file owner (not running as root): %v", err)
		}
		if err := validateBootstrapTokenFile(path); err == nil {
			t.Fatal("validateBootstrapTokenFile should reject wrong owner")
		}
	})

	t.Run("invalid hex content fails", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token")
		if err := os.WriteFile(path, []byte("not-hex-content\n"), 0400); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		if err := validateBootstrapTokenFile(path); err == nil {
			t.Fatal("validateBootstrapTokenFile should reject non-hex content")
		}
	})

	t.Run("wrong length fails", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token")
		if err := os.WriteFile(path, []byte("abcdef\n"), 0400); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		if err := validateBootstrapTokenFile(path); err == nil {
			t.Fatal("validateBootstrapTokenFile should reject wrong length")
		}
	})

	t.Run("directory fails", func(t *testing.T) {
		dir := t.TempDir()
		if err := validateBootstrapTokenFile(dir); err == nil {
			t.Fatal("validateBootstrapTokenFile should reject a directory")
		}
	})

	t.Run("missing file fails", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nonexistent")
		if err := validateBootstrapTokenFile(path); err == nil {
			t.Fatal("validateBootstrapTokenFile should reject missing file")
		}
	})
}

func TestLoadOrCreateBootstrapToken(t *testing.T) {
	t.Run("creates new token file", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		token, err := LoadOrCreateBootstrapToken()
		if err != nil {
			t.Fatalf("LoadOrCreateBootstrapToken() error = %v", err)
		}
		if len(token) != bootstrapTokenHexLen {
			t.Fatalf("token length = %d, want %d", len(token), bootstrapTokenHexLen)
		}

		// Verify file was created with correct permissions.
		info, err := os.Stat(GetBootstrapTokenPath())
		if err != nil {
			t.Fatalf("Stat error = %v", err)
		}
		if !info.Mode().IsRegular() {
			t.Fatal("token file is not a regular file")
		}
		if runtime.GOOS != "windows" && info.Mode()&0o077 != 0 {
			t.Fatalf("token file has group/other permissions: %s", info.Mode())
		}

		// Verify content.
		data, err := os.ReadFile(GetBootstrapTokenPath())
		if err != nil {
			t.Fatalf("ReadFile error = %v", err)
		}
		got := strings.TrimSpace(string(data))
		if got != token {
			t.Fatalf("file content = %q, want %q", got, token)
		}
	})

	t.Run("loads existing valid token", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		// Create a valid token file manually.
		token := strings.Repeat("cd", 32)
		if err := os.WriteFile(GetBootstrapTokenPath(), []byte(token+"\n"), 0400); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}

		got, err := LoadOrCreateBootstrapToken()
		if err != nil {
			t.Fatalf("LoadOrCreateBootstrapToken() error = %v", err)
		}
		if got != token {
			t.Fatalf("token = %q, want %q", got, token)
		}
	})

	t.Run("fails closed on invalid token file", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		// Create an invalid token file.
		if err := os.WriteFile(GetBootstrapTokenPath(), []byte("invalid\n"), 0400); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}

		// Must fail closed — must NOT delete or overwrite the invalid file.
		_, err := LoadOrCreateBootstrapToken()
		if err == nil {
			t.Fatal("LoadOrCreateBootstrapToken should reject invalid token file")
		}
		if !strings.Contains(err.Error(), "invalid") {
			t.Fatalf("error = %v, want 'invalid'", err)
		}
		// Verify the invalid file was NOT deleted.
		if _, err := os.Stat(GetBootstrapTokenPath()); err != nil {
			t.Fatal("LoadOrCreateBootstrapToken must not delete an invalid token file (fail closed)")
		}
	})

	t.Run("second load returns same token", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		token, err := LoadOrCreateBootstrapToken()
		if err != nil {
			t.Fatalf("LoadOrCreateBootstrapToken() error = %v", err)
		}

		// Load again — must be same token.
		token2, err := LoadOrCreateBootstrapToken()
		if err != nil {
			t.Fatalf("second LoadOrCreateBootstrapToken() error = %v", err)
		}
		if token != token2 {
			t.Fatal("second load returned a different token")
		}
	})

	t.Run("concurrent creation publishes exactly one token", func(t *testing.T) {
		// Publish invariant: a token is fully written, synced, and closed in a
		// unique private temp file before being atomically published at the
		// final pathname via a no-replace hard link. A concurrent reader can
		// therefore never observe a partial or empty final file. Both callers
		// succeed (one creates, the other wins the ErrExist race and loads the
		// already-published file), and the file contains exactly one token.
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		var mu sync.Mutex
		tokens := make([]string, 0, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tok, err := LoadOrCreateBootstrapToken()
				if err != nil {
					t.Logf("concurrent LoadOrCreateBootstrapToken error: %v", err)
					return
				}
				mu.Lock()
				tokens = append(tokens, tok)
				mu.Unlock()
			}()
		}
		wg.Wait()

		if len(tokens) != 2 {
			t.Fatalf("expected 2 successful token loads, got %d", len(tokens))
		}
		// Both tokens must be the same (the file was published once, loaded once).
		if tokens[0] != tokens[1] {
			t.Fatal("concurrent calls returned different tokens — no-replace publication or load-after-race failed")
		}
		// Verify the file exists and contains exactly one token.
		data, err := os.ReadFile(GetBootstrapTokenPath())
		if err != nil {
			t.Fatalf("ReadFile error = %v", err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 line in token file, got %d lines", len(lines))
		}
		if lines[0] != tokens[0] {
			t.Fatalf("file content = %q, want %q", lines[0], tokens[0])
		}
	})
}

func countBootstrapTokenTempFiles(t *testing.T) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(GetDataDir(), BootstrapTokenFile+".tmp.*"))
	if err != nil {
		t.Fatalf("Glob error = %v", err)
	}
	return len(matches)
}

func TestWriteBootstrapTokenTemp(t *testing.T) {
	t.Run("writes mode 0400 temp file with full token", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		token := strings.Repeat("ab", 32) // 64 hex chars
		tempPath, err := writeBootstrapTokenTemp(token)
		if err != nil {
			t.Fatalf("writeBootstrapTokenTemp() error = %v", err)
		}
		t.Cleanup(func() {
			if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
				t.Errorf("Remove temp file: %v", err)
			}
		})

		// The temp file must live in the data directory (same filesystem) so
		// the later hard-link publication is atomic.
		if filepath.Dir(tempPath) != dir {
			t.Fatalf("temp file dir = %q, want %q", filepath.Dir(tempPath), dir)
		}

		// Mode must be 0400 (owner read-only) on platforms that enforce it.
		if runtime.GOOS != "windows" {
			info, err := os.Stat(tempPath)
			if err != nil {
				t.Fatalf("Stat error = %v", err)
			}
			if got := info.Mode().Perm(); got != 0o400 {
				t.Fatalf("temp file permissions = %o, want 0400", got)
			}
		}

		// Content must be the full token plus newline before publication.
		data, err := os.ReadFile(tempPath)
		if err != nil {
			t.Fatalf("ReadFile error = %v", err)
		}
		if string(data) != token+"\n" {
			t.Fatalf("temp file content = %q, want %q", data, token+"\n")
		}
	})

	t.Run("multiple calls create distinct temp files", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		p1, err := writeBootstrapTokenTemp(strings.Repeat("ab", 32))
		if err != nil {
			t.Fatalf("writeBootstrapTokenTemp() error = %v", err)
		}
		t.Cleanup(func() {
			if err := os.Remove(p1); err != nil && !os.IsNotExist(err) {
				t.Errorf("Remove first temp file: %v", err)
			}
		})
		p2, err := writeBootstrapTokenTemp(strings.Repeat("cd", 32))
		if err != nil {
			t.Fatalf("writeBootstrapTokenTemp() error = %v", err)
		}
		t.Cleanup(func() {
			if err := os.Remove(p2); err != nil && !os.IsNotExist(err) {
				t.Errorf("Remove second temp file: %v", err)
			}
		})

		if p1 == p2 {
			t.Fatal("two temp files have the same path")
		}
	})
}

func TestLoadOrCreateBootstrapTokenFailsClosedOnEmptyFile(t *testing.T) {
	// Regression: an existing final token of length 0 (the partial-file
	// symptom the publish invariant eliminates) must fail closed. The final
	// pathname only ever appears fully published now, but a pre-existing
	// malformed file must never be deleted, repaired, or overwritten.
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)

	// Create an empty token file.
	if err := os.WriteFile(GetBootstrapTokenPath(), nil, 0400); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := LoadOrCreateBootstrapToken()
	if err == nil {
		t.Fatal("LoadOrCreateBootstrapToken should reject an empty token file")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("error = %v, want 'invalid'", err)
	}
	// The empty file must still be there — never silently replaced.
	data, err := os.ReadFile(GetBootstrapTokenPath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if len(data) != 0 {
		t.Fatal("LoadOrCreateBootstrapToken must not overwrite an empty token file")
	}
}

func TestLoadOrCreateBootstrapTokenNoTempLeak(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)

	// Success path: the temp file must be cleaned up after publication.
	token, err := LoadOrCreateBootstrapToken()
	if err != nil {
		t.Fatalf("LoadOrCreateBootstrapToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}
	if n := countBootstrapTokenTempFiles(t); n != 0 {
		t.Fatalf("found %d leftover temp files after successful publication", n)
	}

	// Fail-closed path: an invalid existing final token must neither be
	// replaced nor leave temp files behind. Remove the token created above
	// first (the final file is 0400, so it cannot be overwritten in place).
	if err := os.Remove(GetBootstrapTokenPath()); err != nil {
		t.Fatalf("Remove error = %v", err)
	}
	if err := os.WriteFile(GetBootstrapTokenPath(), []byte("invalid\n"), 0400); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if _, err := LoadOrCreateBootstrapToken(); err == nil {
		t.Fatal("LoadOrCreateBootstrapToken should reject invalid token file")
	}
	if n := countBootstrapTokenTempFiles(t); n != 0 {
		t.Fatalf("found %d leftover temp files after fail-closed path", n)
	}
	if _, err := os.Stat(GetBootstrapTokenPath()); err != nil {
		t.Fatal("fail-closed path must keep the invalid final token")
	}
}

func TestLoadOrCreateBootstrapTokenConcurrentStress(t *testing.T) {
	// Hammer the publish path with concurrent callers to verify the publish
	// invariant under contention: every caller observes the same token, the
	// final file always contains exactly one well-formed token, and no temp
	// files leak from the ErrExist load-after-race path.
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)

	const goroutines = 16
	var mu sync.Mutex
	tokens := make([]string, 0, goroutines)
	for round := 0; round < 20; round++ {
		tokens = tokens[:0]
		var wg sync.WaitGroup
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tok, err := LoadOrCreateBootstrapToken()
				if err != nil {
					t.Errorf("concurrent LoadOrCreateBootstrapToken error: %v", err)
					return
				}
				mu.Lock()
				tokens = append(tokens, tok)
				mu.Unlock()
			}()
		}
		wg.Wait()

		if len(tokens) != goroutines {
			t.Fatalf("round %d: expected %d successes, got %d", round, goroutines, len(tokens))
		}
		for _, tok := range tokens[1:] {
			if tok != tokens[0] {
				t.Fatalf("round %d: concurrent callers observed different tokens", round)
			}
		}

		// The final file must always hold exactly one well-formed token.
		if err := validateBootstrapTokenFile(GetBootstrapTokenPath()); err != nil {
			t.Fatalf("round %d: published token file invalid: %v", round, err)
		}
	}

	// No temp files may leak from any publication or ErrExist path.
	if n := countBootstrapTokenTempFiles(t); n != 0 {
		t.Fatalf("found %d leftover temp files after concurrent stress", n)
	}
}

func TestRemoveBootstrapToken(t *testing.T) {
	t.Run("removes existing token file", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		// Create token.
		token, err := LoadOrCreateBootstrapToken()
		if err != nil {
			t.Fatalf("LoadOrCreateBootstrapToken() error = %v", err)
		}
		if token == "" {
			t.Fatal("token should not be empty")
		}

		// Remove it.
		RemoveBootstrapToken()

		// File should no longer exist.
		if _, err := os.Stat(GetBootstrapTokenPath()); !os.IsNotExist(err) {
			t.Fatal("token file should have been removed")
		}
	})

	t.Run("no error on missing file", func(t *testing.T) {
		// Should not panic or error.
		RemoveBootstrapToken()
	})

	t.Run("clears in-memory token", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		// Create token and set it.
		token, err := LoadOrCreateBootstrapToken()
		if err != nil {
			t.Fatalf("LoadOrCreateBootstrapToken() error = %v", err)
		}
		SetBootstrapToken(token)

		// Remove the token.
		RemoveBootstrapToken()

		// In-memory token must be cleared.
		if currentBootstrapToken() != "" {
			t.Fatal("RemoveBootstrapToken did not clear the in-memory token")
		}
	})
}

func TestLoadOrCreateBootstrapTokenDefensiveValidation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific ownership tests")
	}

	t.Run("file not owned by euid", func(t *testing.T) {
		// Validate that a token file with the right content but wrong
		// ownership combination is rejected. Create a file, change its
		// owner if possible, then call LoadOrCreateBootstrapToken.
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)

		// Create a valid content file with wrong owner setup.
		path := GetBootstrapTokenPath()
		if err := os.WriteFile(path, []byte(strings.Repeat("ef", 32)+"\n"), 0400); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}

		// If we can change the owner, verify it's caught.
		if err := os.Chown(path, 65534, -1); err == nil {
			_, err := LoadOrCreateBootstrapToken()
			if err == nil {
				t.Fatal("LoadOrCreateBootstrapToken should reject wrong-owner file")
			}
		} else {
			t.Logf("skipping deep ownership check (not root): %v", err)
		}
	})
}

func TestBootstrapTokenOwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific permission test")
	}

	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)

	// The file created by LoadOrCreateBootstrapToken must be 0400.
	token, err := LoadOrCreateBootstrapToken()
	if err != nil {
		t.Fatalf("LoadOrCreateBootstrapToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	info, err := os.Stat(GetBootstrapTokenPath())
	if err != nil {
		t.Fatalf("Stat error = %v", err)
	}

	// Check 0400 (owner read-only).
	if got := info.Mode().Perm(); got != 0o400 {
		t.Fatalf("file permissions = %o, want 0400", got)
	}
}

func TestValidateDataDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific ownership/permission tests")
	}

	t.Run("valid directory passes", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("DATA_DIR", dir)
		if err := validateDataDir(); err != nil {
			t.Fatalf("validateDataDir() error = %v", err)
		}
	})

	t.Run("missing directory fails", func(t *testing.T) {
		t.Setenv("DATA_DIR", "/nonexistent/path/that/does/not/exist")
		if err := validateDataDir(); err == nil {
			t.Fatal("validateDataDir should reject missing directory")
		}
	})

	t.Run("file instead of directory fails", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "not_a_dir")
		if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		t.Setenv("DATA_DIR", filePath)
		if err := validateDataDir(); err == nil {
			t.Fatal("validateDataDir should reject a file path")
		}
	})

	t.Run("symlinked directory fails", func(t *testing.T) {
		dir := t.TempDir()
		link := filepath.Join(t.TempDir(), "data-link")
		if err := os.Symlink(dir, link); err != nil {
			t.Fatalf("Symlink error = %v", err)
		}
		t.Setenv("DATA_DIR", link)
		if err := validateDataDir(); err == nil {
			t.Fatal("validateDataDir should reject a symlinked directory")
		}
	})

	t.Run("group writable directory fails", func(t *testing.T) {
		dir := t.TempDir()
		// Make the directory group-writable.
		if err := os.Chmod(dir, 0770); err != nil {
			t.Fatalf("Chmod error = %v", err)
		}
		t.Setenv("DATA_DIR", dir)
		if err := validateDataDir(); err == nil {
			t.Fatal("validateDataDir should reject group-writable directory")
		}
		// Restore for cleanup.
		_ = os.Chmod(dir, 0700)
	})

	t.Run("other writable directory fails", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0707); err != nil {
			t.Fatalf("Chmod error = %v", err)
		}
		t.Setenv("DATA_DIR", dir)
		if err := validateDataDir(); err == nil {
			t.Fatal("validateDataDir should reject other-writable directory")
		}
		_ = os.Chmod(dir, 0700)
	})
}

func TestLoadOrCreateBootstrapTokenRejectsInsecureDataDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific permission tests")
	}

	t.Run("group writable data dir fails", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0770); err != nil {
			t.Fatalf("Chmod error = %v", err)
		}
		t.Setenv("DATA_DIR", dir)

		_, err := LoadOrCreateBootstrapToken()
		if err == nil {
			t.Fatal("LoadOrCreateBootstrapToken should reject group-writable data dir")
		}
		if !strings.Contains(err.Error(), "group or other writable") {
			t.Fatalf("error = %v, want 'group or other writable'", err)
		}
		_ = os.Chmod(dir, 0700)
	})

	t.Run("nonexistent data dir fails", func(t *testing.T) {
		t.Setenv("DATA_DIR", "/tmp/nonexistent_bootstrap_test_dir")
		_, err := LoadOrCreateBootstrapToken()
		if err == nil {
			t.Fatal("LoadOrCreateBootstrapToken should reject nonexistent data dir")
		}
	})
}

func TestRemoveBootstrapTokenOnAlreadyInstalledStartup(t *testing.T) {
	// Simulate: system is already installed, but a leftover token file exists.
	// RemoveBootstrapToken must clean it up without errors.
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)

	// Create a token file as if from a previous setup.
	token := strings.Repeat("ab", 32)
	if err := os.WriteFile(GetBootstrapTokenPath(), []byte(token+"\n"), 0400); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	SetBootstrapToken(token)

	// Simulate post-install cleanup.
	RemoveBootstrapToken()

	// File must be gone.
	if _, err := os.Stat(GetBootstrapTokenPath()); !os.IsNotExist(err) {
		t.Fatal("RemoveBootstrapToken should remove leftover token file")
	}

	// In-memory token must be cleared.
	if currentBootstrapToken() != "" {
		t.Fatal("RemoveBootstrapToken must clear in-memory token on already-installed startup")
	}

	// Second call must be a no-op.
	RemoveBootstrapToken()
}
