package setup

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	// BootstrapTokenFile is the name of the file holding the setup wizard credential.
	BootstrapTokenFile = ".bootstrap_token"
	// bootstrapTokenBytes is the number of random bytes (32 → 64 hex chars).
	bootstrapTokenBytes = 32
	// bootstrapTokenHexLen is the expected hex-encoded length.
	bootstrapTokenHexLen = 64
)

// BootstrapTokenHeader is the header the setup handler requires on POST mutations.
const BootstrapTokenHeader = "X-Bootstrap-Token"

// GetBootstrapTokenPath returns the full path to the bootstrap token file.
func GetBootstrapTokenPath() string {
	return filepath.Join(GetDataDir(), BootstrapTokenFile)
}

// generateBootstrapToken creates a new cryptographically random hex token.
func generateBootstrapToken() (string, error) {
	b := make([]byte, bootstrapTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate bootstrap token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// validateBootstrapTokenFile defensively validates an existing token file.
// It checks: regular file (not symlink), owner equals effective UID,
// no group/other permissions, and valid hex encoding of the expected length.
func validateBootstrapTokenFile(path string) error {
	// Lstat first: must not be a symlink.
	lstatInfo, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("token file lstat failed: %w", err)
	}
	if lstatInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("token file is a symlink")
	}
	if !lstatInfo.Mode().IsRegular() {
		return fmt.Errorf("token file is not a regular file (mode=%s)", lstatInfo.Mode())
	}

	// Owner-only permissions check (Unix only — Windows cannot enforce this).
	if runtime.GOOS != "windows" {
		if lstatInfo.Mode()&0o077 != 0 {
			return fmt.Errorf("token file has group/other permissions (mode=%s)", lstatInfo.Mode())
		}
		stat, ok := lstatInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("token file stat info not available")
		}
		if int(stat.Uid) != os.Geteuid() {
			return fmt.Errorf("token file owner uid=%d does not match effective uid=%d", stat.Uid, os.Geteuid())
		}
	}

	// Read and validate content.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("token file read failed: %w", err)
	}
	token := strings.TrimSpace(string(data))
	if len(token) != bootstrapTokenHexLen {
		return fmt.Errorf("token has invalid length: got %d, want %d", len(token), bootstrapTokenHexLen)
	}
	if _, err := hex.DecodeString(token); err != nil {
		return fmt.Errorf("token is not valid hex: %w", err)
	}
	return nil
}

// validateDataDir validates that the DATA_DIR is a real directory owned by the
// effective user and not group/other writable (Unix only). This prevents
// privilege escalation and tampering with the bootstrap token.
func validateDataDir() error {
	dir := GetDataDir()
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("cannot stat data directory %q: %w", dir, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("data directory %q must not be a symlink", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("data directory %q is not a directory", dir)
	}

	if runtime.GOOS != "windows" {
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("cannot get file info for data directory %q", dir)
		}
		if int(stat.Uid) != os.Geteuid() {
			return fmt.Errorf("data directory %q owner uid=%d does not match effective uid=%d",
				dir, stat.Uid, os.Geteuid())
		}
		// Reject group/other writable directories.
		if info.Mode()&0o022 != 0 {
			return fmt.Errorf("data directory %q is group or other writable (mode=%s)",
				dir, info.Mode())
		}
	}
	return nil
}

// LoadOrCreateBootstrapToken loads an existing token or creates one exclusively.
// Only called from runSetupServer (the HTTP setup wizard); never from CLI or AUTO_SETUP.
// A valid existing token is never truncated; an invalid or insecure token file fails
// closed (never silently deleted or overwritten). Token creation uses O_CREATE|O_EXCL
// so no other process can replace an existing token.
func LoadOrCreateBootstrapToken() (string, error) {
	path := GetBootstrapTokenPath()

	// Validate the data directory before any file operations.
	if err := validateDataDir(); err != nil {
		return "", fmt.Errorf("bootstrap token data directory invalid: %w", err)
	}

	// Try to load existing token. Use ReadFile as the probe — it succeeds
	// only for a regular file we can read (no symlink, no directory).
	if data, err := os.ReadFile(path); err == nil {
		// Validate the existing token file is well-formed and secure.
		if err := validateBootstrapTokenFile(path); err != nil {
			// Fail closed: never silently delete or overwrite an invalid existing token.
			return "", fmt.Errorf("existing bootstrap token is invalid: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read bootstrap token: %w", err)
	}

	// File does not exist. Create a new token using O_CREATE|O_EXCL so that
	// we never replace a token created by another process between our probe
	// and write.
	token, err := generateBootstrapToken()
	if err != nil {
		return "", err
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0400)
	if err != nil {
		// O_EXCL failed — the file was created by another process between our
		// read probe and open. Try to load it instead of failing.
		if os.IsExist(err) {
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				valErr := validateBootstrapTokenFile(path)
				if valErr == nil {
					return strings.TrimSpace(string(data)), nil
				}
				// The other process wrote an invalid token. Fail closed.
				return "", fmt.Errorf("existing bootstrap token (created concurrently) is invalid: %w", valErr)
			}
			return "", fmt.Errorf("failed to read concurrently created bootstrap token: %w", readErr)
		}
		return "", fmt.Errorf("failed to create bootstrap token exclusively: %w", err)
	}

	if _, err := fd.Write([]byte(token + "\n")); err != nil {
		_ = fd.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("failed to write bootstrap token: %w", err)
	}
	if err := fd.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("failed to close bootstrap token file: %w", err)
	}
	return token, nil
}

// RemoveBootstrapToken retires the token after successful installation.
// It clears the in-memory token first, then removes the file.
// On failure, logs a value-free warning without undoing the installation.
// Safe to call on an already-installed system (no-op if file is gone).
func RemoveBootstrapToken() {
	// Invalidate the in-memory token first, preventing any race window where
	// a concurrent request could still pass the token check.
	SetBootstrapToken("")

	path := GetBootstrapTokenPath()
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return
		}
		logger.L().Warn("failed to remove bootstrap token file; retry on next startup",
			zap.String("path", path))
	}
}

// constantTimeCompare compares two strings in constant time to prevent
// timing side-channel attacks on the bootstrap token.
func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
