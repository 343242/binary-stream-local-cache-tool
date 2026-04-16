package service

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fastReadFile/internal/codec"
	core "fastReadFile/internal/core"
	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/replay"
	"fastReadFile/internal/segment"
	"fastReadFile/internal/wal"
)

const rawPreviewLimitBytes = 64 << 10

func ListSegments(root string, page, pageSize int) (viewmodel.PagedSegments, error) {
	items, err := loadSegmentList(root)
	if err != nil {
		return viewmodel.PagedSegments{}, wrapServiceError("list_segments", root, err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 200
	}

	start := (page - 1) * pageSize
	if start >= len(items) {
		return viewmodel.PagedSegments{
			Items:      []viewmodel.SegmentListItem{},
			Page:       page,
			PageSize:   pageSize,
			TotalItems: len(items),
			HasNext:    false,
		}, nil
	}

	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	return viewmodel.PagedSegments{
		Items:      items[start:end],
		Page:       page,
		PageSize:   pageSize,
		TotalItems: len(items),
		HasNext:    end < len(items),
	}, nil
}

func GetSegmentDetail(root string, segmentID uint64) (viewmodel.SegmentDetail, error) {
	path := filepath.Join(root, "segments", formatSegmentID(segmentID)+".seg")
	data, err := os.ReadFile(path)
	if err != nil {
		return viewmodel.SegmentDetail{}, wrapServiceError("get_segment_detail", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return viewmodel.SegmentDetail{}, wrapServiceError("get_segment_detail", path, err)
	}

	detail := viewmodel.SegmentDetail{
		SegmentID: segmentID,
		Path:      path,
		SizeBytes: info.Size(),
	}

	switch footer, err := segment.ReadFooter(path); {
	case err == nil:
		blockSummary, summaryErr := scanValidBlocks(data[:footer.DataEndOffset])
		if summaryErr != nil {
			return viewmodel.SegmentDetail{}, wrapServiceError("get_segment_detail", path, summaryErr)
		}
		detail.Sealed = true
		detail.FooterStatus = "valid"
		detail.TailStatus = "clean"
		detail.FirstWriteSeq = footer.FirstWriteSeq
		detail.LastWriteSeq = footer.LastWriteSeq
		detail.RecordCount = footer.RecordCount
		detail.BlockCount = blockSummary.BlockCount
		detail.MinEventTime = footer.MinEventTime
		detail.MaxEventTime = footer.MaxEventTime
		detail.LastBatchSeq = footer.LastBatchSeq
		detail.RawPreviewHex = previewHex(data[:footer.DataEndOffset])
		detail.StructuredPreview = buildSegmentStructuredPreview(detail)
	default:
		footer, ok := recoverFooterWithTail(path, data)
		if ok {
			blockSummary, summaryErr := scanValidBlocks(data[:footer.DataEndOffset])
			if summaryErr != nil {
				return viewmodel.SegmentDetail{}, wrapServiceError("get_segment_detail", path, summaryErr)
			}
			detail.Sealed = false
			detail.FooterStatus = "recovered"
			detail.TailStatus = "repairable"
			detail.FirstWriteSeq = footer.FirstWriteSeq
			detail.LastWriteSeq = footer.LastWriteSeq
			detail.RecordCount = footer.RecordCount
			detail.BlockCount = blockSummary.BlockCount
			detail.MinEventTime = footer.MinEventTime
			detail.MaxEventTime = footer.MaxEventTime
			detail.LastBatchSeq = footer.LastBatchSeq
			detail.RawPreviewHex = previewHex(data[:footer.DataEndOffset])
			detail.StructuredPreview = buildSegmentStructuredPreview(detail)
			return detail, nil
		}

		blockSummary, summaryErr := scanValidBlocks(data)
		if summaryErr != nil && blockSummary.BlockCount == 0 {
			return viewmodel.SegmentDetail{}, wrapServiceError("get_segment_detail", path, summaryErr)
		}
		detail.Sealed = false
		detail.FooterStatus = "missing"
		detail.TailStatus = "clean"
		if blockSummary.ValidEnd < len(data) {
			detail.TailStatus = "truncated"
		}
		detail.FirstWriteSeq = blockSummary.FirstWriteSeq
		detail.LastWriteSeq = blockSummary.LastWriteSeq
		detail.RecordCount = blockSummary.RecordCount
		detail.BlockCount = blockSummary.BlockCount
		detail.MinEventTime = blockSummary.MinEventTime
		detail.MaxEventTime = blockSummary.MaxEventTime
		detail.LastBatchSeq = 0
		detail.RawPreviewHex = previewHex(data[:blockSummary.ValidEnd])
		detail.StructuredPreview = buildSegmentStructuredPreview(detail)
	}

	return detail, nil
}

func GetWALDetail(root string) (viewmodel.WALDetail, error) {
	path := filepath.Join(root, "wal", "active.wal")
	entries, err := wal.ScanReadOnly(root)
	if err != nil {
		return viewmodel.WALDetail{}, wrapServiceError("get_wal_detail", path, err)
	}

	detail := viewmodel.WALDetail{
		Path:      path,
		SizeBytes: fileSizeOrZero(path),
		Health:    "missing",
	}
	if len(entries) == 0 {
		if detail.SizeBytes > 0 {
			detail.Health = "ok"
		}
		return detail, nil
	}

	detail.FirstBatchSeq = entries[0].BatchSeq
	detail.LastBatchSeq = entries[len(entries)-1].BatchSeq
	detail.LastEndOffset = entries[len(entries)-1].EndOffset
	detail.Health = "ok"
	return detail, nil
}

func ListCursors(root string) ([]viewmodel.CursorSummary, error) {
	dir := filepath.Join(root, "meta", "replay")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []viewmodel.CursorSummary{}, nil
		}
		return nil, wrapServiceError("list_cursors", dir, err)
	}

	store := replay.NewCursorStore(root)
	summaries := make([]viewmodel.CursorSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cursor") {
			continue
		}

		destination := strings.TrimSuffix(entry.Name(), ".cursor")
		cursor, err := store.Load(destination)
		if err != nil {
			if errors.Is(err, core.ErrCode(core.ErrCursorCorrupted)) {
				summaries = append(summaries, viewmodel.CursorSummary{
					Destination: destination,
					Status:      "corrupted",
				})
				continue
			}
			return nil, wrapServiceError("list_cursors", filepath.Join(dir, entry.Name()), err)
		}

		summaries = append(summaries, viewmodel.CursorSummary{
			Destination: destination,
			WriteSeq:    cursor.WriteSeq,
			UpdatedAt:   cursor.UpdatedAtUnixMs,
			Status:      "ok",
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Destination < summaries[j].Destination
	})
	return summaries, nil
}

func GetCursorDetail(root, destination string) (viewmodel.CursorDetail, error) {
	cursor, err := replay.NewCursorStore(root).Load(destination)
	if err != nil {
		return viewmodel.CursorDetail{}, wrapServiceError("get_cursor_detail", filepath.Join(root, "meta", "replay", destination+".cursor"), err)
	}

	backupStatus := "absent"
	backupPath := filepath.Join(root, "meta", "replay", destination+".cursor.bak")
	if _, err := os.Stat(backupPath); err == nil {
		backupStatus = "present"
	}

	crcStatus := "valid"
	if err := replay.ValidateCursor(cursor); err != nil {
		crcStatus = "invalid"
	}

	return viewmodel.CursorDetail{
		Destination:  destination,
		SegmentID:    cursor.SegmentID,
		BlockOffset:  cursor.BlockOffset,
		RecordIndex:  cursor.RecordIndex,
		WriteSeq:     cursor.WriteSeq,
		UpdatedAt:    cursor.UpdatedAtUnixMs,
		CRCStatus:    crcStatus,
		BackupStatus: backupStatus,
	}, nil
}

func GetCheckpointDetail(root string) (viewmodel.CheckpointDetail, error) {
	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		return viewmodel.CheckpointDetail{}, wrapServiceError("get_checkpoint_detail", filepath.Join(root, "meta", "checkpoint.meta"), err)
	}

	detail := viewmodel.CheckpointDetail{
		LastBatchSeq:     checkpoint.LastBatchSeq,
		LastWALEndOffset: checkpoint.LastWALEndOffset,
		UpdatedAt:        checkpoint.UpdatedAtUnixMs,
		Version:          checkpoint.Version,
		IntegrityStatus:  "missing",
	}
	if checkpoint != (wal.Checkpoint{}) {
		detail.IntegrityStatus = "ok"
	}
	return detail, nil
}

func loadSegmentList(root string) ([]viewmodel.SegmentListItem, error) {
	dir := filepath.Join(root, "segments")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []viewmodel.SegmentListItem{}, nil
		}
		return nil, err
	}

	items := make([]viewmodel.SegmentListItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".seg" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		item := viewmodel.SegmentListItem{
			FileName:  entry.Name(),
			SizeBytes: info.Size(),
			Health:    "open",
		}
		item.SegmentID, _ = strconv.ParseUint(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), 10, 64)

		if footer, err := segment.ReadFooter(path); err == nil {
			item.Sealed = true
			item.FirstWriteSeq = footer.FirstWriteSeq
			item.LastWriteSeq = footer.LastWriteSeq
			item.RecordCount = footer.RecordCount
			item.MinEventTime = footer.MinEventTime
			item.MaxEventTime = footer.MaxEventTime
			item.LastBatchSeq = footer.LastBatchSeq
			item.Health = "sealed"
		} else if summary, summaryErr := scanValidBlocksFromFile(path); summaryErr == nil {
			item.FirstWriteSeq = summary.FirstWriteSeq
			item.LastWriteSeq = summary.LastWriteSeq
			item.RecordCount = summary.RecordCount
			item.MinEventTime = summary.MinEventTime
			item.MaxEventTime = summary.MaxEventTime
			item.Health = "open"
		}

		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].SegmentID > items[j].SegmentID
	})
	return items, nil
}

type blockSummary struct {
	BlockCount    uint64
	RecordCount   uint64
	FirstWriteSeq uint64
	LastWriteSeq  uint64
	MinEventTime  int64
	MaxEventTime  int64
	ValidEnd      int
}

func scanValidBlocks(data []byte) (blockSummary, error) {
	summary := blockSummary{}
	offset := 0
	for offset < len(data) {
		blockLen, err := codec.BlockLength(data[offset:])
		if err != nil {
			if summary.BlockCount == 0 {
				return summary, err
			}
			return summary, nil
		}
		decoded, err := codec.ParseBlock(data[offset : offset+blockLen])
		if err != nil {
			if summary.BlockCount == 0 {
				return summary, err
			}
			return summary, nil
		}

		if summary.BlockCount == 0 {
			summary.FirstWriteSeq = decoded.FirstWriteSeq
			summary.MinEventTime = decoded.MinEventTime
			summary.MaxEventTime = decoded.MaxEventTime
		}
		summary.BlockCount++
		summary.RecordCount += uint64(decoded.RecordCount)
		summary.LastWriteSeq = decoded.LastWriteSeq
		if decoded.MinEventTime < summary.MinEventTime {
			summary.MinEventTime = decoded.MinEventTime
		}
		if decoded.MaxEventTime > summary.MaxEventTime {
			summary.MaxEventTime = decoded.MaxEventTime
		}
		offset += blockLen
		summary.ValidEnd = offset
	}
	return summary, nil
}

func scanValidBlocksFromFile(path string) (blockSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return blockSummary{}, err
	}
	return scanValidBlocks(data)
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

func previewHex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) > rawPreviewLimitBytes {
		data = data[:rawPreviewLimitBytes]
	}
	return hex.EncodeToString(data)
}

func formatSegmentID(id uint64) string {
	text := strconv.FormatUint(id, 10)
	for len(text) < 6 {
		text = "0" + text
	}
	return text
}

func buildSegmentStructuredPreview(detail viewmodel.SegmentDetail) []viewmodel.KeyValue {
	return []viewmodel.KeyValue{
		{Key: "segmentID", Value: strconv.FormatUint(detail.SegmentID, 10)},
		{Key: "recordCount", Value: strconv.FormatUint(detail.RecordCount, 10)},
		{Key: "blockCount", Value: strconv.FormatUint(detail.BlockCount, 10)},
		{Key: "footerStatus", Value: detail.FooterStatus},
		{Key: "tailStatus", Value: detail.TailStatus},
	}
}
