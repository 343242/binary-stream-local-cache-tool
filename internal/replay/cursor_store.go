package replay

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"

	cache "fastReadFile/internal/core"
)

const cursorSize = 44

type CursorStore struct {
	root string
}

func NewCursorStore(root string) *CursorStore {
	return &CursorStore{root: root}
}

func (s *CursorStore) Load(destination string) (cache.ReplayCursor, error) {
	mainPath, backupPath := s.paths(destination)
	main, mainErr := readCursorFile(mainPath)
	if mainErr == nil {
		return main, nil
	}

	backup, backupErr := readCursorFile(backupPath)
	if backupErr == nil {
		return backup, nil
	}

	if os.IsNotExist(mainErr) && os.IsNotExist(backupErr) {
		return cache.ReplayCursor{}, nil
	}
	return cache.ReplayCursor{}, cache.NewError(cache.ErrCursorCorrupted, "load_cursor", mainPath, "cursor files are corrupted", mainErr)
}

func (s *CursorStore) Save(destination string, cursor cache.ReplayCursor) error {
	mainPath, backupPath := s.paths(destination)
	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		return cache.NewError(cache.ErrIO, "save_cursor", filepath.Dir(mainPath), "create cursor directory", err)
	}

	current, err := s.Load(destination)
	if err != nil && !os.IsNotExist(err) && !isCursorAbsent(current, err) {
		if !isCursorCorrupted(err) {
			return err
		}
	}
	switch {
	case current.WriteSeq > cursor.WriteSeq:
		return cache.NewError(cache.ErrCursorInvalid, "save_cursor", mainPath, "cursor moved backwards", nil)
	case current.WriteSeq == cursor.WriteSeq && current.WriteSeq != 0:
		return nil
	}

	encoded := encodeCursor(cursor)
	if err := atomicWrite(mainPath, encoded); err != nil {
		return err
	}
	if err := atomicWrite(backupPath, encoded); err != nil {
		return err
	}
	return nil
}

func (s *CursorStore) paths(destination string) (string, string) {
	mainPath := filepath.Join(s.root, "meta", "replay", destination+".cursor")
	return mainPath, mainPath + ".bak"
}

func readCursorFile(path string) (cache.ReplayCursor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cache.ReplayCursor{}, err
	}
	if len(data) != cursorSize {
		return cache.ReplayCursor{}, cache.NewError(cache.ErrCursorCorrupted, "read_cursor", path, "cursor length invalid", nil)
	}
	if binary.LittleEndian.Uint32(data[40:44]) != crc32.ChecksumIEEE(data[:40]) {
		return cache.ReplayCursor{}, cache.NewError(cache.ErrCursorCorrupted, "read_cursor", path, "cursor crc mismatch", nil)
	}
	return cache.ReplayCursor{
		Version:         binary.LittleEndian.Uint32(data[0:4]),
		SegmentID:       binary.LittleEndian.Uint64(data[4:12]),
		BlockOffset:     binary.LittleEndian.Uint64(data[12:20]),
		RecordIndex:     binary.LittleEndian.Uint32(data[20:24]),
		WriteSeq:        binary.LittleEndian.Uint64(data[24:32]),
		UpdatedAtUnixMs: int64(binary.LittleEndian.Uint64(data[32:40])),
		CRC32:           binary.LittleEndian.Uint32(data[40:44]),
	}, nil
}

func encodeCursor(cursor cache.ReplayCursor) []byte {
	buf := make([]byte, cursorSize)
	binary.LittleEndian.PutUint32(buf[0:4], cursor.Version)
	binary.LittleEndian.PutUint64(buf[4:12], cursor.SegmentID)
	binary.LittleEndian.PutUint64(buf[12:20], cursor.BlockOffset)
	binary.LittleEndian.PutUint32(buf[20:24], cursor.RecordIndex)
	binary.LittleEndian.PutUint64(buf[24:32], cursor.WriteSeq)
	binary.LittleEndian.PutUint64(buf[32:40], uint64(cursor.UpdatedAtUnixMs))
	binary.LittleEndian.PutUint32(buf[40:44], crc32.ChecksumIEEE(buf[:40]))
	return buf
}

func atomicWrite(path string, data []byte) error {
	tmpPath := path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return cache.NewError(cache.ErrIO, "write_cursor", tmpPath, "open cursor temp file", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return cache.NewError(cache.ErrIO, "write_cursor", tmpPath, "write cursor temp file", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return cache.NewError(cache.ErrIO, "write_cursor", tmpPath, "fsync cursor temp file", err)
	}
	if err := file.Close(); err != nil {
		return cache.NewError(cache.ErrIO, "write_cursor", tmpPath, "close cursor temp file", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return cache.NewError(cache.ErrIO, "write_cursor", path, "replace cursor file", err)
	}

	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return cache.NewError(cache.ErrIO, "write_cursor", filepath.Dir(path), "open cursor directory", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return cache.NewError(cache.ErrIO, "write_cursor", filepath.Dir(path), "fsync cursor directory", err)
	}
	return nil
}

func isCursorAbsent(cursor cache.ReplayCursor, err error) bool {
	return cursor == (cache.ReplayCursor{}) && err == nil
}

func isCursorCorrupted(err error) bool {
	return errors.Is(err, cache.ErrCode(cache.ErrCursorCorrupted))
}
