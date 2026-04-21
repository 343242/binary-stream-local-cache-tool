import styles from "../styles/shell.module.css";
import { getMessages, localizeWorkspaceMode, type LocaleKey } from "../i18n";
import type { PageKey, WorkspaceState } from "../state/app-store";

const items: { key: PageKey; label: string }[] = [
  { key: "home", label: "Home" },
  { key: "overview", label: "Overview" },
  { key: "explorer", label: "Explorer" },
  { key: "config", label: "Config" },
  { key: "operations", label: "Operations" },
];

type SidebarProps = {
  activePage: PageKey;
  locale: LocaleKey;
  workspace: WorkspaceState | null;
  onNavigate: (page: PageKey) => void;
};

export default function Sidebar({ activePage, locale, workspace, onNavigate }: SidebarProps) {
  const m = getMessages(locale);
  const hasWorkspace = Boolean(workspace);
  return (
    <aside aria-label={m.brand.workspace} className={styles.sidebar}>
      <div className={styles.brandBlock}>
        <p className={styles.eyebrow}>{m.brand.product}</p>
        <h1 className={styles.brandName}>{m.brand.console}</h1>
      </div>
      <section className={styles.workspaceRailSummary}>
        <span className={styles.summaryLabel}>{m.brand.workspaceStatus}</span>
        <strong>
          {!workspace
            ? localizeWorkspaceMode(locale, "NoWorkspace")
            : workspace.stale
              ? (locale === "zh-CN" ? "陈旧快照" : "stale snapshot")
              : locale === "zh-CN"
                ? "新鲜快照"
                : "fresh snapshot"}
        </strong>
      </section>
      <nav aria-label={m.brand.primaryNav} className={styles.navList}>
        {items.map((item) => {
          const isLocked = !hasWorkspace && item.key !== "home";

          return (
          <button
            key={item.key}
            aria-disabled={isLocked}
            className={`${styles.navButton} ${activePage === item.key ? styles.navButtonActive : ""} ${isLocked ? styles.navButtonLocked : ""}`}
            onClick={() => onNavigate(item.key)}
            type="button"
          >
            {item.key === "home"
              ? m.nav.home
              : item.key === "overview"
              ? m.nav.overview
              : item.key === "explorer"
                ? m.nav.explorer
                : item.key === "config"
                  ? m.nav.config
                  : m.nav.operations}
          </button>
          );
        })}
      </nav>
    </aside>
  );
}
