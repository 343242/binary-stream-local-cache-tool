## 1. Project Setup and Infrastructure

- [ ] 1.1 Initialize Node.js/TypeScript project structure
- [ ] 1.2 Set up project dependencies (typescript, jest for testing, @types/node)
- [ ] 1.3 Install Snappy compression library (`snappy` npm package)
- [ ] 1.4 Configure TypeScript compiler options (tsconfig.json)
- [ ] 1.5 Set up test framework and directory structure
- [ ] 1.6 Create project directory structure (src/, tests/, benchmarks/)

## 2. Core Data Types and Interfaces

- [ ] 2.1 Define `DataRecord` interface (gatewayId, pointId, value, timestamp)
- [ ] 2.2 Define `StorageConfig` interface (retentionDays, maxDiskUsage, compressionEnabled, etc.)
- [ ] 2.3 Define `WriteResult` and `ReadResult` types
- [ ] 2.4 Define `SegmentMetadata` interface for segment file headers and footers (including applied LSN)
- [ ] 2.5 Define error types (CorruptionError, DiskSpaceError, ValidationError)
- [ ] 2.6 Define stream and cursor interfaces for async operations
- [ ] 2.7 Define `SyncPosition` composite cursor type (segment, blockOffset, recordIndex, LSN)
- [ ] 2.8 Define `SyncState` interface for per-destination tracking

## 3. Binary Format and Serialization

- [ ] 3.1 Design binary record format (magic number, length, payload, checksum)
- [ ] 3.2 Implement `RecordSerializer` class for encoding/decoding records
- [ ] 3.3 Implement CRC32 checksum calculation
- [ ] 3.4 Design segment file header format (version, creation time, record count, compression info)
- [ ] 3.5 Design segment file footer format (applied LSN, block index)
- [ ] 3.6 Implement `SegmentHeader` and `SegmentFooter` serializer/deserializer
- [ ] 3.7 Write unit tests for serialization round-trips

## 4. Write-Ahead Logging (WAL) with LSN

- [ ] 4.1 Implement `WALEntry` format (LSN: uint64, timestamp, payload length, payload, checksum)
- [ ] 4.2 Create `WriteAheadLog` class with append-only writes
- [ ] 4.3 Implement monotonic LSN generation and tracking
- [ ] 4.4 Implement WAL fsync strategy for durability
- [ ] 4.5 Implement applied-LSN marker writing to segment footers
- [ ] 4.6 Implement restart reconciliation logic (max applied LSN detection)
- [ ] 4.7 Implement WAL replay logic for crash recovery with idempotent replay
- [ ] 4.8 Implement checkpoint and WAL truncation after successful commit
- [ ] 4.9 Write unit tests for WAL write/replay/truncate with LSN tracking

## 5. Block Management and Per-Block Compression

- [ ] 5.1 Design block format (magic, uncompressed size, compressed size, CRC32, payload)
- [ ] 5.2 Implement `Block` class for record batching
- [ ] 5.3 Implement block-level Snappy compression on write
- [ ] 5.4 Implement block-level Snappy decompression on read
- [ ] 5.5 Implement block index builder (maps time ranges to block offsets)
- [ ] 5.6 Write unit tests for block operations

## 6. Segment Management with Block Index

- [ ] 6.1 Implement `Segment` class representing a single segment file
- [ ] 6.2 Implement segment write operations (append blocks, update footer with applied LSN)
- [ ] 6.3 Implement segment read operations (scan, seek to block, random access via block index)
- [ ] 6.4 Implement correction segment support for late writes
- [ ] 6.5 Implement segment footer with block index and applied LSN
- [ ] 6.6 Write unit tests for segment operations

## 7. Indexing System

- [ ] 7.1 Implement in-memory time-range index (maps time → segment file)
- [ ] 7.2 Implement per-segment block index for random access
- [ ] 7.3 Implement Bloom filter for point ID existence checks
- [ ] 7.4 Implement correction segment index merging during queries
- [ ] 7.5 Implement index persistence (save/load from disk)
- [ ] 7.6 Implement index rebuilding from segment files
- [ ] 7.7 Write unit tests for index operations

## 8. Core Storage Engine

- [ ] 8.1 Implement `StorageEngine` main class
- [ ] 8.2 Implement initialization and directory structure setup
- [ ] 8.3 Implement `write()` method with batch support and LSN assignment
- [ ] 8.4 Implement late write handling (correction segments for closed partitions)
- [ ] 8.5 Implement `query()` method with time-range filtering and correction segment merging
- [ ] 8.6 Implement segment rotation logic (daily partitions)
- [ ] 8.7 Integrate WAL with LSN for durability
- [ ] 8.8 Write integration tests for core storage operations

## 9. Data Rotation and Retention (Segment-Granular)

- [ ] 9.1 Implement `DataRotationManager` class
- [ ] 9.2 Implement segment-granular retention policy enforcement (not timestamp-accurate)
- [ ] 9.3 Implement disk space monitoring and warnings
- [ ] 9.4 Implement critical disk space rejection logic
- [ ] 9.5 Implement archiving support (copy before delete)
- [ ] 9.6 Implement background cleanup scheduling
- [ ] 9.7 Implement cleanup blocked by unsynced data detection
- [ ] 9.8 Write unit tests for rotation logic

## 10. Integrity Verification

- [ ] 10.1 Implement `IntegrityVerifier` class
- [ ] 10.2 Implement per-record checksum verification
- [ ] 10.3 Implement block-level magic number validation
- [ ] 10.4 Implement corruption detection and reporting
- [ ] 10.5 Implement skip-corrupt-records mode for replay
- [ ] 10.6 Implement background data scrubbing
- [ ] 10.7 Implement segment truncation for end-of-file corruption
- [ ] 10.8 Write unit tests for integrity verification

## 11. Backfill and Replay API with Per-Destination Sync

- [ ] 11.1 Implement `BackfillReplayer` class
- [ ] 11.2 Implement query by time range with composite cursor
- [ ] 11.3 Implement query by gateway/point ID with filtering
- [ ] 11.4 Implement resumable replay cursor with composite position (segment, block, record, LSN)
- [ ] 11.5 Implement per-destination sync state tracking
- [ ] 11.6 Implement "mark as synced" acknowledgment with composite cursor
- [ ] 11.7 Implement "query unsynced data for destination" interface
- [ ] 11.8 Implement sync state persistence to disk
- [ ] 11.9 Write unit tests for replay functionality

## 12. Async I/O Interface

- [ ] 12.1 Implement Promise-based write API
- [ ] 12.2 Implement streaming read API with backpressure
- [ ] 12.3 Implement callback-based API wrappers
- [ ] 12.4 Implement Worker Thread support for I/O offloading
- [ ] 12.5 Write unit tests for async operations

## 13. Performance Optimization and Benchmarking

- [ ] 13.1 Implement write buffering (1MB buffer, 100ms flush)
- [ ] 13.2 Implement vectored I/O using `fs.writev`
- [ ] 13.3 Optimize index lookup with binary search
- [ ] 13.4 Implement read-ahead caching for sequential access
- [ ] 13.5 Define benchmark envelope documentation (hardware, filesystem, record size, etc.)
- [ ] 13.6 Benchmark write performance under benchmark envelope (target: 10K records in ≤100ms)
- [ ] 13.7 Benchmark read performance under benchmark envelope (target: 10K records in ≤100ms)
- [ ] 13.8 Profile and optimize bottlenecks

## 14. Exception Handling and Edge Cases

- [ ] 14.1 Implement handling for disk full scenarios
- [ ] 14.2 Implement handling for permission denied errors
- [ ] 14.3 Implement handling for file corruption (non-WAL)
- [ ] 14.4 Implement handling for concurrent access conflicts
- [ ] 14.5 Implement graceful degradation on resource exhaustion
- [ ] 14.6 Implement recovery procedures for various failure modes
- [ ] 14.7 Write stress tests and chaos tests

## 15. Configuration and Management API

- [ ] 15.1 Implement configuration loading (file + constructor options)
- [ ] 15.2 Implement storage statistics API (size, compression ratio, record counts)
- [ ] 15.3 Implement manual cleanup trigger API
- [ ] 15.4 Implement integrity check trigger API
- [ ] 15.5 Implement storage health check endpoint

## 16. Documentation and Examples

- [ ] 16.1 Write API documentation (JSDoc comments)
- [ ] 16.2 Create usage examples (basic write/read, batch operations, replay)
- [ ] 16.3 Write configuration guide
- [ ] 16.4 Write troubleshooting guide
- [ ] 16.5 Document performance tuning options and benchmark envelope

## 17. Integration Testing

- [ ] 17.1 Write end-to-end test: write → read → verify data integrity
- [ ] 17.2 Write end-to-end test: crash recovery scenario with LSN reconciliation
- [ ] 17.3 Write end-to-end test: 30-day retention simulation with segment granularity
- [ ] 17.4 Write end-to-end test: backfill/replay workflow with sync tracking
- [ ] 17.5 Write end-to-end test: late write handling with correction segments
- [ ] 17.6 Write load test: sustained 10K points at 1-second intervals
- [ ] 17.7 Write performance benchmark suite under defined envelope
