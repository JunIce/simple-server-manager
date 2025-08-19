package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ScriptRecord struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ExecutionLog struct {
	ID         int64     `json:"id"`
	ScriptID   int64     `json:"script_id"`
	ScriptName string    `json:"script_name"`
	Output     string    `json:"output"`
	Status     string    `json:"status"`
	ExecutedAt time.Time `json:"executed_at"`
	Duration   int64     `json:"duration_ms"`
}

var db *sql.DB

func initDatabase() error {
	var err error
	db, err = sql.Open("sqlite3", "./scripts.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// Create tables
	if err = createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	return nil
}

func createTables() error {
	// Create scripts table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS scripts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create execution_logs table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS execution_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			script_id INTEGER NOT NULL,
			script_name TEXT NOT NULL,
			output TEXT NOT NULL,
			status TEXT NOT NULL,
			executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			duration_ms INTEGER,
			FOREIGN KEY (script_id) REFERENCES scripts (id)
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_scripts_path ON scripts(path)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_execution_logs_script_id ON execution_logs(script_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_execution_logs_executed_at ON execution_logs(executed_at)`)
	if err != nil {
		return err
	}

	return nil
}

func InsertScript(name, path, content string) (*ScriptRecord, error) {
	stmt, err := db.Prepare(`
		INSERT INTO scripts (name, path, content) 
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(name, path, content)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ScriptRecord{
		ID:      id,
		Name:    name,
		Path:    path,
		Content: content,
	}, nil
}

func GetScriptByPath(path string) (*ScriptRecord, error) {
	row := db.QueryRow(`
		SELECT id, name, path, content, created_at, updated_at 
		FROM scripts 
		WHERE path = ?
	`, path)

	var script ScriptRecord
	err := row.Scan(&script.ID, &script.Name, &script.Path, &script.Content, &script.CreatedAt, &script.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &script, nil
}

func GetAllScripts() ([]ScriptRecord, error) {
	rows, err := db.Query(`
		SELECT id, name, path, content, created_at, updated_at 
		FROM scripts 
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scripts []ScriptRecord
	for rows.Next() {
		var script ScriptRecord
		err := rows.Scan(&script.ID, &script.Name, &script.Path, &script.Content, &script.CreatedAt, &script.UpdatedAt)
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, script)
	}

	return scripts, nil
}

func UpdateScript(id int64, name, path, content string) (*ScriptRecord, error) {
	stmt, err := db.Prepare(`
		UPDATE scripts 
		SET name = ?, path = ?, content = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, path, content, id)
	if err != nil {
		return nil, err
	}

	return GetScriptByPath(path)
}

func DeleteScript(path string) error {
	// First get the script ID for logging
	script, err := GetScriptByPath(path)
	if err != nil {
		return err
	}

	// Delete associated execution logs
	_, err = db.Exec(`DELETE FROM execution_logs WHERE script_id = ?`, script.ID)
	if err != nil {
		return err
	}

	// Delete the script
	_, err = db.Exec(`DELETE FROM scripts WHERE path = ?`, path)
	return err
}

func InsertExecutionLog(scriptID int64, scriptName, output, status string, duration int64) (*ExecutionLog, error) {
	stmt, err := db.Prepare(`
		INSERT INTO execution_logs (script_id, script_name, output, status, duration_ms) 
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(scriptID, scriptName, output, status, duration)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ExecutionLog{
		ID:         id,
		ScriptID:   scriptID,
		ScriptName: scriptName,
		Output:     output,
		Status:     status,
		Duration:   duration,
	}, nil
}

func GetExecutionLogs(scriptID int64, limit int) ([]ExecutionLog, error) {
	query := `
		SELECT id, script_id, script_name, output, status, executed_at, duration_ms 
		FROM execution_logs 
		WHERE script_id = ? 
		ORDER BY executed_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, scriptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ExecutionLog
	for rows.Next() {
		var log ExecutionLog
		err := rows.Scan(&log.ID, &log.ScriptID, &log.ScriptName, &log.Output, &log.Status, &log.ExecutedAt, &log.Duration)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func GetAllExecutionLogs(limit int) ([]ExecutionLog, error) {
	query := `
		SELECT id, script_id, script_name, output, status, executed_at, duration_ms 
		FROM execution_logs 
		ORDER BY executed_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ExecutionLog
	for rows.Next() {
		var log ExecutionLog
		err := rows.Scan(&log.ID, &log.ScriptID, &log.ScriptName, &log.Output, &log.Status, &log.ExecutedAt, &log.Duration)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func CloseDatabase() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// 数据库清理功能
func CleanupOldLogs(days int) error {
	if days <= 0 {
		return nil
	}

	cutoffDate := time.Now().AddDate(0, 0, -days)
	
	_, err := db.Exec(`
		DELETE FROM execution_logs 
		WHERE executed_at < ?
	`, cutoffDate)
	
	if err != nil {
		return fmt.Errorf("failed to cleanup old logs: %v", err)
	}

	// 获取删除的记录数
	var count int
	err = db.QueryRow("SELECT changes()").Scan(&count)
	if err != nil {
		log.Printf("Failed to get deleted count: %v", err)
	} else {
		log.Printf("Cleaned up %d execution logs older than %d days", count, days)
	}

	return nil
}

// 获取数据库统计信息
func GetDatabaseStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 获取脚本总数
	var scriptCount int
	err := db.QueryRow("SELECT COUNT(*) FROM scripts").Scan(&scriptCount)
	if err != nil {
		log.Printf("Failed to get script count: %v", err)
	} else {
		stats["script_count"] = scriptCount
	}

	// 获取执行日志总数
	var logCount int
	err = db.QueryRow("SELECT COUNT(*) FROM execution_logs").Scan(&logCount)
	if err != nil {
		log.Printf("Failed to get log count: %v", err)
	} else {
		stats["log_count"] = logCount
	}

	// 获取成功的执行次数
	var successCount int
	err = db.QueryRow("SELECT COUNT(*) FROM execution_logs WHERE status = 'success'").Scan(&successCount)
	if err != nil {
		log.Printf("Failed to get success count: %v", err)
	} else {
		stats["success_count"] = successCount
	}

	// 获取失败的执行次数
	var errorCount int
	err = db.QueryRow("SELECT COUNT(*) FROM execution_logs WHERE status = 'error'").Scan(&errorCount)
	if err != nil {
		log.Printf("Failed to get error count: %v", err)
	} else {
		stats["error_count"] = errorCount
	}

	// 获取数据库文件大小
	if stat, err := os.Stat("./scripts.db"); err == nil {
		stats["db_size_bytes"] = stat.Size()
		stats["db_size_mb"] = fmt.Sprintf("%.2f", float64(stat.Size())/1024/1024)
	}

	return stats
}

// 数据库备份
func BackupDatabase(backupPath string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 关闭当前连接
	err := db.Close()
	if err != nil {
		return fmt.Errorf("failed to close database: %v", err)
	}

	// 复制数据库文件
	sourcePath := "./scripts.db"
	data, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read database file: %v", err)
	}

	err = ioutil.WriteFile(backupPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write backup file: %v", err)
	}

	// 重新打开数据库
	db, err = sql.Open("sqlite3", sourcePath)
	if err != nil {
		return fmt.Errorf("failed to reopen database: %v", err)
	}

	log.Printf("Database backed up to: %s", backupPath)
	return nil
}

// 数据库恢复
func RestoreDatabase(backupPath string) error {
	if db != nil {
		db.Close()
	}

	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// 复制备份文件
	data, err := ioutil.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %v", err)
	}

	err = ioutil.WriteFile("./scripts.db", data, 0644)
	if err != nil {
		return fmt.Errorf("failed to restore database file: %v", err)
	}

	// 重新打开数据库
	db, err = sql.Open("sqlite3", "./scripts.db")
	if err != nil {
		return fmt.Errorf("failed to reopen database: %v", err)
	}

	log.Printf("Database restored from: %s", backupPath)
	return nil
}