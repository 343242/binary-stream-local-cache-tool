package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/ops"
	"fastReadFile/pkg/cache"
)

func GetOverview(root string) (viewmodel.Overview, error) {
	valid, reason, err := inspectWorkspaceRoot(root)
	if err != nil {
		return viewmodel.Overview{}, wrapServiceError("get_overview", root, err)
	}
	if !valid {
		return viewmodel.Overview{}, newInvalidRootError("get_overview", root, reason)
	}

	snapshot, err := loadObserverStats(root)
	if err != nil {
		return viewmodel.Overview{}, wrapServiceError("get_overview", root, err)
	}

	activeSegmentID, activeSegmentPath, err := latestSegment(root)
	if err != nil {
		return viewmodel.Overview{}, wrapServiceError("get_overview", root, err)
	}

	overview := viewmodel.Overview{
		RootPath:                  root,
		WorkspaceMode:             unavailableValue,
		LockMode:                  unavailableValue,
		Health:                    unavailableValue,
		SegmentCount:              snapshot.Capacity.SegmentCount,
		ActiveSegmentID:           activeSegmentID,
		ActiveSegmentSizeBytes:    fileSizeOrZero(activeSegmentPath),
		WALSizeBytes:              fileSizeOrZero(filepath.Join(root, "wal", "active.wal")),
		NextWriteSeq:              snapshot.Capacity.NextWriteSeq,
		RetentionDays:             snapshot.Capacity.RetentionDays,
		CheckpointsTotal:          snapshot.IO.CheckpointsTotal,
		SegmentFsyncTotal:         snapshot.IO.SegmentFsyncTotal,
		LastAckedWriteSeq:         snapshot.Replay.LastAckedWriteSeq,
		GracefulShutdownsTotal:    snapshot.Health.GracefulShutdownsTotal,
		UngracefulRecoveriesTotal: snapshot.Health.UngracefulShutdownRecoveriesTotal,
		SegmentTailRepairsTotal:   snapshot.Health.SegmentTailRepairsTotal,
		Warnings: []viewmodel.Warning{
			{
				Code:     "workspace-state-unavailable",
				Severity: "warning",
				Title:    "Workspace Session State Unavailable",
				Message:  "Workspace mode, lock mode, health state, and backlog estimates are unavailable from the read-only desktop service snapshot.",
			},
		},
		LastRefreshedAt:           time.Now().UnixMilli(),
		IsStale:                   false,
	}
	return overview, nil
}

func loadObserverStats(root string) (cache.StatsSnapshot, error) {
	data, err := ops.Stats(root, "json")
	if err != nil {
		return cache.StatsSnapshot{}, err
	}

	var snapshot cache.StatsSnapshot
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		return cache.StatsSnapshot{}, err
	}
	return snapshot, nil
}

func latestSegment(root string) (uint64, string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "segments"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, "", nil
		}
		return 0, "", err
	}

	var latestID uint64
	var latestName string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".seg" {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), 10, 64)
		if err != nil {
			continue
		}
		if id >= latestID {
			latestID = id
			latestName = entry.Name()
		}
	}

	if latestName == "" {
		return 0, "", nil
	}
	return latestID, filepath.Join(root, "segments", latestName), nil
}

func fileSizeOrZero(path string) int64 {
	if path == "" {
		return 0
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
