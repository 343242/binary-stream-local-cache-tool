package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const recentWorkspaceLimit = 5

func recentWorkspacesPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "binary-stream-local-cache-tool", "recent-workspaces.json"), nil
}

func loadRecentWorkspaces() ([]string, error) {
	path, err := recentWorkspacesPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var workspaces []string
	if err := json.Unmarshal(data, &workspaces); err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(workspaces))
	for _, workspace := range workspaces {
		info, err := os.Stat(workspace)
		if err != nil || !info.IsDir() {
			continue
		}
		filtered = append(filtered, workspace)
		if len(filtered) == recentWorkspaceLimit {
			break
		}
	}
	if err := saveRecentWorkspaces(filtered); err != nil {
		return nil, err
	}
	return filtered, nil
}

func saveRecentWorkspaces(workspaces []string) error {
	path, err := recentWorkspacesPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(workspaces) > recentWorkspaceLimit {
		workspaces = workspaces[:recentWorkspaceLimit]
	}
	data, err := json.MarshalIndent(workspaces, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func addRecentWorkspace(root string) error {
	if root == "" {
		return nil
	}
	workspaces, err := loadRecentWorkspaces()
	if err != nil {
		return err
	}
	next := []string{root}
	for _, workspace := range workspaces {
		if workspace == root {
			continue
		}
		next = append(next, workspace)
		if len(next) == recentWorkspaceLimit {
			break
		}
	}
	return saveRecentWorkspaces(next)
}
