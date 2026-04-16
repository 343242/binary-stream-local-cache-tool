package service

import (
	"fmt"
	"os"
	"path/filepath"

	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/lock"
)

func OpenWorkspace(root string) (viewmodel.WorkspaceState, error) {
	state := viewmodel.WorkspaceState{
		RootPath: root,
		LockMode: unavailableValue,
	}

	valid, reason, err := inspectWorkspaceRoot(root)
	if err != nil {
		return viewmodel.WorkspaceState{}, wrapServiceError("open_workspace", root, err)
	}
	if !valid {
		state.Mode = "InvalidWorkspace"
		state.Health = "invalid"
		state.Reason = reason
		return state, nil
	}

	state.Mode = "HealthyObserver"
	state.LockMode = string(lock.ModeObserverShared)
	state.Health = "ok"
	state.CanRefresh = true
	state.CanRunVerify = true
	state.CanRunCloseCheck = true
	state.Reason = ""
	return state, nil
}

func inspectWorkspaceRoot(root string) (bool, string, error) {
	if root == "" {
		return false, "A workspace path is required.", nil
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "The selected path does not exist.", nil
		}
		return false, "", err
	}
	if !info.IsDir() {
		return false, "The selected path is not a directory.", nil
	}

	required := []string{
		filepath.Join(root, "meta"),
		filepath.Join(root, "segments"),
		filepath.Join(root, "wal"),
	}
	for _, path := range required {
		childInfo, childErr := os.Stat(path)
		if childErr != nil {
			if os.IsNotExist(childErr) {
				return false, "The selected directory does not contain the expected cache layout.", nil
			}
			return false, "", childErr
		}
		if !childInfo.IsDir() {
			return false, fmt.Sprintf("Expected %s to be a directory in the cache layout.", filepath.Base(path)), nil
		}
	}

	return true, "", nil
}
