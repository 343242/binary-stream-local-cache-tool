package integration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fastReadFile/internal/codec"
	"fastReadFile/internal/recovery"
	"fastReadFile/internal/segment"
	"fastReadFile/internal/wal"
	"fastReadFile/pkg/cache"
)

func TestRecoveryReplaysWALOnlyBatchIntoSegment(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	block := mustBuildBlock(t, 1, [][]byte{[]byte("a")}, []int64{100})
	log := mustOpenWAL(t, root)
	if _, err := log.Append(1, block); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	state, err := recovery.Recover(context.Background(), root, cfg)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if state.NextWriteSeq != 2 {
		t.Fatalf("NextWriteSeq = %d, want %d", state.NextWriteSeq, 2)
	}

	data, err := os.ReadFile(filepath.Join(root, "segments", "000001.seg"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	decoded, err := codec.ParseBlock(data)
	if err != nil {
		t.Fatalf("ParseBlock() error = %v", err)
	}
	if decoded.LastWriteSeq != 1 {
		t.Fatalf("LastWriteSeq = %d, want %d", decoded.LastWriteSeq, 1)
	}
}

func TestRecoveryResumesNextWriteSeqFromGlobalMaxPlusOne(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	manager, err := segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager() error = %v", err)
	}
	if _, err := manager.AppendBlock(mustBuildBlock(t, 1, [][]byte{[]byte("a"), []byte("b")}, []int64{100, 101}), segment.BlockMeta{
		RecordCount:   2,
		FirstWriteSeq: 1,
		LastWriteSeq:  2,
		MinEventTime:  100,
		MaxEventTime:  101,
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

	state, err := recovery.Recover(context.Background(), root, cfg)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if state.NextWriteSeq != 3 {
		t.Fatalf("NextWriteSeq = %d, want %d", state.NextWriteSeq, 3)
	}
}

func TestRecoveryRebuildsActiveSegmentMetadataBeforeContinuingWrites(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	manager, err := segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager(first open) error = %v", err)
	}

	firstBlock := mustBuildBlock(t, 1, [][]byte{[]byte("a"), []byte("b")}, []int64{200, 300})
	if _, err := manager.AppendBlock(firstBlock, segment.BlockMeta{
		RecordCount:   2,
		FirstWriteSeq: 1,
		LastWriteSeq:  2,
		MinEventTime:  200,
		MaxEventTime:  300,
		LastBatchSeq:  7,
	}); err != nil {
		t.Fatalf("AppendBlock(first) error = %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	state, err := recovery.Recover(context.Background(), root, cfg)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if state.ActiveSegmentID != 1 {
		t.Fatalf("ActiveSegmentID = %d, want %d", state.ActiveSegmentID, 1)
	}
	if state.NextWriteSeq != 3 {
		t.Fatalf("NextWriteSeq = %d, want %d", state.NextWriteSeq, 3)
	}

	manager, err = segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager(reopen) error = %v", err)
	}
	secondBlock := mustBuildBlock(t, 3, [][]byte{[]byte("c"), []byte("d")}, []int64{50, 400})
	appendResult, err := manager.AppendBlock(secondBlock, segment.BlockMeta{
		RecordCount:   2,
		FirstWriteSeq: 3,
		LastWriteSeq:  4,
		MinEventTime:  50,
		MaxEventTime:  400,
		LastBatchSeq:  8,
	})
	if err != nil {
		t.Fatalf("AppendBlock(second) error = %v", err)
	}
	if appendResult.SegmentID != 1 {
		t.Fatalf("SegmentID = %d, want %d", appendResult.SegmentID, 1)
	}
	if appendResult.BlockOffset != uint64(len(firstBlock)) {
		t.Fatalf("BlockOffset = %d, want %d", appendResult.BlockOffset, len(firstBlock))
	}
	if _, err := os.Stat(filepath.Join(root, "segments", "000002.seg")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(000002.seg) error = %v, want not-exist", err)
	}

	footer, err := manager.SealActive()
	if err != nil {
		t.Fatalf("SealActive() error = %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close(second) error = %v", err)
	}

	if footer.DataEndOffset != uint64(len(firstBlock)+len(secondBlock)) {
		t.Fatalf("DataEndOffset = %d, want %d", footer.DataEndOffset, len(firstBlock)+len(secondBlock))
	}
	if footer.RecordCount != 4 {
		t.Fatalf("RecordCount = %d, want %d", footer.RecordCount, 4)
	}
	if footer.FirstWriteSeq != 1 {
		t.Fatalf("FirstWriteSeq = %d, want %d", footer.FirstWriteSeq, 1)
	}
	if footer.LastWriteSeq != 4 {
		t.Fatalf("LastWriteSeq = %d, want %d", footer.LastWriteSeq, 4)
	}
	if footer.MinEventTime != 50 {
		t.Fatalf("MinEventTime = %d, want %d", footer.MinEventTime, 50)
	}
	if footer.MaxEventTime != 400 {
		t.Fatalf("MaxEventTime = %d, want %d", footer.MaxEventTime, 400)
	}
	if footer.LastBatchSeq != 8 {
		t.Fatalf("LastBatchSeq = %d, want %d", footer.LastBatchSeq, 8)
	}
}

func TestRecoveryRebuildsActiveSegmentMetadataBeforeSealAfterCheckpointTruncatesWAL(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	manager, err := segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager(first open) error = %v", err)
	}

	block := mustBuildBlock(t, 1, [][]byte{[]byte("a"), []byte("b")}, []int64{150, 250})
	if _, err := manager.AppendBlock(block, segment.BlockMeta{
		RecordCount:   2,
		FirstWriteSeq: 1,
		LastWriteSeq:  2,
		MinEventTime:  150,
		MaxEventTime:  250,
		LastBatchSeq:  7,
	}); err != nil {
		t.Fatalf("AppendBlock() error = %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	log := mustOpenWAL(t, root)
	meta, err := log.Append(7, block)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close(wal) error = %v", err)
	}
	if err := wal.NewCheckpointStore(root).Save(wal.Checkpoint{
		LastBatchSeq:     7,
		LastWALEndOffset: meta.EndOffset,
		UpdatedAtUnixMs:  1234,
	}); err != nil {
		t.Fatalf("Save(checkpoint) error = %v", err)
	}

	log = mustOpenWAL(t, root)
	if err := log.TruncateAfter(0); err != nil {
		t.Fatalf("TruncateAfter() error = %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close(truncated wal) error = %v", err)
	}

	if _, err := recovery.Recover(context.Background(), root, cfg); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}

	manager, err = segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager(reopen) error = %v", err)
	}
	footer, err := manager.SealActive()
	if err != nil {
		t.Fatalf("SealActive() error = %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close(second) error = %v", err)
	}

	if footer.DataEndOffset != uint64(len(block)) {
		t.Fatalf("DataEndOffset = %d, want %d", footer.DataEndOffset, len(block))
	}
	if footer.RecordCount != 2 {
		t.Fatalf("RecordCount = %d, want %d", footer.RecordCount, 2)
	}
	if footer.FirstWriteSeq != 1 {
		t.Fatalf("FirstWriteSeq = %d, want %d", footer.FirstWriteSeq, 1)
	}
	if footer.LastWriteSeq != 2 {
		t.Fatalf("LastWriteSeq = %d, want %d", footer.LastWriteSeq, 2)
	}
	if footer.MinEventTime != 150 {
		t.Fatalf("MinEventTime = %d, want %d", footer.MinEventTime, 150)
	}
	if footer.MaxEventTime != 250 {
		t.Fatalf("MaxEventTime = %d, want %d", footer.MaxEventTime, 250)
	}
	if footer.LastBatchSeq != 7 {
		t.Fatalf("LastBatchSeq = %d, want %d", footer.LastBatchSeq, 7)
	}
}

func TestRecoveryTruncatesPartialSegmentBlockAndReplaysFromWAL(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	firstBlock := mustBuildBlock(t, 1, [][]byte{[]byte("a")}, []int64{100})
	secondBlock := mustBuildBlock(t, 2, [][]byte{[]byte("b")}, []int64{101})

	segmentPath := filepath.Join(root, "segments", "000001.seg")
	if err := os.MkdirAll(filepath.Dir(segmentPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(segmentPath, append(append([]byte{}, firstBlock...), secondBlock[:len(secondBlock)-5]...), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	log := mustOpenWAL(t, root)
	if _, err := log.Append(1, firstBlock); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if _, err := log.Append(2, secondBlock); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	state, err := recovery.Recover(context.Background(), root, cfg)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if state.NextWriteSeq != 3 {
		t.Fatalf("NextWriteSeq = %d, want %d", state.NextWriteSeq, 3)
	}

	data, err := os.ReadFile(segmentPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	remaining := data
	parsed := 0
	for len(remaining) > 0 {
		blockLen, err := codec.BlockLength(remaining)
		if err != nil {
			t.Fatalf("BlockLength(%d) error = %v", parsed, err)
		}
		if _, err := codec.ParseBlock(remaining[:blockLen]); err != nil {
			t.Fatalf("ParseBlock(%d) error = %v", parsed, err)
		}
		remaining = remaining[blockLen:]
		parsed++
		if parsed == 2 {
			break
		}
	}
	if parsed != 2 {
		t.Fatalf("parsed block count = %d, want %d", parsed, 2)
	}
}

func TestRecoveryRejectsSequenceConflict(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	segmentPath := filepath.Join(root, "segments", "000001.seg")
	if err := os.MkdirAll(filepath.Dir(segmentPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(segmentPath, mustBuildBlock(t, 1, [][]byte{[]byte("segment")}, []int64{100}), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	log := mustOpenWAL(t, root)
	if _, err := log.Append(1, mustBuildBlock(t, 1, [][]byte{[]byte("wal")}, []int64{100})); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_, err := recovery.Recover(context.Background(), root, cfg)
	if !errors.Is(err, cache.ErrCode(cache.ErrSequenceConflict)) {
		t.Fatalf("Recover() error = %v, want sequence conflict", err)
	}
}

func TestRecoveryTruncatesOrphanTailBeyondValidFooter(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	manager, err := segment.OpenManager(root, cfg)
	if err != nil {
		t.Fatalf("OpenManager() error = %v", err)
	}
	block := mustBuildBlock(t, 1, [][]byte{[]byte("a")}, []int64{100})
	if _, err := manager.AppendBlock(block, segment.BlockMeta{
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

	path := filepath.Join(root, "segments", "000001.seg")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := os.WriteFile(path, append(append([]byte{}, original...), []byte("tail")...), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := recovery.Recover(context.Background(), root, cfg); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(after) error = %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("segment file was not truncated back to valid footer boundary")
	}
}

func mustBuildBlock(t *testing.T, firstSeq uint64, payloads [][]byte, eventTimes []int64) []byte {
	t.Helper()

	records := make([][]byte, 0, len(payloads))
	for idx, payload := range payloads {
		record, err := codec.EncodeRecord(firstSeq+uint64(idx), cache.RawRecord{
			EventTimeUnixMs: eventTimes[idx],
			Payload:         payload,
		})
		if err != nil {
			t.Fatalf("EncodeRecord() error = %v", err)
		}
		records = append(records, record)
	}
	block, err := codec.BuildBlock(records, firstSeq, firstSeq+uint64(len(payloads))-1, eventTimes[0], eventTimes[len(eventTimes)-1])
	if err != nil {
		t.Fatalf("BuildBlock() error = %v", err)
	}
	return block
}

func mustOpenWAL(t *testing.T, root string) *wal.Log {
	t.Helper()

	log, err := wal.Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return log
}
