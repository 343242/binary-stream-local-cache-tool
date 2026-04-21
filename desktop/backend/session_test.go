package backend

import (
	"context"
	"testing"

	core "fastReadFile/internal/core"
	"fastReadFile/internal/desktop/service"
	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/desktop/writer"
)

type fakeWriterHost struct {
	startRoot string
	startCfg  core.Config
	status    viewmodel.WriterStatus
	events    []writer.Event
}

func (f *fakeWriterHost) Start(_ context.Context, root string, cfg core.Config) error {
	f.startRoot = root
	f.startCfg = cfg
	f.status = viewmodel.WriterStatus{
		LifecycleState: "running",
		WorkspaceState: "HealthyWriter",
		RootPath:       root,
	}
	f.events = []writer.Event{{
		Kind:    "writer-started",
		Message: "writer started",
	}}
	return nil
}

func (f *fakeWriterHost) Stop(_ context.Context) error {
	f.status = viewmodel.WriterStatus{
		LifecycleState: "stopped",
		WorkspaceState: "HealthyObserver",
		RootPath:       f.startRoot,
	}
	f.events = append(f.events, writer.Event{
		Kind:    "writer-stopped",
		Message: "writer stopped",
	})
	return nil
}

func (f *fakeWriterHost) Status() viewmodel.WriterStatus {
	return f.status
}

func (f *fakeWriterHost) Events() []writer.Event {
	events := make([]writer.Event, len(f.events))
	copy(events, f.events)
	return events
}

func (f *fakeWriterHost) Submit(context.Context, []core.RawRecord) error {
	return nil
}

func TestSessionStartWriterUsesPendingConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatalf("InitializeWorkspace() error = %v", err)
	}

	session := newSession(&eventBus{}, Hooks{})
	if _, err := session.openWorkspace(root); err != nil {
		t.Fatalf("openWorkspace() error = %v", err)
	}

	fakeHost := &fakeWriterHost{
		status: viewmodel.WriterStatus{LifecycleState: "not-started"},
	}
	session.writerHost = fakeHost

	pending := core.DefaultConfig(root)
	pending.RetentionDays = 14
	pending.SegmentTargetSizeBytes = 64 << 20
	pending.CheckpointBytes = 32 << 20
	if err := service.SavePendingConfig(root, pending); err != nil {
		t.Fatalf("SavePendingConfig() error = %v", err)
	}

	if err := session.startWriter(root); err != nil {
		t.Fatalf("startWriter() error = %v", err)
	}

	if fakeHost.startRoot != root {
		t.Fatalf("startRoot = %q, want %q", fakeHost.startRoot, root)
	}
	if fakeHost.startCfg.RetentionDays != pending.RetentionDays {
		t.Fatalf("RetentionDays = %d, want %d", fakeHost.startCfg.RetentionDays, pending.RetentionDays)
	}
	if fakeHost.startCfg.SegmentTargetSizeBytes != pending.SegmentTargetSizeBytes {
		t.Fatalf("SegmentTargetSizeBytes = %d, want %d", fakeHost.startCfg.SegmentTargetSizeBytes, pending.SegmentTargetSizeBytes)
	}
	if fakeHost.startCfg.CheckpointBytes != pending.CheckpointBytes {
		t.Fatalf("CheckpointBytes = %d, want %d", fakeHost.startCfg.CheckpointBytes, pending.CheckpointBytes)
	}
}

func TestAppEmitsWriterRuntimeEvents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatalf("InitializeWorkspace() error = %v", err)
	}

	var statusPayloads []viewmodel.WriterStatus
	var eventPayloads [][]writer.Event
	app := NewAppWithOptions(func(name string, payload any) {
		switch name {
		case EventWriterStatusChanged:
			if status, ok := payload.(viewmodel.WriterStatus); ok {
				statusPayloads = append(statusPayloads, status)
			}
		case EventWriterEventsChanged:
			if events, ok := payload.([]writer.Event); ok {
				eventPayloads = append(eventPayloads, events)
			}
		}
	}, Hooks{})

	if _, err := app.OpenWorkspace(root); err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}
	if err := app.StartWriter(root); err != nil {
		t.Fatalf("StartWriter() error = %v", err)
	}
	if err := app.StopWriter(1000); err != nil {
		t.Fatalf("StopWriter() error = %v", err)
	}

	if len(statusPayloads) == 0 {
		t.Fatalf("expected writer status runtime events")
	}
	if len(eventPayloads) == 0 {
		t.Fatalf("expected writer event runtime events")
	}
	if statusPayloads[len(statusPayloads)-1].LifecycleState != "stopped" {
		t.Fatalf("final lifecycle state = %q, want stopped", statusPayloads[len(statusPayloads)-1].LifecycleState)
	}
	lastEvents := eventPayloads[len(eventPayloads)-1]
	if len(lastEvents) == 0 {
		t.Fatalf("expected writer events in final payload")
	}
	if lastEvents[len(lastEvents)-1].Kind != "writer-stopped" {
		t.Fatalf("final writer event kind = %q, want writer-stopped", lastEvents[len(lastEvents)-1].Kind)
	}
}
