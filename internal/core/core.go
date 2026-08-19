// Package core 核心服务管理：服务生命周期、自动重启守护。
package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"service-manager/internal/config"
	"service-manager/internal/handlers"
	"service-manager/internal/logger"
	"service-manager/internal/monitor"
	"service-manager/internal/platform"
	"service-manager/pkg/process"
	"service-manager/pkg/utils"
)

var (
	ErrServiceNotFound   = errors.New("service not found")
	ErrServiceRunning    = errors.New("service already running")
	ErrServiceStopped    = errors.New("service already stopped")
	ErrInvalidConfig     = errors.New("invalid service config")
	ErrBuildCommand      = errors.New("failed to build command")
)

// ManagedService 托管服务实例。
type ManagedService struct {
	ID          string
	Config      config.ServiceConfig
	Proc        *process.Process
	Status      config.ServiceStatus
	Environment string
	LastError   error
	StartAt     time.Time

	mu        sync.RWMutex
	cancel    context.CancelFunc
	restartCh chan struct{}
	stopCh    chan struct{}
	procMu    sync.Mutex
}

// SetStatus 设置状态。
func (s *ManagedService) SetStatus(st config.ServiceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = st
}

// GetStatus 获取状态。
func (s *ManagedService) GetStatus() config.ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// SetProc 设置进程。
func (s *ManagedService) SetProc(p *process.Process) {
	s.procMu.Lock()
	defer s.procMu.Unlock()
	s.Proc = p
	s.StartAt = time.Now()
}

// GetPID 返回当前 PID。
func (s *ManagedService) GetPID() int {
	s.procMu.Lock()
	defer s.procMu.Unlock()
	if s.Proc != nil {
		return s.Proc.PID
	}
	return 0
}

// GetStartTime 返回启动时间。
func (s *ManagedService) GetStartTime() time.Time {
	s.procMu.Lock()
	defer s.procMu.Unlock()
	return s.StartAt
}

// SignalRestart 通知守护协程立即重启。
func (s *ManagedService) SignalRestart() {
	s.mu.RLock()
	ch := s.restartCh
	s.mu.RUnlock()
	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// ServiceManager 服务管理器。
type ServiceManager struct {
	mu         sync.RWMutex
	services   map[string]*ManagedService
	cfg        *config.Config
	platform   platform.Platform
	logger     *logger.LogManager
	monitor    *monitor.Monitor
	stopGrace  time.Duration
	rootCtx    context.Context
	rootCancel context.CancelFunc
}

// Options 创建 ServiceManager 的选项。
type Options struct {
	Config       *config.Config
	Platform     platform.Platform
	Logger       *logger.LogManager
	Monitor      *monitor.Monitor
	StopGraceful time.Duration
}

// New 创建服务管理器。
func New(opts Options) *ServiceManager {
	if opts.StopGraceful <= 0 {
		opts.StopGraceful = 5 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	sm := &ServiceManager{
		services:   map[string]*ManagedService{},
		cfg:        opts.Config,
		platform:   opts.Platform,
		logger:     opts.Logger,
		monitor:    opts.Monitor,
		stopGrace:  opts.StopGraceful,
		rootCtx:    ctx,
		rootCancel: cancel,
	}
	// 从配置加载已有服务
	for id, svcCfg := range opts.Config.AllServices() {
		sm.services[id] = &ManagedService{
			ID:     id,
			Config: svcCfg,
			Status: config.StatusStopped,
		}
	}
	return sm
}

// Register 注册服务。
func (sm *ServiceManager) Register(svc config.ServiceConfig) error {
	if err := sm.validateService(svc); err != nil {
		return err
	}
	sm.mu.Lock()
	if _, exists := sm.services[svc.ID]; exists {
		sm.mu.Unlock()
		return fmt.Errorf("%w: id %s already exists", ErrInvalidConfig, svc.ID)
	}
	sm.services[svc.ID] = &ManagedService{
		ID:     svc.ID,
		Config: svc,
		Status: config.StatusStopped,
	}
	sm.mu.Unlock()

	if err := sm.cfg.UpdateService(svc); err != nil {
		return err
	}
	return nil
}

// Unregister 注销服务（先停止）。
func (sm *ServiceManager) Unregister(id string) error {
	if err := sm.StopService(id); err != nil && !errors.Is(err, ErrServiceStopped) {
		return err
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if _, ok := sm.services[id]; !ok {
		return fmt.Errorf("%w: %s", ErrServiceNotFound, id)
	}
	delete(sm.services, id)
	return sm.cfg.DeleteService(id)
}

// Update 更新服务配置。
func (sm *ServiceManager) Update(id string, svc config.ServiceConfig) error {
	if err := sm.validateService(svc); err != nil {
		return err
	}
	svc.ID = id
	sm.mu.Lock()
	managed, ok := sm.services[id]
	if !ok {
		sm.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrServiceNotFound, id)
	}
	if managed.GetStatus() == config.StatusRunning || managed.GetStatus() == config.StatusStarting {
		sm.mu.Unlock()
		return fmt.Errorf("%w: stop service %s before update", ErrServiceRunning, id)
	}
	managed.Config = svc
	managed.ID = id
	sm.mu.Unlock()
	return sm.cfg.UpdateService(svc)
}

// Get 获取托管服务。
func (sm *ServiceManager) Get(id string) (*ManagedService, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.services[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, id)
	}
	return s, nil
}

// List 列出所有服务。
func (sm *ServiceManager) List() []*ManagedService {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]*ManagedService, 0, len(sm.services))
	for _, s := range sm.services {
		out = append(out, s)
	}
	return out
}

// StartService 启动服务。ctx 用于调用期超时控制，服务运行期使用 manager 根上下文。
func (sm *ServiceManager) StartService(ctx context.Context, id string) error {
	svc, err := sm.Get(id)
	if err != nil {
		return err
	}
	if st := svc.GetStatus(); st == config.StatusRunning || st == config.StatusStarting {
		return fmt.Errorf("%w: %s", ErrServiceRunning, id)
	}

	svc.SetStatus(config.StatusStarting)
	svc.LastError = nil

	proc, err := sm.startProcess(sm.rootCtx, svc)
	if err != nil {
		svc.SetStatus(config.StatusFailed)
		svc.LastError = err
		return err
	}
	svc.SetProc(proc)
	svc.SetStatus(config.StatusRunning)

	// 启动监控（基于根上下文，不随请求结束而取消）
	ctxSvc, cancel := context.WithCancel(sm.rootCtx)
	svc.mu.Lock()
	svc.cancel = cancel
	svc.restartCh = make(chan struct{})
	svc.stopCh = make(chan struct{})
	svc.mu.Unlock()

	sm.monitor.Watch(ctxSvc, svc.ID, svc.GetPID, svc.GetStatus, svc.GetStartTime)

	// 启动守护
	go sm.supervise(ctxSvc, svc)
	return nil
}

// StopService 停止服务。
func (sm *ServiceManager) StopService(id string) error {
	svc, err := sm.Get(id)
	if err != nil {
		return err
	}
	st := svc.GetStatus()
	if st != config.StatusRunning && st != config.StatusStarting {
		return fmt.Errorf("%w: %s", ErrServiceStopped, id)
	}

	svc.SetStatus(config.StatusStopping)

	// 通知守护协程停止
	svc.mu.Lock()
	if svc.stopCh != nil {
		close(svc.stopCh)
		svc.stopCh = nil
	}
	cancel := svc.cancel
	svc.cancel = nil
	svc.mu.Unlock()
	if cancel != nil {
		cancel()
	}

	// 停止进程
	svc.procMu.Lock()
	proc := svc.Proc
	svc.procMu.Unlock()
	if proc != nil {
		_ = proc.Stop(sm.stopGrace)
	}

	svc.SetStatus(config.StatusStopped)
	return nil
}

// RestartService 重启服务。
func (sm *ServiceManager) RestartService(ctx context.Context, id string) error {
	svc, err := sm.Get(id)
	if err != nil {
		return err
	}
	st := svc.GetStatus()
	if st == config.StatusRunning || st == config.StatusStarting {
		if err := sm.StopService(id); err != nil {
			return err
		}
	}
	return sm.StartService(ctx, id)
}

// validateService 校验服务配置。
func (sm *ServiceManager) validateService(svc config.ServiceConfig) error {
	if svc.ID == "" {
		return fmt.Errorf("%w: id required", ErrInvalidConfig)
	}
	if _, err := handlers.Get(svc.Type, sm.cfg); err != nil {
		return err
	}
	// JDK 存在性校验
	if svc.Type == config.TypeJava && svc.Java != nil {
		if _, err := sm.cfg.ResolveJDK(svc.Java.JDK); err != nil {
			return err
		}
	}
	return nil
}

// startProcess 构建命令并启动进程。
func (sm *ServiceManager) startProcess(ctx context.Context, svc *ManagedService) (*process.Process, error) {
	handler, err := handlers.Get(svc.Config.Type, sm.cfg)
	if err != nil {
		return nil, err
	}
	cmd, args := handler.BuildCommand(svc.Config)
	if cmd == "" {
		return nil, fmt.Errorf("%w: %s", ErrBuildCommand, svc.ID)
	}

	env := append(os.Environ(), utils.MapToEnv(svc.Config.Environment)...)
	if extra := handler.Environment(svc.Config); len(extra) > 0 {
		env = append(env, extra...)
	}

	proc, err := process.Start(process.Options{
		Command:    cmd,
		Args:       args,
		WorkingDir: svc.Config.WorkingDir,
		Env:        env,
		NewGroup:   true,
	})
	if err != nil {
		return nil, err
	}

	// 资源限制
	if svc.Config.ResourceLimits.MaxMemoryMB > 0 || svc.Config.ResourceLimits.MaxCPU > 0 {
		_ = sm.platform.SetResourceLimits(proc.PID, svc.Config.ResourceLimits)
	}

	// 日志采集
	if sm.logger != nil {
		sm.logger.StartCollection(ctx, svc.ID, proc.Stdout, proc.Stderr)
	}
	return proc, nil
}

// supervise 守护协程：等待退出、退避重启。
func (sm *ServiceManager) supervise(ctx context.Context, svc *ManagedService) {
	retries := 0
	for {
		svc.mu.RLock()
		stopCh := svc.stopCh
		restartCh := svc.restartCh
		svc.mu.RUnlock()
		if stopCh == nil {
			return
		}

		exitCh := make(chan error, 1)
		go func() {
			svc.procMu.Lock()
			proc := svc.Proc
			svc.procMu.Unlock()
			if proc != nil {
				exitCh <- proc.Wait()
			} else {
				exitCh <- errors.New("no process")
			}
		}()

		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		case <-restartCh:
			// 手动/健康触发重启
		case <-exitCh:
			if !svc.Config.AutoRestart {
				svc.SetStatus(config.StatusStopped)
				return
			}
		}

		// 到达重试上限
		if retries >= svc.Config.RestartPolicy.MaxRetries && svc.Config.RestartPolicy.MaxRetries > 0 {
			svc.SetStatus(config.StatusFailed)
			svc.LastError = fmt.Errorf("max retries reached: %d", retries)
			return
		}

		delay := utils.BackoffDelay(svc.Config.RestartPolicy.BackoffType,
			svc.Config.RestartPolicy.BackoffDelay,
			svc.Config.RestartPolicy.MaxBackoff,
			retries)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		proc, err := sm.startProcess(ctx, svc)
		if err != nil {
			svc.SetStatus(config.StatusFailed)
			svc.LastError = err
			return
		}
		svc.SetProc(proc)
		svc.SetStatus(config.StatusRunning)
		retries++
	}
}
