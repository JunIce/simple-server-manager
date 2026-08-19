# 功能文档

本文档定义各功能模块的行为、输入输出与交互，作为实现与验收的基线。

> 目标平台：Windows；单机部署，无数据库、无日志索引。

## 1. 服务管理（core）

### 1.1 服务注册
- 输入：`ServiceConfig`（命令、参数、类型、环境、健康检查、资源限制、重启策略等）。
- 输出：注册结果（成功/失败 + 生效配置）。
- 规则：ID 全局唯一；`Type` 必须存在对应 handler；命令不可为空。
- 注册成功后配置持久化到 JSON 文件。

### 1.2 服务生命周期
| 操作 | 行为 |
|------|------|
| start | 构建命令 → 启动进程 → 挂载日志/监控/守护 → 状态 running |
| stop | 发停止信号 → 等待退出 → 超时强制终止 → 状态 stopped |
| restart | stop + start 的原子组合 |
| status | 返回当前状态枚举 + 最近错误 |

### 1.3 自动重启（Supervisor）
- `RestartPolicy`：最大重启次数、退避算法（固定 / 指数）、退避间隔。
- 进程崩溃 → 退避等待 → 重新拉起 → 超过上限置 `failed`。

## 2. 平台适配（platform）

统一接口封装 Windows 差异，预留跨平台扩展：

| 能力 | Windows |
|------|---------|
| 进程管理 | os/exec + `CREATE_NEW_PROCESS_GROUP` |
| 系统服务 | Windows SCM (`golang.org/x/sys/windows/svc`) |
| 资源限制 | Windows Job Object |
| 系统信息 | Windows API（`GetSystemInfo` 等） |

## 3. 服务类型处理器（handlers）

将 `ServiceConfig` 翻译为可执行命令与健康检查逻辑：

| 类型 | 命令构建 | 健康检查 |
|------|---------|---------|
| java | `java [JVM opts] -jar / -cp` | HTTP / JMX / 进程存活 |
| mysql | `mysqld --defaults-file ...` | TCP 端口 / 进程 |
| redis | `redis-server config --port ...` | PING / 端口 |
| generic | 原样 command + args | 进程存活 / 可选 HTTP |

## 4. 监控（monitor）

- **指标**：CPU 使用率、内存占用（RSS）、启动时间、运行状态。
- **健康检查**：周期 ticker 触发，检查失败计数，供 supervisor 决策重启。
- **订阅**：指标变化通过 channel 扇出给 WebSocket 客户端。

## 5. 日志管理（logger）

- **采集**：读取子进程 stdout/stderr pipe，按行解析。
- **缓冲**：环形缓冲区（RingBuffer）保留最近 N 条，避免内存无限增长。
- **落盘**：日志按行写入文本文件，按大小滚动（rotate）。
- **检索**：grep/sed 式流式行匹配，不建索引；支持关键字 / 正则 / 时间范围 / 级别过滤，命中即返回。
- **推送**：实时条目广播给订阅者（WebSocket）。

### 日志查询参数（LogQuery）
| 字段 | 说明 |
|------|------|
| keywords | 关键字列表 |
| level | 日志级别 |
| startTime/endTime | 时间范围（用于裁剪待扫描文件） |
| regex | 正则匹配 |
| caseSensitive | 大小写敏感 |
| maxResults | 最大返回条数（命中即提前终止） |

## 6. API（api）

- REST 资源命名与统一响应体见 `prd.md` §9。
- WebSocket 端点：`/ws/logs`、`/ws/metrics`，按 `serviceId` 订阅，推送 `{type, data}` 帧。

## 7. 配置管理（config）

- 加载：启动时读取 JSON 配置文件（`--config` > 用户目录 > 当前目录）。
- 持久化：注册/更新操作写回 JSON 文件（原子写）。
- 热更新：运行期可更新服务配置，通过 RWMutex 保证一致性。
- 无数据库依赖。

## 8. 运行时管理（runtime）

管理主机上安装的运行时环境（典型为 JDK 多版本），支持不同服务绑定不同版本启动。

- **注册/发现**：手动注册运行时，或自动发现已安装 JDK（JAVA_HOME、常见目录、注册表）。
- **解析**：三级解析——默认运行时 / 精确 ID / 版本前缀。
- **绑定**：Java 服务通过 `java.jdk` 引用运行时，启动时解析 `java.exe` 绝对路径并注入 `JAVA_HOME`。
- **校验**：注册时校验可执行文件存在性。
- **持久化**：运行时注册表随 `Config` 存于 JSON 文件。

### 运行时查询/注册参数
| 字段 | 说明 |
|------|------|
| id | 运行时唯一标识 |
| kind | 类型（`jdk` 等） |
| version | 版本号 |
| home | 安装根目录 |
| default | 是否默认 |

### 用法示例（不同服务用不同 JDK）

```bash
# 发现已安装 JDK
curl -X POST http://localhost:8080/api/runtimes/discover -d '{"kind":"jdk"}'

# 注册一个 JDK
curl -X POST http://localhost:8080/api/runtimes -d '{
  "id":"jdk-1.8","kind":"jdk","version":"1.8.0_401",
  "home":"C:\\Program Files\\Java\\jdk1.8.0_401"
}'

# 注册 Java 服务并绑定 JDK 8
curl -X POST http://localhost:8080/api/services -d '{
  "id":"legacy-app","type":"java",
  "java":{"jdk":"jdk-1.8","jarFile":"legacy.jar"}
}'
```


## 9. 文件目录（Files / Web UI）

基于配置的 `server.fileRoot`（在「配置」页设置）提供远程文件管理：

- **导航**：面包屑 + 上级返回，浏览文件/文件夹。
- **列表**：目录在前、按名称排序；悬停行尾操作。
- **操作**：上传、下载、重命名、删除、复制（剪贴板）、粘贴（自动加"副本"）、新建文件夹。
- **编辑**：双击文件弹出 Monaco 代码编辑器（按扩展名自动识别语言，如 js/json/yaml/py 等），编辑保存。
- **安全**：所有路径经 `securePath` 校验，禁止路径穿越到根目录之外。
