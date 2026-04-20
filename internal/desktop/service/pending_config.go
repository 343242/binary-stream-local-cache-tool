package service

import (
	"encoding/json"
	"os"
	"path/filepath"

	core "fastReadFile/internal/core"
)

const pendingConfigFileName = "gui-pending-config.json"

type PendingConfigFile struct {
	Root   string      `json:"root"`
	Config core.Config `json:"config"`
}

func SavePendingConfig(root string, cfg core.Config) error {
	cfg.RootDir = root
	payload := PendingConfigFile{
		Root:   root,
		Config: cfg,
	}

	path := pendingConfigPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return wrapServiceError("save_pending_config", path, err)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return wrapServiceError("save_pending_config", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return wrapServiceError("save_pending_config", path, err)
	}
	return nil
}

func LoadPendingConfig(root string) (PendingConfigFile, error) {
	path := pendingConfigPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultPendingConfig(root), nil
		}
		return PendingConfigFile{}, wrapServiceError("load_pending_config", path, err)
	}

	var payload PendingConfigFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return PendingConfigFile{}, wrapServiceError("load_pending_config", path, err)
	}

	payload.Root = root
	payload.Config.RootDir = root
	return payload, nil
}

func pendingConfigPath(root string) string {
	return filepath.Join(root, "meta", pendingConfigFileName)
}

func defaultPendingConfig(root string) PendingConfigFile {
	return PendingConfigFile{
		Root:   root,
		Config: core.DefaultConfig(root),
	}
}
