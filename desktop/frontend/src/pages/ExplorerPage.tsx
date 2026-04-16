import DetailPane from "../components/DetailPane";
import EmptyState from "../components/EmptyState";
import styles from "../styles/shell.module.css";
import type { CursorRow, ExplorerTab, SegmentRow } from "../state/app-store";

type ExplorerPageProps = {
  activeTab: ExplorerTab;
  onTabChange: (tab: ExplorerTab) => void;
  segments: SegmentRow[];
  selectedSegment: SegmentRow | null;
  cursors: CursorRow[];
};

const tabs: { key: ExplorerTab; label: string }[] = [
  { key: "segments", label: "Segments" },
  { key: "wal", label: "WAL" },
  { key: "cursors", label: "Cursors" },
  { key: "checkpoint", label: "Checkpoint" },
];

export default function ExplorerPage({ activeTab, onTabChange, segments, selectedSegment, cursors }: ExplorerPageProps) {
  return (
    <div className={styles.pageStack}>
      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <div className={styles.tabRow}>
            {tabs.map((tab) => (
              <button
                key={tab.key}
                className={`${styles.tabButton} ${activeTab === tab.key ? styles.tabButtonActive : ""}`}
                onClick={() => onTabChange(tab.key)}
                type="button"
              >
                {tab.label}
              </button>
            ))}
          </div>
          <div className={styles.badgeRow}>
            <span className={styles.badge}>Manual refresh</span>
            <span className={styles.badge}>Split 45 / 55</span>
          </div>
        </div>
      </section>

      <div className={styles.explorerLayout}>
        <section className={`${styles.panel} ${styles.panelPadding}`}>
          {activeTab === "segments" && segments.length === 0 ? (
            <EmptyState title="No Segments Found" message="This workspace does not currently contain any segment rows." />
          ) : activeTab === "segments" ? (
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Segment</th>
                  <th>Size</th>
                  <th>Last Write Seq</th>
                  <th>Records</th>
                </tr>
              </thead>
              <tbody>
                {segments.map((segment) => (
                  <tr key={segment.segmentID}>
                    <td>{segment.segmentID}</td>
                    <td>{segment.sizeBytes}</td>
                    <td>{segment.lastWriteSeq}</td>
                    <td>{segment.recordCount}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : activeTab === "wal" ? (
            <EmptyState title="No WAL Present" message="The active WAL file is not present in this demo workspace." />
          ) : activeTab === "cursors" ? (
            cursors.length === 0 ? (
              <EmptyState title="No Replay Cursors" message="No replay cursor files were found." />
            ) : (
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th>Destination</th>
                    <th>Write Seq</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {cursors.map((cursor) => (
                    <tr key={cursor.destination}>
                      <td>{cursor.destination}</td>
                      <td>{cursor.writeSeq}</td>
                      <td>{cursor.status}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )
          ) : (
            <EmptyState title="No Checkpoint Written Yet" message="Checkpoint metadata has not been created yet." />
          )}
        </section>

        <DetailPane title="Detail Pane">
          {activeTab !== "segments" || !selectedSegment ? (
            <p className={styles.emptyCopy}>Select a row to inspect detailed fields and raw preview data.</p>
          ) : (
            <div className={styles.fieldList}>
              <div className={styles.fieldRow}>
                <span className={styles.fieldKey}>Segment ID</span>
                <span>{selectedSegment.segmentID}</span>
              </div>
              <div className={styles.fieldRow}>
                <span className={styles.fieldKey}>Health</span>
                <span>{selectedSegment.health}</span>
              </div>
              <div className={styles.fieldRow}>
                <span className={styles.fieldKey}>Write Seq Range</span>
                <span>
                  {selectedSegment.firstWriteSeq} - {selectedSegment.lastWriteSeq}
                </span>
              </div>
              <div className={styles.fieldRow}>
                <span className={styles.fieldKey}>Event Time Range</span>
                <span>
                  {selectedSegment.minEventTime} - {selectedSegment.maxEventTime}
                </span>
              </div>
              <div className={styles.fieldRow}>
                <span className={styles.fieldKey}>Raw Preview</span>
                <span>Preview capped to first 64 KiB</span>
              </div>
            </div>
          )}
        </DetailPane>
      </div>
    </div>
  );
}
