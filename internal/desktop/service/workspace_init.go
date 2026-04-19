package service

import (
	"fmt"
	"os"
	"path/filepath"
)

type WorkspaceInitCode string

const (
	WorkspaceInitInvalidRoot   WorkspaceInitCode = "invalid-root"
	WorkspaceInitPartialLayout WorkspaceInitCode = "partial-layout"
)

type WorkspaceInitError struct {
	Code WorkspaceInitCode
	Path string
}

func (e *WorkspaceInitError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Path)
}

func InitializeWorkspace(root string) error {
	if root == "" {
		return &WorkspaceInitError{
			Code: WorkspaceInitInvalidRoot,
			Path: root,
		}
	}

	layoutState, err := detectWorkspaceLayout(root)
	if err != nil {
		return wrapServiceError("initialize_workspace", root, err)
	}
	if layoutState == workspaceLayoutPartial {
		return &WorkspaceInitError{
			Code: WorkspaceInitPartialLayout,
			Path: root,
		}
	}
	if layoutState == workspaceLayoutValid {
		return nil
	}

	for _, rel := range workspaceLayoutDirectories {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			return wrapServiceError("initialize_workspace", root, err)
		}
	}
	return nil
}

var workspaceLayoutDirectories = []string{
	"meta",
	"meta/replay",
	"segments",
	"wal",
	"tools",
	"tools/reports",
}

type workspaceLayoutState uint8

const (
	workspaceLayoutMissing workspaceLayoutState = iota
	workspaceLayoutEmpty
	workspaceLayoutValid
	workspaceLayoutPartial
)

func detectWorkspaceLayout(root string) (workspaceLayoutState, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return workspaceLayoutMissing, nil
		}
		return workspaceLayoutMissing, err
	}
	if !info.IsDir() {
		return workspaceLayoutPartial, fmt.Errorf("workspace root is not a directory: %s", root)
	}

	entryNames, err := existingEntries(root)
	if err != nil {
		return workspaceLayoutMissing, err
	}

	foundRequired := false
	missingRequired := false
	for _, rel := range workspaceLayoutDirectories {
		path := filepath.Join(root, rel)
		childInfo, childErr := os.Stat(path)
		if childErr != nil {
			if os.IsNotExist(childErr) {
				missingRequired = true
				continue
			}
			return workspaceLayoutMissing, childErr
		}
		if !childInfo.IsDir() {
			return workspaceLayoutPartial, nil
		}
		foundRequired = true
	}

	switch {
	case foundRequired && !missingRequired:
		return workspaceLayoutValid, nil
	case foundRequired && missingRequired:
		return workspaceLayoutPartial, nil
	case len(entryNames) == 0:
		return workspaceLayoutEmpty, nil
	default:
		return workspaceLayoutPartial, nil
	}
}

func existingEntries(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}
