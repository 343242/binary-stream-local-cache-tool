package backend

import "sync"

const (
	EventTaskStarted    = "task:started"
	EventTaskProgress   = "task:progress"
	EventTaskFinished   = "task:finished"
	EventWorkspaceChanged = "workspace:changed"
)

type EventEmitter func(name string, payload any)

type Hooks struct {
	BeforeOpenWorkspace  func()
	BeforeCloseWorkspace func()
}

type eventBus struct {
	mu      sync.RWMutex
	emitter EventEmitter
}

func (b *eventBus) emit(name string, payload any) {
	b.mu.RLock()
	emitter := b.emitter
	b.mu.RUnlock()
	if emitter != nil {
		emitter(name, payload)
	}
}

func (b *eventBus) setEmitter(emitter EventEmitter) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.emitter = emitter
}
