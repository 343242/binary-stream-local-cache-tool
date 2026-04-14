package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fastReadFile/internal/codec"
	"fastReadFile/internal/fsutil"
	"fastReadFile/internal/recovery"
	replaystore "fastReadFile/internal/replay"
	"fastReadFile/internal/segment"
	"fastReadFile/internal/stats"
	"fastReadFile/internal/wal"
)

const maxPayloadSizeBytes = 16 << 20

type StorageEngine struct {
	mu                 sync.Mutex
	cfg                Config
	closed             bool
	closing            bool
	batchSeq           uint64
	nextWriteSeq       uint64
	lastCheckpointAt   time.Time
	latestWALEndOffset int64
	checkpointStore    *wal.CheckpointStore
	lastCheckpoint     wal.Checkpoint
	walLog             *wal.Log
	segments           *segment.Manager
	cursorStore        *replaystore.CursorStore
	replayMgr          *replaystore.Manager
	stats              *stats.Collector
}

func Open(cfg Config) (*StorageEngine, error) {
	cfg = withConfigDefaults(cfg)
	collector := stats.New()
	if wasDirty, err := markLifecycleState(cfg.RootDir, lifecycleStateDirty); err != nil {
		return nil, err
	} else if wasDirty {
		collector.RecordUngracefulRecovery()
	}
	state, err := recovery.Recover(context.Background(), cfg.RootDir, cfg)
	if err != nil {
		return nil, err
	}
	walLog, err := wal.Open(cfg.RootDir)
	if err != nil {
		return nil, err
	}
	checkpointStore := wal.NewCheckpointStore(cfg.RootDir)
	checkpoint, err := checkpointStore.Load()
	if err != nil {
		_ = walLog.Close()
		return nil, err
	}
	entries, err := walLog.Scan()
	if err != nil {
		_ = walLog.Close()
		return nil, err
	}
	segments, err := segment.OpenManager(cfg.RootDir, cfg)
	if err != nil {
		_ = walLog.Close()
		return nil, err
	}
	cursorStore := replaystore.NewCursorStore(cfg.RootDir)
	engine := &StorageEngine{
		cfg:              cfg,
		nextWriteSeq:     state.NextWriteSeq,
		lastCheckpointAt: time.Now(),
		checkpointStore:  checkpointStore,
		lastCheckpoint:   checkpoint,
		walLog:           walLog,
		segments:         segments,
		cursorStore:      cursorStore,
		replayMgr:        replaystore.NewManager(cfg.RootDir, cursorStore),
		stats:            collector,
	}
	if len(entries) > 0 {
		engine.batchSeq = entries[len(entries)-1].BatchSeq
		engine.latestWALEndOffset = entries[len(entries)-1].EndOffset
	} else {
		engine.latestWALEndOffset = checkpoint.LastWALEndOffset
	}
	if checkpoint.UpdatedAtUnixMs != 0 {
		engine.lastCheckpointAt = time.UnixMilli(checkpoint.UpdatedAtUnixMs)
	}
	return engine, nil
}

func (s *StorageEngine) WriteBatch(_ context.Context, records []RawRecord) (WriteBatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.closing {
		return WriteBatchResult{}, NewError(ErrClosed, "write_batch", s.cfg.RootDir, "storage engine is closed", nil)
	}
	if err := validateWriteBatch(records); err != nil {
		return WriteBatchResult{}, err
	}
	if err := fsutil.CheckWritable(s.cfg.RootDir); err != nil {
		return WriteBatchResult{}, err
	}

	s.batchSeq++
	firstSeq := s.nextWriteSeq
	lastSeq := firstSeq + uint64(len(records)) - 1
	blocks, err := buildBlocks(firstSeq, records, s.cfg.BlockTargetSizeBytes)
	if err != nil {
		return WriteBatchResult{}, err
	}

	result := WriteBatchResult{
		BatchSeq:      s.batchSeq,
		FirstWriteSeq: firstSeq,
		LastWriteSeq:  lastSeq,
		RecordCount:   len(records),
	}
	rotationTriggered := false
	for _, block := range blocks {
		currentSegmentID := s.segments.CurrentSegmentID()
		walMeta, err := s.walLog.Append(s.batchSeq, block.Bytes)
		if err != nil {
			return WriteBatchResult{}, err
		}
		appendResult, err := s.segments.AppendBlock(block.Bytes, segment.BlockMeta{
			RecordCount:   uint64(len(block.RecordBytes)),
			FirstWriteSeq: block.FirstWriteSeq,
			LastWriteSeq:  block.LastWriteSeq,
			MinEventTime:  block.MinEventTime,
			MaxEventTime:  block.MaxEventTime,
			LastBatchSeq:  s.batchSeq,
		})
		if err != nil {
			return WriteBatchResult{}, err
		}
		result.SegmentID = appendResult.SegmentID
		result.WALBytesWritten += uint64(walMeta.BytesWritten)
		result.SegmentBytesWritten += appendResult.BytesWritten
		s.latestWALEndOffset = walMeta.EndOffset
		if currentSegmentID != 0 && appendResult.SegmentID != currentSegmentID {
			rotationTriggered = true
		}
	}
	s.nextWriteSeq = lastSeq + 1
	s.stats.RecordWrite(len(records))
	if err := s.maybeSaveCheckpoint(time.Now(), rotationTriggered); err != nil {
		return WriteBatchResult{}, err
	}

	return result, nil
}

func (s *StorageEngine) Replay(ctx context.Context, destination string, limit ReplayLimit) (ReplayBatch, error) {
	batch, err := s.replayMgr.Replay(ctx, destination, limit)
	if err != nil {
		return ReplayBatch{}, err
	}
	s.stats.RecordReplay(batch.RecordCount)
	return batch, nil
}

func (s *StorageEngine) Ack(_ context.Context, destination string, cursor ReplayCursor) (AckResult, error) {
	previous, err := s.cursorStore.Load(destination)
	if err != nil {
		return AckResult{}, err
	}
	if err := s.cursorStore.Save(destination, cursor); err != nil {
		return AckResult{}, err
	}
	s.stats.RecordAck(cursor.WriteSeq)
	return AckResult{
		Destination:      destination,
		AppliedCursor:    cursor,
		PreviousWriteSeq: previous.WriteSeq,
		CurrentWriteSeq:  cursor.WriteSeq,
	}, nil
}

func (s *StorageEngine) Recover(ctx context.Context) error {
	state, err := recovery.Recover(ctx, s.cfg.RootDir, s.cfg)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextWriteSeq = state.NextWriteSeq
	return nil
}

func (s *StorageEngine) Stats(_ context.Context) (StatsSnapshot, error) {
	snapshot := s.stats.Snapshot()
	snapshot.Capacity.RetentionDays = s.cfg.RetentionDays
	snapshot.Capacity.NextWriteSeq = s.nextWriteSeq
	snapshot.Capacity.SegmentCount = countSegments(filepath.Join(s.cfg.RootDir, "segments"))
	return snapshot, nil
}

func (s *StorageEngine) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closing = true
	if err := s.saveCheckpoint(time.Now()); err != nil {
		return err
	}
	if s.segments != nil {
		if err := s.segments.Close(); err != nil {
			return err
		}
	}
	if s.walLog != nil {
		if err := s.walLog.Close(); err != nil {
			return err
		}
	}
	if err := writeLifecycleState(s.cfg.RootDir, lifecycleStateClean); err != nil {
		return err
	}
	s.stats.RecordGracefulShutdown()
	s.closed = true
	return nil
}

func (s *StorageEngine) Shutdown(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return NewError(ErrTimeout, "shutdown", s.cfg.RootDir, "shutdown context expired", err)
	}
	return s.Close()
}

func withConfigDefaults(cfg Config) Config {
	defaults := DefaultConfig(cfg.RootDir)
	if cfg.RootDir != "" {
		defaults.RootDir = cfg.RootDir
	}
	if cfg.SegmentTargetSizeBytes != 0 {
		defaults.SegmentTargetSizeBytes = cfg.SegmentTargetSizeBytes
	}
	if cfg.SegmentSlackSizeBytes != 0 {
		defaults.SegmentSlackSizeBytes = cfg.SegmentSlackSizeBytes
	}
	if cfg.BlockTargetSizeBytes != 0 {
		defaults.BlockTargetSizeBytes = cfg.BlockTargetSizeBytes
	}
	if cfg.CheckpointInterval != 0 {
		defaults.CheckpointInterval = cfg.CheckpointInterval
	}
	if cfg.CheckpointBytes != 0 {
		defaults.CheckpointBytes = cfg.CheckpointBytes
	}
	if cfg.SegmentFsyncInterval != 0 {
		defaults.SegmentFsyncInterval = cfg.SegmentFsyncInterval
	}
	if cfg.SegmentFsyncBytes != 0 {
		defaults.SegmentFsyncBytes = cfg.SegmentFsyncBytes
	}
	if cfg.RetentionDays != 0 {
		defaults.RetentionDays = cfg.RetentionDays
	}
	return defaults
}

func validateWriteBatch(records []RawRecord) error {
	if len(records) == 0 {
		return NewError(ErrValidation, "write_batch", "", "empty batch", nil)
	}
	for idx, record := range records {
		switch {
		case record.EventTimeUnixMs < 0:
			return NewError(ErrValidation, "write_batch", "", fmt.Sprintf("record %d has negative event time", idx), nil)
		case len(record.Payload) == 0:
			return NewError(ErrValidation, "write_batch", "", fmt.Sprintf("record %d has empty payload", idx), nil)
		case len(record.Payload) > maxPayloadSizeBytes:
			return NewError(ErrValidation, "write_batch", "", fmt.Sprintf("record %d payload exceeds %d bytes", idx, maxPayloadSizeBytes), nil)
		}
	}
	return nil
}

func (s *StorageEngine) Config() Config {
	return s.cfg
}

const (
	lifecycleStateDirty = "dirty"
	lifecycleStateClean = "clean"
)

func markLifecycleState(rootDir, nextState string) (bool, error) {
	path := lifecyclePath(rootDir)
	wasDirty := false
	if data, err := os.ReadFile(path); err == nil && string(data) == lifecycleStateDirty {
		wasDirty = true
	}
	if err := writeLifecycleState(rootDir, nextState); err != nil {
		return false, err
	}
	return wasDirty, nil
}

func writeLifecycleState(rootDir, state string) error {
	dir := filepath.Join(rootDir, "meta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return NewError(ErrIO, "lifecycle", dir, "create lifecycle directory", err)
	}
	if err := os.WriteFile(lifecyclePath(rootDir), []byte(state), 0o644); err != nil {
		return NewError(ErrIO, "lifecycle", lifecyclePath(rootDir), "write lifecycle state", err)
	}
	return nil
}

func lifecyclePath(rootDir string) string {
	return filepath.Join(rootDir, "meta", "lifecycle.state")
}

func countSegments(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".seg" {
			count++
		}
	}
	return count
}

func (s *StorageEngine) maybeSaveCheckpoint(now time.Time, force bool) error {
	if s.batchSeq == 0 {
		return nil
	}
	if force {
		return s.saveCheckpoint(now)
	}
	if s.cfg.CheckpointBytes > 0 && s.latestWALEndOffset-s.lastCheckpoint.LastWALEndOffset >= s.cfg.CheckpointBytes {
		return s.saveCheckpoint(now)
	}
	if s.cfg.CheckpointInterval > 0 && now.Sub(s.lastCheckpointAt) >= s.cfg.CheckpointInterval {
		return s.saveCheckpoint(now)
	}
	return nil
}

func (s *StorageEngine) saveCheckpoint(now time.Time) error {
	if s.checkpointStore == nil {
		return nil
	}
	if err := s.syncActiveSegment(); err != nil {
		return err
	}
	checkpoint := wal.Checkpoint{
		LastBatchSeq:     s.batchSeq,
		LastWALEndOffset: s.latestWALEndOffset,
		UpdatedAtUnixMs:  now.UnixMilli(),
	}
	if err := s.checkpointStore.Save(checkpoint); err != nil {
		return err
	}
	s.lastCheckpoint = checkpoint
	s.lastCheckpointAt = now
	return nil
}

func (s *StorageEngine) syncActiveSegment() error {
	if s.segments == nil {
		return nil
	}
	segmentID := s.segments.CurrentSegmentID()
	if segmentID == 0 {
		return nil
	}
	path := filepath.Join(s.cfg.RootDir, "segments", fmt.Sprintf("%06d.seg", segmentID))
	file, err := os.Open(path)
	if err != nil {
		return NewError(ErrIO, "sync_active_segment", path, "open active segment for fsync", err)
	}
	defer file.Close()
	if err := file.Sync(); err != nil {
		return NewError(ErrIO, "sync_active_segment", path, "fsync active segment", err)
	}
	return nil
}

type builtBlock struct {
	Bytes         []byte
	RecordBytes   [][]byte
	FirstWriteSeq uint64
	LastWriteSeq  uint64
	MinEventTime  int64
	MaxEventTime  int64
}

func buildBlocks(firstSeq uint64, records []RawRecord, blockTargetBytes int64) ([]builtBlock, error) {
	blocks := make([]builtBlock, 0, 1)
	currentRecords := make([][]byte, 0, len(records))
	var currentSize int64
	var blockFirstSeq uint64
	var blockMinTime int64
	var blockMaxTime int64

	flush := func(lastSeq uint64) error {
		if len(currentRecords) == 0 {
			return nil
		}
		blockBytes, err := codec.BuildBlock(currentRecords, blockFirstSeq, lastSeq, blockMinTime, blockMaxTime)
		if err != nil {
			return err
		}
		blocks = append(blocks, builtBlock{
			Bytes:         blockBytes,
			RecordBytes:   append([][]byte(nil), currentRecords...),
			FirstWriteSeq: blockFirstSeq,
			LastWriteSeq:  lastSeq,
			MinEventTime:  blockMinTime,
			MaxEventTime:  blockMaxTime,
		})
		currentRecords = currentRecords[:0]
		currentSize = 0
		blockFirstSeq = 0
		blockMinTime = 0
		blockMaxTime = 0
		return nil
	}

	for idx, record := range records {
		writeSeq := firstSeq + uint64(idx)
		encoded, err := codec.EncodeRecord(writeSeq, record)
		if err != nil {
			return nil, err
		}
		if len(currentRecords) == 0 {
			blockFirstSeq = writeSeq
			blockMinTime = record.EventTimeUnixMs
			blockMaxTime = record.EventTimeUnixMs
		}
		if blockTargetBytes > 0 && len(currentRecords) > 0 && currentSize+int64(len(encoded)) > blockTargetBytes {
			if err := flush(writeSeq - 1); err != nil {
				return nil, err
			}
			blockFirstSeq = writeSeq
			blockMinTime = record.EventTimeUnixMs
			blockMaxTime = record.EventTimeUnixMs
		}
		currentRecords = append(currentRecords, encoded)
		currentSize += int64(len(encoded))
		if record.EventTimeUnixMs < blockMinTime {
			blockMinTime = record.EventTimeUnixMs
		}
		if record.EventTimeUnixMs > blockMaxTime {
			blockMaxTime = record.EventTimeUnixMs
		}
	}

	if err := flush(firstSeq + uint64(len(records)) - 1); err != nil {
		return nil, err
	}
	return blocks, nil
}
