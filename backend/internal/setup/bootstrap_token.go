package setup

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
// It checks: regular file (not symlink), platform-specific ownership and
// permission safety, and valid hex encoding of the expected length.
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

	if err := validateBootstrapTokenFileMetadata(lstatInfo); err != nil {
		return err
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
// effective user and not group/other writable where the platform supports
// those checks. This prevents privilege escalation and token tampering.
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

	if err := validateBootstrapTokenDataDirMetadata(info, dir); err != nil {
		return err
	}
	return nil
}

// writeBootstrapTokenTemp writes the token to a uniquely named private temp
// file in the data directory, fsyncs it, and closes it. The returned path is
// fully written and durable before any publication step, and the file is mode
// 0400 so the published final file never exposes a broader permission.
// The caller owns cleanup of the returned temp path.
func writeBootstrapTokenTemp(token string) (string, error) {
	dir := GetDataDir()
	fd, err := os.CreateTemp(dir, BootstrapTokenFile+".tmp.*")
	if err != nil {
		return "", fmt.Errorf("failed to create bootstrap token temp file: %w", err)
	}
	tempPath := fd.Name()

	cleanup := func() {
		_ = fd.Close()
		_ = os.Remove(tempPath)
	}

	// Private: owner-only read. Set explicitly so the published final file is
	// 0400 regardless of CreateTemp's defaults or the process umask.
	if err := fd.Chmod(0400); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to set bootstrap token temp file mode: %w", err)
	}

	if _, err := fd.Write([]byte(token + "\n")); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to write bootstrap token: %w", err)
	}
	// Sync before close so the token is durable on disk before publication.
	if err := fd.Sync(); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to sync bootstrap token temp file: %w", err)
	}
	if err := fd.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("failed to close bootstrap token temp file: %w", err)
	}
	return tempPath, nil
}

// LoadOrCreateBootstrapToken loads an existing token or creates one.
// Only called from runSetupServer (the HTTP setup wizard); never from CLI or AUTO_SETUP.
// A valid existing token is never truncated; an invalid or insecure token file fails
// closed (never silently deleted or overwritten).
//
// Publish invariant: a new token is written, fsynced, and closed in a uniquely
// named private temp file inside the data directory, and only then atomically
// published at the final pathname via a no-replace hard link (os.Link). The
// final pathname therefore never exposes a partial or empty token: at every
// instant it either does not exist or is a complete, durable token. If another
// process published first, os.Link fails with ErrExist and the fully published
// file is loaded and validated; the existing final token is never overwritten
// or removed, and a malformed one fails closed.
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

	// File does not exist. Generate a token and stage it in a temp file that
	// is complete before publication.
	token, err := generateBootstrapToken()
	if err != nil {
		return "", err
	}

	tempPath, err := writeBootstrapTokenTemp(token)
	if err != nil {
		return "", err
	}
	// Always clean up the temp file: after a successful link it is merely an
	// extra hard link to the published inode; after a failure it must not
	// linger in the data directory.
	defer func() {
		_ = os.Remove(tempPath)
	}()

	// Publish with a no-replace hard link. The final pathname appears only
	// once the temp file is fully written, synced, and closed, so a
	// concurrent reader can never observe a partial token there.
	if err := os.Link(tempPath, path); err != nil {
		if os.IsExist(err) {
			// Another process published a fully written token first. Load and
			// validate the published file; never overwrite or remove it.
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return "", fmt.Errorf("failed to read concurrently created bootstrap token: %w", readErr)
			}
			if valErr := validateBootstrapTokenFile(path); valErr != nil {
				// The concurrent publisher wrote an invalid token. Fail closed.
				return "", fmt.Errorf("existing bootstrap token (created concurrently) is invalid: %w", valErr)
			}
			return strings.TrimSpace(string(data)), nil
		}
		return "", fmt.Errorf("failed to publish bootstrap token: %w", err)
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
