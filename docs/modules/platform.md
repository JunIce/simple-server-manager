# 模块：平台适配层（internal/platform）

## 职责

抽象操作系统差异，向上层提供统一的进程/系统服务/资源/系统信息接口。当前优先实现 **Windows**，接口层预留跨平台扩展。

## 接口定义

```go
type Platform interface {
    // 进程管理
    StartProcess(config ProcessConfig) (*Process, error)
    StopProcess(pid int, force bool) error

    // 系统服务管理
    InstallService(config ServiceConfig) error
    UninstallService(serviceID string) error
    ServiceStatus(serviceID string) (ServiceStatus, error)

    // 系统信息
    GetSystemInfo() (*SystemInfo, error)
    GetProcessInfo(pid int) (*ProcessInfo, error)
    GetSystemStats() (*SystemStats, error)

    // 资源限制
    SetResourceLimits(pid int, limits ResourceLimits) error
}
```

## 平台实现

### WindowsPlatform（Windows SCM + Job Object）

```go
type WindowsPlatform struct{}
```

- **进程启动**：`exec.Command`，通过 `CREATE_NEW_PROCESS_GROUP` 标志建立独立进程组。

```go
func (p *WindowsPlatform) StartProcess(config ProcessConfig) (*Process, error) {
    cmd := exec.Command(config.Command, config.Args...)
    cmd.SysProcAttr = &syscall.SysProcAttr{
        CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
    }
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    return &Process{ PID: cmd.Process.Pid, Cmd: cmd }, nil
}
```

- **系统服务安装**：使用 `golang.org/x/sys/windows/svc` 的 `mgm` 包操作服务控制管理器（SCM），`CreateService` / `DeleteService`。
- **资源限制**：Windows Job Object（`CreateJobObject` + `AssignProcessToJobObject`）限制内存与 CPU。
- **进程信息**：通过 Windows API（`OpenProcess` + `GetProcessMemoryInfo` / `GetProcessTimes`）获取内存与 CPU 使用率。
- **主机资源**：`GetSystemStats` 采集 CPU（`GetSystemTimes` 增量）、内存（`GlobalMemoryStatusEx`）、磁盘（`GetDiskFreeSpaceExW` 枚举驱动器）、进程数（`CreateToolhelp32Snapshot`）。
- **进程终止**：`GenerateConsoleCtrlEvent` 发 CTRL_BREAK（优雅），超时 `TerminateProcess`（强制）。

## 辅助结构

```go
type ProcessConfig struct {
    Command    string
    Args       []string
    WorkingDir string
    Env        []string
    Limits     ResourceLimits
}

type Process struct {
    PID int
    Cmd *exec.Cmd
}

type SystemInfo struct {
    OS        string
    Arch      string
    Hostname  string
    CPUs      int
    MemoryMB  int64
    UptimeSec int64
}

type ProcessInfo struct {
    PID      int
    CPUUsage float64
    MemoryKB int64
    Status   string
}

type ResourceLimits struct {
    MaxMemoryMB int64
    MaxCPU      float64
}
```

## 工厂选择

```go
func New() Platform {
    switch runtime.GOOS {
    case "windows":
        return &WindowsPlatform{}
    default:
        // 预留：未来扩展 linux/darwin
        return &WindowsPlatform{}
    }
}
```

## 一致性保障

- Windows 实现通过 `platform_test.go` 接口测试矩阵保障行为正确。
- Windows 特有差异（SCM / Job Object / CreateProcess flags）封装在实现内部，不泄漏到 core 层。
- 接口保持稳定，未来扩展 Linux/macOS 无需改动上层。
