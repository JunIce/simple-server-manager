// Package platform 抽象操作系统差异，向上层提供统一的进程/系统服务/资源/系统信息接口。
// 当前优先实现 Windows，接口层预留跨平台扩展。
package platform

import (
	"errors"
	"time"

	"service-manager/internal/config"
)

// ProcessConfig 进程启动参数。
type ProcessConfig struct {
	Command    string
	Args       []string
	WorkingDir string
	Env        []string
	Limits     config.ResourceLimits
}

// Process 表示已启动的进程。
type Process struct {
	PID       int
	StartTime time.Time
}

// SystemInfo 系统信息。
type SystemInfo struct {
	OS        string
	Arch      string
	Hostname  string
	CPUs      int
	MemoryMB  int64
	UptimeSec int64
}

// DiskUsage 磁盘分区使用情况。
type DiskUsage struct {
	Mount     string  `json:"mount"`
	TotalMB   int64   `json:"totalMB"`
	FreeMB    int64   `json:"freeMB"`
	UsedMB    int64   `json:"usedMB"`
	UsagePct  float64 `json:"usagePct"`
}

// SystemStats 主机实时资源使用情况。
type SystemStats struct {
	CPUs        int         `json:"cpus"`
	CPUUsage    float64     `json:"cpuUsage"` // 百分比
	MemoryTotal int64       `json:"memoryTotal"`
	MemoryUsed  int64       `json:"memoryUsed"`
	MemoryPct   float64     `json:"memoryPct"`
	Disks       []DiskUsage `json:"disks"`
	Processes   int         `json:"processes"`
	UptimeSec   int64       `json:"uptimeSec"`
	Timestamp   time.Time   `json:"timestamp"`
}

// ProcessInfo 进程信息。
type ProcessInfo struct {
	PID      int     `json:"pid"`
	Name     string  `json:"name"`     // 进程名/可执行文件名
	CPUUsage float64 `json:"cpuUsage"` // 百分比
	MemoryKB int64   `json:"memoryKB"`
	Status   string  `json:"status"`
}

// Platform 平台接口。
type Platform interface {
	// 进程管理
	StartProcess(config ProcessConfig) (*Process, error)
	StopProcess(pid int, force bool) error

	// 系统服务管理
	InstallService(svc config.ServiceConfig, binPath, args string) error
	UninstallService(serviceID string) error
	ServiceStatus(serviceID string) (config.ServiceStatus, error)

	// 系统信息
	GetSystemInfo() (*SystemInfo, error)
	GetProcessInfo(pid int) (*ProcessInfo, error)
	GetSystemStats() (*SystemStats, error)
	GetTopProcesses(n int) ([]ProcessInfo, error)
	KillProcess(pid int) error

	// 资源限制
	SetResourceLimits(pid int, limits config.ResourceLimits) error
}

// ErrProcessNotFound 进程不存在。
var ErrProcessNotFound = errors.New("process not found")

// New 创建平台实例。
func New() Platform {
	return newWindows()
}
