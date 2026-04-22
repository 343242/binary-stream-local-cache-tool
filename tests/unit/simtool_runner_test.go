package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fastReadFile/internal/simtool"
)

func TestValidateRootAcceptsNonexistentDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := simtool.ValidateRoot(root); err != nil {
		t.Fatalf("ValidateRoot(nonexistent) error = %v", err)
	}
}

func TestValidateRootAcceptsEmptyDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := simtool.ValidateRoot(root); err != nil {
		t.Fatalf("ValidateRoot(empty dir) error = %v", err)
	}
}

func TestValidateRootRejectsFilePath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.WriteFile(root, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := simtool.ValidateRoot(root); err == nil {
		t.Fatal("ValidateRoot(file path) unexpectedly succeeded")
	}
}

func TestValidateRootRejectsNonEmptyDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "data.bin"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := simtool.ValidateRoot(root); err == nil {
		t.Fatal("ValidateRoot(non-empty dir) unexpectedly succeeded")
	}
}

func TestFormatSimReportIncludesKeyFields(t *testing.T) {
	report := simtool.Report{
		RootDir:          "/tmp/cache-sim",
		ProfileName:      "medium",
		Records:          200000,
		PayloadBytes:     24,
		BatchSize:        1000,
		TotalDuration:    2 * time.Second,
		AvgBatchDuration: 10 * time.Millisecond,
		MaxBatchDuration: 15 * time.Millisecond,
		SegmentCount:     3,
		WALFileCount:     1,
		WorkspaceBytes:   4096,
	}
	output := simtool.FormatReport(report)
	for _, want := range []string{
		"root=/tmp/cache-sim",
		"profile=medium",
		"records=200000",
		"segment_count=3",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output %q does not contain %q", output, want)
		}
	}
}
