package core

import "time"

const (
	DefaultSegmentTargetSizeBytes = 256 << 20
	DefaultSegmentSlackSizeBytes  = 4 << 20
	DefaultBlockTargetSizeBytes   = 1 << 20
	DefaultCheckpointInterval     = 5 * time.Second
	DefaultCheckpointBytes        = 64 << 20
	DefaultSegmentFsyncInterval   = 100 * time.Millisecond
	DefaultSegmentFsyncBytes      = 4 << 20
	DefaultRetentionDays          = 30
)

type Config struct {
	RootDir                string
	SegmentTargetSizeBytes int64
	SegmentSlackSizeBytes  int64
	BlockTargetSizeBytes   int64
	CheckpointInterval     time.Duration
	CheckpointBytes        int64
	SegmentFsyncInterval   time.Duration
	SegmentFsyncBytes      int64
	RetentionDays          int
}

func DefaultConfig(rootDir string) Config {
	return Config{
		RootDir:                rootDir,
		SegmentTargetSizeBytes: DefaultSegmentTargetSizeBytes,
		SegmentSlackSizeBytes:  DefaultSegmentSlackSizeBytes,
		BlockTargetSizeBytes:   DefaultBlockTargetSizeBytes,
		CheckpointInterval:     DefaultCheckpointInterval,
		CheckpointBytes:        DefaultCheckpointBytes,
		SegmentFsyncInterval:   DefaultSegmentFsyncInterval,
		SegmentFsyncBytes:      DefaultSegmentFsyncBytes,
		RetentionDays:          DefaultRetentionDays,
	}
}
