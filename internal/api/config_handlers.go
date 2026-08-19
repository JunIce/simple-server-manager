package api

import (
	"encoding/json"
	"net/http"

	"service-manager/internal/config"
)

// configView 统一系统配置视图（不含服务定义）。
type configView struct {
	Server  config.ServerConfig `json:"server"`
	Log     config.LogConfig    `json:"log"`
	Monitor config.MonitorConfig `json:"monitor"`
	JDKs    []config.JDK        `json:"jdks"`
	Path    string              `json:"configPath"`
}

// getConfig 读取统一系统配置。
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	jdks := s.cfg.AllJDKs()
	list := make([]config.JDK, 0, len(jdks))
	for _, j := range jdks {
		list = append(list, j)
	}
	writeJSON(w, configView{
		Server:  s.cfg.GetServer(),
		Log:     s.cfg.GetLog(),
		Monitor: s.cfg.GetMonitor(),
		JDKs:    list,
		Path:    s.cfg.Path,
	})
}

// updateConfig 整体更新统一系统配置（服务端/日志/监控/JDK）。
func (s *Server) updateConfig(w http.ResponseWriter, r *http.Request) {
	var v configView
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		writeError(w, http.StatusBadRequest, 1002, "invalid request body: "+err.Error())
		return
	}
	if err := s.cfg.UpdateSystem(v.Server, v.Log, v.Monitor, v.JDKs); err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	writeJSON(w, v)
}

// discoverJDKs 自动发现已安装 JDK。
func (s *Server) discoverJDKs(w http.ResponseWriter, r *http.Request) {
	found, err := config.DiscoverJDKs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	writeJSON(w, found)
}
