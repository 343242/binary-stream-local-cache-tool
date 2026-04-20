package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"fastReadFile/desktop/backend"
	core "fastReadFile/internal/core"
	"fastReadFile/internal/desktop/service"
	"fastReadFile/internal/desktop/writer"
	"fastReadFile/internal/lock"
)

func TestDesktopBackend(t *testing.T) {
	t.Run("TestOpenWorkspaceAndCloseWorkspaceAreSerialized", runTestOpenWorkspaceAndCloseWorkspaceAreSerialized)
	t.Run("TestCloseWorkspaceCancelsCancellableTasks", runTestCloseWorkspaceCancelsCancellableTasks)
	t.Run("TestCloseWorkspaceBlocksOnNonCancellableTasks", runTestCloseWorkspaceBlocksOnNonCancellableTasks)
	t.Run("TestAcquireMaintenanceLockTransitionsWorkspaceMode", runTestAcquireMaintenanceLockTransitionsWorkspaceMode)
	t.Run("TestTaskLifecycleEmitsStartedProgressFinished", runTestTaskLifecycleEmitsStartedProgressFinished)
	t.Run("TestRecentWorkspacesPersistAndPruneMissingEntries", runTestRecentWorkspacesPersistAndPruneMissingEntries)
}

func TestOpenWorkspaceAndCloseWorkspaceAreSerialized(t *testing.T) {
	runTestOpenWorkspaceAndCloseWorkspaceAreSerialized(t)
}

func TestCloseWorkspaceCancelsCancellableTasks(t *testing.T) {
	runTestCloseWorkspaceCancelsCancellableTasks(t)
}

func TestCloseWorkspaceBlocksOnNonCancellableTasks(t *testing.T) {
	runTestCloseWorkspaceBlocksOnNonCancellableTasks(t)
}

func TestAcquireMaintenanceLockTransitionsWorkspaceMode(t *testing.T) {
	runTestAcquireMaintenanceLockTransitionsWorkspaceMode(t)
}

func TestTaskLifecycleEmitsStartedProgressFinished(t *testing.T) {
	runTestTaskLifecycleEmitsStartedProgressFinished(t)
}

func TestRecentWorkspacesPersistAndPruneMissingEntries(t *testing.T) {
	runTestRecentWorkspacesPersistAndPruneMissingEntries(t)
}

func TestDesktopBackendStartWriterReleasesObserverLock(t *testing.T) {
	t.Parallel()

	app := backend.NewApp()
	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatal(err)
	}
	if _, err := app.OpenWorkspace(root); err != nil {
		t.Fatal(err)
	}

	if err := app.StartWriter(root); err != nil {
		t.Fatalf("StartWriter() error = %v", err)
	}
	state, err := app.GetWorkspaceState()
	if err != nil {
		t.Fatalf("GetWorkspaceState() error = %v", err)
	}
	if state.Mode != "HealthyWriter" {
		t.Fatalf("Mode = %q, want %q", state.Mode, "HealthyWriter")
	}
	if state.LockMode != string(lock.ModeWriterExclusive) {
		t.Fatalf("LockMode = %q, want %q", state.LockMode, lock.ModeWriterExclusive)
	}

	status, err := app.GetWriterStatus()
	if err != nil {
		t.Fatalf("GetWriterStatus() error = %v", err)
	}
	if status.LifecycleState != "running" {
		t.Fatalf("WriterStatus.LifecycleState = %q, want %q", status.LifecycleState, "running")
	}
	if status.WorkspaceState != "HealthyWriter" {
		t.Fatalf("WriterStatus.WorkspaceState = %q, want %q", status.WorkspaceState, "HealthyWriter")
	}
}

func TestDesktopBackendStopWriterRestoresObserverLock(t *testing.T) {
	t.Parallel()

	app := backend.NewApp()
	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatal(err)
	}
	if _, err := app.OpenWorkspace(root); err != nil {
		t.Fatal(err)
	}
	if err := app.StartWriter(root); err != nil {
		t.Fatalf("StartWriter() error = %v", err)
	}

	if err := app.StopWriter(1000); err != nil {
		t.Fatalf("StopWriter() error = %v", err)
	}
	state, err := app.GetWorkspaceState()
	if err != nil {
		t.Fatalf("GetWorkspaceState() error = %v", err)
	}
	if state.Mode != "HealthyObserver" {
		t.Fatalf("Mode = %q, want %q", state.Mode, "HealthyObserver")
	}
	if state.LockMode != string(lock.ModeObserverShared) {
		t.Fatalf("LockMode = %q, want %q", state.LockMode, lock.ModeObserverShared)
	}

	status, err := app.GetWriterStatus()
	if err != nil {
		t.Fatalf("GetWriterStatus() error = %v", err)
	}
	if status.LifecycleState != "stopped" {
		t.Fatalf("WriterStatus.LifecycleState = %q, want %q", status.LifecycleState, "stopped")
	}
	if status.WorkspaceState != "HealthyObserver" {
		t.Fatalf("WriterStatus.WorkspaceState = %q, want %q", status.WorkspaceState, "HealthyObserver")
	}
	if status.RootPath != root {
		t.Fatalf("WriterStatus.RootPath = %q, want %q", status.RootPath, root)
	}
}

func TestPendingConfigAppBindingsKeepEffectiveConfigSeparate(t *testing.T) {
	t.Parallel()

	app := backend.NewApp()
	root := makeBackendWorkspaceRoot(t)
	if _, err := app.OpenWorkspace(root); err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	pending := core.DefaultConfig(root)
	pending.RetentionDays = 14

	if err := app.SavePendingConfig(root, pending); err != nil {
		t.Fatalf("SavePendingConfig() error = %v", err)
	}

	got, err := app.LoadPendingConfig(root)
	if err != nil {
		t.Fatalf("LoadPendingConfig() error = %v", err)
	}
	if got.Config.RetentionDays != pending.RetentionDays {
		t.Fatalf("pending RetentionDays = %d, want %d", got.Config.RetentionDays, pending.RetentionDays)
	}

	effective, err := app.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	wantEffective := strconv.Itoa(core.DefaultConfig(root).RetentionDays)
	if effective.RetentionDays.CurrentValue != wantEffective {
		t.Fatalf("effective RetentionDays = %q, want %q", effective.RetentionDays.CurrentValue, wantEffective)
	}
}

func TestDesktopBackendInitializeWorkspaceBinding(t *testing.T) {
	t.Parallel()

	app := backend.NewApp()
	root := filepath.Join(t.TempDir(), "cache-root")

	if err := app.InitializeWorkspace(root); err != nil {
		t.Fatalf("InitializeWorkspace() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "meta", "replay")); err != nil {
		t.Fatalf("meta/replay missing after InitializeWorkspace(): %v", err)
	}
}

func TestWriterEventsKeepOnlyLatestEntries(t *testing.T) {
	feed := writer.NewEventFeed(3, 10, 5*time.Second)
	for i := 0; i < 5; i++ {
		feed.Emit(writer.Event{Kind: "batch-persisted", Message: fmt.Sprintf("batch-%d", i)})
	}

	events := feed.Snapshot()
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	if events[0].Message != "batch-2" {
		t.Fatalf("oldest kept event = %q, want batch-2", events[0].Message)
	}
	if events[2].Message != "batch-4" {
		t.Fatalf("latest kept event = %q, want batch-4", events[2].Message)
	}
}

func TestWriterEventsThrottleBurstEmissions(t *testing.T) {
	feed := writer.NewEventFeed(10, 2, 5*time.Second)
	for i := 0; i < 5; i++ {
		feed.Emit(writer.Event{Kind: "batch-persisted", Message: fmt.Sprintf("batch-%d", i)})
	}

	events := feed.Snapshot()
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Message != "batch-0" {
		t.Fatalf("events[0].Message = %q, want batch-0", events[0].Message)
	}
	if events[1].Message != "batch-1" {
		t.Fatalf("events[1].Message = %q, want batch-1", events[1].Message)
	}
}

func runTestOpenWorkspaceAndCloseWorkspaceAreSerialized(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeBackendWorkspaceRoot(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	closed := make(chan struct{})

	app := backend.NewAppWithOptions(nil, backend.Hooks{
		BeforeOpenWorkspace: func() {
			close(entered)
			<-release
		},
	})

	openDone := make(chan struct{})
	go func() {
		defer close(openDone)
		if _, err := app.OpenWorkspace(root); err != nil {
			t.Errorf("OpenWorkspace() error = %v", err)
		}
	}()

	<-entered

	go func() {
		if err := app.CloseWorkspace(); err != nil {
			t.Errorf("CloseWorkspace() error = %v", err)
		}
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatalf("CloseWorkspace() finished before OpenWorkspace() released the session lock")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	<-openDone
	<-closed
}

func runTestCloseWorkspaceCancelsCancellableTasks(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeBackendWorkspaceRoot(t)
	app := backend.NewApp()
	if _, err := app.OpenWorkspace(root); err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	task, err := app.StartTask("verify", root, true, cancel)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if task.Status != "running" {
		t.Fatalf("task.Status = %q, want %q", task.Status, "running")
	}

	if err := app.CloseWorkspace(); err != nil {
		t.Fatalf("CloseWorkspace() error = %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatalf("task context was not cancelled on CloseWorkspace()")
	}
}

func runTestAcquireMaintenanceLockTransitionsWorkspaceMode(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeBackendWorkspaceRoot(t)
	app := backend.NewApp()
	state, err := app.OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}
	if state.Mode != "HealthyObserver" {
		t.Fatalf("open Mode = %q, want %q", state.Mode, "HealthyObserver")
	}

	state, err = app.AcquireMaintenanceLock()
	if err != nil {
		t.Fatalf("AcquireMaintenanceLock() error = %v", err)
	}
	if state.Mode != "HealthyMaintenance" {
		t.Fatalf("Mode = %q, want %q", state.Mode, "HealthyMaintenance")
	}
	if state.LockMode != "MaintenanceExclusive" {
		t.Fatalf("LockMode = %q, want %q", state.LockMode, "MaintenanceExclusive")
	}
	if !state.CanRunRepairTail || !state.CanRunShutdown {
		t.Fatalf("maintenance capabilities = repair:%v shutdown:%v, want both true", state.CanRunRepairTail, state.CanRunShutdown)
	}

	state, err = app.ReleaseMaintenanceLock()
	if err != nil {
		t.Fatalf("ReleaseMaintenanceLock() error = %v", err)
	}
	if state.Mode != "HealthyObserver" {
		t.Fatalf("release Mode = %q, want %q", state.Mode, "HealthyObserver")
	}
	if state.LockMode != "ObserverShared" {
		t.Fatalf("release LockMode = %q, want %q", state.LockMode, "ObserverShared")
	}
}

func runTestCloseWorkspaceBlocksOnNonCancellableTasks(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeBackendWorkspaceRoot(t)
	app := backend.NewApp()
	if _, err := app.OpenWorkspace(root); err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	if _, err := app.StartTask("shutdown", root, false, nil); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if err := app.CloseWorkspace(); err == nil {
		t.Fatalf("CloseWorkspace() error = nil, want non-cancellable task block")
	}
}

func runTestTaskLifecycleEmitsStartedProgressFinished(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := makeBackendWorkspaceRoot(t)
	var (
		mu     sync.Mutex
		events []string
	)
	app := backend.NewAppWithOptions(func(name string, payload any) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, name)
	}, backend.Hooks{})
	if _, err := app.OpenWorkspace(root); err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	task, err := app.StartTask("verify", root, true, nil)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	current, total := uint64(1), uint64(3)
	if err := app.UpdateTaskProgress(task.TaskID, "scan", "reading", &current, &total); err != nil {
		t.Fatalf("UpdateTaskProgress() error = %v", err)
	}
	if err := app.FinishTask(task.TaskID, "succeeded", nil, nil); err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) < 4 {
		t.Fatalf("len(events) = %d, want at least 4", len(events))
	}
	lastThree := events[len(events)-3:]
	want := []string{"task:started", "task:progress", "task:finished"}
	for idx := range want {
		if lastThree[idx] != want[idx] {
			t.Fatalf("events[%d] = %q, want %q (tail=%v)", idx, lastThree[idx], want[idx], lastThree)
		}
	}
}

func runTestRecentWorkspacesPersistAndPruneMissingEntries(t *testing.T) {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	rootOne := makeBackendWorkspaceRoot(t)
	rootTwo := makeBackendWorkspaceRoot(t)
	missing := filepath.Join(t.TempDir(), "missing-workspace")

	recentPath := filepath.Join(configHome, "binary-stream-local-cache-tool", "recent-workspaces.json")
	if err := os.MkdirAll(filepath.Dir(recentPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(recentPath) error = %v", err)
	}
	seed, err := json.Marshal([]string{missing, rootOne})
	if err != nil {
		t.Fatalf("Marshal(seed) error = %v", err)
	}
	if err := os.WriteFile(recentPath, seed, 0o644); err != nil {
		t.Fatalf("WriteFile(seed) error = %v", err)
	}

	app := backend.NewApp()
	if _, err := app.OpenWorkspace(rootTwo); err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	recent, err := app.GetRecentWorkspaces()
	if err != nil {
		t.Fatalf("GetRecentWorkspaces() error = %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("len(recent) = %d, want %d", len(recent), 2)
	}
	if recent[0] != rootTwo {
		t.Fatalf("recent[0] = %q, want %q", recent[0], rootTwo)
	}
	if recent[1] != rootOne {
		t.Fatalf("recent[1] = %q, want %q", recent[1], rootOne)
	}
}

func makeBackendWorkspaceRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, rel := range []string{"meta", "segments", "wal", filepath.Join("meta", "replay")} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", rel, err)
		}
	}
	return root
}
