# Desktop Frontend Editorial Hybrid Design

## 1. Goal

Redesign the phase-1 desktop frontend so it feels like an operator-grade desktop console instead of a thin demo shell.

The frontend must:

- preserve the observer-first safety model
- read clearly for SRE / operations engineers
- keep routine inspection pages calm and readable
- make destructive flows feel formal, gated, and auditable
- stay compatible with the current `Wails + React + Zustand + CSS modules` stack

This design is for the existing desktop frontend only. It does not replace the backend bindings, task model, or Wails shell.

## 2. Audience And Tone

### 2.1 Primary audience

- SRE / operations engineers

Primary jobs:

- open one cache workspace safely
- understand whether the workspace is healthy, stale, invalid, or degraded
- inspect storage and replay state without noise
- decide whether a maintenance action is safe to run
- review the result and audit posture of an operation

### 2.2 Tone

The chosen tone is:

- `Editorial Hybrid`

Meaning:

- calmer than a classic operations dashboard
- more designed than a generic admin panel
- still unmistakably a technical tool

Locked visual posture:

- use the balanced editorial structure from direction `B`
- use the stronger dangerous-action posture from direction `A`

## 3. Product Decisions Locked In

The following decisions are fixed for this redesign:

- shell layout remains desktop-first
- stable primary navigation remains visible during normal desktop widths
- page structure remains `Landing / Overview / Explorer / Config / Operations`
- frontend stack remains `React 18 + TypeScript + Zustand + CSS modules`
- no new frontend dependency is introduced for this redesign
- `Operations` must visually separate read-only actions from destructive actions
- runtime event subscription remains the source of truth for live task progress

## 4. Recommended Layout Model

Use `Split Editorial Console`.

Rationale:

- it preserves the desktop-tool skeleton the codebase already uses
- it gives `Overview`, `Explorer`, and `Config` one shared visual grammar
- it lets `Operations` become more severe without making the rest of the app feel alarmist
- it fits the current `Sidebar + TopBar + page body` structure with low migration risk

Rejected alternatives:

- `Wide Editorial`
  - rejected because it over-emphasizes the landing and overview pages while forcing denser pages back into a different rhythm
- `Stacked Audit Desk`
  - rejected because it weakens routine inspection efficiency and makes the app feel heavier than necessary for daily use

## 5. Visual System

### 5.1 Theme direction

Theme choice is `light`.

Reason:

- this tool is primarily read during active operational work, not night-first media consumption
- tables, audit text, and diff-like details read better on warm light surfaces for this audience

### 5.2 Palette

Palette direction:

- base surface: warm paper neutrals
- primary ink: deep navy / slate
- structural neutrals: tinted stone / blue-gray
- action accent: muted teal-blue
- danger accent: rust / burnt orange

Hard rules:

- no purple-blue AI gradients
- no pure black or pure white
- danger emphasis must rely on full-surface contrast and copy hierarchy, not side-stripe borders
- routine pages must not look like alert pages

### 5.3 Typography

Typography direction:

- display face for major page headers and operational callouts
- restrained sans-serif body face for tables, labels, and metadata

Behavior:

- major headers get more contrast and wider breathing room
- labels use compact uppercase tracking
- table and metric values use tabular numerics where helpful
- long copy stays under readable line length

### 5.4 Layout rules

- avoid wrapping every section in identical cards
- keep a stable navigation rail
- let page headers breathe
- use asymmetric two-column and three-column compositions where they help scanning
- treat `Overview` as the narrative page, not a generic card grid
- treat `Operations` as a formal action review surface, not just a button list

## 6. Page Architecture

### 6.1 Global shell

The app shell has three persistent layers:

- `Sidebar`
  - navigation
  - workspace status summary
  - freshness / stale cue
- `TopBar`
  - workspace path
  - mode / lock / health snapshot
  - refresh action
- `Page body`
  - page-specific editorial layout

### 6.2 Landing page

Intent:

- desktop-tool cover state before any workspace opens

Must show:

- product identity
- one clear primary action: `Open Cache Directory`
- recent workspaces
- supported workspace expectation
- invalid-workspace recovery when needed

The landing page should feel composed and welcoming, not like an error screen.

### 6.3 Overview page

Intent:

- tell the operator what matters right now

Structure:

- editorial headline / context block
- asymmetric metric region
- warnings and audit posture region
- recent segments and cursors region

`Overview` is the highest-level decision page and should be the most visually expressive page in the app.

### 6.4 Explorer page

Intent:

- inspect structures without losing context

Structure:

- inspection lens tabs
- left inspection list
- right detail pane

Rules:

- changing tabs must not cause full-page visual collapse
- selecting a row only updates the detail pane
- the detail pane should feel like an audit notebook, not a raw dump

### 6.5 Config page

Intent:

- audit effective configuration safely

Structure:

- clear read-only statement
- grouped sections by config domain
- each group presented as an audit ledger

The page must look like reference material, not an editable form.

### 6.6 Operations page

Intent:

- enforce a formal, risk-aware action flow

Structure:

- operation selector rail
- selected action detail panel
- preconditions / impact preview
- task timeline
- latest result / audit result

Danger posture requirements:

- destructive actions use denser contrast and stronger hierarchy
- blocked state is explicit before confirm state
- confirmation is treated as a risk gate, not a generic modal
- task progress and result are visually separate concepts

## 7. ASCII Page Blueprints

### 7.1 Overview

```text
+------------------------------------------------------------------------------------------------------------------+
| Binary Stream Cache Tool                                                                  [snapshot] [refresh]   |
| /workspace/cache-root                                               observer mode · shared lock · healthy        |
+---------------------------+--------------------------------------------------------------------------------------+
| Sidebar                    | OVERVIEW                                                                             |
|                            |                                                                                      |
| Overview                   |  "Readable enough for routine checks. Severe enough for maintenance windows."       |
| Explorer                   |                                                                                      |
| Config                     |  +-------------------+  +-------------------+  +-------------------+                  |
| Operations                 |  | segments          |  | replay lag        |  | lock posture       |                  |
|                            |  | 24                |  | 3m                |  | observer          |                  |
| fresh snapshot             |  | 2 flagged         |  | within policy     |  | shared read       |                  |
| last refresh 18s ago       |  +-------------------+  +-------------------+  +-------------------+                  |
|                            |                                                                                      |
|                            |  +----------------------------------------------+  +--------------------------------+ |
|                            |  | recent activity                              |  | warnings / audit posture      | |
|                            |  | verify completed                      ok     |  | tail anomaly needs review    | |
|                            |  | cursor advanced                 seq 18421    |  | checkpoint stale by 2m      | |
|                            |  | lock transition blocked            review    |  | destructive ops gated        | |
|                            |  +----------------------------------------------+  +--------------------------------+ |
+---------------------------+--------------------------------------------------------------------------------------+
```

### 7.2 Operations

```text
+------------------------------------------------------------------------------------------------------------------+
| Operations                                                                                   maintenance required |
+---------------------------------------------+--------------------------------------------------------------------+
| operation list                               | selected operation                                                 |
|                                             |                                                                    |
|  verify                                     |  REPAIR TAIL                                                       |
|  close-check                                |  destructive maintenance action                                    |
| > repair-tail                               |                                                                    |
|  shutdown                                   |  This action mutates segment/WAL tail data and requires            |
|                                             |  explicit lock transition before execution.                        |
|                                             |                                                                    |
|                                             |  preconditions                                                     |
|                                             |  [workspace healthy-maintenance] [exclusive lock] [fresh snapshot] |
|                                             |                                                                    |
|                                             |  impact preview                                                    |
|                                             |  - may truncate corrupted tail region                              |
|                                             |  - emits operation event and audit log                             |
|                                             |  - requires explicit confirm step                                  |
|                                             |                                                                    |
|                                             |                                   [review impact] [run repair]     |
+---------------------------------------------+--------------------------------------------------------------------+
| task timeline: queued -> lock-check -> impact-scan -> confirm-required -> applying -> completed / failed         |
+------------------------------------------------------------------------------------------------------------------+
| latest result / audit trail                                                                                      |
+------------------------------------------------------------------------------------------------------------------+
```

## 8. Frontend Architecture

The redesign keeps the current code-path shape and tightens responsibilities.

```text
+------------------------------------------------------------------------------------------------------------------+
| React App Shell                                                                                                  |
| App.tsx -> Sidebar + TopBar + Page Container                                                                     |
+------------------------------------------------------------------------------------------------------------------+
| Pages                                                                                                            |
| LandingPage | OverviewPage | ExplorerPage | ConfigPage | OperationsPage                                          |
+------------------------------------------------------------------------------------------------------------------+
| Shared UI                                                                                                        |
| Status cards | detail pane | task timeline | confirm dialog | toasts | empty / loading states                 |
+------------------------------------------------------------------------------------------------------------------+
| useAppStore (Zustand)                                                                                            |
| page state | workspace resource state | selection state | operation state | risk gates | task / toast state     |
+------------------------------------------------------------------------------------------------------------------+
| Integrations                                                                                                     |
| bindings.ts -> request/response backend calls                                                                    |
| runtime.ts  -> event subscriptions                                                                               |
+------------------------------------------------------------------------------------------------------------------+
| Wails backend                                                                                                    |
| workspace queries | overview/config/segments/cursors | operation start | task lifecycle events                  |
+------------------------------------------------------------------------------------------------------------------+
```

Responsibility rules:

- `App.tsx` stays thin
- pages own layout composition, not shared data fetching primitives
- store owns UI state orchestration
- bindings own request-response calls only
- runtime owns event subscription only

## 9. State Flow

### 9.1 Workspace lifecycle

```text
NoWorkspace
  -> ChoosingWorkspace
  -> InvalidWorkspace | WorkspaceOpened
  -> HydratingFromBindings
  -> ReadyForInspection | DegradedReadOnly
  -> Refreshing
  -> ReadyForInspection
```

### 9.2 Inspection flow

```text
ReadyForInspection
  -> Overview
  -> Explorer -> tab change -> row selection -> detail pane update
  -> Config
  -> Operations
```

### 9.3 Operations flow

```text
Operations idle
  -> operation selected
  -> blocked notice | confirm gate | task requested
  -> task:started
  -> task:progress
  -> task:finished
  -> latest result + toast + audit summary
```

Locked interaction rules:

- destructive flows are a distinct state branch
- blocked is visible before confirm
- confirm is visible before run
- task timeline persists independently from latest result
- stale/degraded/invalid workspace state must be reflected in shell and operation gating

## 10. Component Refactor Targets

### 10.1 Shell components

- `Sidebar`
  - upgrade from plain nav list to workspace rail
- `TopBar`
  - upgrade from compact info bar to snapshot header

### 10.2 Shared content components

- `StatusCard`
  - support different emphasis tiers and richer secondary copy
- `DetailPane`
  - support audit-note presentation and grouped fields
- `EmptyState`
  - support landing-grade and inline-grade variants
- `LoadingSkeleton`
  - match each page shape instead of generic rows

### 10.3 Operations components

- `TaskPanel`
  - replace static field list with timeline / phase presentation
- `ConfirmDialog`
  - replace generic confirmation with impact review
- `ToastRegion`
  - keep quiet for info, more formal for warning/error

## 11. Implementation Order

Recommended order:

1. establish design tokens and shell layout rules
2. refactor shared shell components
3. rebuild `Landing` and `Overview`
4. rebuild `Explorer` and `Config`
5. rebuild `Operations`
6. update responsive behavior and tests

## 12. Testing And Verification

The redesign must preserve current behavior while upgrading structure and appearance.

Verification requirements:

- frontend tests continue passing
- no broken runtime binding fallback
- no broken operation gating
- no broken task event rendering
- no layout collapse at narrower desktop widths

Manual verification must cover:

- no workspace
- invalid workspace
- healthy observer workspace
- degraded read-only workspace
- blocked dangerous operation
- confirmable dangerous operation
- running task
- finished task with warning/error/success result

## 13. Out Of Scope

- backend API redesign
- new routing library
- charting library introduction
- multi-workspace support
- mobile-first redesign
- remote management features
- configuration editing
