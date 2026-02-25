package storage

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	// 使用纯 Go 的 SQLite 驱动
	dsn := ":memory:"
	sqlDB, err := sql.Open("sqlite", dsn)
	assert.NoError(t, err)

	// 使用 GORM 包装数据库连接
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	assert.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(
		&models.UploadedFile{},
		&models.FileShare{},
		&models.UserStorageQuota{},
	)
	assert.NoError(t, err)

	return db
}

// cleanupTestDB 清理测试数据库
func cleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	sqlDB.Close()
}

// NewTestFileService 创建测试文件服务
func NewTestFileService(t *testing.T) *FileService {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				BasePath: t.TempDir(),
				BaseURL:  "http://localhost:8080/files",
			},
			MaxFileSize:       100 * 1024 * 1024, // 100MB for testing
			AllowedExtensions: []string{".txt", ".jpg", ".png"},
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	db := setupTestDB(t)
	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})

	return NewFileService(db, logger, cfg)
}

// TestLocalStorageDriver 测试本地存储驱动
func TestLocalStorageDriver(t *testing.T) {
	logger := zap.NewNop()
	basePath := filepath.Join(t.TempDir(), "files")
	baseURL := "http://localhost:8080/files"

	driver := NewLocalStorageDriver(basePath, baseURL, logger)

	t.Run("创建目录", func(t *testing.T) {
		_, err := os.Stat(basePath)
		assert.NoError(t, err)
	})

	t.Run("UploadAndDownload", func(t *testing.T) {
		// 创建测试文件内容
		content := []byte("test content")
		fileHeader := createTestFileHeader(t, "test.txt", content)

		ctx := context.Background()
		result, err := driver.Upload(ctx, fileHeader, "uploads/test.txt")
		assert.NoError(t, err)
		assert.NotEmpty(t, result.FilePath)

		// 验证文件已创建
		fullPath := filepath.Join(basePath, result.FilePath)
		_, err = os.Stat(fullPath)
		assert.NoError(t, err)

		// 下载文件
		reader, err := driver.Download(ctx, "uploads/test.txt")
		assert.NoError(t, err)

		downloadedContent, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, downloadedContent)

		// 显式关闭 reader 以释放文件句柄（特别是在 Windows 上）
		reader.Close()

		// 删除文件
		err = driver.Delete(ctx, "uploads/test.txt")
		assert.NoError(t, err)

		_, err = os.Stat(fullPath)
		assert.True(t, os.IsNotExist(err))
	})
}

// createTestFileHeader 创建测试用的 multipart.FileHeader
func createTestFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	// 创建一个缓冲区来写入 multipart form 数据
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	assert.NoError(t, err)

	_, err = part.Write(content)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	// 创建 http.Request 并解析 multipart form
	req, err := http.NewRequest("POST", "/upload", body)
	assert.NoError(t, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	err = req.ParseMultipartForm(32 << 20) // 32MB
	assert.NoError(t, err)

	_, fileHeader, err := req.FormFile("file")
	assert.NoError(t, err)

	return fileHeader
}

// TestFileServiceUpload 测试文件上传
func TestFileServiceUpload(t *testing.T) {
	service := NewTestFileService(t)

	t.Run("上传文件", func(t *testing.T) {
		// 创建测试文件内容
		content := []byte("test content for upload")
		fileHeader := createTestFileHeader(t, "test.txt", content)

		ctx := context.Background()
		req := &UploadRequest{
			File:     fileHeader,
			Folder:   "test",
			IsPublic: false,
		}

		result, err := service.Upload(ctx, 1, req)
		assert.NoError(t, err)
		assert.NotZero(t, result.FileID)
		assert.NotEmpty(t, result.FileName)
		assert.Equal(t, int64(len(content)), result.FileSize)
	})

	t.Run("检查用户配额", func(t *testing.T) {
		ctx := context.Background()
		quota, err := service.GetUserQuota(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(10*1024*1024*1024), quota.TotalQuota) // 10GB
		assert.Greater(t, quota.UsedQuota, int64(0))
	})

	t.Run("查询文件列表", func(t *testing.T) {
		ctx := context.Background()
		req := &QueryRequest{
			Page:     1,
			PageSize: 20,
		}

		result, err := service.Query(ctx, 1, req)
		assert.NoError(t, err)
		assert.NotNil(t, result.Items)
		assert.Greater(t, len(result.Items), 0)
	})
}

// TestFileServiceMD5 秒传测试
func TestFileServiceMD5(t *testing.T) {
	service := NewTestFileService(t)

	t.Run("MD5相同秒传", func(t *testing.T) {
		// 创建相同内容的文件
		content := []byte("duplicate content")

		// 第一次上传
		fileHeader := createTestFileHeader(t, "duplicate.txt", content)

		ctx := context.Background()
		req := &UploadRequest{
			File:     fileHeader,
			Folder:   "test",
			IsPublic: false,
		}

		result1, err := service.Upload(ctx, 1, req)
		assert.NoError(t, err)
		assert.NotZero(t, result1.FileID)

		// 第二次上传相同内容（应该返回相同文件ID）
		fileHeader2 := createTestFileHeader(t, "duplicate2.txt", content)

		req2 := &UploadRequest{
			File:     fileHeader2,
			Folder:   "test",
			IsPublic: false,
		}

		result2, err := service.Upload(ctx, 1, req2)
		assert.NoError(t, err)
		assert.Equal(t, result1.FileID, result2.FileID) // 秒传成功
	})
}

// TestFileServiceShare 测试文件分享
func TestFileServiceShare(t *testing.T) {
	service := NewTestFileService(t)

	t.Run("创建分享链接", func(t *testing.T) {
		// 首先上传一个文件
		content := []byte("share content")
		fileHeader := createTestFileHeader(t, "share.txt", content)

		ctx := context.Background()
		req := &UploadRequest{
			File:     fileHeader,
			Folder:   "test",
			IsPublic: false,
		}

		result, err := service.Upload(ctx, 1, req)
		assert.NoError(t, err)

		// 创建分享链接
		shareURL, err := service.ShareFile(ctx, result.FileID, 1, 24*time.Hour, "")
		assert.NoError(t, err)
		assert.Contains(t, shareURL, "/api/v1/files/share/")

		// 提取分享令牌
		parts := strings.Split(shareURL, "/")
		token := parts[len(parts)-1]

		// 通过分享链接下载
		reader, filename, err := service.GetShareDownload(ctx, token, "")
		assert.NoError(t, err)
		assert.Equal(t, "share.txt", filename)
		defer reader.Close()

		downloadedContent, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, downloadedContent)
	})

	t.Run("带密码的分享", func(t *testing.T) {
		// 上传文件
		content := []byte("protected content")
		fileHeader := createTestFileHeader(t, "protected.txt", content)

		ctx := context.Background()
		req := &UploadRequest{
			File:     fileHeader,
			Folder:   "test",
			IsPublic: false,
		}

		result, err := service.Upload(ctx, 1, req)
		assert.NoError(t, err)

		// 创建带密码的分享链接
		shareURL, err := service.ShareFile(ctx, result.FileID, 1, 24*time.Hour, "password123")
		assert.NoError(t, err)

		// 提取分享令牌
		parts := strings.Split(shareURL, "/")
		token := parts[len(parts)-1]

		// 使用正确密码下载
		reader, filename, err := service.GetShareDownload(ctx, token, "password123")
		assert.NoError(t, err)
		assert.Equal(t, "protected.txt", filename)
		defer reader.Close()

		downloadedContent, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, downloadedContent)

		// 使用错误密码下载
		_, _, err = service.GetShareDownload(ctx, token, "wrongpassword")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "密码错误")
	})
}

// TestFileServiceDelete 测试文件删除
func TestFileServiceDelete(t *testing.T) {
	service := NewTestFileService(t)

	t.Run("删除文件", func(t *testing.T) {
		// 上传文件
		content := []byte("delete content")
		fileHeader := createTestFileHeader(t, "delete.txt", content)

		ctx := context.Background()
		req := &UploadRequest{
			File:     fileHeader,
			Folder:   "test",
			IsPublic: false,
		}

		result, err := service.Upload(ctx, 1, req)
		assert.NoError(t, err)

		// 删除文件
		err = service.Delete(ctx, result.FileID, 1)
		assert.NoError(t, err)

		// 验证文件状态已更新
		var file models.UploadedFile
		err = service.db.First(&file, result.FileID).Error
		assert.NoError(t, err)
		assert.Equal(t, string(models.FileStatusDeleted), file.Status)
	})
}

// TestQuotaExceeded 测试配额超出
func TestQuotaExceeded(t *testing.T) {
	service := NewTestFileService(t)

	t.Run("配额不足", func(t *testing.T) {
		// 先创建一个小的配额记录
		quota := &models.UserStorageQuota{
			UserID:     1,
			TotalQuota: 100, // 100 bytes
			UsedQuota:  90,
			FileCount:  0,
		}
		err := service.db.Create(quota).Error
		assert.NoError(t, err)

		// 创建一个大文件 (200 bytes)
		content := make([]byte, 200)
		fileHeader := createTestFileHeader(t, "large.txt", content)

		ctx := context.Background()
		req := &UploadRequest{
			File:     fileHeader,
			Folder:   "test",
			IsPublic: false,
		}

		_, err = service.Upload(ctx, 1, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "存储空间不足")
	})
}
