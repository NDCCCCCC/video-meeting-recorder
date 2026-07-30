package huawei

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestHuaweiSanitizeResponseBody(t *testing.T) {
	got := string(huaweiSanitizeResponseBody([]byte(`{"username":"admin","password":"secret123","nested":{"certBase64String":"certificate"},"safe":"ok"}`)))
	for _, secret := range []string{"admin", "secret123", "certificate"} {
		if strings.Contains(got, secret) {
			t.Fatalf("sanitized response leaks %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, `"safe":"ok"`) {
		t.Fatalf("safe field missing: %s", got)
	}
}

// TestHuaweiClient_StopExitsKeepAliveGoroutine 验证 PERF-006 修复：
// Stop(ctx) 必须在 ctx 内退出 keep-alive goroutine，goroutine 计数归 0。
func TestHuaweiClient_StopExitsKeepAliveGoroutine(t *testing.T) {
	before := goroutineCount()

	cfg := &Config{
		Server:            "127.0.0.1",
		Port:              0,
		APITimeout:        500 * time.Millisecond,
		SessionTimeout:    time.Second,
		KeepAliveInterval: 50 * time.Millisecond,
		MinTLSVersion:     0x0303,
	}

	c := NewHuaweiClient(cfg, zapNopForTest())
	c.StartKeepAlive(context.Background())

	// 给 goroutine 一点时间真正进入 ticker 阻塞
	time.Sleep(100 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Stop(stopCtx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	// 等待一小段时间让 runtime 调度
	time.Sleep(100 * time.Millisecond)
	after := goroutineCount()

	// 允许 ±2 抖动（runtime 内部 goroutine 噪声）
	if after > before+2 {
		t.Fatalf("keep-alive goroutine leaked: before=%d after=%d", before, after)
	}
}

func goroutineCount() int {
	return runtimeNumGoroutine()
}
