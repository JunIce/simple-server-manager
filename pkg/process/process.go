// Package process 提供跨平台安全的子进程管理能力，是 platform 层的底层依赖。
package process

import (
	"io"
	"os/exec"
	"time"
)

// Process 表示一个被托管的子进程。
type Process struct {
	PID    int
	Cmd    *exec.Cmd
	Stdout io.ReadCloser
	Stderr io.ReadCloser
}

// Options 描述进程启动选项。
type Options struct {
	Command    string
	Args       []string
	WorkingDir string
	Env        []string
	// NewGroup 在 Windows 上设置 CREATE_NEW_PROCESS_GROUP，
	// 在 Unix 上设置 Setpgid，用于进程组信号管理。
	NewGroup bool
}

// StartedAt 记录进程启动时间（由平台实现填充）。
func (p *Process) StartedAt() time.Time { return startedAt(p) }
