package unit

import (
	"os"
	"path/filepath"
	"testing"

	"fastReadFile/internal/segment"
	"fastReadFile/pkg/cache"
)

func TestSegmentEmptySystemStartsAtOne(t *testing.T) {
	t.Parallel()

	manager, err := segment.OpenManager(t.TempDir(), cache.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("OpenManager() error = %v", err)
	}
	defer manager.Close()

	if manager.CurrentSegmentID() != 1 {
		t.Fatalf("CurrentSegmentID = %d, want %d", manager.CurrentSegmentID(), 1)
	}
}

func TestSegmentRecoveredSystemUsesMaxValidIDPlusOne(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	manager, err := segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager(first open) error = %v", err)
	}
	if _, err := manager.AppendBlock([]byte("block"), segment.BlockMeta{
		RecordCount:   1,
		FirstWriteSeq: 1,
		LastWriteSeq:  1,
		MinEventTime:  100,
		MaxEventTime:  100,
		LastBatchSeq:  1,
	}); err != nil {
		t.Fatalf("AppendBlock() error = %v", err)
	}
	if _, err := manager.SealActive(); err != nil {
		t.Fatalf("SealActive() error = %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	manager, err = segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager(reopen) error = %v", err)
	}
	defer manager.Close()

	if manager.CurrentSegmentID() != 2 {
		t.Fatalf("CurrentSegmentID = %d, want %d", manager.CurrentSegmentID(), 2)
	}
}

func TestSegmentSealWritesValidFooter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, err := segment.OpenManager(root, cache.DefaultConfig(root))
	if err != nil {
		t.Fatalf("OpenManager() error = %v", err)
	}
	if _, err := manager.AppendBlock([]byte("block"), segment.BlockMeta{
		RecordCount:   3,
		FirstWriteSeq: 7,
		LastWriteSeq:  9,
		MinEventTime:  10,
		MaxEventTime:  20,
		LastBatchSeq:  2,
	}); err != nil {
		t.Fatalf("AppendBlock() error = %v", err)
	}

	footer, err := manager.SealActive()
	if err != nil {
		t.Fatalf("SealActive() error = %v", err)
	}
	if footer.LastWriteSeq != 9 {
		t.Fatalf("LastWriteSeq = %d, want %d", footer.LastWriteSeq, 9)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	footer, err = segment.ReadFooter(filepath.Join(root, "segments", "000001.seg"))
	if err != nil {
		t.Fatalf("ReadFooter() error = %v", err)
	}
	if footer.RecordCount != 3 {
		t.Fatalf("RecordCount = %d, want %d", footer.RecordCount, 3)
	}
}

func TestSegmentRejectsCorruptFooter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, err := segment.OpenManager(root, cache.DefaultConfig(root))
	if err != nil {
		t.Fatalf("OpenManager() error = %v", err)
	}
	if _, err := manager.AppendBlock([]byte("block"), segment.BlockMeta{
		RecordCount:   1,
		FirstWriteSeq: 1,
		LastWriteSeq:  1,
		MinEventTime:  1,
		MaxEventTime:  1,
		LastBatchSeq:  1,
	}); err != nil {
		t.Fatalf("AppendBlock() error = %v", err)
	}
	if _, err := manager.SealActive(); err != nil {
		t.Fatalf("SealActive() error = %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	path := filepath.Join(root, "segments", "000001.seg")
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	stat, err := file.Stat()
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if _, err := file.WriteAt([]byte{0xFF}, stat.Size()-1); err != nil {
		t.Fatalf("WriteAt() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := segment.ReadFooter(path); err == nil {
		t.Fatalf("expected footer corruption error")
	}
}

func TestSegmentRotatesWhenNextBlockExceedsTargetPlusSlack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.SegmentTargetSizeBytes = 10
	cfg.SegmentSlackSizeBytes = 2

	manager, err := segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager() error = %v", err)
	}
	defer manager.Close()

	if _, err := manager.AppendBlock([]byte("12345678"), segment.BlockMeta{
		RecordCount:   1,
		FirstWriteSeq: 1,
		LastWriteSeq:  1,
		MinEventTime:  1,
		MaxEventTime:  1,
		LastBatchSeq:  1,
	}); err != nil {
		t.Fatalf("AppendBlock(first) error = %v", err)
	}
	result, err := manager.AppendBlock([]byte("abcde"), segment.BlockMeta{
		RecordCount:   1,
		FirstWriteSeq: 2,
		LastWriteSeq:  2,
		MinEventTime:  2,
		MaxEventTime:  2,
		LastBatchSeq:  2,
	})
	if err != nil {
		t.Fatalf("AppendBlock(second) error = %v", err)
	}
	if result.SegmentID != 2 {
		t.Fatalf("SegmentID = %d, want %d", result.SegmentID, 2)
	}
}
