package config

import (
	"testing"
	"time"
)

// TestExpandEnvRegex_Reusable 验证 expandEnvRegex 是包级单例
// （PERF-008）且 1000 次匹配性能可接受。
func TestExpandEnvRegex_Reusable(t *testing.T) {
	if expandEnvRegex == nil {
		t.Fatal("expandEnvRegex 未初始化")
	}

	// 单次匹配能正确捕获 \${VAR:default} 形式
	if !expandEnvRegex.MatchString("${VAR:default}") {
		t.Fatal("expandEnvRegex 未匹配 ${VAR:default}")
	}

	// 性能：包级 regex 1000 次匹配应在 50ms 内
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_ = expandEnvRegex.MatchString("${MY_VAR:fallback}")
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("expandEnvRegex 1000 次匹配耗时 %v（包级应 < 50ms）", elapsed)
	}
}

// TestExpandEnvWithDefault 端到端：包级 regex 用于字符串替换仍正确。
func TestExpandEnvWithDefault(t *testing.T) {
	t.Setenv("TEST_VAR_1", "hello")
	got := expandEnvWithDefault("prefix-${TEST_VAR_1:default}-suffix")
	want := "prefix-hello-suffix"
	if got != want {
		t.Fatalf("expandEnvWithDefault = %q, want %q", got, want)
	}

	// 默认值路径
	got = expandEnvWithDefault("${TEST_VAR_1_NOT_SET:fallback}")
	want = "fallback"
	if got != want {
		t.Fatalf("expandEnvWithDefault fallback = %q, want %q", got, want)
	}
}
