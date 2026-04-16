package wal

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"

	cache "fastReadFile/internal/core"
)

const (
	walMagic      uint32 = 0x57414C31
	walVersion    uint16 = 1
	walHeaderSize        = 24
)

type EntryMeta struct {
	BatchSeq     uint64
	Offset       int64
	EndOffset    int64
	BytesWritten int64
}

type Entry struct {
	BatchSeq  uint64
	Offset    int64
	EndOffset int64
	Block     []byte
}

type Log struct {
	mu   sync.Mutex
	root string
	path string
	file *os.File
}

func Open(root string) (*Log, error) {
	walDir := filepath.Join(root, "wal")
	if err := os.MkdirAll(walDir, 0o755); err != nil {
		return nil, cache.NewError(cache.ErrIO, "open_wal", walDir, "create wal directory", err)
	}
	if err := syncWalDir(walDir); err != nil {
		return nil, err
	}

	path := filepath.Join(walDir, "active.wal")
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, cache.NewError(cache.ErrIO, "open_wal", path, "open wal file", err)
	}
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		_ = file.Close()
		return nil, cache.NewError(cache.ErrIO, "open_wal", path, "seek wal end", err)
	}

	return &Log{
		root: root,
		path: path,
		file: file,
	}, nil
}

func (l *Log) Append(batchSeq uint64, block []byte) (EntryMeta, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	start, err := l.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return EntryMeta{}, cache.NewError(cache.ErrIO, "append_wal", l.path, "read wal offset", err)
	}

	buf := make([]byte, walHeaderSize+len(block))
	binary.LittleEndian.PutUint32(buf[0:4], walMagic)
	binary.LittleEndian.PutUint16(buf[4:6], walVersion)
	binary.LittleEndian.PutUint16(buf[6:8], 0)
	binary.LittleEndian.PutUint64(buf[8:16], batchSeq)
	binary.LittleEndian.PutUint32(buf[16:20], uint32(len(block)))
	copy(buf[walHeaderSize:], block)
	binary.LittleEndian.PutUint32(buf[20:24], walChecksum(buf))

	written, err := l.file.Write(buf)
	if err != nil {
		return EntryMeta{}, cache.NewError(cache.ErrIO, "append_wal", l.path, "write wal entry", err)
	}
	if written != len(buf) {
		return EntryMeta{}, cache.NewError(cache.ErrIO, "append_wal", l.path, "short wal write", nil)
	}
	if err := l.file.Sync(); err != nil {
		return EntryMeta{}, cache.NewError(cache.ErrIO, "append_wal", l.path, "fsync wal", err)
	}

	return EntryMeta{
		BatchSeq:     batchSeq,
		Offset:       start,
		EndOffset:    start + int64(written),
		BytesWritten: int64(written),
	}, nil
}

func (l *Log) Scan() ([]Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.file.Seek(0, io.SeekStart); err != nil {
		return nil, cache.NewError(cache.ErrIO, "scan_wal", l.path, "seek wal start", err)
	}

	data, err := io.ReadAll(l.file)
	if err != nil {
		return nil, cache.NewError(cache.ErrIO, "scan_wal", l.path, "read wal", err)
	}
	if _, err := l.file.Seek(0, io.SeekEnd); err != nil {
		return nil, cache.NewError(cache.ErrIO, "scan_wal", l.path, "restore wal end", err)
	}

	entries := make([]Entry, 0)
	offset := 0
	for offset < len(data) {
		if len(data[offset:]) < walHeaderSize {
			break
		}
		entryLen := walHeaderSize + int(binary.LittleEndian.Uint32(data[offset+16:offset+20]))
		if len(data[offset:]) < entryLen {
			break
		}
		entryBuf := data[offset : offset+entryLen]
		if binary.LittleEndian.Uint32(entryBuf[0:4]) != walMagic {
			break
		}
		if binary.LittleEndian.Uint16(entryBuf[4:6]) != walVersion {
			break
		}
		if binary.LittleEndian.Uint32(entryBuf[20:24]) != walChecksum(entryBuf) {
			break
		}

		entries = append(entries, Entry{
			BatchSeq:  binary.LittleEndian.Uint64(entryBuf[8:16]),
			Offset:    int64(offset),
			EndOffset: int64(offset + entryLen),
			Block:     append([]byte(nil), entryBuf[walHeaderSize:]...),
		})
		offset += entryLen
	}

	return entries, nil
}

func (l *Log) TruncateAfter(offset int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.file.Truncate(offset); err != nil {
		return cache.NewError(cache.ErrIO, "truncate_wal", l.path, "truncate wal", err)
	}
	if err := l.file.Sync(); err != nil {
		return cache.NewError(cache.ErrIO, "truncate_wal", l.path, "fsync truncated wal", err)
	}
	if _, err := l.file.Seek(offset, io.SeekStart); err != nil {
		return cache.NewError(cache.ErrIO, "truncate_wal", l.path, "seek wal offset", err)
	}
	return nil
}

func walChecksum(buf []byte) uint32 {
	hashInput := make([]byte, 0, len(buf)-4)
	hashInput = append(hashInput, buf[:20]...)
	hashInput = append(hashInput, buf[walHeaderSize:]...)
	return crc32.ChecksumIEEE(hashInput)
}

func (l *Log) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.file.Sync(); err != nil {
		return cache.NewError(cache.ErrIO, "sync_wal", l.path, "fsync wal", err)
	}
	return nil
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	if err != nil {
		return cache.NewError(cache.ErrIO, "close_wal", l.path, "close wal file", err)
	}
	return nil
}

func syncWalDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return cache.NewError(cache.ErrIO, "sync_wal_dir", path, "open wal directory", err)
	}
	defer dir.Close()

	if err := dir.Sync(); err != nil {
		return cache.NewError(cache.ErrIO, "sync_wal_dir", path, "fsync wal directory", err)
	}
	return nil
}
