package unit

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fastReadFile/internal/replay"
	"fastReadFile/pkg/cache"
)

func TestCursorSaveWritesMainAndBackup(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := replay.NewCursorStore(root)
	cursor := cache.ReplayCursor{
		Version:         1,
		SegmentID:       2,
		BlockOffset:     64,
		RecordIndex:     3,
		WriteSeq:        99,
		UpdatedAtUnixMs: 123,
	}
	if err := store.Save("main-server", cursor); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	mainPath := filepath.Join(root, "meta", "replay", "main-server.cursor")
	backupPath := mainPath + ".bak"
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("main cursor missing: %v", err)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup cursor missing: %v", err)
	}
}

func TestCursorLoadFallsBackToBackupWhenMainCorrupt(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := replay.NewCursorStore(root)
	cursor := cache.ReplayCursor{
		Version:         1,
		SegmentID:       1,
		BlockOffset:     32,
		RecordIndex:     1,
		WriteSeq:        7,
		UpdatedAtUnixMs: 77,
	}
	if err := store.Save("main-server", cursor); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	mainPath := filepath.Join(root, "meta", "replay", "main-server.cursor")
	if err := os.WriteFile(mainPath, []byte("corrupt"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := store.Load("main-server")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.WriteSeq != cursor.WriteSeq {
		t.Fatalf("WriteSeq = %d, want %d", got.WriteSeq, cursor.WriteSeq)
	}
}

func TestCursorRejectsBackwardAck(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := replay.NewCursorStore(root)
	if err := store.Save("main-server", cache.ReplayCursor{
		Version:         1,
		SegmentID:       3,
		BlockOffset:     64,
		RecordIndex:     2,
		WriteSeq:        10,
		UpdatedAtUnixMs: 100,
	}); err != nil {
		t.Fatalf("Save(initial) error = %v", err)
	}

	err := store.Save("main-server", cache.ReplayCursor{
		Version:         1,
		SegmentID:       2,
		BlockOffset:     32,
		RecordIndex:     1,
		WriteSeq:        9,
		UpdatedAtUnixMs: 101,
	})
	if !errors.Is(err, cache.ErrCode(cache.ErrCursorInvalid)) {
		t.Fatalf("Save(backward) error = %v, want cursor invalid", err)
	}
}

func TestCursorLoadRejectsCRCMismatchWithoutBackup(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := replay.NewCursorStore(root)
	cursor := cache.ReplayCursor{
		Version:         1,
		SegmentID:       5,
		BlockOffset:     48,
		RecordIndex:     4,
		WriteSeq:        44,
		UpdatedAtUnixMs: 111,
	}
	if err := store.Save("main-server", cursor); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	mainPath := filepath.Join(root, "meta", "replay", "main-server.cursor")
	backupPath := mainPath + ".bak"
	if err := os.Remove(backupPath); err != nil {
		t.Fatalf("Remove(backup) error = %v", err)
	}
	file, err := os.OpenFile(mainPath, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if _, err := file.WriteAt([]byte{0xFF}, 0); err != nil {
		t.Fatalf("WriteAt() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_, err = store.Load("main-server")
	if !errors.Is(err, cache.ErrCode(cache.ErrCursorCorrupted)) {
		t.Fatalf("Load() error = %v, want cursor corrupted", err)
	}
}
