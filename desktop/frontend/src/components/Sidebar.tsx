import styles from "../styles/shell.module.css";
import type { PageKey, WorkspaceState } from "../state/app-store";

const items: { key: PageKey; label: string }[] = [
  { key: "overview", label: "Overview" },
  { key: "explorer", label: "Explorer" },
  { key: "config", label: "Config" },
  { key: "operations", label: "Operations" },
];

type SidebarProps = {
  activePage: PageKey;
  workspace: WorkspaceState | null;
  onNavigate: (page: PageKey) => void;
};

export default function Sidebar({ activePage, workspace, onNavigate }: SidebarProps) {
  return (
    <aside aria-label="Primary workspace" className={styles.sidebar}>
      <div className={styles.brandBlock}>
        <p className={styles.eyebrow}>Binary Stream Cache Tool</p>
        <h1 className={styles.brandName}>Observer-first desktop console</h1>
      </div>
      <section className={styles.workspaceRailSummary}>
        <span className={styles.summaryLabel}>Workspace status</span>
        <strong>{workspace?.stale ? "stale snapshot" : "fresh snapshot"}</strong>
      </section>
      <nav aria-label="Primary" className={styles.navList}>
        {items.map((item) => (
          <button
            key={item.key}
            className={`${styles.navButton} ${activePage === item.key ? styles.navButtonActive : ""}`}
            onClick={() => onNavigate(item.key)}
            type="button"
          >
            {item.label}
          </button>
        ))}
      </nav>
    </aside>
  );
}
