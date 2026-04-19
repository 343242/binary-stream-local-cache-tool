package unit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fastReadFile/internal/desktop/service"
	"fastReadFile/pkg/cache"
)

func TestDesktopService(t *testing.T) {
	t.Run("TestOpenWorkspaceReturnsInvalidWorkspaceForEmptyDirectory", runTestOpenWorkspaceReturnsInvalidWorkspaceForEmptyDirectory)
	t.Run("TestGetOverviewUsesWarningsForUnavailableWorkspaceStatusFields", runTestGetOverviewUsesWarningsForUnavailableWorkspaceStatusFields)
	t.Run("TestGetConfigIncludesPhaseOneValidationRanges", runTestGetConfigIncludesPhaseOneValidationRanges)
	t.Run("TestInitializeWorkspaceCreatesRequiredLayout", runTestInitializeWorkspaceCreatesRequiredLayout)
	t.Run("TestInitializeWorkspaceRejectsEmptyRoot", runTestInitializeWorkspaceRejectsEmptyRoot)
	t.Run("TestInitializeWorkspaceRejectsFileRootAsInvalid", runTestInitializeWorkspaceRejectsFileRootAsInvalid)
	t.Run("TestInitializeWorkspaceRejectsPartialLayout", runTestInitializeWorkspaceRejectsPartialLayout)
}

func TestOpenWorkspaceReturnsInvalidWorkspaceForEmptyDirectory(t *testing.T) {
	runTestOpenWorkspaceReturnsInvalidWorkspaceForEmptyDirectory(t)
}

func TestGetOverviewUsesWarningsForUnavailableWorkspaceStatusFields(t *testing.T) {
	runTestGetOverviewUsesWarningsForUnavailableWorkspaceStatusFields(t)
}

func TestGetConfigIncludesPhaseOneValidationRanges(t *testing.T) {
	runTestGetConfigIncludesPhaseOneValidationRanges(t)
}

func TestInitializeWorkspaceCreatesRequiredLayout(t *testing.T) {
	runTestInitializeWorkspaceCreatesRequiredLayout(t)
}

func TestInitializeWorkspaceRejectsEmptyRoot(t *testing.T) {
	runTestInitializeWorkspaceRejectsEmptyRoot(t)
}

func TestInitializeWorkspaceRejectsFileRootAsInvalid(t *testing.T) {
	runTestInitializeWorkspaceRejectsFileRootAsInvalid(t)
}

func TestInitializeWorkspaceRejectsPartialLayout(t *testing.T) {
	runTestInitializeWorkspaceRejectsPartialLayout(t)
}

func runTestOpenWorkspaceReturnsInvalidWorkspaceForEmptyDirectory(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := t.TempDir()

	state, err := service.OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}
	if state.RootPath != root {
		t.Fatalf("RootPath = %q, want %q", state.RootPath, root)
	}
	if state.Mode != "InvalidWorkspace" {
		t.Fatalf("Mode = %q, want %q", state.Mode, "InvalidWorkspace")
	}
	if !strings.Contains(state.Reason, "cache layout") {
		t.Fatalf("Reason = %q, want cache layout explanation", state.Reason)
	}
}

func runTestGetOverviewUsesWarningsForUnavailableWorkspaceStatusFields(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := t.TempDir()
	engine, err := cache.Open(cache.DefaultConfig(root))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := engine.Stats(context.Background()); err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	overview, err := service.GetOverview(root)
	if err != nil {
		t.Fatalf("GetOverview() error = %v", err)
	}
	if overview.WorkspaceMode != "N/A" {
		t.Fatalf("WorkspaceMode = %q, want %q", overview.WorkspaceMode, "N/A")
	}
	if overview.LockMode != "N/A" {
		t.Fatalf("LockMode = %q, want %q", overview.LockMode, "N/A")
	}
	if overview.Health != "N/A" {
		t.Fatalf("Health = %q, want %q", overview.Health, "N/A")
	}
	if overview.BacklogEstimateRecords != nil {
		t.Fatalf("BacklogEstimateRecords = %v, want nil when unavailable", *overview.BacklogEstimateRecords)
	}
	if overview.BacklogEstimateBytes != nil {
		t.Fatalf("BacklogEstimateBytes = %v, want nil when unavailable", *overview.BacklogEstimateBytes)
	}
	if len(overview.Warnings) == 0 {
		t.Fatalf("Warnings = empty, want unavailability warning")
	}
	if overview.Warnings[0].Severity != "warning" {
		t.Fatalf("Warnings[0].Severity = %q, want %q", overview.Warnings[0].Severity, "warning")
	}
	if !strings.Contains(strings.ToLower(overview.Warnings[0].Message), "unavailable") {
		t.Fatalf("Warnings[0].Message = %q, want unavailable explanation", overview.Warnings[0].Message)
	}
}

func runTestGetConfigIncludesPhaseOneValidationRanges(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := t.TempDir()

	cfg, err := service.GetConfig(root)
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}
	if cfg.SegmentTargetSizeBytes.AllowedRange != "64 MiB to 4 GiB" {
		t.Fatalf("SegmentTargetSizeBytes.AllowedRange = %q, want %q", cfg.SegmentTargetSizeBytes.AllowedRange, "64 MiB to 4 GiB")
	}
	if cfg.RetentionDays.AllowedRange != "1 to 365" {
		t.Fatalf("RetentionDays.AllowedRange = %q, want %q", cfg.RetentionDays.AllowedRange, "1 to 365")
	}
	if !cfg.RootDir.StartupOnly {
		t.Fatalf("RootDir.StartupOnly = false, want true")
	}
}

func runTestInitializeWorkspaceCreatesRequiredLayout(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := filepath.Join(t.TempDir(), "cache-root")

	if err := service.InitializeWorkspace(root); err != nil {
		t.Fatalf("InitializeWorkspace() error = %v", err)
	}

	for _, rel := range []string{
		"meta",
		"meta/replay",
		"segments",
		"wal",
		"tools",
		"tools/reports",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}
}

func runTestInitializeWorkspaceRejectsEmptyRoot(t *testing.T) {
	t.Helper()
	t.Parallel()

	err := service.InitializeWorkspace("")
	assertWorkspaceInitErrorCode(t, err, service.WorkspaceInitInvalidRoot)
}

func runTestInitializeWorkspaceRejectsFileRootAsInvalid(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := filepath.Join(t.TempDir(), "cache-root")
	if err := os.WriteFile(root, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := service.InitializeWorkspace(root)
	assertWorkspaceInitErrorCode(t, err, service.WorkspaceInitInvalidRoot)
}

func runTestInitializeWorkspaceRejectsPartialLayout(t *testing.T) {
	t.Helper()
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "meta"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := service.InitializeWorkspace(root)
	assertWorkspaceInitErrorCode(t, err, service.WorkspaceInitPartialLayout)
}

func assertWorkspaceInitErrorCode(t *testing.T, err error, want service.WorkspaceInitCode) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected %q error", want)
	}

	var initErr *service.WorkspaceInitError
	if !errors.As(err, &initErr) {
		t.Fatalf("InitializeWorkspace() error = %T, want *service.WorkspaceInitError", err)
	}
	if initErr.Code != want {
		t.Fatalf("WorkspaceInitError.Code = %q, want %q", initErr.Code, want)
	}
}
