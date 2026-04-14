package core

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrValidation           ErrorCode = "ERR_VALIDATION"
	ErrClosed               ErrorCode = "ERR_CLOSED"
	ErrShutdownInProgress   ErrorCode = "ERR_SHUTDOWN_IN_PROGRESS"
	ErrDiskFull             ErrorCode = "ERR_DISK_FULL"
	ErrReadOnlyFS           ErrorCode = "ERR_READ_ONLY_FS"
	ErrIO                   ErrorCode = "ERR_IO"
	ErrCorruption           ErrorCode = "ERR_CORRUPTION"
	ErrCursorInvalid        ErrorCode = "ERR_CURSOR_INVALID"
	ErrCursorCorrupted      ErrorCode = "ERR_CURSOR_CORRUPTED"
	ErrSequenceConflict     ErrorCode = "ERR_SEQUENCE_CONFLICT"
	ErrSegmentFooterInvalid ErrorCode = "ERR_SEGMENT_FOOTER_INVALID"
	ErrWALInvalid           ErrorCode = "ERR_WAL_INVALID"
	ErrRecoveryRequired     ErrorCode = "ERR_RECOVERY_REQUIRED"
	ErrTimeout              ErrorCode = "ERR_TIMEOUT"
)

type StorageError struct {
	Code    ErrorCode
	Op      string
	Path    string
	Message string
	Cause   error
}

func (e *StorageError) Error() string {
	if e == nil {
		return "<nil>"
	}
	switch {
	case e.Op != "" && e.Message != "":
		return fmt.Sprintf("%s %s: %s", e.Code, e.Op, e.Message)
	case e.Message != "":
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	default:
		return string(e.Code)
	}
}

func (e *StorageError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *StorageError) Is(target error) bool {
	var codeErr errorCodeError
	if errors.As(target, &codeErr) {
		return e.Code == codeErr.code
	}
	return false
}

type errorCodeError struct {
	code ErrorCode
}

func (e errorCodeError) Error() string {
	return string(e.code)
}

func ErrCode(code ErrorCode) error {
	return errorCodeError{code: code}
}

func NewError(code ErrorCode, op, path, message string, cause error) error {
	return &StorageError{
		Code:    code,
		Op:      op,
		Path:    path,
		Message: message,
		Cause:   cause,
	}
}
