package unit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"fastReadFile/internal/wal"
)

func TestWALAppendAndScanRoundTrip(t *testing.T) {
	t.Parallel()

	log, err := wal.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer log.Close()

	block := []byte("block-one")
	meta, err := log.Append(1, block)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if meta.BatchSeq != 1 {
		t.Fatalf("BatchSeq = %d, want %d", meta.BatchSeq, 1)
	}

	entries, err := log.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 1)
	}
	if !bytes.Equal(entries[0].Block, block) {
		t.Fatalf("Block = %q, want %q", entries[0].Block, block)
	}
}

func TestWALTruncateCorruptedTail(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	log, err := wal.Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	first, err := log.Append(1, []byte("first"))
	if err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if _, err := log.Append(2, []byte("second")); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	path := filepath.Join(root, "wal", "active.wal")
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if err := file.Truncate(stat.Size() - 3); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}

	log, err = wal.Open(root)
	if err != nil {
		t.Fatalf("Open(reopen) error = %v", err)
	}
	defer log.Close()

	entries, err := log.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 1)
	}
	if entries[0].BatchSeq != 1 {
		t.Fatalf("BatchSeq = %d, want %d", entries[0].BatchSeq, 1)
	}
	if err := log.TruncateAfter(first.EndOffset); err != nil {
		t.Fatalf("TruncateAfter() error = %v", err)
	}

	entries, err = log.Scan()
	if err != nil {
		t.Fatalf("Scan(after truncate) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries after truncate) = %d, want %d", len(entries), 1)
	}
}

func TestWALCheckpointRoundTrip(t *testing.T) {
	t.Parallel()

	store := wal.NewCheckpointStore(t.TempDir())
	want := wal.Checkpoint{
		LastBatchSeq:     9,
		LastWALEndOffset: 1024,
		UpdatedAtUnixMs:  77,
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.LastBatchSeq != want.LastBatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", got.LastBatchSeq, want.LastBatchSeq)
	}
	if got.LastWALEndOffset != want.LastWALEndOffset {
		t.Fatalf("LastWALEndOffset = %d, want %d", got.LastWALEndOffset, want.LastWALEndOffset)
	}
}
