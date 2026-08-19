package logger

import (
	"sync"
	"time"
)

// LogEntry 日志条目。
type LogEntry struct {
	ServiceID string    `json:"serviceId"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// RingBuffer 环形缓冲，固定容量，覆盖最旧。
type RingBuffer struct {
	data  []LogEntry
	size  int
	start int
	count int
	mu    sync.RWMutex
}

func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 10000
	}
	return &RingBuffer{data: make([]LogEntry, size), size: size}
}

func (rb *RingBuffer) Enqueue(e LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.count < rb.size {
		rb.data[(rb.start+rb.count)%rb.size] = e
		rb.count++
	} else {
		rb.data[rb.start] = e
		rb.start = (rb.start + 1) % rb.size
	}
}

// Snapshot 返回全部条目（按时间正序）。
func (rb *RingBuffer) Snapshot() []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	out := make([]LogEntry, 0, rb.count)
	for i := 0; i < rb.count; i++ {
		out = append(out, rb.data[(rb.start+i)%rb.size])
	}
	return out
}

// Recent 返回最近 n 条（正序）。
func (rb *RingBuffer) Recent(n int) []LogEntry {
	if n <= 0 {
		return nil
	}
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	if n > rb.count {
		n = rb.count
	}
	out := make([]LogEntry, 0, n)
	start := rb.start + rb.count - n
	for i := 0; i < n; i++ {
		out = append(out, rb.data[(start+i)%rb.size])
	}
	return out
}
