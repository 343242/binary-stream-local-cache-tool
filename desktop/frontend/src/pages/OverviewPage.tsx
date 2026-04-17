import EmptyState from "../components/EmptyState";
import StatusCard from "../components/StatusCard";
import styles from "../styles/shell.module.css";
import type { CursorRow, OverviewCard, SegmentRow } from "../state/app-store";

type OverviewPageProps = {
  cards: OverviewCard[];
  warnings: string[];
  segments: SegmentRow[];
  cursors: CursorRow[];
};

export default function OverviewPage({ cards, warnings, segments, cursors }: OverviewPageProps) {
  return (
    <div className={styles.pageStack}>
      <section className={`${styles.panel} ${styles.hero}`}>
        <p className={styles.eyebrow}>Overview</p>
        <h2 className={styles.heroTitle}>Readable enough for routine checks. Severe enough for maintenance windows.</h2>
        <p className={styles.heroCopy}>Phase-1 desktop console for cache inspection, replay diagnostics, and guarded operations.</p>
      </section>

      <section className={styles.cardGrid}>
        {cards.map((card) => (
          <StatusCard key={card.label} label={card.label} value={card.value} secondary={card.secondary} />
        ))}
      </section>

      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>Maintenance windows</p>
            <h3 className={styles.sectionTitle}>Warnings and operator cautions</h3>
          </div>
        </div>
        <p className={styles.emptyCopy}>Use this region to spot the items that would block a routine inspection from becoming a repair or shutdown handoff.</p>
        {warnings.length ? (
          <div className={styles.fieldList}>
            {warnings.map((warning) => (
              <div key={warning} className={styles.badge}>
                {warning}
              </div>
            ))}
          </div>
        ) : (
          <EmptyState
            eyebrow="Maintenance windows"
            variant="inline"
            title="No active warnings"
            message="The current workspace is readable without escalations or pending repair cues."
          />
        )}
      </section>

      <section className={styles.pageStack}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>Recent activity</p>
            <h3 className={styles.sectionTitle}>Segments and cursor handoff</h3>
          </div>
        </div>
        <p className={styles.emptyCopy}>Track the latest write surfaces and downstream consumers before moving into deeper explorer or operations work.</p>
        <div className={styles.cardGrid}>
          <div className={`${styles.panel} ${styles.panelPadding}`}>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>Recent Segments</h3>
            </div>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Segment</th>
                  <th>Records</th>
                  <th>Health</th>
                </tr>
              </thead>
              <tbody>
                {segments.slice(0, 8).map((segment) => (
                  <tr key={segment.segmentID}>
                    <td>{segment.segmentID}</td>
                    <td>{segment.recordCount}</td>
                    <td>{segment.health}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={`${styles.panel} ${styles.panelPadding}`}>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>Recent Cursors</h3>
            </div>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Destination</th>
                  <th>Write Seq</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {cursors.slice(0, 8).map((cursor) => (
                  <tr key={cursor.destination}>
                    <td>{cursor.destination}</td>
                    <td>{cursor.writeSeq}</td>
                    <td>{cursor.status}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </div>
  );
}
