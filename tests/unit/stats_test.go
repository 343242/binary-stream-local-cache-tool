package unit

import (
	"context"
	"testing"

	"fastReadFile/pkg/cache"
)

func TestStatsReflectWritesAndReplays(t *testing.T) {
	cfg := cache.DefaultConfig(t.TempDir())
	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
	if _, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10}); err != nil {
		t.Fatalf("Replay() error = %v", err)
	}

	stats, err := engine.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.IO.WriteBatchesTotal != 1 {
		t.Fatalf("WriteBatchesTotal = %d, want %d", stats.IO.WriteBatchesTotal, 1)
	}
	if stats.IO.ReplayBatchesTotal != 1 {
		t.Fatalf("ReplayBatchesTotal = %d, want %d", stats.IO.ReplayBatchesTotal, 1)
	}
}
