import styles from "../styles/shell.module.css";
import type { ToastState } from "../state/app-store";

type ToastRegionProps = {
  toasts: ToastState[];
  onDismiss: (id: number) => void;
};

export default function ToastRegion({ toasts, onDismiss }: ToastRegionProps) {
  if (toasts.length === 0) {
    return null;
  }

  return (
    <aside className={styles.toastRegion} aria-label="Notifications">
      {toasts.slice(-3).map((toast) => (
        <article key={toast.id} className={`${styles.toast} ${styles[`toast${capitalize(toast.level)}`]}`}>
          <div className={styles.sectionHeader}>
            <strong>{toast.title}</strong>
            <button className={styles.toastDismiss} onClick={() => onDismiss(toast.id)} type="button">
              Dismiss
            </button>
          </div>
          <p className={styles.emptyCopy}>{toast.message}</p>
          <p className={styles.cardLabel}>
            {toast.level} · {toast.durationLabel}
          </p>
        </article>
      ))}
    </aside>
  );
}

function capitalize(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
