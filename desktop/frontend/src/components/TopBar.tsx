import styles from "../styles/shell.module.css";
import { getMessages, localizeHealth, localizeLockMode, localizeWorkspaceMode, type LocaleKey } from "../i18n";
import type { WorkspaceState, WorkspaceLoadState, WriterAlert, WriterStatusState } from "../state/app-store";

type TopBarProps = {
  locale: LocaleKey;
  workspace: WorkspaceState | null;
  writerStatus: WriterStatusState;
  writerAlerts: WriterAlert[];
  workspaceLoadState: WorkspaceLoadState;
  onRefresh: () => void;
  onToggleLocale: () => void;
};

export default function TopBar({ locale, workspace, writerStatus, writerAlerts, workspaceLoadState, onRefresh, onToggleLocale }: TopBarProps) {
  const m = getMessages(locale);
  const title = workspace ? workspace.rootPath : m.topBar.noWorkspaceOpen;
  const isRefreshing = workspaceLoadState === "refreshing";
  const isHydrating = workspaceLoadState === "hydrating" || workspaceLoadState === "choosing";
  const latestAlert = writerAlerts[0];

  return (
    <header className={styles.topBar}>
      <div className={styles.topBarIdentity}>
        <p className={styles.eyebrow}>{m.topBar.currentSnapshot}</p>
        <h2 className={styles.topBarTitle}>{title}</h2>
        <div className={styles.topBarMeta}>
          <span>{m.topBar.mode}: {localizeWorkspaceMode(locale, workspace?.mode ?? "NoWorkspace")}</span>
          <span>{m.topBar.lock}: {localizeLockMode(locale, workspace?.lockMode ?? m.common.na)}</span>
          <span>{m.topBar.health}: {localizeHealth(locale, workspace?.health ?? m.common.na)}</span>
        </div>
      </div>
      <div className={styles.badgeRow}>
        {isHydrating ? <span className={styles.badge}>{m.topBar.openingWorkspace}</span> : null}
        {isRefreshing ? <span className={styles.badge}>{m.topBar.refreshInProgress}</span> : null}
        <span className={styles.badge}>{m.topBar.writer}: {localizeLifecycle(locale, writerStatus.lifecycleState)}</span>
        {latestAlert ? <span className={styles.badge}>{latestAlert.title}</span> : null}
        <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onToggleLocale} type="button">
          {m.brand.localeSwitch}
        </button>
        <button className={`${styles.secondaryButton} ${styles.focusable}`} disabled={isRefreshing} onClick={onRefresh} type="button">
          {isRefreshing ? m.common.refreshing : m.common.refresh}
        </button>
      </div>
    </header>
  );
}

function localizeLifecycle(locale: LocaleKey, lifecycleState: string) {
  const labels = getMessages(locale).home.lifecycle;
  return labels[lifecycleState as keyof typeof labels] ?? lifecycleState;
}
