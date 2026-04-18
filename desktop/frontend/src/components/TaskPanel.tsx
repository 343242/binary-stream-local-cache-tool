import styles from "../styles/shell.module.css";
import { getMessages } from "../i18n";
import { useAppStore } from "../state/app-store";
import type { TaskState } from "../state/app-store";

type TaskPanelProps = {
  task: TaskState | null;
  onCancel: () => void;
};

export default function TaskPanel({ task, onCancel }: TaskPanelProps) {
  const locale = useAppStore((state) => state.locale);
  const m = getMessages(locale);
  if (!task) {
    return (
      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <p className={styles.eyebrow}>{m.task.eyebrow}</p>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>{m.task.title}</h3>
        </div>
        <p className={styles.emptyCopy}>{m.task.empty}</p>
      </section>
    );
  }

  const progressLabel =
    task.progressCurrent !== null && task.progressTotal !== null
      ? `${task.progressCurrent} / ${task.progressTotal}`
      : null;

  const timelineItems = [
    {
      label: m.task.requested,
      active: true,
      detail: task.startedAt === "N/A" ? m.task.awaitingStart : task.startedAt,
    },
    {
      label: m.task.inProgress,
      active: task.status === "running" || task.status === "succeeded" || task.status === "failed" || task.status === "cancelled",
      detail: task.phase || task.message || m.task.waitingPhase,
    },
    {
      label: task.status === "failed" ? m.task.failed : task.status === "cancelled" ? m.task.cancelled : m.task.completed,
      active: task.status !== "running",
      detail: task.updatedAt === "N/A" ? task.message : task.updatedAt,
    },
  ];

  return (
    <section className={`${styles.panel} ${styles.panelPadding}`}>
      <p className={styles.eyebrow}>{m.task.eyebrow}</p>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>{m.task.title}</h3>
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
          <span className={styles.fieldKey}>{m.task.task}</span>
          <span>{task.kind}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>{m.task.lifecycle}</span>
          <span>{task.phase || m.task.pending} {"->"} {task.status}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>{m.task.status}</span>
          <span>{task.status}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>{m.task.phase}</span>
          <span>{task.phase}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>{m.task.message}</span>
          <span>{task.message}</span>
        </div>
        {task.target ? (
          <div className={styles.fieldRow}>
            <span className={styles.fieldKey}>{m.task.target}</span>
            <span>{task.target}</span>
          </div>
        ) : null}
        {progressLabel ? (
          <div className={styles.fieldRow}>
            <span className={styles.fieldKey}>{m.task.progress}</span>
            <span>{progressLabel}</span>
          </div>
        ) : null}
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>{m.task.started}</span>
          <span>{task.startedAt}</span>
        </div>
        <div className={styles.fieldRow}>
          <span className={styles.fieldKey}>{m.task.updated}</span>
          <span>{task.updatedAt}</span>
        </div>
        {task.error ? (
          <div className={styles.fieldRow}>
            <span className={styles.fieldKey}>{m.task.error}</span>
            <span>{task.error}</span>
          </div>
        ) : null}
        {task.canCancel && task.status === "running" ? (
          <div>
            <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onCancel} type="button">
              {m.task.cancelTask}
            </button>
          </div>
        ) : null}
      </div>
    </section>
  );
}
