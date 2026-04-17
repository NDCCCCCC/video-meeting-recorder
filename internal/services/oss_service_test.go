package services

import (
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"go.uber.org/zap"
)

func TestOSSServiceDisabledByDefault(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, err := NewOSSService(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewOSSService with disabled config should not error, got: %v", err)
	}
	if svc.IsEnabled() {
		t.Error("OSSService should be disabled when config.Enabled=false")
	}
}

func TestOSSServiceRejectsIncompleteConfig(t *testing.T) {
	cfg := &config.OSSConfig{
		Enabled:    true,
		Endpoint:   "",
		BucketName: "",
	}
	_, err := NewOSSService(cfg, zap.NewNop())
	if err == nil {
		t.Error("NewOSSService should error when Enabled=true but credentials are empty")
	}
}

func TestOSSServiceDisabledUploadFails(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, _ := NewOSSService(cfg, zap.NewNop())
	_, err := svc.UploadFile(nil, "/tmp/test.mp4", "test-key")
	if err == nil {
		t.Error("UploadFile on disabled service should return error")
	}
}

func TestOSSServiceDisabledDeleteFails(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, _ := NewOSSService(cfg, zap.NewNop())
	err := svc.DeleteFile(nil, "test-key")
	if err == nil {
		t.Error("DeleteFile on disabled service should return error")
	}
}

func TestOSSServiceDisabledLifecycleFails(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, _ := NewOSSService(cfg, zap.NewNop())
	err := svc.SetLifecycleRule(nil, "rule-1", "prefix/", 1)
	if err == nil {
		t.Error("SetLifecycleRule on disabled service should return error")
	}
}
