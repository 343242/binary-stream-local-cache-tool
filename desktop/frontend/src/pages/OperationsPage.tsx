import ConfirmDialog from "../components/ConfirmDialog";
import TaskPanel from "../components/TaskPanel";
import ToastRegion from "../components/ToastRegion";
import styles from "../styles/shell.module.css";
import type {
  ConfirmDialogState,
  OperationKey,
  OperationResultState,
  TaskState,
  ToastState,
  WorkspaceState,
} from "../state/app-store";

const operations: { key: OperationKey; title: string; description: string; maintenanceRequired: boolean }[] = [
  { key: "verify", title: "Verify", description: "Run read-only verification and surface repairable corruption.", maintenanceRequired: false },
  { key: "close-check", title: "Close-check", description: "Inspect lifecycle state before restart or maintenance.", maintenanceRequired: false },
  { key: "repair-tail", title: "Repair-tail", description: "Repair a damaged segment tail with explicit confirmation.", maintenanceRequired: true },
  { key: "shutdown", title: "Shutdown", description: "Perform a guarded shutdown action.", maintenanceRequired: true },
];

type OperationsPageProps = {
  workspace: WorkspaceState | null;
  selectedSegmentID: number | null;
  selectedOperation: OperationKey;
  latestResult: OperationResultState | null;
  currentTask: TaskState | null;
  toasts: ToastState[];
  confirmDialog: ConfirmDialogState | null;
  onSelectOperation: (operation: OperationKey) => void;
  onRunOperation: (operation: OperationKey) => void;
  onConfirmDialog: () => void;
  onDismissDialog: () => void;
  onOpenRepair: () => void;
  onCancelTask: () => void;
  onDismissToast: (id: number) => void;
};

export default function OperationsPage({
  workspace,
  selectedSegmentID,
  selectedOperation,
  latestResult,
  currentTask,
  toasts,
  confirmDialog,
  onSelectOperation,
  onRunOperation,
  onConfirmDialog,
  onDismissDialog,
  onOpenRepair,
  onCancelTask,
  onDismissToast,
}: OperationsPageProps) {
  const operation = operations.find((item) => item.key === selectedOperation) ?? operations[0];
  const unhealthyWorkspace =
    !workspace ||
    workspace.mode === "DegradedReadOnly" ||
    workspace.mode === "InvalidWorkspace";
  const needsMaintenance = operation.maintenanceRequired && workspace?.mode !== "HealthyMaintenance";
  const needsSegment = operation.key === "repair-tail" && !selectedSegmentID;
  const staleSnapshot = operation.maintenanceRequired && Boolean(workspace?.stale);
  const blocked = unhealthyWorkspace || needsMaintenance || needsSegment || staleSnapshot;
  const preconditions = [
    {
      label: "workspace posture",
      value: workspace?.mode === "HealthyMaintenance" ? "maintenance ready" : workspace?.mode ?? "no workspace",
      met: workspace?.mode === "HealthyMaintenance" || !operation.maintenanceRequired,
    },
    {
      label: "lock posture",
      value: workspace?.lockMode ?? "N/A",
      met: workspace?.lockMode === "MaintenanceExclusive" || !operation.maintenanceRequired,
    },
    {
      label: "snapshot",
      value: workspace?.stale ? "stale snapshot" : "fresh snapshot",
      met: !workspace?.stale,
    },
    {
      label: "repair target",
      value: selectedSegmentID ? `segment ${selectedSegmentID}` : "select a segment",
      met: operation.key !== "repair-tail" || Boolean(selectedSegmentID),
    },
  ];
  const impactLines =
    operation.key === "repair-tail"
      ? [
          "May truncate a damaged tail region to restore read consistency.",
          "Emits an auditable operation result and task lifecycle trail.",
          selectedSegmentID ? `Selected segment ${selectedSegmentID} will be passed to the backend operation.` : "A repair candidate must be selected before the confirm gate can open.",
        ]
      : operation.key === "shutdown"
        ? [
            "Requests a guarded stop through the backend shutdown flow.",
            "Emits an auditable operation result and task lifecycle trail.",
            "Requires maintenance posture before the confirm gate can open.",
          ]
        : [
            "Keeps the workspace read-only throughout the request.",
            "Emits an auditable operation result and task lifecycle trail.",
            "Leaves the latest result surface separate from the running task timeline.",
          ];
  const destructive = operation.maintenanceRequired;
  const actionLabel = destructive ? "Review impact" : `Run ${operation.title}`;

  return (
    <div className={styles.pageStack}>
      <div className={styles.operationsLayout}>
        <section className={`${styles.panel} ${styles.panelPadding} ${styles.operationRail}`}>
          <p className={styles.eyebrow}>Action desk</p>
          <div className={styles.pageStack}>
            <div className={styles.pageStack}>
              <span className={styles.summaryLabel}>Read-only actions</span>
              {operations.filter((item) => !item.maintenanceRequired).map((item) => (
                <button
                  key={item.key}
                  className={`${styles.operationCard} ${selectedOperation === item.key ? styles.operationCardActive : ""}`}
                  onClick={() => onSelectOperation(item.key)}
                  type="button"
                >
                  <strong>{item.title}</strong>
                  <span>{item.description}</span>
                </button>
              ))}
            </div>
            <div className={styles.pageStack}>
              <span className={styles.summaryLabel}>Destructive maintenance actions</span>
              {operations.filter((item) => item.maintenanceRequired).map((item) => (
                <button
                  key={item.key}
                  className={`${styles.operationCard} ${selectedOperation === item.key ? styles.operationCardActive : ""}`}
                  onClick={() => onSelectOperation(item.key)}
                  type="button"
                >
                  <strong>{item.title}</strong>
                  <span>{item.description}</span>
                </button>
              ))}
            </div>
          </div>
        </section>

        <section className={styles.pageStack}>
          <section className={`${styles.panel} ${styles.panelPadding} ${destructive ? styles.operationHeroDanger : styles.operationHero}`}>
            <p className={styles.eyebrow}>{destructive ? "Destructive maintenance action" : "Read-only inspection action"}</p>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>{operation.title}</h3>
              {operation.maintenanceRequired ? <span className={`${styles.badge} ${styles.dangerBadge}`}>danger</span> : null}
            </div>
            <p className={styles.emptyCopy}>{operation.description}</p>
            <div className={styles.preconditionList}>
              {preconditions.map((item) => (
                <div key={item.label} className={`${styles.preconditionChip} ${item.met ? styles.preconditionChipMet : styles.preconditionChipBlocked}`}>
                  <span className={styles.fieldKey}>{item.label}</span>
                  <strong>{item.value}</strong>
                </div>
              ))}
            </div>
            <div className={styles.operationImpactBlock}>
              <h4 className={styles.operationSubheading}>Impact preview</h4>
              <ul className={styles.dialogList}>
                {impactLines.map((line) => (
                  <li key={line}>{line}</li>
                ))}
              </ul>
            </div>
            {blocked ? (
              <div className={styles.operationBlockedPanel}>
                <h4 className={styles.operationSubheading}>Blocked before confirmation</h4>
                <p className={styles.emptyCopy}>
                  {unhealthyWorkspace
                    ? "This action is blocked until a healthy workspace is open and readable in the shell."
                    : staleSnapshot
                      ? "Refresh the workspace before the confirm gate can open. Destructive actions require a fresh snapshot."
                      : needsSegment
                    ? "Select a repair candidate from Explorer or from the latest verify result before the confirm gate can open."
                    : "This action is blocked until the workspace holds MaintenanceExclusive and reaches maintenance posture."}
                </p>
              </div>
            ) : (
              <div className={styles.operationActionRow}>
                <button className={`${styles.primaryButton} ${styles.focusable}`} onClick={() => onRunOperation(operation.key)} type="button">
                  {actionLabel}
                </button>
              </div>
            )}
          </section>

          <TaskPanel task={currentTask} onCancel={onCancelTask} />

          <section className={`${styles.panel} ${styles.panelPadding}`}>
            <p className={styles.eyebrow}>Latest result</p>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>Latest result</h3>
            </div>
            {!latestResult ? (
              <p className={styles.emptyCopy}>No operation result has been recorded yet.</p>
            ) : (
              <div className={styles.fieldList}>
                <p className={styles.emptyCopy}>{latestResult.summary}</p>
                {latestResult.details.map((detail) => (
                  <div key={detail.key} className={styles.fieldRow}>
                    <span className={styles.fieldKey}>{detail.key}</span>
                    <span>{detail.value}</span>
                  </div>
                ))}
                {latestResult.repairableSegment ? (
                  <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onOpenRepair} type="button">
                    Open Repair
                  </button>
                ) : null}
              </div>
            )}
          </section>
        </section>
      </div>

      <ConfirmDialog dialog={confirmDialog} onConfirm={onConfirmDialog} onCancel={onDismissDialog} />
      <ToastRegion toasts={toasts} onDismiss={onDismissToast} />
    </div>
  );
}
