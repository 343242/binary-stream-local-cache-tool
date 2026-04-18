import styles from "../styles/shell.module.css";

type LoadingSkeletonProps = {
  rows?: number;
  variant?: "default" | "compact";
  ariaLabel?: string;
};

const widthPresets = {
  default: ["100%", "88%", "72%"],
  compact: ["72%", "64%", "56%"],
} as const;

export default function LoadingSkeleton({ rows = 3, variant = "default", ariaLabel = "Loading" }: LoadingSkeletonProps) {
  const widths = widthPresets[variant];

  return (
    <div aria-label={ariaLabel} aria-live="polite" className={styles.pageStack} role="status">
      {Array.from({ length: rows }, (_, index) => (
        <div
          key={index}
          className={styles.skeleton}
          style={{ width: widths[index % widths.length] }}
        />
      ))}
    </div>
  );
}
