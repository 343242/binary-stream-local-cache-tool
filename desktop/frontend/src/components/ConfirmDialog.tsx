import styles from "../styles/shell.module.css";
import { getMessages } from "../i18n";
import { useAppStore } from "../state/app-store";
import type { ConfirmDialogState } from "../state/app-store";

type ConfirmDialogProps = {
  dialog: ConfirmDialogState | null;
  onConfirm: () => void;
  onCancel: () => void;
};

export default function ConfirmDialog({ dialog, onConfirm, onCancel }: ConfirmDialogProps) {
  const locale = useAppStore((state) => state.locale);
  const m = getMessages(locale);
  if (!dialog) {
    return null;
  }

  return (
    <div className={styles.dialogBackdrop} role="presentation">
      <div className={styles.dialog} aria-label={`${m.dialog.impactReview}: ${dialog.title}`} aria-modal="true" role="dialog">
        <p className={styles.eyebrow}>{m.dialog.impactReview}</p>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>{dialog.title}</h3>
          <span className={`${styles.badge} ${dialog.riskLevel === "danger" ? styles.dangerBadge : ""}`}>
            {dialog.riskLevel === "danger" ? m.dialog.danger : m.dialog.accent}
          </span>
        </div>
        <p className={styles.emptyCopy}>{dialog.summary}</p>
        <ul className={styles.dialogList}>
          {dialog.impactLines.map((line) => (
            <li key={line}>{line}</li>
          ))}
        </ul>
        <p className={styles.dialogNote}>{m.dialog.reviewNote}</p>
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
