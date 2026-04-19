# Desktop GUI Writer-Control Redesign

## 1. Goal

Refine the desktop GUI so it can manage the local write lifecycle directly inside the existing `Wails` application while staying aligned with the original product goal:

- receive and cache industrial acquisition data locally when network or upstream server is unavailable
- let operators confirm that writes are still happening
- surface risk and failure conditions quickly
- preserve structure-inspection and maintenance workflows for diagnosis and recovery

This redesign corrects two assumptions that were explored and then rejected during design discussion:

- there is **no separate backend service process**
- the app **does not restore the last visited page on startup**; it opens to `Home`

The Wails desktop app remains a single local application with an embedded Go backend.

## 2. Product Decisions Locked In

The following decisions are fixed for this redesign:

- desktop shell technology remains `Wails v2`
- frontend stack remains `React 18 + TypeScript + Zustand + CSS modules`
- the desktop app directly owns the local writer lifecycle through its embedded Go backend
- no standalone daemon, RPC layer, or separately launched local service is introduced
- startup lands on `Home` every time
- the shell keeps stable desktop navigation
- `Home` becomes the control-and-awareness page
- `Overview` becomes the health-and-capacity page
- `Explorer` remains the storage-structure inspection page
- `Config` remains the configuration audit page, but gains a modal entry point for startup configuration
- `Operations` remains the guarded maintenance page

## 3. Why This Redesign Is Needed

The original desktop GUI design was observer-first and structurally sound, but it under-served the original requirement in one specific way:

- the requirement is not only to inspect cached files after the fact
- the operator also needs proof that the system is actively receiving and persisting data during degraded upstream conditions

That means the GUI must expose:

- a clear `start writing / stop writing` lifecycle
- workspace selection and initialization before writing starts
- a lightweight real-time write log
- a direct exception surface for write failures and storage risk

At the same time, the design must **not** drift into a remote control system or a generic service dashboard. It is still a local tool for one machine and one cache workspace at a time.

## 4. Primary User Jobs

The primary user is a local operator or engineer responsible for keeping the cache alive during upstream outages.

Their main jobs are:

- choose or initialize a valid cache workspace
- start local writing with an explicit startup configuration
- verify that records are continuing to persist locally
- notice when writes stall, fail, or become unsafe
- inspect storage structures when recovery or diagnosis is needed
- run guarded maintenance actions only when conditions are safe

## 5. Correct System Boundary

The GUI and backend boundary is:

- `Wails window + React frontend` for shell, layout, interaction, and display
- `embedded Go backend` for workspace lifecycle, engine lifecycle, file inspection, maintenance, and write control

The app does **not**:

- connect to a separately running backend service
- send control signals to another local process and wait for out-of-process acknowledgements
- model availability as "service connected / disconnected"

Instead, the app directly executes the local actions:

- initialize workspace
- open workspace
- start writer
- stop writer
- change workspace
- inspect config/state/files
- run maintenance tasks

### 5.1 Record Ingestion Boundary

The GUI does **not** call `WriteBatch` directly.

`WriteBatch` must be invoked by an in-process writer host inside the Wails application:

- `WriterHost`
  - owns the active `StorageEngine`
  - accepts records from the local acquisition/input adapter path
  - calls `WriteBatch`
  - emits coarse write events and lifecycle events to the desktop frontend

The desktop frontend is responsible for:

- start / stop intent
- workspace choice
- startup configuration
- event display
- alert display

The desktop frontend is not responsible for:

- constructing write batches
- calling `WriteBatch` itself
- pushing one UI event per input record

### 5.2 `WriterHost` Contract

`WriterHost` must be defined as an explicit backend contract, not only as prose.

Go-like shape for planning purposes:

```go
type WriterStatus struct {
    LifecycleState string
    WorkspaceState string
    RootPath       string
    LastError      string
    StartedAtUnixMs int64
    StoppedAtUnixMs int64
}

type WriterHost interface {
    Start(ctx context.Context, root string, cfg core.Config) error
    Stop(ctx context.Context) error
    Status() WriterStatus
}
```

Rules:

- `Start` owns engine open + workspace initialization/open + writer activation
- `Stop` owns graceful writer shutdown
- `Status` is the single backend truth for writer lifecycle posture exposed to the GUI

### 5.3 Acquisition Adapter Boundary

The local acquisition/input path must connect to `WriterHost` through an in-process adapter, not through a daemon protocol.

Planning contract:

- the acquisition adapter submits **batches** of `core.RawRecord`
- `WriterHost` is the only component allowed to transform those batches into `WriteBatch` calls
- transport is in-process only
- first implementation should use a bounded in-memory queue/channel between adapter and `WriterHost`

Backpressure rule:

- the queue is bounded
- when full, the producer blocks until capacity is available or the caller context expires
- the first implementation does not introduce HTTP, RPC, or cross-process transport for ingestion

## 6. Information Architecture

Primary navigation:

- `Home`
- `Overview`
- `Explorer`
- `Config`
- `Operations`

The key rule is:

- `Home` answers "Can I control this? Is writing still happening? Is anything wrong right now?"
- `Overview` answers "Is the cache system healthy overall?"

## 7. Global Shell

The shell keeps the desktop-first structure:

- `Sidebar`
- `TopBar`
- `Page body`

### 7.1 Sidebar

Sidebar responsibilities:

- product identity
- primary navigation
- compact workspace summary
- current writer state badge

### 7.2 TopBar

TopBar responsibilities:

- current workspace path
- app mode summary
- refresh action
- language toggle if retained

TopBar language and copy must describe **application state**, not remote service state.

Correct examples:

- `Writer running`
- `Writer stopped`
- `Workspace open`
- `Snapshot refreshed`

Incorrect examples:

- `Service connected`
- `Signal sent`
- `Waiting for backend confirmation`

## 8. Startup Behavior

Startup behavior is fixed:

- the app always opens to `Home`
- it may restore the last used workspace path as a convenience reference, but it does not restore the last page
- if no workspace is open, `Home` shows the empty-start posture

Rationale:

- `Home` is now the operational landing page
- going directly to `Overview`, `Explorer`, or `Operations` before the operator sees current control posture is less safe and less legible

## 9. Home Page

`Home` is the operational landing page.

It is intentionally narrow in responsibility. It does **not** try to summarize the entire system.

### 9.1 Home Must Show

- top control strip
- writer lifecycle actions
- workspace/path actions
- real-time write log
- exception and alert surface

### 9.2 Home Must Not Show

- the full health summary
- segment/cursor/checkpoint summaries
- recent maintenance results as a dedicated panel
- dense structural audit content

### 9.3 Home Layout

Top row:

- current workspace path
- writer state
- `Start Writing` or `Stop Writing`
- `Change Storage Location`
- `Writer Config`

Bottom-left:

- `Real-Time Write Log`

Bottom-right:

- `Exceptions & Alerts`

### 9.4 Home Log Content

The real-time feed is a lightweight write-event stream, not a business-data viewer.

Examples:

- `Workspace opened at /data/cache-root-01`
- `1 record stored to segments/000021.seg`
- `Batch 18421 persisted successfully`
- `Checkpoint advanced to writeSeq 99887766`
- `Writer stopped cleanly`
- `Workspace initialization failed`

The feed does **not** attempt to render:

- gateway ID
- process point ID
- sensor value
- decoded payload fields

That keeps the page aligned with current engine capabilities and avoids turning `Home` into a business-record console.

The feed is **batch-oriented and lifecycle-oriented**, not record-oriented.

Allowed event families:

- workspace opened / initialized / switched
- writer started / stopping / stopped
- batch persisted
- checkpoint advanced
- warning raised
- write failure raised

Explicitly disallowed:

- one frontend event per input record
- raw payload echoing
- sensor-field decoding in the `Home` feed

### 9.5 Home Alert Content

Alerts and exceptions include:

- disk free space warning
- write stall detected
- write failure
- checkpoint/fsync failure
- invalid workspace
- degraded read-only state
- workspace initialization failure
- workspace switch failure

The right-side panel is the operator's "look here now" zone.

## 10. Overview Page

`Overview` becomes the full health and capacity page.

It answers:

- is the cache healthy?
- how much capacity remains?
- is backlog growing?
- how long can the system retain data?
- is the write path behaving within policy?

### 10.1 Overview Must Show

- workspace posture
- lock posture
- total storage usage
- segment count
- backlog estimate
- retention horizon / sustainable days estimate
- checkpoint summary
- fsync summary
- latest successful write time
- warning synthesis

`Overview` is now where the heavier status story lives.

### 10.2 Retention Semantics

The `30 days` retention target remains a **soft lower-bound policy**, not a hard cap that may delete unacknowledged data.

Inherited storage rule:

- configured retention defines the minimum cleanup age target
- unacknowledged data may force data to remain longer than the configured period
- the GUI must present this as "retention target" or "retention horizon", not as a guaranteed hard deletion cutoff

## 11. Explorer Page

`Explorer` remains the structural inspection page.

Tabs remain:

- `Segments`
- `WAL`
- `Cursors`
- `Checkpoint`

Layout remains:

- left list / table pane
- right detail pane

This page is intentionally separate from `Home` because the operator should not need structural detail to determine whether writing is currently active or whether immediate exceptions exist.

## 12. Config Page

`Config` remains the reference and audit page for the effective configuration.

It still shows:

- grouped configuration sections
- effective value
- startup default
- allowed range
- notes / audit meaning

### 12.1 New Entry Point

`Config` gains an entry point for `Writer Config`.

That entry point also appears on `Home`.

## 13. Writer Config Modal

The writer configuration interaction moves into a modal dialog.

This is a deliberate redesign choice:

- `Home` stays clean and operational
- startup configuration remains accessible
- future config expansion does not force a new page layout

### 13.1 Modal Structure

The modal is structured as a **full writer configuration** surface, but only a small subset of fields is editable in the current phase.

Editable now:

- storage location
- retention days
- segment target size
- fsync / checkpoint policy controls

Displayed but disabled:

- all other writer configuration fields in the full grouped config model

Behavior:

- disabled fields are shown using normal disabled controls
- no special "coming later" or "future release" copy is shown

### 13.2 Modal Purpose

The modal is a startup-configuration surface for the embedded writer lifecycle.

It is **not**:

- a remote service config editor
- a live mutable runtime tuning panel for an external process

### 13.3 Configuration Apply Timing

Writer configuration is applied only when starting a new writer instance.

Rules:

- editing values in the modal does not mutate a running engine instance
- confirming the modal updates the pending startup configuration
- the next successful `Start Writing` action uses that configuration
- a running writer continues with the configuration it was opened with
- changing configuration while writing is active requires stop -> reconfigure -> start

The modal must clearly distinguish:

- effective config of the currently running writer
- pending startup config for the next writer start

## 14. Operations Page

`Operations` remains the guarded maintenance surface.

It still owns:

- `verify`
- `close-check`
- `repair-tail`
- `shutdown`

Rules remain:

- destructive actions stay formally gated
- blocked state is visible before confirmation
- confirmation is explicit
- long-running progress stays in the task panel

`Operations` does not become the place to start normal writing.

## 15. Core Workflows

### 15.1 Start Writing

The operator flow is:

1. open `Home`
2. click `Start Writing`
3. choose one of:
   - open existing workspace
   - initialize new workspace in selected directory
4. review `Writer Config` modal
5. confirm
6. embedded Go backend releases observer posture if held
7. embedded Go backend opens/initializes the workspace and starts the local writer lifecycle through `WriterHost`
8. `WriterHost` opens the engine and becomes the only in-process component allowed to call `WriteBatch`
9. `Home` log and status surfaces update

### 15.2 Stop Writing

The operator flow is:

1. click `Stop Writing`
2. app triggers local stop/shutdown through the embedded backend
3. `WriterHost` stops accepting new records
4. state changes to `stopping`
5. on success, the engine closes and the app may reacquire observer posture for inspection
6. state becomes `stopped`
7. `Home` log records the outcome

Stop timeout:

- default stop timeout is `30s`
- timeout expiration must surface a stop failure result and alert
- timeout expiration must not silently report a successful stop

### 15.3 Change Storage Location

Rules:

- changing location is not allowed while the writer is active
- the operator must stop writing first
- once stopped, the operator can choose a different path
- after path selection, the app can either:
  - open an existing workspace
  - initialize a new workspace

### 15.4 Open Existing Workspace vs Initialize New Workspace

Both paths are supported.

Default recommendation:

- initializing a new directory is the recommended primary path when starting fresh
- opening an existing workspace remains supported for inspection or continuation

### 15.5 Lock Transition Protocol

The GUI redesign must respect the existing shared lock model.

Required protocol:

1. inspection flow opens a workspace in observer posture
2. before starting the writer, the app releases the observer lock
3. `WriterHost` opens the engine under `WriterExclusive`
4. while the writer is active, the GUI reads state through in-process APIs and event streams instead of taking a second observer lock on the same workspace
5. after stop, the app may reacquire observer posture for inspection pages

The spec does not allow simultaneous in-process observer and writer locks on the same workspace without an explicit lock model change.

### 15.6 Rapid Start/Stop Protection

The GUI must prevent rapid conflicting lifecycle actions.

Rules:

- `Start Writing` is disabled while state is `starting` or `stopping`
- `Stop Writing` is disabled while state is `starting` or `stopping`
- repeated clicks must not enqueue overlapping start/stop operations
- lifecycle controls re-enable only after a confirmed terminal transition

## 16. State Model

The UI must distinguish two related but different state families.

### 16.1 Writer Lifecycle State

- `not-started`
- `starting`
- `running`
- `stopping`
- `stopped`
- `start-failed`

`stop-failed` is not a stable lifecycle mode.

If stop fails:

- the failure is surfaced as an operation result and alert
- the writer remains in its last confirmed lifecycle state until a later successful transition proves otherwise

### 16.2 Workspace State

- `NoWorkspace`
- `Opening`
- `HealthyObserver`
- `HealthyWriter`
- `HealthyMaintenance`
- `DegradedReadOnly`
- `InvalidWorkspace`

Reason:

- an operator needs to know whether the writer lifecycle succeeded
- independently, they also need to know whether the workspace itself is structurally valid and safe

### 16.3 Legal State Combinations

Required legal combinations:

- `(not-started | stopped | start-failed, NoWorkspace)`
- `(not-started | stopped | start-failed, HealthyObserver)`
- `(starting, Opening)`
- `(running, HealthyWriter)`
- `(stopping, HealthyWriter)`
- `(not-started | stopped | start-failed, DegradedReadOnly)`
- `(not-started | stopped | start-failed, InvalidWorkspace)`

Explicitly illegal as stable states:

- `(running, DegradedReadOnly)`
- `(starting, InvalidWorkspace)`
- `(running, HealthyObserver)`

Runtime fault rule:

- if a fatal write-path fault occurs while running, the writer must leave `running`
- the UI may show an alert immediately
- after the transition settles, the workspace may land in `DegradedReadOnly` if inspection remains possible

## 17. Page Gating

Navigation rules:

- `Home` is always available
- `Overview`, `Explorer`, `Config`, and `Operations` require a valid workspace
- `Operations` still requires its existing safety gates for destructive work

This preserves clarity:

- `Home` is the safe universal landing page
- deeper pages remain contextual

## 18. Required Backend Contract Evolution

This redesign does not require a separate service, but it **does** require backend contract growth inside the Wails app.

New or expanded backend responsibilities include:

- initialize workspace from GUI
- start writer from GUI
- stop writer from GUI
- host the in-process `WriterHost`
- connect a local acquisition/input adapter path to `WriterHost`
- expose writer lifecycle state
- expose a lightweight write-event feed
- apply startup configuration from the modal

### 18.1 Workspace Initialization

`InitializeWorkspace(root)` must become a first-class backend capability.

It must create the required workspace structure for a valid empty cache root:

- `meta/`
- `meta/replay/`
- `segments/`
- `wal/`
- `tools/`
- `tools/reports/`

After successful initialization, opening the workspace must return a valid inspection posture instead of `InvalidWorkspace`.

Initialization error semantics:

- non-existent target path with writable parent: create the path and initialize
- existing empty directory: initialize in place
- existing valid workspace: return a typed "already initialized" result without destructive overwrite
- existing partially initialized directory: fail with a typed partial-layout error
- non-writable target path or parent: fail with permission/read-only error

### 18.2 Event Feed Contract

The backend event feed must be coarse and rate-limited.

Contract rules:

- no per-record UI events
- write success events are emitted per committed batch
- checkpoint/fsync events are emitted only on meaningful transitions
- repeated warning conditions should be deduplicated or throttled
- the frontend maintains a bounded ring buffer for display
- frontend ring buffer size: `200` entries
- max feed emission rate to the frontend: `10` events/second
- identical warning dedupe window: `5s`

The feed is for operator awareness, not audit-grade full replay history.

### 18.3 Shutdown Contract

`RunShutdown` must eventually become a true writer shutdown action, not a handoff placeholder.

When the writer is active, shutdown behavior must:

- stop new intake
- close the active engine through the embedded backend
- update lifecycle state
- emit success or failure back to the UI

### 18.4 Crash-Recovery Signal

The existing engine lifecycle marker must be reflected back into GUI posture.

Required behavior:

- if startup detects an ungraceful previous lifecycle, the GUI must surface a recovery warning
- the app may still open the workspace if recovery succeeds
- the warning must remain visible until the operator dismisses or refreshes away from that startup event

These actions must be implemented in the embedded Go backend, not delegated to a daemon.

## 19. Copy And Tone

All copy must reinforce that this is a local tool.

Preferred tone:

- direct
- operational
- local
- concrete

Examples:

- `Writer running`
- `Change storage location`
- `Open existing workspace`
- `Initialize new workspace`
- `Write stalled for 12s`

Avoid:

- `service`
- `connected`
- `signal`
- `remote`

unless a future design truly adds those concepts.

## 20. Testing Implications

Manual and automated verification must cover:

- startup lands on `Home`
- empty `Home` state before writing starts
- start-writing flow with existing workspace
- start-writing flow with new workspace initialization
- stop-writing flow
- blocked storage-location change while running
- allowed storage-location change after stop
- writer-config modal opens from `Home` and `Config`
- editable vs disabled config fields render correctly
- real-time write log updates
- alert surface updates on write failures and workspace faults
- lock transition from observer posture to writer posture and back
- event-feed throttling / batching behavior under sustained write load
- configuration changes only affecting the next writer start
- shutdown invoking real writer close behavior instead of a placeholder response
- illegal lifecycle action suppression during rapid start/stop clicking
- startup warning behavior after an ungraceful previous writer shutdown

## 21. Out Of Scope

Still out of scope for this redesign:

- separate backend service process
- RPC control plane
- multi-workspace tabs
- business-record decoding in `Home`
- turning `Home` into a sensor-data viewer
- arbitrary payload query UI
- chart-heavy observability dashboard

## 22. Acceptance Criteria

This redesign is complete only if all of the following are true:

- app startup always lands on `Home`
- the app keeps the existing Wails single-process structure
- `Home` shows control actions, real-time write log, and alerts
- `Overview` carries the health and capacity summary
- `Config` exposes the writer-config modal entry point
- the writer-config modal renders the full grouped config structure
- only the approved small parameter subset is editable
- all other config fields are visibly disabled
- the GUI supports both opening an existing workspace and initializing a new workspace
- changing storage location is blocked while writing is active
- page copy consistently reflects local in-app execution rather than external service control
- `WriteBatch` ownership is explicitly limited to the in-process `WriterHost` path
- writer start/stop follows the lock transition protocol without conflicting observer/write locks
- event-feed activity does not invalidate the write benchmark target through unbounded per-record UI emission
- shutdown is implemented as a real embedded-writer lifecycle action
- `WriterHost` is implemented behind an explicit backend interface rather than implicit helper logic
- only legal writer-state/workspace-state combinations are reachable in steady state

## 23. Supersession Notes

This document refines and partially supersedes the following earlier assumptions:

- the observer-first desktop GUI design from `2026-04-15-desktop-gui-design.md`
- the editorial-hybrid frontend structure from `2026-04-17-desktop-frontend-editorial-hybrid-design.md`

Specifically, it changes:

- startup landing behavior
- page responsibility between `Home` and `Overview`
- the existence of an explicit writer-control flow
- the presence of a writer-configuration modal
- the treatment of live awareness on the landing page

### 23.1 Authority Order

For overlapping desktop-GUI topics, authority order is:

1. `2026-04-19-desktop-gui-writer-control-design.md`
2. `2026-04-17-desktop-frontend-editorial-hybrid-design.md`
3. `2026-04-15-desktop-gui-design.md`

The newer document overrides older ones only where their subjects overlap.
