//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !windows

package ctx

import (
	"fmt"
	"os"
)

// Platforms outside the distributed Darwin/Linux targets use an exclusive
// lock file. Normal completion removes it; after an interrupted process the
// error names the stale path so the user can inspect and remove it explicitly.
func acquireFileLock(path string) (func() error, error) {
	exclusivePath := path + ".exclusive"
	f, err := os.OpenFile(exclusivePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", exclusivePath, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(exclusivePath)
		return nil, err
	}
	return func() error { return os.Remove(exclusivePath) }, nil
}
