import EmptyState from "../components/EmptyState";
import { getMessages } from "../i18n";
import { useAppStore } from "../state/app-store";
import styles from "../styles/shell.module.css";

type LandingPageProps = {
  recentWorkspaces: string[];
  invalidWorkspace?: { path: string; reason: string };
  onOpenWorkspace: (path?: string) => void;
};

export default function LandingPage({ recentWorkspaces, invalidWorkspace, onOpenWorkspace }: LandingPageProps) {
  const locale = useAppStore((state) => state.locale);
  const m = getMessages(locale);
  if (invalidWorkspace) {
    return (
      <EmptyState
        eyebrow={m.landing.eyebrow}
        title={m.landing.invalidTitle}
        message={`${invalidWorkspace.path} does not contain the expected cache layout. ${invalidWorkspace.reason}`}
        actionLabel={m.common.chooseAnotherDirectory}
        onAction={() => onOpenWorkspace()}
      />
    );
  }

  return (
    <section className={`${styles.panel} ${styles.hero}`}>
      <p className={styles.eyebrow}>{m.landing.eyebrow}</p>
      <h2 className={styles.heroTitle}>{m.landing.title}</h2>
      <p className={styles.heroCopy}>{m.landing.copy}</p>
      <div>
        <button className={`${styles.primaryButton} ${styles.focusable}`} onClick={() => onOpenWorkspace()} type="button">
          {m.common.openCacheDirectory}
        </button>
      </div>
      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.pageStack}>
          <p className={styles.eyebrow}>{m.landing.supportedWorkspaceEyebrow}</p>
          <h3 className={styles.sectionTitle}>{m.landing.supportedWorkspaceTitle}</h3>
          <p className={styles.emptyCopy}>{m.landing.supportedWorkspaceCopy}</p>
        </div>
      </section>
      <section className={styles.pageStack}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.landing.recentEyebrow}</p>
            <h3 className={styles.sectionTitle}>{m.landing.recentTitle}</h3>
          </div>
        </div>
        <p className={styles.emptyCopy}>{m.landing.recentCopy}</p>
        <div className={styles.recentList}>
          {recentWorkspaces.slice(0, 5).map((workspace) => (
            <div key={workspace} className={styles.recentItem}>
              <span>{workspace}</span>
              <button
                className={`${styles.secondaryButton} ${styles.focusable}`}
                onClick={() => onOpenWorkspace(workspace)}
                type="button"
              >
                {m.common.reopen}
              </button>
            </div>
          ))}
        </div>
      </section>
    </section>
  );
}
