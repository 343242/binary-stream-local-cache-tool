import styles from "../styles/shell.module.css";

type EmptyStateProps = {
  title: string;
  message: string;
  eyebrow?: string;
  actionLabel?: string;
  onAction?: () => void;
};

export default function EmptyState({ title, message, eyebrow = "Inspection note", actionLabel, onAction }: EmptyStateProps) {
  return (
    <section className={`${styles.panel} ${styles.panelPadding} ${styles.emptyState}`}>
      <p className={styles.eyebrow}>{eyebrow}</p>
      <h3 className={styles.emptyTitle}>{title}</h3>
      <p className={styles.emptyCopy}>{message}</p>
      {actionLabel ? (
        <button className={`${styles.primaryButton} ${styles.focusable}`} onClick={onAction} type="button">
          {actionLabel}
        </button>
      ) : null}
    </section>
  );
}
