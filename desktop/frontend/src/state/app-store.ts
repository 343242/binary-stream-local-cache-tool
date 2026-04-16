import { create } from "zustand";

import { bindings, hasBindings, type ConfigVM, type OverviewVM, type PagedSegmentsVM, type WorkspaceState as BoundWorkspaceState } from "../bindings";
import { hasRuntime, subscribeToEvent } from "../runtime";

export type PageKey = "overview" | "explorer" | "config" | "operations";
export type ExplorerTab = "segments" | "wal" | "cursors" | "checkpoint";
export type OperationKey = "verify" | "close-check" | "repair-tail" | "shutdown";

export type WorkspaceState = {
  rootPath: string;
  mode: string;
  lockMode: string;
  health: string;
  stale: boolean;
  invalidReason?: string;
};

export type OverviewCard = {
  label: string;
  value: string;
  secondary: string;
};

export type SegmentRow = {
  segmentID: number;
  sizeBytes: string;
  sealed: boolean;
  firstWriteSeq: number;
  lastWriteSeq: number;
  recordCount: number;
  minEventTime: string;
  maxEventTime: string;
  lastBatchSeq: number;
  health: string;
};

export type CursorRow = {
  destination: string;
  writeSeq: number;
  updatedAt: string;
  status: string;
};

export type ConfigRow = {
  field: string;
  effective: string;
  defaultValue: string;
  allowedRange: string;
  note: string;
};

export type OperationResultState = {
  summary: string;
  details: { key: string; value: string }[];
  changed: boolean;
  repairableSegment?: number;
};

export type TaskState = {
  taskID: string;
  kind: OperationKey;
  status: "running" | "succeeded" | "failed" | "cancelled";
  phase: string;
  message: string;
};

export type ToastState = {
  id: number;
  level: "info" | "success" | "warning" | "error";
  title: string;
  message: string;
  durationLabel: string;
};

export type ConfirmDialogState = {
  operation: OperationKey;
  title: string;
  riskLevel: "accent" | "danger";
  summary: string;
  impactLines: string[];
  confirmLabel: string;
  cancelLabel: string;
};

type ShellState = {
  title: string;
  page: PageKey;
  explorerTab: ExplorerTab;
  workspace: WorkspaceState | null;
  recentWorkspaces: string[];
  overviewCards: OverviewCard[];
  warningSummary: string[];
  recentSegments: SegmentRow[];
  recentCursors: CursorRow[];
  selectedSegment: SegmentRow | null;
  configSections: Record<string, ConfigRow[]>;
  selectedOperation: OperationKey;
  latestResult: OperationResultState | null;
  currentTask: TaskState | null;
  toasts: ToastState[];
  confirmDialog: ConfirmDialogState | null;
  nextToastID: number;
  setTitle: (title: string) => void;
  setPage: (page: PageKey) => void;
  setExplorerTab: (tab: ExplorerTab) => void;
  setSelectedOperation: (operation: OperationKey) => void;
  loadDemoWorkspace: (rootPath?: string) => void;
  markInvalidWorkspace: (path: string, reason: string) => void;
  refresh: () => void;
  requestOperation: (operation: OperationKey) => void;
  confirmOperation: () => void;
  dismissDialog: () => void;
  openRepairFromResult: () => void;
  dismissToast: (id: number) => void;
  initialiseRuntime: () => void;
};

const demoOverviewCards: OverviewCard[] = [
  { label: "Workspace", value: "HealthyObserver", secondary: "Local cache root is readable" },
  { label: "Lock State", value: "ObserverShared", secondary: "Maintenance actions remain gated" },
  { label: "Storage Footprint", value: "48 segments", secondary: "Active segment 000048 · 128 MiB" },
  { label: "Replay Progress", value: "12,441", secondary: "Last acked write sequence" },
  { label: "Checkpoint / Fsync", value: "96 / 311", secondary: "Checkpoint total / segment fsync total" },
  { label: "Warnings", value: "1 warning", secondary: "Backlog estimate unavailable in demo mode" },
];

const demoSegments: SegmentRow[] = [
  {
    segmentID: 48,
    sizeBytes: "134,217,728",
    sealed: false,
    firstWriteSeq: 12442,
    lastWriteSeq: 12910,
    recordCount: 468,
    minEventTime: "2026-04-15 14:20",
    maxEventTime: "2026-04-15 14:35",
    lastBatchSeq: 812,
    health: "open",
  },
  {
    segmentID: 47,
    sizeBytes: "134,217,728",
    sealed: true,
    firstWriteSeq: 11950,
    lastWriteSeq: 12441,
    recordCount: 492,
    minEventTime: "2026-04-15 13:50",
    maxEventTime: "2026-04-15 14:18",
    lastBatchSeq: 811,
    health: "sealed",
  },
];

const demoCursors: CursorRow[] = [
  { destination: "analytics", writeSeq: 12441, updatedAt: "2026-04-15 14:18", status: "ok" },
  { destination: "warehouse", writeSeq: 12388, updatedAt: "2026-04-15 14:14", status: "lagging" },
];

const demoConfigSections: Record<string, ConfigRow[]> = {
  Segment: [
    { field: "Segment Target Size", effective: "134217728", defaultValue: "134217728", allowedRange: "64 MiB to 4 GiB", note: "Startup-only" },
    { field: "Segment Slack Size", effective: "4194304", defaultValue: "4194304", allowedRange: "1 MiB to 64 MiB", note: "Startup-only" },
  ],
  Block: [
    { field: "Block Target Size", effective: "1048576", defaultValue: "1048576", allowedRange: "256 KiB to 4 MiB", note: "Startup-only" },
  ],
  Checkpoint: [
    { field: "Checkpoint Interval", effective: "5s", defaultValue: "5s", allowedRange: "1s to 60s", note: "Startup-only" },
    { field: "Checkpoint Bytes", effective: "67108864", defaultValue: "67108864", allowedRange: "4 MiB to 1 GiB", note: "Startup-only" },
  ],
  Fsync: [
    { field: "Segment Fsync Interval", effective: "250ms", defaultValue: "250ms", allowedRange: "10ms to 5s", note: "Startup-only" },
    { field: "Segment Fsync Bytes", effective: "8388608", defaultValue: "8388608", allowedRange: "1 MiB to 64 MiB", note: "Startup-only" },
  ],
  Retention: [
    { field: "Retention Days", effective: "14", defaultValue: "14", allowedRange: "1 to 365", note: "Startup-only" },
  ],
};

export const createInitialState = (): Omit<
  ShellState,
  | "setTitle"
  | "setPage"
  | "setExplorerTab"
  | "setSelectedOperation"
  | "loadDemoWorkspace"
  | "markInvalidWorkspace"
  | "refresh"
  | "requestOperation"
  | "confirmOperation"
  | "dismissDialog"
  | "openRepairFromResult"
  | "dismissToast"
  | "initialiseRuntime"
> => ({
  title: "Binary Stream Cache Tool",
  page: "overview",
  explorerTab: "segments",
  workspace: null,
  recentWorkspaces: [
    "/var/lib/binary-stream/cache-alpha",
    "/srv/cache/replica-west",
  ],
  overviewCards: demoOverviewCards,
  warningSummary: ["Backlog estimate unavailable in demo mode."],
  recentSegments: demoSegments,
  recentCursors: demoCursors,
  selectedSegment: demoSegments[0],
  configSections: demoConfigSections,
  selectedOperation: "verify",
  latestResult: null,
  currentTask: null,
  toasts: [],
  confirmDialog: null,
  nextToastID: 1,
});

export const useAppStore = create<ShellState>((set, get) => ({
  ...createInitialState(),
  setTitle: (title) => set({ title }),
  setPage: (page) => set({ page }),
  setExplorerTab: (tab) => set({ explorerTab: tab }),
  setSelectedOperation: (operation) => set({ selectedOperation: operation }),
  loadDemoWorkspace: (rootPath) => {
    if (hasBindings()) {
      if (rootPath) {
        void hydrateFromBindings(rootPath, set);
      } else {
        void bindings.chooseWorkspace().then((workspace) => {
          if (workspace.rootPath) {
            void hydrateFromBindings(workspace.rootPath, set);
          }
        }).catch(() => undefined);
      }
      return;
    }
    set({
      workspace: {
        rootPath: rootPath ?? "/var/lib/binary-stream/cache-alpha",
        mode: "HealthyObserver",
        lockMode: "ObserverShared",
        health: "ok",
        stale: false,
      },
    });
  },
  markInvalidWorkspace: (path, reason) =>
    set({
      workspace: {
        rootPath: path,
        mode: "InvalidWorkspace",
        lockMode: "N/A",
        health: "invalid",
        stale: false,
        invalidReason: reason,
      },
    }),
  refresh: () => {
    if (hasBindings()) {
      const workspace = get().workspace;
      if (workspace?.rootPath) {
        void hydrateFromBindings(workspace.rootPath, set);
        return;
      }
    }
    set((state) => ({
      workspace: state.workspace ? { ...state.workspace, stale: false } : state.workspace,
    }));
  },
  requestOperation: (operation) => {
    if (operation === "repair-tail" || operation === "shutdown") {
      set({
        confirmDialog: {
          operation,
          title: operation === "repair-tail" ? "Confirm Repair Tail" : "Confirm Shutdown",
          riskLevel: "danger",
          summary:
            operation === "repair-tail"
              ? "Repairing a segment tail will mutate on-disk workspace state."
              : "Shutdown is a high-risk maintenance action.",
          impactLines:
            operation === "repair-tail"
              ? ["Requires MaintenanceExclusive lock", "May rewrite truncated segment tail"]
              : ["Requires MaintenanceExclusive lock", "Cannot be cancelled after confirmation"],
          confirmLabel: operation === "repair-tail" ? "Run Repair Tail" : "Run Shutdown",
          cancelLabel: "Cancel",
        },
      });
      return;
    }
    startOperation(operation, set, get);
  },
  confirmOperation: () => {
    const dialog = get().confirmDialog;
    if (!dialog) {
      return;
    }
    set({ confirmDialog: null });
    startOperation(dialog.operation, set, get);
  },
  dismissDialog: () => set({ confirmDialog: null }),
  openRepairFromResult: () =>
    set({
      selectedOperation: "repair-tail",
      page: "operations",
    }),
  dismissToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((toast) => toast.id !== id),
    })),
  initialiseRuntime: () => {
    if (hasBindings()) {
      void bindings.getRecentWorkspaces().then((recentWorkspaces) => set({ recentWorkspaces })).catch(() => undefined);
      void bindings.getWorkspaceState().then((workspace) => {
        if (workspace.rootPath) {
          void hydrateFromBindings(workspace.rootPath, set);
        }
      }).catch(() => undefined);
    }
    if (hasRuntime() && !runtimeInitialised) {
      runtimeInitialised = true;
      runtimeUnsubscribers = [
        subscribeToEvent("workspace:changed", (payload) => {
          const workspace = pickPayload(payload);
          if (!workspace || typeof workspace !== "object") {
            return;
          }
          const mapped = mapWorkspaceState(workspace as BoundWorkspaceState);
          set({ workspace: mapped });
          if (mapped.rootPath && mapped.mode !== "InvalidWorkspace") {
            void hydrateFromBindings(mapped.rootPath, set);
          }
        }),
        subscribeToEvent("task:started", (payload) => {
          const task = pickPayload(payload);
          if (!task || typeof task !== "object") {
            return;
          }
          set({
            currentTask: mapTask(task as Record<string, unknown>),
          });
        }),
        subscribeToEvent("task:progress", (payload) => {
          const task = pickPayload(payload);
          if (!task || typeof task !== "object") {
            return;
          }
          set({
            currentTask: mapTask(task as Record<string, unknown>),
          });
        }),
        subscribeToEvent("task:finished", (payload) => {
          const task = pickPayload(payload);
          if (!task || typeof task !== "object") {
            return;
          }
          const mappedTask = mapTask(task as Record<string, unknown>);
          const result = mapTaskResult(task as Record<string, unknown>);
          set((state) => ({
            currentTask: mappedTask,
            latestResult: result ?? state.latestResult,
            toasts: [
              ...state.toasts,
              makeToastFromTask(mappedTask, result, state.nextToastID),
            ],
            nextToastID: state.nextToastID + 1,
          }));
        }),
      ];
    }
  },
}));

let runtimeInitialised = false;
let runtimeUnsubscribers: Array<() => void> = [];

async function hydrateFromBindings(rootPath: string, set: typeof useAppStore.setState) {
  try {
    const workspace = await bindings.openWorkspace(rootPath);
    if (workspace.mode === "InvalidWorkspace") {
      set({
        workspace: mapWorkspaceState(workspace),
      });
      return;
    }

    const [overview, segments, cursors, config] = await Promise.all([
      bindings.getOverview(),
      bindings.listSegments(1, 8),
      bindings.listCursors(),
      bindings.getConfig(),
    ]);

    set({
      workspace: mapWorkspaceState(workspace),
      overviewCards: mapOverviewCards(overview),
      warningSummary: overview.warnings.map((warning) => warning.message),
      recentSegments: mapSegments(segments),
      recentCursors: cursors.map((cursor) => ({
        destination: cursor.destination,
        writeSeq: cursor.writeSeq,
        updatedAt: formatTimestamp(cursor.updatedAt),
        status: cursor.status,
      })),
      selectedSegment: mapSegments(segments)[0] ?? null,
      configSections: mapConfigSections(config),
    });
  } catch {
    set({
      workspace: {
        rootPath,
        mode: "DegradedReadOnly",
        lockMode: "N/A",
        health: "degraded",
        stale: true,
        invalidReason: "Backend binding failed; showing fallback shell state.",
      },
    });
  }
}

function mapWorkspaceState(workspace: BoundWorkspaceState): WorkspaceState {
  return {
    rootPath: workspace.rootPath,
    mode: workspace.mode,
    lockMode: workspace.lockMode,
    health: workspace.health,
    stale: false,
    invalidReason: workspace.reason || undefined,
  };
}

function mapOverviewCards(overview: OverviewVM): OverviewCard[] {
  return [
    { label: "Workspace", value: overview.workspaceMode, secondary: overview.rootPath },
    { label: "Lock State", value: overview.lockMode, secondary: overview.health },
    { label: "Storage Footprint", value: `${overview.segmentCount} segments`, secondary: `Active segment ${overview.activeSegmentID}` },
    { label: "Replay Progress", value: `${overview.lastAckedWriteSeq}`, secondary: `Next write seq ${overview.nextWriteSeq}` },
    { label: "Checkpoint / Fsync", value: `${overview.checkpointsTotal} / ${overview.segmentFsyncTotal}`, secondary: `Retention ${overview.retentionDays} days` },
    { label: "Warnings", value: `${overview.warnings.length} warning(s)`, secondary: overview.warnings[0]?.message ?? "No warnings" },
  ];
}

function mapSegments(segments: PagedSegmentsVM): SegmentRow[] {
  return segments.items.map((segment) => ({
    segmentID: segment.segmentID,
    sizeBytes: `${segment.sizeBytes}`,
    sealed: segment.sealed,
    firstWriteSeq: segment.firstWriteSeq,
    lastWriteSeq: segment.lastWriteSeq,
    recordCount: segment.recordCount,
    minEventTime: formatTimestamp(segment.minEventTime),
    maxEventTime: formatTimestamp(segment.maxEventTime),
    lastBatchSeq: segment.lastBatchSeq,
    health: segment.health,
  }));
}

function mapConfigSections(config: ConfigVM): Record<string, ConfigRow[]> {
  return {
    Segment: [
      mapConfigField(config.segmentTargetSizeBytes),
      mapConfigField(config.segmentSlackSizeBytes),
    ],
    Block: [mapConfigField(config.blockTargetSizeBytes)],
    Checkpoint: [
      mapConfigField(config.checkpointInterval),
      mapConfigField(config.checkpointBytes),
    ],
    Fsync: [
      mapConfigField(config.segmentFsyncInterval),
      mapConfigField(config.segmentFsyncBytes),
    ],
    Retention: [mapConfigField(config.retentionDays)],
  };
}

function mapConfigField(field: ConfigVM[keyof ConfigVM]): ConfigRow {
  return {
    field: field.displayName,
    effective: field.currentValue,
    defaultValue: field.defaultValue,
    allowedRange: field.allowedRange,
    note: field.startupOnly ? "Startup-only" : "Mutable",
  };
}

function formatTimestamp(value: number) {
  if (!value) {
    return "N/A";
  }
  return new Date(value).toISOString().replace("T", " ").slice(0, 16);
}

function startOperation(operation: OperationKey, set: typeof useAppStore.setState, get: typeof useAppStore.getState) {
  if (hasBindings()) {
    void startBoundOperation(operation, set, get);
    return;
  }
  const { nextToastID } = get();
  const taskID = `ui-${Date.now()}`;
  set({
    selectedOperation: operation,
    currentTask: {
      taskID,
      kind: operation,
      status: "running",
      phase: "starting",
      message: "Preparing operation",
    },
  });

  window.setTimeout(() => {
    const finish = finishOperation(operation, nextToastID);
    set((state) => ({
      latestResult: finish.result,
      currentTask: {
        taskID,
        kind: operation,
        status: finish.status,
        phase: "finished",
        message: finish.result.summary,
      },
      toasts: [...state.toasts, finish.toast],
      nextToastID: state.nextToastID + 1,
    }));
  }, 40);
}

async function startBoundOperation(operation: OperationKey, set: typeof useAppStore.setState, get: typeof useAppStore.getState) {
  try {
    switch (operation) {
      case "verify": {
        const task = await bindings.runVerify();
        set({ currentTask: mapTask(task as Record<string, unknown>) });
        return;
      }
      case "close-check": {
        const task = await bindings.runCloseCheck();
        set({ currentTask: mapTask(task as Record<string, unknown>) });
        return;
      }
      case "repair-tail": {
        const segmentID = get().selectedSegment?.segmentID ?? 0;
        const task = await bindings.runRepairTail(segmentID);
        set({ currentTask: mapTask(task as Record<string, unknown>) });
        return;
      }
      case "shutdown": {
        const task = await bindings.runShutdown();
        set({ currentTask: mapTask(task as Record<string, unknown>) });
        return;
      }
    }
  } catch {
    set({
      currentTask: {
        taskID: `ui-${Date.now()}`,
        kind: operation,
        status: "failed",
        phase: "finished",
        message: "Backend operation failed to start.",
      },
    });
  }
}

function finishOperation(
  operation: OperationKey,
  toastID: number,
): { status: "succeeded" | "failed"; result: OperationResultState; toast: ToastState } {
  switch (operation) {
    case "verify":
      return {
        status: "succeeded",
        result: {
          summary: "Repairable corruption detected during verify.",
          changed: false,
          repairableSegment: 48,
          details: [
            { key: "segment", value: "48" },
            { key: "recommendation", value: "Open Repair" },
          ],
        },
        toast: {
          id: toastID,
          level: "warning",
          title: "Verify completed",
          message: "Repairable corruption found in segment 48.",
          durationLabel: "6s",
        },
      };
    case "close-check":
      return {
        status: "succeeded",
        result: {
          summary: "Close-check completed successfully.",
          changed: false,
          details: [{ key: "lifecycle", value: "clean" }],
        },
        toast: {
          id: toastID,
          level: "success",
          title: "Close-check completed",
          message: "Lifecycle state is clean.",
          durationLabel: "4s",
        },
      };
    case "repair-tail":
      return {
        status: "succeeded",
        result: {
          summary: "Repair-tail finished for segment 48.",
          changed: true,
          details: [{ key: "segment", value: "48" }],
        },
        toast: {
          id: toastID,
          level: "success",
          title: "Repair-tail completed",
          message: "Segment 48 tail was repaired.",
          durationLabel: "4s",
        },
      };
    case "shutdown":
      return {
        status: "failed",
        result: {
          summary: "Shutdown requires explicit handoff and remains blocked in this demo shell.",
          changed: false,
          details: [{ key: "state", value: "blocked" }],
        },
        toast: {
          id: toastID,
          level: "error",
          title: "Shutdown blocked",
          message: "Manual acknowledgement is still required.",
          durationLabel: "persistent",
        },
      };
  }
}

function mapTask(task: Record<string, unknown>): TaskState {
  return {
    taskID: `${task.taskID ?? task.TaskID ?? ""}`,
    kind: `${task.kind ?? task.Kind ?? "verify"}` as OperationKey,
    status: `${task.status ?? task.Status ?? "running"}` as TaskState["status"],
    phase: `${task.phase ?? task.Phase ?? ""}`,
    message: `${task.message ?? task.Message ?? ""}`,
  };
}

function mapTaskResult(task: Record<string, unknown>): OperationResultState | null {
  const raw = (task.result ?? task.Result) as Record<string, unknown> | null | undefined;
  if (!raw) {
    return null;
  }
  const details = Array.isArray(raw.details ?? raw.Details)
    ? ((raw.details ?? raw.Details) as Array<Record<string, unknown>>).map((detail) => ({
        key: `${detail.key ?? detail.Key ?? ""}`,
        value: `${detail.value ?? detail.Value ?? ""}`,
      }))
    : [];
  return {
    summary: `${raw.summary ?? raw.Summary ?? ""}`,
    changed: Boolean(raw.changed ?? raw.Changed),
    details,
  };
}

function makeToastFromTask(task: TaskState, result: OperationResultState | null, id: number): ToastState {
  const level =
    task.status === "failed" ? "error" :
    result?.changed ? "success" :
    "info";
  return {
    id,
    level,
    title: `${task.kind} ${task.status}`,
    message: result?.summary || task.message || "Task update received.",
    durationLabel: level === "error" ? "persistent" : "4s",
  };
}

function pickPayload(payload: unknown) {
  if (Array.isArray(payload)) {
    return payload[0];
  }
  return payload;
}
