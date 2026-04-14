package core

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
	Destination      string
	AppliedCursor    ReplayCursor
	PreviousWriteSeq uint64
	CurrentWriteSeq  uint64
}

type CapacityStats struct {
	SegmentCount  int
	RetentionDays int
	NextWriteSeq  uint64
}

type IOStats struct {
	WriteBatchesTotal   uint64
	RecordsWrittenTotal uint64
	ReplayBatchesTotal  uint64
	ReplayRecordsTotal  uint64
	CheckpointsTotal    uint64
	SegmentFsyncTotal   uint64
}

type ReplayStats struct {
	AckOperationsTotal    uint64
	CursorCorruptionTotal uint64
	LastAckedWriteSeq     uint64
}

type HealthStats struct {
	GracefulShutdownsTotal            uint64
	UngracefulShutdownRecoveriesTotal uint64
	SegmentTailRepairsTotal           uint64
}

type StatsSnapshot struct {
	Capacity CapacityStats
	IO       IOStats
	Replay   ReplayStats
	Health   HealthStats
}
