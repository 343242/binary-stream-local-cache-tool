import EmptyState from "../components/EmptyState";
import styles from "../styles/shell.module.css";

type LandingPageProps = {
  recentWorkspaces: string[];
  invalidWorkspace?: { path: string; reason: string };
  onOpenWorkspace: (path?: string) => void;
};

export default function LandingPage({ recentWorkspaces, invalidWorkspace, onOpenWorkspace }: LandingPageProps) {
  if (invalidWorkspace) {
    return (
      <EmptyState
        eyebrow="Landing"
        title="Not a Cache Workspace"
        message={`${invalidWorkspace.path} does not contain the expected cache layout. ${invalidWorkspace.reason}`}
        actionLabel="Choose Another Directory"
        onAction={() => onOpenWorkspace()}
      />
    );
  }

  return (
    <section className={`${styles.panel} ${styles.hero}`}>
      <p className={styles.eyebrow}>Landing</p>
      <h2 className={styles.heroTitle}>Open one cache root and move into inspection with context already in frame.</h2>
      <p className={styles.heroCopy}>
        Phase-1 desktop console for cache inspection, replay diagnostics, and guarded operations. Start in observer mode,
        keep the shell readable for routine checks, and escalate deliberately only when the shell calls for operator handoff.
      </p>
      <div>
        <button className={`${styles.primaryButton} ${styles.focusable}`} onClick={() => onOpenWorkspace()} type="button">
          Open Cache Directory
        </button>
      </div>
      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.pageStack}>
          <p className={styles.eyebrow}>Supported workspace</p>
          <h3 className={styles.sectionTitle}>What this shell can open</h3>
          <p className={styles.emptyCopy}>
            Open a cache root that contains the expected segment, WAL, cursor, and checkpoint layout. Unsupported roots stay on the landing desk so you can choose another directory safely.
          </p>
        </div>
      </section>
      <section className={styles.pageStack}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>Recent workspaces</p>
            <h3 className={styles.sectionTitle}>Recent Directories</h3>
          </div>
        </div>
        <p className={styles.emptyCopy}>Resume the last operator roots inspected from this shell or open a new cache workspace.</p>
        <div className={styles.recentList}>
          {recentWorkspaces.slice(0, 5).map((workspace) => (
            <div key={workspace} className={styles.recentItem}>
              <span>{workspace}</span>
              <button
                className={`${styles.secondaryButton} ${styles.focusable}`}
                onClick={() => onOpenWorkspace(workspace)}
                type="button"
              >
                Reopen
              </button>
            </div>
          ))}
        </div>
      </section>
    </section>
  );
}
