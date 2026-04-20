package backend

import (
	"context"
	"errors"
	"time"

	core "fastReadFile/internal/core"
	"fastReadFile/internal/desktop/service"
	"fastReadFile/internal/desktop/viewmodel"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	session *Session
	events  *eventBus
	ctx     context.Context
}

func NewApp() *App {
	return NewAppWithOptions(nil, Hooks{})
}

func NewAppWithOptions(emitter EventEmitter, hooks Hooks) *App {
	events := &eventBus{}
	events.setEmitter(emitter)
	return &App{
		session: newSession(events, hooks),
		events:  events,
	}
}

func (a *App) SetEventEmitter(emitter EventEmitter) {
	a.events.setEmitter(emitter)
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.events.setEmitter(func(name string, payload any) {
		wruntime.EventsEmit(ctx, name, payload)
	})
}

func (a *App) OpenWorkspace(rootPath string) (viewmodel.WorkspaceState, error) {
	state, err := a.session.openWorkspace(rootPath)
	if err == nil && state.Mode != "InvalidWorkspace" {
		_ = addRecentWorkspace(rootPath)
	}
	return state, err
}

func (a *App) InitializeWorkspace(rootPath string) error {
	return service.InitializeWorkspace(rootPath)
}

func (a *App) CloseWorkspace() error {
	return a.session.closeWorkspace()
}

func (a *App) GetWorkspaceState() (viewmodel.WorkspaceState, error) {
	return a.session.getWorkspaceState(), nil
}

func (a *App) StartWriter(rootPath string) error {
	return a.session.startWriter(rootPath)
}

func (a *App) StopWriter(timeoutMs int) error {
	return a.session.stopWriter(time.Duration(timeoutMs) * time.Millisecond)
}

func (a *App) GetWriterStatus() (viewmodel.WriterStatus, error) {
	return a.session.getWriterStatus(), nil
}

func (a *App) AcquireMaintenanceLock() (viewmodel.WorkspaceState, error) {
	return a.session.acquireMaintenanceLock()
}

func (a *App) ReleaseMaintenanceLock() (viewmodel.WorkspaceState, error) {
	return a.session.releaseMaintenanceLock()
}

func (a *App) Refresh() error {
	return a.session.refresh()
}

func (a *App) GetRecentWorkspaces() ([]string, error) {
	return loadRecentWorkspaces()
}

func (a *App) ChooseWorkspace() (viewmodel.WorkspaceState, error) {
	if a.ctx == nil {
		return viewmodel.WorkspaceState{}, errors.New("desktop runtime is not ready")
	}
	root, err := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Open Cache Directory",
	})
	if err != nil {
		return viewmodel.WorkspaceState{}, err
	}
	if root == "" {
		return a.GetWorkspaceState()
	}
	return a.OpenWorkspace(root)
}

func (a *App) GetOverview() (viewmodel.Overview, error) {
	root, err := a.session.currentRoot()
	if err != nil {
		return viewmodel.Overview{}, err
	}
	return service.GetOverview(root)
}

func (a *App) ListSegments(page int, pageSize int) (viewmodel.PagedSegments, error) {
	root, err := a.session.currentRoot()
	if err != nil {
		return viewmodel.PagedSegments{}, err
	}
	return service.ListSegments(root, page, pageSize)
}

func (a *App) GetSegmentDetail(segmentID uint64) (viewmodel.SegmentDetail, error) {
	root, err := a.session.currentRoot()
	if err != nil {
		return viewmodel.SegmentDetail{}, err
	}
	return service.GetSegmentDetail(root, segmentID)
}

func (a *App) GetWALDetail() (viewmodel.WALDetail, error) {
	root, err := a.session.currentRoot()
	if err != nil {
		return viewmodel.WALDetail{}, err
	}
	return service.GetWALDetail(root)
}

func (a *App) ListCursors() ([]viewmodel.CursorSummary, error) {
	root, err := a.session.currentRoot()
	if err != nil {
		return nil, err
	}
	return service.ListCursors(root)
}

func (a *App) GetCursorDetail(destination string) (viewmodel.CursorDetail, error) {
	root, err := a.session.currentRoot()
	if err != nil {
		return viewmodel.CursorDetail{}, err
	}
	return service.GetCursorDetail(root, destination)
}

func (a *App) GetCheckpointDetail() (viewmodel.CheckpointDetail, error) {
	root, err := a.session.currentRoot()
	if err != nil {
		return viewmodel.CheckpointDetail{}, err
	}
	return service.GetCheckpointDetail(root)
}

func (a *App) GetConfig() (viewmodel.Config, error) {
	root, err := a.session.currentRoot()
	if err != nil {
		return viewmodel.Config{}, err
	}
	return service.GetConfig(root)
}

func (a *App) LoadPendingConfig(rootPath string) (service.PendingConfigFile, error) {
	return service.LoadPendingConfig(rootPath)
}

func (a *App) SavePendingConfig(rootPath string, cfg core.Config) error {
	return service.SavePendingConfig(rootPath, cfg)
}

func (a *App) StartTask(kind, target string, canCancel bool, cancel context.CancelFunc) (viewmodel.Task, error) {
	return a.session.startTask(kind, target, canCancel, cancel), nil
}

func (a *App) UpdateTaskProgress(taskID, phase, message string, current, total *uint64) error {
	return a.session.updateTaskProgress(taskID, phase, message, current, total)
}

func (a *App) FinishTask(taskID, status string, result *viewmodel.OperationResult, guiErr *viewmodel.GUIError) error {
	return a.session.finishTask(taskID, status, result, guiErr)
}

func (a *App) RunVerify() (viewmodel.Task, error) {
	return a.runAsyncOperation("verify", "", true, func(root string) (viewmodel.OperationResult, error) {
		return service.RunVerify(root)
	})
}

func (a *App) RunCloseCheck() (viewmodel.Task, error) {
	return a.runAsyncOperation("close-check", "", true, func(root string) (viewmodel.OperationResult, error) {
		return service.RunCloseCheck(root)
	})
}

func (a *App) RunRepairTail(segmentID uint64) (viewmodel.Task, error) {
	return a.runAsyncOperation("repair-tail", "segment", true, func(root string) (viewmodel.OperationResult, error) {
		return service.RunRepairTail(root, segmentID)
	})
}

func (a *App) RunShutdown() (viewmodel.Task, error) {
	return a.runAsyncOperation("shutdown", "", false, func(root string) (viewmodel.OperationResult, error) {
		return service.RunShutdown(root)
	})
}

func (a *App) CancelTask(taskID string) error {
	return a.session.cancelTask(taskID)
}

func (a *App) runAsyncOperation(kind, target string, canCancel bool, operation func(root string) (viewmodel.OperationResult, error)) (viewmodel.Task, error) {
	state, err := a.GetWorkspaceState()
	if err != nil {
		return viewmodel.Task{}, err
	}
	if state.RootPath == "" {
		return viewmodel.Task{}, errors.New("no workspace is open")
	}

	ctx, cancel := context.WithCancel(context.Background())
	task, err := a.StartTask(kind, target, canCancel, cancel)
	if err != nil {
		cancel()
		return viewmodel.Task{}, err
	}

	go func(root, taskID string) {
		defer cancel()
		current, total := uint64(1), uint64(2)
		_ = a.UpdateTaskProgress(taskID, "starting", "Preparing operation", &current, &total)
		time.Sleep(10 * time.Millisecond)
		if ctx.Err() != nil {
			return
		}
		current = 2
		_ = a.UpdateTaskProgress(taskID, "running", "Running operation", &current, &total)
		result, opErr := operation(root)
		if opErr != nil {
			guiErr := &viewmodel.GUIError{
				Code:        "GUI_ERR_INTERNAL",
				Title:       "Operation Failed",
				Message:     opErr.Error(),
				Recoverable: true,
			}
			_ = a.FinishTask(taskID, "failed", nil, guiErr)
			return
		}
		_ = a.FinishTask(taskID, "succeeded", &result, nil)
	}(state.RootPath, task.TaskID)

	return task, nil
}
