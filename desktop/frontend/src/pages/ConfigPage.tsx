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

  return (
    <div className={styles.pageStack}>
      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>Effective Configuration</h3>
        </div>
        <p className={styles.readonlyNote}>Read-only inspection. Persistent configuration editing is deferred.</p>
      </section>

      {Object.entries(sections).map(([section, rows]) => (
        <section key={section} className={`${styles.panel} ${styles.panelPadding} ${styles.configSection}`}>
          <div className={styles.sectionHeader}>
            <h3 className={styles.sectionTitle}>{section}</h3>
          </div>
          <div className={styles.configTableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Field</th>
                  <th>Effective Value</th>
                  <th>Default Value</th>
                  <th>Allowed Range</th>
                  <th>Note</th>
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
