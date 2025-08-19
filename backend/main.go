package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

type FileInfo struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	Path    string `json:"path"`
}

type PortInfo struct {
	Port      string `json:"port"`
	Process   string `json:"process"`
	PID       string `json:"pid"`
	Listening bool   `json:"listening"`
}

type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	PID    string `json:"pid"`
	Memory string `json:"memory"`
	CPU    string `json:"cpu"`
}

type DockerScript struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	Exists    bool      `json:"exists"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ContainerInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Ports   string `json:"ports"`
	Created string `json:"created"`
}

type ImageInfo struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	Created    string `json:"created"`
}

func main() {
	// 初始化数据库
	if err := initDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer CloseDatabase()

	r := mux.NewRouter()

	// 静态文件服务
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("../frontend/static/"))))

	// 主页
	r.HandleFunc("/", serveHome).Methods("GET")

	// 目录管理API
	r.HandleFunc("/api/directory", getDirectoryList).Methods("GET")
	r.HandleFunc("/api/directory", createDirectory).Methods("POST")
	r.HandleFunc("/api/directory", deleteDirectory).Methods("DELETE")
	r.HandleFunc("/api/directory/rename", renameDirectory).Methods("PUT")

	// Docker脚本管理API
	r.HandleFunc("/api/docker-scripts", getDockerScripts).Methods("GET")
	r.HandleFunc("/api/docker-scripts", createDockerScript).Methods("POST")
	r.HandleFunc("/api/docker-scripts", deleteDockerScript).Methods("DELETE")
	r.HandleFunc("/api/docker-scripts/execute", executeDockerScript).Methods("POST")
	r.HandleFunc("/api/docker-scripts/logs", getExecutionLogs).Methods("GET")
	r.HandleFunc("/api/docker-scripts/logs/all", getAllExecutionLogs).Methods("GET")

	// 数据库管理API
	r.HandleFunc("/api/database/stats", getDatabaseStats).Methods("GET")
	r.HandleFunc("/api/database/cleanup", cleanupDatabase).Methods("POST")
	r.HandleFunc("/api/database/backup", backupDatabase).Methods("POST")
	r.HandleFunc("/api/database/restore", restoreDatabase).Methods("POST")

	// 端口管理API
	r.HandleFunc("/api/ports", getPortInfo).Methods("GET")

	// 服务管理API
	r.HandleFunc("/api/services", getServiceStatus).Methods("GET")
	r.HandleFunc("/api/services/start", startService).Methods("POST")
	r.HandleFunc("/api/services/stop", stopService).Methods("POST")
	r.HandleFunc("/api/services/restart", restartService).Methods("POST")

	// 容器管理API
	r.HandleFunc("/api/containers", getContainers).Methods("GET")
	r.HandleFunc("/api/containers/start", startContainer).Methods("POST")
	r.HandleFunc("/api/containers/stop", stopContainer).Methods("POST")
	r.HandleFunc("/api/containers/restart", restartContainer).Methods("POST")
	r.HandleFunc("/api/containers/remove", removeContainer).Methods("POST")
	r.HandleFunc("/api/containers/logs", getContainerLogs).Methods("POST")

	// 镜像管理API
	r.HandleFunc("/api/images", getImages).Methods("GET")
	r.HandleFunc("/api/images/remove", removeImage).Methods("POST")
	r.HandleFunc("/api/images/pull", pullImage).Methods("POST")
	r.HandleFunc("/api/images/prune", pruneImages).Methods("POST")

	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "../frontend/templates/index.html")
}

func getDirectoryList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var fileInfos []FileInfo
	for _, file := range files {
		fileInfo := FileInfo{
			Name:    file.Name(),
			IsDir:   file.IsDir(),
			Size:    file.Size(),
			ModTime: file.ModTime().Format("2006-01-02 15:04:05"),
			Path:    filepath.Join(path, file.Name()),
		}
		fileInfos = append(fileInfos, fileInfo)
	}

	json.NewEncoder(w).Encode(fileInfos)
}

func createDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(req.Path, req.Name)
	if err := os.Mkdir(fullPath, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func deleteDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := os.RemoveAll(req.Path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func renameDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPath string `json:"oldPath"`
		NewPath string `json:"newPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := os.Rename(req.OldPath, req.NewPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getDockerScripts(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}

	var scripts []DockerScript

	// 首先从数据库获取脚本
	dbScripts, err := GetAllScripts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 转换数据库脚本为DockerScript格式
	for _, dbScript := range dbScripts {
		scripts = append(scripts, DockerScript{
			ID:        dbScript.ID,
			Name:      dbScript.Name,
			Path:      dbScript.Path,
			Content:   dbScript.Content,
			Exists:    true,
			CreatedAt: dbScript.CreatedAt,
			UpdatedAt: dbScript.UpdatedAt,
		})
	}

	// 如果指定了路径，也扫描文件系统中的脚本（用于向后兼容）
	if _, err := os.Stat(path); err == nil {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && (strings.HasSuffix(strings.ToLower(info.Name()), ".sh") ||
				strings.Contains(strings.ToLower(info.Name()), "docker")) {

				// 检查是否已经在数据库中
				found := false
				for _, script := range scripts {
					if script.Path == filePath {
						found = true
						break
					}
				}

				if !found {
					content, err := ioutil.ReadFile(filePath)
					if err != nil {
						// 如果无法读取文件内容，仍然添加脚本信息，但内容为空
						script := DockerScript{
							Name:    info.Name(),
							Path:    filePath,
							Content: "",
							Exists:  false,
						}
						scripts = append(scripts, script)
						return nil
					}

					script := DockerScript{
						Name:    info.Name(),
						Path:    filePath,
						Content: string(content),
						Exists:  true,
					}
					scripts = append(scripts, script)
				}
			}
			return nil
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 确保总是返回数组，即使是空的
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scripts)
}

func createDockerScript(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Name    string `json:"name"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(req.Path, req.Name)

	// 保存到文件系统
	if err := ioutil.WriteFile(fullPath, []byte(req.Content), 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 保存到数据库
	scriptRecord, err := InsertScript(req.Name, fullPath, req.Content)
	if err != nil {
		// 如果数据库保存失败，仍然返回成功，但记录错误日志
		log.Printf("Failed to save script to database: %v", err)
	}

	// 返回创建的脚本信息
	response := map[string]interface{}{
		"id":         scriptRecord.ID,
		"name":       scriptRecord.Name,
		"path":       scriptRecord.Path,
		"content":    scriptRecord.Content,
		"created_at": scriptRecord.CreatedAt,
		"message":    "Script created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func deleteDockerScript(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 从数据库删除
	if err := DeleteScript(req.Path); err != nil {
		log.Printf("Failed to delete script from database: %v", err)
	}

	// 从文件系统删除
	if err := os.Remove(req.Path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func executeDockerScript(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 获取脚本信息
	script, err := GetScriptByPath(req.Path)
	if err != nil {
		log.Printf("Failed to get script from database: %v", err)
	}

	// 执行脚本
	startTime := time.Now()
	cmd := exec.Command("bash", req.Path)
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime).Milliseconds()

	// 记录执行日志
	status := "success"
	if err != nil {
		status = "error"
	}

	scriptName := "Unknown"
	var scriptID int64 = 0
	if script != nil {
		scriptName = script.Name
		scriptID = script.ID
	}

	// 保存执行日志到数据库
	_, err = InsertExecutionLog(scriptID, scriptName, string(output), status, duration)
	if err != nil {
		log.Printf("Failed to save execution log: %v", err)
	}

	response := map[string]interface{}{
		"output":     string(output),
		"status":     status,
		"duration":   duration,
		"script_id":  scriptID,
		"script_name": scriptName,
		"executed_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getPortInfo(w http.ResponseWriter, r *http.Request) {
	// 使用netstat或ss命令获取端口信息
	cmd := exec.Command("netstat", "-tlnp")
	output, err := cmd.Output()
	if err != nil {
		// 如果netstat不可用，尝试使用ss
		cmd = exec.Command("ss", "-tlnp")
		output, err = cmd.Output()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	lines := strings.Split(string(output), "\n")
	var ports []PortInfo

	for _, line := range lines {
		if strings.Contains(line, "LISTEN") {
			fields := strings.Fields(line)
			if len(fields) >= 7 {
				addr := fields[3]
				processInfo := fields[6]

				// 提取端口号
				port := ""
				if strings.Contains(addr, ":") {
					parts := strings.Split(addr, ":")
					port = parts[len(parts)-1]
				}

				// 提取进程信息
				pid := ""
				process := ""
				if strings.Contains(processInfo, "/") {
					parts := strings.Split(processInfo, "/")
					pid = parts[0]
					process = strings.Join(parts[1:], "/")
				}

				if port != "" {
					ports = append(ports, PortInfo{
						Port:      port,
						Process:   process,
						PID:       pid,
						Listening: true,
					})
				}
			} else if len(fields) >= 4 {
				// 如果进程信息不存在，只提取端口信息
				addr := fields[3]
				port := ""
				if strings.Contains(addr, ":") {
					parts := strings.Split(addr, ":")
					port = parts[len(parts)-1]
				}

				if port != "" {
					ports = append(ports, PortInfo{
						Port:      port,
						Process:   "N/A",
						PID:       "N/A",
						Listening: true,
					})
				}
			}
		}
	}

	json.NewEncoder(w).Encode(ports)
}

func getServiceStatus(w http.ResponseWriter, r *http.Request) {
	// 使用systemctl获取服务状态（如果可用）
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-pager")
	output, err := cmd.Output()

	var services []ServiceStatus

	if err == nil {
		// 解析systemctl输出
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, ".service") && !strings.Contains(line, "●") {
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					name := fields[0]
					status := fields[2]
					if status == "running" {
						status = "active"
					} else if status == "dead" {
						status = "inactive"
					}

					// 获取进程ID
					pid := ""
					if status == "active" {
						pidCmd := exec.Command("systemctl", "show", "--property=MainPID", name)
						pidOutput, err := pidCmd.Output()
						if err == nil {
							pid = strings.TrimSpace(strings.TrimPrefix(string(pidOutput), "MainPID="))
						}
					}

					services = append(services, ServiceStatus{
						Name:   name,
						Status: status,
						PID:    pid,
						Memory: "N/A",
						CPU:    "N/A",
					})
				}
			}
		}
	} else {
		// 如果systemctl不可用，使用ps命令获取进程信息
		cmd = exec.Command("ps", "aux")
		output, err := cmd.Output()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 11 {
				pid := fields[1]
				cpu := fields[2]
				memory := fields[3]
				process := strings.Join(fields[10:], " ")

				services = append(services, ServiceStatus{
					Name:   process,
					Status: "running",
					PID:    pid,
					Memory: memory,
					CPU:    cpu,
				})
			}
		}
	}

	json.NewEncoder(w).Encode(services)
}

func startService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("systemctl", "start", req.Name)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func stopService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("systemctl", "stop", req.Name)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func restartService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("systemctl", "restart", req.Name)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// 容器管理功能
func getContainers(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}\t{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var containers []ContainerInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		fields := strings.Split(line, "\t")
		if len(fields) >= 6 {
			containers = append(containers, ContainerInfo{
				ID:      fields[0],
				Name:    fields[1],
				Image:   fields[2],
				Status:  fields[3],
				Ports:   fields[4],
				Created: fields[5],
			})
		}
	}

	json.NewEncoder(w).Encode(containers)
}

func startContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("docker", "start", req.ID)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func stopContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("docker", "stop", req.ID)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func restartContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("docker", "restart", req.ID)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func removeContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("docker", "rm", "-f", req.ID)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getContainerLogs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("docker", "logs", req.ID)
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"logs": string(output),
	}

	json.NewEncoder(w).Encode(response)
}

// 镜像管理功能
func getImages(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("docker", "images", "--format", "{{.ID}}\t{{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var images []ImageInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		fields := strings.Split(line, "\t")
		if len(fields) >= 5 {
			images = append(images, ImageInfo{
				ID:         fields[0],
				Repository: fields[1],
				Tag:        fields[2],
				Size:       fields[3],
				Created:    fields[4],
			})
		}
	}

	json.NewEncoder(w).Encode(images)
}

func removeImage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("docker", "rmi", "-f", req.ID)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func pullImage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Image string `json:"image"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command("docker", "pull", req.Image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error pulling image: %v\nOutput: %s", err, string(output)), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"output": string(output),
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

func pruneImages(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("docker", "image", "prune", "-f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error pruning images: %v\nOutput: %s", err, string(output)), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"output": string(output),
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

// 获取执行日志
func getExecutionLogs(w http.ResponseWriter, r *http.Request) {
	scriptID := r.URL.Query().Get("script_id")
	limit := 10 // 默认限制10条记录

	if l := r.URL.Query().Get("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			limit = 10
		}
	}

	if scriptID == "" {
		http.Error(w, "script_id parameter is required", http.StatusBadRequest)
		return
	}

	var id int64
	_, err := fmt.Sscanf(scriptID, "%d", &id)
	if err != nil {
		http.Error(w, "invalid script_id format", http.StatusBadRequest)
		return
	}

	logs, err := GetExecutionLogs(id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// 获取所有执行日志
func getAllExecutionLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50 // 默认限制50条记录

	if l := r.URL.Query().Get("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			limit = 50
		}
	}

	logs, err := GetAllExecutionLogs(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// 数据库统计信息
func getDatabaseStats(w http.ResponseWriter, r *http.Request) {
	stats := GetDatabaseStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// 清理数据库
func cleanupDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Days int `json:"days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Days <= 0 {
		req.Days = 30 // 默认清理30天前的数据
	}

	if err := CleanupOldLogs(req.Days); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": fmt.Sprintf("Successfully cleaned up logs older than %d days", req.Days),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 备份数据库
func backupDatabase(w http.ResponseWriter, r *http.Request) {
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("./scripts_backup_%s.db", timestamp)

	if err := BackupDatabase(backupPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message":    "Database backup completed successfully",
		"backup_path": backupPath,
		"timestamp":  timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 恢复数据库
func restoreDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BackupPath string `json:"backup_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.BackupPath == "" {
		http.Error(w, "backup_path is required", http.StatusBadRequest)
		return
	}

	if err := RestoreDatabase(req.BackupPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message":    "Database restore completed successfully",
		"backup_path": req.BackupPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
