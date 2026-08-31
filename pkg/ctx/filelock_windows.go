//go:build windows

package ctx

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const lockfileExclusiveLock = 0x00000002

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

func acquireFileLock(path string) (func() error, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	overlapped := &syscall.Overlapped{}
	result, _, callErr := procLockFileEx.Call(
		f.Fd(), lockfileExclusiveLock, 0, 1, 0, uintptr(unsafe.Pointer(overlapped)),
	)
	if result == 0 {
		_ = f.Close()
		return nil, windowsLockError("LockFileEx", callErr)
	}
	return func() error {
		result, _, callErr := procUnlockFileEx.Call(
			f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(overlapped)),
		)
		closeErr := f.Close()
		if result == 0 {
			return windowsLockError("UnlockFileEx", callErr)
		}
		return closeErr
	}, nil
}

func windowsLockError(operation string, err error) error {
	if err == nil || err == syscall.Errno(0) {
		return fmt.Errorf("%s failed", operation)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
