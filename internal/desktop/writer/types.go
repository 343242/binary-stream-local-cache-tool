package writer

import "fastReadFile/internal/core"

const (
	EventBufferSize       = 200
	EventRateLimitPerSec  = 10
	WarningDedupeWindowMs = 5000
	BatchQueueCapacity    = 64
)

type LifecycleState string

const (
	LifecycleNotStarted  LifecycleState = "not-started"
	LifecycleStarting    LifecycleState = "starting"
	LifecycleRunning     LifecycleState = "running"
	LifecycleStopping    LifecycleState = "stopping"
	LifecycleStopped     LifecycleState = "stopped"
	LifecycleStartFailed LifecycleState = "start-failed"
)

type PendingConfig struct {
	Config core.Config
	Root   string
}
