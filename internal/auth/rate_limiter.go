package auth

import (
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/config"
)

// DecryptFailureTracker 记录解密失败次数
type DecryptFailureTracker struct {
	mu       sync.RWMutex
	failures map[string][]time.Time // username -> 失败时间列表
}

var decryptTracker = &DecryptFailureTracker{
	failures: make(map[string][]time.Time),
}

// RecordFailure 记录解密失败
func (t *DecryptFailureTracker) RecordFailure(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.failures[username] = append(t.failures[username], now)
}

// ShouldBlock 检查是否应该阻止该用户的解密尝试
func (t *DecryptFailureTracker) ShouldBlock(username string, maxFailures int, window time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-window)

	// 清理过期的失败记录
	var recentFailures []time.Time
	for _, failTime := range t.failures[username] {
		if failTime.After(windowStart) {
			recentFailures = append(recentFailures, failTime)
		}
	}
	t.failures[username] = recentFailures

	// 检查失败次数是否超过限制
	return len(recentFailures) >= maxFailures
}

// Clear 清除该用户的失败记录（成功登录后调用）
func (t *DecryptFailureTracker) Clear(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.failures, username)
}

// NewRateLimiterFromConfig 从配置创建速率限制器
func NewRateLimiterFromConfig(cfg *config.Config) *RateLimiter {
	return &RateLimiter{
		maxFailures: cfg.Auth.MaxDecryptFailures,
		window:      time.Duration(cfg.Auth.DecryptFailureWindow) * time.Second,
		tracker:     decryptTracker,
	}
}

// RateLimiter 速率限制器
type RateLimiter struct {
	maxFailures int
	window      time.Duration
	tracker     *DecryptFailureTracker
}

// RecordFailure 记录失败
func (r *RateLimiter) RecordFailure(username string) {
	r.tracker.RecordFailure(username)
}

// ShouldBlock 检查是否应该阻止
func (r *RateLimiter) ShouldBlock(username string) bool {
	return r.tracker.ShouldBlock(username, r.maxFailures, r.window)
}

// Clear 清除记录
func (r *RateLimiter) Clear(username string) {
	r.tracker.Clear(username)
}
