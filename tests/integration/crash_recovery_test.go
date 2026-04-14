package integration

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"fastReadFile/pkg/cache"
)

func TestCrashRecoveryReplaysWALWhenProcessDiesAfterWALAppend(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperCrashAfterWALAppend")
	cmd.Env = append(os.Environ(),
		"FASTREADFILE_HELPER_CRASH=1",
		"FASTREADFILE_ROOT="+root,
	)
	cmd.Dir = projectRoot(t)
	err := cmd.Run()
	if err == nil {
		t.Fatalf("helper process succeeded, want crash")
	}

	engine, err := cache.Open(cache.DefaultConfig(root))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if batch.RecordCount != 1 {
		t.Fatalf("RecordCount = %d, want %d", batch.RecordCount, 1)
	}
	if !bytes.Equal(batch.Records[0].Payload, []byte("crash-payload")) {
		t.Fatalf("payload = %q, want %q", batch.Records[0].Payload, []byte("crash-payload"))
	}
	segmentPath := filepath.Join(root, "segments", "000001.seg")
	if _, err := os.Stat(segmentPath); err != nil {
		t.Fatalf("Stat(segment) error = %v", err)
	}
}

func TestHelperCrashAfterWALAppend(t *testing.T) {
	if os.Getenv("FASTREADFILE_HELPER_CRASH") != "1" {
		return
	}

	root := os.Getenv("FASTREADFILE_ROOT")
	engine, err := cache.Open(cache.DefaultConfig(root))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	cache.SetWriteBatchHookForTesting(func(stage string) {
		if stage == "after_wal_append_before_segment" {
			os.Exit(137)
		}
	})
	defer cache.SetWriteBatchHookForTesting(nil)

	_, _ = engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("crash-payload")},
	})
	t.Fatalf("helper did not crash")
}
