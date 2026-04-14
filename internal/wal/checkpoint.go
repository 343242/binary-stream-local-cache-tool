package wal

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"

	cache "fastReadFile/internal/core"
)

const checkpointVersion uint32 = 1

type Checkpoint struct {
	Version          uint32
	LastBatchSeq     uint64
	LastWALEndOffset int64
	UpdatedAtUnixMs  int64
}

type CheckpointStore struct {
	path string
}

func NewCheckpointStore(root string) *CheckpointStore {
	return &CheckpointStore{
		path: filepath.Join(root, "meta", "checkpoint.meta"),
	}
}

func (s *CheckpointStore) Load() (Checkpoint, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Checkpoint{}, nil
		}
		return Checkpoint{}, cache.NewError(cache.ErrIO, "load_checkpoint", s.path, "read checkpoint", err)
	}
	if len(data) < 32 {
		return Checkpoint{}, cache.NewError(cache.ErrCorruption, "load_checkpoint", s.path, "checkpoint too short", nil)
	}
	if binary.LittleEndian.Uint32(data[28:32]) != crc32.ChecksumIEEE(data[:28]) {
		return Checkpoint{}, cache.NewError(cache.ErrCorruption, "load_checkpoint", s.path, "checkpoint crc mismatch", nil)
	}

	return Checkpoint{
		Version:          binary.LittleEndian.Uint32(data[0:4]),
		LastBatchSeq:     binary.LittleEndian.Uint64(data[4:12]),
		LastWALEndOffset: int64(binary.LittleEndian.Uint64(data[12:20])),
		UpdatedAtUnixMs:  int64(binary.LittleEndian.Uint64(data[20:28])),
	}, nil
}

func (s *CheckpointStore) Save(checkpoint Checkpoint) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return cache.NewError(cache.ErrIO, "save_checkpoint", filepath.Dir(s.path), "create checkpoint directory", err)
	}

	checkpoint.Version = checkpointVersion
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint32(buf[0:4], checkpoint.Version)
	binary.LittleEndian.PutUint64(buf[4:12], checkpoint.LastBatchSeq)
	binary.LittleEndian.PutUint64(buf[12:20], uint64(checkpoint.LastWALEndOffset))
	binary.LittleEndian.PutUint64(buf[20:28], uint64(checkpoint.UpdatedAtUnixMs))
	binary.LittleEndian.PutUint32(buf[28:32], crc32.ChecksumIEEE(buf[:28]))

	tmpPath := s.path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return cache.NewError(cache.ErrIO, "save_checkpoint", tmpPath, "open checkpoint temp file", err)
	}
	if _, err := file.Write(buf); err != nil {
		_ = file.Close()
		return cache.NewError(cache.ErrIO, "save_checkpoint", tmpPath, "write checkpoint temp file", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return cache.NewError(cache.ErrIO, "save_checkpoint", tmpPath, "fsync checkpoint temp file", err)
	}
	if err := file.Close(); err != nil {
		return cache.NewError(cache.ErrIO, "save_checkpoint", tmpPath, "close checkpoint temp file", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return cache.NewError(cache.ErrIO, "save_checkpoint", s.path, "replace checkpoint file", err)
	}
	if err := syncDir(filepath.Dir(s.path)); err != nil {
		return err
	}
	return nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return cache.NewError(cache.ErrIO, "sync_checkpoint_dir", path, "open checkpoint directory", err)
	}
	defer dir.Close()

	if err := dir.Sync(); err != nil && err != io.EOF {
		return cache.NewError(cache.ErrIO, "sync_checkpoint_dir", path, "fsync checkpoint directory", err)
	}
	return nil
}
