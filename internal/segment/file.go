package segment

import (
	"io"
	"os"
	"time"

	cache "fastReadFile/internal/core"
)

type BlockMeta struct {
	RecordCount   uint64
	FirstWriteSeq uint64
	LastWriteSeq  uint64
	MinEventTime  int64
	MaxEventTime  int64
	LastBatchSeq  uint64
}

type AppendResult struct {
	SegmentID    uint64
	BlockOffset  uint64
	BytesWritten uint64
	Synced       bool
}

type recoveredSegmentState struct {
	recordCount   uint64
	firstWriteSeq uint64
	lastWriteSeq  uint64
	minEventTime  int64
	maxEventTime  int64
	lastBatchSeq  uint64
	tailRepairs   uint64
}

type SegmentFile struct {
	cfg            cache.Config
	id             uint64
	path           string
	file           *os.File
	createdAtMs    int64
	size           int64
	bytesSinceSync int64
	lastSyncAt     time.Time
	recordCount    uint64
	firstWriteSeq  uint64
	lastWriteSeq   uint64
	minEventTime   int64
	maxEventTime   int64
	lastBatchSeq   uint64
}

var segmentSyncHook func(*os.File) error

func SetSyncHookForTesting(hook func(*os.File) error) {
	segmentSyncHook = hook
}

func openSegmentFile(path string, segmentID uint64, cfg cache.Config) (*SegmentFile, error) {
	return openSegmentFileWithState(path, segmentID, cfg, nil)
}

func openSegmentFileWithState(path string, segmentID uint64, cfg cache.Config, recovered *recoveredSegmentState) (*SegmentFile, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, cache.NewError(cache.ErrIO, "open_segment", path, "open segment file", err)
	}
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, cache.NewError(cache.ErrIO, "open_segment", path, "stat segment file", err)
	}
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		_ = file.Close()
		return nil, cache.NewError(cache.ErrIO, "open_segment", path, "seek segment end", err)
	}
	now := time.Now()
	segmentFile := &SegmentFile{
		cfg:            cfg,
		id:             segmentID,
		path:           path,
		file:           file,
		createdAtMs:    now.UnixMilli(),
		size:           stat.Size(),
		lastSyncAt:     now,
		minEventTime:   0,
		maxEventTime:   0,
		bytesSinceSync: 0,
	}
	if recovered != nil {
		segmentFile.recordCount = recovered.recordCount
		segmentFile.firstWriteSeq = recovered.firstWriteSeq
		segmentFile.lastWriteSeq = recovered.lastWriteSeq
		segmentFile.minEventTime = recovered.minEventTime
		segmentFile.maxEventTime = recovered.maxEventTime
		segmentFile.lastBatchSeq = recovered.lastBatchSeq
	}
	return segmentFile, nil
}

func (s *SegmentFile) AppendBlock(block []byte, meta BlockMeta) (AppendResult, error) {
	offset := s.size
	written, err := s.file.Write(block)
	if err != nil {
		return AppendResult{}, cache.NewError(cache.ErrIO, "append_segment", s.path, "write segment block", err)
	}
	if written != len(block) {
		return AppendResult{}, cache.NewError(cache.ErrIO, "append_segment", s.path, "short segment write", nil)
	}

	s.size += int64(written)
	s.bytesSinceSync += int64(written)
	s.recordCount += meta.RecordCount
	if s.firstWriteSeq == 0 {
		s.firstWriteSeq = meta.FirstWriteSeq
		s.minEventTime = meta.MinEventTime
		s.maxEventTime = meta.MaxEventTime
	} else if meta.MinEventTime < s.minEventTime {
		s.minEventTime = meta.MinEventTime
	}
	if s.firstWriteSeq != 0 && meta.MaxEventTime > s.maxEventTime {
		s.maxEventTime = meta.MaxEventTime
	}
	s.lastWriteSeq = meta.LastWriteSeq
	s.lastBatchSeq = meta.LastBatchSeq

	synced := false
	if s.shouldSync() {
		if err := syncSegmentFile(s.file); err != nil {
			return AppendResult{}, cache.NewError(cache.ErrIO, "append_segment", s.path, "fsync segment", err)
		}
		s.bytesSinceSync = 0
		s.lastSyncAt = time.Now()
		synced = true
	}

	return AppendResult{
		SegmentID:    s.id,
		BlockOffset:  uint64(offset),
		BytesWritten: uint64(written),
		Synced:       synced,
	}, nil
}

func (s *SegmentFile) Seal() (Footer, error) {
	footer := Footer{
		SegmentID:       s.id,
		CreatedAtUnixMs: s.createdAtMs,
		SealedAtUnixMs:  time.Now().UnixMilli(),
		DataEndOffset:   uint64(s.size),
		RecordCount:     s.recordCount,
		FirstWriteSeq:   s.firstWriteSeq,
		LastWriteSeq:    s.lastWriteSeq,
		MinEventTime:    s.minEventTime,
		MaxEventTime:    s.maxEventTime,
		LastBatchSeq:    s.lastBatchSeq,
	}
	encoded, err := EncodeFooter(footer)
	if err != nil {
		return Footer{}, err
	}
	originalSize := s.size
	if _, err := s.file.Write(encoded); err != nil {
		return Footer{}, cache.NewError(cache.ErrIO, "seal_segment", s.path, "write segment footer", err)
	}
	if err := syncSegmentFile(s.file); err != nil {
		_ = s.file.Truncate(originalSize)
		_, _ = s.file.Seek(originalSize, io.SeekStart)
		return Footer{}, cache.NewError(cache.ErrIO, "seal_segment", s.path, "fsync sealed segment", err)
	}
	s.size += int64(len(encoded))
	return footer, nil
}

func (s *SegmentFile) Close() error {
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	if err != nil {
		return cache.NewError(cache.ErrIO, "close_segment", s.path, "close segment file", err)
	}
	return nil
}

func (s *SegmentFile) Sync() error {
	if s.file == nil {
		return nil
	}
	if err := syncSegmentFile(s.file); err != nil {
		return cache.NewError(cache.ErrIO, "sync_segment", s.path, "fsync active segment", err)
	}
	s.bytesSinceSync = 0
	s.lastSyncAt = time.Now()
	return nil
}

func (s *SegmentFile) shouldSync() bool {
	if s.cfg.SegmentFsyncBytes > 0 && s.bytesSinceSync >= s.cfg.SegmentFsyncBytes {
		return true
	}
	if s.cfg.SegmentFsyncInterval > 0 && time.Since(s.lastSyncAt) >= s.cfg.SegmentFsyncInterval {
		return true
	}
	return false
}

func syncSegmentFile(file *os.File) error {
	if segmentSyncHook != nil {
		if err := segmentSyncHook(file); err != nil {
			return err
		}
	}
	return file.Sync()
}
