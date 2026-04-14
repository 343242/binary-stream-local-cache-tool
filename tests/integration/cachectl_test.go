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

func TestCachectlCommands(t *testing.T) {
	root := t.TempDir()
	cfg := cache.DefaultConfig(root)

	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte("a")},
	}); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
	replayBatch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{MaxRecords: 10})
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if _, err := engine.Ack(context.Background(), "main-server", replayBatch.NextCursor); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
	if _, err := engine.Stats(context.Background()); err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	assertCommandContains(t, root, []string{"stats", "--format", "json"}, "\"WriteBatchesTotal\"")
	assertCommandContains(t, root, []string{"inspect-wal"}, "1 entries")
	assertCommandContains(t, root, []string{"inspect-cursor", "--destination", "main-server"}, "WriteSeq")
	assertCommandContains(t, root, []string{"close-check"}, "clean")
	assertCommandContains(t, root, []string{"verify"}, "ok")

	path := filepath.Join(root, "segments", "000001.seg")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := os.WriteFile(path, append(append([]byte{}, original...), []byte("tail")...), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	assertCommandContains(t, root, []string{"repair-tail", "--segment", "1"}, "repaired")
	repaired, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(repaired) error = %v", err)
	}
	if !bytes.Equal(repaired, original) {
		t.Fatalf("repair-tail did not restore the valid footer boundary")
	}

	assertCommandContains(t, root, []string{"inspect-segment", "--segment", "1"}, "SegmentID")
}

func assertCommandContains(t *testing.T, root string, args []string, want string) {
	t.Helper()

	cmdArgs := append([]string{"run", "./cmd/cachectl", "--root", root}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = projectRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %v failed: %v\n%s", cmdArgs, err, output)
	}
	if !bytes.Contains(output, []byte(want)) {
		t.Fatalf("output %q does not contain %q", output, want)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	return filepath.Dir(filepath.Dir(dir))
}
