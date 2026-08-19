// Package monitor 周期采集托管服务的运行指标并广播给订阅者。
package monitor

import (
	"context"
	"sync"
	"time"

	"service-manager/internal/config"
	"service-manager/internal/platform"
)

// ServiceMetrics 服务指标。
type ServiceMetrics struct {
	ServiceID string    `json:"serviceId"`
	Status    string    `json:"status"`
	CPUUsage  float64   `json:"cpuUsage"`
	MemoryKB  int64     `json:"memoryKB"`
	UptimeSec int64     `json:"uptimeSec"`
	LastCheck time.Time `json:"lastCheck"`
}

// MetricsUpdate 指标更新消息。
type MetricsUpdate struct {
	ServiceID string          `json:"serviceId"`
	Metrics   *ServiceMetrics `json:"metrics"`
}

// HostMetricsUpdate 主机资源更新消息。
type HostMetricsUpdate struct {
	Stats *platform.SystemStats `json:"stats"`
}

// Monitor 监控器。
type Monitor struct {
	mu             sync.RWMutex
	services       map[string]*ServiceMetrics
	subscribers    map[string]map[chan MetricsUpdate]struct{}
	hostStats      *platform.SystemStats
	hostSubscribers map[chan HostMetricsUpdate]struct{}
	platform       platform.Platform
	interval       time.Duration
}

// New 创建监控器。
func New(p platform.Platform, interval time.Duration) *Monitor {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Monitor{
		services:        map[string]*ServiceMetrics{},
		subscribers:     map[string]map[chan MetricsUpdate]struct{}{},
		hostSubscribers: map[chan HostMetricsUpdate]struct{}{},
		platform:        p,
		interval:        interval,
	}
}

// WatchHost 启动主机资源监控协程，周期采集并广播。
func (m *Monitor) WatchHost(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		// 立即采集一次
		m.collectHost()
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
	if err != nil {
		return
	}
	m.mu.Lock()
	m.hostStats = stats
	subs := make([]chan HostMetricsUpdate, 0, len(m.hostSubscribers))
	for ch := range m.hostSubscribers {
		subs = append(subs, ch)
	}
	m.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- HostMetricsUpdate{Stats: stats}:
		default: // 慢消费者丢弃
		}
	}
}

// GetHostStats 获取最近的主机资源统计。
func (m *Monitor) GetHostStats() (*platform.SystemStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.hostStats == nil {
		return nil, false
	}
	return m.hostStats, true
}

// SubscribeHost 订阅主机资源统计。
func (m *Monitor) SubscribeHost() (<-chan HostMetricsUpdate, func()) {
	ch := make(chan HostMetricsUpdate, 16)
	m.mu.Lock()
	m.hostSubscribers[ch] = struct{}{}
	m.mu.Unlock()
	return ch, func() { m.UnsubscribeHost(ch) }
}

// UnsubscribeHost 取消主机资源订阅。
func (m *Monitor) UnsubscribeHost(ch chan HostMetricsUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.hostSubscribers[ch]; ok {
		delete(m.hostSubscribers, ch)
		close(ch)
	}
}

// Watch 启动某服务的监控协程。
// getPID 返回当前进程 PID，statusFunc 返回当前状态。
func (m *Monitor) Watch(ctx context.Context, serviceID string, getPID func() int, statusFunc func() config.ServiceStatus, startTime func() time.Time) {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pid := getPID()
				if pid <= 0 {
					continue
				}
				info, err := m.platform.GetProcessInfo(pid)
				if err != nil {
					m.update(serviceID, &ServiceMetrics{
						ServiceID: serviceID,
						Status:    string(statusFunc()),
						LastCheck: time.Now(),
					})
					continue
				}
				up := int64(0)
				if st := startTime(); !st.IsZero() {
					up = int64(time.Since(st).Seconds())
				}
				metrics := &ServiceMetrics{
					ServiceID: serviceID,
					Status:    string(statusFunc()),
					CPUUsage:  info.CPUUsage,
					MemoryKB:  info.MemoryKB,
					UptimeSec: up,
					LastCheck: time.Now(),
				}
				m.update(serviceID, metrics)
				m.broadcast(serviceID, metrics)
			}
		}
	}()
}

func (m *Monitor) update(serviceID string, metrics *ServiceMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[serviceID] = metrics
}

// Get 获取某服务最近指标。
func (m *Monitor) Get(serviceID string) (*ServiceMetrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metrics, ok := m.services[serviceID]
	return metrics, ok
}

// All 返回全部指标。
func (m *Monitor) All() map[string]*ServiceMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]*ServiceMetrics, len(m.services))
	for k, v := range m.services {
		out[k] = v
	}
	return out
}

// Subscribe 订阅某服务指标。
func (m *Monitor) Subscribe(serviceID string) (<-chan MetricsUpdate, func()) {
	ch := make(chan MetricsUpdate, 16)
	m.mu.Lock()
	if m.subscribers[serviceID] == nil {
		m.subscribers[serviceID] = map[chan MetricsUpdate]struct{}{}
	}
	m.subscribers[serviceID][ch] = struct{}{}
	m.mu.Unlock()
	return ch, func() { m.Unsubscribe(serviceID, ch) }
}

// Unsubscribe 取消订阅。
func (m *Monitor) Unsubscribe(serviceID string, ch chan MetricsUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if subs, ok := m.subscribers[serviceID]; ok {
		if _, ok := subs[ch]; ok {
			delete(subs, ch)
			close(ch)
		}
	}
}

func (m *Monitor) broadcast(serviceID string, u *ServiceMetrics) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for ch := range m.subscribers[serviceID] {
		select {
		case ch <- MetricsUpdate{ServiceID: serviceID, Metrics: u}:
		default: // 慢消费者丢弃
		}
	}
}
