import styles from "../styles/shell.module.css";

type StatusCardProps = {
  label: string;
  value: string;
  secondary: string;
};

export default function StatusCard({ label, value, secondary }: StatusCardProps) {
  return (
    <article className={styles.statusCard}>
      <p className={styles.cardLabel}>{label}</p>
      <h3 className={styles.cardValue}>{value}</h3>
      <p className={styles.cardSecondary}>{secondary}</p>
    </article>
  );
}
