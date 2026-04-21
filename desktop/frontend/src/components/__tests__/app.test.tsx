import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";

import App from "../../App";
import EmptyState from "../EmptyState";
import LoadingSkeleton from "../LoadingSkeleton";
import StatusCard from "../StatusCard";
import LandingPage from "../../pages/LandingPage";
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
    expect(screen.getByText("打开缓存目录")).toBeInTheDocument();
    expect(screen.getByText("最近目录")).toBeInTheDocument();
    expect(screen.getByText("支持的工作区")).toBeInTheDocument();
  });

  it("renders workspace status inside the primary workspace rail", () => {
    render(<App />);
    const workspaceRail = screen.getByRole("complementary", { name: "主工作区" });

    expect(within(workspaceRail).getByRole("heading", { level: 1, name: "以观测为先的桌面控制台" })).toBeInTheDocument();
    expect(within(workspaceRail).getByText("工作区状态")).toBeInTheDocument();
    expect(within(workspaceRail).getByText("未打开工作区")).toBeInTheDocument();
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

    expect(within(snapshotHeader).getByText("当前快照")).toBeInTheDocument();
    expect(within(snapshotHeader).getByRole("heading", { level: 2, name: "/var/lib/binary-stream/cache-alpha" })).toBeInTheDocument();
    expect(within(snapshotHeader).getByText("模式: 健康观察模式")).toBeInTheDocument();
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
    expect(screen.getByText("工作区")).toBeInTheDocument();
    expect(screen.getByText("健康观察模式")).toBeInTheDocument();
    expect(screen.getByText("最近段文件")).toBeInTheDocument();
    expect(screen.getByTestId("overview-metric-lead")).toBeInTheDocument();
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
        name: "日常巡检保持可读，维护窗口保持足够严肃。",
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("维护窗口")).toBeInTheDocument();
    expect(screen.getByText("近期活动")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "开始写入" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "停止写入" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "写入配置" })).not.toBeInTheDocument();
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
    expect(screen.getByText("未发现段文件")).toBeInTheDocument();
  });

  it("keeps explorer detail content separate from the row list", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "explorer",
    });

    render(<App />);

    expect(screen.getByText("详情面板")).toBeInTheDocument();
    expect(screen.getByText("检查说明")).toBeInTheDocument();
    expect(screen.getByText("选择一行后，可以在这里查看详细字段和原始预览数据。")).toBeInTheDocument();
  });

  it("keeps explorer audit copy visible when wal detail is unavailable", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "explorer",
      explorerTab: "wal",
      walDetail: null,
      explorerDetailLoading: false,
      explorerDetailError: null,
    });

    render(<App />);

    expect(screen.getByText("当前没有 WAL")).toBeInTheDocument();
    expect(screen.getByText("当前工作区没有可供检查的 WAL 快照。")).toBeInTheDocument();
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
    expect(screen.getByText("当前仅支持只读检查，持久化配置编辑仍然暂缓。")).toBeInTheDocument();
    expect(screen.getAllByText("允许范围").length).toBeGreaterThan(0);
  });

  test("defaults to Home when a workspace is open", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
    });

    render(<App />);

    expect(screen.getByRole("heading", { level: 2, name: "本地写入控制" })).toBeInTheDocument();
    expect(screen.queryByText("日常巡检保持可读，维护窗口保持足够严肃。")).not.toBeInTheDocument();
  });

  test("config page opens the writer-config modal", () => {
    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "config" as any,
    });

    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "写入配置" }));

    expect(screen.getByRole("dialog", { name: "写入配置" })).toBeInTheDocument();
  });

  it("marks config as read-only audit content", () => {
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

    expect(screen.getByText("配置审计账本")).toBeInTheDocument();
    expect(screen.getByText("当前仅支持只读检查，持久化配置编辑仍然暂缓。")).toBeInTheDocument();
  });

  test("reopen uses the selected recent workspace path", () => {
    render(<App />);
    fireEvent.click(screen.getAllByText("重新打开")[1]);
    expect(screen.getByText("/srv/cache/replica-west")).toBeInTheDocument();
  });

  test("locale toggle switches the shell from Chinese to English", () => {
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "EN" }));

    expect(screen.getByText("Open Cache Directory")).toBeInTheDocument();
    expect(screen.getByText("Recent Directories")).toBeInTheDocument();
  });

  test("navigation is gated with a friendly toast before a workspace opens", () => {
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "浏览器" }));

    expect(screen.getByText("请先打开缓存目录")).toBeInTheDocument();
    expect(screen.getByText("打开缓存目录")).toBeInTheDocument();
  });

  test("overview navigation stays gated until a workspace opens", () => {
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "概览" }));

    expect(screen.getByText("请先打开缓存目录")).toBeInTheDocument();
    expect(screen.getByText("打开缓存目录")).toBeInTheDocument();
  });

  test("invalid workspace recovery reopens the chooser instead of auto-reopening a recent path", () => {
    const onOpenWorkspace = vi.fn();

    render(
      <LandingPage
        recentWorkspaces={["/var/lib/binary-stream/cache-alpha"]}
        invalidWorkspace={{ path: "/tmp/bad-root", reason: "layout mismatch" }}
        onOpenWorkspace={onOpenWorkspace}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "重新选择目录" }));
    expect(onOpenWorkspace).toHaveBeenCalledWith();
  });

  test("hydrated stale workspaces surface stale snapshot state in the shell", async () => {
    window.go = {
      backend: {
        App: {
          OpenWorkspace: vi.fn().mockResolvedValue({
            rootPath: "/var/lib/binary-stream/cache-alpha",
            mode: "HealthyMaintenance",
            lockMode: "MaintenanceExclusive",
            health: "ok",
            canRefresh: true,
            canRunVerify: true,
            canRunCloseCheck: true,
            canRunRepairTail: true,
            canRunShutdown: true,
            reason: "",
          }),
          GetRecentWorkspaces: vi.fn().mockResolvedValue([]),
          GetWorkspaceState: vi.fn().mockResolvedValue({
            rootPath: "/var/lib/binary-stream/cache-alpha",
            mode: "HealthyMaintenance",
            lockMode: "MaintenanceExclusive",
            health: "ok",
            canRefresh: true,
            canRunVerify: true,
            canRunCloseCheck: true,
            canRunRepairTail: true,
            canRunShutdown: true,
            reason: "",
          }),
          GetOverview: vi.fn().mockResolvedValue({
            rootPath: "/var/lib/binary-stream/cache-alpha",
            workspaceMode: "HealthyMaintenance",
            lockMode: "MaintenanceExclusive",
            health: "ok",
            segmentCount: 48,
            activeSegmentID: 48,
            activeSegmentSizeBytes: 134217728,
            walSizeBytes: 32768,
            nextWriteSeq: 12911,
            retentionDays: 14,
            checkpointsTotal: 96,
            segmentFsyncTotal: 311,
            lastAckedWriteSeq: 12441,
            gracefulShutdownsTotal: 0,
            ungracefulRecoveriesTotal: 0,
            segmentTailRepairsTotal: 0,
            backlogEstimateRecords: null,
            backlogEstimateBytes: null,
            warnings: [],
            lastRefreshedAt: Date.now(),
            isStale: true,
          }),
          ListSegments: vi.fn().mockResolvedValue({ items: [], page: 1, pageSize: 8, totalItems: 0, hasNext: false }),
          ListCursors: vi.fn().mockResolvedValue([]),
          GetConfig: vi.fn().mockResolvedValue({
            segmentTargetSizeBytes: { displayName: "Segment Target Size", currentValue: "1", defaultValue: "1", allowedRange: "1", startupOnly: true },
            segmentSlackSizeBytes: { displayName: "Segment Slack Size", currentValue: "1", defaultValue: "1", allowedRange: "1", startupOnly: true },
            blockTargetSizeBytes: { displayName: "Block Target Size", currentValue: "1", defaultValue: "1", allowedRange: "1", startupOnly: true },
            checkpointInterval: { displayName: "Checkpoint Interval", currentValue: "1s", defaultValue: "1s", allowedRange: "1s", startupOnly: true },
            checkpointBytes: { displayName: "Checkpoint Bytes", currentValue: "1", defaultValue: "1", allowedRange: "1", startupOnly: true },
            segmentFsyncInterval: { displayName: "Segment Fsync Interval", currentValue: "1s", defaultValue: "1s", allowedRange: "1s", startupOnly: true },
            segmentFsyncBytes: { displayName: "Segment Fsync Bytes", currentValue: "1", defaultValue: "1", allowedRange: "1", startupOnly: true },
            retentionDays: { displayName: "Retention Days", currentValue: "14", defaultValue: "14", allowedRange: "1-365", startupOnly: true },
          }),
        },
      },
    } as any;

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("陈旧快照")).toBeInTheDocument();
    });
  });

  test("shows loading feedback while a workspace is hydrating", () => {
    resetStore({
      workspaceLoadState: "hydrating",
    });
    render(<App />);
    expect(screen.getByText("正在加载工作区")).toBeInTheDocument();
    expect(screen.getByLabelText("加载中")).toBeInTheDocument();
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
    expect(screen.getByText("选择一行后，可以在这里查看详细字段和原始预览数据。")).toBeInTheDocument();
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

  test("writer runtime events update the home page without refresh", async () => {
    const listeners: Record<string, (...payload: any[]) => void> = {};

    (window as any).runtime = {
      EventsOn: vi.fn((eventName: string, callback: (...payload: any[]) => void) => {
        listeners[eventName] = callback;
        return () => {
          delete listeners[eventName];
        };
      }),
    };

    resetStore({
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
      page: "home" as any,
    });

    render(<App />);

    act(() => {
      listeners["writer:status-changed"]?.({
        lifecycleState: "running",
        workspaceState: "HealthyWriter",
        rootPath: "/var/lib/binary-stream/cache-alpha",
        lastError: "",
        startedAtUnixMs: 1713600000000,
        stoppedAtUnixMs: 0,
      });
      listeners["writer:events-changed"]?.([
        {
          kind: "writer-started",
          message: "Writer started for /var/lib/binary-stream/cache-alpha",
          timestampUnixMs: 1713600000000,
        },
      ]);
    });

    await waitFor(() => {
      expect(screen.getByText((_, node) => node?.textContent === "写入器: 运行中")).toBeInTheDocument();
      expect(screen.getByText("Writer started for /var/lib/binary-stream/cache-alpha")).toBeInTheDocument();
    });
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
        <StatusCard eyebrow="Overview" label="Warnings" value="1 warning" secondary="Backlog estimate unavailable" tier="hero" />
      </>,
    );

    expect(screen.queryByText("Inspection")).not.toBeInTheDocument();
    expect(screen.getByText("Overview")).toBeInTheDocument();
    const cards = screen.getAllByRole("article");
    expect(cards[0]).not.toHaveClass(styles.statusCardHero);
    expect(cards[1]).toHaveClass(styles.statusCardHero);
  });
});
