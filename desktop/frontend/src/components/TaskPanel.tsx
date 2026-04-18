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
        <p className={styles.eyebrow}>Task timeline</p>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>Task timeline</h3>
        </div>
        <p className={styles.emptyCopy}>No task has been requested from this desk yet.</p>
      </section>
    );
  }

  const progressLabel =
    task.progressCurrent !== null && task.progressTotal !== null
      ? `${task.progressCurrent} / ${task.progressTotal}`
      : null;

  const timelineItems = [
    {
      label: "Requested",
      active: true,
      detail: task.startedAt === "N/A" ? "Awaiting task start time" : task.startedAt,
    },
    {
      label: "In progress",
      active: task.status === "running" || task.status === "succeeded" || task.status === "failed" || task.status === "cancelled",
      detail: task.phase || task.message || "Waiting for phase update",
    },
    {
      label: task.status === "failed" ? "Failed" : task.status === "cancelled" ? "Cancelled" : "Completed",
      active: task.status !== "running",
      detail: task.updatedAt === "N/A" ? task.message : task.updatedAt,
    },
  ];

  return (
    <section className={`${styles.panel} ${styles.panelPadding}`}>
      <p className={styles.eyebrow}>Task timeline</p>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>Task timeline</h3>
      </div>
      <ol className={styles.timelineList}>
        {timelineItems.map((item) => (
          <li
            key={item.label}
            className={`${styles.timelineItem} ${item.active ? styles.timelineItemActive : ""}`}
          >
            <div>
              <strong>{item.label}</strong>
              <p className={styles.emptyCopy}>{item.detail}</p>
            </div>
          </li>
        ))}
      </ol>
      <div className={styles.fieldList}>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Task</span>
          <span>{task.kind}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>Lifecycle</span>
          <span>{task.phase || "pending"} {"->"} {task.status}</span>
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
