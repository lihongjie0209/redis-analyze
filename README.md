# redis-analyze

Redis 内存分析 CLI 工具 — 连接 Redis 服务、扫描指定前缀的 key、统计各类型的数量和内存占用。

## 快速开始

```bash
# 安装
go install github.com/gituser/redis-analyze@latest

# 扫描全部 key
redis-analyze -H 127.0.0.1 -p 6379 -P '*'

# 扫描指定前缀
redis-analyze -H 127.0.0.1 -p 6379 -P 'user:*' -P 'session:*' --top 10

# JSON 输出
redis-analyze -H 127.0.0.1 -p 6379 -P '*' -f json
```

## 功能特性

- **安全扫描** — 使用 `SCAN` 命令迭代，不阻塞 Redis
- **多种部署模式** — 支持 Standalone / Cluster / Sentinel
- **ACL 认证** — Redis 6.0+ username + password
- **类型分组** — 按 string / hash / list / set / zset / stream 分组统计
- **前缀分组** — 通过分隔符自动推导命名空间
- **Top N 大 key** — 自动排序并展示最大 key
- **渐进式报告** — `--report-interval` 定时输出中间结果
- **批量 pipeline** — `--batch-size` 控制并发查 key 信息
- **自动降级** — 批量失败时自动回退到逐 key 模式
- **三种输出格式** — table（默认，彩色终端）/ json / csv

## 输出示例

```
  Redis Memory Analysis Report
  ───────────────────────────────────────────────────────

  Host: 127.0.0.1:6379 DB: 0  Mode: standalone  Duration: 14.5s
  Redis: 7.4.9  OS: Linux x86_64  Uptime: 2d
  Prefixes: *  Scanned: 1,000,000 keys

  ── Summary ──
    Total Keys:     1,000,000     Total Memory:   214.5 MB
    Avg Key Size:   224 B         Max Key Size:   568 B

  ── By Type ──
  TYPE   |  COUNT   |  MEMORY   |  AVG   |  MIN   |  MAX
  -------+----------+-----------+--------+--------+--------
  hash   |  250,000 |  90.8 MB  |  380 B |  376 B |  384 B
  string |  250,000 |  85.6 MB  |  358 B |  112 B |  568 B
  list   |  250,000 |  19.1 MB  |  79 B  |  72 B  |  80 B
  set    |  250,000 |  19.1 MB  |  79 B  |  64 B  |  80 B

  ── By Prefix ──
  PREFIX  |  COUNT   |  MEMORY   |  TYPES
  --------+----------+-----------+--------------
  profile |  250,000 |  90.8 MB  |  hash:250000
  user    |  250,000 |  85.6 MB  |  string:250000
  session |  250,000 |  19.1 MB  |  list:250000
  tags    |  250,000 |  19.1 MB  |  set:250000

  ── Top Largest Keys ──
  # |  KEY         |  TYPE   |  SIZE  |  IDLE
  ---+--------------+---------+--------+-------
  1 |  user:110137 |  string |  568 B |  6m
```

## 全部参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--host` | `-H` | `127.0.0.1` | Redis 主机 |
| `--port` | `-p` | `6379` | Redis 端口 |
| `--password` | `-a` | `""` | 密码 |
| `--username` | `-u` | `""` | ACL 用户名（Redis 6.0+） |
| `--db` | `-n` | `0` | 数据库编号（standalone） |
| `--mode` | `-m` | `standalone` | 部署模式：`standalone`, `cluster`, `sentinel` |
| `--addrs` | | | Cluster/Sentinel 节点列表（逗号分隔） |
| `--master-name` | | `mymaster` | Sentinel master 名称 |
| `--sentinel-password` | | `""` | Sentinel 独立密码 |
| `--prefix` | `-P` | `*` | Key 前缀模式（可多次指定） |
| `--scan-mode` | | `auto` | 扫描策略：`auto`, `pipeline`, `sequential` |
| `--batch-size` | | `50` | 每批 pipeline 查询的 key 数 |
| `--samples` | `-s` | `5` | MEMORY USAGE 采样数 |
| `--top` | `-t` | `20` | Top N 大 key 展示数 |
| `--format` | `-f` | `table` | 输出格式：`table`, `json`, `csv` |
| `--separator` | | `:` | 前缀分组分隔符 |
| `--depth` | `-d` | `1` | 前缀分组深度 |
| `--timeout` | | `30` | 连接/扫描超时（秒） |
| `--report-interval` | | `0` | 中间报告间隔（秒，0=禁用） |
| `--no-progress` | | `false` | 隐藏进度条 |
| `--tls` | | `false` | 启用 TLS |

## 部署模式

### Standalone（默认）
```bash
redis-analyze -H 127.0.0.1 -p 6379 -P '*'
```

### Cluster
```bash
redis-analyze --mode cluster --addrs 10.0.0.1:6379,10.0.0.2:6379 -P '*'
```
每个主节点会被逐一 SCAN，结果自动合并。

### Sentinel
```bash
redis-analyze --mode sentinel --addrs 10.0.0.1:26379 --master-name mymaster -P '*'
```

## 扫描策略

三种策略按性能排序，`auto` 模式自动降级：

| 策略 | 速度 | 原理 |
|------|------|------|
| `pipeline` | 🚀 快 | 50 key 一批用 pipeline 查询 TYPE+MEMORY USAGE+IDLETIME |
| `sequential` | 🐢 慢 | 逐 key 查询，单 key 失败不影响其余 |
| `auto`（默认） | 🚀→🐢 | 先 pipeline，失败逐条降级 |

```bash
# 强制顺序模式（最稳定）
redis-analyze --scan-mode sequential

# 缩短 batch 以减少单次失败影响
redis-analyze --batch-size 10
```

## 定时中间报告

扫描大量 key 时可以定时输出阶段结果：

```bash
redis-analyze -H 127.0.0.1 -p 6379 -P '*' --report-interval 5
```

每 5 秒输出一次（到 stderr）：
```
  ═══════ Intermediate Report #2 ═══════
  Scanned: 700,250 keys  |  Elapsed: 10s
  Total Memory: 150.1 MB  |  Avg: 224 B  |  Max: 568 B
  By Type: hash: 174,881/63.5 MB, string: 174,781/59.8 MB, list: 175,334/13.4 MB, set: 175,254/13.4 MB
  Top Key: user:199298 (string, 568 B)
```

## 构建

### 本地编译
```bash
make build
./build/redis-analyze --help
```

### 跨平台静态编译
```bash
# 全部平台
make release

# 仅 Linux（x86_64 + ARM64）
make release-linux

# 仅 macOS（Intel + Apple Silicon）
make release-darwin

# 仅 Windows
make release-windows
```

所有二进制均为**完全静态链接**（`CGO_ENABLED=0`），不依赖 glibc / musl 等系统库，可在 alpine、busybox、任何 Linux 发行版上直接运行。

```bash
$ file build/redis-analyze-linux-*
redis-analyze-linux-amd64: ELF 64-bit LSB executable, x86-64, statically linked
redis-analyze-linux-arm64:  ELF 64-bit LSB executable, ARM aarch64, statically linked
```

### 快捷测试
```bash
# 启动本地 Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine

# 连接测试（默认 127.0.0.1:6379）
make test-redis

# 指定地址
make test-redis HOST=10.0.0.1 PORT=6379

# 对比两种扫描模式性能
make test-bench
```

## 技术原理

### 扫描架构

```
  ┌──────────────┐     SCAN cursor 0 MATCH * COUNT 100
  │   Scanner    │ ──────────────────────────────────►  Redis
  │              │ ◄──────────────────────────────────
  │  ┌────────┐  │     keys: [user:1, user:2, ...]
  │  │ Batch  │  │
  │  │  50 key │  │     pipeline: TYPE + MEMORY USAGE + OBJECT IDLETIME
  │  └────┬───┘  │ ──────────────────────────────────►  Redis
  │       │      │ ◄──────────────────────────────────
  │  ┌────▼───┐  │     [{type, size, idle}, ...]
  │  │Analyze │  │
  │  └────────┘  │     → Group by type + prefix → Top N
  └──────┬───────┘
         │
         ▼
   ┌────────────┐
   │  Reporter  │   table / json / csv
   └────────────┘
```

### 性能优化

1. **SCAN **❌** KEYS** — SCAN 是游标迭代，不阻塞 Redis；KEYS 会全量阻塞
2. **Pipeline 批量查询** — 50 key 一批，一次网络往返查 150 个命令（TYPE + MEMORY USAGE + IDLETIME × 50），而不是 50 次往返
3. **自动降级** — 批量失败自动逐 key 重试，不丢数据

## License

MIT
