package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"fastReadFile/pkg/cache"
)

func main() {
	records := flag.Int("records", 10000, "record count")
	payloadBytes := flag.Int("payload-bytes", 32, "payload size in bytes")
	flag.Parse()

	root, err := os.MkdirTemp("", "cachebench-*")
	if err != nil {
		exitErr(err)
	}
	defer os.RemoveAll(root)

	engine, err := cache.Open(cache.DefaultConfig(root))
	if err != nil {
		exitErr(err)
	}
	defer engine.Close()

	input := generateRecords(*records, *payloadBytes)
	if _, err := engine.WriteBatch(context.Background(), input[:min(100, len(input))]); err != nil {
		exitErr(err)
	}

	startWrite := time.Now()
	if _, err := engine.WriteBatch(context.Background(), input); err != nil {
		exitErr(err)
	}
	writeDuration := time.Since(startWrite)

	startReplay := time.Now()
	replayBatch, err := engine.Replay(context.Background(), "benchmark", cache.ReplayLimit{MaxRecords: *records + 100})
	if err != nil {
		exitErr(err)
	}
	replayDuration := time.Since(startReplay)

	fmt.Printf("records=%d payload_bytes=%d write_duration=%s replay_duration=%s replayed=%d target_note=<100ms on recommended hardware; CI threshold is relaxed>\n",
		*records,
		*payloadBytes,
		writeDuration,
		replayDuration,
		replayBatch.RecordCount,
	)
}

func generateRecords(count, payloadBytes int) []cache.RawRecord {
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

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
