package middleware

import (
	"context"
	"sync"
	"time"
)

// memCounter is a single fixed-window counter.
type memCounter struct {
	count   int
	resetAt time.Time
}

// MemoryRateLimiter is an in-process fixed-window rate-limit store. Counters
// live in this process only, which is fine for the current single-instance
// deployment. A background sweep drops expired windows so the map cannot grow
// unbounded.
type MemoryRateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	buckets map[string]*memCounter
}

// NewMemoryRateLimiter creates a limiter whose windows are `window` long and
// starts its background cleanup goroutine.
func NewMemoryRateLimiter(window time.Duration) *MemoryRateLimiter {
	m := &MemoryRateLimiter{
		window:  window,
		buckets: make(map[string]*memCounter),
	}
	go m.cleanupLoop()
	return m
}

// Incr satisfies IncrFunc: it increments key's counter, opening a fresh window
// when the previous one has elapsed, and returns the count within the window.
func (m *MemoryRateLimiter) Incr(_ context.Context, key string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	c := m.buckets[key]
	if c == nil || now.After(c.resetAt) {
		c = &memCounter{resetAt: now.Add(m.window)}
		m.buckets[key] = c
	}
	c.count++
	return c.count, nil
}

// cleanupLoop periodically evicts expired windows.
func (m *MemoryRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(m.window)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		for k, c := range m.buckets {
			if now.After(c.resetAt) {
				delete(m.buckets, k)
			}
		}
		m.mu.Unlock()
	}
}
