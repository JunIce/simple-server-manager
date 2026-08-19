//go:build !windows

package handlers

import (
	"errors"
	"os"
	"syscall"
)

// checkProcessAlive 判断进程存活（Unix 用 signal 0 探测）。
func (b *baseHandler) checkProcessAlive(pid int) error {
	if pid <= 0 {
		return errors.New("invalid pid")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.Signal(0))
}
