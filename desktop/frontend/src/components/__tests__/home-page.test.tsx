import { fireEvent, render, screen } from "@testing-library/react";

import HomePage from "../../pages/HomePage";
import type { WriterAlert, WriterEventVM, WriterStatusState, WorkspaceState } from "../../state/app-store";

const workspace: WorkspaceState = {
  rootPath: "/var/lib/binary-stream/cache-alpha",
  mode: "HealthyObserver",
  lockMode: "ObserverShared",
  health: "ok",
  stale: false,
};

const stoppedStatus: WriterStatusState = {
  lifecycleState: "stopped",
  workspaceState: "HealthyObserver",
  rootPath: workspace.rootPath,
  lastError: "",
  startedAtUnixMs: 0,
  stoppedAtUnixMs: 0,
};

const runningStatus: WriterStatusState = {
  lifecycleState: "running",
  workspaceState: "HealthyWriter",
  rootPath: workspace.rootPath,
  lastError: "",
  startedAtUnixMs: 1713600000000,
  stoppedAtUnixMs: 0,
};

const alerts: WriterAlert[] = [
  {
    id: "writer-running",
    level: "info",
    title: "写入运行中",
    message: "写入器当前正在本地工作区中运行。",
  },
];

const events: WriterEventVM[] = [
  {
    kind: "writer-started",
    message: "Writer started for /var/lib/binary-stream/cache-alpha",
    timestampUnixMs: 1713600000000,
    timestampLabel: "2026-04-20 08:00",
  },
];

describe("home page", () => {
  test("renders writer controls, alerts, and event feed", () => {
    const onStartWriter = vi.fn();
    const onStopWriter = vi.fn();
    const onOpenWriterConfig = vi.fn();

    render(
      <HomePage
        workspace={workspace}
        writerStatus={stoppedStatus}
        writerAlerts={alerts}
        writerEvents={events}
        onStartWriter={onStartWriter}
        onStopWriter={onStopWriter}
        onOpenWriterConfig={onOpenWriterConfig}
      />,
    );

    expect(screen.getByRole("heading", { level: 2, name: "本地写入控制" })).toBeInTheDocument();
    expect(screen.getByText("写入运行中")).toBeInTheDocument();
    expect(screen.getByText("Writer started for /var/lib/binary-stream/cache-alpha")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "开始写入" }));
    fireEvent.click(screen.getByRole("button", { name: "写入配置" }));

    expect(onStartWriter).toHaveBeenCalledTimes(1);
    expect(onStopWriter).not.toHaveBeenCalled();
    expect(onOpenWriterConfig).toHaveBeenCalledTimes(1);
  });

  test("shows stop action while the writer is running", () => {
    const onStartWriter = vi.fn();
    const onStopWriter = vi.fn();

    render(
      <HomePage
        workspace={workspace}
        writerStatus={runningStatus}
        writerAlerts={alerts}
        writerEvents={events}
        onStartWriter={onStartWriter}
        onStopWriter={onStopWriter}
        onOpenWriterConfig={vi.fn()}
      />,
    );

    expect(screen.getByText((_, node) => node?.textContent === "状态: 运行中")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "停止写入" }));

    expect(onStartWriter).not.toHaveBeenCalled();
    expect(onStopWriter).toHaveBeenCalledTimes(1);
  });
});
