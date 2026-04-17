import styles from "../styles/shell.module.css";
import type { WorkspaceState } from "../state/app-store";
import type { WorkspaceLoadState } from "../state/app-store";

type TopBarProps = {
  workspace: WorkspaceState | null;
  workspaceLoadState: WorkspaceLoadState;
  onRefresh: () => void;
};

export default function TopBar({ workspace, workspaceLoadState, onRefresh }: TopBarProps) {
  const title = workspace ? workspace.rootPath : "No workspace open";
  const isRefreshing = workspaceLoadState === "refreshing";
  const isHydrating = workspaceLoadState === "hydrating" || workspaceLoadState === "choosing";

  return (
    <header className={styles.topBar}>
      <div>
        <h2 className={styles.topBarTitle}>{title}</h2>
        <div className={styles.topBarMeta}>
          <span>Mode: {workspace?.mode ?? "NoWorkspace"}</span>
          <span>Lock: {workspace?.lockMode ?? "N/A"}</span>
          <span>Health: {workspace?.health ?? "N/A"}</span>
          {isHydrating ? <span>Opening workspace</span> : null}
          {isRefreshing ? <span>Refresh in progress</span> : null}
        </div>
      </div>
      <div className={styles.badgeRow}>
        <span className={`${styles.badge} ${workspace?.stale ? styles.staleBadge : ""}`}>
          {workspace?.stale ? "Data may be stale" : "Fresh snapshot"}
        </span>
        <button className={`${styles.secondaryButton} ${styles.focusable}`} disabled={isRefreshing} onClick={onRefresh} type="button">
          {isRefreshing ? "Refreshing..." : "Refresh"}
        </button>
      </div>
    </header>
  );
}
