package recovery

import (
	"context"
	"hash/crc32"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"fastReadFile/internal/codec"
	cache "fastReadFile/internal/core"
	"fastReadFile/internal/segment"
	"fastReadFile/internal/wal"
)

type State struct {
	NextWriteSeq            uint64
	ActiveSegmentID         uint64
	SegmentTailRepairsTotal uint64
}

var (
	activeSegmentSyncMu   sync.Mutex
	activeSegmentSyncHook func(*os.File) error
)

func SetActiveSegmentSyncHookForTesting(hook func(*os.File) error) {
	activeSegmentSyncMu.Lock()
	activeSegmentSyncHook = hook
	activeSegmentSyncMu.Unlock()
}

func Recover(ctx context.Context, root string, _ cache.Config) (State, error) {
	if err := ctx.Err(); err != nil {
		return State{}, cache.NewError(cache.ErrTimeout, "recover", root, "context expired before recovery", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "segments"), 0o755); err != nil {
		return State{}, cache.NewError(cache.ErrIO, "recover", root, "create segment directory", err)
	}
	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		return State{}, err
	}

	knownRecords := make(map[uint64]uint32)
	maxWriteSeq := uint64(0)
	maxSegmentID := uint64(0)
	activeSegmentPath := ""
	tailRepairs := uint64(0)

	segmentPaths, err := listSegmentPaths(filepath.Join(root, "segments"))
	if err != nil {
		return State{}, err
	}
	for _, segmentPath := range segmentPaths {
		if err := ctx.Err(); err != nil {
			return State{}, cache.NewError(cache.ErrTimeout, "recover", root, "context expired during segment scan", err)
		}

		segmentID, err := segmentIDFromPath(segmentPath)
		if err != nil {
			return State{}, cache.NewError(cache.ErrIO, "recover", segmentPath, "parse segment id", err)
		}
		if segmentID > maxSegmentID {
			maxSegmentID = segmentID
		}

		footer, footerErr := segment.ReadFooter(segmentPath)
		if footerErr == nil {
			expectedSize := int64(footer.DataEndOffset) + segment.FooterSize
			repaired, err := segmentRepairIfLarger(segmentPath, expectedSize)
			if err != nil {
				return State{}, err
			}
			tailRepairs += repaired
			data, err := os.ReadFile(segmentPath)
			if err != nil {
				return State{}, cache.NewError(cache.ErrIO, "recover", segmentPath, "read sealed segment", err)
			}
			if _, err := absorbBlocks(data[:footer.DataEndOffset], knownRecords, &maxWriteSeq, false); err != nil {
				return State{}, err
			}
			continue
		}
		if recoveredFooter, recovered := recoverFooterWithTail(segmentPath); recovered {
			expectedSize := int64(recoveredFooter.DataEndOffset) + segment.FooterSize
			repaired, err := segmentRepairIfLarger(segmentPath, expectedSize)
			if err != nil {
				return State{}, err
			}
			tailRepairs += repaired
			data, err := os.ReadFile(segmentPath)
			if err != nil {
				return State{}, cache.NewError(cache.ErrIO, "recover", segmentPath, "read recovered-footer segment", err)
			}
			if _, err := absorbBlocks(data[:recoveredFooter.DataEndOffset], knownRecords, &maxWriteSeq, false); err != nil {
				return State{}, err
			}
			continue
		}

		validEnd, repaired, err := absorbSegmentFile(segmentPath, knownRecords, &maxWriteSeq)
		if err != nil {
			return State{}, err
		}
		tailRepairs += repaired
		repaired, err = segmentRepairIfLarger(segmentPath, validEnd)
		if err != nil {
			return State{}, err
		}
		tailRepairs += repaired
		if segmentID >= maxSegmentID {
			activeSegmentPath = segmentPath
		}
	}

	activeSegmentID := maxSegmentID + 1
	if activeSegmentPath == "" {
		if activeSegmentID == 0 {
			activeSegmentID = 1
		}
		activeSegmentPath = filepath.Join(root, "segments", formatSegmentID(activeSegmentID)+".seg")
	} else {
		var err error
		activeSegmentID, err = segmentIDFromPath(activeSegmentPath)
		if err != nil {
			return State{}, cache.NewError(cache.ErrIO, "recover", activeSegmentPath, "parse active segment id", err)
		}
	}

	log, err := wal.Open(root)
	if err != nil {
		return State{}, err
	}
	defer log.Close()

	entries, err := log.Scan()
	if err != nil {
		return State{}, err
	}
	validWALEndOffset := int64(0)
	if len(entries) > 0 {
		validWALEndOffset = entries[len(entries)-1].EndOffset
	}
	replayStartOffset := checkpointReplayStartOffset(checkpoint, validWALEndOffset)

	if err := os.MkdirAll(filepath.Dir(activeSegmentPath), 0o755); err != nil {
		return State{}, cache.NewError(cache.ErrIO, "recover", filepath.Dir(activeSegmentPath), "create active segment directory", err)
	}
	activeFile, err := os.OpenFile(activeSegmentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return State{}, cache.NewError(cache.ErrIO, "recover", activeSegmentPath, "open active segment", err)
	}
	defer activeFile.Close()

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return State{}, cache.NewError(cache.ErrTimeout, "recover", root, "context expired during wal replay", err)
		}
		if entry.EndOffset <= replayStartOffset && entry.BatchSeq <= checkpoint.LastBatchSeq {
			continue
		}

		shouldAppend, err := absorbWALBlock(entry.Block, knownRecords, &maxWriteSeq)
		if err != nil {
			return State{}, err
		}
		if !shouldAppend {
			continue
		}
		if _, err := activeFile.Write(entry.Block); err != nil {
			return State{}, cache.NewError(cache.ErrIO, "recover", activeSegmentPath, "append recovered wal block", err)
		}
	}
	if err := syncActiveFile(activeFile); err != nil {
		return State{}, cache.NewError(cache.ErrIO, "recover", activeSegmentPath, "fsync active segment", err)
	}
	if err := log.TruncateAfter(validWALEndOffset); err != nil {
		return State{}, err
	}

	return State{
		NextWriteSeq:            maxWriteSeq + 1,
		ActiveSegmentID:         activeSegmentID,
		SegmentTailRepairsTotal: tailRepairs,
	}, nil
}

func checkpointReplayStartOffset(checkpoint wal.Checkpoint, validWALEndOffset int64) int64 {
	if checkpoint.LastWALEndOffset <= 0 {
		return 0
	}
	if checkpoint.LastWALEndOffset > validWALEndOffset {
		return 0
	}
	return checkpoint.LastWALEndOffset
}

func recoverFooterWithTail(path string) (segment.Footer, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < segment.FooterSize {
		return segment.Footer{}, false
	}
	return segment.RecoverFooterWithTail(path, data)
}

func absorbSegmentFile(path string, knownRecords map[uint64]uint32, maxWriteSeq *uint64) (int64, uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, cache.NewError(cache.ErrIO, "recover", path, "read active segment", err)
	}
	validEnd, err := absorbBlocks(data, knownRecords, maxWriteSeq, true)
	if err != nil {
		return 0, 0, err
	}
	if validEnd < uint64(len(data)) {
		return int64(validEnd), 1, nil
	}
	return int64(validEnd), 0, nil
}

func absorbBlocks(data []byte, knownRecords map[uint64]uint32, maxWriteSeq *uint64, stopOnTailCorruption bool) (uint64, error) {
	offset := 0
	for offset < len(data) {
		blockLen, err := codec.BlockLength(data[offset:])
		if err != nil {
			if stopOnTailCorruption {
				return uint64(offset), nil
			}
			return 0, cache.NewError(cache.ErrCorruption, "recover", "", "corrupt block in segment", err)
		}
		if _, err := absorbDecodedBlock(data[offset:offset+blockLen], knownRecords, maxWriteSeq); err != nil {
			return 0, err
		}
		offset += blockLen
	}
	return uint64(offset), nil
}

func absorbWALBlock(block []byte, knownRecords map[uint64]uint32, maxWriteSeq *uint64) (bool, error) {
	duplicate, err := absorbDecodedBlock(block, knownRecords, maxWriteSeq)
	if err != nil {
		return false, err
	}
	return !duplicate, nil
}

func absorbDecodedBlock(block []byte, knownRecords map[uint64]uint32, maxWriteSeq *uint64) (bool, error) {
	decoded, err := codec.ParseBlock(block)
	if err != nil {
		return false, cache.NewError(cache.ErrWALInvalid, "recover", "", "parse wal block", err)
	}

	missing := 0
	seen := 0
	fingerprints := make([]uint32, 0, len(decoded.RecordBytes))
	seqs := make([]uint64, 0, len(decoded.RecordBytes))
	for _, recordBytes := range decoded.RecordBytes {
		record, err := codec.DecodeRecord(recordBytes)
		if err != nil {
			return false, cache.NewError(cache.ErrCorruption, "recover", "", "decode record during recovery", err)
		}
		fingerprint := crc32.ChecksumIEEE(recordBytes)
		if existing, ok := knownRecords[record.WriteSeq]; ok {
			if existing != fingerprint {
				return false, cache.NewError(cache.ErrSequenceConflict, "recover", "", "write sequence content mismatch", nil)
			}
			seen++
		} else {
			missing++
		}
		if record.WriteSeq > *maxWriteSeq {
			*maxWriteSeq = record.WriteSeq
		}
		fingerprints = append(fingerprints, fingerprint)
		seqs = append(seqs, record.WriteSeq)
	}
	if seen > 0 && missing > 0 {
		return false, cache.NewError(cache.ErrSequenceConflict, "recover", "", "partial block overlap detected", nil)
	}
	if missing == 0 {
		return true, nil
	}
	for idx, seq := range seqs {
		knownRecords[seq] = fingerprints[idx]
	}
	return false, nil
}

func RepairSegmentTail(_ string, path string) (segment.TailRepairResult, error) {
	return segment.RepairTail(path)
}

func syncActiveFile(file *os.File) error {
	activeSegmentSyncMu.Lock()
	hook := activeSegmentSyncHook
	activeSegmentSyncMu.Unlock()
	if hook != nil {
		return hook(file)
	}
	return file.Sync()
}

func segmentRepairIfLarger(path string, size int64) (uint64, error) {
	repaired, err := segment.TruncateIfLarger(path, size)
	if err != nil {
		return 0, err
	}
	if repaired {
		return 1, nil
	}
	return 0, nil
}

func listSegmentPaths(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, cache.NewError(cache.ErrIO, "recover", dir, "read segment directory", err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".seg") {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Slice(paths, func(i, j int) bool {
		leftID, _ := segmentIDFromPath(paths[i])
		rightID, _ := segmentIDFromPath(paths[j])
		return leftID < rightID
	})
	return paths, nil
}

func segmentIDFromPath(path string) (uint64, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strconv.ParseUint(name, 10, 64)
}

func formatSegmentID(id uint64) string {
	text := strconv.FormatUint(id, 10)
	for len(text) < 6 {
		text = "0" + text
	}
	return text
}
