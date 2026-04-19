package writer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"fastReadFile/internal/core"
	"fastReadFile/internal/desktop/service"
	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/pkg/cache"
)

const (
	workspaceStateHealthyObserver = "HealthyObserver"
	workspaceStateHealthyWriter   = "HealthyWriter"
	workspaceStateOpening         = "Opening"
)

type WriterStatus = viewmodel.WriterStatus

type WriterHost interface {
	Start(ctx context.Context, root string, cfg core.Config) error
	Stop(ctx context.Context) error
	Status() WriterStatus
	Submit(ctx context.Context, records []core.RawRecord) error
}

type host struct {
	mu            sync.RWMutex
	engine        *cache.StorageEngine
	status        WriterStatus
	queue         chan []core.RawRecord
	queueCapacity int
	stopCh        chan struct{}
	workerCh      chan struct{}
}

var _ WriterHost = (*host)(nil)

func NewHost() *host {
	return NewHostWithQueueCapacity(BatchQueueCapacity)
}

func NewHostWithQueueCapacity(capacity int) *host {
	if capacity <= 0 {
		capacity = 1
	}
	return &host{
		queueCapacity: capacity,
		status: WriterStatus{
			LifecycleState: string(LifecycleNotStarted),
		},
		queue: make(chan []core.RawRecord, capacity),
	}
}

func (h *host) Start(ctx context.Context, root string, cfg core.Config) error {
	ctx = normalizeContext(ctx)

	h.mu.Lock()
	if h.engine != nil {
		h.mu.Unlock()
		return errors.New("writer host already running")
	}
	queue := make(chan []core.RawRecord, h.queueCapacity)
	h.queue = queue
	h.status = WriterStatus{
		LifecycleState: string(LifecycleStarting),
		WorkspaceState: workspaceStateOpening,
		RootPath:       root,
	}
	h.mu.Unlock()

	if err := ctx.Err(); err != nil {
		h.failStart(root, err)
		return err
	}
	if err := service.InitializeWorkspace(root); err != nil {
		h.failStart(root, err)
		return err
	}

	cfg.RootDir = root
	engine, err := cache.Open(cfg)
	if err != nil {
		h.failStart(root, err)
		return err
	}

	stopCh := make(chan struct{})
	workerCh := make(chan struct{})

	h.mu.Lock()
	h.engine = engine
	h.stopCh = stopCh
	h.workerCh = workerCh
	h.status = WriterStatus{
		LifecycleState:  string(LifecycleRunning),
		WorkspaceState:  workspaceStateHealthyWriter,
		RootPath:        root,
		StartedAtUnixMs: time.Now().UnixMilli(),
	}
	h.mu.Unlock()

	go h.runIngestionLoop(engine, queue, stopCh, workerCh)
	return nil
}

func (h *host) Stop(ctx context.Context) error {
	ctx = normalizeContext(ctx)

	h.mu.Lock()
	engine := h.engine
	stopCh := h.stopCh
	workerCh := h.workerCh
	root := h.status.RootPath
	startedAtUnixMs := h.status.StartedAtUnixMs
	if engine == nil {
		h.status.LifecycleState = string(LifecycleStopped)
		h.status.WorkspaceState = workspaceStateHealthyObserver
		h.status.StoppedAtUnixMs = time.Now().UnixMilli()
		h.queue = make(chan []core.RawRecord, h.queueCapacity)
		h.mu.Unlock()
		return nil
	}
	h.status.LifecycleState = string(LifecycleStopping)
	h.stopCh = nil
	h.workerCh = nil
	h.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	if workerCh != nil {
		select {
		case <-workerCh:
		case <-ctx.Done():
			h.mu.Lock()
			h.status.LastError = ctx.Err().Error()
			h.mu.Unlock()
			return ctx.Err()
		}
	}
	if err := engine.Shutdown(ctx); err != nil {
		h.mu.Lock()
		h.status.LastError = err.Error()
		h.mu.Unlock()
		return err
	}

	h.mu.Lock()
	h.engine = nil
	h.queue = make(chan []core.RawRecord, h.queueCapacity)
	h.status = WriterStatus{
		LifecycleState:  string(LifecycleStopped),
		WorkspaceState:  workspaceStateHealthyObserver,
		RootPath:        root,
		StartedAtUnixMs: startedAtUnixMs,
		StoppedAtUnixMs: time.Now().UnixMilli(),
	}
	h.mu.Unlock()
	return nil
}

func (h *host) Status() WriterStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status
}

func (h *host) Submit(ctx context.Context, records []core.RawRecord) error {
	ctx = normalizeContext(ctx)

	select {
	case h.queue <- records:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *host) DebugFillQueueForTest() {
	select {
	case h.queue <- []core.RawRecord{{EventTimeUnixMs: 1, Payload: []byte("debug")}}:
	default:
	}
}

func (h *host) DebugQueueLenForTest() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.queue)
}

func (h *host) runIngestionLoop(engine *cache.StorageEngine, queue <-chan []core.RawRecord, stopCh <-chan struct{}, workerCh chan<- struct{}) {
	defer close(workerCh)

	for {
		select {
		case <-stopCh:
			return
		case records := <-queue:
			if len(records) == 0 {
				continue
			}
			if _, err := engine.WriteBatch(context.Background(), records); err != nil {
				h.mu.Lock()
				h.status.LastError = fmt.Sprintf("write batch failed: %v", err)
				h.mu.Unlock()
			}
		}
	}
}

func (h *host) failStart(root string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = WriterStatus{
		LifecycleState: string(LifecycleStartFailed),
		RootPath:       root,
		LastError:      err.Error(),
	}
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
