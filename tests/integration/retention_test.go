package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fastReadFile/internal/fsutil"
	"fastReadFile/internal/replay"
	"fastReadFile/internal/retention"
	"fastReadFile/internal/segment"
	"fastReadFile/pkg/cache"
)

func TestRetentionDeletesOnlyAckedExpiredSegments(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.RetentionDays = 30

	createSealedSegment(t, root, 1, 1, 10)
	createSealedSegment(t, root, 2, 11, 20)
	setAgeDays(t, filepath.Join(root, "segments", "000001.seg"), 31)
	setAgeDays(t, filepath.Join(root, "segments", "000002.seg"), 5)

	store := replay.NewCursorStore(root)
	if err := store.Save("main-server", cache.ReplayCursor{
		Version:         1,
		SegmentID:       2,
		BlockOffset:     0,
		RecordIndex:     0,
		WriteSeq:        20,
		UpdatedAtUnixMs: time.Now().UnixMilli(),
	}); err != nil {
		t.Fatalf("Save(cursor) error = %v", err)
	}

	manager := retention.NewManager(root, cfg, store)
	result, err := manager.Cleanup("main-server")
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if len(result.DeletedSegmentIDs) != 1 || result.DeletedSegmentIDs[0] != 1 {
		t.Fatalf("DeletedSegmentIDs = %v, want [1]", result.DeletedSegmentIDs)
	}
}

func TestRetentionUsesOldestUnackedAgeAsEffectiveMinimum(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.RetentionDays = 30

	createSealedSegment(t, root, 1, 1, 10)
	createSealedSegment(t, root, 2, 11, 20)
	setAgeDays(t, filepath.Join(root, "segments", "000001.seg"), 35)
	setAgeDays(t, filepath.Join(root, "segments", "000002.seg"), 40)

	store := replay.NewCursorStore(root)
	if err := store.Save("main-server", cache.ReplayCursor{
		Version:         1,
		SegmentID:       1,
		BlockOffset:     0,
		RecordIndex:     0,
		WriteSeq:        10,
		UpdatedAtUnixMs: time.Now().UnixMilli(),
	}); err != nil {
		t.Fatalf("Save(cursor) error = %v", err)
	}

	manager := retention.NewManager(root, cfg, store)
	result, err := manager.Cleanup("main-server")
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if len(result.DeletedSegmentIDs) != 0 {
		t.Fatalf("DeletedSegmentIDs = %v, want none", result.DeletedSegmentIDs)
	}
}

func TestDiskReadOnlyDisablesWritesAndReplayStillWorks(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	}); err != nil {
		t.Fatalf("WriteBatch(initial) error = %v", err)
	}

	fsutil.SetTestState(engine.Config().RootDir, fsutil.TestState{ReadOnly: true})
	defer fsutil.ClearTestState(engine.Config().RootDir)

	_, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 2, Payload: []byte("b")},
	})
	if !errors.Is(err, cache.ErrCode(cache.ErrReadOnlyFS)) {
		t.Fatalf("WriteBatch(read-only) error = %v, want read-only", err)
	}

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if batch.RecordCount != 1 {
		t.Fatalf("RecordCount = %d, want %d", batch.RecordCount, 1)
	}
}

func TestDiskFullReturnsErrDiskFull(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	fsutil.SetTestState(engine.Config().RootDir, fsutil.TestState{DiskFull: true})
	defer fsutil.ClearTestState(engine.Config().RootDir)

	_, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	})
	if !errors.Is(err, cache.ErrCode(cache.ErrDiskFull)) {
		t.Fatalf("WriteBatch(disk full) error = %v, want disk full", err)
	}
}

func createSealedSegment(t *testing.T, root string, segmentID, firstSeq, lastSeq uint64) {
	t.Helper()

	manager, err := segment.OpenManager(root, cache.DefaultConfig(root))
	if err != nil {
		t.Fatalf("OpenManager() error = %v", err)
	}
	block := mustBuildBlock(t, firstSeq, [][]byte{[]byte("x")}, []int64{100})
	if _, err := manager.AppendBlock(block, segment.BlockMeta{
		RecordCount:   1,
		FirstWriteSeq: firstSeq,
		LastWriteSeq:  lastSeq,
		MinEventTime:  100,
		MaxEventTime:  100,
		LastBatchSeq:  segmentID,
	}); err != nil {
		t.Fatalf("AppendBlock() error = %v", err)
	}
	if _, err := manager.SealActive(); err != nil {
		t.Fatalf("SealActive() error = %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func setAgeDays(t *testing.T, path string, days int) {
	t.Helper()

	when := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}
}
