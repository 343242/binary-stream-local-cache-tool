import styles from "../styles/shell.module.css";

type StatusCardProps = {
  label: string;
  value: string;
  secondary: string;
  eyebrow?: string;
  tier?: "default" | "hero";
  testId?: string;
};

export default function StatusCard({ label, value, secondary, eyebrow, tier = "default", testId }: StatusCardProps) {
  return (
    <article className={`${styles.statusCard} ${tier === "hero" ? styles.statusCardHero : ""}`} data-testid={testId}>
      {eyebrow ? <p className={styles.eyebrow}>{eyebrow}</p> : null}
      <p className={styles.cardLabel}>{label}</p>
      <h3 className={styles.cardValue}>{value}</h3>
      <p className={styles.cardSecondary}>{secondary}</p>
    </article>
  );
}
