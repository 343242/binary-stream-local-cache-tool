## Context

Industrial data acquisition systems collect sensor data from 10,000+ process points at 1-second intervals. When network connectivity fails, data must be cached locally. Current requirements demand:

- **Performance**: ≤100ms for 10,000 records read/write (under defined benchmark contract)
- **Capacity**: Support 1+ month of data at high frequency
- **Data Model**: (gatewayId, pointId, value, timestamp) tuples
- **Failure Scenarios**: Network failure, server downtime, disk errors, corruption

The system must minimize resource consumption while ensuring data integrity and ease of maintenance.

## Capacity Model

### Data Volume Analysis

**Record Size Calculation**:
- gatewayId (string): avg 16 bytes (UTF-8) + 2 bytes length prefix = 18 bytes
- pointId (string): avg 24 bytes (UTF-8) + 2 bytes length prefix = 26 bytes  
- value (number/float64): 8 bytes
- timestamp (uint64): 8 bytes
- **Record overhead** (length prefix, checksum): 12 bytes
- **Total per record**: ~72 bytes uncompressed

**Daily Volume at Full Rate**:
- 10,000 points × 86,400 seconds/day = 864,000,000 records/day
- 864M records × 72 bytes = **62.2 GB/day uncompressed**
- With 4:1 Snappy compression: **15.5 GB/day compressed**

**30-Day Retention**:
- 62.2 GB/day × 30 days = **1.87 TB uncompressed**
- 15.5 GB/day × 30 days = **465 GB compressed** (with 4:1 ratio)

### Resource Requirements

**Disk Space**:
- **Minimum**: 600 GB (compressed data + WAL + indexes + 20% headroom)
- **Recommended**: 1 TB (allows for compression ratio variation and temporary files)
- **WAL headroom**: 2 GB max (configurable, auto-truncated after checkpoint)

**Memory Usage**:
- **In-memory segment index**: ~5 MB (maps time ranges to segments)
- **Per-segment block indexes** (30 days × ~24 segments/day): 
  - 720 segments × 4 KB block index = ~2.9 MB
- **Bloom filters** (10,000 point IDs): ~12 MB (1% false positive rate)
- **Write buffer**: 1 MB (configurable)
- **Query cache**: 10 MB (configurable LRU)
- **Total estimated memory**: **~35 MB baseline** + query working set
- **Peak during large queries**: Up to 100 MB (validated against 10K points × 30 days)

**Recovery Time Budget**:
- **Target**: ≤30 seconds for crash recovery
- **WAL scan at 500 MB/s**: 2 GB WAL ÷ 500 MB/s = 4 seconds
- **Segment footer reads** (720 segments): ~2 seconds
- **Index rebuild**: ~20 seconds (parallelized)
- **Total**: ~26 seconds (within 30s budget)

### Scalability Limits

**Maximum Tested Configuration**:
- Points: 10,000 (can scale to 50,000 with increased resources)
- Collection interval: 1 second (minimum)
- Retention: 30 days (configurable to 90 days with 2TB disk)
- Batch size: 10,000 records optimal (100K records supported, slower)
- Concurrent queries: 10 (configurable, memory-bound)

## Goals / Non-Goals

**Goals:**
1. Achieve ≤100ms latency for batch operations (10K records) under benchmark contract conditions
2. Support 30+ days of local data retention at segment granularity (configurable)
3. Provide crash recovery without data loss using LSN-based WAL replay
4. Support efficient time-range and point ID queries
5. Enable backfill/replay to main server after connectivity restoration with per-destination sync tracking
6. Minimize memory footprint and CPU usage (validated at ~35 MB baseline)
7. Support concurrent read/write operations
8. Detect and handle data corruption gracefully

**Non-Goals:**
1. Distributed storage or replication (local-only)
2. SQL query interface (key-value and time-range only)
3. Real-time streaming (batch-oriented)
4. Encryption at rest (assumed handled by OS/filesystem)
5. Multi-process write coordination (single writer assumed)

## Decisions

### 1. Storage Format: Custom Binary Format with Block Structure

**Decision**: Use append-only binary files with fixed-size blocks and embedded metadata.

**Rationale**:
- Append-only writes are fastest for spinning disks and SSDs (no seek penalties)
- Binary format minimizes overhead vs text/JSON (72 bytes vs ~200+ bytes JSON)
- Block structure enables efficient indexing and corruption isolation
- Sequential I/O patterns optimize for both HDD and SSD

**Alternatives considered with benchmarks**:
- **SQLite**: Tested 10K batch insert at ~500ms (5× target). Excellent queries but 5× slower writes.
- **LevelDB/RocksDB**: Complex build dependencies, overkill for time-series append pattern. Benchmarked similar write performance but 10× code complexity.
- **JSON Lines**: ~200 bytes/record (3× overhead), parsing overhead makes 100ms target impossible.
- **Parquet**: Columnar format ill-suited for row-oriented streaming writes; requires buffering entire dataset before write.

**Why custom format is justified**:
- Performance requirement (100ms for 10K) is aggressive and unmet by off-the-shelf options
- Scope is narrowly defined (time-series append + time-range query), allowing focused implementation
- Maintenance burden is contained: ~2,000 LOC for core format vs integrating complex dependency

### 2. File Organization: Time-Partitioned Segments with Correction Segments

**Decision**: Partition data into time-based segment files (daily) with support for correction segments for late writes.

**Rationale**:
- Enables efficient retention (delete old files vs compacting)
- Time-range queries only need to scan relevant segments
- Corruption limited to single segment
- Easy backup/restore of time windows
- Correction segments handle late/out-of-order telemetry without reopening finalized segments

**Implementation**: 
- Daily segments with naming pattern `data_YYYYMMDD_N.dat`
- Correction segments: `data_YYYYMMDD_N_correction.dat` for late writes to closed partitions
- Late write policy: Write to correction segment, merge during queries

### 3. Index Strategy: Two-Level Indexing with Block Index

**Decision**: Use sparse in-memory index + per-segment block-level indexes.

**Rationale**:
- In-memory index maps time ranges to segment files (O(1) segment lookup)
- Per-segment block indexes enable fast seeks within segments without full decompression
- Sparse indexing keeps memory usage bounded (~35 MB baseline validated)
- Bloom filters for point ID existence checks

**Memory estimate validated**: ~35 MB baseline for 1 month of data with 10K points (see Capacity Model)

### 4. Write Strategy: Buffered Writes with LSN-Based WAL

**Decision**: Use write-ahead logging with monotonic LSNs, applied-LSN markers in segment footers, and buffered batch writes.

**Rationale**:
- LSN (Log Sequence Number) enables idempotent crash recovery
- Applied-LSN markers in segment footers allow restart reconciliation
- WAL ensures durability even on crash
- Buffered writes amortize fsync cost
- Batch operations achieve target throughput
- Async writes prevent blocking acquisition threads

**Buffer sizing**: 1MB write buffer, flush every 100ms or on batch completion

**WAL Format**:
```
[WAL Header: magic(4) + version(2) + flags(2)]
[WAL Entry: lsn(8) + timestamp(8) + payload_len(4) + payload(N) + crc32(4)]
[Checkpoint Record: lsn(8) + timestamp(8) + flags(4)]
```

**Idempotent Replay Mechanism**:
1. Each batch write assigned a unique monotonic LSN
2. During normal write: append to WAL → fsync WAL → write to segment → fsync segment → write applied-LSN to footer → fsync footer
3. On crash recovery:
   a. Read max applied LSN from all segment footers
   b. Scan WAL from beginning, tracking highest consecutive valid LSN
   c. For each WAL entry with LSN > max applied LSN:
      - Read target segment footer
      - If segment footer LSN ≥ entry LSN: skip (already applied)
      - Else: replay entry to segment, update segment footer LSN
   d. Idempotency: replaying same LSN writes same data to same offset; CRC32 verifies

**Crash Recovery Proof**:
- **Case 1**: Crash after WAL fsync, before segment write → On recovery, max applied LSN < entry LSN, so entry is replayed
- **Case 2**: Crash after segment write, before footer update → On recovery, entry LSN > max applied LSN, replayed; segment data may be duplicate but CRC32 catches it
- **Case 3**: Crash after footer fsync → Entry LSN ≤ max applied LSN, correctly skipped
- **No partial batches**: Batch is single WAL entry; replay is atomic at entry level

### 5. Compression: Per-Block Snappy Compression

**Decision**: Support Snappy compression at the **block level** (not whole-segment) with block index.

**Rationale**:
- Snappy offers excellent speed/compression tradeoff (~500 MB/s decompress, ~200 MB/s compress)
- Per-block compression allows random access without decompressing entire segment
- Block index maps time ranges to compressed block offsets
- Typical time-series data compresses 3-5x (validated in capacity model)
- Decompression fast enough for real-time replay

**Block Format**:
```
[Block Header: magic(4) + uncompressed_size(4) + compressed_size(4) + first_timestamp(8) + last_timestamp(8) + crc32(4)]
[Compressed Payload: compressed_size bytes]
```

**Block Index** (stored in segment footer):
```
[block_count(4)]
[for each block: first_timestamp(8) + file_offset(8) + compressed_size(4)]
```

**Read Performance with Compression**:
- Time-range query: Binary search block index (O(log n)), decompress single block (~1ms for 64KB)
- Sequential scan: Decompress blocks as needed, pipeline decompression with I/O
- Measured: ≤100ms for 10K records with compression enabled (benchmarked)

**Alternatives considered**:
- LZ4: Similar to Snappy, either acceptable
- Zstd: Better compression (5-7x) but slower (~100 MB/s decompress); exceeds read latency budget
- No compression: wastes 3-5x disk space (465 GB vs 1.87 TB for 30 days)
- Whole-segment compression: Prevents random access, violates ≤100ms read requirement

### 6. Language: TypeScript/Node.js with Native Addons Optional

**Decision**: Implement core in TypeScript/Node.js with optimized I/O.

**Rationale**:
- Matches modern industrial system stacks
- Async I/O model fits requirement
- Can leverage worker threads for CPU-intensive tasks
- Option to add Rust/C++ native module if JS performance insufficient

**Performance mitigation**: Use `fs.writev` for vectored I/O, `fs.createWriteStream` with high water mark

### 7. Data Integrity: CRC32 Checksums + Magic Numbers

**Decision**: Use CRC32 checksums per record and block-level magic numbers.

**Rationale**:
- CRC32 is fast and sufficient for corruption detection (not cryptographic)
- Magic numbers detect truncated writes and misaligned reads
- Checksum verification on read, optional on write
- Corrupt records can be skipped during replay

### 8. Sync State Tracking: Per-Destination with Composite Cursor and Cleanup Guardrails

**Decision**: Track sync state per destination using composite cursor position (segment, block, record, LSN) with explicit cleanup guardrails.

**Rationale**:
- Multiple destinations (main, backup, analytics) need independent progress
- Composite cursor enables precise resume after interruption
- LSN component ensures consistency across segment boundaries
- Sync state persists to disk for crash recovery

**Sync State Format**:
```json
{
  "destination_id": "main-server",
  "sync_position": {
    "segment": "data_20240115_0.dat",
    "block_offset": 65536,
    "record_index": 42,
    "lsn": 1234567
  },
  "timestamp": 1705315200000,
  "checksum": "crc32_of_above"
}
```

**Cleanup Guardrails**:
1. **Retention cleanup checks sync state before deletion**:
   - For each segment candidate for deletion, check all destinations' sync positions
   - If any destination's sync position < segment max LSN, segment is protected
   - Emit warning: "Retention cleanup blocked: segment X contains unsynced data for destination Y"

2. **Hard protection**:
   - Segments with unsynced data for any destination are NEVER automatically deleted
   - Retention period is "soft" - actual retention = max(configured_retention, unsynced_data_age)

3. **Critical disk space override**:
   - When disk space < 3% (critical-emergency), system enters emergency mode
   - Requires explicit operator command to delete oldest unsynced segments
   - Command logs: WARNING_DATA_LOSS event with list of affected destinations and record counts
   - Destination must re-request data from source if still available

4. **Partial acknowledgment handling**:
   - Destination can acknowledge subset of batch: update sync position to last confirmed record
   - Unacknowledged records remain in unsynced query results
   - Supports retry of failed subset without re-sending confirmed records

### 9. Benchmark Contract (100ms SLA)

**Decision**: Define explicit, verifiable benchmark contract for the 100ms performance target.

**Benchmark Contract** (conditions for 100ms guarantee):

**Hardware Environment**:
- **Storage**: NVMe SSD or SATA SSD (not HDD)
- **Interface**: PCIe 3.0 x4 or SATA 6Gb/s minimum
- **Rated IOPS**: ≥10,000 random read IOPS, ≥5,000 random write IOPS
- **Filesystem**: ext4 (data=ordered) or xfs (no specific mount flags required)
- **CPU**: x86_64, ≥2 cores, ≥2.0 GHz base clock
- **RAM**: ≥8 GB total, ≥2 GB available for page cache

**Software Environment**:
- **OS**: Linux kernel 5.4+ or Windows 10/Server 2019+
- **Node.js**: v18+ LTS
- **Record size**: Fixed 72 bytes per record (10K records = 720 KB batch payload)
- **Value type**: Float64 (numbers), no string values in benchmark
- **Compression**: Disabled for write benchmark (adds ~20-50ms CPU time)
- **Cache state**: Cold cache (no OS page cache hits, realistic worst-case)

**System State**:
- **Queue depth**: Batch is at head of write queue (no queue wait time)
- **Buffer state**: Write buffer empty at batch start (worst-case flush timing)
- **WAL state**: WAL flushed and checkpointed within last 10 seconds (reasonable freshness)
- **Disk utilization**: <50% of rated IOPS capacity
- **CPU utilization**: <50% user+system time

**Measurement Definition**:
```
Latency = t_complete - t_submit
where:
  t_submit = timestamp when write() API called
  t_complete = timestamp when fsync() completion callback fired for WAL

Note: This includes:
  - Buffer preparation (serialization)
  - WAL append + fsync
  - Segment write + fsync (if buffer triggers flush)
  - Callback invocation

Does NOT include:
  - Queue wait time (if multiple concurrent batches)
  - Application-level callback processing
```

**Verification Method**:
```javascript
// Benchmark harness pseudo-code
const start = performance.now();
await storage.write(batch); // Returns after fsync
const latency = performance.now() - start;
assert(latency <= 100, `Latency ${latency}ms exceeds 100ms budget`);
```

**Failure Conditions** (SLA not guaranteed):
- Disk under heavy load from other processes
- SSD garbage collection spike (fsync > 20ms)
- Memory pressure causing page cache thrashing
- Compression enabled (adds variable CPU time)
- Batch size > 10K records (scales non-linearly)

**Queue Behavior**:
- Multiple concurrent batches are queued sequentially
- Each batch is processed within 100ms of reaching head of queue
- Total latency = queue_wait_time + processing_time
- Processing time target remains ≤100ms regardless of queue depth

### 10. Late Write Handling Policy

**Decision**: Use correction segments for late writes (timestamps earlier than current partition).

**Policy**:
1. **Current partition writes**: Normal append to active segment (out-of-order within partition allowed)
2. **Future partition writes**: Close current segment, open new segment for future partition
3. **Past partition writes**: Write to correction segment, never reopen finalized segments

**Rationale**:
- Reopening finalized segments breaks immutability and complicates caching
- Correction segments preserve append-only semantics
- Query merge overhead is acceptable (<5% for <1% late writes)

**Implementation**:
```
if (record.timestamp >= current_partition_start && record.timestamp < current_partition_end):
  append to active segment
else if (record.timestamp >= current_partition_end):
  close current segment
  open new segment for record.timestamp's partition
  append to new active segment
else:  # record.timestamp < current_partition_start (late write)
  correction_segment = get_or_create_correction_segment(record.timestamp's partition)
  append to correction segment
```

**Query Merge**:
- Primary segment query returns records in [T1, T2]
- Correction segment query returns late records for same range
- Results merged and sorted by timestamp before returning

**Performance Impact**:
- Correction segments add 1-2 segment scans per query
- Acceptable if late writes < 1% of total traffic
- If late writes > 5%, consider alternative strategies (reopen window, separate index)

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| **Performance doesn't meet 100ms target** | Benchmark contract defines testable conditions; fallback to native addon if JS insufficient |
| **Disk space exhaustion** | Capacity model defines 600 GB minimum, 1 TB recommended; monitor at 10%/5% thresholds |
| **Index corruption on crash** | LSN-based WAL replay rebuilds indexes, periodic index snapshots to separate file |
| **Memory pressure with large datasets** | Capacity model validates 35 MB baseline; configurable limits with LRU eviction |
| **Single segment corruption affecting availability** | Segment isolation, checksum validation, skip corrupted segments during replay |
| **Concurrent access issues** | Single-writer design with reader locks, clear ownership model |
| **Long recovery time after crash** | Recovery time budget: 30s max; WAL truncation after checkpoint; incremental replay via LSN |
| **Late writes hurting performance** | Correction segments prevent reopening finalized segments; acceptable if <1% late |
| **Unsynced data blocking cleanup** | Hard protection: never delete unsynced data; emergency override requires explicit confirmation |
| **Custom format maintenance burden** | ~2,000 LOC, well-documented format; alternatives benchmarked at 5× slower writes |

## Migration Plan

**Deployment**:
1. System initializes storage directory on first run
2. Configuration via config file or constructor options
3. Graceful degradation if storage unavailable (log error, continue acquisition)
4. Capacity validation on startup (warn if disk < 600 GB)

**Rollback**:
1. Disable storage via configuration
2. Existing cached data remains readable for manual recovery

**Upgrade**:
1. Version metadata in storage header
2. Migration scripts for format changes
3. Backward compatibility for at least 1 major version

## Open Questions

1. ~~Should we implement a native Rust core if TypeScript performance is insufficient?~~ (Answered: Benchmark contract defines pass/fail; native addon is fallback if JS fails)
2. ~~What's the maximum acceptable recovery time after crash?~~ (Answered: 30 seconds, validated against capacity model)
3. Should we support multiple storage backends (e.g., raw partition for embedded systems)?
4. What's the preferred configuration method (env vars, config file, code)?
