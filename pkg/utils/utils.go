// Package utils 提供无业务语义的通用工具函数。
package utils

import (
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackoffDelay 计算退避延迟（fixed / exponential + 抖动）。
// backoffType: "fixed" 或 "exponential"。
func BackoffDelay(backoffType string, baseDelay, maxBackoff time.Duration, retries int) time.Duration {
	var d time.Duration
	switch backoffType {
	case "exponential":
		if retries > 60 {
			retries = 60
		}
		d = baseDelay << uint(retries)
		if d <= 0 || (maxBackoff > 0 && d > maxBackoff) {
			d = maxBackoff
		}
	default: // fixed
		d = baseDelay
	}
	// ±10% 抖动
	var jitter time.Duration
	if d > 0 {
		half := d / 10
		if half <= 0 {
			half = 1
		}
		jitter = time.Duration(rand.Int63n(int64(half*2))) - half
	}
	return d + jitter
}

// AtomicWrite 临时文件 + rename 原子写，避免中途崩溃损坏文件。
func AtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ParseLogLevel 从日志行提取级别，无法识别时返回 INFO。
func ParseLogLevel(line string) string {
	// 常见格式：[INFO]、[ERROR]、[WARN]、level=info 等
	if i := strings.Index(line, "["); i >= 0 {
		if j := strings.Index(line[i:], "]"); j > 1 {
			level := strings.ToUpper(strings.TrimSpace(line[i+1 : i+j]))
			switch level {
			case "INFO", "DEBUG", "WARN", "WARNING", "ERROR", "FATAL", "TRACE":
				return level
			}
		}
	}
	for _, l := range []string{"ERROR", "WARN", "INFO", "DEBUG"} {
		if strings.Contains(strings.ToUpper(line), "level="+strings.ToLower(l)) ||
			strings.Contains(strings.ToUpper(line), "level="+l) {
			return l
		}
	}
	return "INFO"
}

// ContainsAll 判断 s 是否包含所有子串（忽略大小写）。
func ContainsAll(s string, subs []string) bool {
	lower := strings.ToLower(s)
	for _, sub := range subs {
		if sub == "" {
			continue
		}
		if !strings.Contains(lower, strings.ToLower(sub)) {
			return false
		}
	}
	return true
}

// TruncateString 截断字符串到 n 个字符（按 rune）。
func TruncateString(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// MapToEnv map[string]string 转 "KEY=value" 切片，稳定排序便于测试与调试。
func MapToEnv(m map[string]string) []string {
	env := make([]string, 0, len(m))
	for k, v := range m {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	return env
}

// EnsureDir 确保目录存在，不存在则创建。
func EnsureDir(dir string) error {
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
