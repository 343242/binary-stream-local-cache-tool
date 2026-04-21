import EmptyState from "../components/EmptyState";
import { defaultLocale, getMessages, type LocaleKey } from "../i18n";
import styles from "../styles/shell.module.css";
import type { WriterAlert, WriterEventVM, WriterStatusState, WorkspaceState } from "../state/app-store";

type HomePageProps = {
  workspace: WorkspaceState | null;
  writerStatus: WriterStatusState;
  writerAlerts: WriterAlert[];
  writerEvents: WriterEventVM[];
  onStartWriter: () => void;
  onStopWriter: () => void;
  onOpenWriterConfig: () => void;
  locale?: LocaleKey;
};

export default function HomePage({
  workspace,
  writerStatus,
  writerAlerts,
  writerEvents,
  onStartWriter,
  onStopWriter,
  onOpenWriterConfig,
  locale = defaultLocale,
}: HomePageProps) {
  const m = getMessages(locale);
  const lifecycleLabel = localizeWriterLifecycle(locale, writerStatus.lifecycleState);
  const canStart = Boolean(workspace?.rootPath) && !isWriterBusy(writerStatus.lifecycleState);
  const canStop = writerStatus.lifecycleState === "running" || writerStatus.lifecycleState === "starting";

  if (!workspace?.rootPath) {
    return <EmptyState title={m.home.emptyTitle} message={m.home.emptyCopy} />;
  }

  return (
    <div className={styles.pageStack}>
      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.home.eyebrow}</p>
            <h2 className={styles.sectionTitle}>{m.home.title}</h2>
            <p className={styles.emptyCopy}>{m.home.copy}</p>
          </div>
          <div className={styles.badgeRow}>
            <span className={styles.badge}>{m.home.statusLabel}: {lifecycleLabel}</span>
            <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onOpenWriterConfig} type="button">
              {m.home.configAction}
            </button>
            {canStop ? (
              <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onStopWriter} type="button">
                {m.home.stopAction}
              </button>
            ) : (
              <button className={`${styles.primaryButton} ${styles.focusable}`} disabled={!canStart} onClick={onStartWriter} type="button">
                {m.home.startAction}
              </button>
            )}
          </div>
        </div>
      </section>

      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.home.alertsEyebrow}</p>
            <h3 className={styles.sectionTitle}>{m.home.alertsTitle}</h3>
          </div>
        </div>
        {writerAlerts.length === 0 ? (
          <p className={styles.emptyCopy}>{m.home.noAlerts}</p>
        ) : (
          <div className={styles.pageStack}>
            {writerAlerts.map((alert) => (
              <article
                key={alert.id}
                className={`${styles.operationBlockedPanel} ${alert.level === "error" ? styles.homeAlertError : alert.level === "warning" ? styles.homeAlertWarning : ""}`}
              >
                <strong>{alert.title}</strong>
                <p className={styles.emptyCopy}>{alert.message}</p>
              </article>
            ))}
          </div>
        )}
      </section>

      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.home.eventsEyebrow}</p>
            <h3 className={styles.sectionTitle}>{m.home.eventsTitle}</h3>
          </div>
        </div>
        {writerEvents.length === 0 ? (
          <p className={styles.emptyCopy}>{m.home.noEvents}</p>
        ) : (
          <ul className={styles.timelineList}>
            {writerEvents.slice().reverse().map((event) => (
              <li key={`${event.kind}-${event.timestampUnixMs}`} className={styles.timelineItem}>
                <strong>{event.message}</strong>
                <p className={styles.emptyCopy}>{event.timestampLabel} · {event.kind}</p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function localizeWriterLifecycle(locale: LocaleKey, lifecycleState: string) {
  const labels = getMessages(locale).home.lifecycle;
  return labels[lifecycleState as keyof typeof labels] ?? lifecycleState;
}

function isWriterBusy(lifecycleState: string) {
  return lifecycleState === "running" || lifecycleState === "starting" || lifecycleState === "stopping";
}
