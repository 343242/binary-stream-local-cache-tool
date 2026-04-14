## ADDED Requirements

### Requirement: Configurable retention period with segment granularity and unsynced data protection
The system SHALL support configurable data retention with a default of 30 days, minimum of 1 day, and hard protection for unsynced data.

**Retention Policy**:
- Configured retention period: N days (default 30, min 1, max 365)
- Actual retention: max(configured_retention, oldest_unsynced_data_age)
- Cleanup granularity: per segment (not timestamp-accurate)
- Unsynced data: NEVER automatically deleted

#### Scenario: Automatic expiration at segment granularity
- **GIVEN** retention period is 15 days
- **AND** all data in segment S is synced to all destinations
- **AND** segment S partition date is > 15 days old
- **WHEN** cleanup runs
- **THEN** segment S SHALL be marked for deletion
- **AND** SHALL be permanently removed during cleanup

#### Scenario: Retain segments with unsynced data
- **GIVEN** retention period is 15 days
- **AND** segment S is 20 days old (exceeds retention)
- **AND** segment S has records not synced to destination D
- **WHEN** cleanup runs
- **THEN** segment S SHALL NOT be deleted
- **AND** system SHALL emit retention_blocked_by_unsynced warning
- **AND** segment S SHALL be retained until all destinations sync

#### Scenario: Segment-granular retention precision
- **GIVEN** retention period is 15 days
- **AND** segment S spans timestamps from day 14 to day 16
- **WHEN** cleanup runs
- **THEN** the entire segment S SHALL be retained (not partially trimmed)
- **AND** actual retention SHALL be 16 days (exceeds configured by up to 1 segment duration)

### Requirement: Disk space monitoring with tiered thresholds
The system SHALL monitor available disk space with warning and critical thresholds.

#### Scenario: Low disk space warning (10% threshold)
- **WHEN** available disk space falls below 10% of total capacity
- **THEN** the system SHALL emit a warning (disk_space_low)
- **AND** SHALL include current usage percentage and bytes remaining
- **AND** SHALL continue accepting writes

#### Scenario: Critical disk space rejection (5% threshold)
- **WHEN** available disk space falls below 5% of total capacity
- **THEN** the system SHALL reject new write operations
- **AND** SHALL return DiskSpaceError with message "Critical disk space: X% remaining"
- **AND** SHALL preserve all existing data
- **AND** SHALL continue allowing read operations

#### Scenario: Emergency disk space (3% threshold) - unsynced data cleanup
- **WHEN** available disk space falls below 3% of total capacity
- **THEN** the system SHALL enter emergency mode
- **AND** SHALL emit critical alert (disk_space_emergency)
- **AND** SHALL provide force-cleanup API for operator to delete oldest segments
- **AND** SHALL warn about potential data loss for unsynced data

#### Scenario: Recovery from critical disk space
- **GIVEN** the system was rejecting writes due to critical disk space
- **WHEN** available space increases above 7% (recovery threshold)
- **THEN** the system SHALL resume accepting writes
- **AND** SHALL emit disk_space_recovered notification

### Requirement: Data archiving support
The system SHALL support archiving segments before deletion for long-term storage.

#### Scenario: Archive before delete
- **WHEN** a segment is marked for deletion due to retention policy
- **AND** archiving is enabled
- **THEN** the system SHALL copy the segment to the archive location
- **AND** SHALL verify archive integrity (size + CRC32)
- **AND** SHALL only delete the original after successful verification

### Requirement: Cleanup scheduling without blocking operations
The system SHALL perform cleanup operations without blocking read/write operations.

#### Scenario: Background cleanup
- **WHEN** cleanup is triggered by schedule or manual request
- **THEN** the system SHALL perform deletions in a background task
- **AND** read/write operations SHALL continue uninterrupted
- **AND** cleanup SHALL report completion status with segments_deleted count

#### Scenario: Cleanup blocked by unsynced data tracking
- **GIVEN** cleanup identifies segment S for deletion
- **AND** segment S has unsynced data for destination D
- **THEN** the system SHALL skip segment S
- **AND** SHALL record "cleanup_blocked: {segment: S, destination: D, unsynced_lsn_range: [X, Y]}"
- **AND** SHALL emit metric: retention_segments_blocked_by_unsynced +1

#### Scenario: Manual cleanup trigger
- **WHEN** operator calls manualCleanup() API
- **THEN** the system SHALL run cleanup immediately (background)
- **AND** SHALL return promise that resolves with cleanup report
- **AND** cleanup report SHALL include: segments_deleted, segments_blocked, bytes_freed
