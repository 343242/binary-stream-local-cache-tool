import EmptyState from "../components/EmptyState";
import styles from "../styles/shell.module.css";
import type { ConfigRow } from "../state/app-store";

type ConfigPageProps = {
  hasWorkspace: boolean;
  sections: Record<string, ConfigRow[]>;
};

export default function ConfigPage({ hasWorkspace, sections }: ConfigPageProps) {
  if (!hasWorkspace) {
    return (
      <EmptyState
        title="Open a workspace to inspect configuration"
        message="Phase-1 configuration is read-only and only becomes available after a valid workspace opens."
      />
    );
  }

  const sectionCount = Object.keys(sections).length;
  const fieldCount = Object.values(sections).reduce((count, rows) => count + rows.length, 0);

  return (
    <div className={styles.pageStack}>
      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>Configuration audit surface</p>
            <h3 className={styles.sectionTitle}>Configuration audit ledger</h3>
          </div>
          <div className={styles.badgeRow}>
            <span className={styles.badge}>{sectionCount} sections</span>
            <span className={styles.badge}>{fieldCount} captured fields</span>
          </div>
        </div>
        <p className={styles.readonlyNote}>Read-only inspection. Persistent configuration editing is deferred.</p>
        <p className={styles.emptyCopy}>
          Effective values, startup defaults, and allowed ranges are preserved here as an audit ledger for operator review.
        </p>
      </section>

      {Object.entries(sections).map(([section, rows]) => (
        <section key={section} className={`${styles.panel} ${styles.panelPadding} ${styles.configSection}`}>
          <div className={styles.sectionHeader}>
            <div className={styles.pageStack}>
              <p className={styles.eyebrow}>Audit section</p>
              <h3 className={styles.sectionTitle}>{section} ledger</h3>
            </div>
            <span className={styles.badge}>{rows.length} fields</span>
          </div>
          <p className={styles.emptyCopy}>Captured startup-only values for audit comparison and maintenance review.</p>
          <div className={styles.configTableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Setting</th>
                  <th>Effective Snapshot</th>
                  <th>Startup Default</th>
                  <th>Allowed Range</th>
                  <th>Audit Note</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.field}>
                    <td>{row.field}</td>
                    <td>{row.effective}</td>
                    <td>{row.defaultValue}</td>
                    <td>{row.allowedRange}</td>
                    <td>{row.note}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ))}
    </div>
  );
}
