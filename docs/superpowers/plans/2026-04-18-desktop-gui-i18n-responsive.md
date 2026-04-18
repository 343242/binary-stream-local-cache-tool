# Desktop GUI i18n And Responsive Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Chinese-first bilingual GUI, lock desktop layout structure on small screens, gate navigation before a workspace is opened, and publish a Chinese GUI user manual.

**Architecture:** Keep the existing React + Zustand + CSS-module shell. Add a lightweight local i18n dictionary, route shell copy through it, move empty-workspace navigation gating into the store/sidebar shell boundary, and replace narrow-screen reflow rules with scroll-preserving minimum-width constraints.

**Tech Stack:** React, Zustand, TypeScript, CSS Modules, Vitest, Testing Library, Wails frontend bindings

---

### Task 1: Lock Behavior With Frontend Tests

**Files:**
- Modify: `desktop/frontend/src/components/__tests__/app.test.tsx`
- Modify: `desktop/frontend/src/__tests__/app-shell.test.tsx`

- [ ] **Step 1: Write failing tests for default Chinese copy, gated navigation, and narrow-layout policy**
- [ ] **Step 2: Run `cd desktop/frontend && npm test -- --runInBand app app-shell` and confirm the new assertions fail for the expected reasons**
- [ ] **Step 3: Keep only the failing expectations that map to real requirements**

### Task 2: Add Lightweight i18n Infrastructure

**Files:**
- Create: `desktop/frontend/src/i18n/messages.ts`
- Create: `desktop/frontend/src/i18n/index.ts`
- Modify: `desktop/frontend/src/state/app-store.ts`

- [ ] **Step 1: Add locale types, default locale, and message dictionaries for `zh-CN` / `en-US`**
- [ ] **Step 2: Add minimal store support for reading and changing locale**
- [ ] **Step 3: Run the focused tests and confirm they still fail only on untranslated UI surfaces**

### Task 3: Route Shell And Page Copy Through i18n

**Files:**
- Modify: `desktop/frontend/src/App.tsx`
- Modify: `desktop/frontend/src/components/Sidebar.tsx`
- Modify: `desktop/frontend/src/components/TopBar.tsx`
- Modify: `desktop/frontend/src/components/EmptyState.tsx`
- Modify: `desktop/frontend/src/components/DetailPane.tsx`
- Modify: `desktop/frontend/src/components/TaskPanel.tsx`
- Modify: `desktop/frontend/src/components/ConfirmDialog.tsx`
- Modify: `desktop/frontend/src/components/ToastRegion.tsx`
- Modify: `desktop/frontend/src/pages/LandingPage.tsx`
- Modify: `desktop/frontend/src/pages/OverviewPage.tsx`
- Modify: `desktop/frontend/src/pages/ExplorerPage.tsx`
- Modify: `desktop/frontend/src/pages/ConfigPage.tsx`
- Modify: `desktop/frontend/src/pages/OperationsPage.tsx`

- [ ] **Step 1: Replace hard-coded user-facing copy with translated strings without changing component responsibilities**
- [ ] **Step 2: Keep fallback/demo data labels consistent with the selected locale**
- [ ] **Step 3: Run focused tests and confirm the i18n assertions now pass**

### Task 4: Gate Navigation Before Workspace Open

**Files:**
- Modify: `desktop/frontend/src/state/app-store.ts`
- Modify: `desktop/frontend/src/components/Sidebar.tsx`
- Modify: `desktop/frontend/src/components/__tests__/app.test.tsx`

- [ ] **Step 1: Add a store action that guards page navigation when no workspace is open**
- [ ] **Step 2: Show disabled navigation styling plus friendly toast feedback on blocked clicks**
- [ ] **Step 3: Run focused tests and confirm blocked navigation stays on Landing and emits feedback**

### Task 5: Replace Narrow Reflow With Scroll-Preserving Desktop Layout

**Files:**
- Modify: `desktop/frontend/src/styles/shell.module.css`
- Modify: `desktop/frontend/src/__tests__/app-shell.test.tsx`

- [ ] **Step 1: Remove media-query-driven single-column structural changes**
- [ ] **Step 2: Add minimum-width and overflow rules so the shell keeps the desktop composition at small widths**
- [ ] **Step 3: Run focused tests and confirm shell rendering still holds at narrow widths**

### Task 6: Publish Chinese GUI User Manual

**Files:**
- Create: `docs/desktop-gui-user-manual.zh-CN.md`

- [ ] **Step 1: Write a Chinese manual based on the actual GUI flow and current runtime limits**
- [ ] **Step 2: Include startup, workspace opening, page-by-page usage, state explanations, and known limitations**

### Task 7: Full Verification And Integration

**Files:**
- Verify only

- [ ] **Step 1: Run `cd desktop/frontend && npm test -- --runInBand`**
- [ ] **Step 2: Run `cd desktop/frontend && npm run build`**
- [ ] **Step 3: Review the diff for accidental copy regressions or layout churn before commit**
