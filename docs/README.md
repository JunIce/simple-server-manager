# service-manager 技术文档

基于 Go 实现的 Windows 服务管理系统。本文档集涵盖产品需求、系统架构、模块实现细节与架构决策记录。

> 设计约束：不引入分布式/agent、不依赖任何数据库、不使用日志全文索引（ES/bleve），日志检索采用 grep/sed 式行匹配，配置以 JSON 文件持久化到本地或用户目录。

## 文档导航

| 文档 | 说明 |
|------|------|
| [prd.md](../prd.md) | 产品需求文档（PRD），功能范围、用户故事、验收标准 |
| [docs/architecture.md](architecture.md) | 系统架构：C4 模型、模块划分、部署拓扑、数据流 |
| [docs/features.md](features.md) | 功能文档：各功能模块的行为定义与交互 |
| [docs/adr/](adr/) | 架构决策记录（ADR） |
| [docs/modules/](modules/) | 各模块实现细节 |

## 模块实现细节

| 模块 | 路径 | 文档 |
|------|------|------|
| 核心服务管理 | `internal/core/` | [modules/core.md](modules/core.md) |
| 平台适配层 | `internal/platform/` | [modules/platform.md](modules/platform.md) |
| 服务类型处理器 | `internal/handlers/` | [modules/handlers.md](modules/handlers.md) |
| 运行时管理 | `internal/runtime/` | [modules/runtime.md](modules/runtime.md) |
| 监控模块 | `internal/monitor/` | [modules/monitor.md](modules/monitor.md) |
| 日志管理 | `internal/logger/` | [modules/logger.md](modules/logger.md) |
| HTTP API | `internal/api/` | [modules/api.md](modules/api.md) |
| 配置管理 | `internal/config/` | [modules/config.md](modules/config.md) |
| 进程管理库 | `pkg/process/` | [modules/process.md](modules/process.md) |
| 通用工具 | `pkg/utils/` | [modules/utils.md](modules/utils.md) |

## 架构决策记录

| ADR | 标题 |
|-----|------|
| [ADR-001](adr/0001-go-language.md) | 选择 Go 而非 Java/Rust |
| [ADR-002](adr/0002-platform-abstraction.md) | 平台适配层抽象 |
| [ADR-003](adr/0003-log-search.md) | 日志检索采用 grep/sed 式行匹配 |
| [ADR-004](adr/0004-websocket.md) | WebSocket 实时通信 |
| [ADR-005](adr/0005-process-management.md) | 进程 vs 系统服务双模式管理 |
| [ADR-006](adr/0006-json-persistence.md) | 配置以 JSON 文件持久化，不引入数据库 |
| [ADR-007](adr/0007-single-node-windows.md) | 单机 Windows 优先，不做分布式 |
| [ADR-008](adr/0008-runtime-management.md) | 运行时/JDK 多版本管理 |

## Web UI

前端基于 **React + Vite + TailwindCSS**，源码位于 `web/`，构建产物嵌入二进制（`internal/webui/static`）。

| 页面 | 功能 |
|------|------|
| 主机监控 | CPU/内存/磁盘/进程数实时仪表盘（WebSocket `/ws/system/stats`） |
| 服务管理 | 服务列表、注册/编辑/启停/重启/删除、指标展示 |
| 日志 | 实时流 + grep/sed 式关键字检索 |
| 运行时(JDK) | JDK 自动发现/注册/删除，服务绑定不同版本 |

构建：`cd web && npm install && npm run build`
