import styles from "../styles/shell.module.css";

type LoadingSkeletonProps = {
  rows?: number;
};

export default function LoadingSkeleton({ rows = 3 }: LoadingSkeletonProps) {
  return (
    <div className={styles.pageStack} aria-label="Loading">
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className={styles.skeleton} />
      ))}
    </div>
  );
}
