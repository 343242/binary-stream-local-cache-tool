package integration

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"fastReadFile/internal/codec"
	"fastReadFile/internal/desktop/service"
	"fastReadFile/internal/replay"
	"fastReadFile/internal/segment"
	"fastReadFile/internal/wal"
	"fastReadFile/pkg/cache"
)

func TestDesktopService(t *testing.T) {
	t.Run("TestListCursorsScansReplayDirectory", runTestListCursorsScansReplayDirectory)
	t.Run("TestGetSegmentDetailCountsBlocksFromSegmentData", runTestGetSegmentDetailCountsBlocksFromSegmentData)
	t.Run("TestListSegmentsIncludesSpecFieldsAndPagination", runTestListSegmentsIncludesSpecFieldsAndPagination)
	t.Run("TestGetWALAndCheckpointDetailExposeReadOnlyContracts", runTestGetWALAndCheckpointDetailExposeReadOnlyContracts)
	t.Run("TestGetCursorDetailReportsBackupStatus", runTestGetCursorDetailReportsBackupStatus)
}

func TestListCursorsScansReplayDirectory(t *testing.T) {
	runTestListCursorsScansReplayDirectory(t)
}

func TestGetSegmentDetailCountsBlocksFromSegmentData(t *testing.T) {
	runTestGetSegmentDetailCountsBlocksFromSegmentData(t)
}

func TestListSegmentsIncludesSpecFieldsAndPagination(t *testing.T) {
	runTestListSegmentsIncludesSpecFieldsAndPagination(t)
}

func TestGetWALAndCheckpointDetailExposeReadOnlyContracts(t *testing.T) {
	runTestGetWALAndCheckpointDetailExposeReadOnlyContracts(t)
}

func TestGetCursorDetailReportsBackupStatus(t *testing.T) {
	runTestGetCursorDetailReportsBackupStatus(t)
}

func runTestListCursorsScansReplayDirectory(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeDesktopWorkspaceRoot(t)
	store := replay.NewCursorStore(root)
	if err := store.Save("alpha", cache.ReplayCursor{
		Version:         1,
		SegmentID:       3,
		BlockOffset:     96,
		RecordIndex:     2,
		WriteSeq:        11,
		UpdatedAtUnixMs: 1234,
	}); err != nil {
		t.Fatalf("Save(alpha) error = %v", err)
	}

	replayDir := filepath.Join(root, "meta", "replay")
	if err := os.MkdirAll(replayDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(replayDir, "broken.cursor"), []byte("corrupt"), 0o644); err != nil {
		t.Fatalf("WriteFile(broken) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(replayDir, "ignored.cursor.bak"), []byte("backup"), 0o644); err != nil {
		t.Fatalf("WriteFile(backup) error = %v", err)
	}

	cursors, err := service.ListCursors(root)
	if err != nil {
		t.Fatalf("ListCursors() error = %v", err)
	}
	if len(cursors) != 2 {
		t.Fatalf("len(cursors) = %d, want %d", len(cursors), 2)
	}

	got := make(map[string]cacheCursorSummary, len(cursors))
	for _, cursor := range cursors {
		got[cursor.Destination] = cacheCursorSummary{
			WriteSeq:  cursor.WriteSeq,
			UpdatedAt: cursor.UpdatedAt,
			Status:    cursor.Status,
		}
	}

	if got["alpha"].WriteSeq != 11 {
		t.Fatalf("alpha.WriteSeq = %d, want %d", got["alpha"].WriteSeq, 11)
	}
	if got["alpha"].Status != "ok" {
		t.Fatalf("alpha.Status = %q, want %q", got["alpha"].Status, "ok")
	}
	if got["broken"].Status != "corrupted" {
		t.Fatalf("broken.Status = %q, want %q", got["broken"].Status, "corrupted")
	}
}

func runTestGetSegmentDetailCountsBlocksFromSegmentData(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeDesktopWorkspaceRoot(t)
	segmentsDir := filepath.Join(root, "segments")

	firstBlock := mustBuildDesktopBlock(t, 1, [][]byte{[]byte("a")}, []int64{10})
	secondBlock := mustBuildDesktopBlock(t, 2, [][]byte{[]byte("b")}, []int64{11})
	partialTail := secondBlock[:len(secondBlock)-4]
	segmentPath := filepath.Join(segmentsDir, "000001.seg")
	if err := os.WriteFile(segmentPath, append(append([]byte{}, firstBlock...), partialTail...), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	detail, err := service.GetSegmentDetail(root, 1)
	if err != nil {
		t.Fatalf("GetSegmentDetail() error = %v", err)
	}
	if detail.BlockCount != 1 {
		t.Fatalf("BlockCount = %d, want %d", detail.BlockCount, 1)
	}
	if detail.FooterStatus != "missing" {
		t.Fatalf("FooterStatus = %q, want %q", detail.FooterStatus, "missing")
	}
	if detail.TailStatus != "truncated" {
		t.Fatalf("TailStatus = %q, want %q", detail.TailStatus, "truncated")
	}
	if detail.MinEventTime != 10 {
		t.Fatalf("MinEventTime = %d, want %d", detail.MinEventTime, 10)
	}
	if detail.MaxEventTime != 10 {
		t.Fatalf("MaxEventTime = %d, want %d", detail.MaxEventTime, 10)
	}
	if detail.RawPreviewHex == "" {
		t.Fatalf("RawPreviewHex = empty, want preview bytes")
	}
	if _, err := hex.DecodeString(detail.RawPreviewHex); err != nil {
		t.Fatalf("DecodeString(RawPreviewHex) error = %v", err)
	}
	if len(detail.StructuredPreview) == 0 {
		t.Fatalf("StructuredPreview = empty, want summary fields")
	}
	if detail.StructuredPreview[0].Key == "" || detail.StructuredPreview[0].Value == "" {
		t.Fatalf("StructuredPreview[0] = %+v, want populated key/value", detail.StructuredPreview[0])
	}
}

func runTestListSegmentsIncludesSpecFieldsAndPagination(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeDesktopWorkspaceRoot(t)
	firstBlock := mustBuildDesktopBlock(t, 1, [][]byte{[]byte("alpha")}, []int64{111})
	secondBlock := mustBuildDesktopBlock(t, 2, [][]byte{[]byte("bravo")}, []int64{222})
	writeSealedDesktopSegment(t, filepath.Join(root, "segments", "000001.seg"), 1, firstBlock, segment.Footer{
		SegmentID:     1,
		DataEndOffset: uint64(len(firstBlock)),
		RecordCount:   1,
		FirstWriteSeq: 1,
		LastWriteSeq:  1,
		MinEventTime:  111,
		MaxEventTime:  111,
		LastBatchSeq:  7,
	})
	writeSealedDesktopSegment(t, filepath.Join(root, "segments", "000002.seg"), 2, secondBlock, segment.Footer{
		SegmentID:     2,
		DataEndOffset: uint64(len(secondBlock)),
		RecordCount:   1,
		FirstWriteSeq: 2,
		LastWriteSeq:  2,
		MinEventTime:  222,
		MaxEventTime:  222,
		LastBatchSeq:  8,
	})

	paged, err := service.ListSegments(root, 1, 1)
	if err != nil {
		t.Fatalf("ListSegments(page=1) error = %v", err)
	}
	if paged.Page != 1 || paged.PageSize != 1 {
		t.Fatalf("page metadata = (%d,%d), want (1,1)", paged.Page, paged.PageSize)
	}
	if paged.TotalItems != 2 {
		t.Fatalf("TotalItems = %d, want %d", paged.TotalItems, 2)
	}
	if !paged.HasNext {
		t.Fatalf("HasNext = false, want true")
	}
	if len(paged.Items) != 1 {
		t.Fatalf("len(Items) = %d, want %d", len(paged.Items), 1)
	}
	if paged.Items[0].SegmentID != 2 {
		t.Fatalf("Items[0].SegmentID = %d, want %d", paged.Items[0].SegmentID, 2)
	}
	if paged.Items[0].MinEventTime != 222 || paged.Items[0].MaxEventTime != 222 {
		t.Fatalf("Items[0] event times = (%d,%d), want (222,222)", paged.Items[0].MinEventTime, paged.Items[0].MaxEventTime)
	}
	if paged.Items[0].LastBatchSeq != 8 {
		t.Fatalf("Items[0].LastBatchSeq = %d, want %d", paged.Items[0].LastBatchSeq, 8)
	}
	if paged.Items[0].Health != "sealed" {
		t.Fatalf("Items[0].Health = %q, want %q", paged.Items[0].Health, "sealed")
	}

	nextPage, err := service.ListSegments(root, 2, 1)
	if err != nil {
		t.Fatalf("ListSegments(page=2) error = %v", err)
	}
	if nextPage.HasNext {
		t.Fatalf("page 2 HasNext = true, want false")
	}
	if len(nextPage.Items) != 1 || nextPage.Items[0].SegmentID != 1 {
		t.Fatalf("page 2 items = %+v, want segment 1", nextPage.Items)
	}
}

func runTestGetWALAndCheckpointDetailExposeReadOnlyContracts(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeDesktopWorkspaceRoot(t)
	log, err := wal.Open(root)
	if err != nil {
		t.Fatalf("wal.Open() error = %v", err)
	}
	block := mustBuildDesktopBlock(t, 5, [][]byte{[]byte("wal-one")}, []int64{500})
	meta, err := log.Append(9, block)
	if err != nil {
		t.Fatalf("wal.Append() error = %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("wal.Close() error = %v", err)
	}
	if err := wal.NewCheckpointStore(root).Save(wal.Checkpoint{
		LastBatchSeq:     9,
		LastWALEndOffset: meta.EndOffset,
		UpdatedAtUnixMs:  999,
	}); err != nil {
		t.Fatalf("Checkpoint.Save() error = %v", err)
	}

	walDetail, err := service.GetWALDetail(root)
	if err != nil {
		t.Fatalf("GetWALDetail() error = %v", err)
	}
	if walDetail.FirstBatchSeq != 9 || walDetail.LastBatchSeq != 9 {
		t.Fatalf("WAL batch range = (%d,%d), want (9,9)", walDetail.FirstBatchSeq, walDetail.LastBatchSeq)
	}
	if walDetail.LastEndOffset != meta.EndOffset {
		t.Fatalf("LastEndOffset = %d, want %d", walDetail.LastEndOffset, meta.EndOffset)
	}
	if walDetail.Health != "ok" {
		t.Fatalf("Health = %q, want %q", walDetail.Health, "ok")
	}

	checkpoint, err := service.GetCheckpointDetail(root)
	if err != nil {
		t.Fatalf("GetCheckpointDetail() error = %v", err)
	}
	if checkpoint.LastBatchSeq != 9 {
		t.Fatalf("LastBatchSeq = %d, want %d", checkpoint.LastBatchSeq, 9)
	}
	if checkpoint.LastWALEndOffset != meta.EndOffset {
		t.Fatalf("LastWALEndOffset = %d, want %d", checkpoint.LastWALEndOffset, meta.EndOffset)
	}
	if checkpoint.UpdatedAt != 999 {
		t.Fatalf("UpdatedAt = %d, want %d", checkpoint.UpdatedAt, 999)
	}
	if checkpoint.IntegrityStatus != "ok" {
		t.Fatalf("IntegrityStatus = %q, want %q", checkpoint.IntegrityStatus, "ok")
	}
}

func runTestGetCursorDetailReportsBackupStatus(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeDesktopWorkspaceRoot(t)
	store := replay.NewCursorStore(root)
	if err := store.Save("beta", cache.ReplayCursor{
		Version:         1,
		SegmentID:       4,
		BlockOffset:     512,
		RecordIndex:     3,
		WriteSeq:        44,
		UpdatedAtUnixMs: 777,
	}); err != nil {
		t.Fatalf("Save(beta) error = %v", err)
	}

	detail, err := service.GetCursorDetail(root, "beta")
	if err != nil {
		t.Fatalf("GetCursorDetail() error = %v", err)
	}
	if detail.Destination != "beta" {
		t.Fatalf("Destination = %q, want %q", detail.Destination, "beta")
	}
	if detail.WriteSeq != 44 || detail.SegmentID != 4 {
		t.Fatalf("cursor detail = %+v, want writeSeq=44 segmentID=4", detail)
	}
	if detail.CRCStatus != "valid" {
		t.Fatalf("CRCStatus = %q, want %q", detail.CRCStatus, "valid")
	}
	if detail.BackupStatus != "present" {
		t.Fatalf("BackupStatus = %q, want %q", detail.BackupStatus, "present")
	}
}

type cacheCursorSummary struct {
	WriteSeq  uint64
	UpdatedAt int64
	Status    string
}

func mustBuildDesktopBlock(t *testing.T, firstSeq uint64, payloads [][]byte, eventTimes []int64) []byte {
	t.Helper()

	records := make([][]byte, 0, len(payloads))
	for idx, payload := range payloads {
		record, err := codec.EncodeRecord(firstSeq+uint64(idx), cache.RawRecord{
			EventTimeUnixMs: eventTimes[idx],
			Payload:         payload,
		})
		if err != nil {
			t.Fatalf("EncodeRecord() error = %v", err)
		}
		records = append(records, record)
	}

	block, err := codec.BuildBlock(records, firstSeq, firstSeq+uint64(len(records))-1, eventTimes[0], eventTimes[len(eventTimes)-1])
	if err != nil {
		t.Fatalf("BuildBlock() error = %v", err)
	}
	return block
}

func makeDesktopWorkspaceRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, rel := range []string{"meta", "meta/replay", "segments", "wal"} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", rel, err)
		}
	}
	return root
}

func writeSealedDesktopSegment(t *testing.T, path string, segmentID uint64, block []byte, footer segment.Footer) {
	t.Helper()

	footer.SegmentID = segmentID
	footer.DataEndOffset = uint64(len(block))
	encodedFooter, err := segment.EncodeFooter(footer)
	if err != nil {
		t.Fatalf("EncodeFooter() error = %v", err)
	}
	if err := os.WriteFile(path, append(append([]byte{}, block...), encodedFooter...), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
