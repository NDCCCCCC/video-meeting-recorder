package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// OSSService handles Aliyun OSS file operations
type OSSService struct {
	client *oss.Client
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
		return nil, fmt.Errorf("OSS配置不完整: endpoint, bucket_name, access_key_id, access_key_secret 不能为空: %w", apperrors.ErrInvalidInput)
	}

	// Use OSS SDK v2's own credentials package (NOT the standalone credentials-go)
	provider := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret)

	ossCfg := oss.NewConfig().
		WithEndpoint(cfg.Endpoint).
		WithCredentialsProvider(provider)

	client := oss.NewClient(ossCfg)

	logger.Info("OSS服务初始化成功",
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", cfg.BucketName))

	return &OSSService{
		client: client,
		logger: logger,
		config: cfg,
	}, nil
}

// UploadFile uploads a local file to OSS and returns a presigned URL
func (s *OSSService) UploadFile(ctx context.Context, localPath, objectKey string) (string, error) {
	if !s.config.Enabled {
		return "", fmt.Errorf("OSS服务未启用: %w", apperrors.ErrServiceUnavailable)
	}
	if localPath == "" {
		return "", fmt.Errorf("本地文件路径不能为空: %w", apperrors.ErrInvalidInput)
	}
	if objectKey == "" {
		return "", fmt.Errorf("OSS对象键不能为空: %w", apperrors.ErrInvalidInput)
	}

	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("打开本地文件失败: %w: %w", apperrors.ErrInternal, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %w: %w", apperrors.ErrInternal, err)
	}

	// Upload to OSS using PutObject
	_, err = s.client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket:        oss.Ptr(s.config.BucketName),
		Key:           oss.Ptr(objectKey),
		Body:          file,
		ContentLength: oss.Ptr(fileInfo.Size()),
	})
	if err != nil {
		return "", fmt.Errorf("OSS上传失败: %w: %w", apperrors.ErrInternal, err)
	}

	s.logger.Info("文件已上传到OSS",
		zap.String("local_path", localPath),
		zap.String("object_key", objectKey),
		zap.Int64("file_size", fileInfo.Size()))

	// Generate presigned URL for downloading
	ttl := time.Duration(s.config.PresignedURLTTL) * time.Second
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	presignResult, err := s.client.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(s.config.BucketName),
		Key:    oss.Ptr(objectKey),
	}, oss.PresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("生成预签名URL失败: %w: %w", apperrors.ErrInternal, err)
	}

	return presignResult.URL, nil
}

// SetLifecycleRule sets an expiration rule for uploaded files (per OSS-02)
func (s *OSSService) SetLifecycleRule(ctx context.Context, ruleID, prefix string, days int) error {
	if !s.config.Enabled {
		return fmt.Errorf("OSS服务未启用")
	}

	daysInt32 := int32(days)
	_, err := s.client.PutBucketLifecycle(ctx, &oss.PutBucketLifecycleRequest{
		Bucket: oss.Ptr(s.config.BucketName),
		LifecycleConfiguration: &oss.LifecycleConfiguration{
			Rules: []oss.LifecycleRule{
				{
					ID:     oss.Ptr(ruleID),
					Prefix: oss.Ptr(prefix),
					Status: oss.Ptr("Enabled"),
					Expiration: &oss.LifecycleRuleExpiration{
						Days: &daysInt32,
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("设置生命周期规则失败: %w: %w", apperrors.ErrInternal, err)
	}

	s.logger.Info("OSS生命周期规则已设置",
		zap.String("rule_id", ruleID),
		zap.String("prefix", prefix),
		zap.Int("days", days))

	return nil
}

// DeleteFile deletes a file from OSS
func (s *OSSService) DeleteFile(ctx context.Context, objectKey string) error {
	if !s.config.Enabled {
		return fmt.Errorf("OSS服务未启用")
	}

	_, err := s.client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(s.config.BucketName),
		Key:    oss.Ptr(objectKey),
	})
	if err != nil {
		return fmt.Errorf("删除OSS文件失败: %w: %w", apperrors.ErrInternal, err)
	}

	s.logger.Info("文件已从OSS删除", zap.String("object_key", objectKey))
	return nil
}

// IsEnabled returns whether OSS is configured and enabled
func (s *OSSService) IsEnabled() bool {
	return s.config.Enabled
}

// IsStub returns true when the OSS service is running in stub mode (no actual SDK integration)
func (s *OSSService) IsStub() bool {
	return s.client == nil // Stub mode when client is nil (disabled or no real initialization)
}
