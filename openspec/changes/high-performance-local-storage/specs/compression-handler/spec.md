## ADDED Requirements

### Requirement: Per-block Snappy compression with block index
The system SHALL support optional Snappy compression at the block level (64KB blocks) with a block index for random access.

**Block Format**:
- Block header: magic(4) + uncompressed_size(4) + compressed_size(4) + first_timestamp(8) + last_timestamp(8) + crc32(4)
- Payload: Snappy-compressed records

**Block Index** (stored in segment footer):
- Array of: first_timestamp(8) + file_offset(8) + compressed_size(4)
- Enables O(log n) block lookup for time-range queries

#### Scenario: Compress block on write
- **WHEN** a block of records (up to 64KB uncompressed) is written to a segment
- **AND** compression is enabled
- **THEN** the system SHALL compress the block using Snappy
- **AND** SHALL store compression metadata in the block header
- **AND** SHALL add the block to the block index with timestamp range and offset

#### Scenario: Transparent block decompression on read
- **WHEN** reading records from a compressed segment
- **AND** the records are in compressed block N
- **THEN** the system SHALL use the block index to locate block N
- **AND** SHALL read only block N from disk (not entire segment)
- **AND** SHALL decompress block N
- **AND** SHALL extract matching records
- **AND** SHALL return original records without caller awareness

#### Scenario: Random access via block index
- **GIVEN** a segment with compressed blocks
- **WHEN** querying for records at timestamp T
- **THEN** the system SHALL binary search the block index for the block containing T
- **AND** SHALL decompress only that block
- **AND** SHALL scan the block for matching records
- **AND** SHALL complete within read latency budget

### Requirement: Block-level compression only
The system SHALL NOT use whole-segment compression.

#### Scenario: Verify per-block compression
- **GIVEN** compression is enabled
- **WHEN** writing a segment with multiple blocks
- **THEN** each block SHALL be independently compressed
- **AND** the segment SHALL be readable without decompressing entire file
- **AND** random access to any timestamp SHALL require decompressing ≤2 blocks

### Requirement: Compression algorithm and configuration
The system SHALL support Snappy compression (v1) with extensibility for future algorithms.

#### Scenario: Enable Snappy compression
- **WHEN** compression is enabled with algorithm "snappy"
- **THEN** the system SHALL use Snappy for compression
- **AND** compression SHALL be applied per-block at write time
- **AND** NOT at segment close time

#### Scenario: Compression performance requirements
- **GIVEN** Snappy compression is enabled
- **WHEN** writing blocks
- **THEN** compression speed SHALL be ≥200 MB/s
- **AND** decompression speed SHALL be ≥500 MB/s
- **AND** compression ratio for time-series data SHALL be 3:1 to 5:1

#### Scenario: Disable compression
- **WHEN** compression is disabled via configuration
- **THEN** blocks SHALL be written uncompressed
- **AND** block headers SHALL indicate uncompressed size equals compressed size

### Requirement: Compression statistics
The system SHALL track and report compression statistics.

#### Scenario: Report compression ratio
- **WHEN** querying storage statistics
- **THEN** the system SHALL report:
  - Total uncompressed bytes
  - Total compressed bytes
  - Overall compression ratio
  - Per-segment compression metrics
  - Per-block compression distribution (min, max, avg)
