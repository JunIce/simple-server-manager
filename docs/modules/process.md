# 模块：进程管理库（pkg/process）

## 职责

提供 Windows 下安全的子进程管理能力，是 platform 层的底层依赖。

## 核心类型

```go
type Process struct {
    PID    int
    Cmd    *exec.Cmd
    Stdout io.ReadCloser
    Stderr io.ReadCloser
}

type Options struct {
    Command    string
    Args       []string
    WorkingDir string
    Env        []string
    NewGroup   bool   // 建立独立进程组（CREATE_NEW_PROCESS_GROUP）
}
```

## 启动

```go
func Start(opts Options) (*Process, error) {
    cmd := exec.Command(opts.Command, opts.Args...)
    if opts.WorkingDir != "" {
        cmd.Dir = opts.WorkingDir
    }
    if opts.Env != nil {
        cmd.Env = opts.Env
    }
    if opts.NewGroup {
        cmd.SysProcAttr = &syscall.SysProcAttr{
            CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
        }
    }

    stdout, err := cmd.StdoutPipe()
    if err != nil { return nil, err }
    stderr, err := cmd.StderrPipe()
    if err != nil { return nil, err }

    if err := cmd.Start(); err != nil {
        return nil, err
    }
    return &Process{ PID: cmd.Process.Pid, Cmd: cmd, Stdout: stdout, Stderr: stderr }, nil
}
```

## 停止（优雅 → 强制）

```go
func (p *Process) Stop(gracePeriod time.Duration) error {
    if p.Cmd.Process == nil {
        return nil
    }
    // 先发 CTRL_BREAK（控制台事件，进程组内传播）
    p.sendCtrlBreak()

    done := make(chan error, 1)
    go func() { done <- p.Cmd.Wait() }()

    select {
    case err := <-done:
        return err
    case <-time.After(gracePeriod):
        return p.Kill()  // 超时 TerminateProcess
    }
}

func (p *Process) Kill() error {
    return p.Cmd.Process.Kill()  // 底层调 TerminateProcess
}
```

## 控制台事件（优雅停止）

```go
// 向进程组发送 CTRL_BREAK_EVENT，类似 POSIX 的 SIGTERM
func (p *Process) sendCtrlBreak() error {
    return windows.GenerateConsoleCtrlEvent(
        windows.CTRL_BREAK_EVENT,
        uint32(p.PID),
    )
}
```

## 等待退出

```go
func (p *Process) Wait() error {
    return p.Cmd.Wait()
}
```

## Windows 平台要点

| 能力 | 说明 |
|------|------|
| 终止信号 | `GenerateConsoleCtrlEvent`(CTRL_BREAK) → `TerminateProcess` |
| 进程组 | `CREATE_NEW_PROCESS_GROUP` |
| 资源限制 | 上层 platform 通过 Job Object 绑定 |
| 退出状态 | exit code（`ProcessState.ExitCode()`） |

Windows 特定实现通过构建标签 `//go:build windows` 与 POSIX 版本隔离。

## 使用示例

```go
proc, err := process.Start(process.Options{
    Command:  "java",
    Args:     []string{"-jar", "app.jar"},
    NewGroup: true,
})
if err != nil { return err }

go func() {
    proc.Stop(5 * time.Second)  // 优雅停止（CTRL_BREAK → 超时强杀）
}()
```
