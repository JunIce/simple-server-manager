// Package config 定义数据模型与配置管理：JSON 持久化到本地或用户目录，零数据库依赖。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"service-manager/pkg/utils"
)

// ServiceType 服务类型。
type ServiceType string

const (
	TypeJava    ServiceType = "java"
	TypeMySQL   ServiceType = "mysql"
	TypeRedis   ServiceType = "redis"
	TypeGeneric ServiceType = "generic"
)

// ServiceStatus 服务状态枚举。
type ServiceStatus string

const (
	StatusStopped  ServiceStatus = "stopped"
	StatusStarting ServiceStatus = "starting"
	StatusRunning  ServiceStatus = "running"
	StatusStopping ServiceStatus = "stopping"
	StatusFailed   ServiceStatus = "failed"
	StatusUnknown  ServiceStatus = "unknown"
)

// RestartPolicy 重启策略。
type RestartPolicy struct {
	MaxRetries   int           `json:"maxRetries"`
	BackoffType  string        `json:"backoffType"` // fixed / exponential
	BackoffDelay time.Duration `json:"backoffDelay"`
	MaxBackoff   time.Duration `json:"maxBackoff"`
}

// Default 返回默认重启策略。
func (p RestartPolicy) Default() RestartPolicy {
	if p.BackoffType == "" {
		p.BackoffType = "exponential"
	}
	if p.BackoffDelay == 0 {
		p.BackoffDelay = 2 * time.Second
	}
	if p.MaxBackoff == 0 {
		p.MaxBackoff = 60 * time.Second
	}
	return p
}

// HealthCheckConfig 健康检查配置。
type HealthCheckConfig struct {
	HTTPEndpoint string `json:"httpEndpoint"`
	IntervalSec  int    `json:"intervalSec"` // 默认 5
	MaxFailures  int    `json:"maxFailures"` // 默认 3
	TimeoutMS    int    `json:"timeoutMS"`   // 默认 3000
}

// ResourceLimits 资源限制。
type ResourceLimits struct {
	MaxMemoryMB int64   `json:"maxMemoryMB"`
	MaxCPU      float64 `json:"maxCPU"` // 0~100 百分比
}

// LogConfig 日志配置。
type LogConfig struct {
	OutputDir  string `json:"outputDir"`
	MaxSizeMB  int64  `json:"maxSizeMB"` // 默认 50
	MaxFiles   int    `json:"maxFiles"`  // 默认 10
	BufferSize int    `json:"bufferSize"`
}

// Default 返回默认日志配置。
func (l LogConfig) Default() LogConfig {
	if l.MaxSizeMB == 0 {
		l.MaxSizeMB = 50
	}
	if l.MaxFiles == 0 {
		l.MaxFiles = 10
	}
	if l.BufferSize == 0 {
		l.BufferSize = 10000
	}
	return l
}

// JavaConfig Java 服务专属配置。
type JavaConfig struct {
	JDK         string   `json:"jdk"` // 运行时 ID，如 jdk-17；为空用默认
	JarFile     string   `json:"jarFile"`
	MainClass   string   `json:"mainClass"`
	JVMOptions  []string `json:"jvmOptions"`
	MemoryLimit string   `json:"memoryLimit"` // 如 "512m"
}

// MySQLConfig MySQL 服务专属配置。
type MySQLConfig struct {
	BinaryPath string `json:"binaryPath"`
	ConfigFile string `json:"configFile"`
	DataDir    string `json:"dataDir"`
	Port       int    `json:"port"`
}

// RedisConfig Redis 服务专属配置。
type RedisConfig struct {
	ConfigFile  string `json:"configFile"`
	Port        int    `json:"port"`
	BindAddress string `json:"bindAddress"`
}

// ServiceConfig 服务静态配置。
type ServiceConfig struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           ServiceType       `json:"type"`
	Command        string            `json:"command"`
	Args           []string          `json:"args"`
	WorkingDir     string            `json:"workingDir"`
	Environment    map[string]string `json:"environment"`
	Environments   []string          `json:"environments"` // dev/test/prod
	AutoRestart    bool              `json:"autoRestart"`
	RestartPolicy  RestartPolicy     `json:"restartPolicy"`
	HealthCheck    HealthCheckConfig `json:"healthCheck"`
	ResourceLimits ResourceLimits    `json:"resourceLimits"`
	LogConfig      LogConfig         `json:"logConfig"`
	Java           *JavaConfig       `json:"java,omitempty"`
	MySQL          *MySQLConfig      `json:"mysql,omitempty"`
	Redis          *RedisConfig      `json:"redis,omitempty"`
}

// JDK JDK 版本配置项，集中管理系统上可用的不同 JDK 地址。
type JDK struct {
	ID      string `json:"id"` // 唯一标识，如 jdk-17、jdk-1.8
	Name    string `json:"name"`
	Version string `json:"version"`
	Home    string `json:"home"` // 安装根目录
	Default bool   `json:"default"`
}

// ServerConfig 服务端配置。
type ServerConfig struct {
	Port     int        `json:"port"`
	Host     string     `json:"host"`
	Auth     AuthConfig `json:"auth"`
	DataDir  string     `json:"dataDir"`
	FileRoot string     `json:"fileRoot"` // 文件目录页面的根目录
}

// AuthConfig 鉴权配置。
type AuthConfig struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
}

// MonitorConfig 监控配置。
type MonitorConfig struct {
	IntervalSeconds int `json:"intervalSeconds"`
}

// Config 顶层配置。
type Config struct {
	mu       sync.RWMutex
	Server   ServerConfig            `json:"server"`
	Services map[string]ServiceConfig `json:"services"`
	JDKs     map[string]JDK          `json:"jdks"`
	Log      LogConfig               `json:"log"`
	Monitor  MonitorConfig           `json:"monitor"`
	Path     string                  `json:"-"`
}

// Default 返回默认配置。
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     8080,
			Host:     "0.0.0.0",
			Auth:     AuthConfig{Enabled: false, Token: ""},
			DataDir:  "./data",
			FileRoot: "./data",
		},
		Services: map[string]ServiceConfig{},
		JDKs:     map[string]JDK{},
		Log: LogConfig{
			OutputDir:  "./data/logs",
			MaxSizeMB:  50,
			MaxFiles:   10,
			BufferSize: 10000,
		},
		Monitor: MonitorConfig{IntervalSeconds: 5},
	}
}

var (
	ErrInvalidConfig = errors.New("invalid config")
)

// Validate 校验配置。
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("%w: port %d out of range", ErrInvalidConfig, c.Server.Port)
	}
	if c.Monitor.IntervalSeconds <= 0 {
		return fmt.Errorf("%w: monitor interval must be positive", ErrInvalidConfig)
	}
	return nil
}

// validateService 校验单个服务配置。
func validateService(svc ServiceConfig) error {
	if svc.ID == "" {
		return fmt.Errorf("%w: service id is required", ErrInvalidConfig)
	}
	switch svc.Type {
	case TypeJava:
		if svc.Java == nil {
			return fmt.Errorf("%w: service %s type java requires java config", ErrInvalidConfig, svc.ID)
		}
		if svc.Java.JarFile == "" && svc.Java.MainClass == "" {
			return fmt.Errorf("%w: service %s java requires jarFile or mainClass", ErrInvalidConfig, svc.ID)
		}
	case TypeMySQL:
		if svc.MySQL == nil {
			return fmt.Errorf("%w: service %s type mysql requires mysql config", ErrInvalidConfig, svc.ID)
		}
	case TypeRedis:
		if svc.Redis == nil {
			return fmt.Errorf("%w: service %s type redis requires redis config", ErrInvalidConfig, svc.ID)
		}
	case TypeGeneric:
		if svc.Command == "" {
			return fmt.Errorf("%w: service %s generic requires command", ErrInvalidConfig, svc.ID)
		}
	default:
		return fmt.Errorf("%w: unknown service type %q", ErrInvalidConfig, svc.Type)
	}
	return nil
}

// resolvePath 确定配置文件路径：--config > 用户目录 > 当前目录。
func resolvePath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "service-manager", "config.json"), nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".service-manager", "config.json"), nil
	}
	return "config.json", nil
}

// Load 从指定路径加载配置，路径为空时自动解析。
func Load(explicit string) (*Config, error) {
	path, err := resolvePath(explicit)
	if err != nil {
		return nil, err
	}
	cfg := Default()
	cfg.Path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.Path = path
	return cfg, nil
}

// Save 原子写保存配置到 JSON 文件。
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	path := c.Path
	if path == "" {
		path, _ = resolvePath("")
		c.Path = path
	}
	if err := utils.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return utils.AtomicWrite(path, data)
}

// UpdateService 更新服务配置并持久化。
func (c *Config) UpdateService(svc ServiceConfig) error {
	if err := validateService(svc); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Services == nil {
		c.Services = map[string]ServiceConfig{}
	}
	c.Services[svc.ID] = svc
	return c.saveLocked()
}

// DeleteService 删除服务配置并持久化。
func (c *Config) DeleteService(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.Services[id]; !ok {
		return fmt.Errorf("service %s not found", id)
	}
	delete(c.Services, id)
	return c.saveLocked()
}

// GetService 读取服务配置。
func (c *Config) GetService(id string) (ServiceConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	svc, ok := c.Services[id]
	return svc, ok
}

// AllServices 返回服务配置副本。
func (c *Config) AllServices() map[string]ServiceConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]ServiceConfig, len(c.Services))
	for k, v := range c.Services {
		out[k] = v
	}
	return out
}

// GetServer 读取服务端配置。
func (c *Config) GetServer() ServerConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Server
}

// GetLog 读取日志配置。
func (c *Config) GetLog() LogConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Log
}

// GetMonitor 读取监控配置。
func (c *Config) GetMonitor() MonitorConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Monitor
}

// GetJDK 读取 JDK 配置。
func (c *Config) GetJDK(id string) (JDK, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	j, ok := c.JDKs[id]
	return j, ok
}

// AllJDKs 返回 JDK 配置映射副本。
func (c *Config) AllJDKs() map[string]JDK {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]JDK, len(c.JDKs))
	for k, v := range c.JDKs {
		out[k] = v
	}
	return out
}

// UpdateJDK 更新单个 JDK 配置并持久化。
func (c *Config) UpdateJDK(j JDK) error {
	if j.ID == "" || j.Home == "" {
		return fmt.Errorf("%w: jdk id and home are required", ErrInvalidConfig)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.updateJDKLocked(j)
}

// DeleteJDK 删除 JDK 配置并持久化。
func (c *Config) DeleteJDK(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.JDKs[id]; !ok {
		return fmt.Errorf("jdk %s not found", id)
	}
	delete(c.JDKs, id)
	return c.saveLocked()
}

// ResolveJDK 三级解析：ref 为空→默认；精确 ID→返回；版本/ID 前缀→返回。
func (c *Config) ResolveJDK(ref string) (JDK, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.JDKs) == 0 {
		return JDK{}, fmt.Errorf("%w: no jdk configured", ErrInvalidConfig)
	}
	if ref != "" {
		if j, ok := c.JDKs[ref]; ok {
			return j, nil
		}
		for _, j := range c.JDKs {
			if strings.HasPrefix(j.Version, ref) || strings.HasPrefix(j.ID, ref) {
				return j, nil
			}
		}
		return JDK{}, fmt.Errorf("%w: jdk %q not found", ErrInvalidConfig, ref)
	}
	for _, j := range c.JDKs {
		if j.Default {
			return j, nil
		}
	}
	// 无默认则返回第一个（字典序稳定）
	var first *JDK
	for _, j := range c.JDKs {
		if first == nil || j.ID < first.ID {
			copy := j
			first = &copy
		}
	}
	return *first, nil
}

// JDKHome 返回 JDK 的 java.exe 所在目录（bin）。
func (c *Config) JDKHome(ref string) (string, error) {
	j, err := c.ResolveJDK(ref)
	if err != nil {
		return "", err
	}
	return filepath.Join(j.Home, "bin", "java.exe"), nil
}

// UpdateSystem 整体更新系统配置（服务端/日志/监控/JDK），并持久化。
func (c *Config) UpdateSystem(server ServerConfig, log LogConfig, monitor MonitorConfig, jdks []JDK) error {
	if monitor.IntervalSeconds <= 0 {
		return fmt.Errorf("%w: monitor interval must be positive", ErrInvalidConfig)
	}
	if server.Port < 1 || server.Port > 65535 {
		return fmt.Errorf("%w: port %d out of range", ErrInvalidConfig, server.Port)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Server = server
	c.Log = log.Default()
	c.Monitor = monitor
	if c.Services == nil {
		c.Services = map[string]ServiceConfig{}
	}
	c.JDKs = make(map[string]JDK, len(jdks))
	for _, j := range jdks {
		if j.ID == "" {
			continue
		}
		c.JDKs[j.ID] = j
	}
	// 同类型仅保留一个默认
	c.enforceSingleDefaultLocked()
	return c.saveLocked()
}

// updateJDKLocked 需在持有写锁时调用。
func (c *Config) updateJDKLocked(j JDK) error {
	if c.JDKs == nil {
		c.JDKs = map[string]JDK{}
	}
	c.JDKs[j.ID] = j
	c.enforceSingleDefaultLocked()
	return c.saveLocked()
}

// enforceSingleDefaultLocked 需在持有写锁时调用：同类型仅保留一个默认。
func (c *Config) enforceSingleDefaultLocked() {
	hasDefault := false
	for id, j := range c.JDKs {
		if !j.Default {
			continue
		}
		if hasDefault {
			j.Default = false
			c.JDKs[id] = j
			continue
		}
		hasDefault = true
	}
}

// saveLocked 需在持有写锁时调用。
func (c *Config) saveLocked() error {
	path := c.Path
	if path == "" {
		path, _ = resolvePath("")
		c.Path = path
	}
	if err := utils.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return utils.AtomicWrite(path, data)
}
