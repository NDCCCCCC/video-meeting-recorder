package services

import (
	"fmt"
	"os"

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
