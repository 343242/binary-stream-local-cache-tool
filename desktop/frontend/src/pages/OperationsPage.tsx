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
  const blocked =
    (operation.maintenanceRequired && workspace?.mode !== "HealthyMaintenance") ||
    (operation.key === "repair-tail" && !selectedSegmentID);

  return (
    <div className={styles.pageStack}>
      <div className={styles.operationsLayout}>
        <section className={styles.pageStack}>
          {operations.map((item) => (
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
        </section>

        <section className={styles.pageStack}>
          <section className={`${styles.panel} ${styles.panelPadding}`}>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>{operation.title}</h3>
              {operation.maintenanceRequired ? <span className={`${styles.badge} ${styles.dangerBadge}`}>danger</span> : null}
            </div>
            <p className={styles.emptyCopy}>{operation.description}</p>
            {blocked ? (
              <p className={styles.emptyCopy}>
                {operation.key === "repair-tail" && !selectedSegmentID
                  ? "Select a segment from Explorer or use Open Repair from a verify result before running repair-tail."
                  : "This action is blocked until the workspace holds MaintenanceExclusive."}
              </p>
            ) : (
              <button className={`${styles.primaryButton} ${styles.focusable}`} onClick={() => onRunOperation(operation.key)} type="button">
                Run {operation.title}
              </button>
            )}
          </section>

          <TaskPanel task={currentTask} onCancel={onCancelTask} />

          <section className={`${styles.panel} ${styles.panelPadding}`}>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>Latest Result</h3>
            </div>
            {!latestResult ? (
              <p className={styles.emptyCopy}>No Operations Run Yet</p>
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
