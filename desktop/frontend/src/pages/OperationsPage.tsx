import ConfirmDialog from "../components/ConfirmDialog";
import TaskPanel from "../components/TaskPanel";
import { formatMessage, getMessages, localizeLockMode, localizeWorkspaceMode } from "../i18n";
import { useAppStore } from "../state/app-store";
import styles from "../styles/shell.module.css";
import type {
  ConfirmDialogState,
  OperationKey,
  OperationResultState,
  TaskState,
  WorkspaceState,
} from "../state/app-store";

type OperationsPageProps = {
  workspace: WorkspaceState | null;
  selectedSegmentID: number | null;
  selectedOperation: OperationKey;
  latestResult: OperationResultState | null;
  currentTask: TaskState | null;
  confirmDialog: ConfirmDialogState | null;
  onSelectOperation: (operation: OperationKey) => void;
  onRunOperation: (operation: OperationKey) => void;
  onConfirmDialog: () => void;
  onDismissDialog: () => void;
  onOpenRepair: () => void;
  onCancelTask: () => void;
};

export default function OperationsPage({
  workspace,
  selectedSegmentID,
  selectedOperation,
  latestResult,
  currentTask,
  confirmDialog,
  onSelectOperation,
  onRunOperation,
  onConfirmDialog,
  onDismissDialog,
  onOpenRepair,
  onCancelTask,
}: OperationsPageProps) {
  const locale = useAppStore((state) => state.locale);
  const m = getMessages(locale);
  const operations: { key: OperationKey; title: string; description: string; maintenanceRequired: boolean }[] = [
    { key: "verify", title: m.operations.operationTitles.verify, description: m.operations.operationDescriptions.verify, maintenanceRequired: false },
    { key: "close-check", title: m.operations.operationTitles.closeCheck, description: m.operations.operationDescriptions.closeCheck, maintenanceRequired: false },
    { key: "repair-tail", title: m.operations.operationTitles.repairTail, description: m.operations.operationDescriptions.repairTail, maintenanceRequired: true },
    { key: "shutdown", title: m.operations.operationTitles.shutdown, description: m.operations.operationDescriptions.shutdown, maintenanceRequired: true },
  ];
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
      label: m.operations.preconditions.workspacePosture,
      value: workspace?.mode === "HealthyMaintenance" ? m.operations.preconditions.maintenanceReady : localizeWorkspaceMode(locale, workspace?.mode ?? "NoWorkspace"),
      met: workspace?.mode === "HealthyMaintenance" || !operation.maintenanceRequired,
    },
    {
      label: m.operations.preconditions.lockPosture,
      value: localizeLockMode(locale, workspace?.lockMode ?? m.common.na),
      met: workspace?.lockMode === "MaintenanceExclusive" || !operation.maintenanceRequired,
    },
    {
      label: m.operations.preconditions.snapshot,
      value: workspace?.stale ? (locale === "zh-CN" ? "陈旧快照" : "stale snapshot") : (locale === "zh-CN" ? "新鲜快照" : "fresh snapshot"),
      met: !workspace?.stale,
    },
    {
      label: m.operations.preconditions.repairTarget,
      value: selectedSegmentID ? formatMessage(locale === "zh-CN" ? "段文件 {segmentID}" : "segment {segmentID}", { segmentID: selectedSegmentID }) : m.operations.preconditions.selectSegment,
      met: operation.key !== "repair-tail" || Boolean(selectedSegmentID),
    },
  ];
  const impactLines =
    operation.key === "repair-tail"
      ? [
          m.operations.impactLines.repairTail1,
          m.operations.impactLines.repairTail2,
          selectedSegmentID ? formatMessage(m.operations.impactLines.repairTailWithSegment, { segmentID: selectedSegmentID }) : m.operations.impactLines.repairTailWithoutSegment,
        ]
      : operation.key === "shutdown"
        ? [
            m.operations.impactLines.shutdown1,
            m.operations.impactLines.shutdown2,
            m.operations.impactLines.shutdown3,
          ]
        : [
            m.operations.impactLines.readonly1,
            m.operations.impactLines.readonly2,
            m.operations.impactLines.readonly3,
          ];
  const destructive = operation.maintenanceRequired;
  const actionLabel = destructive ? m.operations.reviewImpact : `${m.operations.runPrefix} ${operation.title}`;

  return (
    <div className={styles.pageStack}>
      <div className={styles.operationsLayout}>
        <section className={`${styles.panel} ${styles.panelPadding} ${styles.operationRail}`}>
          <p className={styles.eyebrow}>{m.operations.actionDesk}</p>
          <div className={styles.pageStack}>
            <div className={styles.pageStack}>
              <span className={styles.summaryLabel}>{m.operations.readonlyActions}</span>
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
              <span className={styles.summaryLabel}>{m.operations.destructiveActions}</span>
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
            <p className={styles.eyebrow}>{destructive ? m.operations.destructiveEyebrow : m.operations.readonlyEyebrow}</p>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>{operation.title}</h3>
              {operation.maintenanceRequired ? <span className={`${styles.badge} ${styles.dangerBadge}`}>{m.operations.danger}</span> : null}
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
              <h4 className={styles.operationSubheading}>{m.operations.impactPreview}</h4>
              <ul className={styles.dialogList}>
                {impactLines.map((line) => (
                  <li key={line}>{line}</li>
                ))}
              </ul>
            </div>
            {blocked ? (
              <div className={styles.operationBlockedPanel}>
                <h4 className={styles.operationSubheading}>{m.operations.blockedTitle}</h4>
                <p className={styles.emptyCopy}>
                  {unhealthyWorkspace
                    ? m.operations.blockedUnhealthy
                    : staleSnapshot
                      ? m.operations.blockedStale
                      : needsSegment
                    ? m.operations.blockedSegment
                    : m.operations.blockedMaintenance}
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
            <p className={styles.eyebrow}>{m.operations.latestResultEyebrow}</p>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>{m.operations.latestResultTitle}</h3>
            </div>
            {!latestResult ? (
              <p className={styles.emptyCopy}>{m.operations.noLatestResult}</p>
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
                    {m.operations.openRepair}
                  </button>
                ) : null}
              </div>
            )}
          </section>
        </section>
      </div>

      <ConfirmDialog dialog={confirmDialog} onConfirm={onConfirmDialog} onCancel={onDismissDialog} />
    </div>
  );
}
