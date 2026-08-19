//go:build windows

package api

import "golang.org/x/sys/windows"

// drivesBitmask 返回Windows逻辑磁盘位掩码。
func drivesBitmask() (uint32, error) {
	return windows.GetLogicalDrives()
}
