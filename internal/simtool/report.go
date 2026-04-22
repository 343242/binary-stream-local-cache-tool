package simtool

import (
	"fmt"
	"time"
)

type Report struct {
	RootDir          string
	ProfileName      string
	Gateways         int
	PointsPerGateway int
	Rounds           int
	Records          int
	PayloadBytes     int
	BatchSize        int
	TotalDuration    time.Duration
	AvgBatchDuration time.Duration
	MaxBatchDuration time.Duration
	SegmentCount     int
	WALFileCount     int
	WorkspaceBytes   int64
}

func FormatReport(report Report) string {
	return fmt.Sprintf(
		"root=%s profile=%s gateways=%d points_per_gateway=%d rounds=%d records=%d payload_bytes=%d batch_size=%d total_duration=%s avg_batch_duration=%s max_batch_duration=%s segment_count=%d wal_file_count=%d workspace_bytes=%d\n",
		report.RootDir,
		report.ProfileName,
		report.Gateways,
		report.PointsPerGateway,
		report.Rounds,
		report.Records,
		report.PayloadBytes,
		report.BatchSize,
		report.TotalDuration,
		report.AvgBatchDuration,
		report.MaxBatchDuration,
		report.SegmentCount,
		report.WALFileCount,
		report.WorkspaceBytes,
	)
}
