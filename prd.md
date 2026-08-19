# service-manager 服务管理系统 PRD

## 1. 文档信息

| 字段 | 内容 |
|------|------|
| 产品名称 | service-manager（服务管理系统） |
| 版本 | v1.0 |
| 状态 | Draft |
| 技术栈 | Go 1.21+ |
| 目标平台 | Windows（运维主场景） |
| 关联文档 | docs/architecture.md（架构）、docs/modules/（模块实现细节）、docs/adr/（架构决策记录） |

## 2. 背景与目标

### 2.1 业务背景
在 Windows 主机上，运维与开发团队需要统一管理大量异构后台服务（Java、MySQL、Redis、自定义脚本等）。现有方案存在以下痛点：

- 依赖 JVM 的解决方案资源占用高（常驻 200–500MB），不适合作为轻量常驻服务。
- 各服务启动方式、健康检查、日志格式不统一，缺乏统一的控制入口。
- 日志分散在文件/终端中，无集中检索与实时流式查看能力。
- 配置与状态缺乏轻量、可靠的本地持久化（不引入数据库）。

### 2.2 用户痛点
- **运维**：需要在 Windows 主机上统一注册、启停、重启、查看状态，快速定位异常日志。
- **开发**：需要在多环境（dev/test/prod）间切换配置，实时查看服务日志与指标。

### 2.3 产品目标（SMART）
- 将单节点服务管理的内存常驻控制在 **20–50MB**（对比 Java 方案降低一个数量级）。
- 单个管理实例在 Windows 上可管理 **数百个** 服务，启停延迟 < 1s。
- 提供 **REST API + WebSocket**，实时日志推送延迟 < 500ms。
- 配置与服务状态以 **JSON 文件** 持久化到本地或用户目录，**不依赖任何数据库**。

### 2.4 成功指标
| 指标 | 目标值 |
|------|--------|
| 常驻内存 | ≤ 50MB |
| 服务注册到启动完成 | < 1s |
| 实时日志推送延迟 | < 500ms |
| 服务状态监控采集周期 | 可配置，默认 5s |
| 主机资源采集周期 | 可配置，默认 5s |
| 日志关键字检索（grep 式） | 秒级返回 |

## 3. 术语定义

| 术语 | 说明 |
|------|------|
| ManagedService | 被管理系统托管的服务实例 |
| ServiceConfig | 服务的静态配置（命令、环境、重启策略等） |
| Supervisor | 服务守护/看护组件，负责重启与状态跟踪 |
| Platform | 平台适配层抽象，封装 OS 差异 |
| Handler | 服务类型处理器，将 ServiceConfig 翻译为启动命令 |
| Environment | 运行环境标记（dev / test / prod 等） |

## 4. 用户角色

| 角色 | 诉求 |
|------|------|
| 运维工程师 | 在 Windows 主机上注册/启停/重启服务，配置资源限制与重启策略 |
| 开发工程师 | 切换环境启动服务，实时查看日志与指标，快速定位问题 |
| 系统集成方 | 通过 REST API 将服务管理能力嵌入现有平台 |

## 5. 功能范围

### 5.1 本期范围（In Scope）
- 服务注册 / 注销 / 更新配置
- 服务生命周期管理：启动 / 停止 / 重启 / 状态查询
- 多环境支持与运行环境切换
- 自动重启与可配置重启策略（退避）
- 健康检查（HTTP / 进程存活 / 类型自定义）
- 资源限制（内存 / CPU）
- 日志实时收集、grep/sed 式快速检索与 WebSocket 实时流
- 指标采集（CPU / 内存 / 运行状态）
- **主机资源监控**：CPU / 内存 / 磁盘 / 进程数实时统计
- 服务类型处理器（Java / MySQL / Redis / 通用）
- 运行时管理：注册/发现多个 JDK 版本，不同服务可绑定不同 JDK 启动
- **Web UI**：基于 React + TailwindCSS 的控制台（主机监控 / 服务管理 / 日志 / 运行时）
- REST API + WebSocket 通信
- 配置与状态以 JSON 文件持久化到本地 / 用户目录

### 5.2 明确不做（Out of Scope）
- 分布式 / 多主机联邦管理（不引入 agent）
- 任何数据库依赖
- 日志全文索引（Elasticsearch / bleve 等）
- 细粒度权限体系（RBAC）与审计
- 告警通知（邮件 / 企业微信 / Webhook）
- Web 管理界面完善

## 6. 用户故事与验收标准

### 故事 1：运维注册并启动一个 Java 服务
> 作为运维工程师，我想要注册并启动一个 Java 服务，以便在 Windows 上统一托管并自动重启。

**场景：注册并启动**
- Given 系统中不存在 id 为 `my-java-app` 的服务
- When 我调用 `POST /api/services` 提交服务配置，随后调用 `POST /api/services/my-java-app/start`
- Then 服务进程被启动，`GET /api/services/my-java-app/status` 返回 `running`

**场景：配置持久化**
- Given 服务已注册
- When 管理进程重启后加载配置
- Then 服务配置从 JSON 文件恢复，无需重新注册

**场景：异常自动重启**
- Given 服务 `my-java-app` 已配置 `autoRestart: true` 且处于 `running`
- When 服务进程崩溃退出
- Then Supervisor 按重启策略在退避间隔后自动拉起进程，状态恢复为 `running`

### 故事 2：运维用不同 JDK 版本启动不同服务
> 作为运维工程师，我想要管理多个 JDK 版本并让不同服务绑定不同版本，以便兼容遗留应用。

**场景：注册多个 JDK 运行时**
- Given 主机上安装了 JDK 8 与 JDK 17
- When 我调用 `POST /api/runtimes/discover` 自动发现，或 `POST /api/runtimes` 手动注册
- Then 运行时列表包含 `jdk-1.8` 与 `jdk-17`，各自 `home` 正确

**场景：绑定指定 JDK 启动服务**
- Given 运行时 `jdk-1.8` 已注册
- When 我注册 Java 服务 `legacy-app`，其 `java.jdk = "jdk-1.8"`，并调用 start
- Then 服务使用 `C:\Program Files\Java\jdk1.8.0_401\bin\java.exe` 启动，进程 `JAVA_HOME` 指向该 JDK

**场景：引用不存在的运行时**
- Given 服务 `app` 的 `java.jdk` 指向未注册的 `jdk-99`
- When 我调用 start
- Then 返回错误码 `3001`（运行时不存在），服务不启动

### 故事 3：开发者实时查看与检索日志
> 作为开发工程师，我想要实时流式查看服务日志，并快速按关键字检索，以便快速定位问题。

**场景：实时日志流**
- Given 服务 `my-java-app` 正在运行并持续输出日志
- When 我通过 `ws://host/ws/logs?serviceId=my-java-app` 建立 WebSocket
- Then 客户端在 500ms 内收到新产生的日志条目

**场景：历史日志检索（grep 式）**
- Given 服务日志已落盘为文本文件
- When 我调用 `GET /api/services/my-java-app/logs?keyword=error&limit=100`
- Then 系统以 grep/sed 式行扫描快速匹配，按时间倒序返回最多 100 条

## 7. 功能详述

### 7.1 服务生命周期状态机

```
                    +-----------+
       register --->|  stopped  |
                    +-----+-----+
                          | start
                          v
                    +-----+-----+    crash/exit    +-----------+
                    |  running  |----------------->|  failed   |
                    +-----+-----+                  +-----+-----+
                          | stop                        | autoRestart
                          v                             v
                    +-----------+               (backoff 后重新 start)
                    |  stopped  |
                    +-----------+
```

状态枚举：`stopped` / `starting` / `running` / `stopping` / `failed` / `unknown`

### 7.2 服务注册流程
1. 客户端提交 `ServiceConfig`（命令、参数、环境、健康检查、资源限制等）。
2. 系统校验配置（命令非空、ID 唯一、类型处理器存在）。
3. 持久化配置到 JSON 文件（本地或用户目录）。
4. 返回注册结果与最终生效配置。

### 7.3 服务启动流程
1. 根据服务类型选择对应 Handler，构建启动命令与环境变量。
2. 通过 Platform 层启动进程（Windows 进程组 / 资源限制）。
3. 启动日志采集与指标监控协程。
4. Supervisor 开始守护，状态置为 `running`。

### 7.4 异常与边界情况
| 情况 | 处理 |
|------|------|
| 命令不存在 / 启动失败 | 状态置 `failed`，记录 `LastError`，返回明确错误 |
| 进程运行时崩溃 | Supervisor 依据 `RestartPolicy` 退避重启，超过上限置 `failed` |
| 重复启动 | 返回幂等成功或明确的状态冲突错误 |
| 停止超时 | 先发终止信号，超时后 `TerminateProcess` 强制终止 |
| 日志缓冲溢出 | 环形缓冲区覆盖最旧条目，避免内存无限增长 |

## 8. 数据模型与持久化

### 8.1 ServiceConfig（服务静态配置）
```go
type ServiceConfig struct {
    ID             string            `json:"id"`
    Name           string            `json:"name"`
    Type           ServiceType       `json:"type"` // java/mysql/redis/generic
    Command        string            `json:"command"`
    Args           []string          `json:"args"`
    WorkingDir     string            `json:"workingDir"`
    Environment    map[string]string `json:"environment"`
    Environments   []string          `json:"environments"` // dev, test, prod
    AutoRestart    bool              `json:"autoRestart"`
    RestartPolicy  RestartPolicy     `json:"restartPolicy"`
    HealthCheck    HealthCheckConfig `json:"healthCheck"`
    ResourceLimits ResourceLimits    `json:"resourceLimits"`
    LogConfig      LogConfig         `json:"logConfig"`
}
```

### 8.2 JavaConfig（Java 类型专属配置）

```go
type JavaConfig struct {
    JDK         string   `json:"jdk"`         // 引用运行时 ID，如 "jdk-17"；为空用默认
    JarFile     string   `json:"jarFile"`
    MainClass   string   `json:"mainClass"`
    JVMOptions  []string `json:"jvmOptions"`
    MemoryLimit string   `json:"memoryLimit"`
}
```

### 8.3 Runtime（运行时注册项）

```go
type Runtime struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    Kind    string `json:"kind"`    // 默认 "jdk"
    Version string `json:"version"`
    Home    string `json:"home"`
    Default bool   `json:"default"`
}
```

### 8.4 持久化格式（无数据库）
- 配置与服务状态统一以 **JSON 文件** 存储。
- 默认位置优先级：
  1. 启动参数 `--config` 指定路径；
  2. 系统用户目录：`%LOCALAPPDATA%\service-manager\config.json`（Windows）；
  3. 兜底：当前工作目录 `config.json`。
- 通过 `os.UserConfigDir()` / `os.UserHomeDir()` 定位用户目录。
- 原子写（临时文件 + rename），避免中途崩溃损坏配置。

## 9. API 规格

统一响应结构：
```json
{ "code": 0, "message": "success", "data": {}, "requestId": "uuid", "timestamp": 1715788800 }
```

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/services | 服务列表 |
| POST | /api/services | 注册服务 |
| GET | /api/services/{id} | 服务详情 |
| PUT | /api/services/{id} | 更新配置 |
| DELETE | /api/services/{id} | 注销服务 |
| POST | /api/services/{id}/start | 启动 |
| POST | /api/services/{id}/stop | 停止 |
| POST | /api/services/{id}/restart | 重启 |
| GET | /api/services/{id}/status | 状态 |
| GET | /api/services/{id}/logs | 日志检索（grep 式） |
| GET | /api/services/{id}/metrics | 指标 |
| GET | /api/system/info | 系统信息 |
| GET | /api/system/stats | 主机实时资源统计（CPU/内存/磁盘/进程） |
| GET | /api/runtimes | 运行时列表 |
| POST | /api/runtimes | 注册运行时 |
| POST | /api/runtimes/discover | 自动发现运行时（JDK） |
| GET | /api/runtimes/{id} | 运行时详情 |
| PUT | /api/runtimes/{id} | 更新运行时 |
| DELETE | /api/runtimes/{id} | 删除运行时 |
| WS | /ws/logs?serviceId={id} | 实时日志流 |
| WS | /ws/metrics?serviceId={id} | 实时指标流 |
| WS | /ws/system/stats | 实时主机资源统计流 |
| GET | / | Web UI 控制台（React） |

完整 API 规格见 `docs/modules/api.md`。

## 10. 非功能需求

| 类别 | 要求 |
|------|------|
| 性能 | 启停 < 1s，日志推送 < 500ms，单实例管理数百服务 |
| 资源 | 常驻内存 ≤ 50MB |
| 可用性 | 单节点部署，Supervisor 保障托管进程高可用 |
| 目标平台 | Windows 优先；接口层预留跨平台扩展 |
| 存储 | JSON 文件本地持久化，零数据库依赖 |
| 可观测性 | 日志 + 指标 + 健康状态，预留 Prometheus 暴露 |
| 安全 | 预留鉴权中间件位置；API 参数严格校验防注入 |

## 11. 架构概览

架构设计、C4 模型、部署拓扑与关键技术决策详见：
- `docs/architecture.md`
- `docs/adr/`

## 12. 风险与依赖

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| Windows 进程/服务管理 API 差异 | 行为不一致 | 中 | Platform 抽象 + Windows 单测矩阵 |
| 日志文件随运行增长 | 磁盘占用 | 中 | 按大小滚动（rotate） + 环形缓冲限流 |
| grep 式检索大文件耗时 | 检索变慢 | 低 | 滚动分文件 + 按时间范围裁剪扫描 |
| 进程孤儿/僵尸 | 资源泄漏 | 中 | Windows Job Object / 进程组管理 + 退出清理 |
| 配置热更新竞态 | 状态不一致 | 中 | RWMutex + 状态机约束 |

## 13. 路线图

| 阶段 | 内容 |
|------|------|
| Q1 (MVP) | core + Windows platform + 通用 handler + REST API + 日志/监控基础 + JSON 持久化 |
| Q2 | Java/MySQL/Redis 处理器 + WebSocket 实时流 + grep 式日志检索 + 运行时（JDK 多版本）管理 |
| Q3 | 资源限制细化（Job Object）+ 配置热更新增强 |
| Q4 | 告警通知、Web 管理界面 |
