package services

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
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

func TestOSSServiceIsStubWhenDisabled(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, _ := NewOSSService(cfg, zap.NewNop())
	if !svc.IsStub() {
		t.Error("OSSService.IsStub() should return true when service is disabled")
	}
}

func TestOSSServiceEnabledWithValidConfig(t *testing.T) {
	cfg := &config.OSSConfig{
		Enabled:         true,
		Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		BucketName:      "test-bucket",
		AccessKeyID:     "test-key-id",
		AccessKeySecret: "test-key-secret",
	}
	svc, err := NewOSSService(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewOSSService with valid config should not error, got: %v", err)
	}
	if svc.IsStub() {
		t.Error("OSSService.IsStub() should return false when service is enabled with valid credentials")
	}
}

func TestOSSServiceEnabledRejectsEmptyCredentials(t *testing.T) {
	tests := []struct {
		name        string
		accessKeyID string
		accessKey   string
	}{
		{"empty access key id", "", "secret"},
		{"empty access key secret", "key-id", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.OSSConfig{
				Enabled:         true,
				Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
				BucketName:      "test-bucket",
				AccessKeyID:     tt.accessKeyID,
				AccessKeySecret: tt.accessKey,
			}
			_, err := NewOSSService(cfg, zap.NewNop())
			if err == nil {
				t.Error("NewOSSService should error when Enabled=true but credentials are empty")
			}
		})
	}
}

func TestOSSServiceUploadFileValidatesInputs(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, _ := NewOSSService(cfg, zap.NewNop())

	// Test with disabled service
	_, err := svc.UploadFile(context.Background(), "/tmp/test.mp4", "test-key")
	if err == nil {
		t.Error("UploadFile on disabled service should return error")
	}

	// Test with enabled service but empty localPath
	cfg.Enabled = true
	cfg.Endpoint = "oss-cn-hangzhou.aliyuncs.com"
	cfg.BucketName = "test-bucket"
	cfg.AccessKeyID = "test-key-id"
	cfg.AccessKeySecret = "test-key-secret"
	svc, _ = NewOSSService(cfg, zap.NewNop())

	_, err = svc.UploadFile(context.Background(), "", "test-key")
	if err == nil {
		t.Error("UploadFile should return error when localPath is empty")
	}

	_, err = svc.UploadFile(context.Background(), "/tmp/test.mp4", "")
	if err == nil {
		t.Error("UploadFile should return error when objectKey is empty")
	}
}

func TestOSSServiceDisabledUploadFails(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, _ := NewOSSService(cfg, zap.NewNop())
	_, err := svc.UploadFile(context.Background(), "/tmp/test.mp4", "test-key")
	if err == nil {
		t.Error("UploadFile on disabled service should return error")
	}
}

func TestOSSServiceDisabledDeleteFails(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, _ := NewOSSService(cfg, zap.NewNop())
	err := svc.DeleteFile(context.Background(), "test-key")
	if err == nil {
		t.Error("DeleteFile on disabled service should return error")
	}
}

func TestOSSServiceDisabledLifecycleFails(t *testing.T) {
	cfg := &config.OSSConfig{Enabled: false}
	svc, _ := NewOSSService(cfg, zap.NewNop())
	err := svc.SetLifecycleRule(context.Background(), "rule-1", "prefix/", 1)
	if err == nil {
		t.Error("SetLifecycleRule on disabled service should return error")
	}
}
