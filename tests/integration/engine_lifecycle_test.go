package integration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fastReadFile/internal/codec"
	"fastReadFile/internal/core"
	"fastReadFile/internal/desktop/service"
	"fastReadFile/internal/desktop/writer"
	"fastReadFile/internal/segment"
	"fastReadFile/internal/wal"
	"fastReadFile/pkg/cache"
)

func TestWriterHostStartAndStop(t *testing.T) {
	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatal(err)
	}

	host := writer.NewHost()
	cfg := core.DefaultConfig(root)

	if err := host.Start(context.Background(), root, cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got := host.Status().LifecycleState; got != string(writer.LifecycleRunning) {
		t.Fatalf("LifecycleState = %q, want %q", got, writer.LifecycleRunning)
	}
	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestWriterHostSubmitBlocksWhenQueueFull(t *testing.T) {
	host := writer.NewHostWithQueueCapacity(1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	host.DebugFillQueueForTest()
	err := host.Submit(ctx, []core.RawRecord{{EventTimeUnixMs: 1, Payload: []byte("x")}})
	if err == nil {
		t.Fatal("expected context expiration while queue is full")
	}
}

func TestWriterHostSubmitRejectsWhenNotRunning(t *testing.T) {
	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatal(err)
	}

	host := writer.NewHostWithQueueCapacity(1)
	records := []core.RawRecord{{EventTimeUnixMs: 1, Payload: []byte("x")}}

	if err := host.Submit(context.Background(), records); err == nil {
		t.Fatal("expected Submit() rejection before start")
	}
	if got := host.DebugQueueLenForTest(); got != 0 {
		t.Fatalf("queue len before start = %d, want %d", got, 0)
	}

	cfg := core.DefaultConfig(root)
	if err := host.Start(context.Background(), root, cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if err := host.Submit(context.Background(), records); err == nil {
		t.Fatal("expected Submit() rejection after stop")
	}
	if got := host.DebugQueueLenForTest(); got != 0 {
		t.Fatalf("queue len after stop = %d, want %d", got, 0)
	}
}

func TestWriterHostStopFailurePreservesStoppingLifecycle(t *testing.T) {
	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatal(err)
	}

	host := writer.NewHost()
	cfg := core.DefaultConfig(root)

	if err := host.Start(context.Background(), root, cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := host.Stop(ctx)
	if err == nil {
		t.Fatal("expected Stop() error from expired context")
	}

	status := host.Status()
	if status.LifecycleState != string(writer.LifecycleStopping) {
		t.Fatalf("LifecycleState = %q, want %q", status.LifecycleState, writer.LifecycleStopping)
	}
	if status.LastError == "" {
		t.Fatal("LastError = empty, want shutdown failure recorded")
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(cleanup) error = %v", err)
	}
}

func TestWriterHostStartResetsQueueAcrossRestarts(t *testing.T) {
	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatal(err)
	}

	host := writer.NewHostWithQueueCapacity(1)
	cfg := core.DefaultConfig(root)

	host.DebugFillQueueForTest()
	if got := host.DebugQueueLenForTest(); got != 1 {
		t.Fatalf("queue len before first start = %d, want %d", got, 1)
	}

	if err := host.Start(context.Background(), root, cfg); err != nil {
		t.Fatalf("Start(first) error = %v", err)
	}
	if got := host.DebugQueueLenForTest(); got != 0 {
		t.Fatalf("queue len after first start = %d, want %d", got, 0)
	}
	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(first) error = %v", err)
	}

	host.DebugFillQueueForTest()
	if got := host.DebugQueueLenForTest(); got != 1 {
		t.Fatalf("queue len before restart = %d, want %d", got, 1)
	}

	if err := host.Start(context.Background(), root, cfg); err != nil {
		t.Fatalf("Start(second) error = %v", err)
	}
	if got := host.DebugQueueLenForTest(); got != 0 {
		t.Fatalf("queue len after restart = %d, want %d", got, 0)
	}
	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(second) error = %v", err)
	}
}

func TestWriterHostStopDrainsAcceptedBatches(t *testing.T) {
	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatal(err)
	}

	host := writer.NewHostWithQueueCapacity(1)
	cfg := core.DefaultConfig(root)
	if err := host.Start(context.Background(), root, cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	writeStarted := make(chan struct{}, 1)
	releaseWrite := make(chan struct{})
	cache.SetWriteBatchHookForTesting(func(stage string) {
		if stage != "after_wal_append_before_segment" {
			return
		}
		select {
		case writeStarted <- struct{}{}:
		default:
		}
		<-releaseWrite
	})
	defer cache.SetWriteBatchHookForTesting(nil)

	if err := host.Submit(context.Background(), []core.RawRecord{{EventTimeUnixMs: 1, Payload: []byte("a")}}); err != nil {
		t.Fatalf("Submit(first) error = %v", err)
	}
	<-writeStarted

	if err := host.Submit(context.Background(), []core.RawRecord{{EventTimeUnixMs: 2, Payload: []byte("b")}}); err != nil {
		t.Fatalf("Submit(second) error = %v", err)
	}

	stopErrCh := make(chan error, 1)
	go func() {
		stopErrCh <- host.Stop(context.Background())
	}()

	deadline := time.Now().Add(time.Second)
	for !host.DebugStopSignaledForTest() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for stop signal")
		}
		time.Sleep(time.Millisecond)
	}

	close(releaseWrite)
	if err := <-stopErrCh; err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	engine, err := cache.Open(cache.DefaultConfig(root))
	if err != nil {
		t.Fatalf("Open(replay) error = %v", err)
	}
	defer engine.Close()

	batch, err := engine.Replay(context.Background(), "writer-host-stop-drain", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if batch.RecordCount != 2 {
		t.Fatalf("RecordCount = %d, want %d", batch.RecordCount, 2)
	}
}

func TestClosePersistsAckedCursorState(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
		{EventTimeUnixMs: 2, Payload: []byte("b")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if _, err := engine.Ack(context.Background(), "main-server", batch.NextCursor); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	engine, err = cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open(reopen) error = %v", err)
	}
	defer engine.Close()

	nextBatch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay(reopen) error = %v", err)
	}
	if nextBatch.RecordCount != 0 {
		t.Fatalf("RecordCount = %d, want %d", nextBatch.RecordCount, 0)
	}
}

func TestWriteBatchPersistsCheckpointWhenThresholdCrossed(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1
	cfg.CheckpointInterval = time.Hour

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	result, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	})
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if checkpoint.LastBatchSeq != result.BatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, result.BatchSeq)
	}
	if checkpoint.LastWALEndOffset <= 0 {
		t.Fatalf("LastWALEndOffset = %d, want > 0", checkpoint.LastWALEndOffset)
	}
}

func TestWriteBatchForcesCheckpointWhenSegmentRotates(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1 << 30
	cfg.CheckpointInterval = time.Hour
	cfg.SegmentTargetSizeBytes = 1
	cfg.SegmentSlackSizeBytes = 1

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	first, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	})
	if err != nil {
		t.Fatalf("WriteBatch(first) error = %v", err)
	}
	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load(first checkpoint) error = %v", err)
	}
	if checkpoint.LastBatchSeq != 0 {
		t.Fatalf("LastBatchSeq after first write = %d, want %d", checkpoint.LastBatchSeq, 0)
	}

	second, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 2, Payload: []byte("b")},
	})
	if err != nil {
		t.Fatalf("WriteBatch(second) error = %v", err)
	}
	if second.SegmentID == first.SegmentID {
		t.Fatalf("SegmentID = %d, want rotation to a new segment", second.SegmentID)
	}

	checkpoint, err = wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load(second checkpoint) error = %v", err)
	}
	if checkpoint.LastBatchSeq != second.BatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, second.BatchSeq)
	}
	if checkpoint.LastWALEndOffset <= 0 {
		t.Fatalf("LastWALEndOffset = %d, want > 0", checkpoint.LastWALEndOffset)
	}
}

func TestShutdownReturnsTimeoutWhenContextExpires(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := engine.Shutdown(ctx)
	if !errors.Is(err, cache.ErrCode(cache.ErrTimeout)) {
		t.Fatalf("Shutdown() error = %v, want timeout", err)
	}
}

func TestClosePersistsCheckpointForLatestWALState(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1 << 30
	cfg.CheckpointInterval = time.Hour

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	result, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
		{EventTimeUnixMs: 2, Payload: []byte("b")},
	})
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if checkpoint.LastBatchSeq != result.BatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, result.BatchSeq)
	}

	log, err := wal.Open(root)
	if err != nil {
		t.Fatalf("Open(wal) error = %v", err)
	}
	defer log.Close()

	entries, err := log.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("len(entries) = 0, want > 0")
	}
	if checkpoint.LastWALEndOffset != entries[len(entries)-1].EndOffset {
		t.Fatalf("LastWALEndOffset = %d, want %d", checkpoint.LastWALEndOffset, entries[len(entries)-1].EndOffset)
	}
}

func TestCloseFailsBeforeCheckpointWhenActiveSegmentCannotBeSynced(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1 << 30
	cfg.CheckpointInterval = time.Hour

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	result, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	})
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
	segment.SetSyncHookForTesting(func(_ *os.File) error { return os.ErrPermission })
	defer segment.SetSyncHookForTesting(nil)

	if err := engine.Close(); err == nil {
		t.Fatalf("Close() error = nil, want sync failure")
	}

	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if checkpoint.LastBatchSeq != 0 {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, 0)
	}
	if checkpoint.LastWALEndOffset != 0 {
		t.Fatalf("LastWALEndOffset = %d, want %d", checkpoint.LastWALEndOffset, 0)
	}

	_ = result
}

func TestCloseCanRetryAfterCheckpointSaveFailure(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1 << 30
	cfg.CheckpointInterval = time.Hour

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	result, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	})
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
	failPath := filepath.Join(root, "meta", "checkpoint.meta.tmp")
	if err := os.MkdirAll(failPath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := engine.Close(); err == nil {
		t.Fatalf("Close() error = nil, want checkpoint failure")
	}
	if err := os.RemoveAll(failPath); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close(retry) error = %v", err)
	}

	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if checkpoint.LastBatchSeq != result.BatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, result.BatchSeq)
	}
	if checkpoint.LastWALEndOffset <= 0 {
		t.Fatalf("LastWALEndOffset = %d, want > 0", checkpoint.LastWALEndOffset)
	}
}

func TestShutdownPersistsCheckpointForLatestWALState(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1 << 30
	cfg.CheckpointInterval = time.Hour

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	result, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	})
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
	if err := engine.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if checkpoint.LastBatchSeq != result.BatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, result.BatchSeq)
	}
	if checkpoint.LastWALEndOffset <= 0 {
		t.Fatalf("LastWALEndOffset = %d, want > 0", checkpoint.LastWALEndOffset)
	}
}

func TestOpenUsesCheckpointToSkipConflictingDurableWALPrefix(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1 << 30
	cfg.CheckpointInterval = time.Hour

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}
	firstResult, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	})
	if err != nil {
		t.Fatalf("WriteBatch(first) error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	checkpointStore := wal.NewCheckpointStore(root)
	checkpoint, err := checkpointStore.Load()
	if err != nil {
		t.Fatalf("Load(checkpoint) error = %v", err)
	}
	if checkpoint.LastBatchSeq != firstResult.BatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, firstResult.BatchSeq)
	}

	log, err := wal.Open(root)
	if err != nil {
		t.Fatalf("Open(wal) error = %v", err)
	}
	conflictingFirstBlock := mustBuildLifecycleBlock(t, 1, []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("z")},
	})
	secondBlock := mustBuildLifecycleBlock(t, 2, []cache.RawRecord{
		{EventTimeUnixMs: 2, Payload: []byte("b")},
	})
	if err := log.TruncateAfter(0); err != nil {
		t.Fatalf("TruncateAfter() error = %v", err)
	}
	firstMeta, err := log.Append(1, conflictingFirstBlock)
	if err != nil {
		t.Fatalf("Append(conflicting first) error = %v", err)
	}
	if _, err := log.Append(2, secondBlock); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close(wal) error = %v", err)
	}
	if firstMeta.EndOffset != checkpoint.LastWALEndOffset {
		t.Fatalf("checkpoint WAL end offset = %d, want %d", checkpoint.LastWALEndOffset, firstMeta.EndOffset)
	}

	engine, err = cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open(reopen) error = %v", err)
	}
	defer engine.Close()

	stats, err := engine.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Capacity.NextWriteSeq != 3 {
		t.Fatalf("NextWriteSeq = %d, want %d", stats.Capacity.NextWriteSeq, 3)
	}

	replayBatch, err := engine.Replay(context.Background(), "checkpoint-test", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if replayBatch.RecordCount != 2 {
		t.Fatalf("RecordCount = %d, want %d", replayBatch.RecordCount, 2)
	}
	if !bytes.Equal(replayBatch.Records[0].Payload, []byte("a")) {
		t.Fatalf("first payload = %q, want %q", replayBatch.Records[0].Payload, []byte("a"))
	}
	if !bytes.Equal(replayBatch.Records[1].Payload, []byte("b")) {
		t.Fatalf("second payload = %q, want %q", replayBatch.Records[1].Payload, []byte("b"))
	}
}

func TestUngracefulRecoveryIncrementsHealthCounter(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}
	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	secondEngine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open(second) error = %v", err)
	}
	defer secondEngine.Close()

	stats, err := secondEngine.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Health.UngracefulShutdownRecoveriesTotal != 1 {
		t.Fatalf("UngracefulShutdownRecoveriesTotal = %d, want %d", stats.Health.UngracefulShutdownRecoveriesTotal, 1)
	}
	_ = engine.Close()
}

func TestOpenRecordsSegmentTailRepairsFromRecovery(t *testing.T) {
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

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	stats, err := engine.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Health.SegmentTailRepairsTotal == 0 {
		t.Fatalf("SegmentTailRepairsTotal = 0, want > 0")
	}
}

func TestRecoverReloadsRuntimeStateAfterRepair(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1 << 30
	cfg.CheckpointInterval = time.Hour

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	}); err != nil {
		t.Fatalf("WriteBatch(first) error = %v", err)
	}

	walPath := filepath.Join(root, "wal", "active.wal")
	if err := os.Truncate(walPath, 0); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}

	if err := engine.Recover(context.Background()); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 2, Payload: []byte("b")},
	}); err != nil {
		t.Fatalf("WriteBatch(second) error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	log, err := wal.Open(root)
	if err != nil {
		t.Fatalf("Open(wal) error = %v", err)
	}
	defer log.Close()

	entries, err := log.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 1)
	}
	if entries[0].Offset != 0 {
		t.Fatalf("Offset = %d, want %d", entries[0].Offset, 0)
	}
	block, err := codec.ParseBlock(entries[0].Block)
	if err != nil {
		t.Fatalf("ParseBlock() error = %v", err)
	}
	if block.FirstWriteSeq != 2 || block.LastWriteSeq != 2 {
		t.Fatalf("write seq bounds = %d..%d, want 2..2", block.FirstWriteSeq, block.LastWriteSeq)
	}
}

func TestOpenReconcilesCheckpointAheadOfRepairedWAL(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)
	cfg.CheckpointBytes = 1 << 30
	cfg.CheckpointInterval = time.Hour

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}
	result, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	})
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load(initial checkpoint) error = %v", err)
	}
	if checkpoint.LastBatchSeq != result.BatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, result.BatchSeq)
	}

	walPath := filepath.Join(root, "wal", "active.wal")
	stat, err := os.Stat(walPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if err := os.Truncate(walPath, stat.Size()-1); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}

	engine, err = cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open(repaired) error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close(repaired) error = %v", err)
	}

	checkpoint, err = wal.NewCheckpointStore(root).Load()
	if err != nil {
		t.Fatalf("Load(reconciled checkpoint) error = %v", err)
	}
	if checkpoint.LastBatchSeq != result.BatchSeq {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, result.BatchSeq)
	}
	if checkpoint.LastWALEndOffset != 0 {
		t.Fatalf("LastWALEndOffset = %d, want %d", checkpoint.LastWALEndOffset, 0)
	}
}

func mustBuildLifecycleBlock(t *testing.T, firstSeq uint64, records []cache.RawRecord) []byte {
	t.Helper()

	encodedRecords := make([][]byte, 0, len(records))
	minEventTime := records[0].EventTimeUnixMs
	maxEventTime := records[0].EventTimeUnixMs
	for idx, record := range records {
		encoded, err := codec.EncodeRecord(firstSeq+uint64(idx), record)
		if err != nil {
			t.Fatalf("EncodeRecord() error = %v", err)
		}
		encodedRecords = append(encodedRecords, encoded)
		if record.EventTimeUnixMs < minEventTime {
			minEventTime = record.EventTimeUnixMs
		}
		if record.EventTimeUnixMs > maxEventTime {
			maxEventTime = record.EventTimeUnixMs
		}
	}

	block, err := codec.BuildBlock(encodedRecords, firstSeq, firstSeq+uint64(len(records))-1, minEventTime, maxEventTime)
	if err != nil {
		t.Fatalf("BuildBlock() error = %v", err)
	}
	return block
}
