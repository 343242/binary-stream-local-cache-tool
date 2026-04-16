package retention

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	cache "fastReadFile/internal/core"
	"fastReadFile/internal/replay"
	"fastReadFile/internal/segment"
)

type Result struct {
	DeletedSegmentIDs      []uint64
	EffectiveRetentionDays int
}

type Manager struct {
	root  string
	cfg   cache.Config
	store *replay.CursorStore
}

func NewManager(root string, cfg cache.Config, store *replay.CursorStore) *Manager {
	return &Manager{
		root:  root,
		cfg:   cfg,
		store: store,
	}
}

func (m *Manager) Cleanup(destination string) (Result, error) {
	cursor, err := m.store.Load(destination)
	if err != nil {
		return Result{}, err
	}

	segmentsDir := filepath.Join(m.root, "segments")
	entries, err := os.ReadDir(segmentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{EffectiveRetentionDays: m.cfg.RetentionDays}, nil
		}
		return Result{}, cache.NewError(cache.ErrIO, "cleanup_retention", segmentsDir, "read segment directory", err)
	}

	type candidate struct {
		id      uint64
		path    string
		ageDays int
		footer  segment.Footer
	}

	candidates := make([]candidate, 0, len(entries))
	oldestUnackedAge := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".seg") {
			continue
		}
		path := filepath.Join(segmentsDir, entry.Name())
		footer, err := segment.ReadFooter(path)
		if err != nil {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return Result{}, cache.NewError(cache.ErrIO, "cleanup_retention", path, "stat segment file", err)
		}
		ageDays := int(time.Since(info.ModTime()) / (24 * time.Hour))
		candidates = append(candidates, candidate{
			id:      footer.SegmentID,
			path:    path,
			ageDays: ageDays,
			footer:  footer,
		})
		if footer.LastWriteSeq > cursor.WriteSeq && ageDays > oldestUnackedAge {
			oldestUnackedAge = ageDays
		}
	}

	effectiveDays := m.cfg.RetentionDays
	if oldestUnackedAge > effectiveDays {
		effectiveDays = oldestUnackedAge
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].id < candidates[j].id })
	result := Result{EffectiveRetentionDays: effectiveDays}
	for _, candidate := range candidates {
		if candidate.ageDays <= effectiveDays {
			continue
		}
		cursor, err = m.store.Load(destination)
		if err != nil {
			return Result{}, err
		}
		if candidate.footer.LastWriteSeq > cursor.WriteSeq {
			continue
		}
		if err := os.Remove(candidate.path); err != nil {
			return Result{}, cache.NewError(cache.ErrIO, "cleanup_retention", candidate.path, "remove expired segment", err)
		}
		result.DeletedSegmentIDs = append(result.DeletedSegmentIDs, candidate.id)
	}
	return result, nil
}

func segmentIDFromPath(path string) (uint64, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strconv.ParseUint(name, 10, 64)
}
