package stats

import (
	"sync"

	cache "fastReadFile/internal/core"
)

type Collector struct {
	mu       sync.Mutex
	snapshot cache.StatsSnapshot
}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) RecordWrite(recordCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.IO.WriteBatchesTotal++
	c.snapshot.IO.RecordsWrittenTotal += uint64(recordCount)
}

func (c *Collector) RecordReplay(recordCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.IO.ReplayBatchesTotal++
	c.snapshot.IO.ReplayRecordsTotal += uint64(recordCount)
}

func (c *Collector) RecordAck(writeSeq uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.Replay.AckOperationsTotal++
	c.snapshot.Replay.LastAckedWriteSeq = writeSeq
}

func (c *Collector) RecordCheckpoint() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.IO.CheckpointsTotal++
}

func (c *Collector) RecordSegmentFsync() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.IO.SegmentFsyncTotal++
}

func (c *Collector) RecordUngracefulRecovery() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.Health.UngracefulShutdownRecoveriesTotal++
}

func (c *Collector) RecordGracefulShutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.Health.GracefulShutdownsTotal++
}

func (c *Collector) RecordSegmentTailRepair(count uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.Health.SegmentTailRepairsTotal += count
}

func (c *Collector) Snapshot() cache.StatsSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshot
}
