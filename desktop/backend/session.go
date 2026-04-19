package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fastReadFile/internal/core"
	"fastReadFile/internal/desktop/service"
	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/desktop/writer"
	"fastReadFile/internal/lock"
)

type Session struct {
	mu         sync.RWMutex
	generation uint64
	state      viewmodel.WorkspaceState
	root       string
	lockHandle *lock.Handle
	lockMode   lock.Mode
	writerHost writer.WriterHost
	tasks      map[string]*taskRecord
	taskSeq    uint64
	hooks      Hooks
	events     *eventBus
}

func newSession(events *eventBus, hooks Hooks) *Session {
	return &Session{
		state: viewmodel.WorkspaceState{
			Mode:   "NoWorkspace",
			Health: "N/A",
		},
		writerHost: writer.NewHost(),
		tasks:      make(map[string]*taskRecord),
		hooks:      hooks,
		events:     events,
	}
}

func (s *Session) openWorkspace(root string) (viewmodel.WorkspaceState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hooks.BeforeOpenWorkspace != nil {
		s.hooks.BeforeOpenWorkspace()
	}

	state, err := service.OpenWorkspace(root)
	if err != nil {
		return viewmodel.WorkspaceState{}, err
	}
	if err := s.resetWorkspaceLocked(); err != nil {
		return viewmodel.WorkspaceState{}, err
	}

	if state.Mode == "InvalidWorkspace" {
		s.generation++
		s.root = root
		s.state = state
		s.events.emit(EventWorkspaceChanged, state)
		return state, nil
	}

	handle, err := lock.Acquire(root, lock.ModeObserverShared, lock.Metadata{Program: "cache-desktop"})
	if err != nil {
		return viewmodel.WorkspaceState{}, err
	}

	if err := s.releaseLockLocked(); err != nil {
		_ = handle.Close()
		return viewmodel.WorkspaceState{}, err
	}

	s.generation++
	s.root = root
	s.lockHandle = handle
	s.lockMode = lock.ModeObserverShared
	s.state = observerState(state)
	s.events.emit(EventWorkspaceChanged, s.state)
	return s.state, nil
}

func (s *Session) closeWorkspace() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hooks.BeforeCloseWorkspace != nil {
		s.hooks.BeforeCloseWorkspace()
	}

	if s.hasBlockingTasksLocked() {
		return fmt.Errorf("workspace close blocked by non-cancellable tasks")
	}
	if err := s.resetWorkspaceLocked(); err != nil {
		return err
	}

	s.generation++
	s.root = ""
	s.state = viewmodel.WorkspaceState{
		Mode:   "NoWorkspace",
		Health: "N/A",
	}
	s.events.emit(EventWorkspaceChanged, s.state)
	return nil
}

func (s *Session) getWorkspaceState() viewmodel.WorkspaceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Session) currentRoot() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.root == "" {
		return "", fmt.Errorf("no workspace is open")
	}
	return s.root, nil
}

func (s *Session) acquireMaintenanceLock() (viewmodel.WorkspaceState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.root == "" || s.state.Mode == "NoWorkspace" || s.state.Mode == "InvalidWorkspace" {
		return viewmodel.WorkspaceState{}, fmt.Errorf("no valid workspace is open")
	}
	if s.lockMode == lock.ModeWriterExclusive {
		return viewmodel.WorkspaceState{}, fmt.Errorf("writer must stop before entering maintenance mode")
	}
	if s.lockMode == lock.ModeMaintenanceExclusive {
		return s.state, nil
	}

	if err := s.releaseLockLocked(); err != nil {
		return viewmodel.WorkspaceState{}, err
	}

	handle, err := lock.Acquire(s.root, lock.ModeMaintenanceExclusive, lock.Metadata{Program: "cache-desktop"})
	if err != nil {
		observer, reacquireErr := lock.Acquire(s.root, lock.ModeObserverShared, lock.Metadata{Program: "cache-desktop"})
		if reacquireErr == nil {
			s.lockHandle = observer
			s.lockMode = lock.ModeObserverShared
			s.state = observerState(s.state)
		} else {
			s.state = degradedState(s.state, "Unable to restore observer lock after maintenance lock failure.")
		}
		return viewmodel.WorkspaceState{}, err
	}

	s.lockHandle = handle
	s.lockMode = lock.ModeMaintenanceExclusive
	s.state = maintenanceState(s.state)
	s.events.emit(EventWorkspaceChanged, s.state)
	return s.state, nil
}

func (s *Session) releaseMaintenanceLock() (viewmodel.WorkspaceState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lockMode != lock.ModeMaintenanceExclusive {
		return s.state, nil
	}
	if err := s.releaseLockLocked(); err != nil {
		return viewmodel.WorkspaceState{}, err
	}

	handle, err := lock.Acquire(s.root, lock.ModeObserverShared, lock.Metadata{Program: "cache-desktop"})
	if err != nil {
		s.state = degradedState(s.state, "Unable to restore observer lock after releasing maintenance mode.")
		return viewmodel.WorkspaceState{}, err
	}

	s.lockHandle = handle
	s.lockMode = lock.ModeObserverShared
	s.state = observerState(s.state)
	s.events.emit(EventWorkspaceChanged, s.state)
	return s.state, nil
}

func (s *Session) refresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.root == "" {
		return nil
	}

	state, err := service.OpenWorkspace(s.root)
	if err != nil {
		return err
	}
	if s.lockMode == lock.ModeMaintenanceExclusive {
		s.state = maintenanceState(state)
	} else if s.lockMode == lock.ModeWriterExclusive {
		s.state = writerState(state)
	} else if state.Mode == "InvalidWorkspace" {
		s.state = state
	} else {
		s.state = observerState(state)
	}
	s.events.emit(EventWorkspaceChanged, s.state)
	return nil
}

func (s *Session) startWriter(rootPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.root == "" || s.state.Mode == "NoWorkspace" || s.state.Mode == "InvalidWorkspace" {
		return fmt.Errorf("no valid workspace is open")
	}
	if rootPath != "" && rootPath != s.root {
		return fmt.Errorf("writer root %q does not match the open workspace %q", rootPath, s.root)
	}

	status := s.writerHost.Status()
	switch status.LifecycleState {
	case string(writer.LifecycleRunning), string(writer.LifecycleStarting):
		return nil
	case string(writer.LifecycleStopping):
		return fmt.Errorf("writer is stopping")
	}
	if s.lockMode == lock.ModeMaintenanceExclusive {
		return fmt.Errorf("maintenance mode must be released before starting writer")
	}

	root := s.root
	baseState := s.state
	if err := s.releaseLockLocked(); err != nil {
		return err
	}

	handle, err := lock.Acquire(root, lock.ModeWriterExclusive, lock.Metadata{Program: "cache-desktop"})
	if err != nil {
		s.restoreObserverLockLocked(baseState, "Unable to restore observer lock after writer lock failure.")
		return err
	}

	if err := s.writerHost.Start(context.Background(), root, core.DefaultConfig(root)); err != nil {
		_ = handle.Close()
		s.restoreObserverLockLocked(baseState, "Unable to restore observer lock after writer start failure.")
		s.events.emit(EventWriterStatusChanged, s.writerHost.Status())
		s.events.emit(EventWorkspaceChanged, s.state)
		return err
	}

	s.lockHandle = handle
	s.lockMode = lock.ModeWriterExclusive
	s.state = writerState(baseState)
	s.events.emit(EventWriterStatusChanged, s.writerHost.Status())
	s.events.emit(EventWorkspaceChanged, s.state)
	return nil
}

func (s *Session) stopWriter(timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := s.writerHost.Status()
	switch status.LifecycleState {
	case string(writer.LifecycleNotStarted), string(writer.LifecycleStopped):
		if s.lockMode == lock.ModeWriterExclusive {
			if err := s.releaseLockLocked(); err != nil {
				return err
			}
			if s.root != "" && s.state.Mode != "InvalidWorkspace" {
				s.restoreObserverLockLocked(s.state, "Unable to restore observer lock after releasing writer mode.")
				s.events.emit(EventWorkspaceChanged, s.state)
			}
		}
		s.events.emit(EventWriterStatusChanged, s.writerHost.Status())
		return nil
	}

	ctx := context.Background()
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	if err := s.writerHost.Stop(ctx); err != nil {
		s.events.emit(EventWriterStatusChanged, s.writerHost.Status())
		return err
	}

	if err := s.releaseLockLocked(); err != nil {
		s.events.emit(EventWriterStatusChanged, s.writerHost.Status())
		return err
	}

	if s.root != "" && s.state.Mode != "InvalidWorkspace" {
		s.restoreObserverLockLocked(s.state, "Unable to restore observer lock after writer shutdown.")
	}
	s.events.emit(EventWriterStatusChanged, s.writerHost.Status())
	s.events.emit(EventWorkspaceChanged, s.state)
	return nil
}

func (s *Session) getWriterStatus() viewmodel.WriterStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.writerHost.Status()
}

func (s *Session) resetWorkspaceLocked() error {
	if err := s.stopWriterLocked(); err != nil {
		return err
	}
	s.cancelTasksLocked()
	return s.releaseLockLocked()
}

func (s *Session) hasBlockingTasksLocked() bool {
	for _, record := range s.tasks {
		if !record.vm.CanCancel {
			return true
		}
	}
	return false
}

func (s *Session) releaseLockLocked() error {
	if s.lockHandle == nil {
		s.lockMode = ""
		return nil
	}
	err := s.lockHandle.Close()
	s.lockHandle = nil
	s.lockMode = ""
	return err
}

func (s *Session) stopWriterLocked() error {
	status := s.writerHost.Status()
	switch status.LifecycleState {
	case string(writer.LifecycleRunning), string(writer.LifecycleStarting), string(writer.LifecycleStopping):
	default:
		return nil
	}

	if err := s.writerHost.Stop(context.Background()); err != nil {
		s.events.emit(EventWriterStatusChanged, s.writerHost.Status())
		return err
	}
	s.events.emit(EventWriterStatusChanged, s.writerHost.Status())
	return nil
}

func (s *Session) restoreObserverLockLocked(baseState viewmodel.WorkspaceState, reason string) {
	if s.root == "" || baseState.Mode == "InvalidWorkspace" {
		s.lockHandle = nil
		s.lockMode = ""
		s.state = baseState
		return
	}

	handle, err := lock.Acquire(s.root, lock.ModeObserverShared, lock.Metadata{Program: "cache-desktop"})
	if err != nil {
		s.lockHandle = nil
		s.lockMode = ""
		s.state = degradedState(baseState, reason)
		return
	}

	s.lockHandle = handle
	s.lockMode = lock.ModeObserverShared
	s.state = observerState(baseState)
}

func observerState(state viewmodel.WorkspaceState) viewmodel.WorkspaceState {
	state.Mode = "HealthyObserver"
	state.LockMode = string(lock.ModeObserverShared)
	state.CanRefresh = true
	state.CanRunVerify = true
	state.CanRunCloseCheck = true
	state.CanRunRepairTail = false
	state.CanRunShutdown = false
	state.Reason = ""
	return state
}

func writerState(state viewmodel.WorkspaceState) viewmodel.WorkspaceState {
	state.Mode = "HealthyWriter"
	state.LockMode = string(lock.ModeWriterExclusive)
	state.CanRefresh = true
	state.CanRunVerify = true
	state.CanRunCloseCheck = true
	state.CanRunRepairTail = false
	state.CanRunShutdown = false
	state.Reason = ""
	return state
}

func maintenanceState(state viewmodel.WorkspaceState) viewmodel.WorkspaceState {
	state.Mode = "HealthyMaintenance"
	state.LockMode = string(lock.ModeMaintenanceExclusive)
	state.CanRefresh = true
	state.CanRunVerify = true
	state.CanRunCloseCheck = true
	state.CanRunRepairTail = true
	state.CanRunShutdown = true
	state.Reason = ""
	return state
}

func degradedState(state viewmodel.WorkspaceState, reason string) viewmodel.WorkspaceState {
	state.Mode = "DegradedReadOnly"
	state.LockMode = "N/A"
	state.CanRefresh = true
	state.CanRunVerify = true
	state.CanRunCloseCheck = true
	state.CanRunRepairTail = false
	state.CanRunShutdown = false
	state.Reason = reason
	return state
}
