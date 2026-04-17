import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";

import App from "../../App";
import EmptyState from "../EmptyState";
import LoadingSkeleton from "../LoadingSkeleton";
import StatusCard from "../StatusCard";
import { createInitialState, useAppStore } from "../../state/app-store";
import styles from "../../styles/shell.module.css";

function resetStore(partial?: Partial<ReturnType<typeof createInitialState>>) {
  useAppStore.setState({
    ...createInitialState(),
    ...partial,
  });
}

describe("desktop app pages", () => {
  beforeEach(() => {
    resetStore();
    delete window.go;
  });

  test("shows the landing empty state before a workspace is open", () => {
    render(<App />);
    expect(screen.getByText("Open Cache Directory")).toBeInTheDocument();
    expect(screen.getByText("Recent Directories")).toBeInTheDocument();
  });

  it("renders workspace status inside the primary workspace rail", () => {
    render(<App />);
    const workspaceRail = screen.getByRole("complementary", { name: /primary workspace/i });

    expect(within(workspaceRail).getByRole("heading", { level: 1, name: /observer-first desktop console/i })).toBeInTheDocument();
    expect(within(workspaceRail).getByText("Workspace status")).toBeInTheDocument();
    expect(within(workspaceRail).getByText(/fresh snapshot/i)).toBeInTheDocument();
  });

  it("renders the current snapshot header around the workspace path", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "overview",
    });

    render(<App />);

    const snapshotHeader = screen.getByRole("banner");

    expect(within(snapshotHeader).getByText("Current snapshot")).toBeInTheDocument();
    expect(within(snapshotHeader).getByRole("heading", { level: 2, name: "/var/lib/binary-stream/cache-alpha" })).toBeInTheDocument();
    expect(within(snapshotHeader).getByText("Mode: HealthyObserver")).toBeInTheDocument();
  });

  test("renders overview cards after overview data loads", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "overview",
    });
    render(<App />);
    expect(screen.getByText("Workspace")).toBeInTheDocument();
    expect(screen.getByText("HealthyObserver")).toBeInTheDocument();
    expect(screen.getByText("Recent Segments")).toBeInTheDocument();
  });

  it("renders overview as a narrative page with approved hero, warnings, and activity regions", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "overview",
    });

    render(<App />);
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /readable enough for routine checks\. severe enough for maintenance windows\./i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/^Maintenance windows$/i)).toBeInTheDocument();
    expect(screen.getByText(/^Recent activity$/i)).toBeInTheDocument();
  });

  test("shows no segments empty state when explorer has zero rows", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "explorer",
      recentSegments: [],
      selectedSegment: null,
    });
    render(<App />);
    expect(screen.getByText("No Segments Found")).toBeInTheDocument();
  });

  test("renders the config page as read-only", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "config",
    });
    render(<App />);
    expect(screen.getByText("Read-only inspection. Persistent configuration editing is deferred.")).toBeInTheDocument();
    expect(screen.getAllByText("Allowed Range").length).toBeGreaterThan(0);
  });

  test("reopen uses the selected recent workspace path", () => {
    render(<App />);
    fireEvent.click(screen.getAllByText("Reopen")[1]);
    expect(screen.getByText("/srv/cache/replica-west")).toBeInTheDocument();
  });

  test("shows loading feedback while a workspace is hydrating", () => {
    resetStore({
      workspaceLoadState: "hydrating",
    });
    render(<App />);
    expect(screen.getByText("Loading workspace")).toBeInTheDocument();
    expect(screen.getByLabelText("Loading")).toBeInTheDocument();
  });

  test("clicking a segment row updates the detail pane", async () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "explorer",
      selectedSegment: null,
    });
    render(<App />);
    expect(screen.getByText("Select a row to inspect detailed fields and raw preview data.")).toBeInTheDocument();
    fireEvent.click(screen.getByText("48"));
    await waitFor(() => {
      expect(screen.getByText("segments/48.seg")).toBeInTheDocument();
    });
  });

  test("explorer requests bound detail when bindings are available", async () => {
    const getSegmentDetail = vi.fn().mockResolvedValue({
      segmentID: 48,
      path: "segments/48.seg",
      sizeBytes: 134217728,
      sealed: false,
      footerStatus: "healthy",
      tailStatus: "healthy",
      firstWriteSeq: 12442,
      lastWriteSeq: 12910,
      recordCount: 468,
      blockCount: 12,
      minEventTime: 1713171600000,
      maxEventTime: 1713172500000,
      lastBatchSeq: 812,
      rawPreviewHex: "00ff",
      structuredPreview: [],
    });

    window.go = {
      backend: {
        App: {
          OpenWorkspace: vi.fn(),
          GetRecentWorkspaces: vi.fn().mockResolvedValue([]),
          GetWorkspaceState: vi.fn().mockResolvedValue({ rootPath: "", mode: "NoWorkspace", lockMode: "N/A", health: "N/A" }),
          GetSegmentDetail: getSegmentDetail,
        },
      },
    } as any;

    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "explorer",
      selectedSegment: null,
    });

    render(<App />);
    fireEvent.click(screen.getByText("48"));
    await waitFor(() => expect(getSegmentDetail).toHaveBeenCalledWith(48));
  });
});

describe("editorial component boundaries", () => {
  test("renders empty state chrome only for panel usage", () => {
    const { container, rerender } = render(<EmptyState title="No warnings" message="Inline copy" variant="inline" />);

    const inlineRoot = container.querySelector("section");
    expect(inlineRoot).not.toHaveClass(styles.panel);
    expect(inlineRoot).not.toHaveClass(styles.panelPadding);

    rerender(<EmptyState title="No warnings" message="Panel copy" variant="panel" />);

    const panelRoot = container.querySelector("section");
    expect(panelRoot).toHaveClass(styles.panel);
    expect(panelRoot).toHaveClass(styles.panelPadding);
  });

  test("supports loading skeleton width presets through variants", () => {
    const { container, rerender } = render(<LoadingSkeleton rows={3} />);

    const defaultWidths = Array.from(container.querySelectorAll(`.${styles.skeleton}`)).map(
      (row) => (row as HTMLElement).style.width,
    );
    expect(defaultWidths).toEqual(["100%", "88%", "72%"]);

    rerender(<LoadingSkeleton rows={3} variant="compact" />);

    const compactWidths = Array.from(container.querySelectorAll(`.${styles.skeleton}`)).map(
      (row) => (row as HTMLElement).style.width,
    );
    expect(compactWidths).toEqual(["72%", "64%", "56%"]);
  });

  test("renders status-card eyebrow only when provided", () => {
    render(
      <>
        <StatusCard label="Workspace" value="HealthyObserver" secondary="Observer mode" />
        <StatusCard eyebrow="Overview" label="Warnings" value="1 warning" secondary="Backlog estimate unavailable" />
      </>,
    );

    expect(screen.queryByText("Inspection")).not.toBeInTheDocument();
    expect(screen.getByText("Overview")).toBeInTheDocument();
  });
});
