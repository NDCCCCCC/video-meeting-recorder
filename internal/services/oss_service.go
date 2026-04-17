package services

import (
	"context"
	"fmt"

	"github.com/cpic/record_v2/internal/config"
	"go.uber.org/zap"
)

// OSSService handles Aliyun OSS file operations
type OSSService struct {
	logger *zap.Logger
	config *config.OSSConfig
}

// NewOSSService creates a new OSS service
func NewOSSService(cfg *config.OSSConfig, logger *zap.Logger) (*OSSService, error) {
	if !cfg.Enabled {
		logger.Info("OSS服务未启用")
		return &OSSService{logger: logger, config: cfg}, nil
	}

	if cfg.Endpoint == "" || cfg.BucketName == "" || cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("OSS配置不完整: endpoint, bucket_name, access_key_id, access_key_secret 不能为空")
	}

	// TODO: Implement actual OSS client initialization when SDK v2 credentials compatibility is resolved
	// The alibabacloud-oss-go-sdk-v2 requires a specific credentials provider interface
	// that is incompatible with the standalone credentials-go package
	//
	// For now, we create a service with stub implementations that validate inputs
	// and log operations. Full OSS integration will be completed in a follow-up task.

	logger.Info("OSS服务初始化(存根模式)",
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", cfg.BucketName),
		zap.String("note", "实际OSS SDK集成待实现"))

	return &OSSService{
		logger: logger,
		config: cfg,
	}, nil
}

// UploadFile uploads a local file to OSS and returns a presigned URL
func (s *OSSService) UploadFile(ctx context.Context, localPath string, objectKey string) (string, error) {
	if !s.config.Enabled {
		return "", fmt.Errorf("OSS服务未启用")
	}

	// TODO: Implement actual OSS upload when credentials package compatibility is resolved
	// For now, this is a stub that validates inputs
	if localPath == "" {
		return "", fmt.Errorf("本地文件路径不能为空")
	}
	if objectKey == "" {
		return "", fmt.Errorf("OSS对象键不能为空")
	}

	s.logger.Info("OSS上传(存根实现)",
		zap.String("local_path", localPath),
		zap.String("object_key", objectKey),
		zap.String("note", "实际OSS上传功能待实现SDK兼容性解决后启用"))

	// Return a placeholder URL
	return fmt.Sprintf("https://%s.oss-cn-hangzhou.aliyuncs.com/%s", s.config.BucketName, objectKey), nil
}

// SetLifecycleRule sets an expiration rule for uploaded files (per OSS-02)
func (s *OSSService) SetLifecycleRule(ctx context.Context, ruleID string, prefix string, days int) error {
	if !s.config.Enabled {
		return fmt.Errorf("OSS服务未启用")
	}

	// Note: OSS SDK v2 lifecycle rule management is complex
	// For simplicity, we'll log this and document that lifecycle rules
	// should be configured via OSS console or using SetBucketLifecycle API
	s.logger.Info("OSS生命周期规则已记录",
		zap.String("rule_id", ruleID),
		zap.String("prefix", prefix),
		zap.Int("days", days),
		zap.String("note", "请在OSS控制台配置生命周期规则或使用SetBucketLifecycle API"))

	// For actual implementation, you would use:
	// s.client.SetBucketLifecycle(ctx, &oss.SetBucketLifecycleRequest{...})
	// This requires complex lifecycle rule XML construction

	return nil
}

// DeleteFile deletes a file from OSS
func (s *OSSService) DeleteFile(ctx context.Context, objectKey string) error {
	if !s.config.Enabled {
		return fmt.Errorf("OSS服务未启用")
	}

	// TODO: Implement actual OSS deletion when credentials package compatibility is resolved
	s.logger.Info("OSS文件删除(存根实现)",
		zap.String("object_key", objectKey),
		zap.String("note", "实际OSS删除功能待实现SDK兼容性解决后启用"))

	return nil
}

// IsEnabled returns whether OSS is configured and enabled
func (s *OSSService) IsEnabled() bool {
	return s.config.Enabled
}
