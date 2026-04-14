# 基于本机文件的二进制流高速缓存工具设计

## 1. 背景与目标

工业数据采集系统在网络或主服务器不可用时，需要把采集数据可靠地暂存在本机文件系统，并在恢复后按顺序回放未上传数据。

本设计遵守以下明确约束：

- 单机部署
- 单进程写入
- 数据以二进制流方式存储
- 写入链路必须保持顺序追加
- 文件必须按大小滚动，不能无限增长
- 文件滚动时允许少量超出空间，避免最后一个完整数据块被截断
- 第一优先读取场景是“按顺序回放未上传数据”
- 指标目标：`10000` 条数据读或写单次目标耗时 `<100ms`
- 至少支持一个月缓存
- 容量规划采用混合负载假设：日常平均 `4000` 条/秒，峰值 `10000` 条/秒

## 2. 设计目标

### 2.1 功能目标

- 批量写入原始字节流
- 本地缓存一个月及以上
- 按顺序回放未上传数据
- 记录回放确认位点，支持断点续传
- 进程异常退出后可恢复
- 文件结构稳定，可扩展，但不要求业务数据格式固定

### 2.2 非功能目标

- 主写路径只做顺序追加
- 控制 CPU、内存和磁盘元数据开销
- 具备基本运维、巡检、修复能力
- 遇到尾损坏、断电、磁盘不足、游标损坏时有明确恢复策略

### 2.3 容量规划假设

容量规划以“平均负载保留 + 峰值吞吐承载”作为基准，而不是把峰值默认视为 30 天持续常态。

已确认业务假设：

- 日常平均写入速率：`4000` 条/秒
- 峰值写入速率：`10000` 条/秒
- 最低保留期：`30` 天

以“平均 payload 大小约 `50B`”估算：

- 平均场景 payload 写入量：
  - `4000 * 86400 = 345,600,000` 条/天
  - `345,600,000 * 50B ≈ 17.28GB/天`
  - `30` 天约 `518.4GB`
- 峰值场景 payload 写入量：
  - `10000 * 86400 = 864,000,000` 条/天
  - `864,000,000 * 50B ≈ 43.2GB/天`
  - `30` 天约 `1.296TB`

存储系统实际物理开销还包括：

- 固定记录头
- Block 头
- WAL 冗余窗口
- Segment 尾元数据
- Cursor 和 checkpoint 元数据

因此容量规划结论为：

- 若按“平均 4000 条/秒保留 30 天”设计，建议可用磁盘不低于 `1TB`
- 若要求“10000 条/秒长期持续并保留 30 天”，建议可用磁盘不低于 `2.5TB`
- 第一版实现和 benchmark 默认以“平均容量规划 + 峰值吞吐验证”作为验收口径

## 3. 总体架构

系统由 8 个核心部件组成：

1. `StorageEngine`
   - 对外提供 `WriteBatch`、`Replay`、`Ack`、`Recover`、`Stats`
2. `RecordCodec`
   - 把调用方传入的字节流组装成可落盘的记录和块
3. `WALManager`
   - 负责写前日志、checkpoint、崩溃恢复
4. `SegmentManager`
   - 负责段文件创建、顺序追加、按大小滚动、段尾元数据
5. `ReplayManager`
   - 负责按顺序读取未上传数据并维护回放游标
6. `RetentionManager`
   - 负责按保留策略清理已确认且过期的段
7. `RecoveryManager`
   - 负责启动扫描、尾部截断、WAL 重放
8. `OpsToolkit`
   - 提供巡检、统计、校验、修复、导出工具

## 3.1 核心标识初始化规则

### 3.1.1 `segment_id` 初始化

- 全新空目录启动时，首个 `segment_id = 1`
- 恢复启动时，扫描 `segments/` 目录中的有效段文件名
- 取“最大有效 `segment_id` + 1”作为下一个新段 ID
- 允许历史段号存在空洞，不回收、不复用旧 ID

判定“有效段”的最低条件：

- 文件名可解析出 `segment_id`
- 段尾元数据可读且通过校验
- 或者该段是当前活动段，虽未封口，但至少包含一个完整 `block`

### 3.1.2 `writeSeq` 初始化

`writeSeq` 的下一可用值必须在恢复完成后确定，而不是只看单一来源。

启动时按以下顺序执行：

1. 扫描所有有效段，得到 `maxSegmentWriteSeq`
2. 扫描有效 WAL，得到 `maxWalWriteSeq`
3. 执行 WAL 重放，使段文件状态追平到最后一个有效 WAL 批次
4. 重放完成后重新计算当前系统可见的最大顺序号 `maxRecoveredWriteSeq`
5. 设置 `nextWriteSeq = maxRecoveredWriteSeq + 1`

初始化规则总结：

- 新系统：`nextWriteSeq = 1`
- 恢复系统：`nextWriteSeq = max(recovered segment writeSeq, replayed WAL writeSeq) + 1`

冲突规则：

- 如果发现同一个 `writeSeq` 对应两份不同 payload，则视为严重损坏
- 系统启动失败，返回 `ERR_SEQUENCE_CONFLICT`
- 不尝试自动合并冲突数据

这样定义后，`writeSeq` 的唯一可信来源是“恢复完成后的整体状态”，而不是“只看段尾”或“只看 WAL”。

## 3.2 对外 API 契约

第一版采用 Go 库接口，签名固定如下：

```go
type RawRecord struct {
    EventTimeUnixMs int64
    Payload         []byte
}

type WriteBatchResult struct {
    BatchSeq        uint64
    FirstWriteSeq   uint64
    LastWriteSeq    uint64
    RecordCount     int
    SegmentID       uint64
    WALBytesWritten uint64
    SegmentBytesWritten uint64
}

type ReplayLimit struct {
    MaxRecords int
    MaxBytes   int64
}

type ReplayRecord struct {
    EventTimeUnixMs int64
    WriteSeq        uint64
    Payload         []byte
}

type ReplayBatch struct {
    Destination     string
    SegmentID        uint64
    NextCursor       ReplayCursor
    Records          []ReplayRecord
    RecordCount      int
    TotalPayloadBytes int64
}

type ReplayCursor struct {
    Version          uint32
    SegmentID        uint64
    BlockOffset      uint64
    RecordIndex      uint32
    WriteSeq         uint64
    UpdatedAtUnixMs  int64
    CRC32            uint32
}

type AckResult struct {
    Destination      string
    AppliedCursor     ReplayCursor
    PreviousWriteSeq  uint64
    CurrentWriteSeq   uint64
}

func (s *StorageEngine) WriteBatch(ctx context.Context, records []RawRecord) (WriteBatchResult, error)
func (s *StorageEngine) Replay(ctx context.Context, destination string, limit ReplayLimit) (ReplayBatch, error)
func (s *StorageEngine) Ack(ctx context.Context, destination string, cursor ReplayCursor) (AckResult, error)
func (s *StorageEngine) Recover(ctx context.Context) error
func (s *StorageEngine) Stats(ctx context.Context) (StatsSnapshot, error)
func (s *StorageEngine) Close() error
func (s *StorageEngine) Shutdown(ctx context.Context) error
```

接口语义：

- `WriteBatch`：写入一批原始字节流并返回已分配的顺序区间
- `Replay`：从目标端当前游标之后读取一批未确认数据
- `Ack`：用调用方确认过的 `ReplayCursor` 推进目标端游标
- `Recover`：执行显式恢复，正常启动时也会自动调用
- `Stats`：返回当前快照指标

`Ack` 必须满足：

- 只能前进，不能回退
- 只能接受系统曾发出的 `ReplayCursor`
- 若游标无效、越界、CRC 错误或倒退，返回 `ERR_CURSOR_INVALID`

## 4. 核心设计决策

### 4.1 只保证“系统元信息”，不规定业务数据格式

底层存储只关心顺序缓存系统运行所需的最小公共元信息，不解释业务 payload 的结构。

系统只定义：

- 记录边界
- 采集时间
- 写入顺序号
- payload 长度
- 校验信息

系统不定义：

- gatewayId 类型
- pointId 类型
- value 类型
- 任何业务字段编码方式

业务方只需要把一条采集数据编码为 `[]byte` 传入系统即可。

### 4.2 文件按大小滚动，不采用单大文件

系统按 `segment` 文件存储数据。

每个活动段包含两个核心阈值：

- `targetSize`：目标文件大小，例如 `256MB`
- `slackSize`：允许超出的缓冲，例如 `4MB`

滚动规则：

- 写入前先组装完整 `block`
- 若 `currentSize + blockSize <= targetSize + slackSize`，则写入当前段
- 否则先封口当前段，再切换到新段写入整个 `block`
- 不拆分 `block`
- 不拆分单条记录

### 4.3 整个系统以顺序追加为核心

该系统本质是顺序日志型本地缓存，不是数据库。

顺序性规则：

- WAL 追加写
- Segment 追加写
- 已封口段不重开写
- 回放按物理顺序扫描
- 不在主写路径做复杂索引
- 不为了点位查询引入随机写

### 4.4 回放顺序以“写入顺序”定义，不依赖传感器时间绝对单调

传感器时钟可能回拨，因此系统不能假设 `eventTime` 永远单调递增。

为保证回放稳定性，每条记录增加系统写入顺序号 `writeSeq`：

- `eventTime`：采集时间，由调用方提供
- `writeSeq`：本地单进程递增序号，由系统生成

默认回放顺序定义为：

1. 按 `segment_id`
2. 按 `block_offset`
3. 按 `record_index`
4. 等价于按 `writeSeq` 递增

也就是说：

- 保留原始采集时间
- 不因时间回拨打乱物理回放顺序
- 保证断点续传和恢复逻辑稳定

## 5. 二进制文件格式

### 5.1 记录格式

每条记录由固定头和原始 payload 组成。

建议格式：

```text
[magic(2)][version(2)]
[event_time_unix_ms(8)]
[write_seq(8)]
[payload_len(4)]
[header_crc32(4)]
[payload_crc32(4)]
[payload(N)]
```

字段说明：

- `magic`：记录起始标识
- `version`：物理格式版本
- `event_time_unix_ms`：采集时间，由上层传入
- `write_seq`：系统写入顺序号，单进程内单调递增
- `payload_len`：原始业务字节流长度
- `header_crc32`：头部校验
- `payload_crc32`：payload 校验
- `payload`：调用方提供的原始字节流

这一层不再定义 `source_id`、`gateway_id`、`point_id` 或字典编码。

### 5.2 Block 格式

段文件按 `block` 组织写入，`block` 是最小完整写入单元：

```text
[block_magic(4)][block_version(2)][record_count(2)]
[block_body_len(4)][first_write_seq(8)][last_write_seq(8)]
[min_event_time(8)][max_event_time(8)][block_crc32(4)]
[records...]
```

用途：

- 减少系统调用
- 支持整块校验
- 作为滚动与恢复边界

Block 大小策略：

- 默认 `blockTargetSize = 1MB`
- 可配置范围：`256KB - 4MB`
- 第一版不支持 `16MB` 级别大块作为默认值

选择 `1MB` 的原因：

- 足以容纳典型 `10000` 条批次的压缩前数据块级聚合
- 单次刷盘粒度和恢复粒度适中
- 内存占用和尾损坏修复成本较低

### 5.3 WAL 格式

WAL 记录批次级写入：

```text
[wal_magic(4)][wal_version(2)][reserved(2)]
[batch_seq(8)][payload_len(4)][wal_crc32(4)]
[block_bytes(N)]
```

说明：

- WAL 保存已组装好的 `block` 字节
- 崩溃恢复按 `batch_seq` 重放
- 重放成功后由 checkpoint 推进

## 5.4 错误码体系

第一版采用“错误码 + typed error”方式，而不是只返回字符串。

```go
type ErrorCode string

const (
    ErrValidation           ErrorCode = "ERR_VALIDATION"
    ErrClosed               ErrorCode = "ERR_CLOSED"
    ErrShutdownInProgress   ErrorCode = "ERR_SHUTDOWN_IN_PROGRESS"
    ErrDiskFull             ErrorCode = "ERR_DISK_FULL"
    ErrReadOnlyFS           ErrorCode = "ERR_READ_ONLY_FS"
    ErrIO                   ErrorCode = "ERR_IO"
    ErrCorruption           ErrorCode = "ERR_CORRUPTION"
    ErrCursorInvalid        ErrorCode = "ERR_CURSOR_INVALID"
    ErrCursorCorrupted      ErrorCode = "ERR_CURSOR_CORRUPTED"
    ErrSequenceConflict     ErrorCode = "ERR_SEQUENCE_CONFLICT"
    ErrSegmentFooterInvalid ErrorCode = "ERR_SEGMENT_FOOTER_INVALID"
    ErrWALInvalid           ErrorCode = "ERR_WAL_INVALID"
    ErrRecoveryRequired     ErrorCode = "ERR_RECOVERY_REQUIRED"
    ErrTimeout              ErrorCode = "ERR_TIMEOUT"
)
```

错误码语义：

- `ERR_VALIDATION`：入参非法，如空 payload、空批次、超大记录
- `ERR_CLOSED`：实例已关闭
- `ERR_SHUTDOWN_IN_PROGRESS`：系统正在优雅停机，不再接受新写入
- `ERR_DISK_FULL`：磁盘空间低于拒绝阈值
- `ERR_READ_ONLY_FS`：目标目录处于只读文件系统
- `ERR_IO`：一般 I/O 错误
- `ERR_CORRUPTION`：检测到但尚未细分的损坏
- `ERR_CURSOR_INVALID`：调用方传入的游标非法或倒退
- `ERR_CURSOR_CORRUPTED`：磁盘上的游标文件损坏
- `ERR_SEQUENCE_CONFLICT`：相同 `writeSeq` 出现不同内容
- `ERR_SEGMENT_FOOTER_INVALID`：段尾元数据损坏
- `ERR_WAL_INVALID`：WAL 条目损坏或不可解析
- `ERR_RECOVERY_REQUIRED`：系统需先恢复才能继续提供服务
- `ERR_TIMEOUT`：显式超时

建议统一错误结构：

```go
type StorageError struct {
    Code    ErrorCode
    Op      string
    Path    string
    Message string
    Cause   error
}
```

## 6. 磁盘布局

```text
storage/
  wal/
    active.wal
    checkpoint.meta
  segments/
    000001.seg
    000002.seg
    000003.seg
  metadata/
    replay/
      main-server.cursor
      main-server.cursor.bak
  tools/
    reports/
```

说明：

- `wal/` 保存写前日志和 checkpoint
- `segments/` 保存实际缓存数据
- `metadata/replay/` 保存目标端回放位点和备份位点
- `tools/reports/` 保存巡检和校验输出

## 7. 写入流程

### 7.1 WriteBatch 主流程

`WriteBatch(records)` 执行顺序：

1. 校验批次入参
2. 为每条记录分配 `writeSeq`
3. 为每条记录组装固定头和原始 payload
4. 将记录聚合为一个或多个完整 `block`
5. 将 `block` 追加写入 `active.wal`
6. `fsync(active.wal)`
7. 将完整 `block` 追加写入当前 `segment`
8. 更新段尾内存状态
9. 按“7.4 Segment fsync 策略”执行 `fsync(segment)`
10. 按“7.3 WAL checkpoint 策略”推进 checkpoint
11. 返回成功

### 7.2 写入确认语义

- WAL 已 `fsync` 后，该批次进入“可恢复”状态
- Segment 已写入并刷盘后，该批次进入“可回放”状态
- 任何时刻都不确认半个 `block`

### 7.3 WAL checkpoint 策略

checkpoint 不是每批次执行，而是按阈值触发。

第一版默认规则：

- 当 WAL 自上次 checkpoint 后新增数据达到 `64MB` 时触发
- 或距离上次 checkpoint 已达到 `5s` 时触发
- 二者满足其一即执行
- 在 `segment seal` 时强制执行一次
- 在 `Close()` / `Shutdown()` 时强制执行一次

这样做的目的是在恢复时间和运行期开销之间取得平衡。

### 7.4 Segment fsync 策略

Segment 不对每个 block 单独 `fsync`。

第一版默认规则：

- 当自上次 segment `fsync` 后新增数据达到 `4MB` 时触发
- 或距离上次 segment `fsync` 已达到 `100ms` 时触发
- 二者满足其一即执行
- 在 `segment seal` 时强制执行
- 在 `Close()` / `Shutdown()` 时强制执行

设计依据：

- durability 主要由 `WAL fsync` 提供
- segment 刷盘以吞吐优先
- 避免每个 block `fsync` 导致吞吐和尾延迟明显恶化

### 7.5 时间回拨处理

系统接受 `eventTime` 回拨，不拒绝写入。

处理方式：

- `eventTime` 原样保留
- 回放默认按 `writeSeq` 顺序，不按 `eventTime` 排序
- 统计指标单独记录 `clock_rollback_events`
- 若当前批次 `eventTime` 小于最近写入记录时间，则写入告警日志

这保证：

- 不丢真实采集时间
- 不让时间回拨破坏顺序回放

## 8. 回放与断点续传

### 8.1 回放优先级

第一版优先优化“按物理顺序回放未上传数据”。

### 8.2 游标结构

每个目标端维护一个游标文件：

```json
{
  "version": 1,
  "segment_id": 12,
  "block_offset": 1048576,
  "record_index": 128,
  "write_seq": 99887766,
  "updated_at_unix_ms": 1710000000000,
  "crc32": 123456789
}
```

### 8.3 回放流程

`Replay(destination, limit)` 执行顺序：

1. 加载目标端游标
2. 定位起始段
3. 按段号递增扫描
4. 按块顺序读取
5. 按记录顺序输出
6. 到达 `limit` 后返回
7. 调用方确认后，通过 `Ack` 推进游标

### 8.4 游标损坏恢复

游标文件采用“双文件 + 校验 + 原子替换”：

- 主文件：`main-server.cursor`
- 备份文件：`main-server.cursor.bak`

更新规则：

1. 先写临时文件
2. `fsync`
3. 原子替换主文件
4. 每次 `Ack` 成功后立即刷新备份文件

备份刷新规则：

1. 读取最新有效主游标内容
2. 写入 `.bak.tmp`
3. `fsync(.bak.tmp)`
4. 原子替换 `main-server.cursor.bak`

这样定义后，备份文件不会“周期性”刷新，而是在每次成功推进游标后刷新一次。

恢复规则：

- 主文件有效：使用主文件
- 主文件损坏且备份有效：回退到备份文件
- 主文件和备份都损坏：进入保守恢复模式

保守恢复模式：

- 从最早未清理的段起重新回放
- 以 `writeSeq` 去重由上游接收端负责兜底
- 系统输出 `cursor_rebuild_required` 告警

## 9. 按大小滚动策略

### 9.1 滚动条件

当“下一个完整 `block` 写入后将超过 `targetSize + slackSize`”时，当前段封口并滚动。

### 9.2 滚动步骤

1. 刷新当前段缓存
2. 写入段尾元数据
3. `fsync` 当前段
4. 关闭当前段
5. 创建新段
6. 新批次写入新段

### 9.3 段尾元数据

段尾至少包含：

- `segment_id`
- `created_at`
- `sealed_at`
- `record_count`
- `first_write_seq`
- `last_write_seq`
- `min_event_time`
- `max_event_time`
- `last_batch_seq`
- `footer_crc32`

第一版仅保留轻量段尾元数据，不引入复杂段内索引。

段尾损坏判定标准：

- `footer magic` 不匹配
- 段尾长度字段非法
- `footer_crc32` 校验失败
- `segment_id` 与文件名中的段号不一致
- `first_write_seq > last_write_seq`
- `min_event_time > max_event_time`
- 段尾声明的偏移超出文件实际大小

满足任一条件，即判定该段尾无效，返回 `ERR_SEGMENT_FOOTER_INVALID`。

## 10. 恢复与异常处理

### 10.1 启动恢复

启动时执行：

1. 扫描最后一个或几个段
2. 截断尾部不完整 `block`
3. 扫描 `active.wal`
4. 找到最后有效 `batch_seq`
5. 重放尚未体现在段尾元数据中的批次
6. 重建内存状态

### 10.1.1 WAL 与 Segment 不一致时的仲裁规则

恢复时以“有效 WAL + 有效段尾元数据 + 完整 block 边界”共同仲裁，但优先级明确如下：

1. 已通过 CRC 校验且位于最后有效 checkpoint 之后的 WAL 条目，是恢复写入意图的权威来源
2. 已通过校验的 sealed segment footer，是恢复段完成状态的权威来源
3. 超出最后有效 footer 且无法由有效 WAL 解释的 segment 尾部数据，视为脏尾部，直接截断

具体规则：

- WAL 有、segment 无：重放 WAL 到 segment
- WAL 有、segment 只有部分 block：截断残缺 block 后重放 WAL
- segment 有、WAL 无、且该数据位于最后有效 footer 之后：视为未受保护数据，截断
- segment 有、WAL 无、且数据已被最后有效 footer 覆盖：保留该 segment

因此可以简化为一句话：

- “checkpoint 之后，以有效 WAL 为准；footer 之外的孤立 segment 尾部数据不保留”

### 10.2 WAL 损坏

- 若 WAL 尾部存在不完整条目，截断到最后一条完整条目
- 若中间校验失败，停在最后一个有效边界并上报告警

### 10.3 Segment 尾损坏

- 只修复尾部
- 修复单位为完整 `block`
- 不尝试修复中间损坏

### 10.4 磁盘满

- 低水位：告警
- 拒绝水位：拒绝新写入
- 不自动删除未确认上传数据

### 10.4.1 只读磁盘与一般 I/O 错误

写路径必须区分“只读文件系统”和“一般 I/O 错误”：

- 遇到 `EROFS` 或等价错误：
  - 立即将实例标记为 `write-disabled`
  - 后续 `WriteBatch` 返回 `ERR_READ_ONLY_FS`
  - 允许 `Replay`、`Stats`、`inspect` 等只读操作继续
- 遇到临时或一般 I/O 错误：
  - 当前操作返回 `ERR_IO`
  - 实例进入 `degraded` 状态
  - 允许调用方重试或人工介入

任何 I/O 错误都不得让系统错误确认“写入成功”。

### 10.5 时间异常

- 记录 `eventTime` 回拨次数和最近回拨幅度
- 不因回拨阻塞写入
- 回放和恢复仍按 `writeSeq` 进行

### 10.6 Close / Shutdown 语义

系统必须提供优雅停机接口：

- `Close() error`
- `Shutdown(ctx context.Context) error`

语义定义：

1. 停止接收新写入请求
2. 等待当前写入中的批次完成
3. 刷出当前正在组装的 block
4. `fsync(active.wal)`
5. `fsync(active.segment)`
6. 推进 checkpoint
7. 刷新 cursor 主文件和备份文件
8. 写出最终运行统计
9. 关闭所有文件句柄

`Close()` 适用于无超时的本地同步关闭。

`Shutdown(ctx)` 适用于服务进程优雅退出：

- 若 `ctx` 超时，则返回超时错误
- 即使超时，也必须保证已经完成的 WAL 记录不丢失

## 11. 保留与清理

### 11.1 基本策略

- 默认配置保留期为 `30` 天
- 配置最小保留期为 `1` 天
- 配置最大保留期为 `365` 天
- 实际保留期定义为 `max(configuredRetentionDays, oldestUnackedSegmentAgeDays)`
- 清理粒度为整段文件
- 仅清理“超期且已确认”的段

### 11.2 清理约束

- 未确认回放完成的段禁止删除
- 清理后台执行
- 清理不阻塞写入主路径

## 12. 运维工具

第一版必须提供最小运维工具集，而不是只写“可人工巡检”。

建议提供以下 CLI：

1. `cachectl stats`
   - 输出总体运行统计和磁盘占用
2. `cachectl inspect-segment <id>`
   - 输出段头、段尾、记录数、时间范围、顺序号范围
3. `cachectl inspect-wal`
   - 输出 WAL 大小、最后有效批次、损坏位置
4. `cachectl inspect-cursor <destination>`
   - 输出主游标、备份游标、校验状态
5. `cachectl verify`
   - 扫描所有段和 WAL，输出损坏报告
6. `cachectl repair-tail`
   - 截断损坏尾部并生成修复报告
7. `cachectl benchmark`
   - 执行 `10000` 条读写基准
8. `cachectl close-check`
   - 验证上次关闭是否完整、是否存在未完成 flush

这些工具的输出应支持：

- 终端摘要
- JSON 机器可读格式
- 落盘报告

## 13. 监控指标与 Stats 接口

`Stats()` 不能只返回一个模糊对象，第一版至少定义如下字段：

```text
storage_bytes_total
storage_bytes_segments
storage_bytes_wal
segment_count_total
segment_active_id
records_written_total
batches_written_total
records_replayed_total
write_latency_p50_ms
write_latency_p95_ms
replay_latency_p50_ms
replay_latency_p95_ms
wal_replay_count_total
wal_truncate_count_total
checkpoint_count_total
checkpoint_last_duration_ms
segment_fsync_count_total
segment_fsync_last_duration_ms
cursor_corruption_count_total
segment_tail_repair_count_total
clock_rollback_events_total
oldest_unacked_event_time_unix_ms
oldest_unacked_write_seq
disk_free_bytes
disk_free_ratio
write_reject_count_disk_full
retention_blocked_segments
last_successful_checkpoint_unix_ms
last_successful_shutdown_unix_ms
graceful_shutdown_count_total
ungraceful_shutdown_recoveries_total
```

`Stats()` 返回结构按 4 类组织：

- `CapacityStats`
- `IOStats`
- `ReplayStats`
- `HealthStats`

## 14. 第一版交付范围

### 14.1 包含

- Go 本地库
- 批量写入原始字节流
- 二进制流段文件
- WAL
- 按大小滚动
- 单目标端回放与确认
- 游标双文件保护
- 启动恢复
- 30 天保留策略
- 运维 CLI
- benchmark 与基础测试

### 14.2 不包含

- 压缩
- 多目标端同步位点
- 复杂查询索引
- 按 payload 内容过滤
- 在线重排或重写旧段
- 归档

## 15. 风险与权衡

- 只存原始字节流意味着 payload 可扩展性强，但系统无法理解其业务语义
- 不做复杂索引能守住顺序写性能，但随机查询能力弱
- 双游标文件可提高恢复成功率，但上游接收端仍应支持幂等
- 以 `writeSeq` 保证顺序，会让“按采集时间严格排序”成为上层语义，而不是存储层语义

## 16. 结论

本方案现在明确为：

- 原始业务数据只按字节流存储
- 存储层只维护最小公共头，不绑定业务字段
- 通过 `writeSeq` 保证时间回拨场景下的稳定顺序回放
- 通过 WAL + 按大小滚动的 Segment 实现顺序高吞吐写入
- 通过双游标文件、校验和运维工具保证可恢复和可维护

这版设计修正了前一版的过度具体化问题，并补齐了游标恢复、时间回拨、运维工具和监控指标定义。
