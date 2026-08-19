// Package handlers 将 ServiceConfig 翻译为特定服务类型的启动命令与健康检查逻辑。
package handlers

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"service-manager/internal/config"
)

var (
	ErrUnknownServiceType = errors.New("unknown service type")
)

// Handler 服务类型处理器接口。
type Handler interface {
	// BuildCommand 构建启动命令（command, args）。
	BuildCommand(cfg config.ServiceConfig) (string, []string)
	// Environment 返回额外环境变量（如 JAVA_HOME）。
	Environment(cfg config.ServiceConfig) []string
	// HealthCheck 健康检查。
	HealthCheck(cfg config.ServiceConfig, pid int) error
}

// baseHandler 提供通用能力。
type baseHandler struct{}

func (b *baseHandler) Environment(cfg config.ServiceConfig) []string { return nil }

// checkHTTPHealth GET endpoint，2xx 视为健康。
func (b *baseHandler) checkHTTPHealth(endpoint string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http health check failed: status %d", resp.StatusCode)
	}
	return nil
}

// checkTCPPort 检查 TCP 端口可达。
func (b *baseHandler) checkTCPPort(host string, port int, timeout time.Duration) error {
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// JavaHandler Java 服务处理器。
type JavaHandler struct {
	baseHandler
	cfg *config.Config
}

func (h *JavaHandler) BuildCommand(cfg config.ServiceConfig) (string, []string) {
	if cfg.Java == nil {
		return "", nil
	}
	jdk, err := h.cfg.ResolveJDK(cfg.Java.JDK)
	if err != nil {
		return "", nil
	}
	javaBin := filepath.Join(jdk.Home, "bin", "java.exe")

	args := []string{}
	if cfg.Java.JVMOptions != nil {
		args = append(args, cfg.Java.JVMOptions...)
	}
	if cfg.Java.MemoryLimit != "" {
		args = append(args, "-Xmx"+cfg.Java.MemoryLimit)
	}
	if cfg.Java.JarFile != "" {
		args = append(args, "-jar", cfg.Java.JarFile)
	} else if cfg.Java.MainClass != "" {
		args = append(args, cfg.Java.MainClass)
	}
	args = append(args, cfg.Args...)
	return javaBin, args
}

func (h *JavaHandler) Environment(cfg config.ServiceConfig) []string {
	jdk, err := h.cfg.ResolveJDK(cfg.Java.JDK)
	if err != nil {
		return nil
	}
	return []string{"JAVA_HOME=" + jdk.Home}
}

func (h *JavaHandler) HealthCheck(cfg config.ServiceConfig, pid int) error {
	hc := cfg.HealthCheck
	timeout := time.Duration(hc.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if hc.HTTPEndpoint != "" {
		return h.checkHTTPHealth(hc.HTTPEndpoint, timeout)
	}
	return h.checkProcessAlive(pid)
}

// MySQLHandler MySQL 服务处理器。
type MySQLHandler struct {
	baseHandler
}

func (h *MySQLHandler) BuildCommand(cfg config.ServiceConfig) (string, []string) {
	if cfg.MySQL == nil {
		return "", nil
	}
	args := []string{
		"--defaults-file=" + cfg.MySQL.ConfigFile,
		"--datadir=" + cfg.MySQL.DataDir,
	}
	if cfg.MySQL.Port > 0 {
		args = append(args, "--port="+fmt.Sprint(cfg.MySQL.Port))
	}
	return cfg.MySQL.BinaryPath, args
}

func (h *MySQLHandler) HealthCheck(cfg config.ServiceConfig, pid int) error {
	timeout := 3 * time.Second
	if cfg.MySQL != nil && cfg.MySQL.Port > 0 {
		return h.checkTCPPort("127.0.0.1", cfg.MySQL.Port, timeout)
	}
	return h.checkProcessAlive(pid)
}

// RedisHandler Redis 服务处理器。
type RedisHandler struct {
	baseHandler
}

func (h *RedisHandler) BuildCommand(cfg config.ServiceConfig) (string, []string) {
	args := []string{}
	if cfg.Redis != nil && cfg.Redis.ConfigFile != "" {
		args = append(args, cfg.Redis.ConfigFile)
	}
	if cfg.Redis != nil && cfg.Redis.Port > 0 {
		args = append(args, "--port", fmt.Sprint(cfg.Redis.Port))
	}
	if cfg.Redis != nil && cfg.Redis.BindAddress != "" {
		args = append(args, "--bind", cfg.Redis.BindAddress)
	}
	return "redis-server", args
}

func (h *RedisHandler) HealthCheck(cfg config.ServiceConfig, pid int) error {
	timeout := 3 * time.Second
	if cfg.Redis != nil && cfg.Redis.Port > 0 {
		return h.checkTCPPort("127.0.0.1", cfg.Redis.Port, timeout)
	}
	return h.checkProcessAlive(pid)
}

// GenericHandler 通用处理器：原样 command + args。
type GenericHandler struct {
	baseHandler
}

func (h *GenericHandler) BuildCommand(cfg config.ServiceConfig) (string, []string) {
	return cfg.Command, cfg.Args
}

func (h *GenericHandler) HealthCheck(cfg config.ServiceConfig, pid int) error {
	hc := cfg.HealthCheck
	if hc.HTTPEndpoint != "" {
		timeout := time.Duration(hc.TimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = 3 * time.Second
		}
		return h.checkHTTPHealth(hc.HTTPEndpoint, timeout)
	}
	return h.checkProcessAlive(pid)
}

// Get 返回指定类型的处理器。
func Get(t config.ServiceType, cfg *config.Config) (Handler, error) {
	switch t {
	case config.TypeJava:
		return &JavaHandler{cfg: cfg}, nil
	case config.TypeMySQL:
		return &MySQLHandler{}, nil
	case config.TypeRedis:
		return &RedisHandler{}, nil
	case config.TypeGeneric:
		return &GenericHandler{}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownServiceType, t)
	}
}
