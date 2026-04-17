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

export type KeyValueVM = {
  key: string;
  value: string;
};

export type SegmentDetailVM = {
  segmentID: number;
  path: string;
  sizeBytes: number;
  sealed: boolean;
  footerStatus: string;
  tailStatus: string;
  firstWriteSeq: number;
  lastWriteSeq: number;
  recordCount: number;
  blockCount: number;
  minEventTime: number;
  maxEventTime: number;
  lastBatchSeq: number;
  rawPreviewHex: string;
  structuredPreview: KeyValueVM[];
};

export type CursorSummaryVM = {
  destination: string;
  writeSeq: number;
  updatedAt: number;
  status: string;
};

export type CursorDetailVM = {
  destination: string;
  segmentID: number;
  blockOffset: number;
  recordIndex: number;
  writeSeq: number;
  updatedAt: number;
  crcStatus: string;
  backupStatus: string;
};

export type CheckpointDetailVM = {
  lastBatchSeq: number;
  lastWALEndOffset: number;
  updatedAt: number;
  version: number;
  integrityStatus: string;
};

export type WALDetailVM = {
  path: string;
  sizeBytes: number;
  firstBatchSeq: number;
  lastBatchSeq: number;
  lastEndOffset: number;
  health: string;
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

export type GUIErrorVM = {
  code: string;
  title: string;
  message: string;
  operation: string;
  path: string;
  recoverable: boolean;
  details: string;
  suggestedAction: string;
};

export type OperationResultVM = {
  summary: string;
  details: KeyValueVM[];
  changed: boolean;
};

export type TaskVM = {
  taskID: string;
  kind: string;
  target: string;
  status: string;
  phase: string;
  startedAt: number;
  updatedAt: number;
  progressCurrent?: number | null;
  progressTotal?: number | null;
  message: string;
  canCancel: boolean;
  result?: OperationResultVM | null;
  error?: GUIErrorVM | null;
};

type BackendBindings = {
  OpenWorkspace?: (rootPath: string) => Promise<WorkspaceState>;
  ChooseWorkspace?: () => Promise<WorkspaceState>;
  CloseWorkspace?: () => Promise<void>;
  GetWorkspaceState?: () => Promise<WorkspaceState>;
  GetRecentWorkspaces?: () => Promise<string[]>;
  GetOverview?: () => Promise<OverviewVM>;
  ListSegments?: (page: number, pageSize: number) => Promise<PagedSegmentsVM>;
  GetSegmentDetail?: (segmentID: number) => Promise<SegmentDetailVM>;
  GetWALDetail?: () => Promise<WALDetailVM>;
  ListCursors?: () => Promise<CursorSummaryVM[]>;
  GetCursorDetail?: (destination: string) => Promise<CursorDetailVM>;
  GetCheckpointDetail?: () => Promise<CheckpointDetailVM>;
  GetConfig?: () => Promise<ConfigVM>;
  RunVerify?: () => Promise<TaskVM>;
  RunCloseCheck?: () => Promise<TaskVM>;
  RunRepairTail?: (segmentID: number) => Promise<TaskVM>;
  RunShutdown?: () => Promise<TaskVM>;
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
  getSegmentDetail(segmentID: number) {
    return window.go?.backend?.App?.GetSegmentDetail?.(segmentID) ?? missingBinding("GetSegmentDetail");
  },
  getWALDetail() {
    return window.go?.backend?.App?.GetWALDetail?.() ?? missingBinding("GetWALDetail");
  },
  listCursors() {
    return window.go?.backend?.App?.ListCursors?.() ?? missingBinding("ListCursors");
  },
  getCursorDetail(destination: string) {
    return window.go?.backend?.App?.GetCursorDetail?.(destination) ?? missingBinding("GetCursorDetail");
  },
  getCheckpointDetail() {
    return window.go?.backend?.App?.GetCheckpointDetail?.() ?? missingBinding("GetCheckpointDetail");
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
