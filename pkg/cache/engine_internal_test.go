package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestShutdownCanReturnOnContextCancelDuringClose(t *testing.T) {
	cfg := DefaultConfig(t.TempDir())
	engine, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		closeStepHook = nil
		_ = engine.Close()
	}()

	if _, err := engine.WriteBatch(context.Background(), []RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	release := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	closeStepHook = func(step string) {
		if step == "after_checkpoint" {
			cancel()
			<-release
		}
	}

	err = engine.Shutdown(ctx)
	close(release)
	if err == nil {
		t.Fatalf("Shutdown() error = nil, want timeout")
	}
	if errCode, ok := err.(*StorageError); !ok || errCode.Code != ErrTimeout {
		t.Fatalf("Shutdown() error = %v, want timeout", err)
	}
}

func TestConcurrentEngineAccessIsSerialized(t *testing.T) {
	cfg := DefaultConfig(t.TempDir())
	engine, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	if _, err := engine.WriteBatch(context.Background(), []RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
		{EventTimeUnixMs: 2, Payload: []byte("b")},
	}); err != nil {
		t.Fatalf("WriteBatch(seed) error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = engine.Stats(context.Background())
			batch, _ := engine.Replay(context.Background(), "main-server", ReplayLimit{MaxRecords: 1})
			if batch.RecordCount > 0 {
				_, _ = engine.Ack(context.Background(), "main-server", batch.NextCursor)
			}
			_, _ = engine.WriteBatch(context.Background(), []RawRecord{
				{EventTimeUnixMs: time.Now().UnixMilli() + int64(idx), Payload: []byte("x")},
			})
		}(i)
	}
	wg.Wait()
}
