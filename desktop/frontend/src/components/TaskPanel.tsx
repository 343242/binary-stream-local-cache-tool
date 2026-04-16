import styles from "../styles/shell.module.css";
import type { TaskState } from "../state/app-store";

type TaskPanelProps = {
  task: TaskState | null;
};

export default function TaskPanel({ task }: TaskPanelProps) {
  return (
    <section className={`${styles.panel} ${styles.panelPadding}`}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>Task Panel</h3>
      </div>
      {!task ? (
        <p className={styles.emptyCopy}>No Operations Run Yet</p>
      ) : (
        <div className={styles.fieldList}>
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
        </div>
      )}
    </section>
  );
}
