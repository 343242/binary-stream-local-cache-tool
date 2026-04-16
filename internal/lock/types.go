package lock

type Mode string

const (
	ModeWriterExclusive      Mode = "WriterExclusive"
	ModeObserverShared       Mode = "ObserverShared"
	ModeMaintenanceExclusive Mode = "MaintenanceExclusive"
)

type Metadata struct {
	Mode      Mode   `json:"mode"`
	Program   string `json:"program"`
	PID       int    `json:"pid"`
	Hostname  string `json:"hostname"`
	StartedAt int64  `json:"started_at"`
	UpdatedAt int64  `json:"updated_at"`
}
