package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
)

func setupPPTServiceTest(t *testing.T) (*PPTFileService, *gorm.DB, string) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.VideoFile{}, &models.PPTFile{})
	require.NoError(t, err)

	logger := zap.NewNop()
	tempDir := t.TempDir()
	cfg := &config.Config{}

	service := NewPPTFileService(db, logger, cfg)
	return service, db, tempDir
}

func TestPPTFileService_RenamePPTFile_Success(t *testing.T) {
	service, db, tempDir := setupPPTServiceTest(t)
	userID := uint(1)
	
	videoFile := &models.VideoFile{
		FileName:  "source.mp4",
		FilePath:  createTestVideoFile(t, tempDir, "source.mp4", "video"),
		CreatedBy: userID,
		Status:    models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	pptPath := filepath.Join(tempDir, "test.pptx")
	require.NoError(t, os.WriteFile(pptPath, []byte("pptx"), 0644))

	pptFile := &models.PPTFile{
		FileName:          "test.pptx",
		FilePath:          pptPath,
		FileSize:          1024,
		Format:            "pptx",
		SourceType:        models.PPTSourceTypeTranscription,
		SourceVideoFileID: &videoFile.ID,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.RenamePPTFile(pptFile.ID, "new_name", userID, false)
	assert.NoError(t, err)

	var updated models.PPTFile
	db.Preload("SourceVideoFile").First(&updated, pptFile.ID)
	assert.Equal(t, "new_name.pptx", updated.FileName)
}

func TestPPTFileService_RenamePPTFile_PreservesExtension(t *testing.T) {
	service, db, tempDir := setupPPTServiceTest(t)
	userID := uint(1)

	videoFile := &models.VideoFile{
		FileName:  "source.mp4",
		FilePath:  createTestVideoFile(t, tempDir, "source.mp4", "video"),
		CreatedBy: userID,
		Status:    models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	pptPath := filepath.Join(tempDir, "test.pptx")
	require.NoError(t, os.WriteFile(pptPath, []byte("pptx"), 0644))

	pptFile := &models.PPTFile{
		FileName:          "test.pptx",
		FilePath:          pptPath,
		FileSize:          1024,
		Format:            "pptx",
		SourceType:        models.PPTSourceTypeTranscription,
		SourceVideoFileID: &videoFile.ID,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.RenamePPTFile(pptFile.ID, "new.ppt", userID, false)
	assert.NoError(t, err)

	var updated models.PPTFile
	db.First(&updated, pptFile.ID)
	assert.Equal(t, "new.pptx", updated.FileName)
}

func TestPPTFileService_RenamePPTFile_OwnershipValidation(t *testing.T) {
	service, db, tempDir := setupPPTServiceTest(t)
	userID := uint(1)
	otherUserID := uint(2)

	videoFile := &models.VideoFile{
		FileName:  "source.mp4",
		FilePath:  createTestVideoFile(t, tempDir, "source.mp4", "video"),
		CreatedBy: userID,
		Status:    models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	pptPath := filepath.Join(tempDir, "test.pptx")
	require.NoError(t, os.WriteFile(pptPath, []byte("pptx"), 0644))

	pptFile := &models.PPTFile{
		FileName:          "test.pptx",
		FilePath:          pptPath,
		FileSize:          1024,
		Format:            "pptx",
		SourceType:        models.PPTSourceTypeTranscription,
		SourceVideoFileID: &videoFile.ID,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.RenamePPTFile(pptFile.ID, "new", otherUserID, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无权重命名")
}

func TestPPTFileService_RenamePPTFile_RollbackOnFilesystemError(t *testing.T) {
	service, db, tempDir := setupPPTServiceTest(t)
	userID := uint(1)

	videoFile := &models.VideoFile{
		FileName:  "source.mp4",
		FilePath:  createTestVideoFile(t, tempDir, "source.mp4", "video"),
		CreatedBy: userID,
		Status:    models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	pptPath := filepath.Join(tempDir, "test.pptx")
	require.NoError(t, os.WriteFile(pptPath, []byte("pptx"), 0644))

	pptFile := &models.PPTFile{
		FileName:          "test.pptx",
		FilePath:          pptPath,
		FileSize:          1024,
		Format:            "pptx",
		SourceType:        models.PPTSourceTypeTranscription,
		SourceVideoFileID: &videoFile.ID,
	}
	require.NoError(t, db.Create(pptFile).Error)

	require.NoError(t, os.Remove(pptPath))

	err := service.RenamePPTFile(pptFile.ID, "new", userID, false)
	assert.Error(t, err)

	var updated models.PPTFile
	db.First(&updated, pptFile.ID)
	assert.Equal(t, pptFile.FileName, updated.FileName)
}

func TestPPTFileService_RenamePPTFile_DuplicateDetection(t *testing.T) {
	service, db, tempDir := setupPPTServiceTest(t)
	userID := uint(1)

	videoFile := &models.VideoFile{
		FileName:  "source.mp4",
		FilePath:  createTestVideoFile(t, tempDir, "source.mp4", "video"),
		CreatedBy: userID,
		Status:    models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	pptPath1 := filepath.Join(tempDir, "existing.pptx")
	require.NoError(t, os.WriteFile(pptPath1, []byte("pptx1"), 0644))

	pptFile1 := &models.PPTFile{
		FileName:          "existing.pptx",
		FilePath:          pptPath1,
		FileSize:          1024,
		Format:            "pptx",
		SourceType:        models.PPTSourceTypeTranscription,
		SourceVideoFileID: &videoFile.ID,
	}
	require.NoError(t, db.Create(pptFile1).Error)

	pptPath2 := filepath.Join(tempDir, "test.pptx")
	require.NoError(t, os.WriteFile(pptPath2, []byte("pptx2"), 0644))

	pptFile2 := &models.PPTFile{
		FileName:          "test.pptx",
		FilePath:          pptPath2,
		FileSize:          1024,
		Format:            "pptx",
		SourceType:        models.PPTSourceTypeTranscription,
		SourceVideoFileID: &videoFile.ID,
	}
	require.NoError(t, db.Create(pptFile2).Error)

	err := service.RenamePPTFile(pptFile2.ID, "existing", userID, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已存在")
}

func TestPPTFileService_RenamePPTFile_UpdatesSlideCachePath(t *testing.T) {
	service, db, tempDir := setupPPTServiceTest(t)
	userID := uint(1)

	videoFile := &models.VideoFile{
		FileName:  "source.mp4",
		FilePath:  createTestVideoFile(t, tempDir, "source.mp4", "video"),
		CreatedBy: userID,
		Status:    models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	cacheDir := filepath.Join(tempDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	pptPath := filepath.Join(tempDir, "test.pptx")
	require.NoError(t, os.WriteFile(pptPath, []byte("pptx"), 0644))

	pptFile := &models.PPTFile{
		FileName:          "test.pptx",
		FilePath:          pptPath,
		FileSize:          1024,
		Format:            "pptx",
		SourceType:        models.PPTSourceTypeTranscription,
		SourceVideoFileID: &videoFile.ID,
		SlideCachePath:    cacheDir,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.RenamePPTFile(pptFile.ID, "new_ppt", userID, false)
	assert.NoError(t, err)

	var updated models.PPTFile
	db.First(&updated, pptFile.ID)
	assert.NotEqual(t, cacheDir, updated.SlideCachePath)
	assert.Contains(t, updated.SlideCachePath, "new_ppt")
}
