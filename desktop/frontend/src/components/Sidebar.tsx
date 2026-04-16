import styles from "../styles/shell.module.css";
import type { PageKey } from "../state/app-store";

const items: { key: PageKey; label: string }[] = [
  { key: "overview", label: "Overview" },
  { key: "explorer", label: "Explorer" },
  { key: "config", label: "Config" },
  { key: "operations", label: "Operations" },
];

type SidebarProps = {
  activePage: PageKey;
  onNavigate: (page: PageKey) => void;
};

export default function Sidebar({ activePage, onNavigate }: SidebarProps) {
  return (
    <aside className={styles.sidebar}>
      <div className={styles.brand}>
        <h1 className={styles.brandName}>Binary Stream Cache Tool</h1>
        <p className={styles.brandMeta}>Observer-first desktop console</p>
      </div>
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
