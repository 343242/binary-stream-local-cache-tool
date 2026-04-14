package unit

import (
	"context"
	"errors"
	"testing"

	"fastReadFile/pkg/cache"
)

func TestStorageErrorSupportsErrorsIs(t *testing.T) {
	err := cache.NewError(cache.ErrDiskFull, "write", "/tmp/demo", "disk full", nil)
	if !errors.Is(err, cache.ErrCode(cache.ErrDiskFull)) {
		t.Fatalf("errors.Is should match error code")
	}
}

func TestWriteBatchRejectsEmptyPayload(t *testing.T) {
	cfg := cache.DefaultConfig(t.TempDir())
	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	_, err = engine.WriteBatch(context.Background(), []cache.RawRecord{{EventTimeUnixMs: 1}})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, cache.ErrCode(cache.ErrValidation)) {
		t.Fatalf("expected validation error code, got %v", err)
	}
}
