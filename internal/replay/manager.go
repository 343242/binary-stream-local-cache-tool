package replay

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"fastReadFile/internal/codec"
	cache "fastReadFile/internal/core"
	"fastReadFile/internal/segment"
)

type Manager struct {
	root  string
	store *CursorStore
}

func NewManager(root string, store *CursorStore) *Manager {
	return &Manager{
		root:  root,
		store: store,
	}
}

func (m *Manager) Replay(ctx context.Context, destination string, limit cache.ReplayLimit) (cache.ReplayBatch, error) {
	cursor, err := m.store.Load(destination)
	if err != nil {
		return cache.ReplayBatch{}, err
	}

	segmentPaths, err := listSegmentPaths(filepath.Join(m.root, "segments"))
	if err != nil {
		return cache.ReplayBatch{}, err
	}

	batch := cache.ReplayBatch{Destination: destination}
	for _, path := range segmentPaths {
		if err := ctx.Err(); err != nil {
			return cache.ReplayBatch{}, cache.NewError(cache.ErrTimeout, "replay", path, "context expired", err)
		}

		segmentID, err := segmentIDFromPath(path)
		if err != nil {
			return cache.ReplayBatch{}, cache.NewError(cache.ErrIO, "replay", path, "parse segment id", err)
		}
		if cursor.SegmentID != 0 && segmentID < cursor.SegmentID {
			continue
		}

		data, err := loadReplayableSegmentData(path)
		if err != nil {
			return cache.ReplayBatch{}, err
		}
		offset := 0
		for offset < len(data) {
			blockLen, err := codec.BlockLength(data[offset:])
			if err != nil {
				break
			}
			if cursor.SegmentID == segmentID && uint64(offset) < cursor.BlockOffset {
				offset += blockLen
				continue
			}
			block, err := codec.ParseBlock(data[offset : offset+blockLen])
			if err != nil {
				return cache.ReplayBatch{}, cache.NewError(cache.ErrCorruption, "replay", path, "parse replay block", err)
			}
			for recordIndex, recordBytes := range block.RecordBytes {
				if cursor.SegmentID == segmentID && uint64(offset) == cursor.BlockOffset && uint32(recordIndex) <= cursor.RecordIndex {
					continue
				}
				if limit.MaxRecords > 0 && batch.RecordCount >= limit.MaxRecords {
					return batch, nil
				}
				record, err := codec.DecodeRecord(recordBytes)
				if err != nil {
					return cache.ReplayBatch{}, cache.NewError(cache.ErrCorruption, "replay", path, "decode replay record", err)
				}
				payloadBytes := int64(len(record.Payload))
				if limit.MaxBytes > 0 && batch.RecordCount > 0 && batch.TotalPayloadBytes+payloadBytes > limit.MaxBytes {
					return batch, nil
				}

				batch.Records = append(batch.Records, cache.ReplayRecord{
					EventTimeUnixMs: record.EventTimeUnixMs,
					WriteSeq:        record.WriteSeq,
					Payload:         record.Payload,
				})
				batch.RecordCount++
				batch.TotalPayloadBytes += payloadBytes
				batch.SegmentID = segmentID
				batch.NextCursor = FinalizeCursor(cache.ReplayCursor{
					Version:         1,
					SegmentID:       segmentID,
					BlockOffset:     uint64(offset),
					RecordIndex:     uint32(recordIndex),
					WriteSeq:        record.WriteSeq,
					UpdatedAtUnixMs: time.Now().UnixMilli(),
				})
			}
			offset += blockLen
		}
	}

	return batch, nil
}

func loadReplayableSegmentData(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, cache.NewError(cache.ErrIO, "replay", path, "read segment file", err)
	}
	if footer, err := segment.ReadFooter(path); err == nil {
		return data[:footer.DataEndOffset], nil
	}
	if footer, ok := segment.RecoverFooterWithTail(path, data); ok {
		return data[:footer.DataEndOffset], nil
	}
	return data, nil
}

func listSegmentPaths(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, cache.NewError(cache.ErrIO, "replay", dir, "read segment directory", err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".seg") {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Slice(paths, func(i, j int) bool {
		left, _ := segmentIDFromPath(paths[i])
		right, _ := segmentIDFromPath(paths[j])
		return left < right
	})
	return paths, nil
}

func segmentIDFromPath(path string) (uint64, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strconv.ParseUint(name, 10, 64)
}
