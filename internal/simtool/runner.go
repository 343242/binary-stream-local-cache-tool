package simtool

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"fastReadFile/pkg/cache"
)

type RunConfig struct {
	RootDir   string
	Profile   Profile
	BatchSize int
	StartTime time.Time
}

func ValidateRoot(root string) error {
	if root == "" {
		return cache.NewError(cache.ErrValidation, "cachesim_validate_root", root, "root is required", nil)
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return cache.NewError(cache.ErrIO, "cachesim_validate_root", root, "stat root", err)
	}
	if !info.IsDir() {
		return cache.NewError(cache.ErrValidation, "cachesim_validate_root", root, "root must be a directory", nil)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return cache.NewError(cache.ErrIO, "cachesim_validate_root", root, "read root directory", err)
	}
	if len(entries) > 0 {
		return cache.NewError(cache.ErrValidation, "cachesim_validate_root", root, "root must be empty", nil)
	}
	return nil
}

func Run(ctx context.Context, cfg RunConfig) (Report, error) {
	if err := ValidateRoot(cfg.RootDir); err != nil {
		return Report{}, err
	}
	profile, err := normalizeProfile(cfg.Profile)
	if err != nil {
		return Report{}, cache.NewError(cache.ErrValidation, "cachesim_run", cfg.RootDir, err.Error(), nil)
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	startTime := cfg.StartTime
	if startTime.IsZero() {
		startTime = time.UnixMilli(1_710_000_000_000)
	}

	engine, err := cache.Open(cache.DefaultConfig(cfg.RootDir))
	if err != nil {
		return Report{}, err
	}

	generator := NewBatchGenerator(profile, batchSize, startTime.UnixMilli())
	startWall := time.Now()
	var (
		batchCount    int
		totalBatchDur time.Duration
		maxBatchDur   time.Duration
	)
	for {
		if err := ctx.Err(); err != nil {
			_ = engine.Close()
			return Report{}, err
		}
		batch, ok := generator.NextBatch()
		if !ok {
			break
		}
		batchStart := time.Now()
		if _, err := engine.WriteBatch(ctx, batch); err != nil {
			_ = engine.Close()
			return Report{}, err
		}
		batchDur := time.Since(batchStart)
		totalBatchDur += batchDur
		if batchDur > maxBatchDur {
			maxBatchDur = batchDur
		}
		batchCount++
	}
	totalDuration := time.Since(startWall)
	if err := engine.Close(); err != nil {
		return Report{}, err
	}

	segmentCount, err := countFilesWithExt(filepath.Join(cfg.RootDir, "segments"), ".seg")
	if err != nil {
		return Report{}, err
	}
	walFileCount, err := countFilesWithExt(filepath.Join(cfg.RootDir, "wal"), ".wal")
	if err != nil {
		return Report{}, err
	}
	workspaceBytes, err := measureWorkspaceBytes(cfg.RootDir)
	if err != nil {
		return Report{}, err
	}

	avgBatchDur := time.Duration(0)
	if batchCount > 0 {
		avgBatchDur = totalBatchDur / time.Duration(batchCount)
	}
	return Report{
		RootDir:          cfg.RootDir,
		ProfileName:      string(profile.Name),
		Gateways:         profile.Gateways,
		PointsPerGateway: profile.PointsPerGateway,
		Rounds:           profile.Rounds,
		Records:          profile.TotalRecords(),
		PayloadBytes:     PayloadBytes,
		BatchSize:        batchSize,
		TotalDuration:    totalDuration,
		AvgBatchDuration: avgBatchDur,
		MaxBatchDuration: maxBatchDur,
		SegmentCount:     segmentCount,
		WALFileCount:     walFileCount,
		WorkspaceBytes:   workspaceBytes,
	}, nil
}

func countFilesWithExt(dir, ext string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, cache.NewError(cache.ErrIO, "cachesim_count_files", dir, "read directory", err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ext {
			count++
		}
	}
	return count, nil
}

func measureWorkspaceBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, cache.NewError(cache.ErrIO, "cachesim_measure_workspace", root, fmt.Sprintf("walk workspace %s", root), err)
	}
	return total, nil
}
