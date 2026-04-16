package backend

import (
	"context"
	"fmt"
	"time"

	"fastReadFile/internal/desktop/viewmodel"
)

type taskRecord struct {
	vm     viewmodel.Task
	cancel context.CancelFunc
}

func (s *Session) startTask(kind, target string, canCancel bool, cancel context.CancelFunc) viewmodel.Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.taskSeq++
	taskID := fmt.Sprintf("task-%d", s.taskSeq)
	now := time.Now().UnixMilli()
	task := viewmodel.Task{
		TaskID:    taskID,
		Kind:      kind,
		Target:    target,
		Status:    "running",
		StartedAt: now,
		UpdatedAt: now,
		CanCancel: canCancel,
	}
	s.tasks[taskID] = &taskRecord{vm: task, cancel: cancel}
	s.events.emit(EventTaskStarted, task)
	return task
}

func (s *Session) updateTaskProgress(taskID, phase, message string, current, total *uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	var progressCurrent *uint64
	if current != nil {
		value := *current
		progressCurrent = &value
	}
	var progressTotal *uint64
	if total != nil {
		value := *total
		progressTotal = &value
	}
	record.vm.Phase = phase
	record.vm.Message = message
	record.vm.ProgressCurrent = progressCurrent
	record.vm.ProgressTotal = progressTotal
	record.vm.UpdatedAt = time.Now().UnixMilli()
	s.events.emit(EventTaskProgress, record.vm)
	return nil
}

func (s *Session) finishTask(taskID string, status string, result *viewmodel.OperationResult, guiErr *viewmodel.GUIError) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	record.vm.Status = status
	record.vm.Result = result
	record.vm.Error = guiErr
	record.vm.UpdatedAt = time.Now().UnixMilli()
	s.events.emit(EventTaskFinished, record.vm)
	delete(s.tasks, taskID)
	return nil
}

func (s *Session) cancelTasksLocked() bool {
	blocked := false
	for taskID, record := range s.tasks {
		if !record.vm.CanCancel {
			blocked = true
			continue
		}
		if record.cancel != nil {
			record.cancel()
		}
		record.vm.Status = "cancelled"
		record.vm.UpdatedAt = time.Now().UnixMilli()
		s.events.emit(EventTaskFinished, record.vm)
		delete(s.tasks, taskID)
	}
	return blocked
}

func (s *Session) cancelTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	if !record.vm.CanCancel {
		return fmt.Errorf("task %s is not cancellable", taskID)
	}
	if record.cancel != nil {
		record.cancel()
	}
	record.vm.Status = "cancelled"
	record.vm.UpdatedAt = time.Now().UnixMilli()
	s.events.emit(EventTaskFinished, record.vm)
	delete(s.tasks, taskID)
	return nil
}
