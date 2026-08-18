//go:build windows

package setup

import "os"

// Windows does not expose Unix uid/mode semantics. The web setup listener is
// disabled there, so these checks have no network-reachable caller.
func validateBootstrapTokenFileMetadata(_ os.FileInfo) error {
	return nil
}

func validateBootstrapTokenDataDirMetadata(_ os.FileInfo, _ string) error {
	return nil
}
