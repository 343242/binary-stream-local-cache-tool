import { act, fireEvent, render, screen } from "@testing-library/react";

import App from "../../App";
import { createInitialState, useAppStore } from "../../state/app-store";

function setOperationState(partial = {}) {
  useAppStore.setState({
    ...createInitialState(),
    workspace: {
      rootPath: "/var/lib/binary-stream/cache-alpha",
      mode: "HealthyMaintenance",
      lockMode: "MaintenanceExclusive",
      health: "ok",
      stale: false,
    },
    page: "operations",
    ...partial,
  });
}

describe("operations page", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    setOperationState();
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
  });

  test("verify result with repairable corruption exposes Open Repair action", async () => {
    render(<App />);
    act(() => {
      fireEvent.click(screen.getByText("执行 校验"));
    });
    act(() => {
      vi.runAllTimers();
    });
    expect(screen.getByRole("button", { name: "打开修复" })).toBeInTheDocument();
  });

  test("repair-tail requires danger confirmation", () => {
    const initial = createInitialState();
    setOperationState({ selectedOperation: "repair-tail", selectedSegment: initial.recentSegments[0] });
    render(<App />);
    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "审查影响" }));
    });
    expect(screen.getByRole("dialog", { name: "影响审查: 修复尾部" })).toBeInTheDocument();
    expect(screen.getByText("影响审查")).toBeInTheDocument();
    expect(screen.getAllByText("只有在工作区处于维护姿态并持有 MaintenanceExclusive 锁时，才能继续这个动作。").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/段文件 48/).length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "执行 修复尾部" })).toBeInTheDocument();
  });

  test("toasts render warning and error durations correctly", () => {
    setOperationState({
      locale: "zh-CN" as const,
      toasts: [
        { id: 1, level: "warning", title: "校验完成", message: "发现可修复损坏。", durationLabel: "6秒" },
        { id: 2, level: "error", title: "停机被阻止", message: "仍然需要人工确认。", durationLabel: "常驻" },
      ],
    });
    render(<App />);
    expect(screen.getByText("警告 · 6秒")).toBeInTheDocument();
    expect(screen.getByText("错误 · 常驻")).toBeInTheDocument();
  });

  test("task panel updates from started to finished", async () => {
    render(<App />);
    act(() => {
      fireEvent.click(screen.getByText("执行 校验"));
    });
    expect(screen.getByRole("heading", { name: "任务时间线" })).toBeInTheDocument();
    expect(screen.getByText("已请求")).toBeInTheDocument();
    expect(screen.getByText("进行中")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "最近结果" })).toBeInTheDocument();
    act(() => {
      vi.runAllTimers();
    });
    expect(screen.getByText("已完成")).toBeInTheDocument();
  });

  test("task panel renders progress metadata and cancel action", () => {
    setOperationState({
      currentTask: {
        taskID: "task-1",
        kind: "verify",
        status: "running",
        target: "",
        phase: "running",
        message: "Running operation",
        startedAt: "2026-04-17 10:00",
        updatedAt: "2026-04-17 10:01",
        progressCurrent: 1,
        progressTotal: 2,
        canCancel: true,
        error: null,
      },
    });
    render(<App />);
    expect(screen.getByRole("heading", { name: "任务时间线" })).toBeInTheDocument();
    expect(screen.getByText("1 / 2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "取消任务" })).toBeInTheDocument();
  });

  test("repair-tail stays visibly blocked before the confirm gate can open", () => {
    setOperationState({
      selectedOperation: "repair-tail",
      selectedSegment: null,
    });
    render(<App />);
    expect(screen.getByText("在确认前被阻止")).toBeInTheDocument();
    expect(screen.getByText("请先在浏览器中选择一个修复目标，或从最近一次 verify 结果中打开修复入口。")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "执行 修复尾部" })).not.toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  test("stale workspace state blocks destructive actions before confirmation", () => {
    const initial = createInitialState();
    setOperationState({
      selectedOperation: "repair-tail",
      selectedSegment: initial.recentSegments[0],
      workspace: {
        rootPath: "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyMaintenance",
        lockMode: "MaintenanceExclusive",
        health: "ok",
        stale: true,
      },
    });
    render(<App />);
    expect(screen.getByText("在确认前被阻止")).toBeInTheDocument();
    expect(screen.getAllByText(/陈旧快照/i).length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "审查影响" })).not.toBeInTheDocument();
  });
});
