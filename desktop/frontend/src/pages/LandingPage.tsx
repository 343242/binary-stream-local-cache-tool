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
        title="Not a Cache Workspace"
        message={`${invalidWorkspace.path} does not contain the expected cache layout. ${invalidWorkspace.reason}`}
        actionLabel="Choose Another Directory"
        onAction={() => onOpenWorkspace(recentWorkspaces[0])}
      />
    );
  }

  return (
    <section className={`${styles.panel} ${styles.hero}`}>
      <p className={styles.cardLabel}>Landing</p>
      <h2 className={styles.heroTitle}>Open one cache root and inspect it safely.</h2>
      <p className={styles.heroCopy}>
        This phase-1 desktop console opens a single local workspace, defaults to observer mode, and requires explicit
        lock transitions before maintenance actions.
      </p>
      <div>
        <button className={`${styles.primaryButton} ${styles.focusable}`} onClick={() => onOpenWorkspace()} type="button">
          Open Cache Directory
        </button>
      </div>
      <section className={styles.pageStack}>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>Recent Directories</h3>
        </div>
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
