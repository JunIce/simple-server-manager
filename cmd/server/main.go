// Command server 主服务入口。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"service-manager/internal/api"
	"service-manager/internal/config"
	"service-manager/internal/core"
	"service-manager/internal/logger"
	"service-manager/internal/monitor"
	"service-manager/internal/platform"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "server":
		cmdServer(os.Args[2:])
	case "run-service":
		cmdRunService(os.Args[2:])
	case "install-service":
		cmdInstallService(os.Args[2:])
	case "uninstall-service":
		cmdUninstallService(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(`service-manager - Windows 服务管理系统

用法:
  service-manager server [--port N] [--host H] [--config PATH]
  service-manager run-service <serviceID> [--config PATH]
  service-manager install-service <serviceID> [--config PATH]
  service-manager uninstall-service <serviceID> [--config PATH]`)
}

// cmdServer 主服务。
func cmdServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	port := fs.Int("port", 0, "listen port (default from config)")
	host := fs.String("host", "", "listen host (default from config)")
	cfgPath := fs.String("config", "", "config file path")
	fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *host != "" {
		cfg.Server.Host = *host
	}

	pl := platform.New()

	lg, err := logger.New(cfg.Log)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	mon := monitor.New(pl, time.Duration(cfg.Monitor.IntervalSeconds)*time.Second)

	sm := core.New(core.Options{
		Config:   cfg,
		Platform: pl,
		Logger:   lg,
		Monitor:  mon,
	})

	srv := api.New(api.Options{
		Config:   cfg,
		Manager:  sm,
		Logger:   lg,
		Monitor:  mon,
		Platform: pl,
	})

	// 优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动主机资源监控
	mon.WatchHost(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
		shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shCancel()
		_ = srv.Shutdown(shCtx)
		stopAll(sm)
	}()

	log.Printf("service-manager listening on %s:%d (config: %s)", cfg.Server.Host, cfg.Server.Port, cfg.Path)
	if err := srv.ListenAndServe(cfg.Server.Host, cfg.Server.Port); err != nil {
		log.Printf("server stopped: %v", err)
	}
}

func stopAll(sm *core.ServiceManager) {
	for _, svc := range sm.List() {
		_ = sm.StopService(svc.ID)
	}
}

// cmdRunService 以前台方式运行一个托管服务（供 SCM 服务包装使用）。
func cmdRunService(args []string) {
	fs := flag.NewFlagSet("run-service", flag.ExitOnError)
	cfgPath := fs.String("config", "", "config file path")
	fs.Parse(args)

	if fs.NArg() < 1 {
		log.Fatal("usage: service-manager run-service <serviceID> [--config PATH]")
	}
	serviceID := fs.Arg(0)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.GetService(serviceID); !ok {
		log.Fatalf("service %s not found", serviceID)
	}

	pl := platform.New()
	sm := core.New(core.Options{
		Config:   cfg,
		Platform: pl,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := sm.StartService(ctx, serviceID); err != nil {
		log.Fatalf("start service %s: %v", serviceID, err)
	}
	log.Printf("service %s started (config: %s)", serviceID, cfg.Path)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	_ = sm.StopService(serviceID)
}

// cmdInstallService 安装 Windows 系统服务。
func cmdInstallService(args []string) {
	fs := flag.NewFlagSet("install-service", flag.ExitOnError)
	cfgPath := fs.String("config", "", "config file path")
	fs.Parse(args)

	if fs.NArg() < 1 {
		log.Fatal("usage: service-manager install-service <serviceID> [--config PATH]")
	}
	serviceID := fs.Arg(0)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	svcCfg, ok := cfg.GetService(serviceID)
	if !ok {
		log.Fatalf("service %s not found", serviceID)
	}

	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("get executable: %v", err)
	}

	pl := platform.New()
	if err := pl.InstallService(svcCfg, exe, fmt.Sprintf("run-service %s --config \"%s\"", serviceID, cfg.Path)); err != nil {
		log.Fatalf("install service: %v", err)
	}
	log.Printf("service %s installed", serviceID)
}

// cmdUninstallService 卸载 Windows 系统服务。
func cmdUninstallService(args []string) {
	fs := flag.NewFlagSet("uninstall-service", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		log.Fatal("usage: service-manager uninstall-service <serviceID>")
	}
	serviceID := fs.Arg(0)

	pl := platform.New()
	if err := pl.UninstallService(serviceID); err != nil {
		log.Fatalf("uninstall service: %v", err)
	}
	log.Printf("service %s uninstalled", serviceID)
}
