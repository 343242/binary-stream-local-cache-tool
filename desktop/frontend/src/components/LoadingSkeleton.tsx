import styles from "../styles/shell.module.css";

type LoadingSkeletonProps = {
  rows?: number;
};

export default function LoadingSkeleton({ rows = 3 }: LoadingSkeletonProps) {
  return (
    <div aria-label="Loading" aria-live="polite" className={styles.pageStack} role="status">
      {Array.from({ length: rows }, (_, index) => (
        <div
          key={index}
          className={styles.skeleton}
          style={{ width: index === 0 ? "100%" : index % 2 === 0 ? "72%" : "88%" }}
        />
      ))}
    </div>
  );
}
