## ADDED Requirements

### Requirement: High-performance write operations with benchmark contract
The system SHALL provide write operations capable of storing 10,000 records in ≤100 milliseconds under the defined benchmark contract conditions.

#### Benchmark Contract Conditions
**Hardware**: NVMe SSD or SATA SSD (≥10K read/≥5K write IOPS), ext4/xfs filesystem, x86_64 CPU ≥2.0 GHz, ≥8 GB RAM
**Software**: Linux kernel 5.4+ or Windows 10+, Node.js v18+, record size 72 bytes (float64 values)
**State**: Compression disabled, cold cache, batch at head of queue, buffer empty at start, WAL checkpointed within 10s
**Measurement**: From write() API call to fsync completion callback

#### Scenario: Batch write of 10K records within benchmark contract
- **GIVEN** benchmark contract conditions are met
- **WHEN** the system receives a batch of 10,000 valid data records
- **THEN** the system SHALL complete the write operation within 100 milliseconds
- **AND** completion SHALL be measured from API call to fsync callback
- **AND** the latency SHALL include: serialization, WAL append + fsync, segment write + fsync
- **AND** the latency SHALL NOT include: queue wait time, application callback processing

#### Scenario: Concurrent writes with queueing
- **GIVEN** multiple write requests arrive simultaneously
- **WHEN** batches are queued sequentially
- **THEN** each batch SHALL be processed within 100ms of reaching the head of the queue
- **AND** total latency SHALL equal queue_wait_time + processing_time
- **AND** processing time SHALL remain ≤100ms per batch

### Requirement: Time-series data model support with capacity limits
The system SHALL store records containing: gateway ID (string, max 64 chars), process point ID (string, max 64 chars), data value (number/float64), and collection timestamp (uint64).

**Capacity Limits Validated**:
- Record size: ~72 bytes uncompressed
- Throughput: 10,000 points × 1 second = 864M records/day = 62.2 GB/day uncompressed
- 30-day retention: ~1.87 TB uncompressed, ~465 GB compressed (4:1 ratio)
- Memory baseline: ~35 MB (indexes + buffers)

#### Scenario: Store valid record
- **WHEN** a record with valid gatewayId (≤64 chars), pointId (≤64 chars), value (float64), and timestamp (uint64) is submitted
- **THEN** the system SHALL persist the record with all fields intact
- **AND** the record SHALL occupy ~72 bytes uncompressed

#### Scenario: Reject invalid record
- **WHEN** a record with missing required fields or exceeding size limits is submitted
- **THEN** the system SHALL reject the write with a validation error
- **AND** no partial data SHALL be persisted

### Requirement: Indexed read by time range
The system SHALL support efficient queries for records within a specified time range.

#### Scenario: Query time range
- **GIVEN** benchmark contract conditions (SSD, cold cache)
- **WHEN** querying for records between time T1 and T2
- **THEN** the system SHALL return all records with timestamps in [T1, T2]
- **AND** the query SHALL complete in ≤100ms for 10,000 matching records
- **AND** compressed blocks SHALL be decompressed on-demand without full segment decompression

#### Scenario: Query by point ID and time
- **WHEN** querying for records of a specific pointId within a time range
- **THEN** the system SHALL return only matching records
- **AND** the result SHALL be sorted by timestamp ascending
- **AND** Bloom filters SHALL optimize point ID existence checks

### Requirement: Append-only segment files with late-write policy
The system SHALL organize data into time-partitioned, append-only segment files with explicit late-write handling via correction segments.

#### Scenario: Write to current segment
- **GIVEN** the current active segment covers time partition P (e.g., 2024-01-15)
- **WHEN** writing data with timestamp in partition P
- **THEN** the system SHALL append to the active segment file
- **AND** out-of-order timestamps within P are allowed
- **AND** the segment SHALL remain open for subsequent writes

#### Scenario: Roll over to new segment for future timestamps
- **GIVEN** the current active segment covers time partition P
- **WHEN** a write occurs for timestamp in partition P+1 (future)
- **THEN** the system SHALL close the current segment (write footer with applied LSN)
- **AND** create a new segment for partition P+1
- **AND** write to the new active segment

#### Scenario: Late write to closed partition via correction segment
- **GIVEN** a segment for time partition P has been closed and finalized
- **WHEN** a write arrives with timestamp in partition P (late write)
- **THEN** the system SHALL NOT reopen the finalized segment
- **AND** SHALL create or append to correction segment `data_YYYYMMDD_N_correction.dat`
- **AND** SHALL index the correction segment for queries
- **AND** queries SHALL merge primary and correction segments

#### Scenario: Late write performance impact
- **GIVEN** correction segments exist for a query time range
- **WHEN** querying that time range
- **THEN** the system SHALL scan correction segments in addition to primary segments
- **AND** SHALL merge results sorted by timestamp
- **AND** performance impact SHALL be <10% for correction segment ratio <5%
