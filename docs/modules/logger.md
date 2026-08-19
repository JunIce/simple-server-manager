# 模块：日志管理（internal/logger）

## 职责

实时采集托管进程的 stdout/stderr，环形缓冲缓存、按大小滚动落盘，并提供 **grep/sed 式流式检索**（不建索引）与实时订阅推送。

## 核心类型

### LogManager

```go
type LogManager struct {
    collectors map[string]*LogCollector
    searcher   *LogSearcher
    storage    *LogStorage  // 滚动文本文件存储
}
```

### LogCollector

```go
type LogCollector struct {
    serviceID   string
    pipe        io.ReadCloser
    buffer      *RingBuffer
    subscribers map[chan LogEntry]struct{}
}

type LogEntry struct {
    ServiceID string    `json:"serviceId"`
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Message   string    `json:"message"`
}
```

### RingBuffer

```go
type RingBuffer struct {
    data  []LogEntry
    size  int
    start int
    count int
    mu    sync.RWMutex
}

func (rb *RingBuffer) Enqueue(e LogEntry) {
    rb.mu.Lock()
    defer rb.mu.Unlock()
    if rb.count < rb.size {
        rb.data[(rb.start+rb.count)%rb.size] = e
        rb.count++
    } else {
        rb.data[rb.start] = e          // 覆盖最旧
        rb.start = (rb.start + 1) % rb.size
    }
}
```

## 采集流程

```go
func (lc *LogCollector) Start(ctx context.Context) {
    scanner := bufio.NewScanner(lc.pipe)
    scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

    for scanner.Scan() {
        line := scanner.Text()
        entry := parseLogEntry(line)
        lc.buffer.Enqueue(entry)      // 环形缓冲
        lc.broadcast(entry)           // 实时推送
        lc.storage.Append(entry)      // 滚动落盘
    }
}
```

- `scanner.Buffer` 扩大单行上限到 1MB，避免超长日志行截断。

## 日志解析

```go
func parseLogEntry(line string) LogEntry {
    // 尝试解析常见格式：时间戳、级别 [INFO] 等
    // 无法解析时 Level 默认 "INFO"，Timestamp 取当前时间
    return LogEntry{ Timestamp: time.Now(), Message: line }
}
```

## 日志落盘（滚动文本文件）

```go
type LogStorage struct {
    dir      string
    maxSize  int64   // 单文件最大字节数
    maxFiles int     // 保留文件数
}

// 文件命名：<serviceID>.log, <serviceID>.log.1, <serviceID>.log.2 ...
func (s *LogStorage) Append(e LogEntry) {
    f := s.currentFile(e.ServiceID)
    if f.size() >= s.maxSize {
        s.rotate(e.ServiceID)
        f = s.currentFile(e.ServiceID)
    }
    f.write(e)
}
```

- 日志以纯文本按行写入，可按大小滚动，便于 grep/sed 检索与人工查看。

## 日志检索（grep/sed 式，无索引）

```go
type LogQuery struct {
    Keywords      []string  `json:"keywords"`
    Level         string    `json:"level"`
    StartTime     time.Time `json:"startTime"`
    EndTime       time.Time `json:"endTime"`
    Regex         string    `json:"regex"`
    CaseSensitive bool      `json:"caseSensitive"`
    MaxResults    int       `json:"maxResults"`
}

func (ls *LogSearcher) Search(serviceID string, q LogQuery) ([]LogEntry, error) {
    // 1. 依据 StartTime/EndTime 裁剪需要扫描的滚动文件集合
    files := ls.storage.listFiles(serviceID, q.StartTime, q.EndTime)

    // 2. 逐文件流式扫描，逐行匹配（类似 grep）
    var results []LogEntry
    for _, f := range files {
        scanner := bufio.NewScanner(f.open())
        for scanner.Scan() {
            if !match(scanner.Text(), q) {
                continue
            }
            results = append(results, parseLogEntry(scanner.Text()))
            if len(results) >= q.MaxResults {
                return results, nil  // 命中提前终止
            }
        }
    }
    return results, nil
}

func match(line string, q LogQuery) bool {
    if !q.CaseSensitive {
        line = strings.ToLower(line)
    }
    if q.Regex != "" {
        re, _ := regexp.Compile(q.Regex)
        if !re.MatchString(line) {
            return false
        }
    }
    for _, kw := range q.Keywords {
        if !strings.Contains(line, strings.ToLower(kw)) {
            return false
        }
    }
    return true
}
```

- 与 `grep`/`sed` 等价：逐行 `strings.Contains` 子串匹配 + 可选 `regexp` 正则，命中即返回，无索引开销。

## 订阅与广播

```go
func (lm *LogManager) Subscribe(serviceID string) (<-chan LogEntry, func()) {
    ch := make(chan LogEntry, 64)
    lm.collectors[serviceID].mu.Lock()
    lm.collectors[serviceID].subscribers[ch] = struct{}{}
    lm.collectors[serviceID].mu.Unlock()
    return ch, func() { lm.Unsubscribe(serviceID, ch) }
}

func (lc *LogCollector) broadcast(e LogEntry) {
    for ch := range lc.subscribers {
        select {
        case ch <- e:
        default: // 慢消费者丢弃
        }
    }
}
```

## 设计要点

- 环形缓冲固定容量，日志量再大也不至于 OOM。
- 检索零索引、零外部依赖，内存友好；大文件通过滚动 + 时间裁剪 + 提前终止优化。
- 订阅者独立缓冲 + 丢弃策略，采集循环不被阻塞。
