## ADDED Requirements

### Requirement: Per-record checksums
The system SHALL compute and store a checksum for each data record.

#### Scenario: Write with checksum
- **WHEN** writing a record to storage
- **THEN** the system SHALL compute a CRC32 checksum of the record data
- **AND** store the checksum alongside the record

#### Scenario: Verify on read
- **WHEN** reading a record from storage
- **THEN** the system SHALL verify the checksum against record data
- **AND** return the record if checksum matches
- **AND** throw a corruption error if checksum mismatch detected

### Requirement: Block-level integrity
The system SHALL use magic numbers and block-level validation to detect structural corruption.

#### Scenario: Detect truncated write
- **GIVEN** a crash occurred during a block write
- **WHEN** reading the affected block
- **THEN** the system SHALL detect invalid magic number or length
- **AND** report the specific segment and offset of corruption

#### Scenario: Skip corrupt records during replay
- **WHEN** replaying data for backfill
- **AND** a corrupt record is encountered
- **THEN** the system SHALL log the corruption
- **AND** skip the corrupt record
- **AND** continue replaying subsequent valid records

### Requirement: Data scrubbing
The system SHALL support background verification of stored data integrity.

#### Scenario: Schedule integrity check
- **WHEN** a background integrity check is initiated
- **THEN** the system SHALL scan all segments
- **AND** verify checksums for all records
- **AND** report any corruption detected
- **AND** generate a detailed integrity report

### Requirement: Recovery from corruption
The system SHALL provide mechanisms to recover from detectable corruption.

#### Scenario: Truncate corrupt segment
- **WHEN** corruption is detected in a segment
- **AND** the corruption is at the end of the segment
- **THEN** the system SHALL offer to truncate the segment at the last valid record
- **AND** preserve all valid data before the corruption point
