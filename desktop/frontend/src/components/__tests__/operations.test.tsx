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
    fireEvent.click(screen.getByText("Run Verify"));
    act(() => {
      vi.runAllTimers();
    });
    expect(screen.getByRole("button", { name: "Open Repair" })).toBeInTheDocument();
  });

  test("repair-tail requires danger confirmation", () => {
    setOperationState({ selectedOperation: "repair-tail" });
    render(<App />);
    fireEvent.click(screen.getByText("Run Repair-tail"));
    expect(screen.getByText("Confirm Repair Tail")).toBeInTheDocument();
    expect(screen.getAllByText("danger").length).toBeGreaterThan(0);
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
    fireEvent.click(screen.getByText("Run Verify"));
    expect(screen.getByText("running")).toBeInTheDocument();
    act(() => {
      vi.runAllTimers();
    });
    expect(screen.getByText("succeeded")).toBeInTheDocument();
  });
});
