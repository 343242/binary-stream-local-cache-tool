# Binary Stream Local Cache Tool

一个基于 Go 的本地文件缓存工具，用于高吞吐断网缓冲场景。

它面向工业采集、边缘网关、数据转发等场景：当上游网络或主服务器不可用时，系统仍然持续接收数据，并将记录以二进制流形式顺序写入本地磁盘；待上游恢复后，再按物理写入顺序回放未确认数据。

## 适用场景

- 工业采集系统、边缘网关、协议转换器的本地断点缓存
- 主服务器故障、网络抖动、链路中断期间的数据暂存
- 强调顺序写、顺序回放、恢复安全，而不是复杂查询能力的场景

## 当前设计边界

- 单机
- 单进程
- 单写者
- 顺序存储
- 优先支持“回放未上传数据”
- Payload 不解析业务字段，只做二进制字节流持久化

它不是嵌入式数据库，也不是通用查询引擎。

## 核心能力

- 二进制流存储：底层记录结构是 `eventTime + writeSeq + payloadLen + CRC + payload`
- WAL 先行：每个 block 先写 WAL，再写 segment
- 按文件大小滚动：segment 达到 `target + slack` 后滚动到新文件
- 崩溃恢复：可重建 active segment 状态、保守修复段尾、重放尚未 durable 的 WAL
- 顺序回放：回放依据物理写入顺序与 `writeSeq`，不依赖传感器时间单调递增
- 游标持久化：每个 destination 有主游标和备份游标，附带 CRC 校验
- 保留清理：已 seal 且已确认的数据段可按保留期删除
- 运维工具：支持 stats、inspect-wal、inspect-cursor、inspect-segment、verify、repair-tail、close-check
- 基准工具：支持 `10000` 条级别的写入/回放延迟测量
- 离线模拟器：支持把现场导向的采集负载写入真实工作区，并生成真实 `meta/`、`wal/`、`segments/`

## 默认配置

对外配置类型是 [`cache.Config`](/home/instant/projects/fastReadFile/pkg/cache/config.go)，默认值定义在 [`internal/core/config.go`](/home/instant/projects/fastReadFile/internal/core/config.go)。

默认参数如下：

- `SegmentTargetSizeBytes = 256 MiB`
- `SegmentSlackSizeBytes = 4 MiB`
- `BlockTargetSizeBytes = 1 MiB`
- `CheckpointInterval = 5s`
- `CheckpointBytes = 64 MiB`
- `SegmentFsyncInterval = 100ms`
- `SegmentFsyncBytes = 4 MiB`
- `RetentionDays = 30`

## 目录结构

在 `RootDir` 下，运行时会生成如下目录：

```text
<root>/
  meta/
    checkpoint.meta
    lifecycle.state
    replay/
      <destination>.cursor
      <destination>.cursor.bak
  segments/
    000001.seg
    000002.seg
    ...
  wal/
    active.wal
```

## 耐久性流程

单次 `WriteBatch` 的关键流程如下：

1. 将输入记录组装成一个或多个二进制 block
2. 先把 block 追加到 WAL，并对 WAL 执行 `fsync`
3. 再把同一个 block 追加到当前 active segment
4. 按配置阈值或在 checkpoint / seal / close 时对 segment 执行 `fsync`
5. 只有在 active segment durable 之后，才会推进 checkpoint

这个顺序是当前实现最重要的安全约束之一。

## 对外 API

主要入口在 [`pkg/cache/engine.go`](/home/instant/projects/fastReadFile/pkg/cache/engine.go)。

核心方法：

- `cache.Open(cfg)`
- `engine.WriteBatch(ctx, []cache.RawRecord)`
- `engine.Replay(ctx, destination, limit)`
- `engine.Ack(ctx, destination, cursor)`
- `engine.Stats(ctx)`
- `engine.Recover(ctx)`
- `engine.Close()`
- `engine.Shutdown(ctx)`

核心数据类型：

- `RawRecord`
  - `EventTimeUnixMs`
  - `Payload`
- `ReplayLimit`
  - `MaxRecords`
  - `MaxBytes`
- `ReplayBatch`
  - 包含 `Records` 与 `NextCursor`
- `StatsSnapshot`
  - 包含 `Capacity / IO / Replay / Health`

## 快速开始

```go
package main

import (
	"context"
	"log"

	"fastReadFile/pkg/cache"
)

func main() {
	cfg := cache.DefaultConfig("/tmp/local-cache")

	engine, err := cache.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close()

	_, err = engine.WriteBatch(context.Background(), []cache.RawRecord{
		{EventTimeUnixMs: 1710000000000, Payload: []byte{0x01, 0x02, 0x03}},
		{EventTimeUnixMs: 1710000001000, Payload: []byte("sensor-frame")},
	})
	if err != nil {
		log.Fatal(err)
	}

	batch, err := engine.Replay(context.Background(), "main-server", cache.ReplayLimit{
		MaxRecords: 1000,
		MaxBytes:   4 << 20,
	})
	if err != nil {
		log.Fatal(err)
	}

	if batch.RecordCount > 0 {
		if _, err := engine.Ack(context.Background(), "main-server", batch.NextCursor); err != nil {
			log.Fatal(err)
		}
	}
}
```

## CLI 工具

### `cachectl`

[`cmd/cachectl/main.go`](/home/instant/projects/fastReadFile/cmd/cachectl/main.go) 提供基础运维能力：

```bash
rtk go run ./cmd/cachectl --root /tmp/local-cache stats --format json
rtk go run ./cmd/cachectl --root /tmp/local-cache inspect-wal
rtk go run ./cmd/cachectl --root /tmp/local-cache inspect-cursor --destination main-server
rtk go run ./cmd/cachectl --root /tmp/local-cache inspect-segment --segment 1
rtk go run ./cmd/cachectl --root /tmp/local-cache verify
rtk go run ./cmd/cachectl --root /tmp/local-cache repair-tail --segment 1
rtk go run ./cmd/cachectl --root /tmp/local-cache close-check
```

### `cachebench`

[`cmd/cachebench/main.go`](/home/instant/projects/fastReadFile/cmd/cachebench/main.go) 用于基准压测：

```bash
rtk go run ./cmd/cachebench --records 10000 --payload-bytes 32
```

示例输出：

```text
records=10000 payload_bytes=32 write_duration=13.2ms replay_duration=42.1ms replayed=10100 target_note=<100ms on recommended hardware; CI threshold can be overridden via FASTREADFILE_BENCHMARK_MAX_DURATION>
```

这里要区分两件事：

- 产品目标仍然是 `<100ms / 10000 条`
- CI 中 benchmark 测试默认也按 `100ms` 判定
- 仅在极端噪声环境下，才允许通过环境变量覆盖 benchmark 阈值

### `cachesim`

[`cmd/cachesim/main.go`](/home/instant/projects/fastReadFile/cmd/cachesim/main.go) 用于指定目录的离线现场压测：

```bash
rtk go run ./cmd/cachesim --root /tmp/cache-sim
rtk go run ./cmd/cachesim --root /tmp/cache-sim --profile large
```

行为约束：

- `--root` 只能是不存在目录或空目录
- 默认 profile 是 `medium`
- `large` 必须显式指定
- 写入走真实引擎路径，并生成真实 `meta/`、`wal/`、`segments/`

示例输出：

```text
root=/tmp/cache-sim profile=medium gateways=20 points_per_gateway=2000 rounds=5 records=200000 payload_bytes=24 batch_size=1000 total_duration=... avg_batch_duration=... max_batch_duration=... segment_count=... wal_file_count=1 workspace_bytes=...
```

`cachebench` 用于合成 benchmark contract，`cachesim` 用于目录级真实负载生成。

## 测试

测试目录：

- `tests/unit`
- `tests/integration`
- `tests/benchmark`

推荐验证命令：

```bash
rtk go test ./... -count=1
rtk go test ./... -race -count=1
rtk go test ./tests/benchmark/... -count=1
```

## 当前保证

- 恢复过程中不会在 segment 尚未 durable 时提前截断 WAL
- 游标文件有 CRC 校验，并使用主文件 + 备份文件持久化
- 段尾修复采用保守策略：保留有效前缀，裁剪损坏尾部
- 发现写序号冲突时直接报错，不做猜测性恢复
- 即使传感器时间发生回拨，回放顺序仍保持物理写入顺序

## 限制与非目标

- 不支持多进程并发写入
- 不面向任意点位过滤、复杂查询或分析型读取
- 不提供多副本复制能力
- 不替代长期主存储系统，例如 TSDB / historian
- 尚未实现真实断电级黑盒测试，目前以子进程崩溃 + 恢复测试为主

## 容量规划说明

当前项目使用的容量规划假设是：

- 日常平均：`4000 条/秒`
- 峰值：`10000 条/秒`
- 保留期：`30 天`
- 示例 payload：`50 字节/条`

仅 payload 体量大致为：

- 平均场景：`518.4 GB`
- 峰值持续场景：`1.296 TB`

真实磁盘规划还需要额外考虑：

- WAL 开销
- block / footer / metadata 开销
- segment slack
- 文件系统预留空间
- 安全冗余

因此实际部署时，不应仅按 payload 体量去估算磁盘。

## 错误类型

对外暴露的错误码见 [`pkg/cache/errors.go`](/home/instant/projects/fastReadFile/pkg/cache/errors.go)，包括：

- `ERR_VALIDATION`
- `ERR_DISK_FULL`
- `ERR_READ_ONLY_FS`
- `ERR_IO`
- `ERR_CORRUPTION`
- `ERR_CURSOR_INVALID`
- `ERR_CURSOR_CORRUPTED`
- `ERR_SEQUENCE_CONFLICT`
- `ERR_WAL_INVALID`
- `ERR_TIMEOUT`

## 模块路径说明

当前 `go.mod` 的模块路径是 `fastReadFile`。如果你准备直接以 GitHub 仓库形式对外发布或让其他仓库通过 `go get` 引用，需要先把模块路径调整为与你的仓库地址一致。
