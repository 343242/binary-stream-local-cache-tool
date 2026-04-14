package benchmark

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBenchmarkContract(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/cachebench", "--records", "10000", "--payload-bytes", "32")
	cmd.Dir = projectRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run cachebench failed: %v\n%s", err, output)
	}
	if !bytes.Contains(output, []byte("write_duration=")) {
		t.Fatalf("output %q does not contain write duration", output)
	}
	if !bytes.Contains(output, []byte("replay_duration=")) {
		t.Fatalf("output %q does not contain replay duration", output)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	return dir
}
