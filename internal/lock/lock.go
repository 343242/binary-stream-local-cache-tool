package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrLockConflict = errors.New("workspace lock conflict")

type LockConflictError struct {
	Path string
	Mode Mode
	Err  error
}

func (e *LockConflictError) Error() string {
	if e == nil {
		return ErrLockConflict.Error()
	}
	return fmt.Sprintf("workspace lock conflict for %s on %s: %v", e.Mode, e.Path, e.Err)
}

func (e *LockConflictError) Unwrap() []error {
	if e == nil || e.Err == nil {
		return []error{ErrLockConflict}
	}
	return []error{ErrLockConflict, e.Err}
}

type Handle struct {
	file *os.File
	path string
	mode Mode
}

func Acquire(root string, mode Mode, meta Metadata) (*Handle, error) {
	lockPath := filepath.Join(root, "meta", "workspace.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	file, created, err := openLockCarrier(lockPath)
	if err != nil {
		return nil, err
	}

	if err := platformLock(file, mode); err != nil {
		_ = file.Close()
		if isLockConflict(err) {
			return nil, &LockConflictError{
				Path: lockPath,
				Mode: mode,
				Err:  err,
			}
		}
		return nil, fmt.Errorf("acquire workspace lock: %w", err)
	}

	now := time.Now().UnixMilli()
	meta.Mode = mode
	if meta.PID == 0 {
		meta.PID = os.Getpid()
	}
	if meta.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			_ = platformUnlock(file)
			_ = file.Close()
			return nil, fmt.Errorf("resolve hostname: %w", err)
		}
		meta.Hostname = hostname
	}
	if meta.StartedAt == 0 {
		meta.StartedAt = now
	}
	if meta.UpdatedAt == 0 {
		meta.UpdatedAt = now
	}

	if shouldPersistMetadata(mode, created) {
		if err := writeMetadata(file, meta); err != nil {
			_ = platformUnlock(file)
			_ = file.Close()
			return nil, fmt.Errorf("write workspace metadata: %w", err)
		}
	}

	return &Handle{
		file: file,
		path: lockPath,
		mode: mode,
	}, nil
}

func (h *Handle) Close() error {
	if h == nil || h.file == nil {
		return nil
	}

	unlockErr := platformUnlock(h.file)
	closeErr := h.file.Close()
	h.file = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

func openLockCarrier(path string) (*os.File, bool, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if err == nil {
		return file, true, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, false, fmt.Errorf("open workspace lock: %w", err)
	}

	file, err = os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return nil, false, fmt.Errorf("open workspace lock: %w", err)
	}
	return file, false, nil
}

func shouldPersistMetadata(mode Mode, created bool) bool {
	if mode == ModeObserverShared {
		return created
	}
	return true
}
