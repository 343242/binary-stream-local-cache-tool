package cache

import "fastReadFile/internal/core"

type ErrorCode = core.ErrorCode
type StorageError = core.StorageError

const (
	ErrValidation           = core.ErrValidation
	ErrClosed               = core.ErrClosed
	ErrShutdownInProgress   = core.ErrShutdownInProgress
	ErrDiskFull             = core.ErrDiskFull
	ErrReadOnlyFS           = core.ErrReadOnlyFS
	ErrIO                   = core.ErrIO
	ErrCorruption           = core.ErrCorruption
	ErrCursorInvalid        = core.ErrCursorInvalid
	ErrCursorCorrupted      = core.ErrCursorCorrupted
	ErrSequenceConflict     = core.ErrSequenceConflict
	ErrSegmentFooterInvalid = core.ErrSegmentFooterInvalid
	ErrWALInvalid           = core.ErrWALInvalid
	ErrRecoveryRequired     = core.ErrRecoveryRequired
	ErrTimeout              = core.ErrTimeout
)

var ErrCode = core.ErrCode
var NewError = core.NewError
