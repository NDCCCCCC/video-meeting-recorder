package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PPTFileService handles PPT file CRUD operations
type PPTFileService struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
}

// NewPPTFileService creates a new PPTFileService instance
func NewPPTFileService(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *PPTFileService {
	return &PPTFileService{
		db:     db,
		logger: logger,
		config: cfg,
	}
}

// GetPPTFileByID retrieves a PPT file by ID
func (s *PPTFileService) GetPPTFileByID(id uint) (*models.PPTFile, error) {
	var pptFile models.PPTFile
	err := s.db.Where("id = ?", id).First(&pptFile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("PPT文件不存在")
		}
		return nil, err
	}
	return &pptFile, nil
}

// GetPptsByVideoFile retrieves all PPT files for a video, ordered by newest first (per D-12)
func (s *PPTFileService) GetPptsByVideoFile(videoFileID uint) ([]models.PPTFile, error) {
	var pptFiles []models.PPTFile
	err := s.db.Where("source_video_file_id = ?", videoFileID).
		Order("created_at DESC").
		Find(&pptFiles).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query PPT files: %w", err)
	}
	return pptFiles, nil
}

// DeletePPTFile deletes a PPT file record and physical file
func (s *PPTFileService) DeletePPTFile(id uint) error {
	// Load PPT file first
	pptFile, err := s.GetPPTFileByID(id)
	if err != nil {
		return err
	}

	// Delete physical file
	if pptFile.FilePath != "" {
		if err := os.Remove(pptFile.FilePath); err != nil {
			// Log warning but continue with DB deletion
			s.logger.Warn("Failed to delete PPT physical file",
				zap.Uint("ppt_file_id", id),
				zap.String("path", pptFile.FilePath),
				zap.Error(err))
		}
	}

	// Delete slide cache if exists
	if pptFile.SlideCachePath != "" {
		if err := os.RemoveAll(pptFile.SlideCachePath); err != nil {
			s.logger.Warn("Failed to delete slide cache",
				zap.Uint("ppt_file_id", id),
				zap.String("cache_path", pptFile.SlideCachePath),
				zap.Error(err))
		}
	}

	// Delete database record
	if err := s.db.Delete(&models.PPTFile{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete PPT file record: %w", err)
	}

	s.logger.Info("PPT file deleted",
		zap.Uint("ppt_file_id", id),
		zap.String("file_name", pptFile.FileName))

	return nil
}

// CreatePPTFile creates a new PPT file record
func (s *PPTFileService) CreatePPTFile(pptFile *models.PPTFile) error {
	if err := s.db.Create(pptFile).Error; err != nil {
		return fmt.Errorf("failed to create PPT file record: %w", err)
	}
	return nil
}

// UpdatePPTFile updates a PPT file record
func (s *PPTFileService) UpdatePPTFile(id uint, updates map[string]interface{}) error {
	if err := s.db.Model(&models.PPTFile{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update PPT file record: %w", err)
	}
	return nil
}

// RenamePPTFile renames a PPT file with atomic database and filesystem update
// Parameters:
//   - id: PPT file ID
//   - newName: new filename without extension (extension will be preserved)
//   - userID: user ID requesting the rename (for ownership validation)
func (s *PPTFileService) RenamePPTFile(id uint, newName string, userID uint) error {
	// Validation: load PPT file with SourceVideoFile preloaded
	var pptFile models.PPTFile
	if err := s.db.Preload("SourceVideoFile").First(&pptFile, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("PPT文件不存在")
		}
		return fmt.Errorf("查询PPT文件失败: %w", err)
	}

	// Validation: check ownership via SourceVideoFile
	if pptFile.SourceVideoFile == nil {
		return fmt.Errorf("PPT文件没有关联视频文件")
	}
	if pptFile.SourceVideoFile.CreatedBy != userID {
		return fmt.Errorf("无权重命名此文件")
	}

	// Validation: sanitize new name
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("新文件名不能为空")
	}
	if len(newName) > 200 {
		return fmt.Errorf("新文件名过长（最大200字符）")
	}
	// Reject path separators to prevent path traversal attacks
	if strings.ContainsAny(newName, "/\\") {
		return fmt.Errorf("文件名不能包含路径分隔符")
	}

	// Preserve file extension: strip any extension from newName, use original file extension
	ext := filepath.Ext(pptFile.FilePath)
	if ext == "" {
		ext = ".pptx" // Default extension if none exists
	}
	// Strip extension from newName if user provided one
	newName = strings.TrimSuffix(newName, filepath.Ext(newName))
	if newName == "" {
		return fmt.Errorf("新文件名不能为空")
	}
	newFileName := newName + ext

	// Validation: check for duplicate filename in same directory
	dir := filepath.Dir(pptFile.FilePath)
	newFilePath := filepath.Join(dir, newFileName)

	// Check if another file with the same path already exists
	var existingFile models.PPTFile
	err := s.db.Where("file_path = ? AND id != ?", newFilePath, id).First(&existingFile).Error
	if err == nil {
		return fmt.Errorf("目标文件名已存在")
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查文件名重复失败: %w", err)
	}

	// Atomic rename: database transaction + filesystem rename
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Rename physical file
		if err := os.Rename(pptFile.FilePath, newFilePath); err != nil {
			s.logger.Warn("重命名PPT物理文件失败",
				zap.Uint("ppt_file_id", id),
				zap.String("old_path", pptFile.FilePath),
				zap.String("new_path", newFilePath),
				zap.Error(err),
			)
			return fmt.Errorf("重命名物理文件失败: %w", err)
		}

		// Step 2: Update slide cache path if exists
		var newSlideCachePath string
		if pptFile.SlideCachePath != "" {
			// Update cache directory name to match new filename
			cacheDir := filepath.Dir(pptFile.SlideCachePath)
			newSlideCachePath = filepath.Join(cacheDir, newFileName+"_cache")
			
			// Attempt to rename cache directory (non-critical)
			if err := os.Rename(pptFile.SlideCachePath, newSlideCachePath); err != nil {
				s.logger.Warn("重命名slide缓存目录失败",
					zap.Uint("ppt_file_id", id),
					zap.String("old_cache", pptFile.SlideCachePath),
					zap.String("new_cache", newSlideCachePath),
					zap.Error(err),
				)
				// Don't fail on cache rename error - continue with DB update
				newSlideCachePath = pptFile.SlideCachePath
			}
		}

		// Step 3: Update database record
		if err := tx.Model(&pptFile).Updates(map[string]interface{}{
			"file_name":        newFileName,
			"file_path":        newFilePath,
			"slide_cache_path": newSlideCachePath,
		}).Error; err != nil {
			// Rollback: try to revert physical file rename
			s.logger.Error("更新数据库失败，尝试回滚文件重命名",
				zap.Uint("ppt_file_id", id),
				zap.Error(err),
			)
			if rollbackErr := os.Rename(newFilePath, pptFile.FilePath); rollbackErr != nil {
				s.logger.Error("回滚文件重命名失败",
					zap.String("from", newFilePath),
					zap.String("to", pptFile.FilePath),
					zap.Error(rollbackErr),
				)
			}
			return fmt.Errorf("更新数据库记录失败: %w", err)
		}

		s.logger.Info("PPT文件重命名成功",
			zap.Uint("ppt_file_id", id),
			zap.String("old_name", pptFile.FileName),
			zap.String("new_name", newFileName),
		)

		return nil
	})

	return err
}
