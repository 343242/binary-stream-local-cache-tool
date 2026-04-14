# Go Binary Stream Local Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go local file cache library and CLI that stores raw binary payloads with ordered append-only writes, size-rotated segment files, WAL-backed recovery, replay cursors, retention, and operational tooling.

**Architecture:** The implementation is a single-writer storage engine with a narrow public API in `pkg/cache`, backed by internal packages for record/block encoding, WAL, segments, replay cursors, recovery, retention, stats, and CLI inspection tooling. The write path is `WriteBatch -> WAL fsync -> Segment append -> fsync policy -> checkpoint`; the replay path is `cursor -> segment scan -> ack -> dual cursor persist`.

**Tech Stack:** Go 1.22+, standard library only for v1 (`os`, `io`, `bufio`, `encoding/binary`, `hash/crc32`, `context`, `errors`, `filepath`, `syscall`, `flag`, `testing`)

---

## Preconditions

- Current workspace is not a git repository. If commit checkpoints are desired during execution, run `git init` before starting Task 1.
- Keep v1 dependency-free. Do not add Cobra, zap, Prometheus clients, mmap libraries, or third-party WAL/index packages.
- Implement TDD per task. No production code before the targeted failing test exists.

## File Structure

Create this structure and keep boundaries stable:

- `go.mod`
- `pkg/cache/types.go`
- `pkg/cache/config.go`
- `pkg/cache/engine.go`
- `pkg/cache/errors.go`
- `internal/codec/record.go`
- `internal/codec/block.go`
- `internal/wal/log.go`
- `internal/wal/checkpoint.go`
- `internal/segment/file.go`
- `internal/segment/footer.go`
- `internal/segment/manager.go`
- `internal/replay/cursor_store.go`
- `internal/replay/manager.go`
- `internal/recovery/recovery.go`
- `internal/retention/manager.go`
- `internal/stats/stats.go`
- `internal/fsutil/fsutil.go`
- `internal/ops/report.go`
- `cmd/cachectl/main.go`
- `cmd/cachebench/main.go`
- `tests/unit/*.go`
- `tests/integration/*.go`
- `tests/benchmark/*.go`

Boundary rules:

- `pkg/cache` contains only public API, public config, and thin engine orchestration.
- `internal/codec` only knows binary layouts for records and blocks.
- `internal/wal` only knows WAL file and checkpoint persistence.
- `internal/segment` only knows `.seg` files, footer validation, ID allocation, and segment fsync/rotation.
- `internal/replay` only knows cursor loading/persisting and ordered replay reads.
- `internal/recovery` owns startup recovery, `writeSeq` bootstrap, and WAL-vs-segment arbitration.
- `internal/retention` owns cleanup eligibility and disk-watermark checks.
- `internal/stats` owns counters, histograms, and `StatsSnapshot`.
- `cmd/cachectl` uses public API plus read-only internals for ops commands. It must not bypass on-disk validation.

## Build and Test Commands

Use these exact commands during implementation:

- `rtk go test ./tests/unit/... -v`
- `rtk go test ./tests/integration/... -v`
- `rtk go test ./...`
- `rtk go test ./tests/benchmark/... -run TestBenchmarkContract -v`
- `rtk go test ./... -race`

If `rtk` output is insufficient during debugging, use raw `go test` temporarily for the failing package only.

## Task 1: Bootstrap Go Module and Public Skeleton

**Files:**
- Create: `go.mod`
- Create: `pkg/cache/types.go`
- Create: `pkg/cache/config.go`
- Create: `pkg/cache/errors.go`
- Create: `pkg/cache/engine.go`
- Test: `tests/unit/public_api_test.go`

- [ ] **Step 1: Write the failing public API contract test**

Create `tests/unit/public_api_test.go` with compile-time and behavior-level expectations:

```go
package unit

import (
	"context"
	"testing"

	"fastReadFile/pkg/cache"
)

func TestPublicAPIContractsExist(t *testing.T) {
	cfg := cache.Config{RootDir: t.TempDir()}
	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	_, err = engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1, Payload: []byte{0x01}},
	})
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `rtk go test ./tests/unit/... -run TestPublicAPIContractsExist -v`

Expected: compile failure because `cache.Config`, `cache.Open`, `cache.RawRecord`, and `WriteBatch` do not exist.

- [ ] **Step 3: Write minimal public package skeleton**

Create `go.mod`:

```go
module fastReadFile

go 1.22
```

Create `pkg/cache/types.go` with the public structs from the spec:

```go
package cache

type RawRecord struct {
	EventTimeUnixMs int64
	Payload         []byte
}

type ReplayLimit struct {
	MaxRecords int
	MaxBytes   int64
}

type ReplayRecord struct {
	EventTimeUnixMs int64
	WriteSeq        uint64
	Payload         []byte
}

type ReplayCursor struct {
	Version         uint32
	SegmentID       uint64
	BlockOffset     uint64
	RecordIndex     uint32
	WriteSeq        uint64
	UpdatedAtUnixMs int64
	CRC32           uint32
}

type ReplayBatch struct {
	Destination       string
	SegmentID         uint64
	NextCursor        ReplayCursor
	Records           []ReplayRecord
	RecordCount       int
	TotalPayloadBytes int64
}

type WriteBatchResult struct {
	BatchSeq            uint64
	FirstWriteSeq       uint64
	LastWriteSeq        uint64
	RecordCount         int
	SegmentID           uint64
	WALBytesWritten     uint64
	SegmentBytesWritten uint64
}

type AckResult struct {
	Destination     string
	AppliedCursor   ReplayCursor
	PreviousWriteSeq uint64
	CurrentWriteSeq  uint64
}
```

Create `pkg/cache/config.go`:

```go
package cache

import "time"

type Config struct {
	RootDir                 string
	SegmentTargetSizeBytes  int64
	SegmentSlackSizeBytes   int64
	BlockTargetSizeBytes    int64
	CheckpointInterval      time.Duration
	CheckpointBytes         int64
	SegmentFsyncInterval    time.Duration
	SegmentFsyncBytes       int64
	RetentionDays           int
}
```

Create `pkg/cache/errors.go` and `pkg/cache/engine.go` with the spec error codes and stub methods returning `ErrRecoveryRequired`.

- [ ] **Step 4: Run test to verify it passes compile and basic open/write flow is still failing for the right reason**

Run: `rtk go test ./tests/unit/... -run TestPublicAPIContractsExist -v`

Expected: compile succeeds, runtime fails with `ErrRecoveryRequired` or equivalent stub error until later tasks wire internals.

- [ ] **Step 5: Add a second test that default config values are sane**

Add to `tests/unit/public_api_test.go`:

```go
func TestDefaultConfigIsStable(t *testing.T) {
	cfg := cache.DefaultConfig(t.TempDir())
	if cfg.BlockTargetSizeBytes != 1<<20 {
		t.Fatalf("BlockTargetSizeBytes = %d, want %d", cfg.BlockTargetSizeBytes, 1<<20)
	}
	if cfg.CheckpointBytes != 64<<20 {
		t.Fatalf("CheckpointBytes = %d, want %d", cfg.CheckpointBytes, 64<<20)
	}
}
```

- [ ] **Step 6: Implement `DefaultConfig()` and rerun tests**

Run: `rtk go test ./tests/unit/... -v`

Expected: PASS for public API and default config tests.

## Task 2: Implement Error Model and Validation Rules

**Files:**
- Modify: `pkg/cache/errors.go`
- Modify: `pkg/cache/engine.go`
- Test: `tests/unit/errors_test.go`

- [ ] **Step 1: Write failing tests for typed errors**

Create `tests/unit/errors_test.go`:

```go
package unit

import (
	"errors"
	"testing"

	"fastReadFile/pkg/cache"
)

func TestStorageErrorSupportsErrorsIs(t *testing.T) {
	err := cache.NewError(cache.ErrDiskFull, "write", "/tmp/demo", "disk full", nil)
	if !errors.Is(err, cache.ErrCode(cache.ErrDiskFull)) {
		t.Fatalf("errors.Is should match error code")
	}
}

func TestWriteBatchRejectsEmptyPayload(t *testing.T) {
	cfg := cache.DefaultConfig(t.TempDir())
	engine, err := cache.Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer engine.Close()

	_, err = engine.WriteBatch(t.Context(), []cache.RawRecord{{EventTimeUnixMs: 1}})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `rtk go test ./tests/unit/... -run 'TestStorageErrorSupportsErrorsIs|TestWriteBatchRejectsEmptyPayload' -v`

Expected: compile or assertion failures because typed errors and validation are incomplete.

- [ ] **Step 3: Implement `StorageError`, `ErrorCode`, and request validation**

Requirements:

- `StorageError` fields: `Code`, `Op`, `Path`, `Message`, `Cause`
- `Error()` prints `Code`, `Op`, and `Message`
- `Unwrap()` returns `Cause`
- `ErrCode(code ErrorCode) error` returns a sentinel for `errors.Is`
- `WriteBatch` rejects:
  - empty batch
  - empty payload
  - negative `EventTimeUnixMs`
  - payload over `16MB`

- [ ] **Step 4: Rerun unit tests**

Run: `rtk go test ./tests/unit/... -v`

Expected: PASS for error-code and validation tests.

## Task 3: Build Binary Record and Block Codec

**Files:**
- Create: `internal/codec/record.go`
- Create: `internal/codec/block.go`
- Test: `tests/unit/codec_test.go`

- [ ] **Step 1: Write failing round-trip tests for record encode/decode**

Cover:

- record round trip keeps `EventTimeUnixMs`, `WriteSeq`, and payload intact
- CRC mismatch returns corruption error
- block round trip preserves `RecordCount`, `FirstWriteSeq`, `LastWriteSeq`

- [ ] **Step 2: Run codec tests to verify failure**

Run: `rtk go test ./tests/unit/... -run Codec -v`

Expected: package or symbol missing failures.

- [ ] **Step 3: Implement record codec**

Implement:

- `type EncodedRecord struct`
- `func EncodeRecord(writeSeq uint64, input cache.RawRecord) ([]byte, error)`
- `func DecodeRecord(buf []byte) (DecodedRecord, error)`

Binary layout must match the spec:

- magic(2)
- version(2)
- event time(8)
- write seq(8)
- payload len(4)
- header CRC32(4)
- payload CRC32(4)
- payload(N)

- [ ] **Step 4: Implement block codec**

Implement:

- `func BuildBlock(records [][]byte, firstSeq, lastSeq uint64, minEventTime, maxEventTime int64) ([]byte, error)`
- `func ParseBlock(buf []byte) (DecodedBlock, error)`

Enforce:

- max block target default `1MB`
- single record may exceed block target but must remain a valid single-record block

- [ ] **Step 5: Rerun codec tests**

Run: `rtk go test ./tests/unit/... -run Codec -v`

Expected: PASS.

## Task 4: Implement WAL File and Checkpoint Store

**Files:**
- Create: `internal/wal/log.go`
- Create: `internal/wal/checkpoint.go`
- Test: `tests/unit/wal_test.go`

- [ ] **Step 1: Write failing tests for WAL append/read/truncate**

Cover:

- append block bytes with `batchSeq`
- recover all valid entries after restart
- truncate corrupted tail
- persist/load checkpoint

- [ ] **Step 2: Run WAL tests to verify failure**

Run: `rtk go test ./tests/unit/... -run WAL -v`

Expected: missing implementation failures.

- [ ] **Step 3: Implement `WALManager`**

Required methods:

- `Open(root string) (*Log, error)`
- `Append(batchSeq uint64, block []byte) (EntryMeta, error)`
- `Scan() ([]Entry, error)`
- `TruncateAfter(offset int64) error`
- `Sync() error`

Checkpoint store methods:

- `Load() (Checkpoint, error)`
- `Save(Checkpoint) error`

WAL policy to encode:

- fsync every append
- checkpoint threshold `64MB` or `5s`
- force checkpoint on segment seal and shutdown

- [ ] **Step 4: Rerun WAL tests**

Run: `rtk go test ./tests/unit/... -run WAL -v`

Expected: PASS.

## Task 5: Implement Segment Footer, Segment File, and Rotation

**Files:**
- Create: `internal/segment/footer.go`
- Create: `internal/segment/file.go`
- Create: `internal/segment/manager.go`
- Test: `tests/unit/segment_test.go`

- [ ] **Step 1: Write failing tests for segment footer and rotation**

Cover:

- new empty system starts with segment `1`
- recovered system chooses `max(validSegmentID)+1`
- seal writes footer with CRC
- footer validation rejects corrupt footer
- rotation occurs when next block exceeds `target + slack`

- [ ] **Step 2: Run segment tests to verify failure**

Run: `rtk go test ./tests/unit/... -run Segment -v`

Expected: missing file/footer/manager symbols.

- [ ] **Step 3: Implement footer serialization and validation**

Footer must contain:

- `segment_id`
- `created_at`
- `sealed_at`
- `record_count`
- `first_write_seq`
- `last_write_seq`
- `min_event_time`
- `max_event_time`
- `last_batch_seq`
- `footer_crc32`

Implement `ValidateFooter()` with all invalidity checks from the spec.

- [ ] **Step 4: Implement segment append and fsync policy**

Encode:

- append only
- fsync after `4MB` or `100ms`
- force fsync on seal and shutdown
- reject appends if file system became read-only

- [ ] **Step 5: Rerun segment tests**

Run: `rtk go test ./tests/unit/... -run Segment -v`

Expected: PASS.

## Task 6: Implement Replay Cursor Store and Ack Rules

**Files:**
- Create: `internal/replay/cursor_store.go`
- Test: `tests/unit/cursor_test.go`

- [ ] **Step 1: Write failing tests for cursor persistence**

Cover:

- write main cursor and backup cursor atomically
- load main cursor when valid
- fall back to backup when main is corrupt
- reject backwards ack
- reject cursor CRC mismatch

- [ ] **Step 2: Run cursor tests to verify failure**

Run: `rtk go test ./tests/unit/... -run Cursor -v`

Expected: missing store functions or wrong behavior.

- [ ] **Step 3: Implement `CursorStore`**

Methods:

- `Load(destination string) (cache.ReplayCursor, error)`
- `Save(destination string, cursor cache.ReplayCursor) error`
- `Save` must:
  - write `.tmp`
  - fsync `.tmp`
  - rename to main cursor
  - fsync parent dir
  - rebuild `.bak` from new main content

- [ ] **Step 4: Implement cursor ordering checks**

Rules:

- only forward movement
- same `WriteSeq` is idempotent no-op
- lower `WriteSeq` returns `ERR_CURSOR_INVALID`
- corrupt on-disk cursor returns `ERR_CURSOR_CORRUPTED`

- [ ] **Step 5: Rerun cursor tests**

Run: `rtk go test ./tests/unit/... -run Cursor -v`

Expected: PASS.

## Task 7: Implement Recovery Bootstrap and Sequence Arbitration

**Files:**
- Create: `internal/recovery/recovery.go`
- Modify: `internal/wal/log.go`
- Modify: `internal/segment/manager.go`
- Test: `tests/integration/recovery_test.go`

- [ ] **Step 1: Write failing recovery integration tests**

Cover:

- restart resumes `nextWriteSeq` from recovered global maximum + 1
- WAL-only batch is replayed into segment
- partial segment block is truncated and replayed
- same `writeSeq` with different payload fails recovery with `ERR_SEQUENCE_CONFLICT`
- orphan tail beyond valid footer is truncated

- [ ] **Step 2: Run recovery tests to verify failure**

Run: `rtk go test ./tests/integration/... -run Recovery -v`

Expected: recovery pipeline failures.

- [ ] **Step 3: Implement `Recover()` workflow**

Required stages:

- scan segment directory
- load valid footers
- discover `maxSegmentWriteSeq`
- scan WAL and last valid `batchSeq`
- truncate invalid WAL tail
- apply WAL-vs-segment arbitration
- recompute `nextWriteSeq`
- reopen active segment

- [ ] **Step 4: Rerun recovery tests**

Run: `rtk go test ./tests/integration/... -run Recovery -v`

Expected: PASS.

## Task 8: Implement Replay Manager and Ordered Read Path

**Files:**
- Create: `internal/replay/manager.go`
- Modify: `pkg/cache/engine.go`
- Test: `tests/integration/replay_test.go`

- [ ] **Step 1: Write failing replay tests**

Cover:

- replay returns records strictly in physical order
- time rollback does not alter replay order
- replay respects `MaxRecords`
- replay respects `MaxBytes`
- `Ack` advances cursor and next `Replay` starts after last acknowledged record

- [ ] **Step 2: Run replay tests to verify failure**

Run: `rtk go test ./tests/integration/... -run Replay -v`

Expected: replay not implemented or wrong order.

- [ ] **Step 3: Implement replay scan**

Requirements:

- start from cursor if present
- otherwise start from earliest retained segment
- scan segment files in ascending `segment_id`
- scan blocks in file order
- decode records in block order
- return `ReplayBatch.NextCursor` pointing at the last emitted record

- [ ] **Step 4: Implement `Ack` in engine**

Requirements:

- validate cursor came from current replay domain
- persist main and backup cursor
- make repeated ack of same cursor a no-op success

- [ ] **Step 5: Rerun replay tests**

Run: `rtk go test ./tests/integration/... -run Replay -v`

Expected: PASS.

## Task 9: Implement Retention, Disk Guards, and Read-Only Handling

**Files:**
- Create: `internal/retention/manager.go`
- Create: `internal/fsutil/fsutil.go`
- Modify: `pkg/cache/engine.go`
- Test: `tests/integration/retention_test.go`

- [ ] **Step 1: Write failing retention and disk-state tests**

Cover:

- segments newer than configured retention are kept
- segments with unacked data are kept even if over retention
- actual retention uses `max(configuredRetentionDays, oldestUnackedSegmentAgeDays)`
- disk read-only state disables writes and preserves replay
- disk full state returns `ERR_DISK_FULL`

- [ ] **Step 2: Run tests to verify failure**

Run: `rtk go test ./tests/integration/... -run 'Retention|Disk' -v`

Expected: missing retention and disk guard behaviors.

- [ ] **Step 3: Implement retention and disk-watermark logic**

Requirements:

- configurable min/max/default retention validation
- delete only sealed, acked, expired segments
- reject new writes under reject threshold
- classify `EROFS` as `ERR_READ_ONLY_FS`
- classify other write failures as `ERR_IO`

- [ ] **Step 4: Rerun integration tests**

Run: `rtk go test ./tests/integration/... -run 'Retention|Disk' -v`

Expected: PASS.

## Task 10: Implement Stats Snapshot and Engine Orchestration

**Files:**
- Create: `internal/stats/stats.go`
- Modify: `pkg/cache/engine.go`
- Test: `tests/unit/stats_test.go`
- Test: `tests/integration/engine_lifecycle_test.go`

- [ ] **Step 1: Write failing tests for stats and lifecycle**

Cover:

- `Stats()` returns defined counters after writes/replays
- `Close()` flushes WAL, segment, checkpoint, and cursor state
- `Shutdown(ctx)` returns timeout when context expires
- recovery after ungraceful stop increments `ungraceful_shutdown_recoveries_total`

- [ ] **Step 2: Run tests to verify failure**

Run: `rtk go test ./tests/unit/... -run Stats -v`

Run: `rtk go test ./tests/integration/... -run Lifecycle -v`

Expected: missing stats snapshot and lifecycle plumbing.

- [ ] **Step 3: Implement `StatsSnapshot` and lifecycle orchestration**

Include these public groups:

- `CapacityStats`
- `IOStats`
- `ReplayStats`
- `HealthStats`

Wire counters for:

- writes
- replays
- checkpoints
- segment fsyncs
- cursor corruption
- shutdown counts
- rollback counts

- [ ] **Step 4: Rerun stats and lifecycle tests**

Run: `rtk go test ./tests/unit/... -run Stats -v`

Run: `rtk go test ./tests/integration/... -run Lifecycle -v`

Expected: PASS.

## Task 11: Build `cachectl` Operational CLI

**Files:**
- Create: `internal/ops/report.go`
- Create: `cmd/cachectl/main.go`
- Test: `tests/integration/cachectl_test.go`

- [ ] **Step 1: Write failing CLI tests**

Cover subcommands:

- `stats`
- `inspect-segment`
- `inspect-wal`
- `inspect-cursor`
- `verify`
- `repair-tail`
- `close-check`

- [ ] **Step 2: Run CLI tests to verify failure**

Run: `rtk go test ./tests/integration/... -run Cachectl -v`

Expected: command binary missing or unsupported subcommands.

- [ ] **Step 3: Implement CLI with standard library `flag`**

Requirements:

- no Cobra
- support text and JSON output
- exit non-zero on corruption or invalid args
- `repair-tail` must be explicit and never run implicitly

- [ ] **Step 4: Rerun CLI tests**

Run: `rtk go test ./tests/integration/... -run Cachectl -v`

Expected: PASS.

## Task 12: Add Benchmark Harness and Contract Tests

**Files:**
- Create: `cmd/cachebench/main.go`
- Test: `tests/benchmark/benchmark_contract_test.go`

- [ ] **Step 1: Write failing benchmark contract test**

Create `tests/benchmark/benchmark_contract_test.go` that writes and replays `10000` records with fixed-size payloads and reports elapsed times.

Acceptance rules:

- always print elapsed write and replay durations
- fail hard only if runtime exceeds a relaxed CI threshold such as `500ms`
- document that `<100ms` is a target measured on recommended hardware, not a CI guarantee

- [ ] **Step 2: Run benchmark test to verify failure**

Run: `rtk go test ./tests/benchmark/... -run TestBenchmarkContract -v`

Expected: benchmark harness missing.

- [ ] **Step 3: Implement benchmark harness**

Requirements:

- generate deterministic `RawRecord` payloads
- run one warmup pass
- measure `WriteBatch`
- measure `Replay`
- print payload size, block size, record count, durations

- [ ] **Step 4: Rerun benchmark contract**

Run: `rtk go test ./tests/benchmark/... -run TestBenchmarkContract -v`

Expected: PASS with reported durations.

## Task 13: Final Verification Sweep

**Files:**
- Modify: none expected unless failures are found
- Test: all existing tests

- [ ] **Step 1: Run full unit suite**

Run: `rtk go test ./tests/unit/... -v`

Expected: PASS

- [ ] **Step 2: Run full integration suite**

Run: `rtk go test ./tests/integration/... -v`

Expected: PASS

- [ ] **Step 3: Run workspace test suite**

Run: `rtk go test ./...`

Expected: PASS

- [ ] **Step 4: Run race detector**

Run: `rtk go test ./... -race`

Expected: PASS

- [ ] **Step 5: Run benchmark contract**

Run: `rtk go test ./tests/benchmark/... -run TestBenchmarkContract -v`

Expected: PASS with durations printed

## Spec Coverage Check

- Public API, `writeSeq` bootstrap, `segment_id` bootstrap, and error model are covered by Tasks 1-2 and 7.
- Binary record format, block format, WAL format, checkpoint policy, and segment rotation are covered by Tasks 3-5.
- Replay cursor persistence, replay order, `Ack`, cursor recovery, and shutdown flushing are covered by Tasks 6, 8, and 10.
- Retention, disk full, read-only FS, stats, ops CLI, and benchmark tooling are covered by Tasks 9-12.
- No spec item remains intentionally unplanned for v1 scope.

## Notes for Execution

- Keep the engine single-writer. Do not add cross-process file locks in v1.
- Prefer tiny focused files over giant package files. If any file crosses roughly 300-400 lines, split it before adding more logic.
- Preserve the public API signatures in Task 1. If an implementation detail forces signature changes, stop and revise the spec first.
- Because this workspace is not a git repo, execution should either initialize git first or skip commit checkpoints.
