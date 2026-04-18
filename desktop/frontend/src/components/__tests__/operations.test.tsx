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
      fireEvent.click(screen.getByText("Run Verify"));
    });
    act(() => {
      vi.runAllTimers();
    });
    expect(screen.getByRole("button", { name: "Open Repair" })).toBeInTheDocument();
  });

  test("repair-tail requires danger confirmation", () => {
    const initial = createInitialState();
    setOperationState({ selectedOperation: "repair-tail", selectedSegment: initial.recentSegments[0] });
    render(<App />);
    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "Review impact" }));
    });
    expect(screen.getByRole("dialog", { name: "Impact review: Repair-tail" })).toBeInTheDocument();
    expect(screen.getByText("Impact review")).toBeInTheDocument();
    expect(screen.getByText("This action is still blocked until you review the maintenance impact.")).toBeInTheDocument();
    expect(screen.getAllByText(/Selected segment/).length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Authorize Repair Tail" })).toBeInTheDocument();
  });

  test("toasts render warning and error durations correctly", () => {
    setOperationState({
      toasts: [
        { id: 1, level: "warning", title: "Verify completed", message: "Repairable corruption found.", durationLabel: "6s" },
        { id: 2, level: "error", title: "Shutdown blocked", message: "Manual acknowledgement is still required.", durationLabel: "persistent" },
      ],
    });
    render(<App />);
    expect(screen.getByText("warning · 6s")).toBeInTheDocument();
    expect(screen.getByText("error · persistent")).toBeInTheDocument();
  });

  test("task panel updates from started to finished", async () => {
    render(<App />);
    act(() => {
      fireEvent.click(screen.getByText("Run Verify"));
    });
    expect(screen.getByRole("heading", { name: "Task timeline" })).toBeInTheDocument();
    expect(screen.getByText("Requested")).toBeInTheDocument();
    expect(screen.getByText("In progress")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Latest result" })).toBeInTheDocument();
    act(() => {
      vi.runAllTimers();
    });
    expect(screen.getByText("Completed")).toBeInTheDocument();
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
    expect(screen.getByRole("heading", { name: "Task timeline" })).toBeInTheDocument();
    expect(screen.getByText("1 / 2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel Task" })).toBeInTheDocument();
  });

  test("repair-tail stays visibly blocked before the confirm gate can open", () => {
    setOperationState({
      selectedOperation: "repair-tail",
      selectedSegment: null,
    });
    render(<App />);
    expect(screen.getByText("Blocked before confirmation")).toBeInTheDocument();
    expect(screen.getByText("Select a repair candidate from Explorer or from the latest verify result before the confirm gate can open.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Run Repair-tail" })).not.toBeInTheDocument();
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
    expect(screen.getByText("Blocked before confirmation")).toBeInTheDocument();
    expect(screen.getAllByText(/stale snapshot/i).length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Review impact" })).not.toBeInTheDocument();
  });
});
