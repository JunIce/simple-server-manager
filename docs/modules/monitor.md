# 模块：监控（internal/monitor）

## 职责

周期采集托管服务的运行指标，执行健康检查，并向订阅者（WebSocket、supervisor）推送状态。

## 核心类型

### Monitor

```go
type Monitor struct {
    mu        sync.RWMutex
    services  map[string]*ServiceMetrics
    interval  time.Duration
    subscribers map[string]map[chan MetricsUpdate]struct{}
}
```

### ServiceMetrics

```go
type ServiceMetrics struct {
    ServiceID   string    `json:"serviceId"`
    Status      string    `json:"status"`
    CPUUsage    float64   `json:"cpuUsage"`   // 百分比
    MemoryKB    int64     `json:"memoryKB"`   // RSS
    UptimeSec   int64     `json:"uptimeSec"`
    LastCheck   time.Time `json:"lastCheck"`
}
```

## 采集流程

```go
func (m *Monitor) Watch(ctx context.Context, service *ManagedService) {
    ticker := time.NewTicker(m.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            info, err := m.platform.GetProcessInfo(service.Process.PID)
            if err != nil {
                service.LastError = err
                m.notifyFailure(service, err)
                continue
            }
            metrics := &ServiceMetrics{
                ServiceID: service.ID,
                Status:    string(service.Status),
                CPUUsage:  info.CPUUsage,
                MemoryKB:  info.MemoryKB,
                UptimeSec: int64(time.Since(service.StartTime).Seconds()),
            }
            m.update(service.ID, metrics)
            m.broadcast(service.ID, metrics)
        }
    }
}
```

## 主机资源监控（WatchHost）

除服务级指标外，Monitor 还提供**主机资源**监控：周期采集 CPU / 内存 / 磁盘 / 进程数并广播。

```go
func (m *Monitor) WatchHost(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(m.interval)
        defer ticker.Stop()
        m.collectHost()  // 立即采集一次
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                m.collectHost()
            }
        }
    }()
}

func (m *Monitor) collectHost() {
    stats, err := m.platform.GetSystemStats()
    if err != nil { return }
    m.hostStats = stats
    // 广播给订阅者（/ws/system/stats）
}

// 订阅主机资源
func (m *Monitor) SubscribeHost() (<-chan HostMetricsUpdate, func())
func (m *Monitor) GetHostStats() (*platform.SystemStats, bool)
```

### HostMetricsUpdate

```go
type HostMetricsUpdate struct {
    Stats *platform.SystemStats `json:"stats"`
}
```

主机数据源实现见 `platform.GetSystemStats`（Windows 通过 `GetSystemTimes`/`GlobalMemoryStatusEx`/`GetDiskFreeSpaceExW`/`CreateToolhelp32Snapshot`）。

## 健康检查

```go
func (m *Monitor) HealthCheck(service *ManagedService) error {
    handler := handlers.Get(service.Config.Type)
    if err := handler.HealthCheck(service.Config, service.Process.PID); err != nil {
        service.consecutiveFailures++
        if service.consecutiveFailures >= service.Config.HealthCheck.MaxFailures {
            return ErrUnhealthy
        }
        return err
    }
    service.consecutiveFailures = 0
    return nil
}
```

- `HealthCheckConfig.MaxFailures`：连续失败阈值，达到后交由 supervisor 决策重启或置 failed。

## 订阅与广播

```go
func (m *Monitor) Subscribe(serviceID string) (<-chan MetricsUpdate, func()) {
    ch := make(chan MetricsUpdate, 16)
    m.mu.Lock()
    m.subscribers[serviceID][ch] = struct{}{}
    m.mu.Unlock()
    return ch, func() { m.unsubscribe(serviceID, ch) }
}

func (m *Monitor) broadcast(serviceID string, u MetricsUpdate) {
    m.mu.RLock()
    for ch := range m.subscribers[serviceID] {
        select {
        case ch <- u:
        default: // 慢消费者丢弃，避免阻塞
        }
    }
    m.mu.RUnlock()
}
```

- 每个订阅者独立缓冲 channel，写满则丢弃，防止慢消费者拖垮采集循环。

## 指标暴露（预留）

- 预留 Prometheus `/metrics` 端点，将 `ServiceMetrics` 转为 Gauge（cpu_usage、memory_bytes 等）。
- 指标标签：`service_id`、`status`。
