// Package api 对外暴露 REST API 与 WebSocket，作为所有管理操作的统一入口。
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"service-manager/internal/config"
	"service-manager/internal/core"
	"service-manager/internal/logger"
	"service-manager/internal/monitor"
	"service-manager/internal/platform"
	"service-manager/internal/webui"
)

// Server API 服务器。
type Server struct {
	cfg      *config.Config
	manager  *core.ServiceManager
	logger   *logger.LogManager
	monitor  *monitor.Monitor
	platform platform.Platform
	router   *mux.Router
	httpSrv  *http.Server
}

// Options 创建 API Server 的选项。
type Options struct {
	Config   *config.Config
	Manager  *core.ServiceManager
	Logger   *logger.LogManager
	Monitor  *monitor.Monitor
	Platform platform.Platform
}

// New 创建 API Server。
func New(opts Options) *Server {
	s := &Server{
		cfg:      opts.Config,
		manager:  opts.Manager,
		logger:   opts.Logger,
		monitor:  opts.Monitor,
		platform: opts.Platform,
		router:   mux.NewRouter(),
	}
	s.registerRoutes()
	return s
}

// Handler 返回 HTTP Handler。
func (s *Server) Handler() http.Handler {
	return s.middleware(s.router)
}

// ListenAndServe 启动 HTTP 服务。
func (s *Server) ListenAndServe(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}
	return s.httpSrv.ListenAndServe()
}

// Shutdown 优雅关闭。
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

// registerRoutes 注册路由。
func (s *Server) registerRoutes() {
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/services", s.listServices).Methods("GET")
	api.HandleFunc("/services", s.createService).Methods("POST")
	api.HandleFunc("/services/{id}", s.getService).Methods("GET")
	api.HandleFunc("/services/{id}", s.updateService).Methods("PUT")
	api.HandleFunc("/services/{id}", s.deleteService).Methods("DELETE")
	api.HandleFunc("/services/{id}/start", s.startService).Methods("POST")
	api.HandleFunc("/services/{id}/stop", s.stopService).Methods("POST")
	api.HandleFunc("/services/{id}/restart", s.restartService).Methods("POST")
	api.HandleFunc("/services/{id}/status", s.getStatus).Methods("GET")
	api.HandleFunc("/services/{id}/logs", s.getLogs).Methods("GET")
	api.HandleFunc("/services/{id}/metrics", s.getMetrics).Methods("GET")
	api.HandleFunc("/system/info", s.getSystemInfo).Methods("GET")
	api.HandleFunc("/system/stats", s.getSystemStats).Methods("GET")
	api.HandleFunc("/system/topcpu", s.getTopProcesses).Methods("GET")
	api.HandleFunc("/system/kill/{pid}", s.killProcess).Methods("POST")

	// 统一系统配置（服务端/日志/监控/JDK）
	api.HandleFunc("/config", s.getConfig).Methods("GET")
	api.HandleFunc("/config", s.updateConfig).Methods("PUT")
	api.HandleFunc("/config/discover-jdks", s.discoverJDKs).Methods("POST")
	api.HandleFunc("/dirs", s.listDirs).Methods("GET")
	api.HandleFunc("/files", s.listFiles).Methods("GET")
	api.HandleFunc("/files/content", s.readFileContent).Methods("GET")
	api.HandleFunc("/files/content", s.writeFileContent).Methods("PUT")
	api.HandleFunc("/files/download", s.downloadFile).Methods("GET")
	api.HandleFunc("/files/upload", s.uploadFile).Methods("POST")
	api.HandleFunc("/files/rename", s.renameFile).Methods("POST")
	api.HandleFunc("/files/delete", s.deleteFile).Methods("DELETE")
	api.HandleFunc("/files/mkdir", s.mkdirFile).Methods("POST")
	api.HandleFunc("/files/copy", s.copyFile).Methods("POST")

	s.router.HandleFunc("/ws/logs", s.handleWebSocket)
	s.router.HandleFunc("/ws/metrics", s.handleWebSocket)
	s.router.HandleFunc("/ws/system/stats", s.handleWebSocket)

	// 静态资源（Web UI）+ SPA 回退：未知路径返回 index.html，支持前端路由直达/刷新
	sub, _ := fs.Sub(webui.FS, "static")
	webRoot := http.FileServer(http.FS(sub))
	serveIndex := func(w http.ResponseWriter) {
		data, err := sub.Open("index.html")
		if err != nil {
			http.Error(w, "index not found", http.StatusInternalServerError)
			return
		}
		defer data.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, data)
	}
	s.router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// 命中真实静态文件则直接提供，否则回退 index.html（SPA 前端路由）
			if _, err := fs.Stat(sub, strings.TrimPrefix(r.URL.Path, "/")); err == nil {
				webRoot.ServeHTTP(w, r)
				return
			}
		}
		serveIndex(w)
	}))
}

// middleware 中间件链：预留鉴权、panic 恢复、requestId。
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 鉴权（预留）
		if s.cfg.Server.Auth.Enabled && s.cfg.Server.Auth.Token != "" {
			token := r.Header.Get("Authorization")
			if token != "Bearer "+s.cfg.Server.Auth.Token {
				writeError(w, http.StatusUnauthorized, 401, "unauthorized")
				return
			}
		}
		// panic 恢复
		defer func() {
			if rec := recover(); rec != nil {
				writeError(w, http.StatusInternalServerError, 500, fmt.Sprintf("internal error: %v", rec))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Response 统一响应体。
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"requestId"`
	Timestamp int64       `json:"timestamp"`
}

// writeJSON 写成功响应。
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		RequestID: requestID(),
		Timestamp: time.Now().Unix(),
	})
}

// writeError 写错误响应。
func writeError(w http.ResponseWriter, httpCode, bizCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	json.NewEncoder(w).Encode(Response{
		Code:      bizCode,
		Message:   msg,
		RequestID: requestID(),
		Timestamp: time.Now().Unix(),
	})
}

func requestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
