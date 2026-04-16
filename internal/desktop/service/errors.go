package service

import (
	"errors"
	"fmt"
	"os"

	core "fastReadFile/internal/core"
	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/lock"
)

const unavailableValue = "N/A"

type guiError struct {
	viewmodel.GUIError
}

func (e *guiError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func newInvalidRootError(op, path, details string) error {
	return &guiError{
		GUIError: viewmodel.GUIError{
			Code:            "GUI_ERR_INVALID_ROOT",
			Title:           "Not a Cache Workspace",
			Message:         "The selected directory does not contain the expected cache layout.",
			Operation:       op,
			Path:            path,
			Recoverable:     true,
			Details:         details,
			SuggestedAction: "Choose another directory with a cache workspace layout.",
		},
	}
}

func wrapServiceError(op, path string, err error) error {
	if err == nil {
		return nil
	}

	var storageErr *core.StorageError
	switch {
	case errors.Is(err, os.ErrPermission):
		return &guiError{
			GUIError: viewmodel.GUIError{
				Code:            "GUI_ERR_PERMISSION_DENIED",
				Title:           "Permission Denied",
				Message:         "The workspace could not be inspected because access was denied.",
				Operation:       op,
				Path:            path,
				Recoverable:     true,
				Details:         err.Error(),
				SuggestedAction: "Check filesystem permissions for the selected workspace.",
			},
		}
	case errors.Is(err, lock.ErrLockConflict):
		return &guiError{
			GUIError: viewmodel.GUIError{
				Code:            "GUI_ERR_LOCK_CONFLICT",
				Title:           "Workspace Busy",
				Message:         "Another process is already holding an incompatible workspace lock.",
				Operation:       op,
				Path:            path,
				Recoverable:     true,
				Details:         err.Error(),
				SuggestedAction: "Wait for the active owner to release the workspace lock, then retry.",
			},
		}
	case errors.As(err, &storageErr):
		code := "GUI_ERR_INTERNAL"
		title := "Workspace Inspection Failed"
		recoverable := true
		switch storageErr.Code {
		case core.ErrReadOnlyFS:
			code = "GUI_ERR_READ_ONLY"
			title = "Read-Only Workspace"
		case core.ErrCorruption, core.ErrCursorCorrupted, core.ErrSegmentFooterInvalid, core.ErrWALInvalid, core.ErrRecoveryRequired:
			code = "GUI_ERR_PARTIAL_CORRUPTION"
			title = "Workspace Is Partially Corrupted"
		case core.ErrTimeout:
			code = "GUI_ERR_TIMEOUT"
			title = "Workspace Inspection Timed Out"
		}
		return &guiError{
			GUIError: viewmodel.GUIError{
				Code:            code,
				Title:           title,
				Message:         storageErr.Message,
				Operation:       op,
				Path:            firstNonEmpty(storageErr.Path, path),
				Recoverable:     recoverable,
				Details:         err.Error(),
				SuggestedAction: "Retry the operation or inspect the workspace with CLI verification tools.",
			},
		}
	default:
		return &guiError{
			GUIError: viewmodel.GUIError{
				Code:            "GUI_ERR_INTERNAL",
				Title:           "Workspace Inspection Failed",
				Message:         "The desktop service could not inspect the workspace.",
				Operation:       op,
				Path:            path,
				Recoverable:     true,
				Details:         err.Error(),
				SuggestedAction: "Retry the operation and inspect the workspace root for inconsistencies.",
			},
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
