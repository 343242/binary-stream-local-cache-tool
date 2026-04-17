# Desktop Frontend Editorial Hybrid Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the desktop frontend into the approved editorial-hybrid operator console while preserving current bindings, task flow, and safety behavior.

**Architecture:** Keep the current `App -> pages -> shared components -> Zustand store -> bindings/runtime` structure, but upgrade design tokens, shell composition, and page-specific layouts. Make `Overview` and `Explorer` calmer and more legible, while making `Operations` visibly formal and risk-gated.

**Tech Stack:** React 18, TypeScript, Zustand, CSS modules, Vitest, Wails bindings/runtime

---

## File Structure

- Modify: `desktop/frontend/src/styles/tokens.css`
  - replace the current generic palette with editorial-hybrid design tokens
- Modify: `desktop/frontend/src/styles/shell.module.css`
  - rebuild shell, page, and component-level layout rules
- Modify: `desktop/frontend/src/App.tsx`
  - keep shell orchestration thin, add any shell-level variants only if needed
- Modify: `desktop/frontend/src/components/Sidebar.tsx`
  - upgrade navigation into a workspace rail
- Modify: `desktop/frontend/src/components/TopBar.tsx`
  - upgrade compact header into snapshot header
- Modify: `desktop/frontend/src/components/StatusCard.tsx`
  - support emphasis tiers
- Modify: `desktop/frontend/src/components/DetailPane.tsx`
  - support grouped audit detail layout
- Modify: `desktop/frontend/src/components/EmptyState.tsx`
  - support landing and inline variants
- Modify: `desktop/frontend/src/components/LoadingSkeleton.tsx`
  - support page-shaped skeleton variants
- Modify: `desktop/frontend/src/components/TaskPanel.tsx`
  - render task timeline instead of a plain field list
- Modify: `desktop/frontend/src/components/ConfirmDialog.tsx`
  - render impact review content
- Modify: `desktop/frontend/src/components/ToastRegion.tsx`
  - align severity presentation with new tone
- Modify: `desktop/frontend/src/pages/LandingPage.tsx`
- Modify: `desktop/frontend/src/pages/OverviewPage.tsx`
- Modify: `desktop/frontend/src/pages/ExplorerPage.tsx`
- Modify: `desktop/frontend/src/pages/ConfigPage.tsx`
- Modify: `desktop/frontend/src/pages/OperationsPage.tsx`
- Modify: `desktop/frontend/src/state/app-store.ts`
  - add presentation metadata only if required by page rendering
- Test: `desktop/frontend/src/__tests__/app-shell.test.tsx`
- Test: `desktop/frontend/src/components/__tests__/app.test.tsx`
- Test: `desktop/frontend/src/components/__tests__/operations.test.tsx`

### Task 1: Replace tokens and shell-level layout primitives

**Files:**
- Modify: `desktop/frontend/src/styles/tokens.css`
- Modify: `desktop/frontend/src/styles/shell.module.css`
- Test: `desktop/frontend/src/__tests__/app-shell.test.tsx`

- [ ] **Step 1: Write the failing shell expectation test**

```tsx
it("renders the editorial shell landmarks", () => {
  render(<App />);
  expect(screen.getByRole("complementary", { name: /primary workspace/i })).toBeInTheDocument();
  expect(screen.getByRole("banner")).toBeInTheDocument();
  expect(screen.getByRole("main")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app-shell'`
Expected: FAIL because the current shell does not expose the new landmark names.

- [ ] **Step 3: Write the minimal shell token and layout changes**

```css
:root {
  --bg-canvas: oklch(0.96 0.012 85);
  --bg-panel: oklch(0.985 0.008 85);
  --bg-panel-strong: oklch(0.28 0.04 248);
  --ink-primary: oklch(0.22 0.03 248);
  --ink-secondary: oklch(0.45 0.02 248);
  --accent: oklch(0.5 0.08 220);
  --danger: oklch(0.63 0.14 42);
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;
  --space-12: 48px;
}

.frame {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: 272px minmax(0, 1fr);
  gap: var(--space-6);
  padding: var(--space-6);
}
```

- [ ] **Step 4: Run tests to verify shell still mounts**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app-shell'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
rtk git add desktop/frontend/src/styles/tokens.css desktop/frontend/src/styles/shell.module.css desktop/frontend/src/__tests__/app-shell.test.tsx
rtk git commit -m "Establish editorial shell tokens for the desktop frontend

Constraint: Redesign must stay within CSS modules and existing React shell
Rejected: Introduce a new component library | unnecessary dependency and migration cost
Confidence: high
Scope-risk: moderate
Directive: Keep danger styling localized to risk surfaces rather than tinting the whole product
Tested: npm test -- --runInBand app-shell
Not-tested: Full frontend build"
```

### Task 2: Refactor shell components into workspace rail and snapshot header

**Files:**
- Modify: `desktop/frontend/src/components/Sidebar.tsx`
- Modify: `desktop/frontend/src/components/TopBar.tsx`
- Modify: `desktop/frontend/src/App.tsx`
- Test: `desktop/frontend/src/components/__tests__/app.test.tsx`

- [ ] **Step 1: Write the failing component test**

```tsx
it("shows workspace context in the rail and snapshot header", () => {
  render(<App />);
  expect(screen.getByText(/observer-first desktop console/i)).toBeInTheDocument();
  expect(screen.getByText(/fresh snapshot/i)).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app'`
Expected: FAIL because the current shell does not expose the new status summary copy consistently.

- [ ] **Step 3: Implement the workspace rail and snapshot header**

```tsx
<aside aria-label="Primary workspace" className={styles.sidebar}>
  <div className={styles.brandBlock}>
    <p className={styles.eyebrow}>Binary Stream Cache Tool</p>
    <h1 className={styles.brandName}>Observer-first desktop console</h1>
  </div>
  <section className={styles.workspaceRailSummary}>
    <span className={styles.summaryLabel}>Workspace status</span>
    <strong>{workspace?.stale ? "stale snapshot" : "fresh snapshot"}</strong>
  </section>
</aside>
```

- [ ] **Step 4: Run tests to verify the shell copy and landmarks pass**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app app-shell'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
rtk git add desktop/frontend/src/App.tsx desktop/frontend/src/components/Sidebar.tsx desktop/frontend/src/components/TopBar.tsx desktop/frontend/src/components/__tests__/app.test.tsx
rtk git commit -m "Make workspace context legible in the desktop shell

Constraint: Existing App/store split should stay thin and reviewable
Rejected: Move workspace summary into every page | repeated logic and visual noise
Confidence: high
Scope-risk: narrow
Directive: Sidebar owns ambient workspace summary; TopBar owns the current snapshot context
Tested: npm test -- --runInBand app app-shell
Not-tested: Explorer and operations page snapshots"
```

### Task 3: Rebuild landing and overview into editorial inspection pages

**Files:**
- Modify: `desktop/frontend/src/components/StatusCard.tsx`
- Modify: `desktop/frontend/src/components/EmptyState.tsx`
- Modify: `desktop/frontend/src/components/LoadingSkeleton.tsx`
- Modify: `desktop/frontend/src/pages/LandingPage.tsx`
- Modify: `desktop/frontend/src/pages/OverviewPage.tsx`

- [ ] **Step 1: Write the failing overview expectation**

```tsx
it("renders overview as a narrative page with warnings and activity regions", () => {
  render(<App />);
  expect(screen.getByText(/maintenance windows/i)).toBeInTheDocument();
  expect(screen.getByText(/recent activity/i)).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app'`
Expected: FAIL because the current overview page does not include the editorial headline and audit-region wording.

- [ ] **Step 3: Implement the landing and overview layout**

```tsx
<section className={styles.editorialHero}>
  <p className={styles.eyebrow}>Overview</p>
  <h2 className={styles.heroTitle}>Readable enough for routine checks. Severe enough for maintenance windows.</h2>
  <p className={styles.heroCopy}>Phase-1 desktop console for cache inspection, replay diagnostics, and guarded operations.</p>
</section>
```

- [ ] **Step 4: Run focused tests**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
rtk git add desktop/frontend/src/components/StatusCard.tsx desktop/frontend/src/components/EmptyState.tsx desktop/frontend/src/components/LoadingSkeleton.tsx desktop/frontend/src/pages/LandingPage.tsx desktop/frontend/src/pages/OverviewPage.tsx
rtk git commit -m "Turn landing and overview into editorial inspection pages

Constraint: Overview must remain readable with both demo data and bound data
Rejected: Keep uniform metric cards | weak hierarchy and generic dashboard feel
Confidence: medium
Scope-risk: moderate
Directive: Overview is the narrative page; do not flatten it back into a generic card grid
Tested: npm test -- --runInBand app
Not-tested: Narrow-width desktop layout"
```

### Task 4: Rebuild explorer and config as audit surfaces

**Files:**
- Modify: `desktop/frontend/src/components/DetailPane.tsx`
- Modify: `desktop/frontend/src/pages/ExplorerPage.tsx`
- Modify: `desktop/frontend/src/pages/ConfigPage.tsx`

- [ ] **Step 1: Write the failing explorer/config expectations**

```tsx
it("keeps explorer detail content separate from the row list", () => {
  render(<App />);
  expect(screen.getByText(/detail pane/i)).toBeInTheDocument();
});

it("marks config as read-only audit content", () => {
  render(<App />);
  expect(screen.getByText(/read-only inspection/i)).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests to verify they fail or regress after refactor start**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app'`
Expected: If failing, capture missing structure; if already passing, keep the assertions and continue with layout refactor.

- [ ] **Step 3: Implement the audit-surface structure**

```tsx
<div className={styles.explorerLayout}>
  <section className={styles.auditListPanel}>{/* tab lens + rows */}</section>
  <DetailPane title="Inspection notes">{/* grouped fields */}</DetailPane>
</div>
```

- [ ] **Step 4: Run focused tests**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
rtk git add desktop/frontend/src/components/DetailPane.tsx desktop/frontend/src/pages/ExplorerPage.tsx desktop/frontend/src/pages/ConfigPage.tsx
rtk git commit -m "Make explorer and config read like audit surfaces

Constraint: Explorer tab and row selection logic must keep current behavior
Rejected: Add nested routing for explorer tabs | needless complexity for a single desktop page
Confidence: high
Scope-risk: narrow
Directive: Explorer row selection should only update the detail pane, not recompose the whole page
Tested: npm test -- --runInBand app
Not-tested: Manual explorer interaction polish"
```

### Task 5: Rebuild operations into a formal risk-gated action flow

**Files:**
- Modify: `desktop/frontend/src/components/TaskPanel.tsx`
- Modify: `desktop/frontend/src/components/ConfirmDialog.tsx`
- Modify: `desktop/frontend/src/components/ToastRegion.tsx`
- Modify: `desktop/frontend/src/pages/OperationsPage.tsx`
- Modify: `desktop/frontend/src/state/app-store.ts`
- Test: `desktop/frontend/src/components/__tests__/operations.test.tsx`

- [ ] **Step 1: Write the failing operations tests**

```tsx
it("shows blocked dangerous operations before confirmation", () => {
  render(<OperationsPage {...props} selectedOperation="repair-tail" />);
  expect(screen.getByText(/blocked until/i)).toBeInTheDocument();
});

it("renders a task timeline instead of only flat fields", () => {
  render(<TaskPanel task={task} />);
  expect(screen.getByText(/task timeline/i)).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the operations tests to verify the failure**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand operations'`
Expected: FAIL because the current operations surfaces are flatter and do not include the new risk/timeline semantics.

- [ ] **Step 3: Implement the risk-gated operations layout**

```tsx
<section className={styles.operationHero}>
  <p className={styles.operationEyebrow}>destructive maintenance action</p>
  <h3 className={styles.operationTitle}>{operation.title}</h3>
  <div className={styles.preconditionList}>{/* lock/mode/freshness chips */}</div>
  <div className={styles.impactPreview}>{/* impact lines */}</div>
</section>
```

- [ ] **Step 4: Run focused operations tests**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand operations'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
rtk git add desktop/frontend/src/components/TaskPanel.tsx desktop/frontend/src/components/ConfirmDialog.tsx desktop/frontend/src/components/ToastRegion.tsx desktop/frontend/src/pages/OperationsPage.tsx desktop/frontend/src/state/app-store.ts desktop/frontend/src/components/__tests__/operations.test.tsx
rtk git commit -m "Make dangerous operations feel formal and auditable

Constraint: Current bindings and runtime task events remain the source of truth
Rejected: Add a separate operations store | splits task state from the shell with little value
Confidence: high
Scope-risk: moderate
Directive: Task timeline and latest result are separate surfaces; do not collapse them into one card
Tested: npm test -- --runInBand operations
Not-tested: Real runtime task progression from Wails"
```

### Task 6: Finish responsive desktop behavior and run full verification

**Files:**
- Modify: `desktop/frontend/src/styles/shell.module.css`
- Modify: `desktop/frontend/src/components/Sidebar.tsx`
- Modify: `desktop/frontend/src/components/TopBar.tsx`
- Modify: `desktop/frontend/src/pages/LandingPage.tsx`
- Modify: `desktop/frontend/src/pages/OverviewPage.tsx`
- Modify: `desktop/frontend/src/pages/ExplorerPage.tsx`
- Modify: `desktop/frontend/src/pages/ConfigPage.tsx`
- Modify: `desktop/frontend/src/pages/OperationsPage.tsx`
- Test: `desktop/frontend/src/__tests__/app-shell.test.tsx`
- Test: `desktop/frontend/src/components/__tests__/app.test.tsx`
- Test: `desktop/frontend/src/components/__tests__/operations.test.tsx`

- [ ] **Step 1: Add narrow-desktop layout assertions where missing**

```tsx
it("keeps the shell readable at narrow desktop widths", () => {
  window.innerWidth = 1180;
  render(<App />);
  expect(screen.getByRole("main")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the full frontend test suite**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand'`
Expected: PASS

- [ ] **Step 3: Run the production build**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm run build'`
Expected: PASS

- [ ] **Step 4: Review the visual shell for the required states**

Run: `rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand app app-shell operations'`
Expected: PASS, covering shell, general rendering, and operations behavior.

- [ ] **Step 5: Commit**

```bash
rtk git add desktop/frontend/src/styles/shell.module.css desktop/frontend/src/components/Sidebar.tsx desktop/frontend/src/components/TopBar.tsx desktop/frontend/src/pages desktop/frontend/src/__tests__/app-shell.test.tsx desktop/frontend/src/components/__tests__/app.test.tsx desktop/frontend/src/components/__tests__/operations.test.tsx
rtk git commit -m "Finish responsive editorial polish for the desktop frontend

Constraint: Desktop-first layout still needs to survive narrower window widths
Rejected: Build separate mobile navigation patterns | not in scope for the desktop tool
Confidence: medium
Scope-risk: moderate
Directive: Optimize for desktop resizing, not phone-style adaptation
Tested: npm test -- --runInBand; npm run build
Not-tested: Live Wails window verification on Linux"
```

## Self-Review

Spec coverage:

- visual system -> Task 1
- shell redesign -> Task 2
- landing/overview -> Task 3
- explorer/config -> Task 4
- operations posture and risk flow -> Task 5
- responsive verification -> Task 6

Placeholder scan:

- no `TODO`, `TBD`, or “handle appropriately” placeholders remain

Type consistency:

- plan uses the existing `App`, page, component, store, bindings, and runtime file boundaries already present in the repo

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-17-desktop-frontend-editorial-hybrid.md`.

Two execution options:

1. `Subagent-Driven` (recommended)
   - use `superpowers:subagent-driven-development`
   - dispatch a fresh subagent per task and review between tasks

2. `Inline Execution`
   - use `superpowers:executing-plans`
   - execute tasks in this session with review checkpoints
