package storage

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// 默认文件大小限制 5GB
	defaultMaxFileSize = 5 * 1024 * 1024 * 1024
	// 默认配额 10GB
	defaultQuota = 10 * 1024 * 1024 * 1024
)

// FileService 文件服务
type FileService struct {
	db            *gorm.DB
	logger        *zap.Logger
	config        *config.Config
	drivers       map[models.StorageType]StorageDriver
	defaultDriver StorageDriver
}

// UploadRequest 上传请求
type UploadRequest struct {
	File        *multipart.FileHeader
	Folder      string
	IsPublic    bool
	ExpiresIn   time.Duration
	RelatedID   *uint
	RelatedType string
}

// FileUploadResult 文件上传结果
type FileUploadResult struct {
	FileID    uint   `json:"file_id"`
	FileName  string `json:"file_name"`
	FilePath  string `json:"file_path"`
	FileSize  int64  `json:"file_size"`
	MimeType  string `json:"mime_type"`
	AccessURL string `json:"access_url"`
}

// QueryRequest 查询请求
type QueryRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	FileType string `form:"file_type"`
	Keyword  string `form:"keyword"`
}

// QueryResponse 查询响应
type QueryResponse struct {
	Items      []*models.UploadedFile `json:"items"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

// QuotaInfo 配额信息
type QuotaInfo struct {
	TotalQuota     int64   `json:"total_quota"`
	UsedQuota      int64   `json:"used_quota"`
	FileCount      int     `json:"file_count"`
	AvailableQuota int64   `json:"available_quota"`
	UsagePercent   float64 `json:"usage_percent"`
}

// NewFileService 创建文件服务
func NewFileService(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *FileService {
	service := &FileService{
		db:      db,
		logger:  logger,
		config:  cfg,
		drivers: make(map[models.StorageType]StorageDriver),
	}

	// 初始化存储驱动
	service.initDrivers()

	return service
}

// initDrivers 初始化存储驱动
func (s *FileService) initDrivers() {
	// 本地存储
	localDriver := NewLocalStorageDriver(
		s.config.Storage.Local.BasePath,
		s.config.Storage.Local.BaseURL,
		s.logger,
	)
	s.drivers[models.StorageLocal] = localDriver
	s.defaultDriver = localDriver

	s.logger.Info("文件存储服务初始化完成",
		zap.String("default_type", "local"),
		zap.String("base_path", s.config.Storage.Local.BasePath),
	)
}

// Upload 上传文件
func (s *FileService) Upload(ctx context.Context, userID uint, req *UploadRequest) (*FileUploadResult, error) {
	// 1. 验证文件
	if err := s.validateFile(req.File); err != nil {
		return nil, fmt.Errorf("文件验证失败: %w", err)
	}

	// 2. 检查用户配额
	if err := s.checkUserQuota(ctx, userID, req.File.Size); err != nil {
		return nil, err
	}

	// 3. 计算文件MD5
	fileMD5, err := s.calculateMD5(req.File)
	if err != nil {
		s.logger.Warn("计算MD5失败", zap.Error(err))
	}

	// 4. 检查文件是否已存在（秒传）
	if fileMD5 != "" {
		existingFile := s.findExistingFile(ctx, userID, fileMD5)
		if existingFile != nil {
			return &FileUploadResult{
				FileID:    existingFile.ID,
				FileName:  existingFile.FileName,
				FilePath:  existingFile.FilePath,
				FileSize:  existingFile.FileSize,
				MimeType:  existingFile.MimeType,
				AccessURL: existingFile.AccessURL,
			}, nil
		}
	}

	// 5. 生成文件路径 - 使用原始文件名，处理同名冲突
	baseName := filepath.Base(req.File.Filename)
	fileName := s.generateUniqueFileName(ctx, req.Folder, baseName)
	relativePath := filepath.Join(req.Folder, time.Now().Format("2006/01/02"), fileName)

	// 6. 上传文件
	result, err := s.defaultDriver.Upload(ctx, req.File, relativePath)
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %w", err)
	}

	// 7. 生成访问令牌
	accessToken := ""
	accessURL := result.URL
	if !req.IsPublic {
		accessToken = generateAccessToken()
		accessURL = fmt.Sprintf("%s/api/v1/files/download/%s", s.getServerURL(), accessToken)
	}

	// 8. 计算过期时间
	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		expiry := time.Now().Add(req.ExpiresIn)
		expiresAt = &expiry
	}

	// 9. 保存文件记录
	uploadedFile := &models.UploadedFile{
		FileName:     fileName,
		OriginalName: req.File.Filename,
		FilePath:     result.FilePath,
		FileSize:     req.File.Size,
		MimeType:     req.File.Header.Get("Content-Type"),
		FileMD5:      fileMD5,
		StorageType:  string(models.StorageLocal),
		StoragePath:  result.StoragePath,
		IsPublic:     req.IsPublic,
		AccessURL:    accessURL,
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		UploadedBy:   userID,
		UploadedAt:   time.Now(),
		RelatedID:    req.RelatedID,
		RelatedType:  req.RelatedType,
		Status:       string(models.FileStatusActive),
	}

	if err := s.db.Create(uploadedFile).Error; err != nil {
		// 回滚上传
		s.defaultDriver.Delete(ctx, result.FilePath)
		return nil, fmt.Errorf("保存文件记录失败: %w", err)
	}

	// 10. 更新用户配额
	s.updateUserQuota(ctx, userID, req.File.Size)

	s.logger.Info("文件上传成功",
		zap.Uint("user_id", userID),
		zap.Uint("file_id", uploadedFile.ID),
		zap.String("file_name", uploadedFile.OriginalName),
		zap.Int64("file_size", uploadedFile.FileSize),
	)

	return &FileUploadResult{
		FileID:    uploadedFile.ID,
		FileName:  uploadedFile.FileName,
		FilePath:  uploadedFile.FilePath,
		FileSize:  uploadedFile.FileSize,
		MimeType:  uploadedFile.MimeType,
		AccessURL: uploadedFile.AccessURL,
	}, nil
}

// Download 下载文件
func (s *FileService) Download(ctx context.Context, accessToken string, userID uint) (io.ReadCloser, string, error) {
	var file models.UploadedFile
	err := s.db.Where("access_token = ? OR is_public = ?", accessToken, true).
		Where("status = ?", models.FileStatusActive).
		First(&file).Error
	if err != nil {
		return nil, "", fmt.Errorf("文件不存在")
	}

	// 检查过期
	if file.ExpiresAt != nil && time.Now().After(*file.ExpiresAt) {
		return nil, "", fmt.Errorf("文件已过期")
	}

	// 检查权限
	if !file.IsPublic && file.UploadedBy != userID {
		if !s.hasDownloadPermission(ctx, userID, &file) {
			return nil, "", fmt.Errorf("无权限下载")
		}
	}

	// 获取存储驱动
	driver, ok := s.drivers[models.StorageType(file.StorageType)]
	if !ok {
		return nil, "", fmt.Errorf("不支持的存储类型")
	}

	// 下载文件
	reader, err := driver.Download(ctx, file.FilePath)
	if err != nil {
		return nil, "", fmt.Errorf("下载文件失败: %w", err)
	}

	return reader, file.OriginalName, nil
}

// Delete 删除文件
func (s *FileService) Delete(ctx context.Context, fileID uint, userID uint) (*models.UploadedFile, error) {
	var file models.UploadedFile
	err := s.db.First(&file, fileID).Error
	if err != nil {
		return nil, fmt.Errorf("文件不存在")
	}

	// 检查权限
	if file.UploadedBy != userID {
		return nil, fmt.Errorf("无权限删除")
	}

	// Snapshot before soft-delete for audit OldData capture
	oldFile := file

	// 软删除
	if err := s.db.Model(&file).Update("status", models.FileStatusDeleted).Error; err != nil {
		return nil, err
	}

	// 异步删除物理文件（传递上下文并记录错误）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("async file delete goroutine panicked",
					zap.Any("recover", r), zap.Stack("stack"))
			}
		}()
		driver, ok := s.drivers[models.StorageType(file.StorageType)]
		if !ok {
			s.logger.Warn("Storage driver not found for async file deletion",
				zap.String("storage_type", file.StorageType),
				zap.String("file_path", file.FilePath))
			return
		}
		if err := driver.Delete(ctx, file.FilePath); err != nil {
			s.logger.Warn("Async file deletion failed",
				zap.Uint("file_id", fileID),
				zap.String("file_path", file.FilePath),
				zap.Error(err))
		}
	}()

	// 更新用户配额
	s.updateUserQuota(ctx, userID, -file.FileSize)

	s.logger.Info("文件删除成功",
		zap.Uint("user_id", userID),
		zap.Uint("file_id", fileID),
	)

	return &oldFile, nil
}

// ShareFile 生成分享链接
func (s *FileService) ShareFile(ctx context.Context, fileID uint, userID uint, expiresIn time.Duration, password string) (*models.FileShare, *models.FileShare, string, error) {
	var file models.UploadedFile
	err := s.db.First(&file, fileID).Error
	if err != nil {
		return nil, nil, "", fmt.Errorf("文件不存在")
	}

	// 检查权限
	if file.UploadedBy != userID && !file.IsPublic {
		return nil, nil, "", fmt.Errorf("无权限分享")
	}

	// Snapshot most recent existing share (same file + shared_by) before creating a new one.
	// This is nil when no prior share exists; other errors are propagated.
	var oldShare *models.FileShare
	var existing models.FileShare
	if err := s.db.Where("file_id = ? AND shared_by = ?", fileID, userID).
		Order("created_at DESC").First(&existing).Error; err == nil {
		oldShare = &existing
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, "", err
	}

	// 生成分享记录
	share := &models.FileShare{
		FileID:     fileID,
		ShareToken: generateShareToken(),
		SharedBy:   userID,
		Password:   password,
		MaxAccess:  0,
		ExpiresAt:  time.Now().Add(expiresIn),
	}

	if err := s.db.Create(share).Error; err != nil {
		return nil, nil, "", err
	}

	shareURL := fmt.Sprintf("%s/api/v1/files/share/%s", s.getServerURL(), share.ShareToken)
	return oldShare, share, shareURL, nil
}

// GetShareDownload 通过分享链接下载
func (s *FileService) GetShareDownload(ctx context.Context, shareToken, password string) (io.ReadCloser, string, error) {
	var share models.FileShare
	err := s.db.Preload("File").Where("share_token = ?", shareToken).First(&share).Error
	if err != nil {
		return nil, "", fmt.Errorf("分享链接不存在")
	}

	// 检查过期
	if time.Now().After(share.ExpiresAt) {
		return nil, "", fmt.Errorf("分享链接已过期")
	}

	// 检查密码
	if share.Password != "" && share.Password != password {
		return nil, "", fmt.Errorf("密码错误")
	}

	// 检查访问次数
	if share.MaxAccess > 0 && share.AccessCount >= share.MaxAccess {
		return nil, "", fmt.Errorf("分享链接已达到最大访问次数")
	}

	// 更新访问次数
	s.db.Model(&share).UpdateColumn("access_count", share.AccessCount+1)

	// 获取文件
	driver, ok := s.drivers[models.StorageType(share.File.StorageType)]
	if !ok {
		return nil, "", fmt.Errorf("不支持的存储类型")
	}

	reader, err := driver.Download(ctx, share.File.FilePath)
	if err != nil {
		return nil, "", fmt.Errorf("下载文件失败: %w", err)
	}

	return reader, share.File.OriginalName, nil
}

// Query 查询文件列表
func (s *FileService) Query(ctx context.Context, userID uint, req *QueryRequest) (*QueryResponse, error) {
	query := s.db.Model(&models.UploadedFile{}).
		Where("uploaded_by = ?", userID).
		Where("status = ?", models.FileStatusActive)

	// 文件类型筛选
	if req.FileType != "" {
		query = query.Where("mime_type LIKE ?", req.FileType+"%")
	}

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("original_name LIKE ?", "%"+req.Keyword+"%")
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var files []*models.UploadedFile
	offset := (req.Page - 1) * req.PageSize
	err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&files).Error
	if err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &QueryResponse{
		Items:      files,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUserQuota 获取用户存储配额
func (s *FileService) GetUserQuota(ctx context.Context, userID uint) (*QuotaInfo, error) {
	var quota models.UserStorageQuota
	err := s.db.Where("user_id = ?", userID).First(&quota).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建默认配额
			quota = models.UserStorageQuota{
				UserID:     userID,
				TotalQuota: defaultQuota,
				UsedQuota:  0,
				FileCount:  0,
			}
			s.db.Create(&quota)
		} else {
			return nil, err
		}
	}

	available := quota.TotalQuota - quota.UsedQuota
	usagePercent := 0.0
	if quota.TotalQuota > 0 {
		usagePercent = float64(quota.UsedQuota) / float64(quota.TotalQuota) * 100
	}

	return &QuotaInfo{
		TotalQuota:     quota.TotalQuota,
		UsedQuota:      quota.UsedQuota,
		FileCount:      quota.FileCount,
		AvailableQuota: available,
		UsagePercent:   usagePercent,
	}, nil
}

// validateFile 验证文件
func (s *FileService) validateFile(file *multipart.FileHeader) error {
	// 检查文件大小
	maxSize := int64(defaultMaxFileSize)
	if s.config.Storage.MaxFileSize > 0 {
		maxSize = s.config.Storage.MaxFileSize
	}
	if file.Size > maxSize {
		return fmt.Errorf("文件大小超过限制")
	}

	// 检查文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExtensions := s.config.Storage.AllowedExtensions
	if len(allowedExtensions) == 0 {
		// 默认允许的扩展名
		allowedExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".docx",
			".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".zip", ".rar", ".mp4", ".mkv", ".avi"}
	}

	allowed := false
	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("不支持的文件类型: %s", ext)
	}

	return nil
}

// calculateMD5 计算文件MD5
func (s *FileService) calculateMD5(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, src); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// checkUserQuota 检查用户配额
func (s *FileService) checkUserQuota(ctx context.Context, userID uint, fileSize int64) error {
	var quota models.UserStorageQuota
	err := s.db.Where("user_id = ?", userID).First(&quota).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建默认配额
			quota = models.UserStorageQuota{
				UserID:     userID,
				TotalQuota: defaultQuota,
			}
			s.db.Create(&quota)
		} else {
			return err
		}
	}

	if quota.UsedQuota+fileSize > quota.TotalQuota {
		return fmt.Errorf("存储空间不足")
	}

	return nil
}

// updateUserQuota 更新用户配额
func (s *FileService) updateUserQuota(ctx context.Context, userID uint, delta int64) {
	fileCountDelta := 0
	if delta > 0 {
		fileCountDelta = 1
	} else {
		fileCountDelta = -1
	}

	s.db.Model(&models.UserStorageQuota{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"used_quota": gorm.Expr("used_quota + ?", delta),
			"file_count": gorm.Expr("file_count + ?", fileCountDelta),
		})
}

// findExistingFile 查找已存在的文件（用于秒传）
func (s *FileService) findExistingFile(ctx context.Context, userID uint, md5 string) *models.UploadedFile {
	var file models.UploadedFile
	err := s.db.Where("file_md5 = ? AND uploaded_by = ? AND status = ?", md5, userID, models.FileStatusActive).
		First(&file).Error
	if err != nil {
		return nil
	}
	return &file
}

// hasDownloadPermission 检查下载权限
func (s *FileService) hasDownloadPermission(ctx context.Context, userID uint, file *models.UploadedFile) bool {
	// 检查用户是否有下载权限
	// 可以根据关联的业务对象判断权限
	return true
}

// getServerURL 获取服务器URL
func (s *FileService) getServerURL() string {
	if s.config.Storage.Local.BaseURL != "" {
		return s.config.Storage.Local.BaseURL
	}
	return fmt.Sprintf("http://%s:%d", s.config.Server.Host, s.config.Server.Port)
}

// generateUUID 生成UUID（简化版）
func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// generateAccessToken 生成访问令牌
func generateAccessToken() string {
	return fmt.Sprintf("%d%s", time.Now().UnixNano(), generateUUID())
}

// generateShareToken 生成分享令牌
func generateShareToken() string {
	return fmt.Sprintf("share_%d", time.Now().UnixNano())
}

// generateUniqueFileName 生成唯一文件名，保持原始文件名，处理同名冲突
func (s *FileService) generateUniqueFileName(ctx context.Context, folder, baseName string) string {
	// 检查文件名是否已存在
	var count int64
	err := s.db.Model(&models.UploadedFile{}).
		Where("file_name = ? AND status = ?", baseName, models.FileStatusActive).
		Count(&count).Error

	if err != nil || count == 0 {
		// 文件名不存在，直接使用原始文件名
		return baseName
	}

	// 文件名已存在，添加后缀
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// 尝试添加 (1), (2), ... 后缀
	for i := 1; i <= 1000; i++ {
		newName := fmt.Sprintf("%s (%d)%s", nameWithoutExt, i, ext)
		err := s.db.Model(&models.UploadedFile{}).
			Where("file_name = ? AND status = ?", newName, models.FileStatusActive).
			Count(&count).Error
		if err != nil || count == 0 {
			return newName
		}
	}

	// 如果尝试1000次仍然冲突（极不可能），使用时间戳作为后备
	return fmt.Sprintf("%s_%d%s", nameWithoutExt, time.Now().UnixNano(), ext)
}
