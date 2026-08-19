//go:build !windows

package api

// drivesBitmask 非 Windows 平台无盘符概念，返回空。
func drivesBitmask() (uint32, error) {
	return 0, nil
}
