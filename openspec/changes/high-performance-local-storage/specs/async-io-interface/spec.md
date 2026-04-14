## ADDED Requirements

### Requirement: Asynchronous write API
The system SHALL provide non-blocking write operations via Promise-based API.

#### Scenario: Async batch write
- **WHEN** calling the async write method with a batch of records
- **THEN** the system SHALL return a Promise immediately
- **AND** the Promise SHALL resolve when the write is durably persisted
- **AND** the Promise SHALL reject if the write fails

#### Scenario: Concurrent async operations
- **WHEN** multiple async write operations are in flight
- **THEN** the system SHALL process them sequentially
- **AND** each Promise SHALL resolve/reject independently

### Requirement: Streaming read API
The system SHALL provide streaming read operations for efficient data retrieval.

#### Scenario: Stream records
- **WHEN** querying a large time range
- **THEN** the system SHALL return a readable stream
- **AND** records SHALL be yielded as they are read from disk
- **AND** memory usage SHALL remain bounded regardless of result set size

#### Scenario: Backpressure handling
- **WHEN** the consumer is slower than the read rate
- **THEN** the stream SHALL apply backpressure
- **AND** pause reading until the consumer is ready

### Requirement: Callback-based API
The system SHALL provide callback-based APIs for compatibility with legacy code.

#### Scenario: Write with callback
- **WHEN** calling write with a callback function
- **THEN** the system SHALL invoke the callback on completion
- **AND** pass an error object if the operation failed
- **AND** pass null error on success

### Requirement: Worker thread support
The system SHALL support offloading I/O operations to worker threads.

#### Scenario: Use worker thread for writes
- **WHEN** configured to use worker threads
- **THEN** the system SHALL perform disk I/O in a separate thread
- **AND** the main thread SHALL not block on I/O operations
- **AND** results SHALL be communicated via message passing
