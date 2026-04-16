//go:build !windows

package lock

import (
	"errors"
	"os"
	"syscall"
)

func platformLock(file *os.File, mode Mode) error {
	lockMode := syscall.LOCK_SH
	if mode != ModeObserverShared {
		lockMode = syscall.LOCK_EX
	}
	return syscall.Flock(int(file.Fd()), lockMode|syscall.LOCK_NB)
}

func platformUnlock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

func isLockConflict(err error) bool {
	return errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)
}
