import { useEffect } from "react";

import LoadingSkeleton from "./components/LoadingSkeleton";
import Sidebar from "./components/Sidebar";
import TopBar from "./components/TopBar";
import ToastRegion from "./components/ToastRegion";
import WriterConfigModal from "./components/WriterConfigModal";
import { getMessages, nextLocale } from "./i18n";
import ConfigPage from "./pages/ConfigPage";
import ExplorerPage from "./pages/ExplorerPage";
import HomePage from "./pages/HomePage";
import LandingPage from "./pages/LandingPage";
import OperationsPage from "./pages/OperationsPage";
import OverviewPage from "./pages/OverviewPage";
import { useAppStore } from "./state/app-store";
import styles from "./styles/shell.module.css";

export default function App() {
  const {
    locale,
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
    writerStatus,
    pendingConfig,
    effectiveConfig,
    writerEvents,
    writerAlerts,
    isWriterConfigModalOpen,
    writerConfigSavePending,
    configSections,
    explorerTab,
    selectedOperation,
    latestResult,
    currentTask,
    toasts,
    confirmDialog,
    setPage,
    setLocale,
    setExplorerTab,
    setSelectedSegment,
    setSelectedCursor,
    setSelectedOperation,
    initialiseRuntime,
    openWorkspace,
    startWriter,
    stopWriter,
    openWriterConfig,
    closeWriterConfig,
    savePendingWriterConfig,
    refresh,
    requestOperation,
    confirmOperation,
    dismissDialog,
    openRepairFromResult,
    cancelCurrentTask,
    dismissToast,
  } = useAppStore();

  const m = getMessages(locale);
  const isInvalidWorkspace = workspace?.mode === "InvalidWorkspace";
  const isOpeningWorkspace = workspaceLoadState === "choosing" || workspaceLoadState === "hydrating";

  useEffect(() => {
    initialiseRuntime();
  }, [initialiseRuntime]);

  return (
    <main className={`${styles.frame} ${styles.frameScrollable}`}>
      <Sidebar activePage={page} onNavigate={setPage} workspace={workspace} locale={locale} />
      <section className={styles.content}>
        <TopBar
          locale={locale}
          onToggleLocale={() => setLocale(nextLocale(locale))}
          workspace={workspace}
          writerStatus={writerStatus}
          writerAlerts={writerAlerts}
          onRefresh={refresh}
          workspaceLoadState={workspaceLoadState}
        />
        <div className={styles.pageStack}>
          {isOpeningWorkspace && !workspace ? (
            <section className={`${styles.panel} ${styles.panelPadding}`}>
              <div className={styles.pageStack}>
                <h3 className={styles.sectionTitle}>{m.common.loadingWorkspace}</h3>
                <p className={styles.emptyCopy}>{m.common.loadingWorkspaceCopy}</p>
                <LoadingSkeleton rows={4} ariaLabel={m.common.loading} />
              </div>
            </section>
          ) : !workspace || isInvalidWorkspace ? (
            <LandingPage
              recentWorkspaces={recentWorkspaces}
              invalidWorkspace={
                isInvalidWorkspace
                  ? {
                      path: workspace.rootPath,
                      reason: workspace.invalidReason ?? (locale === "zh-CN" ? "所选路径不符合要求的缓存布局。" : "The selected path does not match the required cache layout."),
                    }
                  : undefined
              }
              onOpenWorkspace={openWorkspace}
            />
          ) : page === "home" ? (
            <HomePage
              locale={locale}
              onOpenWriterConfig={openWriterConfig}
              onStartWriter={() => void startWriter()}
              onStopWriter={() => void stopWriter()}
              workspace={workspace}
              writerAlerts={writerAlerts}
              writerEvents={writerEvents}
              writerStatus={writerStatus}
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
            <ConfigPage hasWorkspace={Boolean(workspace)} onOpenWriterConfig={openWriterConfig} sections={configSections} />
          ) : (
            <OperationsPage
              workspace={workspace}
              selectedSegmentID={selectedSegment?.segmentID ?? null}
              selectedOperation={selectedOperation}
              latestResult={latestResult}
              currentTask={currentTask}
              confirmDialog={confirmDialog}
              onSelectOperation={setSelectedOperation}
              onRunOperation={requestOperation}
              onConfirmDialog={confirmOperation}
              onDismissDialog={dismissDialog}
              onOpenRepair={openRepairFromResult}
              onCancelTask={cancelCurrentTask}
            />
          )}
        </div>
      </section>
      <WriterConfigModal
        effectiveConfig={effectiveConfig}
        isOpen={isWriterConfigModalOpen}
        isSaving={writerConfigSavePending}
        locale={locale}
        onClose={closeWriterConfig}
        onSave={(config) => void savePendingWriterConfig(config)}
        pendingConfig={pendingConfig}
        writerStatus={writerStatus}
      />
      <ToastRegion locale={locale} toasts={toasts} onDismiss={dismissToast} />
    </main>
  );
}
