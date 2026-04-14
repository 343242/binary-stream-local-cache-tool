package unit

import (
	"context"
	"testing"

	"fastReadFile/pkg/cache"
)

func TestPublicAPIContractsExist(t *testing.T) {
	cfg := cache.Config{RootDir: t.TempDir()}
	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	_, err = engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte{0x01}},
	})
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
}

func TestDefaultConfigIsStable(t *testing.T) {
	cfg := cache.DefaultConfig(t.TempDir())
	if cfg.BlockTargetSizeBytes != 1<<20 {
		t.Fatalf("BlockTargetSizeBytes = %d, want %d", cfg.BlockTargetSizeBytes, 1<<20)
	}
	if cfg.CheckpointBytes != 64<<20 {
		t.Fatalf("CheckpointBytes = %d, want %d", cfg.CheckpointBytes, 64<<20)
	}
}
