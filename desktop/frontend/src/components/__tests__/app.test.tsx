import { fireEvent, render, screen } from "@testing-library/react";

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
  });

  test("shows the landing empty state before a workspace is open", () => {
    render(<App />);
    expect(screen.getByText("Open Cache Directory")).toBeInTheDocument();
    expect(screen.getByText("Recent Directories")).toBeInTheDocument();
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
});
