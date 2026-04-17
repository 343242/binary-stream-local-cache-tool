import DetailPane from "../components/DetailPane";
import EmptyState from "../components/EmptyState";
import styles from "../styles/shell.module.css";
import LoadingSkeleton from "../components/LoadingSkeleton";
import type { CheckpointDetailVM, CursorDetailVM, SegmentDetailVM, WALDetailVM } from "../bindings";
import type { CursorRow, ExplorerTab, SegmentRow } from "../state/app-store";

type ExplorerPageProps = {
  activeTab: ExplorerTab;
  onTabChange: (tab: ExplorerTab) => void;
  segments: SegmentRow[];
  selectedSegment: SegmentRow | null;
  cursors: CursorRow[];
  selectedCursor: CursorRow | null;
  segmentDetail: SegmentDetailVM | null;
  walDetail: WALDetailVM | null;
  cursorDetail: CursorDetailVM | null;
  checkpointDetail: CheckpointDetailVM | null;
  detailLoading: boolean;
  detailError: string | null;
  onSelectSegment: (segment: SegmentRow) => void;
  onSelectCursor: (cursor: CursorRow) => void;
};

const tabs: { key: ExplorerTab; label: string }[] = [
  { key: "segments", label: "Segments" },
  { key: "wal", label: "WAL" },
  { key: "cursors", label: "Cursors" },
  { key: "checkpoint", label: "Checkpoint" },
];

export default function ExplorerPage({
  activeTab,
  onTabChange,
  segments,
  selectedSegment,
  cursors,
  selectedCursor,
  segmentDetail,
  walDetail,
  cursorDetail,
  checkpointDetail,
  detailLoading,
  detailError,
  onSelectSegment,
  onSelectCursor,
}: ExplorerPageProps) {
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
                  <tr
                    key={segment.segmentID}
                    className={selectedSegment?.segmentID === segment.segmentID ? styles.tableRowSelected : ""}
                    onClick={() => onSelectSegment(segment)}
                  >
                    <td>{segment.segmentID}</td>
                    <td>{segment.sizeBytes}</td>
                    <td>{segment.lastWriteSeq}</td>
                    <td>{segment.recordCount}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : activeTab === "wal" ? (
            <EmptyState
              title={walDetail ? "WAL Detail Available" : "No WAL Present"}
              message={
                walDetail
                  ? "The active WAL snapshot is loaded in the detail pane."
                  : "The active WAL file is not present in this demo workspace."
              }
            />
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
                    <tr
                      key={cursor.destination}
                      className={selectedCursor?.destination === cursor.destination ? styles.tableRowSelected : ""}
                      onClick={() => onSelectCursor(cursor)}
                    >
                      <td>{cursor.destination}</td>
                      <td>{cursor.writeSeq}</td>
                      <td>{cursor.status}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )
          ) : (
            <EmptyState
              title={checkpointDetail ? "Checkpoint Detail Available" : "No Checkpoint Written Yet"}
              message={
                checkpointDetail
                  ? "Checkpoint metadata is loaded in the detail pane."
                  : "Checkpoint metadata has not been created yet."
              }
            />
          )}
        </section>

        <DetailPane title="Detail Pane">
          {detailLoading ? <LoadingSkeleton rows={5} /> : null}
          {!detailLoading && detailError ? <p className={styles.emptyCopy}>{detailError}</p> : null}
          {!detailLoading && !detailError && activeTab === "segments" && !segmentDetail ? (
            <p className={styles.emptyCopy}>Select a row to inspect detailed fields and raw preview data.</p>
          ) : null}
          {!detailLoading && !detailError && activeTab === "segments" && segmentDetail ? (
            <div className={styles.fieldList}>
              <ExplorerField label="Segment ID" value={`${segmentDetail.segmentID}`} />
              <ExplorerField label="Path" value={segmentDetail.path} />
              <ExplorerField label="Footer Health" value={segmentDetail.footerStatus} />
              <ExplorerField label="Tail Status" value={segmentDetail.tailStatus} />
              <ExplorerField label="Block Count" value={`${segmentDetail.blockCount}`} />
              <ExplorerField label="Write Seq Range" value={`${segmentDetail.firstWriteSeq} - ${segmentDetail.lastWriteSeq}`} />
              <ExplorerField label="Event Time Range" value={`${formatTimestamp(segmentDetail.minEventTime)} - ${formatTimestamp(segmentDetail.maxEventTime)}`} />
              <ExplorerField label="Raw Preview" value={segmentDetail.rawPreviewHex || "Preview capped to first 64 KiB"} />
            </div>
          ) : null}
          {!detailLoading && !detailError && activeTab === "wal" && walDetail ? (
            <div className={styles.fieldList}>
              <ExplorerField label="Path" value={walDetail.path} />
              <ExplorerField label="Size" value={`${walDetail.sizeBytes}`} />
              <ExplorerField label="Batch Seq Range" value={`${walDetail.firstBatchSeq} - ${walDetail.lastBatchSeq}`} />
              <ExplorerField label="Last End Offset" value={`${walDetail.lastEndOffset}`} />
              <ExplorerField label="Health" value={walDetail.health} />
            </div>
          ) : null}
          {!detailLoading && !detailError && activeTab === "cursors" && !cursorDetail ? (
            <p className={styles.emptyCopy}>Select a cursor row to inspect offsets, CRC status, and backup state.</p>
          ) : null}
          {!detailLoading && !detailError && activeTab === "cursors" && cursorDetail ? (
            <div className={styles.fieldList}>
              <ExplorerField label="Destination" value={cursorDetail.destination} />
              <ExplorerField label="Write Seq" value={`${cursorDetail.writeSeq}`} />
              <ExplorerField label="Segment ID" value={`${cursorDetail.segmentID}`} />
              <ExplorerField label="Block Offset" value={`${cursorDetail.blockOffset}`} />
              <ExplorerField label="Record Index" value={`${cursorDetail.recordIndex}`} />
              <ExplorerField label="Updated At" value={formatTimestamp(cursorDetail.updatedAt)} />
              <ExplorerField label="CRC Status" value={cursorDetail.crcStatus} />
              <ExplorerField label="Backup Status" value={cursorDetail.backupStatus} />
            </div>
          ) : null}
          {!detailLoading && !detailError && activeTab === "checkpoint" && checkpointDetail ? (
            <div className={styles.fieldList}>
              <ExplorerField label="Last Batch Seq" value={`${checkpointDetail.lastBatchSeq}`} />
              <ExplorerField label="Last WAL End Offset" value={`${checkpointDetail.lastWALEndOffset}`} />
              <ExplorerField label="Updated At" value={formatTimestamp(checkpointDetail.updatedAt)} />
              <ExplorerField label="Version" value={`${checkpointDetail.version}`} />
              <ExplorerField label="Integrity" value={checkpointDetail.integrityStatus} />
            </div>
          ) : null}
        </DetailPane>
      </div>
    </div>
  );
}

function ExplorerField({ label, value }: { label: string; value: string }) {
  return (
    <div className={styles.fieldRow}>
      <span className={styles.fieldKey}>{label}</span>
      <span>{value}</span>
    </div>
  );
}

function formatTimestamp(value: number) {
  if (!value) {
    return "N/A";
  }
  return new Date(value).toISOString().replace("T", " ").slice(0, 16);
}
