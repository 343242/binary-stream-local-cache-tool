# Windows Native Build Verification

This guide is for validating the native Windows desktop build on the branch that contains the Wails desktop app entrypoints such as `main.go`, `wails.json`, and `desktop/frontend/`.

## Scope

Use this checklist when you want proof that the Windows desktop target builds and runs natively, not just that Go tests or the web frontend pass.

Important differences from Linux:

- Do not use the Linux-only `-tags webkit2_41` flag on Windows.
- Windows uses WebView2 rather than GTK/WebKit system packages.

## Host Requirements

Install these prerequisites on the Windows machine first:

- Go matching the repo requirement in `go.mod`
- Node.js LTS and `npm`
- Microsoft Edge WebView2 Runtime
- Visual Studio Build Tools 2022 with the Desktop C++ workload
- Wails CLI v2

Install or update the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Make sure `%USERPROFILE%\\go\\bin` is on `PATH`, or call Wails with its full path.

## Pre-Build Verification

From the repository root on Windows:

```bash
wails doctor
go test ./... -count=1
```

If the desktop frontend exists in the branch, verify it before the native build:

```bash
cd desktop/frontend
npm ci
npm test -- --runInBand
npm run build
cd ../..
```

## Native Build Commands

Build the standard 64-bit Windows target:

```bash
wails build -platform windows/amd64
```

If you need ARM64 validation as well:

```bash
wails build -platform windows/arm64
```

Expected output location:

- `build/bin/<application-name>.exe`

The exact executable name comes from the Wails app configuration for that branch.

## Smoke Test Checklist

After the build succeeds, run the generated `.exe` directly on Windows and verify:

1. The app launches without a missing-runtime or DLL error.
2. The main window renders and the frontend loads without a blank screen.
3. Opening a workspace through the native directory chooser succeeds.
4. Recent workspaces persist across app restart.
5. Read-only overview and explorer data hydrate from the real backend, not fallback demo data.
6. Operations trigger backend work and task status changes appear in the UI.
7. Closing and reopening the app does not lose the expected workspace state.

If the branch includes task-event wiring, explicitly test:

1. Start an operation.
2. Confirm progress or completion updates appear without manual refresh.
3. Confirm success and failure toasts are both surfaced correctly.

## Evidence To Keep

For a real verification record, capture:

- `wails doctor` output
- `go test ./... -count=1` output
- frontend test/build output if applicable
- the exact `wails build -platform windows/...` command used
- the produced `.exe` path
- the commit SHA used for the build

## Common Failure Notes

- `wails doctor` fails on Windows: usually PATH, WebView2, or Build Tools installation is incomplete.
- Native build fails but Go tests pass: this usually points to missing Windows toolchain components rather than application logic.
- App builds but opens a blank window: verify the frontend build step and confirm generated assets are present.
- App starts but native dialogs or runtime events do not work: verify the build is running the Wails-backed desktop branch, not a browser-only fallback path.
