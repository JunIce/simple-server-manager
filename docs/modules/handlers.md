# 模块：服务类型处理器（internal/handlers）

## 职责

将通用的 `ServiceConfig` 翻译为特定服务类型可执行的命令、参数与健康检查逻辑，隔离各服务的启动差异。

## 接口定义

```go
type Handler interface {
    BuildCommand(config ServiceConfig) (string, []string)
    HealthCheck(config ServiceConfig, pid int) error
}

type baseHandler struct{}
func (b *baseHandler) checkProcessAlive(pid int) error {
    // 通过 syscall.Kill(pid, 0) 或平台进程信息判断存活
    return nil
}
func (b *baseHandler) checkHTTPHealth(endpoint string) error {
    // GET endpoint，2xx 视为健康
    return nil
}
```

## 各类型实现

### 1. JavaHandler

```go
type JavaHandler struct {
    baseHandler
    cfg *config.Config  // 持有系统配置，用于解析 JDK
}

// 从 JavaConfig 解析 JDK，返回 java.exe 绝对路径
func (h *JavaHandler) BuildCommand(config ServiceConfig) (string, []string) {
    // 三级解析：config.Java.JDK 为空则取默认 JDK
    jdk, err := h.cfg.ResolveJDK(config.Java.JDK)
    if err != nil {
        return "", nil  // 错误由调用方处理
    }
    javaBin := filepath.Join(jdk.Home, "bin", "java.exe")

    args := []string{}
    if config.Java.JVMOptions != nil {
        args = append(args, config.Java.JVMOptions...)
    }
    if config.Java.MemoryLimit != "" {
        args = append(args, "-Xmx"+config.Java.MemoryLimit)
    }
    if config.Java.JarFile != "" {
        args = append(args, "-jar", config.Java.JarFile)
    } else if config.Java.MainClass != "" {
        args = append(args, config.Java.MainClass)
    }
    args = append(args, config.Args...)

    return javaBin, args
}
```

- 不同服务可绑定不同 JDK 版本：`config.Java.JDK` 指定配置页中登记的 JDK ID（如 `jdk-1.8`、`jdk-17`），为空则用默认 JDK。
- 启动时还需将 `JAVA_HOME` 指向所选 JDK 的 `Home`，注入进程环境变量（见 `JavaHandler.Environment`）。
- 健康检查优先级：HTTP 端点 > 进程存活。

### 2. MySQLHandler

```go
func (h *MySQLHandler) BuildCommand(config ServiceConfig) (string, []string) {
    return config.MySQLConfig.BinaryPath, []string{
        "--defaults-file=" + config.MySQLConfig.ConfigFile,
        "--datadir=" + config.MySQLConfig.DataDir,
        "--port=" + fmt.Sprint(config.MySQLConfig.Port),
    }
}
```

- 健康检查：TCP 连接端口 + 可选 `SELECT 1` 探活。

### 3. RedisHandler

```go
func (h *RedisHandler) BuildCommand(config ServiceConfig) (string, []string) {
    return "redis-server", []string{
        config.RedisConfig.ConfigFile,
        "--port", fmt.Sprint(config.RedisConfig.Port),
        "--bind", config.RedisConfig.BindAddress,
    }
}
```

- 健康检查：`PING` 命令响应 `PONG`。

### 4. GenericHandler

```go
func (h *GenericHandler) BuildCommand(config ServiceConfig) (string, []string) {
    return config.Command, config.Args
}
```

- 健康检查：进程存活；若配置 HTTP 端点则追加 HTTP 检查。

## 处理器注册与分发

```go
func Get(t ServiceType, cfg *config.Config) (Handler, error) {
    switch t {
    case TypeJava:
        return &JavaHandler{cfg: cfg}, nil
    case TypeMySQL:
        return &MySQLHandler{}, nil
    case TypeRedis:
        return &RedisHandler{}, nil
    case TypeGeneric:
        return &GenericHandler{}, nil
    default:
        return nil, ErrUnknownServiceType
    }
}
```

## 类型扩展

新增服务类型只需：
1. 实现 `Handler` 接口。
2. 在 `registry` 注册。
3. 在 `ServiceConfig` 中补充该类型专属配置字段（如 `MySQLConfig`、`JavaConfig`）。

## JavaConfig（Java 类型专属配置）

```go
type JavaConfig struct {
    JDK         string   `json:"jdk"`         // 引用的运行时 ID，如 "jdk-17"；为空用默认
    JarFile     string   `json:"jarFile"`     // jar 包路径
    MainClass   string   `json:"mainClass"`   // 主类名（与 jarFile 二选一）
    JVMOptions  []string `json:"jvmOptions"`  // JVM 参数
    MemoryLimit string   `json:"memoryLimit"` // 如 "512m"
}
```

## JDK 解析流程

```
ServiceConfig.Java.JDK ("jdk-1.8" / "" / "17")
  → config.ResolveJDK(ref)  （默认 / 精确 ID / 版本前缀 三级）
  → 命中 Home
  → javaBin = <home>\bin\java.exe
  → 注入 JAVA_HOME = jdk.Home
  → 构建最终命令
```

