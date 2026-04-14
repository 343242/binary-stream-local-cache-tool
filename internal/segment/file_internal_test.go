package segment

import (
	"os"
	"path/filepath"
	"testing"

	cache "fastReadFile/internal/core"
)

func TestSealRollsBackFooterOnSyncFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "000001.seg")
	file, err := openSegmentFile(path, 1, cache.DefaultConfig(root))
	if err != nil {
		t.Fatalf("openSegmentFile() error = %v", err)
	}
	defer func() {
		segmentSyncHook = nil
		_ = file.Close()
	}()

	if _, err := file.AppendBlock([]byte("block"), BlockMeta{
		RecordCount:   1,
		FirstWriteSeq: 1,
		LastWriteSeq:  1,
		MinEventTime:  10,
		MaxEventTime:  10,
		LastBatchSeq:  1,
	}); err != nil {
		t.Fatalf("AppendBlock() error = %v", err)
	}
	statBefore, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(before) error = %v", err)
	}

	segmentSyncHook = func(_ *os.File) error {
		return os.ErrPermission
	}
	if _, err := file.Seal(); err == nil {
		t.Fatalf("Seal() error = nil, want sync failure")
	}
	statAfter, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(after) error = %v", err)
	}
	if statAfter.Size() != statBefore.Size() {
		t.Fatalf("file size after failed seal = %d, want %d", statAfter.Size(), statBefore.Size())
	}
}
