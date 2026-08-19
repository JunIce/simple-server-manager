//go:build !windows

package process

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func startedAt(p *Process) time.Time {
	return time.Now()
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
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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

// Stop 优雅停止：先 SIGTERM，超时后 SIGKILL。
func (p *Process) Stop(gracePeriod time.Duration) error {
	if p == nil || p.Cmd == nil || p.Cmd.Process == nil {
		return errors.New("process not started")
	}
	_ = p.signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() { done <- p.Cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(gracePeriod):
		return p.Kill()
	}
}

// Kill 强制终止进程（SIGKILL）。
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

// signal 向进程发送信号。
func (p *Process) signal(sig syscall.Signal) error {
	return p.signalGroup(sig)
}

// signalGroup 向整个进程组发信号，避免只杀父进程留下孤儿子进程。
func (p *Process) signalGroup(sig syscall.Signal) error {
	if p == nil || p.Cmd == nil || p.Cmd.Process == nil {
		return errors.New("process not started")
	}
	// 负数 PID 表示进程组
	return syscall.Kill(-p.PID, sig)
}

// SignalGroup 向整个进程组发信号。
func (p *Process) SignalGroup(sig syscall.Signal) error {
	_ = os.Getpid()
	return p.signalGroup(sig)
}
