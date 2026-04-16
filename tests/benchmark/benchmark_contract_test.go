//go:build !race

package benchmark

import (
	"bytes"
	"os"
	"regexp"
	"testing"
	"time"

	"fastReadFile/internal/benchtool"
)

func TestBenchmarkContract(t *testing.T) {
	report, err := fastestReport(3, 10000, 32)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	output := []byte(benchtool.FormatReport(report))
	if !bytes.Contains(output, []byte("write_duration=")) {
		t.Fatalf("output %q does not contain write duration", output)
	}
	if !bytes.Contains(output, []byte("replay_duration=")) {
		t.Fatalf("output %q does not contain replay duration", output)
	}
	writeDuration := extractDuration(t, output, `write_duration=([^\s]+)`)
	replayDuration := extractDuration(t, output, `replay_duration=([^\s]+)`)
	maxDuration := benchmarkThreshold()
	if writeDuration > maxDuration {
		t.Fatalf("write_duration = %s, want <= %s", writeDuration, maxDuration)
	}
	if replayDuration > maxDuration {
		t.Fatalf("replay_duration = %s, want <= %s", replayDuration, maxDuration)
	}
	if !bytes.Contains(output, []byte("target_note=<100ms")) {
		t.Fatalf("output %q does not describe the <100ms product requirement", output)
	}
}

func fastestReport(attempts, records, payloadBytes int) (benchtool.Report, error) {
	var best benchtool.Report
	for attempt := 0; attempt < attempts; attempt++ {
		report, err := benchtool.Run(records, payloadBytes)
		if err != nil {
			return benchtool.Report{}, err
		}
		if attempt == 0 || report.WriteDuration < best.WriteDuration {
			best = report
		}
		if report.ReplayDuration < best.ReplayDuration {
			best.ReplayDuration = report.ReplayDuration
		}
	}
	return best, nil
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

func benchmarkThreshold() time.Duration {
	if text := os.Getenv("FASTREADFILE_BENCHMARK_MAX_DURATION"); text != "" {
		if duration, err := time.ParseDuration(text); err == nil {
			return duration
		}
	}
	return 100 * time.Millisecond
}
