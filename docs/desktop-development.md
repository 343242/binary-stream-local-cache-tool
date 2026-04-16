# Desktop Development

This branch contains the phase-1 desktop GUI work for the binary stream local cache tool.

## Scope

Phase 1 is a local, observer-first desktop console built with:

- `Wails v2`
- `React 18`
- `TypeScript`
- `Vite`
- `Zustand`

Current implementation status in this branch:

- shared workspace locking is integrated into engine and CLI
- desktop read-only service/viewmodel layer exists
- desktop backend session/task scaffolding exists
- Wails shell and frontend scaffold exist
- overview, explorer, config, and operations shell pages exist
- frontend bindings subscribe to runtime events and prefer the real backend over demo fallback data
- recent-workspace persistence and native directory chooser flows are wired through the desktop backend

## Runtime Checklist

Before attempting a native desktop build, verify:

### Windows

- WebView2 runtime is installed

### Linux

- `gcc` is installed
- `libgtk3` development/runtime packages are available
- `libwebkit` / WebKitGTK packages are available
- on Ubuntu 24.04, use `libwebkit2gtk-4.1-dev` and build with `-tags webkit2_41`

## Frontend Commands

From repo root:

```bash
rtk proxy sh -lc 'cd desktop/frontend && npm install'
rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand'
rtk proxy sh -lc 'cd desktop/frontend && npm run build'
```

Notes:

- the frontend test script intentionally accepts the plan's `--runInBand` suffix even though Vitest does not natively use that flag
- generated assets under `desktop/frontend/dist/` and dependencies under `desktop/frontend/node_modules/` must stay out of git

## Go Commands

Recommended desktop-related verification:

```bash
rtk go test ./tests/unit/... -run DesktopService -count=1 -v
rtk go test ./tests/unit/... -run DesktopBackend -count=1 -v
rtk go test ./tests/integration/... -run DesktopService -count=1 -v
```

Full verification target for the branch:

```bash
rtk go test ./tests/unit/... -count=1 -v
rtk go test ./tests/integration/... -count=1 -v
rtk go test ./... -count=1
rtk go test ./... -race -count=1
rtk proxy sh -lc 'cd desktop/frontend && npm test -- --runInBand'
rtk proxy sh -lc 'cd desktop/frontend && npm run build'
```

## Wails Build

After Go and frontend verification pass:

```bash
rtk proxy sh -lc 'wails build -tags webkit2_41 -platform linux/amd64'
```

Linux startup smoke from repo root:

```bash
rtk proxy sh -lc 'timeout 10s dbus-run-session -- xvfb-run -a ./build/bin/binary-stream-cache-tool'
```

Expected result:

- the process stays alive until `timeout` stops it
- GTK/WebKit warnings may still appear in headless environments, but startup must not panic
- run the command from the repository root so the desktop app can resolve `desktop/frontend/dist/`

If the command fails, check:

- runtime dependencies from the checklist above
- Wails CLI availability on the machine
- frontend dependencies under `desktop/frontend`
- that `desktop/frontend/dist/` exists before launching the built binary

## Lock-Aware Operations

Desktop maintenance behavior follows the shared lock protocol:

- observer flows use shared/read-only access
- mutating actions require `MaintenanceExclusive`
- active writer ownership blocks maintenance actions
- lock restoration failures should degrade to safe/read-only behavior rather than optimistic mutation

There is no phase-1 force-unlock path.

## Remaining Gaps

Known follow-up work after the current Linux validation:

- Windows native Wails build and startup still need machine-side verification
- the desktop binary currently resolves packaged frontend assets from `desktop/frontend/dist/` at runtime, so launches are expected from the repo root instead of an arbitrary unpacked location
- branch-level README integration still depends on whether the target branch tracks `README.md` and `README_zh.md`
