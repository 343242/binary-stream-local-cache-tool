import styles from "../styles/shell.module.css";
import type { ConfirmDialogState } from "../state/app-store";

type ConfirmDialogProps = {
  dialog: ConfirmDialogState | null;
  onConfirm: () => void;
  onCancel: () => void;
};

export default function ConfirmDialog({ dialog, onConfirm, onCancel }: ConfirmDialogProps) {
  if (!dialog) {
    return null;
  }

  return (
    <div className={styles.dialogBackdrop} role="presentation">
      <div className={styles.dialog} aria-modal="true" role="dialog">
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>{dialog.title}</h3>
          <span className={`${styles.badge} ${dialog.riskLevel === "danger" ? styles.dangerBadge : ""}`}>
            {dialog.riskLevel}
          </span>
        </div>
        <p className={styles.emptyCopy}>{dialog.summary}</p>
        <ul className={styles.dialogList}>
          {dialog.impactLines.map((line) => (
            <li key={line}>{line}</li>
          ))}
        </ul>
        <div className={styles.dialogActions}>
          <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onCancel} type="button">
            {dialog.cancelLabel}
          </button>
          <button className={`${styles.primaryButton} ${styles.focusable}`} onClick={onConfirm} type="button">
            {dialog.confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
