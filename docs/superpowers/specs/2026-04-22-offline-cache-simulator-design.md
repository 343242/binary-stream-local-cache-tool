# 面向现场负载的离线缓存压测器设计

## 1. 背景

当前项目已经具备两类能力：

- 存储引擎本体：通过 `cache.Open(...).WriteBatch(...)` 写入真实工作区
- 合成 benchmark：通过 `cmd/cachebench` 在临时目录中验证 `<100ms / 10000 条` 的基准契约

但这两者之间仍然缺一层“更接近工业采集现场”的压测工具。用户需要一个可以直接写入指定目录、真实生成 `meta/`、`wal/`、`segments/` 文件的离线压测器，用于模拟采集系统在现场环境下批量写入本地缓存的过程。

## 2. 目标

新增一个独立命令 `cachesim`，用于执行面向现场拓扑的离线压测：

- 写入指定根目录
- 仅允许不存在或空目录作为目标目录
- 通过真实引擎路径落盘，不伪造段文件
- 生成真实工作区结构与真实段文件
- 默认使用中型现场 profile
- 大型 profile 必须显式切换
- 输出可读的压测报告，方便快速判断目录规模与写入表现

## 3. 非目标

本次只实现离线压测器，不实现持续运行的实时模拟器。

本次不做以下内容：

- 不直接拼装 `.seg` / `.wal` 文件
- 不复用非空工作区继续写入
- 不引入 GUI 控制界面
- 不替换现有 `cachebench`
- 不改变存储引擎对外 API
- 不把业务字段扩展为引擎一级字段

## 4. 总体方案

新增独立命令：

```bash
rtk go run ./cmd/cachesim --root /tmp/cache-sim
rtk go run ./cmd/cachesim --root /tmp/cache-sim --profile large
```

总体结构：

1. `cmd/cachesim`
   - 解析 CLI 参数
   - 调用 `internal/simtool`
   - 输出报告
2. `internal/simtool/profile.go`
   - 定义中型 / 大型 profile
3. `internal/simtool/generator.go`
   - 按网关、点位、轮次生成采集记录
4. `internal/simtool/encoder.go`
   - 将结构化采集记录编码为固定长度二进制 payload
5. `internal/simtool/runner.go`
   - 校验目录
   - 打开引擎
   - 分批写入
   - 汇总写入耗时与工作区统计
6. `internal/simtool/report.go`
   - 生成人类可读报告

## 5. 为什么不扩展 `cachebench`

`cachebench` 的职责是合成 benchmark：

- 使用临时目录
- 输入模型是 `records + payload-bytes`
- 目标是验证 benchmark contract

`cachesim` 的职责是现场导向压测：

- 使用指定目录
- 输入模型是 `profile + 批量写入`
- 目标是生成真实工作区并观察现场级目录形态

将两者混在一个命令里会让 CLI 语义和报告语义变得含混，因此本设计采用独立命令。

## 6. 数据模型

逻辑采集记录定义为：

```go
type SensorSample struct {
    GatewayID     uint32
    PointID       uint32
    Value         float64
    CollectTimeMs int64
}
```

落盘时仍通过现有引擎接口写入：

```go
cache.RawRecord{
    EventTimeUnixMs: sample.CollectTimeMs,
    Payload:         EncodeSample(sample),
}
```

其中 `Payload` 使用固定长度二进制编码，按以下顺序编码：

1. `gateway_id` (`uint32`)
2. `point_id` (`uint32`)
3. `value` (`float64`)
4. `collect_time_ms` (`int64`)

固定长度为 `24` 字节。

这样定义的原因：

- 贴近原始业务字段
- 编码成本稳定
- 不把 JSON 序列化波动混入压测结果
- 仍然完全兼容当前引擎的“只存 payload，不解释业务字段”的边界

## 7. 现场 profile

### 7.1 默认 profile：`medium`

默认使用中型现场：

- `20` 个网关
- 每网关 `2000` 个点位
- `5` 轮采样

总记录数：

`20 * 2000 * 5 = 200000`

### 7.2 显式 profile：`large`

大型 profile 必须显式指定：

- `50` 个网关
- 每网关 `5000` 个点位
- `3` 轮采样

总记录数：

`50 * 5000 * 3 = 750000`

### 7.3 profile 原则

- 默认 profile 必须安全且常规
- 大型 profile 不自动触发，必须手动切换
- profile 决定“现场规模心智模型”，而不是要求用户自己计算总条数

## 8. 写入策略

压测器内部使用批量写入，不逐条写入。

默认批次大小：

- `batch_size = 1000`

写入顺序：

1. 按 `round` 遍历采样轮次
2. 每轮内按 `gateway_id` 遍历网关
3. 每个网关内按 `point_id` 遍历点位
4. 将样本编码为 `RawRecord`
5. 分批调用 `WriteBatch`

时间戳规则：

- 每轮对应一个采样时刻
- 同一轮内的所有点位共享同一基础采样时间
- 下一轮时间戳按固定步长推进

默认采样间隔仅用于构造合理时间序列，不用于控制墙钟节奏。

## 9. 目录安全规则

`--root` 只允许两种情况：

1. 目录不存在
   - 允许创建
2. 目录存在但为空
   - 允许使用

以下情况一律拒绝：

- 路径存在且非目录
- 目录存在但非空
- 路径不可写

本次不支持复用已有工作区，不支持“继续写入已有目录”。

## 10. 报告输出

压测结束后输出：

- `root`
- `profile`
- `gateways`
- `points_per_gateway`
- `rounds`
- `records`
- `payload_bytes`
- `batch_size`
- `total_duration`
- `avg_batch_duration`
- `max_batch_duration`
- `segment_count`
- `wal_file_count`
- `workspace_bytes`

报告目标：

- 让调用者快速判断目录是否已按预期生成
- 让调用者快速判断写入总耗时与批次表现
- 不把输出做成复杂报表

## 11. 错误处理

错误要可直接定位，至少覆盖：

- 缺少 `--root`
- 非法 profile
- root 为文件
- root 非空
- root 不可写
- 引擎打开失败
- 批次写入失败
- 工作区统计失败

报错原则：

- 优先指出是哪个路径、哪个 profile、哪个阶段失败
- 不做静默跳过
- 一旦失败立即退出并返回非零码

## 12. 测试要求

至少需要以下测试：

1. profile 解析测试
   - 默认 `medium`
   - `large` 显式切换
2. 编码测试
   - 固定长度 `24` 字节
   - 字段顺序稳定
3. 目录校验测试
   - 不存在目录可用
   - 空目录可用
   - 非空目录拒绝
   - 文件路径拒绝
4. 集成测试
   - 写完后工作区存在 `meta/`、`wal/`、`segments/`
   - 至少生成一个 `.seg`
   - 报告中的记录数与 profile 计算结果一致

## 13. 后续扩展点

本设计为后续实时模拟器预留复用点：

- 复用 `profile`
- 复用 `SensorSample` 生成器
- 复用编码器
- 复用报告与工作区统计逻辑

未来实时模拟器只需要补：

- 墙钟节奏控制
- 停止信号处理
- 周期性输出当前状态

## 14. 结论

本次采用独立 `cachesim` 命令实现面向现场负载的离线压测。

它与 `cachebench` 的分工明确：

- `cachebench` 继续负责合成 benchmark contract
- `cachesim` 负责现场导向、指定目录、真实段文件生成的离线压测

这样可以在不改变引擎 API 的前提下，补齐“真实目录级压测”这一关键能力。
