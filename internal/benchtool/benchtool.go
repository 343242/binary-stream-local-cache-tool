package benchtool

import (
	"context"
	"fmt"
	"os"
	"time"

	"fastReadFile/pkg/cache"
)

type Report struct {
	Records        int
	PayloadBytes   int
	WriteDuration  time.Duration
	ReplayDuration time.Duration
	Replayed       int
}

func Run(records, payloadBytes int) (Report, error) {
	root, err := os.MkdirTemp("", "cachebench-*")
	if err != nil {
		return Report{}, err
	}
	defer os.RemoveAll(root)

	engine, err := cache.Open(cache.DefaultConfig(root))
	if err != nil {
		return Report{}, err
	}
	defer engine.Close()

	input := GenerateRecords(records, payloadBytes)
	if _, err := engine.WriteBatch(context.Background(), input[:min(100, len(input))]); err != nil {
		return Report{}, err
	}

	startWrite := time.Now()
	if _, err := engine.WriteBatch(context.Background(), input); err != nil {
		return Report{}, err
	}
	writeDuration := time.Since(startWrite)

	startReplay := time.Now()
	replayBatch, err := engine.Replay(context.Background(), "benchmark", cache.ReplayLimit{MaxRecords: records + 100})
	if err != nil {
		return Report{}, err
	}
	replayDuration := time.Since(startReplay)

	return Report{
		Records:        records,
		PayloadBytes:   payloadBytes,
		WriteDuration:  writeDuration,
		ReplayDuration: replayDuration,
		Replayed:       replayBatch.RecordCount,
	}, nil
}

func FormatReport(report Report) string {
	return fmt.Sprintf(
		"records=%d payload_bytes=%d write_duration=%s replay_duration=%s replayed=%d target_note=<100ms on recommended hardware; CI threshold can be overridden via FASTREADFILE_BENCHMARK_MAX_DURATION>\n",
		report.Records,
		report.PayloadBytes,
		report.WriteDuration,
		report.ReplayDuration,
		report.Replayed,
	)
}

func GenerateRecords(count, payloadBytes int) []cache.RawRecord {
	records := make([]cache.RawRecord, 0, count)
	for idx := 0; idx < count; idx++ {
		payload := make([]byte, payloadBytes)
		for j := range payload {
			payload[j] = byte((idx + j) % 251)
		}
		records = append(records, cache.RawRecord{
			EventTimeUnixMs: int64(idx + 1),
			Payload:         payload,
		})
	}
	return records
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
