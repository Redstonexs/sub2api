//go:build !windows

package setup

import (
	"fmt"
	"os"
	"syscall"
)

func validateBootstrapTokenFileMetadata(info os.FileInfo) error {
	if info.Mode()&0o077 != 0 {
		return fmt.Errorf("token file has group/other permissions (mode=%s)", info.Mode())
	}
	uid, err := bootstrapTokenOwnerUID(info)
	if err != nil {
		return fmt.Errorf("token file stat info not available: %w", err)
	}
	if uid != os.Geteuid() {
		return fmt.Errorf("token file owner uid=%d does not match effective uid=%d", uid, os.Geteuid())
	}
	return nil
}

func validateBootstrapTokenDataDirMetadata(info os.FileInfo, dir string) error {
	uid, err := bootstrapTokenOwnerUID(info)
	if err != nil {
		return fmt.Errorf("cannot get file info for data directory %q: %w", dir, err)
	}
	if uid != os.Geteuid() {
		return fmt.Errorf("data directory %q owner uid=%d does not match effective uid=%d", dir, uid, os.Geteuid())
	}
	if info.Mode()&0o022 != 0 {
		return fmt.Errorf("data directory %q is group or other writable (mode=%s)", dir, info.Mode())
	}
	return nil
}

func bootstrapTokenOwnerUID(info os.FileInfo) (int, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("stat info not available")
	}
	return int(stat.Uid), nil
}
