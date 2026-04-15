# Desktop GUI Design

## 1. Goal

Build a Windows/Linux desktop management application for the binary stream local cache tool.

Phase 1 is a local, embedded-device-oriented desktop GUI:

- Runs as an independent windowed desktop program
- Opens one local cache root directly
- Prioritizes:
  - status overview
  - storage state browsing
  - effective configuration inspection
  - guarded maintenance operations
- Defaults to safe, observer-first behavior
- Requires explicit confirmation dialogs before any state-changing action

Phase 1 is not a remote management console and is not a replacement for the storage engine itself.

## 2. Product Decisions Locked In

The following decisions are fixed for phase 1:

- Desktop shell technology: `Wails v2`
- Frontend stack: `React 18 + TypeScript + Vite`
- State model: one opened cache root per app session
- Directory access model: direct local filesystem access
- Concurrency safety model: phase 1 refuses mutable operations unless an exclusive maintenance lock is held
- Configuration page: read-only inspection in phase 1
- Theme support: light and dark themes, default follows OS setting

## 3. Scope

### 3.1 In Scope

- Windows support
- Linux support
- Wails-based desktop application
- Empty-state landing page before any directory is opened
- One opened cache root per app session
- Overview dashboard
- Explorer for segment / WAL / cursor / checkpoint
- Read-only configuration page
- Guarded operations page
- Shared cross-process lock foundation used by engine, CLI, and GUI
- Background loading, progress, cancellation, and timeout handling
- Multi-process conflict detection and safe-mode gating
- Structured error reporting

### 3.2 Out of Scope

- macOS support
- Web deployment
- Remote service/API mode
- Multi-directory workspace
- Multi-tab workspace
- TSDB-like arbitrary record query
- Real-time terminal emulator
- Fleet/multi-node management
- Persistent configuration editing

## 4. Why Phase 1 Config Is Read-Only

The current storage engine accepts `Config` at `Open(cfg)` time and does not expose any persisted configuration store or runtime update API.

That means a phase-1 `SaveConfig()` feature would be speculative and unsafe.

Phase 1 therefore limits the Config page to:

- showing the effective configuration currently used by the opened workspace
- showing the default values and allowed ranges
- showing whether each field is startup-only
- exporting/copying the effective configuration snapshot for operator reference

Persistent configuration editing is explicitly deferred until the engine gains a real persisted config contract.

## 5. Technical Approach

### 5.1 Recommended option

Use `Wails v2`.

This choice is based on the official documentation available on `2026-04-15`, which presents the current docs as `v2.12.0` and documents native desktop app development with Go plus web frontend templates.

### 5.2 Frontend stack

Phase 1 frontend stack:

- `React 18`
- `TypeScript 5`
- `Vite`
- `Zustand` for application state
- plain CSS modules plus CSS custom properties for theming

Rationale:

- React + TypeScript has the smoothest hiring/debugging path
- Zustand is enough for one-window desktop state without adding Redux-scale ceremony
- CSS modules plus design tokens keep styling explicit and contained

### 5.3 Rejected options

- `Fyne`
  - rejected because it is a weaker fit for the intended dense operations-console layout
- `Desktop GUI + separate local management service`
  - rejected for phase 1 because it expands the scope with service lifecycle and transport design

## 6. High-Level Architecture

The application is split into five modules.

### 6.1 `desktop/app`

Responsibilities:

- Wails bootstrap
- main window lifecycle
- menu wiring
- native file chooser
- native confirmation dialogs
- theme initialization

### 6.2 `desktop/backend`

Responsibilities:

- expose structured query/command methods to the frontend
- orchestrate session state, background jobs, and validation
- map internal/storage errors into GUI-facing error models

Hard rule:

- this layer must not use `cachectl` stdout as a data source

### 6.3 `desktop/session`

Responsibilities:

- hold the currently opened root
- track workspace mode
- manage loading state
- track long-running tasks
- track stale/fresh snapshots

### 6.4 `desktop/viewmodel`

Responsibilities:

- map raw engine/ops state into UI-oriented, versioned models
- keep frontend insulated from storage-file details

### 6.5 `desktop/frontend`

Responsibilities:

- shell layout
- navigation
- cards/tables/panes/forms
- loading skeletons
- toasts
- task panels
- accessibility and keyboard behavior

## 7. Workspace State Machine

Phase 1 uses an explicit workspace state machine.

### 7.1 States

- `NoWorkspace`
  - app started, no directory opened yet
- `Opening`
  - validation and initial snapshot load are running
- `HealthyObserver`
  - directory is valid and readable
  - no mutating action is currently allowed
- `HealthyMaintenance`
  - directory is valid
  - exclusive maintenance lock is held
  - allowed guarded write operations may run
- `DegradedReadOnly`
  - directory is partially readable, but writes are unsafe or impossible
- `InvalidWorkspace`
  - path is invalid or cannot be opened as a cache root

### 7.2 Degraded mode triggers

The session enters `DegradedReadOnly` when any of the following is true:

- filesystem is read-only
- write permission is missing
- partial corruption is detected but readable prefixes remain available
- checkpoint/cursor/segment metadata can be inspected but not trusted for mutation
- exclusive maintenance lock cannot be acquired
- background validation times out but a partial snapshot is available

The session enters `InvalidWorkspace` when any of the following is true:

- root path does not exist
- required cache subdirectories cannot be recognized
- critical metadata is unreadable and no safe partial read path exists
- version/protocol mismatch prevents safe interpretation

### 7.3 Empty state

When the app launches in `NoWorkspace`, the main window shows a dedicated landing state:

- application title and version
- “Open Cache Directory” primary action
- recent directories list, max 5
- short explanation of supported directory shape
- warning that live writer directories will open in observer mode only or be refused for mutation

No blank screen is permitted.

## 8. Multi-Process Safety Model

The current storage engine is not safe for multi-process concurrent writers.

Phase 1 therefore requires a shared lock protocol across the engine, GUI, and CLI tooling.

This lock protocol is a phase-0 prerequisite for GUI implementation, not a GUI-only concern.

### 8.0 Ownership and placement

The lock implementation must live in a shared package:

- `internal/lock/`

It must be consumed by:

- `pkg/cache` for normal writer lifecycle
- `cachectl` for observer and maintenance operations
- the desktop backend for observer and maintenance operations

The GUI must not implement an independent lock mechanism.

### 8.1 Lock file

Use a lock file at:

- `meta/workspace.lock`

The file has two roles:

- lock carrier for the OS-level lock handle
- human-readable diagnostic metadata such as:
  - `pid`
  - `program`
  - `mode`
  - `hostname`
  - `started_at`
  - `updated_at`

Diagnostic metadata is advisory only. Correctness comes from the OS-level lock, not from parsing file contents.

### 8.2 Lock modes

- `WriterExclusive`
  - held by the storage engine process during normal write operation
- `ObserverShared`
  - held by GUI or CLI for read-only inspection
- `MaintenanceExclusive`
  - held by GUI or CLI for guarded mutating operations when no writer is present

### 8.3 OS-level implementation

The lock backend must use native OS file locking, not an in-memory mutex and not lock-file-content polling.

Required implementation strategy:

- Linux:
  - use `flock`
- Windows:
  - use `LockFileEx`

The lock must remain valid only while the owning process keeps the file descriptor/handle open.

This matches the intended crash behavior:

- if the owning process exits normally, the lock is released by the OS
- if the owning process crashes, the lock is released by the OS
- the lock file itself may remain on disk, but stale file contents must not be treated as an active lock

### 8.4 Platform scope and restrictions

Phase 1 lock semantics are only guaranteed on local filesystems.

Unsupported or degraded targets include:

- NFS
- SMB/CIFS
- other network-mounted filesystems with weaker or inconsistent locking semantics

If the workspace root is on a non-local or unverified filesystem, the GUI must:

- surface a warning
- refuse maintenance mode
- allow observer mode only if safe reads can still be established

### 8.5 Rules

- `WriterExclusive` blocks `MaintenanceExclusive`
- `WriterExclusive` may coexist with `ObserverShared` only if the engine explicitly supports observer-safe reads
- if observer-safe reads cannot be guaranteed for a given operation, the GUI must disable that operation while a writer is active
- any mutating GUI action requires `MaintenanceExclusive`
- if the writer lock is present, `repair-tail`, `shutdown`, and any future write-like action are disabled

Additional rules:

- lock mode upgrade from `ObserverShared` to `MaintenanceExclusive` must not be done in place
- the caller must release `ObserverShared` first, then acquire `MaintenanceExclusive`
- if the exclusive acquire fails, the caller returns to observer mode or surfaces the conflict
- maintenance lock acquisition must fail fast rather than block indefinitely
- lock acquisition failures must return a structured conflict error, not a generic I/O error

### 8.6 Crash and stale-lock handling

Phase 1 must not implement a separate "stale lock cleanup" command for active lock ownership.

Required behavior:

- stale metadata may be overwritten only after a new OS-level lock has been successfully acquired
- lock recovery is therefore implicit through OS handle release
- there is no manual "force unlock" in phase 1

This avoids split-brain behavior caused by deleting a lock file that still has a live owner.

### 8.7 Phase-1 safety stance

Because direct local-directory mode is the initial transport, phase 1 must prefer refusal over unsafe optimism.

If the lock state is ambiguous:

- the GUI opens read-only if safe read paths exist
- otherwise the GUI refuses the workspace and surfaces the reason

## 9. Data Access and Refresh Model

### 9.1 Refresh policy

- Overview:
  - auto-refresh every 5 seconds while the window is focused
  - manual refresh always available
- Explorer:
  - manual refresh by default
  - selected-detail panes can refresh independently
- Config:
  - manual reload only
- Operations:
  - task progress is event-driven while a task is running

Phase 1 does not use file-system watch events.

### 9.2 Staleness indicator

All pages that render data snapshots must show:

- `last refreshed at`
- `refresh in progress`
- `data may be stale` indicator if the snapshot age exceeds 15 seconds

### 9.3 Loading thresholds

- if a query returns within 300ms, no skeleton is shown
- if it exceeds 300ms, render loading skeletons
- if it exceeds 2 seconds, show progress text
- if it exceeds the page timeout, keep partial data if available and surface a timeout warning

### 9.4 Page timeouts

- `OpenWorkspace`: 10 seconds
- `GetOverview`: 3 seconds
- `ListSegments`: 10 seconds
- `GetSegmentDetail`: 5 seconds
- `GetWALDetail`: 5 seconds
- `GetCursorDetail`: 3 seconds
- `GetCheckpointDetail`: 3 seconds
- `RunVerify`: no hard timeout, cancellable
- `RunRepairTail`: no hard timeout, cancellable until irreversible repair begins
- `RunCloseCheck`: 30 seconds, cancellable
- `RunShutdown`: non-cancellable after confirmation

## 10. Large Cache Strategy

The GUI must remain usable against large caches, including multi-hundred-GB and multi-TB roots with thousands of segments.

### 10.1 Overview loading strategy

Overview uses a two-stage load:

1. fast snapshot
   - directory identity
   - lock state
   - current health state
   - segment count
   - active segment size
   - WAL size
2. background enrichment
   - backlog estimates
   - replay/ack summaries
   - warning synthesis

### 10.2 Explorer loading strategy

- segment list is paginated at 200 rows per page
- virtualized rendering activates when the total row count exceeds 500
- detail panels load lazily only after row selection
- raw preview is capped at the first 64 KiB until the user explicitly expands it

### 10.3 Cancellation

Long-running list/detail loads must accept cancellation from:

- page switch
- workspace close
- manual cancel action
- app shutdown

## 11. Information Architecture

Phase 1 contains four primary pages.

### 11.1 Overview

Required fields:

- root path
- workspace mode
- lock state
- health state
- total segment count
- active segment id
- active segment size bytes
- WAL size bytes
- next write sequence
- retention days
- last acked write sequence
- checkpoint count
- segment fsync total
- graceful shutdown total
- ungraceful recovery total
- segment tail repair total
- last refresh timestamp

Allowed derived fields:

- backlog estimated record count
- backlog estimated bytes
- warning summary

### 11.2 Explorer

Required tabs:

- Segments
- WAL
- Cursors
- Checkpoint

Segments list required columns:

- segment id
- file size bytes
- sealed flag
- first write sequence
- last write sequence
- record count
- min event time
- max event time
- last batch sequence
- health marker

Segment detail required fields:

- header/footer health
- block count
- repair status
- byte ranges
- raw/structured preview toggle

WAL detail required fields:

- active wal path
- file size bytes
- first batch sequence
- last batch sequence
- last end offset
- health marker

Cursor detail required fields:

- destination
- write sequence
- segment id
- block offset
- record index
- updated at
- crc status
- backup status

Checkpoint detail required fields:

- last batch sequence
- last wal end offset
- updated at
- version
- integrity status

### 11.3 Config

Phase-1 config page is read-only.

Required fields:

- root dir
- segment target size bytes
- segment slack size bytes
- block target size bytes
- checkpoint interval
- checkpoint bytes
- segment fsync interval
- segment fsync bytes
- retention days

Each field must display:

- current effective value
- default value
- allowed range
- whether it is startup-only

### 11.4 Operations

Required actions:

- `verify`
- `close-check`

Optional-if-implemented in phase 1:

- `repair-tail`
- `shutdown`

All optional actions still require the same safety and confirmation model if they are shipped.

## 12. ViewModel Contracts

The following view models are mandatory in phase 1.

### 12.1 `WorkspaceStateVM`

- `rootPath string`
- `mode string`
- `lockMode string`
- `health string`
- `canRefresh bool`
- `canRunVerify bool`
- `canRunCloseCheck bool`
- `canRunRepairTail bool`
- `canRunShutdown bool`
- `reason string`

### 12.2 `OverviewVM`

- `rootPath string`
- `workspaceMode string`
- `lockMode string`
- `health string`
- `segmentCount int`
- `activeSegmentID uint64`
- `activeSegmentSizeBytes int64`
- `walSizeBytes int64`
- `nextWriteSeq uint64`
- `retentionDays int`
- `checkpointsTotal uint64`
- `segmentFsyncTotal uint64`
- `lastAckedWriteSeq uint64`
- `gracefulShutdownsTotal uint64`
- `ungracefulRecoveriesTotal uint64`
- `segmentTailRepairsTotal uint64`
- `backlogEstimateRecords *uint64`
- `backlogEstimateBytes *uint64`
- `warnings []WarningVM`
- `lastRefreshedAt int64`
- `isStale bool`

### 12.3 `SegmentListItemVM`

- `segmentID uint64`
- `fileName string`
- `sizeBytes int64`
- `sealed bool`
- `firstWriteSeq uint64`
- `lastWriteSeq uint64`
- `recordCount uint64`
- `minEventTime int64`
- `maxEventTime int64`
- `lastBatchSeq uint64`
- `health string`

### 12.4 `SegmentDetailVM`

- `segmentID uint64`
- `path string`
- `sizeBytes int64`
- `sealed bool`
- `headerStatus string`
- `footerStatus string`
- `repairStatus string`
- `firstWriteSeq uint64`
- `lastWriteSeq uint64`
- `recordCount uint64`
- `blockCount uint64`
- `minEventTime int64`
- `maxEventTime int64`
- `lastBatchSeq uint64`
- `rawPreviewHex string`
- `structuredPreview []KeyValueVM`

### 12.5 `CursorDetailVM`

- `destination string`
- `segmentID uint64`
- `blockOffset uint64`
- `recordIndex uint32`
- `writeSeq uint64`
- `updatedAt int64`
- `crcStatus string`
- `backupStatus string`

### 12.6 `CheckpointDetailVM`

- `lastBatchSeq uint64`
- `lastWALEndOffset int64`
- `updatedAt int64`
- `version uint32`
- `integrityStatus string`

### 12.7 `ConfigVM`

- `rootDir ConfigFieldVM`
- `segmentTargetSizeBytes ConfigFieldVM`
- `segmentSlackSizeBytes ConfigFieldVM`
- `blockTargetSizeBytes ConfigFieldVM`
- `checkpointInterval ConfigFieldVM`
- `checkpointBytes ConfigFieldVM`
- `segmentFsyncInterval ConfigFieldVM`
- `segmentFsyncBytes ConfigFieldVM`
- `retentionDays ConfigFieldVM`

### 12.8 `TaskVM`

- `taskID string`
- `kind string`
- `target string`
- `status string`
- `phase string`
- `startedAt int64`
- `updatedAt int64`
- `progressCurrent *uint64`
- `progressTotal *uint64`
- `message string`
- `canCancel bool`
- `result *OperationResultVM`
- `error *GUIError`

### 12.9 `WALDetailVM`

- `path string`
- `sizeBytes int64`
- `firstBatchSeq uint64`
- `lastBatchSeq uint64`
- `lastEndOffset int64`
- `health string`

### 12.10 `CursorSummaryVM`

- `destination string`
- `writeSeq uint64`
- `updatedAt int64`
- `status string`

### 12.11 `PagedSegmentsVM`

- `items []SegmentListItemVM`
- `page int`
- `pageSize int`
- `totalItems int`
- `hasNext bool`

### 12.12 `ConfigFieldVM`

- `name string`
- `displayName string`
- `currentValue string`
- `defaultValue string`
- `allowedRange string`
- `startupOnly bool`
- `description string`

### 12.13 `WarningVM`

- `code string`
- `severity string`
- `title string`
- `message string`

### 12.14 `KeyValueVM`

- `key string`
- `value string`

### 12.15 `OperationResultVM`

- `summary string`
- `details []KeyValueVM`
- `changed bool`

## 13. Error Contract

The backend must return a structured `GUIError`.

### 13.1 `GUIError`

- `code string`
- `title string`
- `message string`
- `operation string`
- `path string`
- `recoverable bool`
- `details string`
- `suggestedAction string`

### 13.2 Required error codes

- `GUI_ERR_INVALID_ROOT`
- `GUI_ERR_PERMISSION_DENIED`
- `GUI_ERR_READ_ONLY`
- `GUI_ERR_LOCK_CONFLICT`
- `GUI_ERR_PARTIAL_CORRUPTION`
- `GUI_ERR_UNSUPPORTED_VERSION`
- `GUI_ERR_TIMEOUT`
- `GUI_ERR_TASK_CANCELLED`
- `GUI_ERR_OPERATION_BLOCKED`
- `GUI_ERR_INTERNAL`

## 14. Async Task Model

Long-running actions must use a task protocol.

### 14.1 Task lifecycle

- `queued`
- `running`
- `succeeded`
- `failed`
- `cancelled`

### 14.2 Commands that must use tasks

- `RunVerify`
- `RunCloseCheck`
- `RunRepairTail`
- `RunShutdown`
- background overview enrichment
- large explorer loads that exceed 2 seconds

### 14.3 Progress delivery

Backend must push task updates to the frontend using Wails eventing.

Required event names:

- `task:started`
- `task:progress`
- `task:finished`
- `workspace:changed`

## 15. Configuration Reference and Validation Ranges

Even though phase-1 config is read-only, the GUI must display the legal range for each field.

Validation ranges:

- `SegmentTargetSizeBytes`
  - minimum: `64 MiB`
  - maximum: `4 GiB`
- `SegmentSlackSizeBytes`
  - minimum: `1 MiB`
  - maximum: `64 MiB`
  - must be smaller than `SegmentTargetSizeBytes`
- `BlockTargetSizeBytes`
  - minimum: `256 KiB`
  - maximum: `4 MiB`
- `CheckpointInterval`
  - minimum: `1s`
  - maximum: `60s`
- `CheckpointBytes`
  - minimum: `4 MiB`
  - maximum: `1 GiB`
- `SegmentFsyncInterval`
  - minimum: `10ms`
  - maximum: `5s`
- `SegmentFsyncBytes`
  - minimum: `1 MiB`
  - maximum: `64 MiB`
- `RetentionDays`
  - minimum: `1`
  - maximum: `365`

These ranges become the required validation rules when persisted configuration editing is introduced later.

## 16. Visual Design Specification

“Operations console” is not enough on its own, so phase 1 defines concrete design tokens.

### 16.1 Typography

- primary UI font: `IBM Plex Sans`
- mono font: `JetBrains Mono`

### 16.2 Spacing scale

- `4, 8, 12, 16, 24, 32`

### 16.3 Radius

- cards: `12px`
- dialogs: `14px`
- inputs/buttons: `10px`

### 16.4 Light theme

- background: `#F3F5F7`
- panel: `#FFFFFF`
- panel-alt: `#E9EEF2`
- text-primary: `#13202B`
- text-secondary: `#4E6375`
- accent: `#0E7490`
- success: `#1F7A4C`
- warning: `#B86A00`
- danger: `#B42318`
- border: `#D7E0E7`

### 16.5 Dark theme

- background: `#0F1720`
- panel: `#16212B`
- panel-alt: `#1D2B36`
- text-primary: `#EAF1F5`
- text-secondary: `#9FB0BD`
- accent: `#4CC9E0`
- success: `#39B56A`
- warning: `#F2A93B`
- danger: `#F97066`
- border: `#29404E`

### 16.6 Component sizing

- left navigation width: `240px`
- top bar height: `56px`
- summary card minimum width: `220px`
- table row height: `40px`
- primary button height: `36px`

### 16.7 Theme behavior

- default follows OS theme
- user can override to light or dark
- preference is stored locally outside the cache workspace

## 17. Windowing, Layout, and Accessibility

### 17.1 Window behavior

- default window size: `1440x960`
- minimum size: `1280x800`

### 17.2 Responsive behavior

- below `1360px`, Overview cards wrap into two columns
- below `1280px`, Explorer detail pane stacks below the list pane
- below minimum size, resizing is blocked rather than allowing broken layouts

### 17.3 Keyboard support

- `Ctrl+O`: open directory
- `Ctrl+R` or `F5`: refresh current page
- `Ctrl+1..4`: switch primary pages
- `Esc`: close active dialog

### 17.4 Accessibility

- visible focus ring on all interactive elements
- semantic headings in all major panels
- ARIA labels for icon-only buttons
- dialogs trap focus until closed
- color is never the sole warning/error signal

## 18. Toast and Feedback System

### 18.1 Placement

- top-right corner of the content area

### 18.2 Max stack

- maximum 3 simultaneous toasts

### 18.3 Durations

- info: `4s`
- success: `4s`
- warning: `6s`
- error: persistent until dismissed

### 18.4 Behavior

- toasts are secondary feedback only
- long-running task details live in the task panel, not inside the toast

## 19. Backend API Surface

### 19.1 Queries

- `OpenWorkspace(rootPath string) (WorkspaceStateVM, *GUIError)`
- `CloseWorkspace() (*GUIError)`
- `GetWorkspaceState() (WorkspaceStateVM, *GUIError)`
- `GetOverview() (OverviewVM, *GUIError)`
- `ListSegments(page int, pageSize int) (PagedSegmentsVM, *GUIError)`
- `GetSegmentDetail(segmentID uint64) (SegmentDetailVM, *GUIError)`
- `GetWALDetail() (WALDetailVM, *GUIError)`
- `ListCursors() ([]CursorSummaryVM, *GUIError)`
- `GetCursorDetail(destination string) (CursorDetailVM, *GUIError)`
- `GetCheckpointDetail() (CheckpointDetailVM, *GUIError)`
- `GetConfig() (ConfigVM, *GUIError)`

### 19.2 Commands

- `Refresh() (*GUIError)`
- `AcquireMaintenanceLock() (WorkspaceStateVM, *GUIError)`
- `ReleaseMaintenanceLock() (WorkspaceStateVM, *GUIError)`
- `RunVerify() (TaskVM, *GUIError)`
- `RunCloseCheck() (TaskVM, *GUIError)`
- `RunRepairTail(segmentID uint64) (TaskVM, *GUIError)`
- `RunShutdown() (TaskVM, *GUIError)`
- `CancelTask(taskID string) (*GUIError)`

## 20. Coexistence With CLI Tools

GUI and CLI tools must share the same lock protocol.

Rules:

- GUI observer mode may coexist with CLI observer operations
- GUI maintenance mode blocks CLI maintenance operations on the same workspace
- CLI maintenance mode blocks GUI maintenance operations on the same workspace
- if the storage engine writer lock is active, both GUI and CLI maintenance actions are disabled
- `cachectl`, GUI, and the storage engine must use the same shared lock package
- `cachectl` and GUI must eventually use the same structured service layer for shared maintenance behavior

## 21. Testing Strategy

### 21.1 Go backend tests

- use existing `go test` patterns
- validate session state transitions
- validate lock-mode gating
- validate crash-release semantics through subprocess tests
- validate local-filesystem-only gating
- validate timeout/cancel behavior
- validate error mapping

### 21.2 Frontend tests

Use:

- `Vitest`
- `@testing-library/react`

Validate:

- landing page
- loading states
- degraded/invalid workspace states
- page navigation
- dialogs
- task panel updates

### 21.3 Desktop integration tests

Use Wails integration harness plus temp cache roots to validate:

- open valid workspace
- open invalid workspace
- observer mode
- maintenance lock acquisition
- observer-to-maintenance reacquire flow
- verify task lifecycle
- operation disablement while writer lock is active

### 21.4 Smoke tests

Run on Windows and Linux CI builds:

- app starts
- app shows landing page
- app opens a valid workspace
- app renders Overview and Explorer
- app runs `verify`

## 22. Build and Packaging

Phase-1 deliverables:

- Windows x64 portable zip containing the desktop executable
- Linux x64 tar.gz containing the desktop executable

Phase 1 does not require installer generation.

CI requirements:

- GitHub Actions matrix:
  - `windows-latest`
  - `ubuntu-latest`
- run Go tests
- run frontend tests
- build Wails desktop artifacts
- publish artifacts for manual QA

## 23. Acceptance Criteria

Phase 1 is complete only if all of the following are true:

- the application builds on Windows and Linux
- the app starts into a non-empty landing state
- the app can open exactly one valid cache root
- the app can distinguish `HealthyObserver`, `HealthyMaintenance`, `DegradedReadOnly`, and `InvalidWorkspace`
- the shared `internal/lock` package is implemented and used by engine, CLI, and desktop backend
- Overview renders every required field listed in section 11.1
- Explorer renders every required tab and required field listed in section 11.2
- Config renders every required field and legal range listed in sections 11.3 and 15
- mutating actions are impossible without `MaintenanceExclusive`
- writer-lock conflicts are surfaced explicitly
- long-running tasks use the task protocol in section 14
- loading skeletons, progress text, and timeout handling follow section 9
- theme switching works for light and dark themes
- keyboard shortcuts in section 17.3 work

## 24. Deferred Upgrades

The following items are explicitly deferred, not rejected:

- persisted config editing once the engine has a real config store
- remote service/API mode
- multiple workspaces per app session
- multi-tab layout
- richer time-series charts
- real-time event/log console
- macOS packaging
- Web management client using the same backend contracts

## 25. Implementation Notes

This design deliberately narrows phase 1 in three places:

- configuration is inspection-only because the engine has no persisted config API yet
- direct local-directory mode is allowed, but mutable operations are gated behind an explicit lock protocol
- large-cache support is achieved with staged loading, pagination, virtualization, and cancellation instead of trying to render everything eagerly

Those constraints keep the GUI implementable without weakening the storage engine’s safety guarantees.
