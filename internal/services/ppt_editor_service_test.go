package services

import (
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// requireExtractSlidesDeps 跳过无法在当前环境运行 PPT 抽取的集成测试。
//
// 抽取依赖 scripts/extract_slides.py，而该脚本依赖 LibreOffice（或 soffice）
// 把 PPTX 转成 PDF/图片。仅 Python/uv 不足以让抽取成功，因此判定跳过需看
// LibreOffice/soffice 是否存在。如要强制执行可设置
// RECORD_V2_FORCE_INTEGRATION_TESTS=1。
func requireExtractSlidesDeps(t *testing.T) {
	t.Helper()
	if os.Getenv("RECORD_V2_FORCE_INTEGRATION_TESTS") == "1" {
		return
	}
	for _, bin := range []string{"libreoffice", "soffice"} {
		if _, err := exec.LookPath(bin); err == nil {
			return
		}
	}
	t.Skip("skipping PPT extraction integration test: LibreOffice/soffice not available")
}

func setupPPTEditorServiceTest(t *testing.T) (*PPTEditorService, *gorm.DB, string) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.VideoFile{}, &models.PPTFile{})
	require.NoError(t, err)

	logger := zap.NewNop()
	tempDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: tempDir,
		},
	}

	slideExtractor := NewSlideExtractor(logger, true) // preferUV=true for testing
	slideCache := NewSlideCacheService(db, logger, cfg, slideExtractor)
	similarityDetector := NewSimilarityDetector(logger)
	pptxGenerator := NewPPTXGenerator(logger, true) // preferUV=true for testing
	timestampMapper := NewTimestampMapper(db, logger)

	service := NewPPTEditorService(db, logger, cfg, slideCache, similarityDetector, pptxGenerator, timestampMapper)
	return service, db, tempDir
}

func createTestPPTFile(t *testing.T, tempDir string, filename string) string {
	pptPath := filepath.Join(tempDir, filename)
	require.NoError(t, os.WriteFile(pptPath, []byte("mock pptx content"), 0644))
	return pptPath
}

func createTestSlideImage(t *testing.T, path string, width, height int) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0755))

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	// Write a simple JPEG header (mock)
	_, err = file.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0}) // JPEG SOI
	require.NoError(t, err)
}

func TestPPTEditorService_CreateBackup_Success(t *testing.T) {
	service, db, tempDir := setupPPTEditorServiceTest(t)

	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	pptFile := &models.PPTFile{
		FileName:   "test.pptx",
		FilePath:   pptPath,
		FileSize:   1024,
		PageCount:  10,
		Format:     "pptx",
		SourceType: models.PPTSourceTypeTranscription,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.CreateBackup(pptFile.ID)
	assert.NoError(t, err)

	var updated models.PPTFile
	db.First(&updated, pptFile.ID)
	assert.NotEmpty(t, updated.BackupPath)
	assert.Contains(t, updated.BackupPath, ".bak.")

	// Verify backup file exists
	_, err = os.Stat(updated.BackupPath)
	assert.NoError(t, err)
}

func TestPPTEditorService_CreateBackup_AlreadyExists(t *testing.T) {
	service, db, tempDir := setupPPTEditorServiceTest(t)

	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	pptFile := &models.PPTFile{
		FileName:   "test.pptx",
		FilePath:   pptPath,
		FileSize:   1024,
		PageCount:  10,
		Format:     "pptx",
		SourceType: models.PPTSourceTypeTranscription,
		BackupPath: "/some/existing/backup.pptx",
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.CreateBackup(pptFile.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestPPTEditorService_CreateBackup_PPTNotFound(t *testing.T) {
	service, _, _ := setupPPTEditorServiceTest(t)

	err := service.CreateBackup(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPPTEditorService_DeleteSlides_Success(t *testing.T) {
	requireExtractSlidesDeps(t)
	service, db, tempDir := setupPPTEditorServiceTest(t)

	// Create test PPT file
	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	// Create slide cache directory with mock slides
	cacheDir := filepath.Join(tempDir, "ppts", "1", "slides")
	fullsizeDir := filepath.Join(cacheDir, "fullsize")
	thumbnailDir := filepath.Join(cacheDir, "thumbnails")
	require.NoError(t, os.MkdirAll(fullsizeDir, 0755))
	require.NoError(t, os.MkdirAll(thumbnailDir, 0755))

	// Create mock slide images
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(fullsizeDir, formatTestSlideFilename(i))
		createTestSlideImage(t, filename, 1920, 1080)

		thumbFilename := filepath.Join(thumbnailDir, formatTestSlideFilename(i))
		createTestSlideImage(t, thumbFilename, 320, 180)
	}

	pptFile := &models.PPTFile{
		FileName:       "test.pptx",
		FilePath:       pptPath,
		FileSize:       1024,
		PageCount:      5,
		Format:         "pptx",
		SourceType:     models.PPTSourceTypeTranscription,
		SlideCachePath: cacheDir,
	}
	require.NoError(t, db.Create(pptFile).Error)

	// Delete slides 2 and 4
	err := service.DeleteSlides(pptFile.ID, []int{2, 4})
	assert.NoError(t, err)

	var updated models.PPTFile
	db.First(&updated, pptFile.ID)

	// Verify backup was created
	assert.NotEmpty(t, updated.BackupPath)

	// Verify deleted slides are recorded
	deletedSlides, err := updated.GetDeletedSlides()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []int{2, 4}, deletedSlides)

	// Verify page count was updated
	assert.Equal(t, 3, updated.PageCount)
}

func TestPPTEditorService_DeleteSlides_EmptySlideArray(t *testing.T) {
	service, db, tempDir := setupPPTEditorServiceTest(t)

	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	pptFile := &models.PPTFile{
		FileName:   "test.pptx",
		FilePath:   pptPath,
		FileSize:   1024,
		PageCount:  5,
		Format:     "pptx",
		SourceType: models.PPTSourceTypeTranscription,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.DeleteSlides(pptFile.ID, []int{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no slides specified")
}

func TestPPTEditorService_DeleteSlides_AllSlides(t *testing.T) {
	service, db, tempDir := setupPPTEditorServiceTest(t)

	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	pptFile := &models.PPTFile{
		FileName:   "test.pptx",
		FilePath:   pptPath,
		FileSize:   1024,
		PageCount:  3,
		Format:     "pptx",
		SourceType: models.PPTSourceTypeTranscription,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.DeleteSlides(pptFile.ID, []int{1, 2, 3})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete all slides")
}

func TestPPTEditorService_DeleteSlides_InvalidSlideNumber(t *testing.T) {
	service, db, tempDir := setupPPTEditorServiceTest(t)

	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	pptFile := &models.PPTFile{
		FileName:   "test.pptx",
		FilePath:   pptPath,
		FileSize:   1024,
		PageCount:  5,
		Format:     "pptx",
		SourceType: models.PPTSourceTypeTranscription,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.DeleteSlides(pptFile.ID, []int{1, 10}) // 10 is out of range
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid slide number")
}

func TestPPTEditorService_Rollback_Success(t *testing.T) {
	service, db, tempDir := setupPPTEditorServiceTest(t)

	// Create PPT file
	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	// Create backup
	backupPath := pptPath + ".bak.123456"
	require.NoError(t, os.WriteFile(backupPath, []byte("original content"), 0644))

	pptFile := &models.PPTFile{
		FileName:      "test.pptx",
		FilePath:      pptPath,
		FileSize:      1024,
		PageCount:     5,
		Format:        "pptx",
		SourceType:    models.PPTSourceTypeTranscription,
		BackupPath:    backupPath,
		DeletedSlides: "[1,3]",
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.Rollback(pptFile.ID)
	assert.NoError(t, err)

	var updated models.PPTFile
	db.First(&updated, pptFile.ID)

	// Verify backup path is cleared
	assert.Empty(t, updated.BackupPath)

	// Verify deleted slides is cleared
	deletedSlides, err := updated.GetDeletedSlides()
	assert.NoError(t, err)
	assert.Empty(t, deletedSlides)
}

func TestPPTEditorService_Rollback_NoBackup(t *testing.T) {
	service, db, tempDir := setupPPTEditorServiceTest(t)

	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	pptFile := &models.PPTFile{
		FileName:   "test.pptx",
		FilePath:   pptPath,
		FileSize:   1024,
		PageCount:  5,
		Format:     "pptx",
		SourceType: models.PPTSourceTypeTranscription,
	}
	require.NoError(t, db.Create(pptFile).Error)

	err := service.Rollback(pptFile.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backup exists")
}

func TestPPTEditorService_Rollback_PPTNotFound(t *testing.T) {
	service, _, _ := setupPPTEditorServiceTest(t)

	err := service.Rollback(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPPTEditorService_DetectDuplicateSlides_LessThanTwoSlides(t *testing.T) {
	requireExtractSlidesDeps(t)
	service, db, tempDir := setupPPTEditorServiceTest(t)

	pptPath := createTestPPTFile(t, tempDir, "test.pptx")

	pptFile := &models.PPTFile{
		FileName:   "test.pptx",
		FilePath:   pptPath,
		FileSize:   1024,
		PageCount:  1,
		Format:     "pptx",
		SourceType: models.PPTSourceTypeTranscription,
	}
	require.NoError(t, db.Create(pptFile).Error)

	groups, err := service.DetectDuplicateSlides(pptFile.ID)
	assert.NoError(t, err)
	assert.Empty(t, groups)
}

func TestPPTEditorService_DetectDuplicateSlides_PPTNotFound(t *testing.T) {
	service, _, _ := setupPPTEditorServiceTest(t)

	groups, err := service.DetectDuplicateSlides(999)
	assert.Error(t, err)
	assert.Nil(t, groups)
	assert.Contains(t, err.Error(), "not found")
}

// Helper function to format slide filename
func formatTestSlideFilename(slideNum int) string {
	return filepath.Join("fullsize", "slide_001.jpg")
}
