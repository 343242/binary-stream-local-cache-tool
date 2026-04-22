# Binary Stream Local Cache Tool

A Go-based local file cache for high-throughput outage buffering.

This project is designed for industrial or edge-side data collection systems that must keep accepting records when the upstream network or primary server is unavailable. It stores records as binary streams on local disk, replays them in physical write order, and recovers safely after process crashes or partial writes.

## What it is for

- Buffering locally when the primary server is unreachable.
- Sustaining sequential writes at high frequency with low overhead.
- Replaying unacknowledged data in append order after recovery.
- Keeping storage behavior predictable: WAL first, then segment append, then cursor ACK.

## Current model

- Single machine
- Single process
- Single writer
- Sequential storage
- Replay-first read pattern
- Binary payload storage without business-field schema enforcement

This is intentionally not a general-purpose embedded database.

## Key features

- Binary stream persistence: records are stored as `eventTime + writeSeq + payloadLen + CRC + payload`.
- WAL-backed durability: every block is persisted to WAL before segment append.
- Size-based segment rolling: active segment rolls when it would exceed `target + slack`.
- Crash recovery: rebuilds active segment state, repairs valid tails conservatively, and replays WAL blocks that are not yet durable in segments.
- Ordered replay: replay is based on physical write order and `writeSeq`, not sensor timestamp monotonicity.
- Cursor persistence: per-destination replay cursor with main/backup files and CRC protection.
- Retention: sealed segments older than the configured retention window can be removed after they are fully acknowledged.
- Ops tooling: `cachectl` supports stats, cursor inspection, WAL inspection, segment inspection, verification, close-check, and tail repair.
- Benchmark tool: `cachebench` measures write and replay latency for a synthetic workload.
- Offline simulator: `cachesim` writes a field-oriented workload into a real workspace directory and produces real segment/WAL output.

## Default configuration

The public config type is [`cache.Config`](/home/instant/projects/fastReadFile/pkg/cache/config.go) and defaults come from [`internal/core/config.go`](/home/instant/projects/fastReadFile/internal/core/config.go).

Default values:

- `SegmentTargetSizeBytes = 256 MiB`
- `SegmentSlackSizeBytes = 4 MiB`
- `BlockTargetSizeBytes = 1 MiB`
- `CheckpointInterval = 5s`
- `CheckpointBytes = 64 MiB`
- `SegmentFsyncInterval = 100ms`
- `SegmentFsyncBytes = 4 MiB`
- `RetentionDays = 30`

## Storage layout

Under a chosen `RootDir`, the engine manages files like:

```text
<root>/
  meta/
    checkpoint.meta
    lifecycle.state
    replay/
      <destination>.cursor
      <destination>.cursor.bak
  segments/
    000001.seg
    000002.seg
    ...
  wal/
    active.wal
```

Durability flow:

1. Build one or more binary blocks from `WriteBatch`.
2. Append each block to WAL and `fsync` WAL.
3. Append the same block to the active segment.
4. `fsync` the segment according to configured thresholds or on checkpoint/seal/close.
5. Save checkpoint only after active segment durability is ensured.

## Public API

The main entry point is [`pkg/cache/engine.go`](/home/instant/projects/fastReadFile/pkg/cache/engine.go).

Primary operations:

- `cache.Open(cfg)`
- `engine.WriteBatch(ctx, []cache.RawRecord)`
- `engine.Replay(ctx, destination, limit)`
- `engine.Ack(ctx, destination, cursor)`
- `engine.Stats(ctx)`
- `engine.Recover(ctx)`
- `engine.Close()`
- `engine.Shutdown(ctx)`

Core types:

- `RawRecord`: `EventTimeUnixMs`, `Payload`
- `ReplayLimit`: `MaxRecords`, `MaxBytes`
- `ReplayBatch`: records plus `NextCursor`
- `StatsSnapshot`: capacity, IO, replay, and health counters

## Quick start

```go
package main

import (
	"context"
	"log"

	"fastReadFile/pkg/cache"
)

func main() {
	cfg := cache.DefaultConfig("/tmp/local-cache")

	engine, err := cache.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close()

	_, err = engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1710000000000, Payload: []byte{0x01, 0x02, 0x03}},
		{EventTimeUnixMs: 1710000001000, Payload: []byte("sensor-frame")},
	})
	if err != nil {
		log.Fatal(err)
	}

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{
		MaxRecords: 1000,
		MaxBytes:   4 << 20,
	})
	if err != nil {
		log.Fatal(err)
	}

	if batch.RecordCount > 0 {
		if _, err := engine.Ack(context.Background(), "main-server", batch.NextCursor); err != nil {
			log.Fatal(err)
		}
	}
}
```

## CLI tools

### `cachectl`

[`cmd/cachectl/main.go`](/home/instant/projects/fastReadFile/cmd/cachectl/main.go) provides operational commands:

```bash
rtk go run ./cmd/cachectl --root /tmp/local-cache stats --format json
rtk go run ./cmd/cachectl --root /tmp/local-cache inspect-wal
rtk go run ./cmd/cachectl --root /tmp/local-cache inspect-cursor --destination main-server
rtk go run ./cmd/cachectl --root /tmp/local-cache inspect-segment --segment 1
rtk go run ./cmd/cachectl --root /tmp/local-cache verify
rtk go run ./cmd/cachectl --root /tmp/local-cache repair-tail --segment 1
rtk go run ./cmd/cachectl --root /tmp/local-cache close-check
```

### `cachebench`

[`cmd/cachebench/main.go`](/home/instant/projects/fastReadFile/cmd/cachebench/main.go) runs a synthetic benchmark:

```bash
rtk go run ./cmd/cachebench --records 10000 --payload-bytes 32
```

Typical output:

```text
records=10000 payload_bytes=32 write_duration=13.2ms replay_duration=42.1ms replayed=10100 target_note=<100ms on recommended hardware; CI threshold can be overridden via FASTREADFILE_BENCHMARK_MAX_DURATION>
```

The product target remains `<100ms / 10000 records` on recommended hardware. Benchmark CI uses the same target by default; an override hook exists only for noisy environments.

### `cachesim`

[`cmd/cachesim/main.go`](/home/instant/projects/fastReadFile/cmd/cachesim/main.go) runs an offline field-oriented simulator against a real workspace root:

```bash
rtk go run ./cmd/cachesim --root /tmp/cache-sim
rtk go run ./cmd/cachesim --root /tmp/cache-sim --profile large
```

Behavior:

- `--root` must point to a nonexistent or empty directory
- default profile is `medium`
- `large` must be selected explicitly
- writes go through the real engine and generate `meta/`, `wal/`, and `segments/`

Typical output:

```text
root=/tmp/cache-sim profile=medium gateways=20 points_per_gateway=2000 rounds=5 records=200000 payload_bytes=24 batch_size=1000 total_duration=... avg_batch_duration=... max_batch_duration=... segment_count=... wal_file_count=1 workspace_bytes=...
```

Use `cachebench` for synthetic benchmark-contract checks and `cachesim` for workspace-level load generation.

## Testing

Test layout:

- `tests/unit`
- `tests/integration`
- `tests/benchmark`

Recommended verification commands:

```bash
rtk go test ./... -count=1
rtk go test ./... -race -count=1
rtk go test ./tests/benchmark/... -count=1
```

## Operational guarantees

- WAL is not truncated before recovered data is durable in the segment.
- Replay cursor files are CRC-protected and persisted as main + backup.
- Tail repair is conservative: keep valid prefix, truncate invalid tail.
- Recovery rejects write-sequence conflicts instead of guessing.
- Replay order follows write order even if event timestamps move backward.

## Limits and non-goals

- Not safe for multi-process concurrent writers.
- Not optimized for ad hoc point filtering or arbitrary analytical queries.
- Not a replicated storage engine.
- Not a replacement for a long-term primary TSDB or historian.
- No real power-loss black-box harness yet; current crash verification is process-level plus recovery tests.

## Capacity planning note

For the confirmed workload assumptions used by the project:

- Average: `4000 records/s`
- Peak: `10000 records/s`
- Retention: `30 days`
- Example payload estimate: `50 bytes/record`

That implies roughly:

- Average payload volume: `518.4 GB`
- Peak payload volume: `1.296 TB`

Practical disk sizing must include WAL, segment headers/footers, slack, file system overhead, and safety margin. The project tests currently enforce the planning rule that usable disk should be materially above raw payload volume.

## Module path note

The current `go.mod` module path is `fastReadFile`. If you intend to publish or consume this repository directly from GitHub, align the module path with your repository path before external use.
