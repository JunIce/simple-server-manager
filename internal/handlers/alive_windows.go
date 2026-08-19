//go:build windows

package handlers

import (
	"errors"
	"golang.org/x/sys/windows"
)

// checkProcessAlive 判断进程存活（Windows 用 OpenProcess 探测）。
func (b *baseHandler) checkProcessAlive(pid int) error {
	if pid <= 0 {
		return errors.New("invalid pid")
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		if err == windows.ERROR_INVALID_PARAMETER {
			return errors.New("process not found")
		}
		return err
	}
	windows.CloseHandle(handle)
	return nil
}
