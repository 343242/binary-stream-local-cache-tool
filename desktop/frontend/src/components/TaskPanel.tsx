import styles from "../styles/shell.module.css";
import type { TaskState } from "../state/app-store";

type TaskPanelProps = {
  task: TaskState | null;
  onCancel: () => void;
};

export default function TaskPanel({ task, onCancel }: TaskPanelProps) {
  if (!task) {
    return (
      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>Task Panel</h3>
        </div>
        <p className={styles.emptyCopy}>No Operations Run Yet</p>
      </section>
    );
  }

  const progressLabel =
    task.progressCurrent !== null && task.progressTotal !== null
      ? `${task.progressCurrent} / ${task.progressTotal}`
      : null;

  return (
    <section className={`${styles.panel} ${styles.panelPadding}`}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>Task Panel</h3>
      </div>
      <div className={styles.fieldList}>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Task timeline</span>
          <span>{task.phase || "pending"} {"->"} {task.status}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Task</span>
          <span>{task.kind}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Status</span>
          <span>{task.status}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Phase</span>
          <span>{task.phase}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Message</span>
          <span>{task.message}</span>
        </div>
        {task.target ? (
          <div className={styles.fieldRow}>
            <span className={styles.fieldKey}>Target</span>
            <span>{task.target}</span>
          </div>
        ) : null}
        {progressLabel ? (
          <div className={styles.fieldRow}>
            <span className={styles.fieldKey}>Progress</span>
            <span>{progressLabel}</span>
          </div>
        ) : null}
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Started</span>
          <span>{task.startedAt}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Updated</span>
          <span>{task.updatedAt}</span>
        </div>
        {task.error ? (
          <div className={styles.fieldRow}>
            <span className={styles.fieldKey}>Error</span>
            <span>{task.error}</span>
          </div>
        ) : null}
        {task.canCancel && task.status === "running" ? (
          <div>
            <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onCancel} type="button">
              Cancel Task
            </button>
          </div>
        ) : null}
      </div>
    </section>
  );
}
