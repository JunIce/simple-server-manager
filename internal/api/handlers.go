package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"service-manager/internal/config"
	"service-manager/internal/core"
	"service-manager/internal/logger"
)

// serviceView 服务对外视图。
type serviceView struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Type        config.ServiceType   `json:"type"`
	Config      config.ServiceConfig `json:"config"`
	Status      config.ServiceStatus `json:"status"`
	PID         int                  `json:"pid"`
	Environment string               `json:"environment"`
	LastError   string               `json:"lastError,omitempty"`
	StartedAt   time.Time            `json:"startedAt,omitempty"`
}

// listServices 服务列表。
func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	services := s.manager.List()
	views := make([]serviceView, 0, len(services))
	for _, svc := range services {
		views = append(views, buildView(svc))
	}
	writeJSON(w, views)
}

// createService 注册服务。
func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	var svc config.ServiceConfig
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		writeError(w, http.StatusBadRequest, 1002, "invalid request body: "+err.Error())
		return
	}
	if err := s.manager.Register(svc); err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	writeJSON(w, svc)
}

// getService 服务详情。
func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	svc, err := s.manager.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, 1001, err.Error())
		return
	}
	writeJSON(w, buildView(svc))
}

// updateService 更新服务配置。
func (s *Server) updateService(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var svc config.ServiceConfig
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		writeError(w, http.StatusBadRequest, 1002, "invalid request body: "+err.Error())
		return
	}
	if err := s.manager.Update(id, svc); err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	writeJSON(w, svc)
}

// deleteService 注销服务。
func (s *Server) deleteService(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.manager.Unregister(id); err != nil {
		writeError(w, http.StatusBadRequest, 1001, err.Error())
		return
	}
	writeJSON(w, map[string]string{"id": id, "deleted": "true"})
}

// startService 启动服务。
func (s *Server) startService(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.manager.StartService(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, 2001, err.Error())
		return
	}
	svc, _ := s.manager.Get(id)
	writeJSON(w, buildView(svc))
}

// stopService 停止服务。
func (s *Server) stopService(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.manager.StopService(id); err != nil {
		writeError(w, http.StatusBadRequest, 2002, err.Error())
		return
	}
	svc, _ := s.manager.Get(id)
	writeJSON(w, buildView(svc))
}

// restartService 重启服务。
func (s *Server) restartService(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := s.manager.RestartService(ctx, id); err != nil {
		writeError(w, http.StatusBadRequest, 2002, err.Error())
		return
	}
	svc, _ := s.manager.Get(id)
	writeJSON(w, buildView(svc))
}

// getStatus 服务状态。
func (s *Server) getStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	svc, err := s.manager.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, 1001, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{
		"id":     id,
		"status": svc.GetStatus(),
		"pid":    svc.GetPID(),
	})
}

// getLogs 日志检索（grep 式）。
func (s *Server) getLogs(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	q := logger.LogQuery{
		Keywords:   r.URL.Query()["keyword"],
		Level:      r.URL.Query().Get("level"),
		Regex:      r.URL.Query().Get("regex"),
		MaxResults: 100,
	}
	if r.URL.Query().Get("caseSensitive") == "true" {
		q.CaseSensitive = true
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			q.MaxResults = n
		}
	}
	if v := r.URL.Query().Get("startTime"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q.StartTime = t
		}
	}
	if v := r.URL.Query().Get("endTime"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q.EndTime = t
		}
	}

	// 无过滤条件时直接返回内存最近日志
	if len(q.Keywords) == 0 && q.Level == "" && q.Regex == "" && q.StartTime.IsZero() && q.EndTime.IsZero() {
		recent := s.logger.Recent(id, q.MaxResults)
		writeJSON(w, recent)
		return
	}
	results, err := s.logger.Search(id, q)
	if err != nil {
		writeError(w, http.StatusBadRequest, 2001, err.Error())
		return
	}
	writeJSON(w, results)
}

// getMetrics 服务指标。
func (s *Server) getMetrics(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	metrics, ok := s.monitor.Get(id)
	if !ok {
		writeJSON(w, map[string]interface{}{"id": id, "metrics": nil})
		return
	}
	writeJSON(w, map[string]interface{}{"id": id, "metrics": metrics})
}

// getSystemInfo 系统信息。
func (s *Server) getSystemInfo(w http.ResponseWriter, r *http.Request) {
	info, err := s.platform.GetSystemInfo()
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	writeJSON(w, info)
}

// getSystemStats 主机实时资源统计。
func (s *Server) getSystemStats(w http.ResponseWriter, r *http.Request) {
	stats, ok := s.monitor.GetHostStats()
	if !ok {
		stats, err := s.platform.GetSystemStats()
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return
		}
		writeJSON(w, stats)
		return
	}
	writeJSON(w, stats)
}

// getTopProcesses 按 CPU 降序返回 TopN 进程。
func (s *Server) getTopProcesses(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	procs, err := s.platform.GetTopProcesses(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	writeJSON(w, procs)
}

// killProcess 强制终止进程。
func (s *Server) killProcess(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["pid"]
	pid, err := strconv.Atoi(idStr)
	if err != nil || pid <= 0 {
		writeError(w, http.StatusBadRequest, 1002, "invalid pid")
		return
	}
	if err := s.platform.KillProcess(pid); err != nil {
		writeError(w, http.StatusBadRequest, 2002, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"pid": pid, "killed": true})
}

// buildView 构建服务视图。
func buildView(svc *core.ManagedService) serviceView {
	view := serviceView{
		ID:          svc.ID,
		Name:        svc.Config.Name,
		Type:        svc.Config.Type,
		Config:      svc.Config,
		Status:      svc.GetStatus(),
		PID:         svc.GetPID(),
		Environment: svc.Environment,
		StartedAt:   svc.GetStartTime(),
	}
	if svc.LastError != nil {
		view.LastError = svc.LastError.Error()
	}
	return view
}
