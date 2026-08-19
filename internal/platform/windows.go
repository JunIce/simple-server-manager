//go:build windows

package platform

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"service-manager/internal/config"
	"service-manager/pkg/process"
)

// WindowsPlatform Windows 平台实现。
type WindowsPlatform struct {
	mu         sync.Mutex
	jobs       map[uint32]windows.Handle // pid -> job object handle
	cpuCache   map[uint32]cpuSample      // pid -> 服务监控的 cpu 采样
	topCpuCache map[uint32]cpuSample     // pid -> topN 枚举的 cpu 采样
	sysCpu     cpuSample                 // 系统 CPU 采样
	kernel32   *windows.LazyDLL
}

func newWindows() Platform {
	return &WindowsPlatform{
		jobs:        map[uint32]windows.Handle{},
		cpuCache:    map[uint32]cpuSample{},
		topCpuCache: map[uint32]cpuSample{},
		kernel32:    windows.NewLazySystemDLL("kernel32.dll"),
	}
}

// StartProcess 启动进程，若配置资源限制则绑定 Job Object。
func (p *WindowsPlatform) StartProcess(cfg ProcessConfig) (*Process, error) {
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
	startTime := time.Now()
	if rt, err := p.startedAt(proc.PID); err == nil {
		startTime = rt
	}

	if cfg.Limits.MaxMemoryMB > 0 || cfg.Limits.MaxCPU > 0 {
		if err := p.SetResourceLimits(proc.PID, cfg.Limits); err != nil {
			// 资源限制失败不阻断启动
			_ = err
		}
	}
	return &Process{PID: proc.PID, StartTime: startTime}, nil
}

func (p *WindowsPlatform) startedAt(pid int) (time.Time, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return time.Time{}, err
	}
	defer windows.CloseHandle(handle)
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, creation.Nanoseconds()), nil
}

// StopProcess 停止进程：force=false 优雅停止，force=true 强杀。
func (p *WindowsPlatform) StopProcess(pid int, force bool) error {
	if force {
		return p.kill(pid)
	}
	proc := &process.Process{PID: pid}
	return proc.Stop(5 * time.Second)
}

func (p *WindowsPlatform) kill(pid int) error {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		if err == windows.ERROR_INVALID_PARAMETER {
			return ErrProcessNotFound
		}
		return err
	}
	defer windows.CloseHandle(handle)
	return windows.TerminateProcess(handle, 1)
}

// InstallService 通过 Windows SCM 安装系统服务。
func (p *WindowsPlatform) InstallService(svc config.ServiceConfig, binPath, args string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	_, err = m.CreateService(
		svc.ID,
		binPath,
		mgr.Config{
			DisplayName:     svc.Name,
			Description:     "managed by service-manager",
			StartType:       mgr.StartAutomatic,
			BinaryPathName:  fmt.Sprintf(`"%s" %s`, binPath, args),
		},
	)
	return err
}

// UninstallService 通过 SCM 卸载系统服务。
func (p *WindowsPlatform) UninstallService(serviceID string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceID)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Delete()
}

// ServiceStatus 查询系统服务状态。
func (p *WindowsPlatform) ServiceStatus(serviceID string) (config.ServiceStatus, error) {
	m, err := mgr.Connect()
	if err != nil {
		return config.StatusUnknown, err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceID)
	if err != nil {
		return config.StatusUnknown, err
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return config.StatusUnknown, err
	}
	switch status.State {
	case svc.Running:
		return config.StatusRunning, nil
	case svc.StartPending:
		return config.StatusStarting, nil
	case svc.StopPending:
		return config.StatusStopping, nil
	case svc.Stopped:
		return config.StatusStopped, nil
	default:
		return config.StatusUnknown, nil
	}
}

// GetSystemInfo 获取系统信息。
func (p *WindowsPlatform) GetSystemInfo() (*SystemInfo, error) {
	hostname, _ := os.Hostname()

	info := &SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname,
		CPUs:     runtime.NumCPU(),
	}

	// 物理内存（通过 kernel32.GlobalMemoryStatusEx，避免平台类型缺失）
	k32 := windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx := k32.NewProc("GlobalMemoryStatusEx")
	var mse memoryStatusEx
	mse.Length = uint32(unsafe.Sizeof(mse))
	if r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mse))); r1 != 0 {
		info.MemoryMB = int64(mse.TotalPhys / (1024 * 1024))
	}

	// 运行时长
	procGetTickCount64 := k32.NewProc("GetTickCount64")
	if r1, _, _ := procGetTickCount64.Call(); r1 != 0 {
		info.UptimeSec = int64(r1 / 1000)
	}
	return info, nil
}

// GetSystemStats 获取主机实时资源使用情况。
func (p *WindowsPlatform) GetSystemStats() (*SystemStats, error) {
	stats := &SystemStats{
		CPUs:      runtime.NumCPU(),
		UptimeSec: p.uptimeSec(),
		Timestamp: time.Now(),
	}
	stats.CPUUsage = p.calcSystemCPU()

	// 内存
	var mse memoryStatusEx
	mse.Length = uint32(unsafe.Sizeof(mse))
	procGlobalMemoryStatusEx := p.kernel32.NewProc("GlobalMemoryStatusEx")
	if r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mse))); r1 != 0 {
		stats.MemoryTotal = int64(mse.TotalPhys / (1024 * 1024))
		stats.MemoryUsed = int64((mse.TotalPhys - mse.AvailPhys) / (1024 * 1024))
		if mse.TotalPhys > 0 {
			stats.MemoryPct = float64(mse.TotalPhys-mse.AvailPhys) / float64(mse.TotalPhys) * 100
		}
	}

	// 磁盘
	stats.Disks = p.diskUsage()

	// 进程数
	stats.Processes = p.processCount()

	return stats, nil
}

func (p *WindowsPlatform) uptimeSec() int64 {
	procGetTickCount64 := p.kernel32.NewProc("GetTickCount64")
	if r1, _, _ := procGetTickCount64.Call(); r1 != 0 {
		return int64(r1 / 1000)
	}
	return 0
}

// calcSystemCPU 基于 GetSystemTimes 增量计算系统 CPU 使用率。
func (p *WindowsPlatform) calcSystemCPU() float64 {
	procGetSystemTimes := p.kernel32.NewProc("GetSystemTimes")
	var idle, kernel, user windows.Filetime
	if r1, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user))); r1 == 0 {
		return 0
	}
	now := uint64(time.Now().UnixNano() / 100)
	cpuTotal := uint64(idle.Nanoseconds()/100) + uint64(kernel.Nanoseconds()/100) + uint64(user.Nanoseconds()/100)

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sysCpu.now == 0 {
		p.sysCpu = cpuSample{cpu: cpuTotal, now: now}
		return 0
	}
	dCPU := cpuTotal - p.sysCpu.cpu
	dNow := now - p.sysCpu.now
	p.sysCpu = cpuSample{cpu: cpuTotal, now: now}
	if dNow <= 0 {
		return 0
	}
	// 计算非空闲占比：(总 - idle增量) / 总增量
	idleDelta := cpuTotal - uint64(kernel.Nanoseconds()/100)
	busy := dCPU - idleDelta
	pct := float64(busy) / float64(dCPU) * 100
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

// diskUsage 枚举各驱动器磁盘使用情况。
func (p *WindowsPlatform) diskUsage() []DiskUsage {
	drives, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	var disks []DiskUsage
	procGetDiskFreeSpaceEx := p.kernel32.NewProc("GetDiskFreeSpaceExW")
	for i := 0; i < 26; i++ {
		if drives&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A'+i)) + ":\\"
		root, _ := windows.UTF16PtrFromString(letter)
		var freeCaller, total, free uint64
		if r1, _, _ := procGetDiskFreeSpaceEx.Call(
			uintptr(unsafe.Pointer(root)),
			uintptr(unsafe.Pointer(&freeCaller)),
			uintptr(unsafe.Pointer(&total)),
			uintptr(unsafe.Pointer(&free))); r1 == 0 {
			continue
		}
		if total == 0 {
			continue
		}
		totalMB := int64(total / (1024 * 1024))
		freeMB := int64(free / (1024 * 1024))
		usedMB := totalMB - freeMB
		pct := float64(usedMB) / float64(totalMB) * 100
		if totalMB <= 0 {
			pct = 0
		}
		disks = append(disks, DiskUsage{
			Mount:    letter,
			TotalMB:  totalMB,
			FreeMB:   freeMB,
			UsedMB:   usedMB,
			UsagePct: pct,
		})
	}
	return disks
}

// processCount 统计系统进程数。
func (p *WindowsPlatform) processCount() int {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(snapshot)

	count := 0
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return 0
	}
	count++
	for {
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
		count++
	}
	return count
}

// GetTopProcesses 按 CPU 使用率降序返回前 n 个进程。
func (p *WindowsPlatform) GetTopProcesses(n int) ([]ProcessInfo, error) {
	if n <= 0 {
		n = 10
	}
	type procStat struct {
		pid    int
		name   string
		cpu    float64
		memKB  int64
	}
	var stats []procStat

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}
	for {
		name := utf16ToString(entry.ExeFile[:])
		info := p.cpuMemOf(int(entry.ProcessID))
		if info != nil {
			stats = append(stats, procStat{
				pid:   int(entry.ProcessID),
				name:  name,
				cpu:   info.CPUUsage,
				memKB: info.MemoryKB,
			})
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}

	sort.Slice(stats, func(i, j int) bool { return stats[i].cpu > stats[j].cpu })
	if len(stats) > n {
		stats = stats[:n]
	}

	out := make([]ProcessInfo, 0, len(stats))
	for _, s := range stats {
		out = append(out, ProcessInfo{
			PID:      s.pid,
			Name:     s.name,
			CPUUsage: s.cpu,
			MemoryKB: s.memKB,
			Status:   "running",
		})
	}
	return out, nil
}

// cpuMemOf 返回进程 CPU 使用率与内存（复用增量采样）。无法访问时返回 nil。
func (p *WindowsPlatform) cpuMemOf(pid int) *ProcessInfo {
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_QUERY_INFORMATION,
		false, uint32(pid))
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(handle)

	info := &ProcessInfo{PID: pid}

	var pmc processMemoryCounters
	pmc.Cb = uint32(unsafe.Sizeof(pmc))
	psapi := windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemoryInfo := psapi.NewProc("GetProcessMemoryInfo")
	if r1, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle), uintptr(unsafe.Pointer(&pmc)), uintptr(unsafe.Sizeof(pmc))); r1 != 0 {
		info.MemoryKB = int64(pmc.WorkingSetSize / 1024)
	}

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err == nil {
		cpu100ns := uint64(kernel.Nanoseconds()/100) + uint64(user.Nanoseconds()/100)
		now100ns := uint64(time.Now().UnixNano() / 100)
		info.CPUUsage = p.calcCPUCache(p.topCpuCache, pid, cpu100ns, now100ns)
	}
	return info
}

// KillProcess 强制终止进程。
func (p *WindowsPlatform) KillProcess(pid int) error {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		if err == windows.ERROR_INVALID_PARAMETER {
			return ErrProcessNotFound
		}
		return err
	}
	defer windows.CloseHandle(handle)
	if err := windows.TerminateProcess(handle, 1); err != nil {
		return err
	}
	// 通知进程组清理
	p.mu.Lock()
	delete(p.cpuCache, uint32(pid))
	delete(p.topCpuCache, uint32(pid))
	if h, ok := p.jobs[uint32(pid)]; ok {
		windows.CloseHandle(h)
		delete(p.jobs, uint32(pid))
	}
	p.mu.Unlock()
	return nil
}

// utf16ToString 将 UTF-16 缓冲区转字符串（遇到 0 截断）。
func utf16ToString(buf []uint16) string {
	for i, v := range buf {
		if v == 0 {
			buf = buf[:i]
			break
		}
	}
	return windows.UTF16ToString(buf)
}

// memoryStatusEx 与 MEMORYSTATUSEX 对齐。
type memoryStatusEx struct {
	Length       uint32
	MemoryLoad   uint32
	TotalPhys    uint64
	AvailPhys    uint64
	TotalPageFile uint64
	AvailPageFile uint64
	TotalVirtual  uint64
	AvailVirtual  uint64
	AvailExtendedVirtual uint64
}

// GetProcessInfo 获取进程信息（内存 + CPU）。
func (p *WindowsPlatform) GetProcessInfo(pid int) (*ProcessInfo, error) {
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_QUERY_INFORMATION,
		false, uint32(pid))
	if err != nil {
		if err == windows.ERROR_INVALID_PARAMETER {
			return nil, ErrProcessNotFound
		}
		return nil, err
	}
	defer windows.CloseHandle(handle)

	info := &ProcessInfo{PID: pid, Status: "running"}

	// 工作集内存
	var pmc processMemoryCounters
	pmc.Cb = uint32(unsafe.Sizeof(pmc))
	psapi := windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemoryInfo := psapi.NewProc("GetProcessMemoryInfo")
	if r1, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle), uintptr(unsafe.Pointer(&pmc)), uintptr(unsafe.Sizeof(pmc))); r1 != 0 {
		info.MemoryKB = int64(pmc.WorkingSetSize / 1024)
	}

	// CPU 使用率（增量计算，100ns 为单位）
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err == nil {
		cpu100ns := uint64(kernel.Nanoseconds()/100) + uint64(user.Nanoseconds()/100)
		now100ns := uint64(time.Now().UnixNano() / 100)
		info.CPUUsage = p.calcCPUCache(p.cpuCache, pid, cpu100ns, now100ns)
	}
	return info, nil
}

// SetResourceLimits 通过 Job Object 限制内存与 CPU。
func (p *WindowsPlatform) SetResourceLimits(pid int, limits config.ResourceLimits) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}

	var ext windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	if limits.MaxMemoryMB > 0 {
		ext.JobMemoryLimit = uintptr(limits.MaxMemoryMB * 1024 * 1024)
		ext.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_JOB_MEMORY
	}
	if limits.MaxCPU > 0 {
		// 每进程 CPU 时间上限（100ns 单位）：1 小时全核
		limit100ns := int64(limits.MaxCPU) * 3600 * 1e7
		ext.BasicLimitInformation.PerProcessUserTimeLimit = limit100ns
		ext.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_PROCESS_TIME
	}

	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&ext)), uint32(unsafe.Sizeof(ext))); err != nil {
		windows.CloseHandle(job)
		return err
	}

	procHandle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		windows.CloseHandle(job)
		return err
	}
	defer windows.CloseHandle(procHandle)

	if err := windows.AssignProcessToJobObject(job, procHandle); err != nil {
		windows.CloseHandle(job)
		return err
	}

	p.mu.Lock()
	p.jobs[uint32(pid)] = job
	p.mu.Unlock()
	return nil
}

// processMemoryCounters 与 PROCESS_MEMORY_COUNTERS 对齐。
type processMemoryCounters struct {
	Cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

func uint64ToFiletime(v uint64) windows.Filetime {
	return windows.Filetime{
		LowDateTime:  uint32(v & 0xFFFFFFFF),
		HighDateTime: uint32(v >> 32),
	}
}

type cpuSample struct {
	cpu uint64
	now uint64
}

// calcCPU 按指定缓存计算进程 CPU 使用率（兼容旧签名，使用服务监控缓存）。
func (p *WindowsPlatform) calcCPU(pid int, cpu100ns, now100ns uint64) float64 {
	return p.calcCPUCache(p.cpuCache, pid, cpu100ns, now100ns)
}

// calcCPUCache 基于独立缓存计算 CPU 使用率百分比，并裁剪到 [0, 100]。
// 使用独立缓存避免多个轮询器互相污染采样窗口导致虚高。
func (p *WindowsPlatform) calcCPUCache(cache map[uint32]cpuSample, pid int, cpu100ns, now100ns uint64) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := uint32(pid)
	prev, ok := cache[key]
	if !ok {
		cache[key] = cpuSample{cpu: cpu100ns, now: now100ns}
		return 0
	}
	if now100ns <= prev.now {
		return 0
	}
	dCPU := cpu100ns - prev.cpu
	dNow := now100ns - prev.now
	cache[key] = cpuSample{cpu: cpu100ns, now: now100ns}
	if dNow <= 0 {
		return 0
	}
	// 单核百分比 = dCPU/dNow*100；除以核数得到占总容量百分比
	pct := float64(dCPU) / float64(dNow) * 100 / float64(runtime.NumCPU())
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}
