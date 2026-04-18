import EmptyState from "../components/EmptyState";
import { getMessages } from "../i18n";
import { useAppStore } from "../state/app-store";
import styles from "../styles/shell.module.css";
import type { ConfigRow } from "../state/app-store";

type ConfigPageProps = {
  hasWorkspace: boolean;
  sections: Record<string, ConfigRow[]>;
};

export default function ConfigPage({ hasWorkspace, sections }: ConfigPageProps) {
  const locale = useAppStore((state) => state.locale);
  const m = getMessages(locale);
  if (!hasWorkspace) {
    return (
      <EmptyState
        title={m.config.emptyTitle}
        message={m.config.emptyCopy}
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
            <p className={styles.eyebrow}>{m.config.eyebrow}</p>
            <h3 className={styles.sectionTitle}>{m.config.title}</h3>
          </div>
          <div className={styles.badgeRow}>
            <span className={styles.badge}>{sectionCount} {m.config.sectionsCaptured}</span>
            <span className={styles.badge}>{fieldCount} {m.config.fieldsCaptured}</span>
          </div>
        </div>
        <p className={styles.readonlyNote}>{m.config.readonly}</p>
        <p className={styles.emptyCopy}>{m.config.copy}</p>
      </section>

      {Object.entries(sections).map(([section, rows]) => (
        <section key={section} className={`${styles.panel} ${styles.panelPadding} ${styles.configSection}`}>
          <div className={styles.sectionHeader}>
            <div className={styles.pageStack}>
              <p className={styles.eyebrow}>{m.config.sectionEyebrow}</p>
              <h3 className={styles.sectionTitle}>{section} {m.config.sectionLedgerSuffix}</h3>
            </div>
            <span className={styles.badge}>{rows.length} {m.config.fieldsCaptured}</span>
          </div>
          <p className={styles.emptyCopy}>{m.config.sectionCopy}</p>
          <div className={styles.configTableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>{m.config.table.setting}</th>
                  <th>{m.config.table.effective}</th>
                  <th>{m.config.table.startupDefault}</th>
                  <th>{m.config.table.allowedRange}</th>
                  <th>{m.config.table.note}</th>
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
