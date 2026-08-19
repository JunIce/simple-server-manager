//go:build windows

package process

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// startedAt 返回进程启动时间。Windows 上通过 GetProcessTimes 获取。
func startedAt(p *Process) time.Time {
	if p == nil || p.PID <= 0 {
		return time.Now()
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(p.PID))
	if err != nil {
		return time.Now()
	}
	defer windows.CloseHandle(handle)

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return time.Now()
	}
	return time.Unix(0, creation.Nanoseconds())
}

// Start 启动子进程。
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
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Process{PID: cmd.Process.Pid, Cmd: cmd, Stdout: stdout, Stderr: stderr}, nil
}

// Stop 优雅停止：先发送 CTRL_BREAK，超时后 TerminateProcess。
func (p *Process) Stop(gracePeriod time.Duration) error {
	if p == nil || p.Cmd == nil || p.Cmd.Process == nil {
		return errors.New("process not started")
	}
	_ = p.sendCtrlBreak()

	done := make(chan error, 1)
	go func() { done <- p.Cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(gracePeriod):
		return p.Kill()
	}
}

// Kill 强制终止进程（TerminateProcess）。
func (p *Process) Kill() error {
	if p == nil || p.Cmd == nil || p.Cmd.Process == nil {
		return errors.New("process not started")
	}
	return p.Cmd.Process.Kill()
}

// Wait 等待进程退出。
func (p *Process) Wait() error {
	if p == nil || p.Cmd == nil {
		return errors.New("process not started")
	}
	return p.Cmd.Wait()
}

// sendCtrlBreak 向进程组发送 CTRL_BREAK_EVENT，类似 POSIX SIGTERM。
func (p *Process) sendCtrlBreak() error {
	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(p.PID))
}

// SignalGroup 向整个进程组发信号。
func (p *Process) SignalGroup(sig syscall.Signal) error {
	// Windows 上 CTRL_BREAK 已经是进程组级别的
	_ = os.Getpid()
	return p.sendCtrlBreak()
}
