# 模块：HTTP API（internal/api）

## 职责

对外暴露 REST API 与 WebSocket 端点，作为所有管理操作的统一入口。

## 路由定义

```go
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

    // 目录浏览（注册服务/配置时选择文件夹）
    api.HandleFunc("/dirs", s.listDirs).Methods("GET")

    // 文件目录（基于配置的 fileRoot，防路径穿越）
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

    // 静态资源（Web UI，React 构建产物）
    s.router.PathPrefix("/").Handler(http.FileServer(http.FS(subFS)))
}
```

## 统一响应体

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "requestId": "uuid",
  "timestamp": 1715788800
}
```

```go
type Response struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data"`
    RequestID string      `json:"requestId"`
    Timestamp int64       `json:"timestamp"`
}

func writeJSON(w http.ResponseWriter, code int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(Response{
        Code:      0,
        Message:   "success",
        Data:      data,
        RequestID: uuid.New().String(),
        Timestamp: time.Now().Unix(),
    })
}
```

## 错误码约定

| 范围 | 含义 |
|------|------|
| 0 | 成功 |
| 4xx | 客户端错误（参数校验、资源不存在） |
| 5xx | 服务端错误（内部异常） |

业务错误码（`code` 字段）：
- `1001` 服务不存在
- `1002` 配置校验失败
- `1003` 状态冲突
- `2001` 启动失败
- `2002` 停止失败
- `3001` JDK 不存在
- `3002` JDK 校验失败（配置页保存时的原子写/校验错误）

## WebSocket 实现

```go
var upgrader = websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool { return true } }

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil { return }
    defer conn.Close()

    serviceID := r.URL.Query().Get("serviceId")
    path := r.URL.Path  // /ws/logs 或 /ws/metrics

    switch path {
    case "/ws/logs":
        s.streamLogs(conn, serviceID, r)
    case "/ws/metrics":
        s.streamMetrics(conn, serviceID, r)
    }
}

func (s *Server) streamLogs(conn *websocket.Conn, serviceID string, r *http.Request) {
    logCh := s.logManager.Subscribe(serviceID)
    defer s.logManager.Unsubscribe(serviceID, logCh)
    for {
        select {
        case e := <-logCh:
            conn.WriteJSON(map[string]interface{}{ "type": "log", "data": e })
        case <-r.Context().Done():
            return
        }
    }
}
```

## 中间件链

```go
func (s *Server) middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 预留：鉴权 (JWT)、限流、审计、panic 恢复
        next.ServeHTTP(w, r)
    })
}
```

## 请求校验

- 注册/更新服务时校验 `ServiceConfig`（ID 唯一、命令非空、类型合法）。
- 日志检索参数 `maxResults` 限上限，防止超大查询。
- `requestId` 全链路透传，便于排障。
