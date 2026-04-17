import { fireEvent, render, screen, waitFor } from "@testing-library/react";

import App from "../../App";
import { createInitialState, useAppStore } from "../../state/app-store";

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

  it("shows workspace context in the rail and snapshot header", () => {
    render(<App />);
    expect(screen.getByText(/observer-first desktop console/i)).toBeInTheDocument();
    expect(screen.getByText(/fresh snapshot/i)).toBeInTheDocument();
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
