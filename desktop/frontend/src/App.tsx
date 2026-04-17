import { useEffect } from "react";

import LoadingSkeleton from "./components/LoadingSkeleton";
import Sidebar from "./components/Sidebar";
import TopBar from "./components/TopBar";
import ConfigPage from "./pages/ConfigPage";
import ExplorerPage from "./pages/ExplorerPage";
import LandingPage from "./pages/LandingPage";
import OperationsPage from "./pages/OperationsPage";
import OverviewPage from "./pages/OverviewPage";
import { useAppStore } from "./state/app-store";
import styles from "./styles/shell.module.css";

export default function App() {
  const {
    page,
    workspace,
    workspaceLoadState,
    recentWorkspaces,
    overviewCards,
    warningSummary,
    recentSegments,
    recentCursors,
    selectedSegment,
    selectedCursor,
    segmentDetail,
    walDetail,
    cursorDetail,
    checkpointDetail,
    explorerDetailLoading,
    explorerDetailError,
    configSections,
    explorerTab,
    selectedOperation,
    latestResult,
    currentTask,
    toasts,
    confirmDialog,
    setPage,
    setExplorerTab,
    setSelectedSegment,
    setSelectedCursor,
    setSelectedOperation,
    initialiseRuntime,
    loadDemoWorkspace,
    refresh,
    requestOperation,
    confirmOperation,
    dismissDialog,
    openRepairFromResult,
    cancelCurrentTask,
    dismissToast,
  } = useAppStore();

  const isInvalidWorkspace = workspace?.mode === "InvalidWorkspace";
  const isOpeningWorkspace = workspaceLoadState === "choosing" || workspaceLoadState === "hydrating";

  useEffect(() => {
    initialiseRuntime();
  }, [initialiseRuntime]);

  return (
    <main className={styles.frame}>
      <Sidebar activePage={page} onNavigate={setPage} />
      <section className={styles.content}>
        <TopBar workspace={workspace} onRefresh={refresh} workspaceLoadState={workspaceLoadState} />
        <div className={styles.pageStack}>
          {isOpeningWorkspace && !workspace ? (
            <section className={`${styles.panel} ${styles.panelPadding}`}>
              <div className={styles.pageStack}>
                <h3 className={styles.sectionTitle}>Loading workspace</h3>
                <p className={styles.emptyCopy}>Validating the selected root and hydrating the initial snapshot.</p>
                <LoadingSkeleton rows={4} />
              </div>
            </section>
          ) : !workspace || isInvalidWorkspace ? (
            <LandingPage
              recentWorkspaces={recentWorkspaces}
              invalidWorkspace={
                isInvalidWorkspace
                  ? {
                      path: workspace.rootPath,
                      reason: workspace.invalidReason ?? "The selected path does not match the required cache layout.",
                    }
                  : undefined
              }
              onOpenWorkspace={loadDemoWorkspace}
            />
          ) : page === "overview" ? (
            <OverviewPage cards={overviewCards} warnings={warningSummary} segments={recentSegments} cursors={recentCursors} />
          ) : page === "explorer" ? (
            <ExplorerPage
              activeTab={explorerTab}
              onTabChange={setExplorerTab}
              segments={recentSegments}
              selectedSegment={selectedSegment}
              cursors={recentCursors}
              selectedCursor={selectedCursor}
              segmentDetail={segmentDetail}
              walDetail={walDetail}
              cursorDetail={cursorDetail}
              checkpointDetail={checkpointDetail}
              detailLoading={explorerDetailLoading}
              detailError={explorerDetailError}
              onSelectSegment={setSelectedSegment}
              onSelectCursor={setSelectedCursor}
            />
          ) : page === "config" ? (
            <ConfigPage hasWorkspace={Boolean(workspace)} sections={configSections} />
          ) : (
            <OperationsPage
              workspace={workspace}
              selectedSegmentID={selectedSegment?.segmentID ?? null}
              selectedOperation={selectedOperation}
              latestResult={latestResult}
              currentTask={currentTask}
              toasts={toasts}
              confirmDialog={confirmDialog}
              onSelectOperation={setSelectedOperation}
              onRunOperation={requestOperation}
              onConfirmDialog={confirmOperation}
              onDismissDialog={dismissDialog}
              onOpenRepair={openRepairFromResult}
              onCancelTask={cancelCurrentTask}
              onDismissToast={dismissToast}
            />
          )}
        </div>
      </section>
    </main>
  );
}
