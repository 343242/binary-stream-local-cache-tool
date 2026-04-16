export type WorkspaceState = {
  rootPath: string;
  mode: string;
  lockMode: string;
  health: string;
  canRefresh: boolean;
  canRunVerify: boolean;
  canRunCloseCheck: boolean;
  canRunRepairTail: boolean;
  canRunShutdown: boolean;
  reason: string;
};

export type WarningVM = {
  code: string;
  severity: string;
  title: string;
  message: string;
};

export type OverviewVM = {
  rootPath: string;
  workspaceMode: string;
  lockMode: string;
  health: string;
  segmentCount: number;
  activeSegmentID: number;
  activeSegmentSizeBytes: number;
  walSizeBytes: number;
  nextWriteSeq: number;
  retentionDays: number;
  checkpointsTotal: number;
  segmentFsyncTotal: number;
  lastAckedWriteSeq: number;
  gracefulShutdownsTotal: number;
  ungracefulRecoveriesTotal: number;
  segmentTailRepairsTotal: number;
  backlogEstimateRecords?: number | null;
  backlogEstimateBytes?: number | null;
  warnings: WarningVM[];
  lastRefreshedAt: number;
  isStale: boolean;
};

export type SegmentListItemVM = {
  segmentID: number;
  fileName: string;
  sizeBytes: number;
  sealed: boolean;
  firstWriteSeq: number;
  lastWriteSeq: number;
  recordCount: number;
  minEventTime: number;
  maxEventTime: number;
  lastBatchSeq: number;
  health: string;
};

export type PagedSegmentsVM = {
  items: SegmentListItemVM[];
  page: number;
  pageSize: number;
  totalItems: number;
  hasNext: boolean;
};

export type CursorSummaryVM = {
  destination: string;
  writeSeq: number;
  updatedAt: number;
  status: string;
};

export type ConfigFieldVM = {
  name: string;
  displayName: string;
  currentValue: string;
  defaultValue: string;
  allowedRange: string;
  startupOnly: boolean;
  description: string;
};

export type ConfigVM = {
  rootDir: ConfigFieldVM;
  segmentTargetSizeBytes: ConfigFieldVM;
  segmentSlackSizeBytes: ConfigFieldVM;
  blockTargetSizeBytes: ConfigFieldVM;
  checkpointInterval: ConfigFieldVM;
  checkpointBytes: ConfigFieldVM;
  segmentFsyncInterval: ConfigFieldVM;
  segmentFsyncBytes: ConfigFieldVM;
  retentionDays: ConfigFieldVM;
};

type BackendBindings = {
  OpenWorkspace?: (rootPath: string) => Promise<WorkspaceState>;
  ChooseWorkspace?: () => Promise<WorkspaceState>;
  CloseWorkspace?: () => Promise<void>;
  GetWorkspaceState?: () => Promise<WorkspaceState>;
  GetRecentWorkspaces?: () => Promise<string[]>;
  GetOverview?: () => Promise<OverviewVM>;
  ListSegments?: (page: number, pageSize: number) => Promise<PagedSegmentsVM>;
  ListCursors?: () => Promise<CursorSummaryVM[]>;
  GetConfig?: () => Promise<ConfigVM>;
  RunVerify?: () => Promise<any>;
  RunCloseCheck?: () => Promise<any>;
  RunRepairTail?: (segmentID: number) => Promise<any>;
  RunShutdown?: () => Promise<any>;
  CancelTask?: (taskID: string) => Promise<void>;
};

declare global {
  interface Window {
    go?: {
      backend?: {
        App?: BackendBindings;
      };
    };
  }
}

function missingBinding(name: string): never {
  throw new Error(`${name} binding is not available yet`);
}

export function hasBindings() {
  return Boolean(window.go?.backend?.App?.OpenWorkspace);
}

export const bindings = {
  openWorkspace(rootPath: string) {
    return window.go?.backend?.App?.OpenWorkspace?.(rootPath) ?? missingBinding("OpenWorkspace");
  },
  chooseWorkspace() {
    return window.go?.backend?.App?.ChooseWorkspace?.() ?? missingBinding("ChooseWorkspace");
  },
  closeWorkspace() {
    return window.go?.backend?.App?.CloseWorkspace?.() ?? missingBinding("CloseWorkspace");
  },
  getWorkspaceState() {
    return window.go?.backend?.App?.GetWorkspaceState?.() ?? missingBinding("GetWorkspaceState");
  },
  getRecentWorkspaces() {
    return window.go?.backend?.App?.GetRecentWorkspaces?.() ?? missingBinding("GetRecentWorkspaces");
  },
  getOverview() {
    return window.go?.backend?.App?.GetOverview?.() ?? missingBinding("GetOverview");
  },
  listSegments(page: number, pageSize: number) {
    return window.go?.backend?.App?.ListSegments?.(page, pageSize) ?? missingBinding("ListSegments");
  },
  listCursors() {
    return window.go?.backend?.App?.ListCursors?.() ?? missingBinding("ListCursors");
  },
  getConfig() {
    return window.go?.backend?.App?.GetConfig?.() ?? missingBinding("GetConfig");
  },
  runVerify() {
    return window.go?.backend?.App?.RunVerify?.() ?? missingBinding("RunVerify");
  },
  runCloseCheck() {
    return window.go?.backend?.App?.RunCloseCheck?.() ?? missingBinding("RunCloseCheck");
  },
  runRepairTail(segmentID: number) {
    return window.go?.backend?.App?.RunRepairTail?.(segmentID) ?? missingBinding("RunRepairTail");
  },
  runShutdown() {
    return window.go?.backend?.App?.RunShutdown?.() ?? missingBinding("RunShutdown");
  },
  cancelTask(taskID: string) {
    return window.go?.backend?.App?.CancelTask?.(taskID) ?? missingBinding("CancelTask");
  },
};
