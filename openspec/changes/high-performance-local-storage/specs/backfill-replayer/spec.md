## ADDED Requirements

### Requirement: Backfill query interface
The system SHALL provide a query interface to retrieve cached data for server backfill.

#### Scenario: Query by time range
- **WHEN** requesting data for backfill from time T1 to T2
- **THEN** the system SHALL return an iterator/cursor over matching records
- **AND** records SHALL be ordered by timestamp ascending
- **AND** the interface SHALL support pagination via cursor position

#### Scenario: Query by gateway and point
- **WHEN** requesting data for a specific gatewayId and pointId combination
- **THEN** the system SHALL return only matching records
- **AND** support filtering by time range

### Requirement: Replay cursor management with composite position
The system SHALL support resumable replay cursors with precise composite position tracking.

**Composite Position Format**:
```
{
  segment: string,        // filename
  block_offset: number,   // byte offset in file
  record_index: number,   // index within block
  lsn: number            // LSN for consistency verification
}
```

#### Scenario: Create replay cursor
- **WHEN** initiating a backfill operation
- **THEN** the system SHALL create a cursor at the specified start position
- **AND** the position SHALL include segment, block_offset, record_index, and LSN
- **AND** the cursor SHALL track position as records are read

#### Scenario: Resume interrupted replay
- **GIVEN** a replay was interrupted at composite position P
- **WHEN** resuming the replay with position P
- **THEN** the system SHALL validate the LSN at position P matches expected
- **AND** SHALL continue from the next record after P
- **AND** SHALL NOT re-send already-transmitted records

### Requirement: Per-destination sync state tracking with hard protection
The system SHALL maintain independent, durable sync state for each destination with cleanup guardrails.

**Sync State Format**:
```json
{
  "destination_id": "main-server",
  "sync_position": { "segment": "...", "block_offset": N, "record_index": M, "lsn": X },
  "timestamp": 1705315200000,
  "checksum": "crc32_of_above"
}
```

#### Scenario: Track sync position per destination
- **GIVEN** multiple destination servers exist (main, backup, analytics)
- **WHEN** recording a sync acknowledgment from destination D
- **THEN** the system SHALL store the sync position (composite cursor) for destination D
- **AND** each destination SHALL have independent progress tracking
- **AND** sync state SHALL be persisted to disk with fsync

#### Scenario: Query unsynced data for specific destination
- **WHEN** requesting unsynced data for destination D
- **THEN** the system SHALL return only records after destination D's last-synced position
- **AND** SHALL exclude records already acknowledged by destination D
- **AND** SHALL support time-range filtering within unsynced data

#### Scenario: Handle partial acknowledgments
- **GIVEN** a batch of 100 records was sent to destination D
- **WHEN** destination D acknowledges only 80 records (network error on last 20)
- **THEN** the system SHALL update destination D's sync position to the 80th record
- **AND** the remaining 20 records SHALL remain in unsynced query results
- **AND** retry SHALL send only unacknowledged records

### Requirement: Cleanup guardrails for unsynced data
The system SHALL prevent deletion of data that has not been synced to all configured destinations.

#### Scenario: Retain unsynced data during retention cleanup
- **GIVEN** retention cleanup is triggered
- **AND** a segment contains data with LSN range [1000, 2000]
- **AND** destination D has sync position LSN 1500 (not all data synced)
- **THEN** the system SHALL retain the segment regardless of age
- **AND** SHALL emit warning: "Cleanup blocked: segment X has unsynced data for destination D"
- **AND** SHALL only delete when all destinations have sync position >= 2000

#### Scenario: Soft retention with unsynced data
- **GIVEN** retention period is configured to 15 days
- **AND** segment S is 20 days old (exceeds retention)
- **AND** segment S has unsynced data for at least one destination
- **THEN** segment S SHALL NOT be deleted
- **AND** actual retention SHALL exceed configured retention
- **AND** the system SHALL track "retention_pressure" metric

#### Scenario: Force cleanup with data loss warning (emergency)
- **GIVEN** disk space is critically low (< 3%)
- **AND** unsynced data blocks cleanup
- **WHEN** operator issues force-cleanup command
- **THEN** the system SHALL list segments that will be deleted with:
  - Affected destinations
  - LSN ranges that will be lost
  - Estimated record counts
- **AND** SHALL require explicit confirmation
- **AND** SHALL log WARNING_DATA_LOSS event
- **AND** destinations SHALL need to re-request lost data from source if available

### Requirement: Sync state durability
The system SHALL ensure sync state survives crashes and restarts.

#### Scenario: Persist sync acknowledgment durably
- **WHEN** a sync acknowledgment is recorded for destination D
- **THEN** the system SHALL write sync state to disk
- **AND** SHALL fsync the sync state file
- **AND** SHALL acknowledge the sync to caller only after fsync completes

#### Scenario: Reload sync state on restart
- **GIVEN** the system restarts after a crash
- **WHEN** initializing
- **THEN** the system SHALL load sync state for all destinations
- **AND** SHALL verify sync state file integrity (checksum)
- **AND** SHALL emit error if sync state is corrupted (manual recovery required)
