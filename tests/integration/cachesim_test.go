package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fastReadFile/internal/simtool"
)

func TestOfflineSimulatorCreatesWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache-sim")
	report, err := simtool.Run(context.Background(), simtool.RunConfig{
		RootDir:   root,
		Profile:   simtool.Profile{Name: "test", Gateways: 2, PointsPerGateway: 4, Rounds: 3, RoundStepMs: 1000},
		BatchSize: 3,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, dir := range []string{"meta", "wal", "segments"} {
		info, err := os.Stat(filepath.Join(root, dir))
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", dir)
		}
	}
	segments, err := filepath.Glob(filepath.Join(root, "segments", "*.seg"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(segments) == 0 {
		t.Fatal("no segment files created")
	}
	if report.Records != 24 {
		t.Fatalf("Records = %d, want %d", report.Records, 24)
	}
	if report.SegmentCount == 0 {
		t.Fatalf("SegmentCount = %d, want > 0", report.SegmentCount)
	}
	if report.WorkspaceBytes == 0 {
		t.Fatalf("WorkspaceBytes = %d, want > 0", report.WorkspaceBytes)
	}
}

func TestCachesimCLIRequiresRoot(t *testing.T) {
	output, err := runGoCommand(t, "run", "./cmd/cachesim")
	if err == nil {
		t.Fatalf("cachesim unexpectedly succeeded:\n%s", output)
	}
	if !strings.Contains(output, "--root is required") {
		t.Fatalf("output %q does not contain missing-root error", output)
	}
}

func runGoCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command("go", args...)
	cmd.Dir = projectRoot(t)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
