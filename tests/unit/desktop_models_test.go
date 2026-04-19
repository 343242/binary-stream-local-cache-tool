package unit

import (
	"testing"

	core "fastReadFile/internal/core"
	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/desktop/writer"
)

func TestWriterModelsExposeLifecycleAndWorkspaceStatusContracts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := core.DefaultConfig(root)

	pending := writer.PendingConfig{
		Config: cfg,
		Root:   root,
	}
	if pending.Root != root {
		t.Fatalf("PendingConfig.Root = %q, want %q", pending.Root, root)
	}
	if pending.Config.RootDir != root {
		t.Fatalf("PendingConfig.Config.RootDir = %q, want %q", pending.Config.RootDir, root)
	}

	if writer.EventBufferSize != 200 {
		t.Fatalf("EventBufferSize = %d, want %d", writer.EventBufferSize, 200)
	}
	if writer.EventRateLimitPerSec != 10 {
		t.Fatalf("EventRateLimitPerSec = %d, want %d", writer.EventRateLimitPerSec, 10)
	}
	if writer.WarningDedupeWindowMs != 5000 {
		t.Fatalf("WarningDedupeWindowMs = %d, want %d", writer.WarningDedupeWindowMs, 5000)
	}
	if writer.BatchQueueCapacity != 64 {
		t.Fatalf("BatchQueueCapacity = %d, want %d", writer.BatchQueueCapacity, 64)
	}

	lifecycleStates := []writer.LifecycleState{
		writer.LifecycleNotStarted,
		writer.LifecycleStarting,
		writer.LifecycleRunning,
		writer.LifecycleStopping,
		writer.LifecycleStopped,
		writer.LifecycleStartFailed,
	}
	expectedStates := []string{
		"not-started",
		"starting",
		"running",
		"stopping",
		"stopped",
		"start-failed",
	}
	if len(lifecycleStates) != len(expectedStates) {
		t.Fatalf("len(lifecycleStates) = %d, want %d", len(lifecycleStates), len(expectedStates))
	}
	for idx, state := range lifecycleStates {
		if string(state) != expectedStates[idx] {
			t.Fatalf("lifecycleStates[%d] = %q, want %q", idx, state, expectedStates[idx])
		}
	}

	status := viewmodel.WriterStatus{
		LifecycleState:  string(writer.LifecycleRunning),
		WorkspaceState:  "HealthyWriter",
		RootPath:        root,
		LastError:       "",
		StartedAtUnixMs: 101,
		StoppedAtUnixMs: 0,
	}
	if status.LifecycleState != "running" {
		t.Fatalf("WriterStatus.LifecycleState = %q, want %q", status.LifecycleState, "running")
	}
	if status.WorkspaceState != "HealthyWriter" {
		t.Fatalf("WriterStatus.WorkspaceState = %q, want %q", status.WorkspaceState, "HealthyWriter")
	}
	if status.RootPath != root {
		t.Fatalf("WriterStatus.RootPath = %q, want %q", status.RootPath, root)
	}
	if status.StartedAtUnixMs != 101 {
		t.Fatalf("WriterStatus.StartedAtUnixMs = %d, want %d", status.StartedAtUnixMs, 101)
	}
	if status.StoppedAtUnixMs != 0 {
		t.Fatalf("WriterStatus.StoppedAtUnixMs = %d, want %d", status.StoppedAtUnixMs, 0)
	}
}
