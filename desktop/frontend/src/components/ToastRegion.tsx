import styles from "../styles/shell.module.css";
import { getMessages, type LocaleKey } from "../i18n";
import type { ToastState } from "../state/app-store";

type ToastRegionProps = {
  locale: LocaleKey;
  toasts: ToastState[];
  onDismiss: (id: number) => void;
};

export default function ToastRegion({ locale, toasts, onDismiss }: ToastRegionProps) {
  const m = getMessages(locale);
  if (toasts.length === 0) {
    return null;
  }

  return (
    <aside className={styles.toastRegion} aria-label={m.toast.ariaLabel}>
      {toasts.slice(-3).map((toast) => (
        <article
          key={toast.id}
          className={`${styles.toast} ${styles[`toast${capitalize(toast.level)}`]}`}
        >
          <div className={styles.sectionHeader}>
            <strong>{toast.title}</strong>
            <button className={styles.toastDismiss} onClick={() => onDismiss(toast.id)} type="button">
              {m.common.dismiss}
            </button>
          </div>
          <p className={styles.eyebrow}>{toast.level === "info" ? m.toast.deskNotice : toast.level === "success" ? m.toast.completedAction : m.toast.auditNotice}</p>
          <p className={styles.emptyCopy}>{toast.message}</p>
          <p className={styles.cardLabel}>
            {m.toast.levels[toast.level]} · {toast.durationLabel}
          </p>
        </article>
      ))}
    </aside>
  );
}

function capitalize(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
