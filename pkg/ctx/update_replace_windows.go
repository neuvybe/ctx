//go:build windows

package ctx

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	moveFileReplaceExisting = 0x00000001
	moveFileWriteThrough    = 0x00000008
)

var procMoveFileExW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func atomicReplace(stagedPath, destinationPath string) error {
	staged, err := syscall.UTF16PtrFromString(stagedPath)
	if err != nil {
		return err
	}
	destination, err := syscall.UTF16PtrFromString(destinationPath)
	if err != nil {
		return err
	}
	result, _, callErr := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(staged)),
		uintptr(unsafe.Pointer(destination)),
		moveFileReplaceExisting|moveFileWriteThrough,
	)
	if result == 0 {
		if callErr == nil || callErr == syscall.Errno(0) {
			return fmt.Errorf("MoveFileExW failed")
		}
		return fmt.Errorf("MoveFileExW: %w", callErr)
	}
	return nil
}
