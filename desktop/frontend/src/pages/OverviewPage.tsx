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
      <section className={styles.cardGrid}>
        {cards.map((card) => (
          <StatusCard key={card.label} label={card.label} value={card.value} secondary={card.secondary} />
        ))}
      </section>

      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>Warnings</h3>
        </div>
        <div className={styles.fieldList}>
          {warnings.map((warning) => (
            <div key={warning} className={styles.badge}>
              {warning}
            </div>
          ))}
        </div>
      </section>

      <section className={styles.cardGrid}>
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
      </section>
    </div>
  );
}
