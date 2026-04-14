package integration

import (
	"bytes"
	"context"
	"testing"

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

func mustOpenEngine(t *testing.T) *cache.StorageEngine {
	t.Helper()

	engine, err := cache.Open(cache.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return engine
}
