package ops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fastReadFile/internal/codec"
	"fastReadFile/internal/replay"
	"fastReadFile/internal/segment"
	"fastReadFile/internal/wal"
	"fastReadFile/pkg/cache"
)

func Stats(root, format string) (string, error) {
	engine, err := cache.Open(cache.DefaultConfig(root))
	if err != nil {
		return "", err
	}
	defer engine.Close()

	snapshot, err := engine.Stats(nil)
	if err != nil {
		return "", err
	}
	if format == "json" {
		data, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return fmt.Sprintf("WriteBatchesTotal=%d ReplayBatchesTotal=%d", snapshot.IO.WriteBatchesTotal, snapshot.IO.ReplayBatchesTotal), nil
}

func InspectWAL(root string) (string, error) {
	log, err := wal.Open(root)
	if err != nil {
		return "", err
	}
	defer log.Close()

	entries, err := log.Scan()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d entries", len(entries)), nil
}

func InspectCursor(root, destination string) (string, error) {
	store := replay.NewCursorStore(root)
	cursor, err := store.Load(destination)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("WriteSeq=%d SegmentID=%d", cursor.WriteSeq, cursor.SegmentID), nil
}

func CloseCheck(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "meta", "lifecycle.state"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Verify(root string) (string, error) {
	if _, err := InspectWAL(root); err != nil {
		return "", err
	}
	segmentsDir := filepath.Join(root, "segments")
	entries, err := os.ReadDir(segmentsDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".seg") {
			continue
		}
		path := filepath.Join(segmentsDir, entry.Name())
		if _, err := loadSegmentData(path); err != nil {
			return "", err
		}
	}
	return "ok", nil
}

func RepairTail(root string, segmentID uint64) (string, error) {
	path := filepath.Join(root, "segments", formatSegmentID(segmentID)+".seg")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if footer, ok := recoverFooterWithTail(path, data); ok {
		size := int64(footer.DataEndOffset) + segment.FooterSize
		if err := os.Truncate(path, size); err != nil {
			return "", err
		}
		return "repaired", nil
	}
	validEnd := 0
	for validEnd < len(data) {
		blockLen, err := codec.BlockLength(data[validEnd:])
		if err != nil {
			break
		}
		validEnd += blockLen
	}
	if err := os.Truncate(path, int64(validEnd)); err != nil {
		return "", err
	}
	return "repaired", nil
}

func InspectSegment(root string, segmentID uint64) (string, error) {
	path := filepath.Join(root, "segments", formatSegmentID(segmentID)+".seg")
	if footer, err := segment.ReadFooter(path); err == nil {
		return fmt.Sprintf("SegmentID=%d RecordCount=%d LastWriteSeq=%d", footer.SegmentID, footer.RecordCount, footer.LastWriteSeq), nil
	}
	data, err := loadSegmentData(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SegmentID=%d DataBytes=%d", segmentID, len(data)), nil
}

func loadSegmentData(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if footer, err := segment.ReadFooter(path); err == nil {
		return data[:footer.DataEndOffset], nil
	}
	if footer, ok := recoverFooterWithTail(path, data); ok {
		return data[:footer.DataEndOffset], nil
	}
	offset := 0
	for offset < len(data) {
		blockLen, err := codec.BlockLength(data[offset:])
		if err != nil {
			break
		}
		offset += blockLen
	}
	return data[:offset], nil
}

func recoverFooterWithTail(path string, data []byte) (segment.Footer, bool) {
	if len(data) < segment.FooterSize {
		return segment.Footer{}, false
	}
	for start := len(data) - segment.FooterSize; start >= 0; start-- {
		footer, err := segment.DecodeFooter(data[start : start+segment.FooterSize])
		if err != nil {
			continue
		}
		if err := segment.ValidateFooter(path, int64(start+segment.FooterSize), footer); err != nil {
			continue
		}
		return footer, true
	}
	return segment.Footer{}, false
}

func formatSegmentID(id uint64) string {
	text := strconv.FormatUint(id, 10)
	for len(text) < 6 {
		text = "0" + text
	}
	return text
}
