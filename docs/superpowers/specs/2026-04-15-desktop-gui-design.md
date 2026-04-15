# Desktop GUI Design

## 1. Goal

Build a Windows/Linux desktop management application for the binary stream local cache tool.

The first release is an embedded-device-oriented local management GUI:

- Runs as an independent windowed desktop program
- Opens a local cache directory directly
- Prioritizes status overview, data browsing, configuration management, and controlled operations
- Defaults to read-only behavior
- Requires explicit confirmation dialogs before any state-changing action

This release does not introduce remote multi-node management.

## 2. Scope

### In Scope

- Windows desktop support
- Linux desktop support
- Wails-based desktop application
- Single opened cache root per application session
- Overview dashboard
- Resource explorer for segment / WAL / cursor / checkpoint
- Configuration view with controlled editing
- Operations view for selected maintenance actions
- Structured error display and health warnings

### Out of Scope

- macOS support
- Multi-directory workspace
- Remote service connection mode
- Web deployment
- Multi-node fleet management
- Real-time terminal console emulation
- Full TSDB-like record query capabilities

## 3. Product Direction

The GUI is a desktop-style operations console, not a generic admin website.

The visual direction is:

- Left navigation
- Top context/status bar
- Right-side main work area
- Dense operational cards and detail panes
- Confirmation-first interaction model for destructive or state-changing actions

The preferred interaction model is closer to a local operations tool than a consumer application.

## 4. Recommended Technical Approach

### Option A: Wails desktop application over the current Go codebase

Recommended.

Why:

- Reuses the existing Go cache engine and operational logic
- Produces native-feeling Windows/Linux desktop applications
- Supports richer UI than pure Go widget stacks
- Keeps a clean path toward a future local-service/API mode

### Option B: Fyne pure-Go GUI

Rejected for phase 1.

Why:

- Simpler stack, but weaker visual flexibility
- Harder to produce the intended dense operations-console layout

### Option C: Desktop GUI plus separate local management service

Rejected for phase 1.

Why:

- Solves a future problem too early
- Expands scope by introducing service lifecycle and API compatibility concerns now

## 5. High-Level Architecture

The desktop application is split into five modules.

### 5.1 `desktop/app`

Responsibilities:

- Wails application bootstrap
- Main window lifecycle
- Native dialogs such as open-directory and confirmations
- Application startup/shutdown handling

### 5.2 `desktop/backend`

Responsibilities:

- Stable API surface exposed to the frontend
- Query methods for overview, explorer, config, and operation results
- Command methods for controlled writes and maintenance actions

This layer must not expose CLI-formatted text as its primary data contract.

### 5.3 `desktop/session`

Responsibilities:

- Track the single active cache root opened by the user
- Manage session lifecycle
- Hold operation state, current path, refresh state, and read-only/degraded mode

First release constraint:

- One session
- One opened cache directory

### 5.4 `desktop/viewmodel`

Responsibilities:

- Translate engine and ops results into frontend-oriented structured data
- Keep UI independent from internal file layout and parsing rules

Examples:

- `OverviewVM`
- `StorageCardVM`
- `SegmentListItemVM`
- `SegmentDetailVM`
- `CursorDetailVM`
- `ConfigVM`
- `OperationTaskVM`

### 5.5 `desktop/frontend`

Responsibilities:

- Render the application shell and pages
- Manage navigation, dialogs, tables, detail panes, status cards, and forms
- Consume only structured backend contracts

The frontend must not parse raw segment/WAL files.

## 6. Data Access Strategy

Phase 1 uses direct local directory access.

Flow:

1. User launches desktop app
2. User opens a cache root directory
3. Backend validates the directory
4. Session enters one of three modes:
   - healthy read/write-capable mode
   - degraded read-only mode
   - invalid/unopenable mode
5. Frontend renders the available views for that state

Future upgrade path:

- Replace or supplement direct filesystem access with a local service/API connection mode
- Preserve frontend contracts as much as possible

## 7. Information Architecture

The first release contains four primary pages, ordered by business priority.

### 7.1 Overview

Purpose:

- Give operators an immediate picture of cache health and backlog status

Primary content:

- Opened root directory
- Health state
- Total disk usage
- Segment count
- Active segment size
- WAL size
- Backlog estimated record count
- Backlog estimated bytes
- Latest replay progress
- Latest ack progress
- Checkpoint summary
- Recent warnings and actionable issues

Primary interaction:

- Refresh
- Jump to active segment
- Jump to cursor detail
- Jump to config page

### 7.2 Explorer

Purpose:

- Browse the storage artifacts and inspect state without editing by default

Primary content:

- Segment list
- WAL summary
- Cursor list/detail
- Checkpoint detail
- Selected object properties
- Structured preview of selected metadata
- Optional raw/hex preview for advanced inspection

Rules:

- Structured view is the default
- Raw view is secondary
- No arbitrary record query engine is introduced

### 7.3 Config

Purpose:

- View and selectively edit runtime-related configuration

Primary content:

- Segment settings
- Block settings
- Checkpoint settings
- Segment fsync settings
- Retention settings
- Any other supported persisted configuration surface

Behavior:

- Every field shows validation hints
- Each configurable item clearly indicates whether it is:
  - effective after reopen
  - effective immediately
- Save requires explicit confirmation
- Invalid edits are blocked before write

### 7.4 Operations

Purpose:

- Provide guarded maintenance actions

Initial operation set:

- `verify`
- `repair-tail`
- `close-check`
- `shutdown`

Behavior:

- Every state-changing action requires confirmation
- High-risk actions show impact text before execution
- Long-running actions present pending/running/success/failure state
- Results are shown as structured output, not only raw logs

## 8. UI Layout

The chosen layout direction is the operations-console style.

### 8.1 Global shell

- Left sidebar navigation
- Top status/context bar
- Main content panel on the right
- Toast/event area for lightweight feedback

### 8.2 Top status/context bar

Must show:

- Current opened root path
- Current health state
- Current session mode: read-write, read-only, invalid
- Selected resource context when applicable
- Manual refresh action

### 8.3 Navigation

Primary navigation items:

- Overview
- Explorer
- Config
- Operations

Secondary actions:

- Open directory
- Refresh
- About/version

## 9. Interaction and Safety Model

### 9.1 Default mode

The application defaults to read-only interaction patterns unless the user explicitly enters an operation that changes state.

### 9.2 Confirmation policy

The following actions require explicit confirmation:

- Saving configuration changes
- `repair-tail`
- `shutdown`
- Any future cleanup or delete-like action

Confirmation dialogs must include:

- Action name
- Affected path/object
- Risk summary
- Confirmation button with explicit wording

### 9.3 Degraded mode behavior

If the cache root is readable but not safe to modify, the session enters degraded read-only mode.

In degraded mode:

- Overview and Explorer stay available
- Config save is disabled
- `verify` and `close-check` remain available
- `repair-tail` and `shutdown` are disabled
- The UI explains why the session is degraded

## 10. Error Handling

Errors must be surfaced as operator-meaningful states, not only raw backend strings.

The backend must classify at least these cases:

- Path does not exist
- Path is not a valid cache root
- Permission denied
- Read-only filesystem
- Corruption detected
- Partial corruption but readable prefix available
- Unsupported or invalid config values
- Operation execution failure

UI requirements:

- Overview exposes health warnings prominently
- Explorer marks corrupted objects without hiding readable data
- Operations report clear success/failure results with next-step hints

## 11. Backend Contracts

The GUI backend must expose structured APIs rather than CLI text blobs.

Planned query surfaces:

- `OpenWorkspace(rootPath)`
- `GetWorkspaceState()`
- `GetOverview()`
- `ListSegments()`
- `GetSegmentDetail(segmentID)`
- `GetWALDetail()`
- `ListCursors()`
- `GetCursorDetail(destination)`
- `GetCheckpointDetail()`
- `GetConfig()`

Planned command surfaces:

- `SaveConfig(input)`
- `RunVerify()`
- `RunRepairTail(segmentID)`
- `RunCloseCheck()`
- `RunShutdown()`
- `Refresh()`

Return values must be view-model oriented and versioned inside the desktop module if needed.
Return values must be view-model oriented and versioned inside the desktop module from phase 1.

## 12. Relationship With Existing Packages

The desktop GUI must reuse the current Go implementation rather than duplicating core logic.

Principles:

- Reuse `pkg/cache` for engine-facing behavior
- Reuse structured operational logic from `internal/ops` and related internal packages
- Extract shared structured service helpers when both `cachectl` and GUI need the same behavior
- Do not make the GUI dependent on parsing CLI command stdout

This is a hard boundary to avoid logic drift.

## 13. Testing Strategy

Testing is divided into four layers.

### 13.1 Backend unit tests

Validate:

- Input validation
- View-model mapping
- Error classification
- Command gating
- Read-only/degraded-mode behavior

### 13.2 Desktop integration tests

Validate:

- Open valid cache root
- Open invalid cache root
- Overview load
- Explorer load
- Config load and validation
- Confirmed operation execution path

### 13.3 Frontend component tests

Validate:

- Navigation behavior
- Summary cards
- Explorer split view
- Config form validation
- Confirmation dialogs
- Disabled state for degraded/read-only mode

### 13.4 Desktop smoke tests

Validate on Windows and Linux build outputs:

- App starts
- App opens a cache directory
- App renders Overview
- App completes at least one safe read flow
- App completes at least one guarded write flow

## 14. Acceptance Criteria

Phase 1 is complete when all of the following are true:

- The application builds on Windows and Linux
- The application starts as an independent windowed desktop program
- A user can open a valid cache root directory
- Overview, Explorer, Config, and Operations pages all function
- Overview shows core health and storage state
- Explorer shows segment, WAL, cursor, and checkpoint information
- Config supports validated controlled edits
- Operations supports `verify` and `close-check` at minimum, with guarded support for `repair-tail` and `shutdown`
- State-changing operations require confirmation dialogs
- Invalid, damaged, or read-only directories produce explicit UI feedback
- Existing cache engine durability and recovery guarantees are not weakened

## 15. Non-Goals For Phase 1

- Replace `cachectl`
- Replace the benchmark tool
- Implement remote node administration
- Implement multi-tab or multi-workspace management
- Implement advanced charts by default
- Implement real-time terminal/log console simulation

## 16. Upgrade Roadmap

The following capabilities are intentionally excluded from phase 1 but explicitly reserved for later upgrades:

- Multiple cache roots in one application session
- Multi-tab workspace
- Richer time-series and trend visualizations
- Remote node management
- Real-time event/log console
- Local-service/API connection mode
- Optional Web management client derived from the same backend contracts
- macOS desktop packaging

These are deferred, not rejected.

## 17. Implementation Notes

The design favors minimum-risk integration:

- Keep the current Go storage engine authoritative
- Add a desktop-specific adapter/service layer
- Keep desktop contracts structured and stable
- Limit phase-1 scope to one local directory at a time
- Favor readable operator workflows over maximal feature breadth

This keeps the first desktop release achievable without diluting the existing storage-system guarantees.
