import { useEffect } from "react";

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
    recentWorkspaces,
    overviewCards,
    warningSummary,
    recentSegments,
    recentCursors,
    selectedSegment,
    configSections,
    explorerTab,
    selectedOperation,
    latestResult,
    currentTask,
    toasts,
    confirmDialog,
    setPage,
    setExplorerTab,
    setSelectedOperation,
    initialiseRuntime,
    loadDemoWorkspace,
    refresh,
    requestOperation,
    confirmOperation,
    dismissDialog,
    openRepairFromResult,
    dismissToast,
  } = useAppStore();

  const isInvalidWorkspace = workspace?.mode === "InvalidWorkspace";

  useEffect(() => {
    initialiseRuntime();
  }, [initialiseRuntime]);

  return (
    <main className={styles.frame}>
      <Sidebar activePage={page} onNavigate={setPage} />
      <section className={styles.content}>
        <TopBar workspace={workspace} onRefresh={refresh} />
        <div className={styles.pageStack}>
          {!workspace || isInvalidWorkspace ? (
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
            />
          ) : page === "config" ? (
            <ConfigPage hasWorkspace={Boolean(workspace)} sections={configSections} />
          ) : (
            <OperationsPage
              workspace={workspace}
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
              onDismissToast={dismissToast}
            />
          )}
        </div>
      </section>
    </main>
  );
}
