# 系统架构

> 目标平台：**Windows**；单机部署，无分布式、无数据库、无日志索引。

## 1. C4 模型

### 1.1 System Context（系统上下文）

```
┌────────────┐      REST/WebSocket      ┌──────────────────┐      exec/Windows API     ┌──────────────┐
│  运维/开发  │ ───────────────────────▶ │  service-manager │ ───────────────────────▶ │  被托管服务    │
│   客户端    │                          │  (Go 单二进制)    │                           │ Java/MySQL/..│
└────────────┘                          └──────────────────┘                           └──────────────┘
                                                    │
                                                    ▼
                                          ┌──────────────────┐
                                          │   JSON 配置文件   │
                                          │ (本地/用户目录)    │
                                          └──────────────────┘
```

- 管理端（server）是唯一对外服务入口，监听 HTTP 端口。
- 通过 `os/exec` + Windows API 直接管理子进程，通过 Windows SCM 管理系统服务。
- 配置与服务状态以 JSON 文件持久化。

### 1.2 Container（容器）视图

单个 Go 二进制，内部逻辑分层：

```
┌──────────────────────────────────────────────────────────┐
│                        HTTP Server                        │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────┐  │
│  │ REST API   │  │ WebSocket  │  │  静态资源(web/)     │  │
│  └─────┬──────┘  └─────┬──────┘  └────────────────────┘  │
└────────┼───────────────┼──────────────────────────────────┘
         │               │
         ▼               ▼
┌──────────────────────────────────────────────────────────┐
│                      internal/api (路由层)                │
└───────────────────────────┬──────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│  internal/core │   │internal/logger│   │internal/monitor│
│  服务管理器     │   │  日志管理      │   │   监控模块     │
└───────┬───────┘   └───────────────┘   └───────────────┘
        │
        ▼
┌───────────────┐   ┌───────────────┐
│internal/handlers│ │internal/platform│
│  类型处理器     │   │ 平台适配层     │
└───────────────┘   └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │  pkg/process  │
                    │  进程管理库     │
                    └───────────────┘
```

### 1.3 Component（组件）视图

每个模块的内部组件详见 [modules/](modules/) 目录。核心依赖关系：

```
api ──▶ core ──▶ handlers ──▶ platform ──▶ process
 │         │                        │
 │         ├──▶ supervisor          └──▶ SCM / Job Object (windows)
 │         └──▶ monitor
 └──▶ logger ◀── core
```

依赖方向单向向下，避免循环依赖：`api → core → {handlers, monitor, logger} → platform → process`。

## 2. 目录结构

```
service-manager/
├── cmd/
│   └── server/            # 主服务入口
│       └── main.go
├── internal/              # 内部包，不对外导出
│   ├── core/              # 核心服务管理
│   ├── platform/          # 平台适配
│   ├── handlers/          # 服务类型处理器
│   ├── monitor/           # 监控
│   ├── logger/            # 日志管理
│   ├── api/               # HTTP API
│   └── config/            # 配置管理
├── web/                   # 前端静态资源与模板
├── pkg/                   # 可复用公共库
│   ├── process/           # 进程管理
│   └── utils/             # 工具函数
└── go.mod
```

## 3. 部署拓扑（单机）

```
┌───────────────────────────── Windows Host ─────────────────────────────┐
│                                                                        │
│   ┌────────────────┐    管理子进程     ┌────────────────────┐           │
│   │ service-manager│ ───────────────▶ │ 被托管服务 (×N)     │           │
│   │   (server)     │                  │ java / mysql / ...  │           │
│   └───────┬────────┘                  └────────────────────┘           │
│           │ JSON                                                       │
│           ▼                                                            │
│   ┌────────────────┐                                                   │
│   │  config.json    │  (%LOCALAPPDATA%\service-manager\config.json)    │
│   └────────────────┘                                                   │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

## 4. 关键数据流

### 4.1 启动服务

```
client → api.StartService → core.StartService
  → handlers.BuildCommand (类型翻译)
  → platform.StartProcess (Windows 适配 + 资源限制)
  → process 启动子进程
  → logger.Subscribe (日志采集)
  → monitor.Watch (指标采集)
  → supervisor.Watch (守护)
  → 返回 status=running
```

### 4.2 实时日志推送

```
托管进程 stdout/stderr
  → pipe
  → LogCollector.Start (scanner)
  → 环形缓冲 buffer
  → broadcast 订阅者 (chan LogEntry)
  → WebSocket 推送客户端
```

### 4.3 日志检索（grep/sed 式）

```
GET /api/services/{id}/logs?keyword=error
  → LogSearcher.Search
  → 定位该服务的日志文件（按时间范围裁剪滚动文件集合）
  → bufio.Scanner 逐行流式匹配（关键字/正则/级别）
  → 命中收集，达到 maxResults 提前终止
  → 返回倒序结果
```

### 4.4 健康检查与自动重启

```
monitor.Watch (ticker 周期)
  → handlers.HealthCheck
  → 失败累计
  → supervisor 依据 RestartPolicy 退避重启
  → 达到上限 → status=failed
```

## 5. 并发模型

- **每服务一协程**：每个托管服务由独立的 `Supervisor` 与 `Monitor` goroutine 守护。
- **共享状态保护**：`ServiceManager.mu sync.RWMutex` 保护 `services map`。
- **日志广播**：`channel` 扇出（fan-out），订阅者独立 channel 避免慢消费者阻塞。
- **退避重启**：`restartCh` 信号 + `time.Sleep` 指数退避，支持外部手动重启打断。

## 6. 技术选型

| 维度 | 选择 | 理由 |
|------|------|------|
| 语言 | Go | 单二进制、低内存、强并发、跨平台编译 |
| HTTP 框架 | 标准库 `net/http` + 可选路由 | 依赖少、性能好 |
| WebSocket | `gorilla/websocket` | 成熟稳定 |
| 日志检索 | grep/sed 式行匹配（bufio.Scanner） | 零索引、零依赖、内存友好 |
| 日志落盘 | 按大小滚动文本文件 | 简单、可读 |
| 配置持久化 | JSON 文件 | 零数据库依赖、可读可迁移 |
| 进程管理 | `os/exec` + Windows API | 原生系统调用 |
| Windows 服务 | `golang.org/x/sys/windows/svc` | 标准 |
| 资源限制 | Windows Job Object | 标准 |
