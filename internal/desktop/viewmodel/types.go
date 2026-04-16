package viewmodel

type WorkspaceState struct {
	RootPath         string
	Mode             string
	LockMode         string
	Health           string
	CanRefresh       bool
	CanRunVerify     bool
	CanRunCloseCheck bool
	CanRunRepairTail bool
	CanRunShutdown   bool
	Reason           string
}

type Overview struct {
	RootPath                  string
	WorkspaceMode             string
	LockMode                  string
	Health                    string
	SegmentCount              int
	ActiveSegmentID           uint64
	ActiveSegmentSizeBytes    int64
	WALSizeBytes              int64
	NextWriteSeq              uint64
	RetentionDays             int
	CheckpointsTotal          uint64
	SegmentFsyncTotal         uint64
	LastAckedWriteSeq         uint64
	GracefulShutdownsTotal    uint64
	UngracefulRecoveriesTotal uint64
	SegmentTailRepairsTotal   uint64
	BacklogEstimateRecords    *uint64
	BacklogEstimateBytes      *uint64
	Warnings                  []Warning
	LastRefreshedAt           int64
	IsStale                   bool
}

type SegmentListItem struct {
	SegmentID     uint64
	FileName      string
	SizeBytes     int64
	Sealed        bool
	FirstWriteSeq uint64
	LastWriteSeq  uint64
	RecordCount   uint64
	MinEventTime  int64
	MaxEventTime  int64
	LastBatchSeq  uint64
	Health        string
}

type SegmentDetail struct {
	SegmentID         uint64
	Path              string
	SizeBytes         int64
	Sealed            bool
	FooterStatus      string
	TailStatus        string
	FirstWriteSeq     uint64
	LastWriteSeq      uint64
	RecordCount       uint64
	BlockCount        uint64
	MinEventTime      int64
	MaxEventTime      int64
	LastBatchSeq      uint64
	RawPreviewHex     string
	StructuredPreview []KeyValue
}

type CursorSummary struct {
	Destination string
	WriteSeq    uint64
	UpdatedAt   int64
	Status      string
}

type CursorDetail struct {
	Destination  string
	SegmentID    uint64
	BlockOffset  uint64
	RecordIndex  uint32
	WriteSeq     uint64
	UpdatedAt    int64
	CRCStatus    string
	BackupStatus string
}

type CheckpointDetail struct {
	LastBatchSeq     uint64
	LastWALEndOffset int64
	UpdatedAt        int64
	Version          uint32
	IntegrityStatus  string
}

type WALDetail struct {
	Path          string
	SizeBytes     int64
	FirstBatchSeq uint64
	LastBatchSeq  uint64
	LastEndOffset int64
	Health        string
}

type ConfigField struct {
	Name         string
	DisplayName  string
	CurrentValue string
	DefaultValue string
	AllowedRange string
	StartupOnly  bool
	Description  string
}

type Config struct {
	RootDir                ConfigField
	SegmentTargetSizeBytes ConfigField
	SegmentSlackSizeBytes  ConfigField
	BlockTargetSizeBytes   ConfigField
	CheckpointInterval     ConfigField
	CheckpointBytes        ConfigField
	SegmentFsyncInterval   ConfigField
	SegmentFsyncBytes      ConfigField
	RetentionDays          ConfigField
}

type GUIError struct {
	Code            string
	Title           string
	Message         string
	Operation       string
	Path            string
	Recoverable     bool
	Details         string
	SuggestedAction string
}

type Warning struct {
	Code     string
	Severity string
	Title    string
	Message  string
}

type KeyValue struct {
	Key   string
	Value string
}

type OperationResult struct {
	Summary string
	Details []KeyValue
	Changed bool
}

type ConfirmDialog struct {
	Title               string
	RiskLevel           string
	Summary             string
	ImpactLines         []string
	ConfirmLabel        string
	CancelLabel         string
	RequiresTypedPhrase *string
}

type Task struct {
	TaskID          string
	Kind            string
	Target          string
	Status          string
	Phase           string
	StartedAt       int64
	UpdatedAt       int64
	ProgressCurrent *uint64
	ProgressTotal   *uint64
	Message         string
	CanCancel       bool
	Result          *OperationResult
	Error           *GUIError
}

type PagedSegments struct {
	Items      []SegmentListItem
	Page       int
	PageSize   int
	TotalItems int
	HasNext    bool
}
