import styles from "../styles/shell.module.css";

type DetailPaneProps = {
  title: string;
  children: React.ReactNode;
};

export default function DetailPane({ title, children }: DetailPaneProps) {
  return (
    <section className={`${styles.panel} ${styles.detailPane}`}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>{title}</h3>
      </div>
      <div className={styles.pageStack}>{children}</div>
    </section>
  );
}
