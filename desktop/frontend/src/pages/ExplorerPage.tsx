import DetailPane from "../components/DetailPane";
import EmptyState from "../components/EmptyState";
import { getMessages, localizeHealth } from "../i18n";
import { useAppStore } from "../state/app-store";
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
  const locale = useAppStore((state) => state.locale);
  const m = getMessages(locale);
  const tabs: { key: ExplorerTab; label: string }[] = [
    { key: "segments", label: m.explorer.tabs.segments },
    { key: "wal", label: m.explorer.tabs.wal },
    { key: "cursors", label: m.explorer.tabs.cursors },
    { key: "checkpoint", label: m.explorer.tabs.checkpoint },
  ];
  return (
    <div className={styles.explorerLayout}>
      <section className={`${styles.panel} ${styles.panelPadding} ${styles.pageStack}`}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.explorer.eyebrow}</p>
            <h3 className={styles.sectionTitle}>{m.explorer.title}</h3>
            <p className={styles.emptyCopy}>{m.explorer.copy}</p>
          </div>
          <div className={styles.badgeRow}>
            <span className={styles.badge}>{m.common.manualRefresh}</span>
            <span className={styles.badge}>{m.common.readonlyRows}</span>
          </div>
        </div>
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
        {activeTab === "segments" && segments.length === 0 ? (
          <EmptyState title={m.explorer.empty.noSegmentsTitle} message={m.explorer.empty.noSegmentsCopy} />
        ) : activeTab === "segments" ? (
          <table className={styles.table}>
            <thead>
              <tr>
                <th>{m.explorer.tables.segment}</th>
                <th>{m.explorer.tables.size}</th>
                <th>{m.explorer.tables.lastWriteSeq}</th>
                <th>{m.explorer.tables.records}</th>
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
            title={walDetail ? m.explorer.empty.walAvailableTitle : m.explorer.empty.noWalTitle}
            message={
              walDetail
                ? m.explorer.empty.walAvailableCopy
                : m.explorer.empty.noWalCopy
            }
          />
        ) : activeTab === "cursors" ? (
          cursors.length === 0 ? (
            <EmptyState title={m.explorer.empty.noCursorsTitle} message={m.explorer.empty.noCursorsCopy} />
          ) : (
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>{m.explorer.tables.destination}</th>
                  <th>{m.explorer.tables.writeSeq}</th>
                  <th>{m.explorer.tables.status}</th>
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
                    <td>{localizeHealth(locale, cursor.status)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )
        ) : (
          <EmptyState
            title={checkpointDetail ? m.explorer.empty.checkpointAvailableTitle : m.explorer.empty.noCheckpointTitle}
            message={
              checkpointDetail
                ? m.explorer.empty.checkpointAvailableCopy
                : m.explorer.empty.noCheckpointCopy
            }
          />
        )}

      </section>

      <DetailPane eyebrow={m.explorer.detailEyebrow} note={detailPaneNote(locale, activeTab)} title={m.explorer.detailTitle}>
        {detailLoading ? <LoadingSkeleton rows={5} /> : null}
        {!detailLoading && detailError ? <p className={styles.emptyCopy}>{detailError}</p> : null}
        {!detailLoading && !detailError && activeTab === "segments" && !segmentDetail ? (
          <p className={styles.emptyCopy}>{m.explorer.empty.selectSegment}</p>
        ) : null}
        {!detailLoading && !detailError && activeTab === "segments" && segmentDetail ? (
          <div className={styles.pageStack}>
            <ExplorerFieldGroup
              title={m.explorer.groups.segmentIdentity}
              fields={[
                { label: m.explorer.fields.segmentId, value: `${segmentDetail.segmentID}` },
                { label: m.explorer.fields.path, value: segmentDetail.path },
                { label: m.explorer.fields.blockCount, value: `${segmentDetail.blockCount}` },
              ]}
            />
            <ExplorerFieldGroup
              title={m.explorer.groups.writeEnvelope}
              fields={[
                { label: m.explorer.fields.writeSeqRange, value: `${segmentDetail.firstWriteSeq} - ${segmentDetail.lastWriteSeq}` },
                { label: m.explorer.fields.eventTimeRange, value: `${formatTimestamp(segmentDetail.minEventTime)} - ${formatTimestamp(segmentDetail.maxEventTime)}` },
              ]}
            />
            <ExplorerFieldGroup
              title={m.explorer.groups.integrityPreview}
              fields={[
                { label: m.explorer.fields.footerHealth, value: localizeHealth(locale, segmentDetail.footerStatus) },
                { label: m.explorer.fields.tailStatus, value: localizeHealth(locale, segmentDetail.tailStatus) },
                { label: m.explorer.fields.rawPreview, value: segmentDetail.rawPreviewHex || "Preview capped to first 64 KiB" },
              ]}
            />
          </div>
        ) : null}
        {!detailLoading && !detailError && activeTab === "wal" && walDetail ? (
          <div className={styles.pageStack}>
            <ExplorerFieldGroup
              title={m.explorer.groups.walSnapshot}
              fields={[
                { label: m.explorer.fields.path, value: walDetail.path },
                { label: m.explorer.fields.size, value: `${walDetail.sizeBytes}` },
                { label: m.explorer.fields.health, value: localizeHealth(locale, walDetail.health) },
              ]}
            />
            <ExplorerFieldGroup
              title={m.explorer.groups.batchEnvelope}
              fields={[
                { label: m.explorer.fields.batchSeqRange, value: `${walDetail.firstBatchSeq} - ${walDetail.lastBatchSeq}` },
                { label: m.explorer.fields.lastEndOffset, value: `${walDetail.lastEndOffset}` },
              ]}
            />
          </div>
        ) : null}
        {!detailLoading && !detailError && activeTab === "wal" && !walDetail ? (
          <p className={styles.emptyCopy}>{m.explorer.empty.noWalDetail}</p>
        ) : null}
        {!detailLoading && !detailError && activeTab === "cursors" && !cursorDetail ? (
          <p className={styles.emptyCopy}>{m.explorer.empty.selectCursor}</p>
        ) : null}
        {!detailLoading && !detailError && activeTab === "cursors" && cursorDetail ? (
          <div className={styles.pageStack}>
            <ExplorerFieldGroup
              title={m.explorer.groups.cursorDestination}
              fields={[
                { label: m.explorer.fields.destination, value: cursorDetail.destination },
                { label: m.explorer.fields.writeSeq, value: `${cursorDetail.writeSeq}` },
                { label: m.explorer.fields.updatedAt, value: formatTimestamp(cursorDetail.updatedAt) },
              ]}
            />
            <ExplorerFieldGroup
              title={m.explorer.groups.replayPosition}
              fields={[
                { label: m.explorer.fields.segmentID, value: `${cursorDetail.segmentID}` },
                { label: m.explorer.fields.blockOffset, value: `${cursorDetail.blockOffset}` },
                { label: m.explorer.fields.recordIndex, value: `${cursorDetail.recordIndex}` },
              ]}
            />
            <ExplorerFieldGroup
              title={m.explorer.groups.integrity}
              fields={[
                { label: m.explorer.fields.crcStatus, value: localizeHealth(locale, cursorDetail.crcStatus) },
                { label: m.explorer.fields.backupStatus, value: cursorDetail.backupStatus },
              ]}
            />
          </div>
        ) : null}
        {!detailLoading && !detailError && activeTab === "checkpoint" && !checkpointDetail ? (
          <p className={styles.emptyCopy}>{m.explorer.empty.noCheckpointDetail}</p>
        ) : null}
        {!detailLoading && !detailError && activeTab === "checkpoint" && checkpointDetail ? (
          <div className={styles.pageStack}>
            <ExplorerFieldGroup
              title={m.explorer.groups.checkpointSummary}
              fields={[
                { label: m.explorer.fields.lastBatchSeq, value: `${checkpointDetail.lastBatchSeq}` },
                { label: m.explorer.fields.lastWALEndOffset, value: `${checkpointDetail.lastWALEndOffset}` },
                { label: m.explorer.fields.version, value: `${checkpointDetail.version}` },
              ]}
            />
            <ExplorerFieldGroup
              title={m.explorer.groups.auditStamp}
              fields={[
                { label: m.explorer.fields.updatedAt, value: formatTimestamp(checkpointDetail.updatedAt) },
                { label: m.explorer.fields.integrityStatus, value: localizeHealth(locale, checkpointDetail.integrityStatus) },
              ]}
            />
          </div>
        ) : null}
      </DetailPane>
    </div>
  );
}

function ExplorerFieldGroup({ title, fields }: { title: string; fields: { label: string; value: string }[] }) {
  return (
    <section className={styles.pageStack}>
      <div className={styles.sectionHeader}>
        <h4 className={styles.sectionTitle}>{title}</h4>
      </div>
      <div className={styles.fieldList}>
        {fields.map((field) => (
          <ExplorerField key={field.label} label={field.label} value={field.value} />
        ))}
      </div>
    </section>
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

function detailPaneNote(locale: "zh-CN" | "en-US", activeTab: ExplorerTab) {
  const notes = getMessages(locale).explorer.detailNotes;
  switch (activeTab) {
    case "segments":
      return notes.segments;
    case "wal":
      return notes.wal;
    case "cursors":
      return notes.cursors;
    case "checkpoint":
      return notes.checkpoint;
    default:
      return notes.segments;
  }
}
