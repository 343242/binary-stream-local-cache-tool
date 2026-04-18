import styles from "../styles/shell.module.css";
import { getMessages } from "../i18n";
import { useAppStore } from "../state/app-store";

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
  eyebrow,
  variant = "panel",
  actionLabel,
  onAction,
}: EmptyStateProps) {
  const locale = useAppStore((state) => state.locale);
  const m = getMessages(locale);
  const containerClassName =
    variant === "panel"
      ? `${styles.panel} ${styles.panelPadding} ${styles.emptyState}`
      : styles.emptyState;

  return (
    <section className={containerClassName}>
      <p className={styles.eyebrow}>{eyebrow ?? m.emptyState.inspectionNote}</p>
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
