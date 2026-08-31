//go:build !windows

package ctx

import "os"

func atomicReplace(stagedPath, destinationPath string) error {
	return os.Rename(stagedPath, destinationPath)
}
