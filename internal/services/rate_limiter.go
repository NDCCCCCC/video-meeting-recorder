package services

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiter 速率限制器
// 使用滑动窗口算法实现内存级别的速率限制
type RateLimiter struct {
	logger    *zap.Logger
	mu        sync.RWMutex
	windows   map[uint]*rateLimitWindow // API Key ID -> 窗口
	cleanTick *time.Ticker
	done      chan struct{}
}

// rateLimitWindow 速率限制窗口
type rateLimitWindow struct {
	requests []time.Time // 请求时间戳队列
	mu       sync.Mutex
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	RequestsPerMinute int // 每分钟请求数限制
	RequestsPerHour   int // 每小时请求数限制
	RequestsPerDay    int // 每天请求数限制
}

// DefaultRateLimitConfig 默认速率限制配置
var DefaultRateLimitConfig = RateLimitConfig{
	RequestsPerMinute: 60,
	RequestsPerHour:   1000,
	RequestsPerDay:    10000,
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		logger:  logger,
		windows: make(map[uint]*rateLimitWindow),
		// PERF-014: 无缓冲 channel 仅用作 stop 信号 (close-only)
		// — 关闭后 cleanupExpiredWindows 在 select 中收到零值并退出。
		done: make(chan struct{}),
	}

	rl.cleanTick = time.NewTicker(5 * time.Minute)
	go rl.cleanupExpiredWindows()

	return rl
}

// CheckRateLimit 检查速率限制
// 返回: (是否允许, 剩余请求数, 重置时间)
// 注意：此方法在检查通过后会立即记录本次请求，保证原子性
func (rl *RateLimiter) CheckRateLimit(apiKeyID uint, config RateLimitConfig) (bool, int64, time.Time) {
	now := time.Now()

	rl.mu.Lock()
	window, exists := rl.windows[apiKeyID]
	if !exists {
		window = &rateLimitWindow{
			requests: make([]time.Time, 0),
		}
		rl.windows[apiKeyID] = window
	}
	rl.mu.Unlock()

	window.mu.Lock()
	defer window.mu.Unlock()

	// 清理过期的请求时间戳（超过1天的）
	window.cleanupOldRequests(now)

	// 检查分钟级限制
	if config.RequestsPerMinute > 0 {
		minuteAgo := now.Add(-time.Minute)
		recentCount := countRequestsAfter(window.requests, minuteAgo)
		if recentCount >= config.RequestsPerMinute {
			// 找到分钟窗口内最早的请求，计算其过期时间
			earliest := window.findEarliestAfter(minuteAgo)
			resetTime := earliest.Add(time.Minute)
			return false, 0, resetTime
		}
	}

	// 检查小时级限制
	if config.RequestsPerHour > 0 {
		hourAgo := now.Add(-time.Hour)
		recentCount := countRequestsAfter(window.requests, hourAgo)
		if recentCount >= config.RequestsPerHour {
			earliest := window.findEarliestAfter(hourAgo)
			resetTime := earliest.Add(time.Hour)
			return false, 0, resetTime
		}
	}

	// 检查天级限制
	if config.RequestsPerDay > 0 {
		dayAgo := now.Add(-24 * time.Hour)
		recentCount := countRequestsAfter(window.requests, dayAgo)
		if recentCount >= config.RequestsPerDay {
			earliest := window.findEarliestAfter(dayAgo)
			resetTime := earliest.Add(24 * time.Hour)
			return false, 0, resetTime
		}
	}

	// 检查通过，立即记录本次请求（在锁内完成，保证原子性）
	window.requests = append(window.requests, now)

	// 计算剩余请求数（基于最严格的限制）
	remaining := calculateRemaining(window.requests, config, now)
	resetTime := now.Add(time.Minute)

	return true, remaining, resetTime
}

// countRequestsAfter 计算指定时间之后的请求数
func countRequestsAfter(requests []time.Time, after time.Time) int {
	count := 0
	for _, t := range requests {
		if t.After(after) {
			count++
		}
	}
	return count
}

// calculateRemaining 计算剩余请求数
func calculateRemaining(requests []time.Time, config RateLimitConfig, now time.Time) int64 {
	var minRemaining int64 = -1

	if config.RequestsPerMinute > 0 {
		minuteAgo := now.Add(-time.Minute)
		used := countRequestsAfter(requests, minuteAgo)
		remaining := int64(config.RequestsPerMinute - used)
		if minRemaining < 0 || remaining < minRemaining {
			minRemaining = remaining
		}
	}

	if config.RequestsPerHour > 0 {
		hourAgo := now.Add(-time.Hour)
		used := countRequestsAfter(requests, hourAgo)
		remaining := int64(config.RequestsPerHour - used)
		if minRemaining < 0 || remaining < minRemaining {
			minRemaining = remaining
		}
	}

	if config.RequestsPerDay > 0 {
		dayAgo := now.Add(-24 * time.Hour)
		used := countRequestsAfter(requests, dayAgo)
		remaining := int64(config.RequestsPerDay - used)
		if minRemaining < 0 || remaining < minRemaining {
			minRemaining = remaining
		}
	}

	if minRemaining < 0 {
		return 0
	}
	return minRemaining
}

// findEarliestAfter 找到指定时间之后的最早请求时间
func (w *rateLimitWindow) findEarliestAfter(after time.Time) time.Time {
	for _, t := range w.requests {
		if t.After(after) {
			return t
		}
	}
	return time.Now()
}

// cleanupOldRequests 清理过期的请求时间戳
func (w *rateLimitWindow) cleanupOldRequests(now time.Time) {
	dayAgo := now.Add(-24 * time.Hour)
	cutoff := 0
	for i, t := range w.requests {
		if t.After(dayAgo) {
			cutoff = i
			break
		}
	}
	if cutoff > 0 {
		w.requests = w.requests[cutoff:]
	}
}

// cleanupExpiredWindows 定期清理长时间未使用的窗口
func (rl *RateLimiter) cleanupExpiredWindows() {
	for {
		select {
		case <-rl.cleanTick.C:
			rl.mu.Lock()
			now := time.Now()
			for key, window := range rl.windows {
				window.mu.Lock()
				if len(window.requests) == 0 || now.Sub(window.requests[len(window.requests)-1]) > time.Hour {
					delete(rl.windows, key)
				}
				window.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

// ResetAPIKey 重置指定 API Key 的速率限制
func (rl *RateLimiter) ResetAPIKey(apiKeyID uint) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if window, exists := rl.windows[apiKeyID]; exists {
		window.mu.Lock()
		window.requests = make([]time.Time, 0)
		window.mu.Unlock()
	}
}

// GetAPIKeyStats 获取指定 API Key 的当前使用统计
func (rl *RateLimiter) GetAPIKeyStats(apiKeyID uint) (minuteCount, hourCount, dayCount int) {
	rl.mu.RLock()
	window, exists := rl.windows[apiKeyID]
	rl.mu.RUnlock()

	if !exists {
		return 0, 0, 0
	}

	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()
	minuteCount = countRequestsAfter(window.requests, now.Add(-time.Minute))
	hourCount = countRequestsAfter(window.requests, now.Add(-time.Hour))
	dayCount = countRequestsAfter(window.requests, now.Add(-24*time.Hour))

	return minuteCount, hourCount, dayCount
}

// Shutdown 关闭速率限制器
func (rl *RateLimiter) Shutdown() {
	rl.cleanTick.Stop()
	close(rl.done)
	rl.logger.Info("速率限制器已关闭")
}
