package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	replaystore "fastReadFile/internal/replay"
	"fastReadFile/pkg/cache"
)

func TestReplayReturnsRecordsInPhysicalOrder(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 300, Payload: []byte("first")},
		{EventTimeUnixMs: 100, Payload: []byte("second")},
		{EventTimeUnixMs: 200, Payload: []byte("third")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if batch.RecordCount != 3 {
		t.Fatalf("RecordCount = %d, want %d", batch.RecordCount, 3)
	}
	if !bytes.Equal(batch.Records[0].Payload, []byte("first")) {
		t.Fatalf("first payload = %q, want %q", batch.Records[0].Payload, []byte("first"))
	}
	if !bytes.Equal(batch.Records[1].Payload, []byte("second")) {
		t.Fatalf("second payload = %q, want %q", batch.Records[1].Payload, []byte("second"))
	}
	if !bytes.Equal(batch.Records[2].Payload, []byte("third")) {
		t.Fatalf("third payload = %q, want %q", batch.Records[2].Payload, []byte("third"))
	}
}

func TestReplayRespectsMaxRecordsAndAckAdvancesCursor(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
		{EventTimeUnixMs: 2, Payload: []byte("b")},
		{EventTimeUnixMs: 3, Payload: []byte("c")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	firstBatch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 2})
	if err != nil {
		t.Fatalf("Replay(first) error = %v", err)
	}
	if firstBatch.RecordCount != 2 {
		t.Fatalf("firstBatch.RecordCount = %d, want %d", firstBatch.RecordCount, 2)
	}
	if _, err := engine.Ack(context.Background(), "main-server", firstBatch.NextCursor); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}

	secondBatch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 2})
	if err != nil {
		t.Fatalf("Replay(second) error = %v", err)
	}
	if secondBatch.RecordCount != 1 {
		t.Fatalf("secondBatch.RecordCount = %d, want %d", secondBatch.RecordCount, 1)
	}
	if !bytes.Equal(secondBatch.Records[0].Payload, []byte("c")) {
		t.Fatalf("second batch payload = %q, want %q", secondBatch.Records[0].Payload, []byte("c"))
	}
}

func TestReplayRespectsMaxBytes(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("1234")},
		{EventTimeUnixMs: 2, Payload: []byte("5678")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxBytes: 5})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if batch.RecordCount != 1 {
		t.Fatalf("RecordCount = %d, want %d", batch.RecordCount, 1)
	}
	if batch.TotalPayloadBytes != 4 {
		t.Fatalf("TotalPayloadBytes = %d, want %d", batch.TotalPayloadBytes, 4)
	}
}

func TestReplayReturnsCursorCRCAndAckRejectsTamperedCursor(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if batch.NextCursor.CRC32 == 0 {
		t.Fatalf("NextCursor.CRC32 = 0, want non-zero")
	}

	tampered := batch.NextCursor
	tampered.CRC32++
	if _, err := engine.Ack(context.Background(), "main-server", tampered); !errors.Is(err, cache.ErrCode(cache.ErrCursorInvalid)) {
		t.Fatalf("Ack(tampered) error = %v, want cursor invalid", err)
	}
}

func TestConcurrentWriteBatchAndReplay(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	const totalRecords = 40
	var wg sync.WaitGroup
	writerDone := make(chan struct{})
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(writerDone)
		for i := 0; i < totalRecords; i++ {
			if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
				{EventTimeUnixMs: int64(i + 1), Payload: []byte(fmt.Sprintf("payload-%02d", i+1))},
			}); err != nil {
				errCh <- err
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		acked := 0
		deadline := time.Now().Add(5 * time.Second)
		for acked < totalRecords && time.Now().Before(deadline) {
			batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 5})
			if err != nil {
				errCh <- err
				return
			}
			if batch.RecordCount == 0 {
				select {
				case <-writerDone:
				default:
				}
				time.Sleep(10 * time.Millisecond)
				continue
			}
			if _, err := engine.Ack(context.Background(), "main-server", batch.NextCursor); err != nil {
				errCh <- err
				return
			}
			acked += batch.RecordCount
		}
		if acked != totalRecords {
			errCh <- fmt.Errorf("acked = %d, want %d", acked, totalRecords)
		}
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent engine access error = %v", err)
		}
	}

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 5})
	if err != nil {
		t.Fatalf("Replay(final) error = %v", err)
	}
	if batch.RecordCount != 0 {
		t.Fatalf("final RecordCount = %d, want %d", batch.RecordCount, 0)
	}
}

func TestAckRejectsCursorPastNextWriteSeq(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	_, err := engine.Ack(context.Background(), "main-server", replaystore.FinalizeCursor(cache.ReplayCursor{
		Version:         1,
		SegmentID:       1,
		BlockOffset:     0,
		RecordIndex:     0,
		WriteSeq:        2,
		UpdatedAtUnixMs: time.Now().UnixMilli(),
	}))
	if !errors.Is(err, cache.ErrCode(cache.ErrCursorInvalid)) {
		t.Fatalf("Ack() error = %v, want cursor invalid", err)
	}
}

func TestReplayPreservesPhysicalOrderAcrossClockRollbackBatches(t *testing.T) {
	engine := mustOpenEngine(t)
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 300, Payload: []byte("first-batch")},
	}); err != nil {
		t.Fatalf("WriteBatch(first) error = %v", err)
	}
	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 100, Payload: []byte("clock-rollback")},
		{EventTimeUnixMs: 50, Payload: []byte("rollback-tail")},
	}); err != nil {
		t.Fatalf("WriteBatch(second) error = %v", err)
	}

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if batch.RecordCount != 3 {
		t.Fatalf("RecordCount = %d, want %d", batch.RecordCount, 3)
	}
	if !bytes.Equal(batch.Records[0].Payload, []byte("first-batch")) {
		t.Fatalf("record[0] = %q, want first-batch", batch.Records[0].Payload)
	}
	if !bytes.Equal(batch.Records[1].Payload, []byte("clock-rollback")) {
		t.Fatalf("record[1] = %q, want clock-rollback", batch.Records[1].Payload)
	}
	if !bytes.Equal(batch.Records[2].Payload, []byte("rollback-tail")) {
		t.Fatalf("record[2] = %q, want rollback-tail", batch.Records[2].Payload)
	}
}

func mustOpenEngine(t *testing.T) *cache.StorageEngine {
	t.Helper()

	engine, err := cache.Open(cache.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return engine
}
