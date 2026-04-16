package service

import (
	"strconv"

	core "fastReadFile/internal/core"
	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/pkg/cache"
)

func GetConfig(root string) (viewmodel.Config, error) {
	cfg := cache.DefaultConfig(root)
	defaults := core.DefaultConfig(root)

	return viewmodel.Config{
		RootDir: configField(
			"RootDir",
			"Root Directory",
			cfg.RootDir,
			defaults.RootDir,
			"Existing directory",
			true,
			"Filesystem path for the cache workspace root.",
		),
		SegmentTargetSizeBytes: configField(
			"SegmentTargetSizeBytes",
			"Segment Target Size",
			strconv.FormatInt(cfg.SegmentTargetSizeBytes, 10),
			strconv.FormatInt(defaults.SegmentTargetSizeBytes, 10),
			"64 MiB to 4 GiB",
			true,
			"Preferred segment size before rotation.",
		),
		SegmentSlackSizeBytes: configField(
			"SegmentSlackSizeBytes",
			"Segment Slack Size",
			strconv.FormatInt(cfg.SegmentSlackSizeBytes, 10),
			strconv.FormatInt(defaults.SegmentSlackSizeBytes, 10),
			"1 MiB to 64 MiB; must be smaller than SegmentTargetSizeBytes",
			true,
			"Maximum extra bytes allowed before a segment rotates.",
		),
		BlockTargetSizeBytes: configField(
			"BlockTargetSizeBytes",
			"Block Target Size",
			strconv.FormatInt(cfg.BlockTargetSizeBytes, 10),
			strconv.FormatInt(defaults.BlockTargetSizeBytes, 10),
			"256 KiB to 4 MiB",
			true,
			"Target encoded block size before a write batch splits.",
		),
		CheckpointInterval: configField(
			"CheckpointInterval",
			"Checkpoint Interval",
			cfg.CheckpointInterval.String(),
			defaults.CheckpointInterval.String(),
			"1s to 60s",
			true,
			"Maximum wall-clock interval between checkpoint saves.",
		),
		CheckpointBytes: configField(
			"CheckpointBytes",
			"Checkpoint Bytes",
			strconv.FormatInt(cfg.CheckpointBytes, 10),
			strconv.FormatInt(defaults.CheckpointBytes, 10),
			"4 MiB to 1 GiB",
			true,
			"Maximum WAL growth before forcing a checkpoint.",
		),
		SegmentFsyncInterval: configField(
			"SegmentFsyncInterval",
			"Segment Fsync Interval",
			cfg.SegmentFsyncInterval.String(),
			defaults.SegmentFsyncInterval.String(),
			"10ms to 5s",
			true,
			"Maximum wall-clock interval between active segment fsyncs.",
		),
		SegmentFsyncBytes: configField(
			"SegmentFsyncBytes",
			"Segment Fsync Bytes",
			strconv.FormatInt(cfg.SegmentFsyncBytes, 10),
			strconv.FormatInt(defaults.SegmentFsyncBytes, 10),
			"1 MiB to 64 MiB",
			true,
			"Maximum unflushed segment bytes before forcing fsync.",
		),
		RetentionDays: configField(
			"RetentionDays",
			"Retention Days",
			strconv.Itoa(cfg.RetentionDays),
			strconv.Itoa(defaults.RetentionDays),
			"1 to 365",
			true,
			"Retention window for eligible segment cleanup.",
		),
	}, nil
}

func configField(name, displayName, currentValue, defaultValue, allowedRange string, startupOnly bool, description string) viewmodel.ConfigField {
	return viewmodel.ConfigField{
		Name:         name,
		DisplayName:  displayName,
		CurrentValue: currentValue,
		DefaultValue: defaultValue,
		AllowedRange: allowedRange,
		StartupOnly:  startupOnly,
		Description:  description,
	}
}
