package services

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cpic/record_v2/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := ":memory:"
	sqlDB, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移所有相关模型
	err = db.AutoMigrate(
		&models.VideoFile{},
		&models.VideoRecordingTask{},
	)
	require.NoError(t, err)

	return db
}

// cleanupTestDB 清理测试数据库
func cleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()
}

// setupTestService 创建测试服务
func setupTestService(t *testing.T) (*VideoFileService, *gorm.DB, string) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})

	logger := zap.NewNop()
	tempDir := t.TempDir()
	// 测试中不需要实际的 ffprobe，传空字符串使用默认值
	service := NewVideoFileService(db, logger, tempDir, "")

	return service, db, tempDir
}

// createTestVideoFile 创建测试视频文件
func createTestVideoFile(t *testing.T, dir string, filename string, content string) string {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// createTestTask 创建测试录制任务
func createTestTask(t *testing.T, db *gorm.DB, mkvPath, mp4Path string) *models.VideoRecordingTask {
	configID := uint(1)
	task := &models.VideoRecordingTask{
		Name:             "测试任务",
		ConferenceNumber: "123456789",
		HuaweiConfigID:   &configID,
		Status:           models.VideoStatusCompleted,
		StartTime:        time.Now().Add(-1 * time.Hour),
		EndTime:          time.Now(),
		MKVFilePath:      mkvPath,
		MP4FilePath:      mp4Path,
		CreatedBy:        1,
		ConversionStatus: models.ConversionStatusCompleted,
	}

	err := db.Create(task).Error
	require.NoError(t, err)

	return task
}

// TestCreateFileFromTask 测试从任务创建文件记录
func TestCreateFileFromTask(t *testing.T) {
	t.Run("成功创建MKV文件记录", func(t *testing.T) {
		service, db, tempDir := setupTestService(t)

		// 创建测试文件
		mkvPath := createTestVideoFile(t, tempDir, "test_video.mkv", "fake mkv content")
		task := createTestTask(t, db, mkvPath, "")

		// 创建文件记录
		mkv := "mkv"
		file, err := service.CreateFileFromTask(task, &mkv)

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, "test_video.mkv", file.FileName)
		assert.Equal(t, mkvPath, file.FilePath)
		assert.Equal(t, "mkv", file.Format)

		// 验证数据库中的记录
		var dbFile models.VideoFile
		err = db.First(&dbFile, file.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, file.FileName, dbFile.FileName)
	})

	t.Run("成功创建MP4文件记录", func(t *testing.T) {
		service, db, tempDir := setupTestService(t)

		// 创建测试文件
		mp4Path := createTestVideoFile(t, tempDir, "test_video.mp4", "fake mp4 content")
		task := createTestTask(t, db, "", mp4Path)

		// 创建文件记录
		mp4 := "mp4"
		file, err := service.CreateFileFromTask(task, &mp4)

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, "test_video.mp4", file.FileName)
		assert.Equal(t, mp4Path, file.FilePath)
		assert.Equal(t, "mp4", file.Format)
	})

	t.Run("文件不存在时返回错误", func(t *testing.T) {
		service, db, _ := setupTestService(t)

		task := createTestTask(t, db, "/nonexistent/path/video.mkv", "")

		// 尝试创建文件记录
		mkv := "mkv"
		file, err := service.CreateFileFromTask(task, &mkv)

		// 验证
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "文件不存在")
	})

	t.Run("nil任务参数", func(t *testing.T) {
		service, _, _ := setupTestService(t)

		// 尝试使用nil任务创建文件
		file, err := service.CreateFileFromTask(nil, nil)

		// 验证
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "任务对象为 nil")
	})

	t.Run("不支持的格式", func(t *testing.T) {
		service, db, tempDir := setupTestService(t)

		mkvPath := createTestVideoFile(t, tempDir, "test.avi", "fake avi content")
		task := createTestTask(t, db, mkvPath, "")

		// 尝试使用不支持的格式
		avi := "avi"
		file, err := service.CreateFileFromTask(task, &avi)

		// 验证
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "不支持的格式")
	})

	t.Run("空文件路径", func(t *testing.T) {
		service, db, _ := setupTestService(t)

		task := createTestTask(t, db, "", "")

		// 尝试创建文件记录
		mkv := "mkv"
		file, err := service.CreateFileFromTask(task, &mkv)

		// 验证
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "文件路径为空")
	})

	t.Run("重复路径幂等性", func(t *testing.T) {
		service, db, tempDir := setupTestService(t)

		// 创建测试文件
		mkvPath := createTestVideoFile(t, tempDir, "test_video.mkv", "fake mkv content")
		task := createTestTask(t, db, mkvPath, "")

		// 第一次创建
		mkv := "mkv"
		file1, err1 := service.CreateFileFromTask(task, &mkv)
		assert.NoError(t, err1)
		assert.NotNil(t, file1)

		// 第二次创建（应该返回相同的记录）
		file2, err2 := service.CreateFileFromTask(task, &mkv)
		assert.NoError(t, err2)
		assert.NotNil(t, file2)
		assert.Equal(t, file1.ID, file2.ID)
		assert.Equal(t, file1.FilePath, file2.FilePath)

		// 验证数据库中只有一条记录
		var count int64
		db.Model(&models.VideoFile{}).Where("file_path = ?", mkvPath).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

// TestCreateFile 测试创建文件记录
func TestCreateFile(t *testing.T) {
	t.Run("文件不存在时返回错误", func(t *testing.T) {
		service, _, _ := setupTestService(t)

		// 尝试创建不存在的文件
		file, err := service.CreateFile("/nonexistent/path/video.mp4", nil, nil)

		// 验证
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "文件不存在")
	})

	t.Run("空文件路径", func(t *testing.T) {
		service, _, _ := setupTestService(t)

		// 尝试使用空路径创建文件
		_, err := service.CreateFile("", nil, nil)

		// 验证
		assert.Error(t, err)
	})

	t.Run("设置录制时间", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建测试文件
		filePath := createTestVideoFile(t, tempDir, "video.mp4", "fake video content")
		expectedTime := time.Now().Add(-1 * time.Hour)

		// 创建文件记录并设置录制时间
		file, err := service.CreateFile(filePath, nil, &expectedTime)

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.NotNil(t, file.RecordedAt)
		assert.WithinDuration(t, expectedTime, *file.RecordedAt, time.Second)
	})

	t.Run("自动识别文件格式", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 测试MKV格式
		mkvPath := createTestVideoFile(t, tempDir, "video.mkv", "fake mkv content")
		mkvFile, err := service.CreateFile(mkvPath, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "mkv", mkvFile.Format)

		// 测试MP4格式
		mp4Path := createTestVideoFile(t, tempDir, "video.mp4", "fake mp4 content")
		mp4File, err := service.CreateFile(mp4Path, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "mp4", mp4File.Format)
	})
}

// TestExtractVideoMetadata 测试元数据提取
func TestExtractVideoMetadata(t *testing.T) {
	t.Run("文件不存在时返回基础元数据", func(t *testing.T) {
		service, _, _ := setupTestService(t)

		// 尝试提取不存在文件的元数据
		metadata, err := service.extractVideoMetadata("/nonexistent/file.mkv")

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, "mkv", metadata.Format) // 默认格式
		// 文件不存在时，返回默认元数据值
		assert.Equal(t, "h264", metadata.Codec)
		assert.Equal(t, "1920x1080", metadata.Resolution)
	})

	t.Run("MP4文件格式识别", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建测试MP4文件
		mp4Path := createTestVideoFile(t, tempDir, "test.mp4", "fake mp4 content")

		// 提取元数据
		metadata, err := service.extractVideoMetadata(mp4Path)

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, "mp4", metadata.Format)
	})

	t.Run("MKV文件格式识别", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建测试MKV文件
		mkvPath := createTestVideoFile(t, tempDir, "test.mkv", "fake mkv content")

		// 提取元数据
		metadata, err := service.extractVideoMetadata(mkvPath)

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, "mkv", metadata.Format)
	})

	t.Run("ffprobe不可用时使用默认元数据", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建测试文件
		filePath := createTestVideoFile(t, tempDir, "test.mkv", "fake mkv content")

		// 提取元数据（ffprobe可能不可用）
		metadata, err := service.extractVideoMetadata(filePath)

		// 验证 - 无论ffprobe是否可用都应该返回元数据
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.NotEmpty(t, metadata.Format)
	})
}

// TestListFiles 测试文件列表功能
func TestListFiles(t *testing.T) {
	setupTestService := func() (*VideoFileService, *gorm.DB, string) {
		db := setupTestDB(t)
		t.Cleanup(func() {
			cleanupTestDB(t, db)
		})

		logger := zap.NewNop()
		tempDir := t.TempDir()
		service := NewVideoFileService(db, logger, tempDir, "")

		return service, db, tempDir
	}

	t.Run("分页查询", func(t *testing.T) {
		service, db, _ := setupTestService()

		// 创建5个测试文件
		for i := 1; i <= 5; i++ {
			filePath := fmt.Sprintf("test_file_%d.mkv", i)
			db.Create(&models.VideoFile{
				FileName: filePath,
				FilePath: filePath,
				FileSize: int64(i * 1024),
				Duration: i * 60,
				Format:   "mkv",
				Status:   models.FileStatusReady,
			})
		}

		// 第一页
		req1 := &ListFilesRequest{
			Page:     1,
			PageSize: 2,
		}
		resp1, err := service.ListFiles(req1)
		assert.NoError(t, err)
		assert.Len(t, resp1.Items, 2)
		assert.Equal(t, int64(5), resp1.Total)

		// 第二页
		req2 := &ListFilesRequest{
			Page:     2,
			PageSize: 2,
		}
		resp2, err := service.ListFiles(req2)
		assert.NoError(t, err)
		assert.Len(t, resp2.Items, 2)
		assert.Equal(t, int64(5), resp2.Total)
	})

	t.Run("关键词搜索", func(t *testing.T) {
		service, _, tempDir := setupTestService()

		// 创建不同名称的文件
		filePath1 := createTestVideoFile(t, tempDir, "meeting_001.mp4", "content 1")
		filePath2 := createTestVideoFile(t, tempDir, "meeting_002.mp4", "content 2")
		filePath3 := createTestVideoFile(t, tempDir, "training_video.mp4", "content 3")

		_, err := service.CreateFile(filePath1, nil, nil)
		require.NoError(t, err)
		_, err = service.CreateFile(filePath2, nil, nil)
		require.NoError(t, err)
		_, err = service.CreateFile(filePath3, nil, nil)
		require.NoError(t, err)

		// 搜索包含"meeting"的文件
		req := &ListFilesRequest{
			Page:     1,
			PageSize: 10,
			Keyword:  "meeting",
		}
		resp, err := service.ListFiles(req)

		// 验证
		assert.NoError(t, err)
		assert.Equal(t, int64(2), resp.Total)
		for _, item := range resp.Items {
			assert.Contains(t, strings.ToLower(item.FileName), "meeting")
		}
	})

	t.Run("格式筛选", func(t *testing.T) {
		service, _, tempDir := setupTestService()

		// 创建不同格式的文件
		mkvPath := createTestVideoFile(t, tempDir, "video1.mkv", "mkv content")
		mp4Path := createTestVideoFile(t, tempDir, "video2.mp4", "mp4 content")
		mp4Path2 := createTestVideoFile(t, tempDir, "video3.mp4", "mp4 content 2")

		_, err := service.CreateFile(mkvPath, nil, nil)
		require.NoError(t, err)
		_, err = service.CreateFile(mp4Path, nil, nil)
		require.NoError(t, err)
		_, err = service.CreateFile(mp4Path2, nil, nil)
		require.NoError(t, err)

		// 筛选MP4格式
		req := &ListFilesRequest{
			Page:     1,
			PageSize: 10,
			Format:   "mp4",
		}
		resp, err := service.ListFiles(req)

		// 验证
		assert.NoError(t, err)
		assert.Equal(t, int64(2), resp.Total)
		for _, item := range resp.Items {
			assert.Equal(t, "mp4", item.Format)
		}
	})

	t.Run("状态筛选", func(t *testing.T) {
		service, db, tempDir := setupTestService()

		// 创建不同状态的文件
		filePath1 := createTestVideoFile(t, tempDir, "video1.mp4", "content 1")
		filePath2 := createTestVideoFile(t, tempDir, "video2.mp4", "content 2")

		_, err := service.CreateFile(filePath1, nil, nil)
		require.NoError(t, err)

		// 创建一个processing状态的文件
		processingFile := &models.VideoFile{
			FileName: "video2.mp4",
			FilePath: filePath2,
			FileSize: 100,
			Status:   models.FileStatusProcessing,
			Format:   "mp4",
		}
		err = db.Create(processingFile).Error
		require.NoError(t, err)

		// 筛选ready状态
		req := &ListFilesRequest{
			Page:     1,
			PageSize: 10,
			Status:   models.FileStatusReady,
		}
		resp, err := service.ListFiles(req)

		// 验证
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.Total)
		assert.Equal(t, models.FileStatusReady, resp.Items[0].Status)
	})

	t.Run("任务ID筛选", func(t *testing.T) {
		service, db, tempDir := setupTestService()

		// 创建录制任务
		task1 := createTestTask(t, db, "", "")
		task2 := createTestTask(t, db, "", "")

		// 创建不同任务的文件
		filePath1 := createTestVideoFile(t, tempDir, "video1.mp4", "content 1")
		filePath2 := createTestVideoFile(t, tempDir, "video2.mp4", "content 2")

		_, err := service.CreateFile(filePath1, &task1.ID, nil)
		require.NoError(t, err)
		_, err = service.CreateFile(filePath2, &task2.ID, nil)
		require.NoError(t, err)

		// 筛选任务1的文件
		req := &ListFilesRequest{
			Page:     1,
			PageSize: 10,
			TaskID:   &task1.ID,
		}
		resp, err := service.ListFiles(req)

		// 验证
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.Total)
		assert.Equal(t, task1.ID, *resp.Items[0].TaskID)
	})

	t.Run("空列表", func(t *testing.T) {
		service, _, _ := setupTestService()

		// 查询空列表
		req := &ListFilesRequest{
			Page:     1,
			PageSize: 10,
		}
		resp, err := service.ListFiles(req)

		// 验证
		assert.NoError(t, err)
		assert.Equal(t, int64(0), resp.Total)
		assert.Len(t, resp.Items, 0)
	})
}

// TestGetFileByID 测试根据ID获取文件
func TestGetFileByID(t *testing.T) {
	t.Run("成功获取文件", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建测试文件
		filePath := createTestVideoFile(t, tempDir, "video.mp4", "fake video content")
		createdFile, err := service.CreateFile(filePath, nil, nil)
		require.NoError(t, err)

		// 根据ID获取文件
		file, err := service.GetFileByID(createdFile.ID)

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, createdFile.ID, file.ID)
		assert.Equal(t, createdFile.FileName, file.FileName)
	})

	t.Run("文件不存在", func(t *testing.T) {
		service, _, _ := setupTestService(t)

		// 尝试获取不存在的文件
		file, err := service.GetFileByID(99999)

		// 验证
		assert.Error(t, err)
		assert.Nil(t, file)
	})
}

// TestDeleteFile 测试删除文件
func TestDeleteFile(t *testing.T) {
	t.Run("成功删除文件", func(t *testing.T) {
		service, db, _ := setupTestService(t)

		// 首先创建一个录制任务
		task := createTestTask(t, db, "", "")
		taskID := task.ID

		// 创建一个文件记录并关联任务（不需要实际物理文件，因为 DeleteFile 删除的是任务目录）
		file := &models.VideoFile{
			FileName: "video.mp4",
			FilePath: "/tmp/video.mp4",
			FileSize: 100,
			Status:   models.FileStatusReady,
			Format:   "mp4",
			TaskID:   &taskID,
		}
		err := db.Create(file).Error
		require.NoError(t, err)

		// 验证文件记录存在
		var countBefore int64
		db.Model(&models.VideoFile{}).Where("id = ?", file.ID).Count(&countBefore)
		assert.Equal(t, int64(1), countBefore)

		// 删除文件
		err = service.DeleteFile(file.ID)
		assert.NoError(t, err)

		// 验证数据库记录已删除
		var countAfter int64
		db.Model(&models.VideoFile{}).Where("id = ?", file.ID).Count(&countAfter)
		assert.Equal(t, int64(0), countAfter)
	})

	t.Run("删除不存在的文件", func(t *testing.T) {
		service, _, _ := setupTestService(t)

		// 尝试删除不存在的文件
		err := service.DeleteFile(99999)

		// 验证
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文件不存在")
	})

	t.Run("删除processing状态的文件", func(t *testing.T) {
		service, db, tempDir := setupTestService(t)

		// 创建一个processing状态的文件
		filePath := createTestVideoFile(t, tempDir, "video.mp4", "fake video content")
		processingFile := &models.VideoFile{
			FileName: "video.mp4",
			FilePath: filePath,
			FileSize: 100,
			Status:   models.FileStatusProcessing,
			Format:   "mp4",
		}
		err := db.Create(processingFile).Error
		require.NoError(t, err)

		// 尝试删除processing状态的文件
		err = service.DeleteFile(processingFile.ID)

		// 验证
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "正在处理中")
	})

	t.Run("删除不存在的物理文件", func(t *testing.T) {
		service, db, _ := setupTestService(t)

		// 首先创建一个录制任务
		task := createTestTask(t, db, "", "")
		taskID := task.ID

		// 创建一个文件记录，但物理文件不存在，并关联任务
		file := &models.VideoFile{
			FileName: "nonexistent.mp4",
			FilePath: "/nonexistent/path/nonexistent.mp4",
			FileSize: 100,
			Status:   models.FileStatusReady,
			Format:   "mp4",
			TaskID:   &taskID,
		}
		err := db.Create(file).Error
		require.NoError(t, err)

		// 删除文件记录（物理文件不存在应该也能删除记录）
		err = service.DeleteFile(file.ID)

		// 验证 - 应该成功删除数据库记录
		assert.NoError(t, err)

		var count int64
		db.Model(&models.VideoFile{}).Where("id = ?", file.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("查找视频文件", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建临时录制目录结构
		recordingsDir := filepath.Join(tempDir, "data", "recordings")
		err := os.MkdirAll(recordingsDir, 0755)
		require.NoError(t, err)

		// 创建测试视频文件 - findVideoFiles 只扫描 .mp4 文件
		mkvFile := filepath.Join(recordingsDir, "video1.mkv")
		mp4File := filepath.Join(recordingsDir, "video2.mp4")
		err = os.WriteFile(mkvFile, []byte("fake mkv"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(mp4File, []byte("fake mp4"), 0644)
		require.NoError(t, err)

		// 测试 findVideoFiles 方法
		files, err := service.findVideoFiles(recordingsDir)
		assert.NoError(t, err)
		assert.Len(t, files, 1) // 只找到 mp4 文件
		assert.Equal(t, mp4File, files[0].filePath)
	})

	t.Run("混合已存在和不存在文件", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建临时录制目录
		recordingsDir := filepath.Join(tempDir, "data", "recordings")
		err := os.MkdirAll(recordingsDir, 0755)
		require.NoError(t, err)

		// 创建第一个 mp4 文件并入库
		filePath1 := filepath.Join(recordingsDir, "video1.mp4")
		err = os.WriteFile(filePath1, []byte("fake mp4 1"), 0644)
		require.NoError(t, err)
		_, err = service.CreateFile(filePath1, nil, nil)
		require.NoError(t, err)

		// 创建第二个 mp4 文件但未入库
		filePath2 := filepath.Join(recordingsDir, "video2.mp4")
		err = os.WriteFile(filePath2, []byte("fake mp4 2"), 0644)
		require.NoError(t, err)

		// 查找所有文件
		files, err := service.findVideoFiles(recordingsDir)
		assert.NoError(t, err)
		assert.Len(t, files, 2)

		// 创建未入库的文件
		file, err := service.CreateFile(filePath2, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, file)

		// 验证幂等性 - 再次创建应该返回相同记录
		file2, err := service.CreateFile(filePath2, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, file.ID, file2.ID)
	})

	t.Run("查找非视频文件", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建目录并混合不同类型的文件
		recordingsDir := filepath.Join(tempDir, "recordings")
		err := os.MkdirAll(recordingsDir, 0755)
		require.NoError(t, err)

		// 创建不同类型的文件
		err = os.WriteFile(filepath.Join(recordingsDir, "video.mkv"), []byte("mkv"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(recordingsDir, "video.mp4"), []byte("mp4"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(recordingsDir, "text.txt"), []byte("text"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(recordingsDir, "image.jpg"), []byte("image"), 0644)
		require.NoError(t, err)

		// 查找文件
		files, err := service.findVideoFiles(recordingsDir)

		// 验证只返回 .mp4 文件（findVideoFiles 只扫描 .mp4）
		assert.NoError(t, err)
		assert.Len(t, files, 1)
		for _, file := range files {
			ext := strings.ToLower(filepath.Ext(file.filePath))
			assert.Equal(t, ".mp4", ext)
		}
	})
}

// TestFindVideoFiles 测试查找视频文件
func TestFindVideoFiles(t *testing.T) {
	t.Run("递归查找子目录", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建多级目录结构
		subDir1 := filepath.Join(tempDir, "level1")
		subDir2 := filepath.Join(subDir1, "level2")
		err := os.MkdirAll(subDir2, 0755)
		require.NoError(t, err)

		// 在不同层级创建文件 - 注意：findVideoFiles 只扫描 .mp4 文件
		err = os.WriteFile(filepath.Join(tempDir, "root.mp4"), []byte("root"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subDir1, "level1.mp4"), []byte("level1"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subDir2, "level2.mp4"), []byte("level2"), 0644)
		require.NoError(t, err)
		// MKV 文件会被忽略
		err = os.WriteFile(filepath.Join(tempDir, "ignored.mkv"), []byte("ignored"), 0644)
		require.NoError(t, err)

		// 查找文件
		files, err := service.findVideoFiles(tempDir)

		// 验证只找到 .mp4 文件（3个）
		assert.NoError(t, err)
		assert.Len(t, files, 3)
	})
}

// TestListFilesWithDateFilter 测试日期筛选
func TestListFilesWithDateFilter(t *testing.T) {
	t.Run("日期范围筛选", func(t *testing.T) {
		service, _, tempDir := setupTestService(t)

		// 创建不同时间的文件
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		twoDaysAgo := now.Add(-48 * time.Hour)

		filePath1 := createTestVideoFile(t, tempDir, "video1.mp4", "content 1")
		filePath2 := createTestVideoFile(t, tempDir, "video2.mp4", "content 2")
		filePath3 := createTestVideoFile(t, tempDir, "video3.mp4", "content 3")

		// 设置不同的创建时间（通过直接操作数据库）
		_, err := service.CreateFile(filePath1, nil, &twoDaysAgo)
		require.NoError(t, err)
		_, err = service.CreateFile(filePath2, nil, &yesterday)
		require.NoError(t, err)
		_, err = service.CreateFile(filePath3, nil, &now)
		require.NoError(t, err)

		// 筛选最近2天的文件
		req := &ListFilesRequest{
			Page:      1,
			PageSize:  10,
			StartDate: yesterday.Format("2006-01-02"),
		}
		resp, err := service.ListFiles(req)

		// 验证
		assert.NoError(t, err)
		// 应该包含昨天和今天的文件
		assert.GreaterOrEqual(t, resp.Total, int64(2))
	})
}

// BenchmarkListFiles 性能测试
func BenchmarkListFiles(b *testing.B) {
	// Benchmark 需要自己设置测试环境
	logger := zap.NewNop()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	err = db.AutoMigrate(&models.VideoFile{})
	if err != nil {
		b.Fatal(err)
	}

	tempDir := b.TempDir()
	service := NewVideoFileService(db, logger, tempDir, "")

	// 创建100个文件用于基准测试
	b.StopTimer()
	for i := 0; i < 100; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("video%d.mp4", i))
		content := []byte(fmt.Sprintf("content %d", i))
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			b.Fatal(err)
		}

		fileInfo, _ := os.Stat(filePath)
		videoFile := &models.VideoFile{
			FileName: filepath.Base(filePath),
			FilePath: filePath,
			FileSize: fileInfo.Size(),
			Status:   models.FileStatusReady,
		}
		if err := db.Create(videoFile).Error; err != nil {
			b.Fatal(err)
		}
	}
	b.StartTimer()

	// 重置计时器并开始基准测试
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &ListFilesRequest{
			Page:     1,
			PageSize: 20,
		}
		service.ListFiles(req)
	}
}

// --- CreateSegmentFile and auto-scan test stubs (Wave 0) ---

func TestVideoFileService_CreateSegmentFile(t *testing.T) {
	t.Skip("waiting for CreateSegmentFile implementation")
	// SCAN-01: Create segment file record with parent_id and source_type
	// Setup: create test service and DB, create a parent VideoFile record
	// Action: call CreateSegmentFile(segmentPath, &parentID, "split", userID)
	// Assert: VideoFile record created with parent_id set, source_type="split"
}

func TestVideoFileService_CreateSegmentFile_Snapshot(t *testing.T) {
	t.Skip("waiting for CreateSegmentFile implementation")
	// SCAN-01: Create snapshot file record
	// Setup: create test service and DB, create a parent VideoFile record
	// Action: call CreateSegmentFile(segmentPath, &parentID, "snapshot", userID)
	// Assert: VideoFile record created with source_type="snapshot"
}

func TestVideoFileService_CreateSegmentFile_Duplicate(t *testing.T) {
	t.Skip("waiting for CreateSegmentFile implementation")
	// SCAN-01: Duplicate segment path returns existing record
	// Setup: create test service, create first segment file record
	// Action: call CreateSegmentFile with same path again
	// Assert: returns existing record without error, no duplicate created
}

func TestVideoFileService_GetSegmentsByParentID(t *testing.T) {
	t.Skip("waiting for GetSegmentsByParentID implementation")
	// SPLIT-04: Retrieve all child segments of a parent video
	// Setup: create test service, create parent + 3 child VideoFile records
	// Action: call GetSegmentsByParentID(parentID)
	// Assert: returns 3 segments ordered by ID ASC
}

func TestVideoFileService_ListFiles_SourceTypeFilter(t *testing.T) {
	t.Skip("waiting for source_type filter implementation")
	// SCAN-01: List files filtered by source_type
	// Setup: create test service, create files with source_type recording/snapshot/split
	// Action: call ListFiles with SourceType="split"
	// Assert: only split files returned
}

func TestVideoFileService_AutoScan_SplitCallback(t *testing.T) {
	t.Skip("waiting for auto-scan callback implementation")
	// SCAN-01: SplittingService calls CreateSegmentFile on completion
	// This test verifies the callback pattern: after split completes,
	// CreateSegmentFile is called for each segment, and all segments
	// appear in the database with correct parent_id and source_type
}

// --- RenameVideoFile tests (Phase 05) ---

func TestVideoFileService_RenameVideoFile_Success(t *testing.T) {
	service, db, tempDir := setupTestService(t)

	// Create test user and video file
	userID := uint(1)
	parentID := uint(42)
	videoFile := &models.VideoFile{
		FileName:   "test_video.mp4",
		FilePath:   createTestVideoFile(t, tempDir, "test_video.mp4", "fake video content"),
		FileSize:   1024,
		Duration:   60,
		Format:     "mp4",
		Resolution: "1920x1080",
		Bitrate:    5000,
		Codec:      "h264",
		SourceType: models.SourceTypeSplit,
		ParentID:   &parentID,
		CreatedBy:  userID,
		Status:     models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	// Rename the file
	err := service.RenameVideoFile(videoFile.ID, "new_video_name", userID)
	assert.NoError(t, err)

	// Verify database was updated
	var updated models.VideoFile
	err = db.First(&updated, videoFile.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "new_video_name.mp4", updated.FileName)
	assert.True(t, strings.HasSuffix(updated.FilePath, "new_video_name.mp4"))

	// Verify physical file was renamed
	_, err = os.Stat(updated.FilePath)
	assert.NoError(t, err, "Renamed file should exist")

	// Verify old file path no longer exists
	_, err = os.Stat(videoFile.FilePath)
	assert.True(t, os.IsNotExist(err), "Old file path should not exist")
}

func TestVideoFileService_RenameVideoFile_PreservesExtension(t *testing.T) {
	service, db, tempDir := setupTestService(t)

	userID := uint(1)
	parentID := uint(42)
	videoFile := &models.VideoFile{
		FileName:   "test_video.mp4",
		FilePath:   createTestVideoFile(t, tempDir, "test_video.mp4", "fake video content"),
		FileSize:   1024,
		Duration:   60,
		Format:     "mp4",
		SourceType: models.SourceTypeSplit,
		ParentID:   &parentID,
		CreatedBy:  userID,
		Status:     models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	// Try to rename with custom extension (should be ignored)
	err := service.RenameVideoFile(videoFile.ID, "new_name.mkv", userID)
	assert.NoError(t, err)

	var updated models.VideoFile
	db.First(&updated, videoFile.ID)
	assert.Equal(t, "new_name.mp4", updated.FileName, "Should preserve .mp4 extension")
	assert.True(t, strings.HasSuffix(updated.FilePath, ".mp4"))
}

func TestVideoFileService_RenameVideoFile_OriginalRecordingImmutable(t *testing.T) {
	service, db, tempDir := setupTestService(t)

	userID := uint(1)
	videoFile := &models.VideoFile{
		FileName:   "test_video.mp4",
		FilePath:   createTestVideoFile(t, tempDir, "test_video.mp4", "fake video content"),
		FileSize:   1024,
		Duration:   60,
		Format:     "mp4",
		SourceType: models.SourceTypeRecording,
		ParentID:   nil, // Original recording
		CreatedBy:  userID,
		Status:     models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	// Try to rename original recording
	err := service.RenameVideoFile(videoFile.ID, "new_name", userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不能重命名原始录制")
}

func TestVideoFileService_RenameVideoFile_OwnershipValidation(t *testing.T) {
	service, db, tempDir := setupTestService(t)

	userID := uint(1)
	otherUserID := uint(2)
	parentID := uint(42)
	videoFile := &models.VideoFile{
		FileName:   "test_video.mp4",
		FilePath:   createTestVideoFile(t, tempDir, "test_video.mp4", "fake video content"),
		FileSize:   1024,
		Duration:   60,
		Format:     "mp4",
		SourceType: models.SourceTypeSplit,
		ParentID:   &parentID,
		CreatedBy:  userID,
		Status:     models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	// Try to rename as different user
	err := service.RenameVideoFile(videoFile.ID, "new_name", otherUserID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无权重命名")
}

func TestVideoFileService_RenameVideoFile_RollbackOnFilesystemError(t *testing.T) {
	service, db, tempDir := setupTestService(t)

	userID := uint(1)
	parentID := uint(42)
	videoFile := &models.VideoFile{
		FileName:   "test_video.mp4",
		FilePath:   createTestVideoFile(t, tempDir, "test_video.mp4", "fake video content"),
		FileSize:   1024,
		Duration:   60,
		Format:     "mp4",
		SourceType: models.SourceTypeSplit,
		ParentID:   &parentID,
		CreatedBy:  userID,
		Status:     models.FileStatusReady,
	}
	require.NoError(t, db.Create(videoFile).Error)

	// Delete the physical file to simulate filesystem error
	require.NoError(t, os.Remove(videoFile.FilePath))

	// Try to rename (should fail due to missing source file)
	err := service.RenameVideoFile(videoFile.ID, "new_name", userID)
	assert.Error(t, err)

	// Verify database was NOT updated (transaction rolled back)
	var updated models.VideoFile
	err = db.First(&updated, videoFile.ID).Error
		require.NoError(t, err)
	assert.Equal(t, videoFile.FileName, updated.FileName, "FileName should not change")
	assert.Equal(t, videoFile.FilePath, updated.FilePath, "FilePath should not change")
}

func TestVideoFileService_RenameVideoFile_DuplicateDetection(t *testing.T) {
	service, db, tempDir := setupTestService(t)

	userID := uint(1)
	parentID := uint(42)

	// Create two split files from same parent
	file1 := &models.VideoFile{
		FileName:   "existing_video.mp4",
		FilePath:   createTestVideoFile(t, tempDir, "existing_video.mp4", "fake video content 1"),
		FileSize:   1024,
		Format:     "mp4",
		SourceType: models.SourceTypeSplit,
		ParentID:   &parentID,
		CreatedBy:  userID,
		Status:     models.FileStatusReady,
	}
	require.NoError(t, db.Create(file1).Error)

	file2 := &models.VideoFile{
		FileName:   "test_video.mp4",
		FilePath:   createTestVideoFile(t, tempDir, "test_video.mp4", "fake video content 2"),
		FileSize:   1024,
		Format:     "mp4",
		SourceType: models.SourceTypeSplit,
		ParentID:   &parentID,
		CreatedBy:  userID,
		Status:     models.FileStatusReady,
	}
	require.NoError(t, db.Create(file2).Error)

	// Rename file2 to the same name as file1
	err := service.RenameVideoFile(file2.ID, "existing_video", userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已存在")
}
