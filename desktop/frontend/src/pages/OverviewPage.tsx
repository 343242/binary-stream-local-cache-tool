import EmptyState from "../components/EmptyState";
import StatusCard from "../components/StatusCard";
import { getMessages, localizeHealth, localizeLockMode, localizeWorkspaceMode } from "../i18n";
import { useAppStore } from "../state/app-store";
import styles from "../styles/shell.module.css";
import type { CursorRow, OverviewCard, SegmentRow } from "../state/app-store";

type OverviewPageProps = {
  cards: OverviewCard[];
  warnings: string[];
  segments: SegmentRow[];
  cursors: CursorRow[];
};

export default function OverviewPage({ cards, warnings, segments, cursors }: OverviewPageProps) {
  const locale = useAppStore((state) => state.locale);
  const m = getMessages(locale);
  const [leadCard, ...remainingCards] = cards;
  const summaryCards = remainingCards.slice(0, 2);
  const supportingCards = remainingCards.slice(2);

  return (
    <div className={styles.pageStack}>
      <section className={`${styles.panel} ${styles.hero}`}>
        <p className={styles.eyebrow}>{m.overview.eyebrow}</p>
        <h2 className={styles.heroTitle}>{m.overview.title}</h2>
        <p className={styles.heroCopy}>{m.overview.copy}</p>
      </section>

      {leadCard ? (
        <section className={styles.overviewMetrics}>
          <div className={styles.overviewMetricLead}>
            <StatusCard
              key={leadCard.key}
              eyebrow={m.overview.prioritySnapshot}
              label={localizeOverviewCardLabel(m, leadCard.key)}
              value={localizeOverviewCardValue(locale, leadCard)}
              secondary={localizeOverviewCardSecondary(locale, leadCard)}
              tier="hero"
              testId="overview-metric-lead"
            />
          </div>
          <div className={styles.overviewMetricSummary}>
            {summaryCards.map((card) => (
              <StatusCard
                key={card.key}
                label={localizeOverviewCardLabel(m, card.key)}
                value={localizeOverviewCardValue(locale, card)}
                secondary={localizeOverviewCardSecondary(locale, card)}
              />
            ))}
          </div>
        </section>
      ) : null}

      {supportingCards.length ? (
        <section className={styles.cardGrid}>
          {supportingCards.map((card) => (
            <StatusCard
              key={card.key}
              label={localizeOverviewCardLabel(m, card.key)}
              value={localizeOverviewCardValue(locale, card)}
              secondary={localizeOverviewCardSecondary(locale, card)}
            />
          ))}
        </section>
      ) : null}

      <section className={`${styles.panel} ${styles.panelPadding}`}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.overview.maintenanceEyebrow}</p>
            <h3 className={styles.sectionTitle}>{m.overview.maintenanceTitle}</h3>
          </div>
        </div>
        <p className={styles.emptyCopy}>{m.overview.maintenanceCopy}</p>
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
            eyebrow={m.overview.maintenanceEyebrow}
            variant="inline"
            title={m.overview.noWarningsTitle}
            message={m.overview.noWarningsCopy}
          />
        )}
      </section>

      <section className={styles.pageStack}>
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.overview.activityEyebrow}</p>
            <h3 className={styles.sectionTitle}>{m.overview.activityTitle}</h3>
          </div>
        </div>
        <p className={styles.emptyCopy}>{m.overview.activityCopy}</p>
        <div className={styles.cardGrid}>
          <div className={`${styles.panel} ${styles.panelPadding}`}>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>{m.overview.recentSegments}</h3>
            </div>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>{m.explorer.tables.segment}</th>
                  <th>{m.explorer.tables.records}</th>
                  <th>{m.topBar.health}</th>
                </tr>
              </thead>
              <tbody>
                {segments.slice(0, 8).map((segment) => (
                  <tr key={segment.segmentID}>
                    <td>{segment.segmentID}</td>
                    <td>{segment.recordCount}</td>
                    <td>{localizeHealth(locale, segment.health)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={`${styles.panel} ${styles.panelPadding}`}>
            <div className={styles.sectionHeader}>
              <h3 className={styles.sectionTitle}>{m.overview.recentCursors}</h3>
            </div>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>{m.explorer.tables.destination}</th>
                  <th>{m.explorer.tables.writeSeq}</th>
                  <th>{m.explorer.tables.status}</th>
                </tr>
              </thead>
              <tbody>
                {cursors.slice(0, 8).map((cursor) => (
                  <tr key={cursor.destination}>
                    <td>{cursor.destination}</td>
                    <td>{cursor.writeSeq}</td>
                    <td>{localizeHealth(locale, cursor.status)}</td>
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

function localizeOverviewCardLabel(messages: ReturnType<typeof getMessages>, key: OverviewCard["key"]) {
  return messages.overview.cards[key];
}

function localizeOverviewCardValue(locale: "zh-CN" | "en-US", card: OverviewCard) {
  if (card.key === "workspace") {
    return localizeWorkspaceMode(locale, card.value);
  }
  if (card.key === "lock") {
    return localizeLockMode(locale, card.value);
  }
  return card.value;
}

function localizeOverviewCardSecondary(locale: "zh-CN" | "en-US", card: OverviewCard) {
  if (card.key === "lock") {
    return localizeHealth(locale, card.secondary);
  }
  return card.secondary;
}
