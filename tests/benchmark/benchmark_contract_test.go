package benchmark

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"
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
	writeDuration := extractDuration(t, output, `write_duration=([^\s]+)`)
	replayDuration := extractDuration(t, output, `replay_duration=([^\s]+)`)
	const maxDuration = 500 * time.Millisecond
	if writeDuration > maxDuration {
		t.Fatalf("write_duration = %s, want <= %s", writeDuration, maxDuration)
	}
	if replayDuration > maxDuration {
		t.Fatalf("replay_duration = %s, want <= %s", replayDuration, maxDuration)
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

func extractDuration(t *testing.T, output []byte, pattern string) time.Duration {
	t.Helper()
	matches := regexp.MustCompile(pattern).FindSubmatch(output)
	if len(matches) != 2 {
		t.Fatalf("output %q does not match %q", output, pattern)
	}
	duration, err := time.ParseDuration(string(matches[1]))
	if err != nil {
		t.Fatalf("ParseDuration(%q) error = %v", matches[1], err)
	}
	return duration
}
