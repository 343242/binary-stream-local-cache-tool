import styles from "../styles/shell.module.css";

type DetailPaneProps = {
  title: string;
  eyebrow?: string;
  note?: string;
  children: React.ReactNode;
};

export default function DetailPane({ title, eyebrow, note, children }: DetailPaneProps) {
  return (
    <section className={`${styles.panel} ${styles.detailPane}`}>
      <div className={styles.sectionHeader}>
        <div className={styles.pageStack}>
          {eyebrow ? <p className={styles.eyebrow}>{eyebrow}</p> : null}
          <h3 className={styles.sectionTitle}>{title}</h3>
          {note ? <p className={styles.emptyCopy}>{note}</p> : null}
        </div>
      </div>
      <div className={styles.pageStack}>{children}</div>
    </section>
  );
}
