//go:build windows

package lock

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

const (
	lockfileFailImmediately = 0x00000001
	lockfileExclusiveLock   = 0x00000002
	errorSharingViolation   = 32
	errorLockViolation      = 33

	// Windows byte-range locks do not have a native whole-file mode.
	// This package reserves the entire uint64 span from offset zero so
	// every process contends on one explicit file-wide lock range.
	lockRangeLengthLow  = ^uint32(0)
	lockRangeLengthHigh = ^uint32(0)
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

func platformLock(file *os.File, mode Mode) error {
	var overlapped syscall.Overlapped
	flags := uint32(lockfileFailImmediately)
	if mode != ModeObserverShared {
		flags |= lockfileExclusiveLock
	}
	result, _, err := procLockFileEx.Call(
		file.Fd(),
		uintptr(flags),
		0,
		uintptr(lockRangeLengthLow),
		uintptr(lockRangeLengthHigh),
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if result == 0 {
		return err
	}
	return nil
}

func platformUnlock(file *os.File) error {
	var overlapped syscall.Overlapped
	result, _, err := procUnlockFileEx.Call(
		file.Fd(),
		0,
		uintptr(lockRangeLengthLow),
		uintptr(lockRangeLengthHigh),
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if result == 0 {
		return err
	}
	return nil
}

func isLockConflict(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	return errno == errorSharingViolation || errno == errorLockViolation
}
