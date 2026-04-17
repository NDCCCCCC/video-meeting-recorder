package services

import (
	"testing"

	"github.com/cpic/record_v2/internal/config"
	"go.uber.org/zap"
)

func TestTingwuClientDisabledByDefault(t *testing.T) {
	cfg := &config.TingwuConfig{Enabled: false}
	client := NewTingwuClient(cfg, zap.NewNop())
	if client.IsEnabled() {
		t.Error("TingwuClient should be disabled when config.Enabled=false")
	}
}

func TestTingwuClientDisabledWithEmptyAppKey(t *testing.T) {
	cfg := &config.TingwuConfig{Enabled: true, AppKey: ""}
	client := NewTingwuClient(cfg, zap.NewNop())
	if client.IsEnabled() {
		t.Error("TingwuClient should be disabled when AppKey is empty")
	}
}

func TestTingwuClientDisabledSubmitFails(t *testing.T) {
	cfg := &config.TingwuConfig{Enabled: false}
	client := NewTingwuClient(cfg, zap.NewNop())
	_, err := client.SubmitTask(nil, "https://example.com/test.mp4")
	if err == nil {
		t.Error("SubmitTask on disabled client should return error")
	}
}

func TestTingwuClientDisabledGetStatusFails(t *testing.T) {
	cfg := &config.TingwuConfig{Enabled: false}
	client := NewTingwuClient(cfg, zap.NewNop())
	_, err := client.GetStatus(nil, "task-123")
	if err == nil {
		t.Error("GetStatus on disabled client should return error")
	}
}

func TestTingwuClientDisabledGetResultFails(t *testing.T) {
	cfg := &config.TingwuConfig{Enabled: false}
	client := NewTingwuClient(cfg, zap.NewNop())
	_, err := client.GetResult(nil, "task-123")
	if err == nil {
		t.Error("GetResult on disabled client should return error")
	}
}

func TestTingwuClientCalculateSignature(t *testing.T) {
	cfg := &config.TingwuConfig{
		Enabled:   true,
		AppKey:    "test-key",
		AppSecret: "test-secret",
		BaseURL:   "https://example.com",
	}
	client := NewTingwuClient(cfg, zap.NewNop())

	// Verify signature is deterministic
	sig1 := client.calculateSignature("GET", "/test/path", "2026-01-01T00:00:00Z", "")
	sig2 := client.calculateSignature("GET", "/test/path", "2026-01-01T00:00:00Z", "")
	if sig1 != sig2 {
		t.Error("Signature should be deterministic for same inputs")
	}

	// Verify signature changes with different method
	sig3 := client.calculateSignature("POST", "/test/path", "2026-01-01T00:00:00Z", "")
	if sig1 == sig3 {
		t.Error("Signature should differ when method changes")
	}
}
