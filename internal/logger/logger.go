// Package logger 日志管理：实时采集、环形缓冲、滚动落盘、grep/sed 式检索与实时订阅推送。
package logger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"service-manager/internal/config"
	"service-manager/pkg/utils"
)

// LogCollector 单服务日志采集器。
type LogCollector struct {
	serviceID   string
	buffer      *RingBuffer
	subscribers map[chan LogEntry]struct{}
	mu          sync.RWMutex
	storage     *LogStorage
}

// LogManager 日志管理器。
type LogManager struct {
	mu         sync.RWMutex
	collectors map[string]*LogCollector
	storage    *LogStorage
	bufferSize int
}

// New 创建日志管理器。
func New(cfg config.LogConfig) (*LogManager, error) {
	cfg = cfg.Default()
	if err := utils.EnsureDir(cfg.OutputDir); err != nil {
		return nil, err
	}
	storage := &LogStorage{
		dir:      cfg.OutputDir,
		maxSize:  cfg.MaxSizeMB * 1024 * 1024,
		maxFiles: cfg.MaxFiles,
	}
	if err := storage.init(); err != nil {
		return nil, err
	}
	return &LogManager{
		collectors: map[string]*LogCollector{},
		storage:    storage,
		bufferSize: cfg.BufferSize,
	}, nil
}

// StartCollection 开始采集某服务的 stdout/stderr。
func (lm *LogManager) StartCollection(ctx context.Context, serviceID string, stdout, stderr io.Reader) {
	lm.mu.Lock()
	if _, ok := lm.collectors[serviceID]; !ok {
		lm.collectors[serviceID] = &LogCollector{
			serviceID:   serviceID,
			buffer:      NewRingBuffer(lm.bufferSize),
			subscribers: map[chan LogEntry]struct{}{},
			storage:     lm.storage,
		}
	}
	lc := lm.collectors[serviceID]
	lm.mu.Unlock()

	go lc.scan(ctx, stdout)
	go lc.scan(ctx, stderr)
}

// StopCollection 停止采集并移除采集器。
func (lm *LogManager) StopCollection(serviceID string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lc, ok := lm.collectors[serviceID]; ok {
		lc.closeAll()
		delete(lm.collectors, serviceID)
	}
}

// scan 逐行读取并处理。Windows 子进程常输出 GBK 编码，统一转为 UTF-8。
func (lc *LogCollector) scan(ctx context.Context, r io.Reader) {
	if r == nil {
		return
	}
	decoder := newTextDecoder()
	scanner := bufio.NewScanner(decoder.Reader(r))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		entry := parseLogEntry(lc.serviceID, scanner.Text())
		lc.buffer.Enqueue(entry)
		lc.broadcast(entry)
		if lc.storage != nil {
			lc.storage.Append(entry)
		}
	}
}

// Subscribe 订阅实时日志，返回 channel 与取消函数。
func (lm *LogManager) Subscribe(serviceID string) (<-chan LogEntry, func()) {
	ch := make(chan LogEntry, 128)
	lm.mu.Lock()
	lc, ok := lm.collectors[serviceID]
	if !ok {
		lc = &LogCollector{
			serviceID:   serviceID,
			buffer:      NewRingBuffer(lm.bufferSize),
			subscribers: map[chan LogEntry]struct{}{},
			storage:     lm.storage,
		}
		lm.collectors[serviceID] = lc
	}
	lc.mu.Lock()
	lc.subscribers[ch] = struct{}{}
	lc.mu.Unlock()
	lm.mu.Unlock()

	return ch, func() {
		lc.mu.Lock()
		delete(lc.subscribers, ch)
		close(ch)
		lc.mu.Unlock()
	}
}

// Unsubscribe 取消订阅（兼容旧接口）。
func (lm *LogManager) Unsubscribe(serviceID string, ch chan LogEntry) {
	lm.mu.RLock()
	lc, ok := lm.collectors[serviceID]
	lm.mu.RUnlock()
	if !ok {
		return
	}
	lc.mu.Lock()
	delete(lc.subscribers, ch)
	close(ch)
	lc.mu.Unlock()
}

func (lc *LogCollector) broadcast(e LogEntry) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	for ch := range lc.subscribers {
		select {
		case ch <- e:
		default: // 慢消费者丢弃，避免阻塞
		}
	}
}

func (lc *LogCollector) closeAll() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	for ch := range lc.subscribers {
		close(ch)
	}
	lc.subscribers = map[chan LogEntry]struct{}{}
}

// Recent 返回内存中最近 n 条日志。
func (lm *LogManager) Recent(serviceID string, n int) []LogEntry {
	lm.mu.RLock()
	lc, ok := lm.collectors[serviceID]
	lm.mu.RUnlock()
	if !ok {
		return nil
	}
	return lc.buffer.Recent(n)
}

// Search 搜索日志。
func (lm *LogManager) Search(serviceID string, q LogQuery) ([]LogEntry, error) {
	return lm.storage.Search(serviceID, q)
}

// parseLogEntry 解析单行日志。
func parseLogEntry(serviceID, line string) LogEntry {
	level := utils.ParseLogLevel(line)
	return LogEntry{
		ServiceID: serviceID,
		Timestamp: time.Now(),
		Level:     level,
		Message:   line,
	}
}

// formatEntry 序列化为一行。
func formatEntry(e LogEntry) string {
	return fmt.Sprintf("%s [%s] %s\n", e.Timestamp.Format(time.RFC3339Nano), e.Level, e.Message)
}

// LogQuery 日志查询参数。
type LogQuery struct {
	Keywords      []string  `json:"keywords"`
	Level         string    `json:"level"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	Regex         string    `json:"regex"`
	CaseSensitive bool      `json:"caseSensitive"`
	MaxResults    int       `json:"maxResults"`
}

// LogStorage 滚动文本文件存储。
type LogStorage struct {
	dir      string
	maxSize  int64
	maxFiles int
	mu       sync.Mutex
}

func (s *LogStorage) init() error {
	return utils.EnsureDir(s.dir)
}

func (s *LogStorage) filePath(serviceID string) string {
	return filepath.Join(s.dir, serviceID+".log")
}

// Append 追加日志，超限滚动。
func (s *LogStorage) Append(e LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.filePath(e.ServiceID)
	if info, err := os.Stat(path); err == nil && info.Size() >= s.maxSize {
		s.rotate(e.ServiceID)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(formatEntry(e))
}

// rotate 滚动：.log → .log.1 → ... 保留 maxFiles 份。
func (s *LogStorage) rotate(serviceID string) {
	// 从最旧开始移动
	for i := s.maxFiles - 1; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", s.filePath(serviceID), i)
		if _, err := os.Stat(old); err == nil {
			os.Remove(old)
		}
	}
	for i := s.maxFiles - 2; i >= 0; i-- {
		src := fmt.Sprintf("%s.%d", s.filePath(serviceID), i)
		if i == 0 {
			src = s.filePath(serviceID)
		}
		dst := fmt.Sprintf("%s.%d", s.filePath(serviceID), i+1)
		if _, err := os.Stat(src); err == nil {
			os.Rename(src, dst)
		}
	}
}

// listFiles 列出某服务的日志文件（正序：最早的在前）。
func (s *LogStorage) listFiles(serviceID string) []string {
	pattern := filepath.Join(s.dir, serviceID+".log*")
	matches, _ := filepath.Glob(pattern)

	// 排序：serviceID.log 为最新，.1 次之……按数字倒序（数字小更新）
	sort.Slice(matches, func(i, j int) bool {
		return fileSuffix(matches[i]) > fileSuffix(matches[j])
	})
	return matches
}

func fileSuffix(path string) int {
	base := filepath.Base(path)
	idx := strings.LastIndex(base, ".log")
	if idx < 0 {
		return 0
	}
	rest := base[idx+4:]
	if rest == "" {
		return -1 // 主文件最新，给最大
	}
	var n int
	fmt.Sscanf(rest, ".%d", &n)
	return n
}

// Search grep/sed 式流式检索，无索引。
func (s *LogStorage) Search(serviceID string, q LogQuery) ([]LogEntry, error) {
	if q.MaxResults <= 0 {
		q.MaxResults = 100
	}
	var re *regexp.Regexp
	if q.Regex != "" {
		var err error
		re, err = regexp.Compile(q.Regex)
		if err != nil {
			return nil, err
		}
	}

	var results []LogEntry
	limit := q.MaxResults

	// 从最新文件开始往回搜，命中即提前终止
	decoder := newTextDecoder()
	files := s.listFiles(serviceID)
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(decoder.Reader(f))
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if matchLine(line, q, re) {
				results = append(results, parseLogEntry(serviceID, line))
				if len(results) >= limit {
					f.Close()
					goto done
				}
			}
		}
		f.Close()
	}

done:
	// 倒序返回（最新的在前）
	reverse(results)
	return results, nil
}

func matchLine(line string, q LogQuery, re *regexp.Regexp) bool {
	// 时间范围
	if e := parseEntryTime(line); e != nil {
		if !q.StartTime.IsZero() && e.Before(q.StartTime) {
			return false
		}
		if !q.EndTime.IsZero() && e.After(q.EndTime) {
			return false
		}
	}
	// 级别
	if q.Level != "" && !strings.EqualFold(utils.ParseLogLevel(line), q.Level) {
		return false
	}
	// 正则
	if re != nil && !re.MatchString(line) {
		return false
	}
	// 关键字
	if !q.CaseSensitive {
		line = strings.ToLower(line)
		for _, kw := range q.Keywords {
			if !strings.Contains(line, strings.ToLower(kw)) {
				return false
			}
		}
	} else {
		for _, kw := range q.Keywords {
			if !strings.Contains(line, kw) {
				return false
			}
		}
	}
	return true
}

// parseEntryTime 从日志行前缀解析时间戳。
func parseEntryTime(line string) *time.Time {
	// 格式：RFC3339Nano 空格开头
	if len(line) < 20 {
		return nil
	}
	spaceIdx := strings.IndexByte(line, ' ')
	if spaceIdx < 10 {
		return nil
	}
	ts := line[:spaceIdx]
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return nil
	}
	return &t
}

func reverse(entries []LogEntry) {
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
}
