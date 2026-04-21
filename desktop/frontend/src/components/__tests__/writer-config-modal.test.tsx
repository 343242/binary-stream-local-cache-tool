import { fireEvent, render, screen } from "@testing-library/react";

import WriterConfigModal from "../WriterConfigModal";
import type { WriterConfigState, WriterStatusState } from "../../state/app-store";

const pendingConfig: WriterConfigState = {
  rootDir: "/var/lib/binary-stream/cache-alpha",
  segmentTargetSizeBytes: 134217728,
  segmentSlackSizeBytes: 4194304,
  blockTargetSizeBytes: 1048576,
  checkpointInterval: 5000000000,
  checkpointBytes: 67108864,
  segmentFsyncInterval: 100000000,
  segmentFsyncBytes: 4194304,
  retentionDays: 14,
};

const effectiveConfig: WriterConfigState = {
  ...pendingConfig,
  retentionDays: 30,
};

const stoppedStatus: WriterStatusState = {
  lifecycleState: "stopped",
  workspaceState: "HealthyObserver",
  rootPath: pendingConfig.rootDir,
  lastError: "",
  startedAtUnixMs: 0,
  stoppedAtUnixMs: 0,
};

describe("writer config modal", () => {
  test("renders the full config structure with only the approved subset enabled", () => {
    render(
      <WriterConfigModal
        isOpen
        pendingConfig={pendingConfig}
        effectiveConfig={effectiveConfig}
        writerStatus={stoppedStatus}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );

    expect(screen.getByRole("dialog", { name: "写入配置" })).toBeInTheDocument();
    expect(screen.getByLabelText("存储位置")).toBeEnabled();
    expect(screen.getByLabelText("保留天数")).toBeEnabled();
    expect(screen.getByLabelText("段文件目标大小")).toBeEnabled();
    expect(screen.getByLabelText("检查点间隔")).toBeEnabled();
    expect(screen.getByLabelText("检查点字节阈值")).toBeEnabled();
    expect(screen.getByLabelText("段文件 fsync 间隔")).toBeEnabled();
    expect(screen.getByLabelText("段文件 fsync 字节阈值")).toBeEnabled();

    expect(screen.getByLabelText("段文件预留空间")).toBeDisabled();
    expect(screen.getByLabelText("数据块目标大小")).toBeDisabled();
  });

  test("saves the edited pending config", () => {
    const onSave = vi.fn();

    render(
      <WriterConfigModal
        isOpen
        pendingConfig={pendingConfig}
        effectiveConfig={effectiveConfig}
        writerStatus={stoppedStatus}
        onClose={vi.fn()}
        onSave={onSave}
      />,
    );

    fireEvent.change(screen.getByLabelText("保留天数"), { target: { value: "21" } });
    fireEvent.change(screen.getByLabelText("存储位置"), { target: { value: "/srv/cache/replica-west" } });
    fireEvent.click(screen.getByRole("button", { name: "保存待启动配置" }));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        retentionDays: 21,
        rootDir: "/srv/cache/replica-west",
      }),
    );
  });
});
