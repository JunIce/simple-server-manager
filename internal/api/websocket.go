package api

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleWebSocket 统一入口，按路径分流。
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	serviceID := r.URL.Query().Get("serviceId")
	switch r.URL.Path {
	case "/ws/logs":
		s.streamLogs(conn, serviceID, r)
	case "/ws/metrics":
		s.streamMetrics(conn, serviceID, r)
	case "/ws/system/stats":
		s.streamHostStats(conn, r)
	}
}

// streamLogs 实时日志流。
func (s *Server) streamLogs(conn *websocket.Conn, serviceID string, r *http.Request) {
	logCh, unsubscribe := s.logger.Subscribe(serviceID)
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-r.Context().Done():
			return
		case e := <-logCh:
			writeWS(conn, "log", e)
		}
	}
}

// streamMetrics 实时指标流。
func (s *Server) streamMetrics(conn *websocket.Conn, serviceID string, r *http.Request) {
	metricsCh, unsubscribe := s.monitor.Subscribe(serviceID)
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-r.Context().Done():
			return
		case u := <-metricsCh:
			writeWS(conn, "metrics", u)
		}
	}
}

// streamHostStats 实时主机资源统计流。
func (s *Server) streamHostStats(conn *websocket.Conn, r *http.Request) {
	statsCh, unsubscribe := s.monitor.SubscribeHost()
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-r.Context().Done():
			return
		case u := <-statsCh:
			writeWS(conn, "stats", u.Stats)
		}
	}
}

func writeWS(conn *websocket.Conn, msgType string, data interface{}) {
	_ = conn.WriteJSON(map[string]interface{}{
		"type": msgType,
		"data": data,
	})
}
