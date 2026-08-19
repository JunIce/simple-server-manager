# 模块：核心服务管理（internal/core）

## 职责

管理所有托管服务的生命周期：注册、启动、停止、重启、状态跟踪、自动重启守护。

## 核心类型

### ServiceManager

```go
type ServiceManager struct {
    mu         sync.RWMutex
    services   map[string]*ManagedService
    supervisor *Supervisor
    logger     *Logger
    monitor    *Monitor
}
```

- `services`：以服务 ID 为键的托管服务表，由 `RWMutex` 保护。
- 依赖 `Supervisor`（守护）、`Logger`（日志）、`Monitor`（监控）。

### ManagedService

```go
type ManagedService struct {
    ID          string
    Config      ServiceConfig
    Process     *process.Process
    Status      ServiceStatus
    Environment string
    Metrics     *ServiceMetrics
    LastError   error

    cancel     context.CancelFunc
    restartCh  chan struct{}
    stopCh     chan struct{}
}
```

- `cancel`：取消该服务所有关联 goroutine。
- `restartCh`：外部触发立即重启的信号。
- `stopCh`：停止守护循环的信号。

### ServiceConfig

```go
type ServiceConfig struct {
    ID              string            `json:"id"`
    Name            string            `json:"name"`
    Type            ServiceType       `json:"type"`
    Command         string            `json:"command"`
    Args            []string          `json:"args"`
    WorkingDir      string            `json:"workingDir"`
    Environment     map[string]string `json:"environment"`
    Environments    []string          `json:"environments"`
    AutoRestart     bool              `json:"autoRestart"`
    RestartPolicy   RestartPolicy     `json:"restartPolicy"`
    HealthCheck     HealthCheckConfig `json:"healthCheck"`
    ResourceLimits  ResourceLimits    `json:"resourceLimits"`
    LogConfig       LogConfig         `json:"logConfig"`
}
```

### RestartPolicy

```go
type RestartPolicy struct {
    MaxRetries    int           `json:"maxRetries"`
    BackoffType   string        `json:"backoffType"` // fixed / exponential
    BackoffDelay  time.Duration `json:"backoffDelay"`
    MaxBackoff    time.Duration `json:"maxBackoff"`
}
```

## 关键实现

### StartService

```go
func (sm *ServiceManager) StartService(ctx context.Context, id string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    service, exists := sm.services[id]
    if !exists {
        return ErrServiceNotFound
    }

    cmd := service.buildCommand()          // 委托 handlers 翻译
    cmd.Env = service.getEnvironment()     // 环境变量注入

    if err := cmd.Start(); err != nil {
        return err
    }

    go sm.supervisor.Watch(service)        // 守护
    service.Status = StatusRunning
    service.Process = cmd.Process
    return nil
}
```

### StopService 流程

1. 调用 `service.cancel()` 停止守护 goroutine。
2. 向进程发送终止信号（`process.Stop`）。
3. 等待 `stopCh` 或超时；超时则 `force kill`。
4. 状态置 `stopped`，清理日志/监控订阅。

### Supervisor.Watch

```go
func (s *Supervisor) Watch(service *ManagedService) {
    retries := 0
    for {
        select {
        case <-service.stopCh:
            return
        case <-service.restartCh:
            // 手动重启
        default:
        }

        err := service.Process.Wait()
        if err == nil && !service.Config.AutoRestart {
            service.Status = StatusStopped
            return
        }

        if retries >= service.Config.RestartPolicy.MaxRetries {
            service.Status = StatusFailed
            service.LastError = err
            return
        }

        delay := s.backoff(service.Config.RestartPolicy, retries)
        time.Sleep(delay)
        if err := s.restart(service); err != nil {
            service.LastError = err
        }
        retries++
    }
}
```

### 退避算法

- **fixed**：`delay = BackoffDelay`。
- **exponential**：`delay = min(BackoffDelay * 2^retries, MaxBackoff)`，抖动 `±10%` 防止惊群。

## 并发与一致性

- 所有对 `services` 的增删改查通过 `RWMutex`。
- 状态变更集中在 `manager` 方法内，避免外部直接写 `Status`。
- 优雅停机：`manager.Stop()` 遍历所有服务逐个停止，`context.WithTimeout` 兜底。

## 错误定义

```go
var (
    ErrServiceNotFound   = errors.New("service not found")
    ErrServiceRunning    = errors.New("service already running")
    ErrServiceStopped    = errors.New("service already stopped")
    ErrInvalidConfig     = errors.New("invalid service config")
)
```
