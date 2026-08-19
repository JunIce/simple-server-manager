//go:build !windows

package platform

import (
	"errors"
	"os"
	"runtime"
	"time"

	"service-manager/internal/config"
	"service-manager/pkg/process"
)

// OtherPlatform 非 Windows 平台的桩实现，保持接口可编译。
type OtherPlatform struct {
	jobs map[int]struct{}
}

func newWindows() Platform {
	return &OtherPlatform{jobs: map[int]struct{}{}}
}

func (p *OtherPlatform) StartProcess(cfg ProcessConfig) (*Process, error) {
	proc, err := process.Start(process.Options{
		Command:    cfg.Command,
		Args:       cfg.Args,
		WorkingDir: cfg.WorkingDir,
		Env:        cfg.Env,
		NewGroup:   true,
	})
	if err != nil {
		return nil, err
	}
	return &Process{PID: proc.PID, StartTime: time.Now()}, nil
}

func (p *OtherPlatform) StopProcess(pid int, force bool) error {
	proc := &process.Process{PID: pid}
	if force {
		return proc.Kill()
	}
	return proc.Stop(5 * time.Second)
}

func (p *OtherPlatform) InstallService(svc config.ServiceConfig, binPath, args string) error {
	return errors.New("system service installation not supported on this platform")
}

func (p *OtherPlatform) UninstallService(serviceID string) error {
	return errors.New("system service uninstallation not supported on this platform")
}

func (p *OtherPlatform) ServiceStatus(serviceID string) (config.ServiceStatus, error) {
	return config.StatusUnknown, errors.New("system service status not supported on this platform")
}

func (p *OtherPlatform) GetSystemInfo() (*SystemInfo, error) {
	hostname, _ := os.Hostname()
	return &SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname,
		CPUs:     runtime.NumCPU(),
	}, nil
}

func (p *OtherPlatform) GetProcessInfo(pid int) (*ProcessInfo, error) {
	return &ProcessInfo{PID: pid, Status: "running"}, nil
}

// GetSystemStats 非 Windows 平台返回基本统计。
func (p *OtherPlatform) GetSystemStats() (*SystemStats, error) {
	return &SystemStats{
		CPUs:      runtime.NumCPU(),
		UptimeSec: 0,
		Timestamp: time.Now(),
	}, nil
}

func (p *OtherPlatform) GetTopProcesses(n int) ([]ProcessInfo, error) {
	return nil, nil
}

func (p *OtherPlatform) KillProcess(pid int) error {
	return errors.New("kill process not supported on this platform")
}

func (p *OtherPlatform) SetResourceLimits(pid int, limits config.ResourceLimits) error {
	return nil
}
