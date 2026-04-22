# Linux / Windows 压测与 CLI 使用文档

## 1. 目的

这份文档用于指导在 Linux 和 Windows 环境下对本项目进行压测，并说明当前 3 个 CLI 工具的使用方式：

- `cachebench`
- `cachesim`
- `cachectl`

适用对象：

- 开发人员
- 测试人员
- 现场验证人员
- 需要验证目录级真实落盘效果的运维人员

## 2. 工具分工

先明确 3 个工具的职责，不要混用：

### `cachebench`

作用：

- 合成 benchmark
- 快速验证 `<100ms / 10000 条` 的 benchmark contract
- 使用临时目录，不保留真实工作区

适合：

- 快速测写入/回放延迟
- CI 或开发机上的基线性能检查

不适合：

- 观察真实 `segments/`、`wal/`、`meta/` 目录结构
- 做目录级现场负载模拟

### `cachesim`

作用：

- 面向现场导向负载的离线压测
- 写入你指定的目录
- 真实生成 `meta/`、`wal/`、`segments/`

适合：

- 模拟工业采集现场的目录级写入
- 观察段文件生成情况
- 验证工作区大小、段文件数、WAL 文件数

不适合：

- 长时间持续运行模拟
- 实时采集场景回放

### `cachectl`

作用：

- 对已经生成的工作区进行检查、统计和修复

适合：

- 查看 stats
- 查看 WAL / cursor / segment
- 校验工作区
- 做段尾修复

## 3. 环境要求

## 3.1 通用要求

- Go 版本与仓库 `go.mod` 一致
- 本地可以正常执行 `go test ./...`
- 压测目录所在磁盘有足够空间

建议：

- 压测目录不要放在系统盘
- 压测目录不要放在网络盘
- 同一轮测试内，不要复用非空目录

## 3.2 Linux 环境

推荐准备：

- `bash` 或兼容 shell
- `go`
- 足够空闲的本地磁盘空间

推荐先验证：

```bash
go version
go test ./... -count=1
```

如果项目环境提供 `rtk`，也可以使用：

```bash
rtk go test ./... -count=1
```

## 3.3 Windows 环境

推荐准备：

- PowerShell
- Go
- 本地 NTFS 磁盘目录作为测试目录

推荐先验证：

```powershell
go version
go test ./... -count=1
```

说明：

- Windows 下路径请优先使用本地磁盘目录，例如 `D:\cache-test`
- 不建议直接把压测目录放在 OneDrive 同步目录里

## 4. 压测前准备

无论 Linux 还是 Windows，都建议先做 3 件事：

1. 确认工作目录是仓库根目录
2. 确认测试目录不存在，或为空目录
3. 确认当前机器没有其他高负载任务在抢占磁盘

建议准备两个目录：

- 一个给 `cachesim` 做真实目录压测
- 一个给后续 `cachectl` 检查

## 5. Linux 环境压测流程

## 5.1 快速基准压测

使用 `cachebench`：

```bash
rtk go run ./cmd/cachebench --records 10000 --payload-bytes 32
```

如果没有 `rtk`：

```bash
go run ./cmd/cachebench --records 10000 --payload-bytes 32
```

典型输出：

```text
records=10000 payload_bytes=32 write_duration=13.8ms replay_duration=45.3ms replayed=10100 target_note=<100ms on recommended hardware; CI threshold can be overridden via FASTREADFILE_BENCHMARK_MAX_DURATION>
```

重点看：

- `write_duration`
- `replay_duration`

判定方式：

- 如果两者都明显低于 `100ms`，说明当前 benchmark contract 通过

## 5.2 真实目录离线压测

使用 `cachesim`：

```bash
rtk go run ./cmd/cachesim --root /tmp/cache-sim
```

大型 profile：

```bash
rtk go run ./cmd/cachesim --root /tmp/cache-sim-large --profile large
```

如果没有 `rtk`：

```bash
go run ./cmd/cachesim --root /tmp/cache-sim
go run ./cmd/cachesim --root /tmp/cache-sim-large --profile large
```

可选批大小：

```bash
go run ./cmd/cachesim --root /tmp/cache-sim --batch-size 2000
```

说明：

- 默认 profile 是 `medium`
- `large` 必须手动指定
- `--root` 只能是空目录或不存在目录

## 5.3 查看生成结果

压测结束后查看目录：

```bash
find /tmp/cache-sim -maxdepth 2 -type f | sort
du -sh /tmp/cache-sim
```

预期能看到：

- `meta/checkpoint.meta`
- `meta/lifecycle.state`
- `segments/*.seg`
- `wal/active.wal`

## 5.4 对工作区做检查

```bash
go run ./cmd/cachectl --root /tmp/cache-sim stats --format json
go run ./cmd/cachectl --root /tmp/cache-sim inspect-wal
go run ./cmd/cachectl --root /tmp/cache-sim verify
go run ./cmd/cachectl --root /tmp/cache-sim close-check
```

如果要检查某个段：

```bash
go run ./cmd/cachectl --root /tmp/cache-sim inspect-segment --segment 1
```

## 6. Windows 环境压测流程

## 6.1 快速基准压测

PowerShell 中执行：

```powershell
go run .\cmd\cachebench --records 10000 --payload-bytes 32
```

如果你的环境有 `rtk`：

```powershell
rtk go run .\cmd\cachebench --records 10000 --payload-bytes 32
```

重点仍然看：

- `write_duration`
- `replay_duration`

## 6.2 真实目录离线压测

中型默认场景：

```powershell
go run .\cmd\cachesim --root D:\cache-sim
```

大型场景：

```powershell
go run .\cmd\cachesim --root D:\cache-sim-large --profile large
```

指定批大小：

```powershell
go run .\cmd\cachesim --root D:\cache-sim --batch-size 2000
```

注意：

- `D:\cache-sim` 不应为已有非空目录
- 如果目录已存在且非空，工具会直接报错退出

## 6.3 查看目录结果

```powershell
Get-ChildItem D:\cache-sim -Recurse
```

查看总大小：

```powershell
Get-ChildItem D:\cache-sim -Recurse -File | Measure-Object -Property Length -Sum
```

## 6.4 对工作区做检查

```powershell
go run .\cmd\cachectl --root D:\cache-sim stats --format json
go run .\cmd\cachectl --root D:\cache-sim inspect-wal
go run .\cmd\cachectl --root D:\cache-sim verify
go run .\cmd\cachectl --root D:\cache-sim close-check
```

检查指定段：

```powershell
go run .\cmd\cachectl --root D:\cache-sim inspect-segment --segment 1
```

## 7. `cachebench` CLI 使用说明

命令：

```bash
go run ./cmd/cachebench --records 10000 --payload-bytes 32
```

参数：

- `--records`
  - 含义：写入记录数
  - 默认：`10000`
- `--payload-bytes`
  - 含义：每条记录的 payload 字节数
  - 默认：`32`

输出字段：

- `records`
- `payload_bytes`
- `write_duration`
- `replay_duration`
- `replayed`

使用建议：

- 用它验证基准契约
- 不要用它代替真实目录压测

## 8. `cachesim` CLI 使用说明

命令：

```bash
go run ./cmd/cachesim --root /tmp/cache-sim
go run ./cmd/cachesim --root /tmp/cache-sim-large --profile large
```

参数：

- `--root`
  - 含义：目标工作区目录
  - 必填
  - 必须是空目录或不存在目录
- `--profile`
  - 可选值：`medium`、`large`
  - 默认：`medium`
- `--batch-size`
  - 含义：每次 `WriteBatch` 写入的记录数
  - 默认：`1000`

当前 profile：

- `medium`
  - `20` 个网关
  - 每网关 `2000` 点
  - `5` 轮
  - 共 `200000` 条
- `large`
  - `50` 个网关
  - 每网关 `5000` 点
  - `3` 轮
  - 共 `750000` 条

输出字段：

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

使用建议：

- 中型压测优先用默认 profile
- 大型压测必须显式指定 `--profile large`
- 如果要比较批大小对写入表现的影响，再调整 `--batch-size`

## 9. `cachectl` CLI 使用说明

通用格式：

```bash
go run ./cmd/cachectl --root <workspace> <subcommand> [flags]
```

支持的子命令：

### `stats`

```bash
go run ./cmd/cachectl --root /tmp/cache-sim stats --format json
```

参数：

- `--format`
  - 默认：`text`
  - 可选：`json`

### `inspect-wal`

```bash
go run ./cmd/cachectl --root /tmp/cache-sim inspect-wal
```

### `inspect-cursor`

```bash
go run ./cmd/cachectl --root /tmp/cache-sim inspect-cursor --destination main-server
```

参数：

- `--destination`
  - 含义：目标回放端名称

### `inspect-segment`

```bash
go run ./cmd/cachectl --root /tmp/cache-sim inspect-segment --segment 1
```

参数：

- `--segment`
  - 含义：段号

### `verify`

```bash
go run ./cmd/cachectl --root /tmp/cache-sim verify
```

作用：

- 对工作区执行一致性检查

### `close-check`

```bash
go run ./cmd/cachectl --root /tmp/cache-sim close-check
```

作用：

- 检查工作区是否是“干净关闭”状态

### `repair-tail`

```bash
go run ./cmd/cachectl --root /tmp/cache-sim repair-tail --segment 1
```

作用：

- 对指定段执行保守型尾部修复

注意：

- `repair-tail` 是维护动作，不要在仍有活动写入时使用

## 10. 建议的压测顺序

推荐顺序：

1. 先跑 `cachebench`
2. 再跑 `cachesim`
3. 再用 `cachectl` 检查结果

对应命令：

```bash
go run ./cmd/cachebench --records 10000 --payload-bytes 32
go run ./cmd/cachesim --root /tmp/cache-sim
go run ./cmd/cachectl --root /tmp/cache-sim stats --format json
go run ./cmd/cachectl --root /tmp/cache-sim verify
```

这样可以分别回答 3 个问题：

1. benchmark contract 是否过关
2. 真实目录级工作区是否按预期生成
3. 生成后的工作区是否结构健康

## 11. 常见错误与处理

### `--root is required`

原因：

- `cachesim` 未传 `--root`

处理：

- 补上目标目录

### `root must be empty`

原因：

- `cachesim` 目标目录非空

处理：

- 更换一个新目录
- 或手工清空后再试

### `unknown simulator profile`

原因：

- `--profile` 不是 `medium` 或 `large`

处理：

- 改成合法 profile

### `missing subcommand`

原因：

- `cachectl` 没有提供子命令

处理：

- 补上 `stats`、`verify`、`inspect-wal` 等具体子命令

## 12. 推荐留档信息

每次正式压测建议保存：

- 操作系统版本
- Go 版本
- 测试目录路径
- 使用的命令行
- 输出结果
- 机器磁盘类型
- 提交 SHA

至少要保留：

- `cachebench` 输出
- `cachesim` 输出
- `cachectl verify` 输出

## 13. 当前限制

- `cachesim` 目前仅支持离线压测，不支持持续运行
- 暂不支持复用已有非空工作区继续写入
- `large` profile 适合显式人工触发，不应作为默认场景
- 当前文档默认从仓库根目录执行命令
