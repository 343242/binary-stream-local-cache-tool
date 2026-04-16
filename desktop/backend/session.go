package backend

import (
	"fmt"
	"sync"

	"fastReadFile/internal/desktop/service"
	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/lock"
)

type Session struct {
	mu         sync.RWMutex
	generation uint64
	state      viewmodel.WorkspaceState
	root       string
	lockHandle *lock.Handle
	lockMode   lock.Mode
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
		tasks:  make(map[string]*taskRecord),
		hooks:  hooks,
		events: events,
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
	} else if state.Mode == "InvalidWorkspace" {
		s.state = state
	} else {
		s.state = observerState(state)
	}
	s.events.emit(EventWorkspaceChanged, s.state)
	return nil
}

func (s *Session) resetWorkspaceLocked() error {
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

func observerState(state viewmodel.WorkspaceState) viewmodel.WorkspaceState {
	state.Mode = "HealthyObserver"
	state.LockMode = string(lock.ModeObserverShared)
	state.CanRefresh = true
	state.CanRunVerify = true
	state.CanRunCloseCheck = true
	state.CanRunRepairTail = false
	state.CanRunShutdown = false
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
