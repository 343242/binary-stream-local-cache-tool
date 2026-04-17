import styles from "../styles/shell.module.css";

type EmptyStateProps = {
  title: string;
  message: string;
  eyebrow?: string;
  variant?: "panel" | "inline";
  actionLabel?: string;
  onAction?: () => void;
};

export default function EmptyState({
  title,
  message,
  eyebrow = "Inspection note",
  variant = "panel",
  actionLabel,
  onAction,
}: EmptyStateProps) {
  const containerClassName =
    variant === "panel"
      ? `${styles.panel} ${styles.panelPadding} ${styles.emptyState}`
      : styles.emptyState;

  return (
    <section className={containerClassName}>
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
