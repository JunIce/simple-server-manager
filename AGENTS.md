# AGENTS.md

本文件为编码代理（AI agent）提供本仓库的工作约定。请先阅读本文再修改代码。

## 项目概览

`service-manager` 是一个基于 Go 的 **Windows 服务管理系统**，单机部署，用于统一托管 Java/MySQL/Redis/通用进程。

核心设计约束（见 `docs/adr/`）：
- **不做分布式**：无 agent、无多主机联邦（ADR-007）
- **零数据库**：配置以 JSON 文件持久化到本地或用户目录（ADR-006）
- **日志无索引**：grep/sed 式流式行匹配，不用 ES/bleve（ADR-003）
- **Windows 优先**：platform 接口层预留跨平台扩展（ADR-002）

## 关键命令

```powershell
# 后端：构建 + 编译检查
go build ./...
go vet ./...
go build -o service-manager.exe ./cmd/server

# 后端：启动（开发）
go run ./cmd/server server --port 8080

# 前端：开发模式（Vite 热更新，代理 /api、/ws 到 8080）
cd web
npm install
npm run dev

# 前端：生产构建（输出到 ../internal/webui/static，被 go:embed 打入二进制）
npm run build

# 跨平台编译检查（修改 platform 相关代码后必须验证）
$env:GOOS="linux"; $env:GOARCH="amd64"; go build ./...; Remove-Item Env:GOOS,Env:GOARCH
```

**注意**：`internal/webui/embed.go` 用 `//go:embed all:static`，若 `internal/webui/static` 目录不存在或为空，`go build ./...` 会直接失败。首次克隆或清空 `web/dist` 后，必须先 `cd web && npm run build` 再构建后端。

## 架构与目录

依赖方向单向向下，禁止循环依赖：

```
api ─▶ core ─▶ handlers ─▶ platform ─▶ process
           ├▶ supervisor
           ├▶ monitor ─▶ platform
           └▶ logger
```

| 路径 | 职责 |
|------|------|
| `cmd/server/` | 入口：server / run-service / install-service / uninstall-service 子命令 |
| `internal/core/` | 服务生命周期、Supervisor 守护、退避重启 |
| `internal/platform/` | 平台抽象（Windows 为主），进程/系统服务/资源/主机统计 |
| `internal/handlers/` | 服务类型处理器（java/mysql/redis/generic） |
| `internal/monitor/` | 服务指标 + 主机资源监控、订阅广播 |
| `internal/logger/` | 环形缓冲、滚动落盘、grep 式检索、实时推送 |
| `internal/api/` | REST + WebSocket + Web UI 静态服务 |
| `internal/config/` | 数据模型 + JSON 持久化（无 DB）+ JDK 版本管理与自动发现 |
| `internal/webui/` | `go:embed` 嵌入前端构建产物 |
| `pkg/process/` | 跨平台子进程管理（构建标签隔离） |
| `pkg/utils/` | 原子写、退避、日志级别解析等工具 |
| `web/` | React + Vite + TailwindCSS 前端源码 |

## 编码约定

- **语言**：Go（后端）、JavaScript/JSX（前端，React 18 + TailwindCSS）。
- **模型归属**：共享数据模型（`ServiceConfig`、`JDK`、`RestartPolicy` 等）定义在 `internal/config`，避免循环导入。
- **构建标签**：平台差异用 `//go:build windows` / `//go:build !windows` 隔离（`pkg/process`、`internal/platform`、`internal/handlers/alive_*.go`）。
- **平台接口**：新平台能力必须加到 `internal/platform.Platform` 接口，并在 `windows.go` 与 `platform_other.go` 同时实现（后者为桩）。
- **并发**：共享状态用 `sync.RWMutex`；日志/指标广播用独立 channel + 慢消费者丢弃。
- **服务长生命周期**：服务进程的日志采集与监控 goroutine 必须基于 manager 根上下文 `sm.rootCtx`，不得使用 HTTP 请求上下文（请求结束会被取消导致采集中断）。
- **子进程日志编码**：中文 Windows 上子进程（如 `ping.exe`）输出 GBK 而非 UTF-8。所有日志采集与检索统一经 `internal/logger/encoding.go` 的 `newTextDecoder()` 自动检测转码；新增读取日志流的代码必须走该解码器，不得直接 `bufio.Scanner` 裸读。且解码必须增量进行，禁止用 `Peek` 大块预读（会阻塞实时流，直到管道攒够数据）。
- **WebSocket 消息**：统一 `{type: "log"|"metrics"|"stats", data: {...}}` 帧结构，前端 `web/src/api.js` 的 `connectWS` 负责解析。
- **API 响应**：统一 `{code, message, data, requestId, timestamp}`；业务错误码约定见 `docs/modules/api.md`。
- **注释**：仅在有必要的说明处加中文注释，不写无意义注释。

## 常见改动模式

- **新增服务类型**：实现 `internal/handlers.Handler` 接口 → 在 `handlers.Get` 注册 → 在 `config.ServiceConfig` 增加专属配置字段。
- **新增运行时类型**（node/python）：JDK 版本统一在 `internal/config` 管理（`JDK` 模型、`DiscoverJDKs` 发现、`ResolveJDK` 解析），新增运行时类型时在此扩展分支。
- **新增 API 端点**：`internal/api/server.go` 注册路由 + `handlers.go` 实现 handler + `web/src/api.js` 添加封装。
- **新增 Web UI 页面**：`web/src/components/` 新建组件 → `web/src/App.jsx` 注册 tab。
- **新增平台能力**：`internal/platform/platform.go` 接口 + `windows.go` 实现 + `platform_other.go` 桩 + `docs/modules/platform.md` 更新。

## 验证清单（改动后必做）

1. `go build ./...` 通过
2. `go vet ./...` 通过
3. 修改 platform/process 后跑 Linux 交叉编译检查
4. 修改前端后 `cd web && npm run build`（确认 `internal/webui/static` 产物更新）
5. 修改 API/核心逻辑后启动服务做一次冒烟测试（注册→启动→查询状态/日志→停止→删除）

仓库无单元测试（无 `*_test.go`），验证靠冒烟测试；冒烟测试用 `curl` 发 JSON 时，Windows PowerShell 会转义破坏引号，建议把 JSON 写入临时文件后用 `curl --data-binary "@file"` 提交，且文件须为无 BOM 的 UTF-8（`Set-Content -Encoding UTF8` 会带 BOM 导致解析失败，用 `[System.IO.File]::WriteAllText`）。

## 文档

需求与设计文档位于 `docs/`（架构、功能、模块实现、ADR）。修改实现时保持 `docs/` 与代码一致；涉及新架构决策时新增 ADR 记录。

## 注意

- 配置文件的默认位置：`os.UserConfigDir()/service-manager/config.json`（Windows 为 `%APPDATA%\service-manager\config.json`，即 Roaming 目录，不是 LOCALAPPDATA）；日志默认落在 `./data/logs/`（工作目录）。
- JDK 版本通过统一配置页管理（`internal/config` 的 `JDK` 模型 + `DiscoverJDKs` 自动发现 + 三级 `ResolveJDK`），Java 服务用 `java.jdk` 引用 JDK ID。
- 不要提交 `web/node_modules`、`web/dist` 到版本库；`internal/webui/static` 是前端构建产物（由 `npm run build` 生成）。
