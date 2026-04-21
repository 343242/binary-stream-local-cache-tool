import { useEffect, useState } from "react";

import { defaultLocale, getMessages, type LocaleKey } from "../i18n";
import styles from "../styles/shell.module.css";
import type { WriterConfigState, WriterStatusState } from "../state/app-store";

type WriterConfigModalProps = {
  isOpen: boolean;
  pendingConfig: WriterConfigState;
  effectiveConfig: WriterConfigState;
  writerStatus: WriterStatusState;
  onClose: () => void;
  onSave: (config: WriterConfigState) => void;
  isSaving?: boolean;
  locale?: LocaleKey;
};

export default function WriterConfigModal({
  isOpen,
  pendingConfig,
  effectiveConfig,
  writerStatus,
  onClose,
  onSave,
  isSaving = false,
  locale = defaultLocale,
}: WriterConfigModalProps) {
  const [draft, setDraft] = useState(pendingConfig);
  const m = getMessages(locale);
  const editingBlocked = writerStatus.lifecycleState === "running" || writerStatus.lifecycleState === "starting" || writerStatus.lifecycleState === "stopping";

  useEffect(() => {
    setDraft(pendingConfig);
  }, [pendingConfig]);

  if (!isOpen) {
    return null;
  }

  return (
    <div className={styles.dialogBackdrop} role="presentation">
      <div aria-label={m.writerConfig.title} aria-modal="true" className={styles.dialogWide} role="dialog">
        <div className={styles.sectionHeader}>
          <div className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.writerConfig.eyebrow}</p>
            <h3 className={styles.sectionTitle}>{m.writerConfig.title}</h3>
            <p className={styles.emptyCopy}>{editingBlocked ? m.writerConfig.blockedWhileRunning : m.writerConfig.copy}</p>
          </div>
          <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onClose} type="button">
            {m.common.dismiss}
          </button>
        </div>

        <div className={styles.configModalGrid}>
          <section className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.writerConfig.pendingSection}</p>
            <div className={styles.fieldList}>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.rootDir}</span>
                <input
                  className={styles.configInput}
                  disabled={editingBlocked}
                  onChange={(event) => setDraft((current) => ({ ...current, rootDir: event.target.value }))}
                  type="text"
                  value={draft.rootDir}
                />
              </label>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.retentionDays}</span>
                <input
                  className={styles.configInput}
                  disabled={editingBlocked}
                  min={1}
                  onChange={(event) => setDraft((current) => ({ ...current, retentionDays: Number(event.target.value) || 0 }))}
                  type="number"
                  value={draft.retentionDays}
                />
              </label>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.segmentTargetSizeBytes}</span>
                <input
                  className={styles.configInput}
                  disabled={editingBlocked}
                  onChange={(event) => setDraft((current) => ({ ...current, segmentTargetSizeBytes: Number(event.target.value) || 0 }))}
                  type="number"
                  value={draft.segmentTargetSizeBytes}
                />
              </label>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.segmentSlackSizeBytes}</span>
                <input className={styles.configInput} disabled type="number" value={draft.segmentSlackSizeBytes} />
              </label>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.blockTargetSizeBytes}</span>
                <input className={styles.configInput} disabled type="number" value={draft.blockTargetSizeBytes} />
              </label>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.checkpointInterval}</span>
                <input
                  className={styles.configInput}
                  disabled={editingBlocked}
                  onChange={(event) => setDraft((current) => ({ ...current, checkpointInterval: parseDurationToNanos(event.target.value) }))}
                  type="text"
                  value={formatDurationValue(draft.checkpointInterval)}
                />
              </label>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.checkpointBytes}</span>
                <input
                  className={styles.configInput}
                  disabled={editingBlocked}
                  onChange={(event) => setDraft((current) => ({ ...current, checkpointBytes: Number(event.target.value) || 0 }))}
                  type="number"
                  value={draft.checkpointBytes}
                />
              </label>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.segmentFsyncInterval}</span>
                <input
                  className={styles.configInput}
                  disabled={editingBlocked}
                  onChange={(event) => setDraft((current) => ({ ...current, segmentFsyncInterval: parseDurationToNanos(event.target.value) }))}
                  type="text"
                  value={formatDurationValue(draft.segmentFsyncInterval)}
                />
              </label>
              <label className={styles.configInputGroup}>
                <span className={styles.fieldKey}>{m.writerConfig.fields.segmentFsyncBytes}</span>
                <input
                  className={styles.configInput}
                  disabled={editingBlocked}
                  onChange={(event) => setDraft((current) => ({ ...current, segmentFsyncBytes: Number(event.target.value) || 0 }))}
                  type="number"
                  value={draft.segmentFsyncBytes}
                />
              </label>
            </div>
          </section>

          <section className={styles.pageStack}>
            <p className={styles.eyebrow}>{m.writerConfig.effectiveSection}</p>
            <div className={styles.fieldList}>
              {[
                [m.writerConfig.fields.rootDir, effectiveConfig.rootDir],
                [m.writerConfig.fields.retentionDays, `${effectiveConfig.retentionDays}`],
                [m.writerConfig.fields.segmentTargetSizeBytes, `${effectiveConfig.segmentTargetSizeBytes}`],
                [m.writerConfig.fields.segmentSlackSizeBytes, `${effectiveConfig.segmentSlackSizeBytes}`],
                [m.writerConfig.fields.blockTargetSizeBytes, `${effectiveConfig.blockTargetSizeBytes}`],
                [m.writerConfig.fields.checkpointInterval, formatDurationValue(effectiveConfig.checkpointInterval)],
                [m.writerConfig.fields.checkpointBytes, `${effectiveConfig.checkpointBytes}`],
                [m.writerConfig.fields.segmentFsyncInterval, formatDurationValue(effectiveConfig.segmentFsyncInterval)],
                [m.writerConfig.fields.segmentFsyncBytes, `${effectiveConfig.segmentFsyncBytes}`],
              ].map(([label, value]) => (
                <div key={label} className={styles.fieldRow}>
                  <span className={styles.fieldKey}>{label}</span>
                  <span>{value}</span>
                </div>
              ))}
            </div>
          </section>
        </div>

        <div className={styles.dialogActions}>
          <button className={`${styles.secondaryButton} ${styles.focusable}`} onClick={onClose} type="button">
            {m.common.dismiss}
          </button>
          <button
            className={`${styles.primaryButton} ${styles.focusable}`}
            disabled={editingBlocked || isSaving}
            onClick={() => onSave(draft)}
            type="button"
          >
            {m.writerConfig.saveAction}
          </button>
        </div>
      </div>
    </div>
  );
}

function formatDurationValue(value: number) {
  if (value % 1_000_000_000 === 0) {
    return `${value / 1_000_000_000}s`;
  }
  if (value % 1_000_000 === 0) {
    return `${value / 1_000_000}ms`;
  }
  return `${value}`;
}

function parseDurationToNanos(value: string) {
  const normalized = value.trim();
  const match = normalized.match(/^(\d+)(ms|s|m|h)$/);
  if (!match) {
    return Number(normalized) || 0;
  }
  const amount = Number(match[1]);
  switch (match[2]) {
    case "ms":
      return amount * 1_000_000;
    case "s":
      return amount * 1_000_000_000;
    case "m":
      return amount * 60 * 1_000_000_000;
    case "h":
      return amount * 60 * 60 * 1_000_000_000;
    default:
      return 0;
  }
}
