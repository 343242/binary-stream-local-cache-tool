## ADDED Requirements

### Requirement: Write-ahead logging with idempotent replay
The system SHALL maintain a write-ahead log (WAL) with monotonic LSNs, applied-LSN markers in segment footers, and idempotent crash recovery.

#### Scenario: Log write operation with transaction ID
- **WHEN** a write operation is initiated
- **THEN** the system SHALL generate a unique monotonic LSN (uint64)
- **AND** SHALL append the operation to the WAL with format: `[LSN(8)][timestamp(8)][payload_len(4)][payload][crc32(4)]`
- **AND** the WAL entry SHALL be fsynced to disk before acknowledging the write

#### Scenario: Track applied LSN in segment footers
- **WHEN** records from a WAL entry are successfully written to segment files
- **THEN** the system SHALL write the applied LSN to the segment footer
- **AND** SHALL fsync the segment footer
- **AND** the footer format SHALL be: `[magic(4)][applied_lsn(8)][block_index...][footer_crc32(4)]`

### Requirement: Idempotent crash recovery with LSN reconciliation
The system SHALL implement restart reconciliation to distinguish between logged-but-not-applied and already-applied operations.

#### Scenario: Recover from crash with LSN reconciliation
- **GIVEN** a crash occurred at any point in the write pipeline
- **WHEN** the system restarts
- **THEN** the system SHALL read the maximum applied LSN from all segment footers
- **AND** SHALL scan the WAL from the beginning
- **AND** for each WAL entry with LSN > max_applied_lsn:
  - Read the target segment's footer
  - If segment.footer.lsn >= entry.lsn: skip (already applied)
  - Else: replay entry to segment
- **AND** replay SHALL be idempotent (replaying same LSN produces same result)

#### Scenario: Handle crash after WAL fsync but before segment write
- **GIVEN** a crash occurred after WAL entry fsync
- **AND** the segment was never written
- **WHEN** the system restarts
- **THEN** max_applied_lsn SHALL be < entry.lsn
- **AND** the entry SHALL be detected as unapplied
- **AND** the entry SHALL be replayed to the segment
- **AND** the segment footer SHALL be updated with entry.lsn

#### Scenario: Handle crash after segment write but before footer update
- **GIVEN** a crash occurred after segment data write but before footer fsync
- **WHEN** the system restarts
- **THEN** max_applied_lsn SHALL be < entry.lsn (footer not updated)
- **AND** the entry SHALL be replayed
- **AND** segment data MAY be duplicate but CRC32 detects duplicates
- **AND** footer SHALL be updated with entry.lsn after replay

#### Scenario: Handle crash after footer fsync
- **GIVEN** a crash occurred after segment footer fsync
- **WHEN** the system restarts
- **THEN** max_applied_lsn SHALL be >= entry.lsn
- **AND** the entry SHALL be correctly skipped
- **AND** no duplicate data SHALL be written

### Requirement: Atomic batch operations via single WAL entry
The system SHALL ensure atomicity for batch write operations by writing the entire batch as a single WAL entry.

#### Scenario: Complete batch or none with single WAL entry
- **WHEN** a batch write of N records is submitted
- **THEN** the system SHALL serialize all N records into a single WAL entry payload
- **AND** SHALL assign a single LSN to the entire batch
- **AND** the WAL append + fsync SHALL be atomic
- **AND** either ALL N records SHALL be persisted (WAL fsync succeeds)
- **OR** NONE SHALL be persisted (WAL fsync fails)
- **AND** partial batch writes SHALL NOT occur

#### Scenario: Batch replay is atomic
- **GIVEN** a batch of N records was written as WAL entry with LSN X
- **AND** a crash occurred
- **WHEN** the system recovers and replays LSN X
- **THEN** all N records SHALL be written to the segment as a unit
- **AND** the segment footer SHALL be updated with LSN X only after all N records are written

### Requirement: WAL truncation after checkpoint
The system SHALL truncate the WAL after successful checkpoint to prevent unbounded growth.

#### Scenario: Checkpoint and truncate
- **WHEN** a checkpoint is successfully completed
- **AND** all WAL entries up to checkpoint LSN are committed to segment files (footers fsynced)
- **THEN** the system SHALL truncate the WAL up to that LSN
- **AND** SHALL update the checkpoint metadata file with the checkpoint LSN

#### Scenario: WAL size limit enforcement
- **GIVEN** the WAL size exceeds the configured maximum (default 2 GB)
- **WHEN** a write operation is initiated
- **THEN** the system SHALL trigger an immediate checkpoint
- **AND** SHALL truncate the WAL before accepting the new write
- **AND** SHALL emit a warning about WAL size limit reached

### Requirement: WAL integrity with corruption detection
The system SHALL detect and handle WAL corruption gracefully.

#### Scenario: Detect WAL corruption at entry boundary
- **WHEN** WAL corruption is detected during startup replay
- **THEN** the system SHALL stop replay at the last valid entry (verified by CRC32)
- **AND** SHALL truncate the corrupted portion
- **AND** SHALL emit a warning with details of lost entries (count, LSN range)
- **AND** SHALL continue with remaining valid data

#### Scenario: Detect corrupted WAL entry CRC32
- **GIVEN** a WAL entry has invalid CRC32
- **WHEN** scanning the WAL
- **THEN** the system SHALL detect the CRC32 mismatch
- **AND** SHALL stop replay before this entry
- **AND** SHALL report the corrupted entry LSN
