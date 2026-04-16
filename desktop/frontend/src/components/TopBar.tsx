import styles from "../styles/shell.module.css";
import type { WorkspaceState } from "../state/app-store";

type TopBarProps = {
  workspace: WorkspaceState | null;
  onRefresh: () => void;
};

export default function TopBar({ workspace, onRefresh }: TopBarProps) {
  const title = workspace ? workspace.rootPath : "No workspace open";

  return (
    <header className={styles.topBar}>
      <div>
        <h2 className={styles.topBarTitle}>{title}</h2>
        <div className={styles.topBarMeta}>
          <span>Mode: {workspace?.mode ?? "NoWorkspace"}</span>
          <span>Lock: {workspace?.lockMode ?? "N/A"}</span>
          <span>Health: {workspace?.health ?? "N/A"}</span>
        </div>
      </div>
      <div className={styles.badgeRow}>
        <span className={`${styles.badge} ${workspace?.stale ? styles.staleBadge : ""}`}>
          {workspace?.stale ? "Data may be stale" : "Fresh snapshot"}
        </span>
        <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onRefresh} type="button">
          Refresh
        </button>
      </div>
    </header>
  );
}
