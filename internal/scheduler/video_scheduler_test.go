package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// mockVideoFileService 模拟视频文件服务
type mockVideoFileService struct{}

func newMockVideoFileService() *mockVideoFileService {
	return &mockVideoFileService{}
}

func (m *mockVideoFileService) CreateFileFromTask(ctx context.Context, task *models.VideoRecordingTask, format *string) (*models.VideoFile, error) {
	// 处理 format 参数：如果为 nil 或空字符串，使用默认值 "mp4"
	formatStr := "mp4"
	if format != nil && *format != "" {
		formatStr = *format
	}
	// 测试中不需要真正创建文件记录，返回空记录即可
	return &models.VideoFile{
		FileName: fmt.Sprintf("test.%s", formatStr),
		FilePath: fmt.Sprintf("/tmp/test.%s", formatStr),
		Format:   formatStr,
		Status:   models.FileStatusReady,
	}, nil
}

// mockTaskService 模拟任务服务
type mockTaskService struct {
	tasks map[uint]*models.VideoRecordingTask
	db    *gorm.DB
}

func newMockTaskService() *mockTaskService {
	// 创建内存数据库用于测试（使用纯Go SQLite驱动）
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(fmt.Sprintf("创建测试数据库失败: %v", err))
	}

	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("创建GORM失败: %v", err))
	}

	// 自动迁移需要的表（Phase 13 重构后使用 InputConfig 架构）
	err = db.AutoMigrate(
		&models.InputConfig{},
		&models.TaskInputConfig{},
		&models.VideoRecordingTask{},
	)
	if err != nil {
		panic(fmt.Sprintf("迁移测试数据库失败: %v", err))
	}

	return &mockTaskService{
		tasks: make(map[uint]*models.VideoRecordingTask),
		db:    db,
	}
}

func (m *mockTaskService) GetTask(ctx context.Context, id uint) (*models.VideoRecordingTask, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (m *mockTaskService) GetPendingTasks(ctx context.Context) ([]models.VideoRecordingTask, error) {
	var result []models.VideoRecordingTask
	for _, task := range m.tasks {
		if task.Status == models.VideoStatusPending {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *mockTaskService) UpdateTaskStatus(ctx context.Context, id uint, status models.VideoRecordingTaskStatus, errorMsg string) error {
	task, ok := m.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	task.Status = status
	task.ErrorMsg = errorMsg
	return nil
}

func (m *mockTaskService) GetInputConfig(ctx context.Context, id uint) (*models.InputConfig, error) {
	return &models.InputConfig{
		Base:            models.Base{ID: id},
		Name:            "Test Config",
		ConfigType:      models.ConfigTypeUSB,
		Server:          "192.168.1.100",
		Port:            80,
		Username:        "admin",
		Password:        "password",
		OutputFormat:    "mp4",
		CameraBackend:   "dshow",
		USBCameraName:   "Integrated Camera",
		USBCameraDevice: "Integrated Camera",
		AudioBackend:    "dshow",
		USBAudioName:    "Microphone",
		USBAudioDevice:  "Microphone",
	}, nil
}

func (m *mockTaskService) UpdateRecordingPaths(ctx context.Context, id uint, mkvPath, hlsPath string) error {
	task, ok := m.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	task.MKVFilePath = mkvPath
	task.HLSPreviewPath = hlsPath
	return nil
}

func (m *mockTaskService) GetDB() *gorm.DB {
	return m.db
}

// mockCoordinator 模拟录制协调器
type mockCoordinator struct{}

func newMockCoordinator() *mockCoordinator {
	return &mockCoordinator{}
}

func (m *mockCoordinator) StartRecording(task *models.VideoRecordingTask, config *models.InputConfig) error {
	return nil
}

func (m *mockCoordinator) StartRecordingWithConfig(task *models.VideoRecordingTask, inputConfig *models.InputConfig, configType string) error {
	return nil
}

func (m *mockCoordinator) StopRecording(taskID uint) error {
	return nil
}

func (m *mockCoordinator) HealthCheck() error {
	return nil
}

// 测试辅助函数
func createTestTask(id uint, name string, startTime time.Time) *models.VideoRecordingTask {
	configID := uint(1)
	return &models.VideoRecordingTask{
		Base:               models.Base{ID: id},
		Name:               name,
		StartTime:          startTime,
		EndTime:            startTime.Add(1 * time.Hour),
		PreJoinMinutes:     5,
		RecordDelayMinutes: 0,
		ConferenceNumber:   "123456",
		InputConfigID:      &configID,
		Status:             models.VideoStatusPending,
	}
}

// setupTestDBData 设置测试数据库中的输入配置数据
func setupTestDBData(db *gorm.DB) {
	// 创建测试输入配置
	config := &models.InputConfig{
		Base:            models.Base{ID: 1},
		Name:            "Test Config",
		ConfigType:      models.ConfigTypeUSB,
		Server:          "192.168.1.100",
		Port:            80,
		Username:        "admin",
		Password:        "password",
		OutputFormat:    "mp4",
		CameraBackend:   "dshow",
		USBCameraName:   "Integrated Camera",
		USBCameraDevice: "Integrated Camera",
		AudioBackend:    "dshow",
		USBAudioName:    "Microphone",
		USBAudioDevice:  "Microphone",
	}
	db.Create(config)
}

// setupTaskWithConfig 设置任务及其关联配置
func setupTaskWithConfig(taskSvc *mockTaskService, task *models.VideoRecordingTask) {
	taskSvc.tasks[task.ID] = task
	// 设置测试数据库数据
	setupTestDBData(taskSvc.db)
	// 创建任务配置关联（使用新的 TaskInputConfig 多对多表）
	if task.InputConfigID != nil {
		taskConfig := &models.TaskInputConfig{
			TaskID:        task.ID,
			InputConfigID: *task.InputConfigID,
			ConfigType:    models.ConfigTypeUSB,
		}
		taskSvc.db.Create(taskConfig)
	}
}

// TestNewVideoSimpleScheduler 测试调度器创建
func TestNewVideoSimpleScheduler(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()

	scheduler := NewVideoSimpleScheduler(taskSvc, coord, nil, nil, newMockVideoFileService(), logger, cfg)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.cron)
	assert.NotNil(t, scheduler.taskEntries)
	assert.NotNil(t, scheduler.entryTasks)
	assert.NotNil(t, scheduler.executing)
	assert.NotNil(t, scheduler.cancelFuncs)
}

// TestCalculateTriggerTime 测试触发时间计算
func TestCalculateTriggerTime(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	scheduler := NewVideoSimpleScheduler(newMockTaskService(), newMockCoordinator(), nil, nil, newMockVideoFileService(), logger, cfg)

	startTime := time.Date(2026, 2, 10, 14, 30, 0, 0, time.UTC)
	triggerTime := scheduler.calculateTriggerTime(startTime, 5)

	expected := time.Date(2026, 2, 10, 14, 25, 0, 0, time.UTC)
	assert.Equal(t, expected, triggerTime)
}

// TestGenerateCronExpression 测试Cron表达式生成
func TestGenerateCronExpression(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	scheduler := NewVideoSimpleScheduler(newMockTaskService(), newMockCoordinator(), nil, nil, newMockVideoFileService(), logger, cfg)

	triggerTime := time.Date(2026, 2, 10, 14, 25, 30, 0, time.UTC)
	cronExpr := scheduler.generateCronExpression(triggerTime)

	expected := "30 25 14 10 2 *"
	assert.Equal(t, expected, cronExpr)
}

// TestAddTask 测试添加任务
func TestAddTask(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, nil, nil, newMockVideoFileService(), logger, cfg)

	// 启动调度器
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()

	// 创建测试任务
	startTime := time.Now().Add(1 * time.Hour)
	task := createTestTask(1, "Test Task", startTime)
	setupTaskWithConfig(taskSvc, task)

	// 添加任务
	err = scheduler.AddTask(task)
	assert.NoError(t, err)

	// 验证任务已添加
	assert.True(t, scheduler.IsTaskScheduled(1))
}

// TestAddTask_DuplicateTask 测试重复添加任务
func TestAddTask_DuplicateTask(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, nil, nil, newMockVideoFileService(), logger, cfg)

	// 启动调度器
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()

	// 创建测试任务
	startTime := time.Now().Add(1 * time.Hour)
	task := createTestTask(1, "Test Task", startTime)
	setupTaskWithConfig(taskSvc, task)

	// 第一次添加
	err = scheduler.AddTask(task)
	assert.NoError(t, err)

	// 第二次添加（应该失败）
	err = scheduler.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已在调度器中")
}

// TestAddTask_ExpiredTime 测试过期任务
func TestAddTask_ExpiredTime(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, nil, nil, newMockVideoFileService(), logger, cfg)

	// 启动调度器
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()

	// 创建已过期的任务（结束时间已过）
	startTime := time.Now().Add(-2 * time.Hour)
	task := createTestTask(1, "Expired Task", startTime)
	// 任务的结束时间是 startTime + 1小时 = 当前时间 - 1小时（已过期）

	// 添加过期任务应该失败
	err = scheduler.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "任务已过期")
}

// TestAddTaskWithPastStartTime 测试开始时间已过但未过期的任务
func TestAddTaskWithPastStartTime(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, nil, nil, newMockVideoFileService(), logger, cfg)

	// 启动调度器
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()

	// 创建开始时间已过但结束时间未到的任务
	startTime := time.Now().Add(-30 * time.Minute)
	task := createTestTask(1, "Immediate Task", startTime)
	// 任务的结束时间是 startTime + 1小时 = 当前时间 + 30分钟（未过期）
	setupTaskWithConfig(taskSvc, task)

	// 添加任务应该成功（立即执行）
	err = scheduler.AddTask(task)
	assert.NoError(t, err)
	// 立即执行的任务不在 cron 调度器中，所以 IsTaskScheduled 返回 false
	assert.False(t, scheduler.IsTaskScheduled(1))
}

// TestRemoveTask 测试移除任务
func TestRemoveTask(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, nil, nil, newMockVideoFileService(), logger, cfg)

	// 启动调度器
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()

	// 创建并添加任务
	startTime := time.Now().Add(1 * time.Hour)
	task := createTestTask(1, "Test Task", startTime)
	setupTaskWithConfig(taskSvc, task)

	err = scheduler.AddTask(task)
	assert.NoError(t, err)
	assert.True(t, scheduler.IsTaskScheduled(1))

	// 移除任务
	err = scheduler.RemoveTask(1)
	assert.NoError(t, err)
	assert.False(t, scheduler.IsTaskScheduled(1))
}

// TestGetScheduledTasks 测试获取已调度任务列表
func TestGetScheduledTasks(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, nil, nil, newMockVideoFileService(), logger, cfg)

	// 启动调度器
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()

	// 添加多个任务
	for i := 1; i <= 3; i++ {
		startTime := time.Now().Add(time.Duration(i) * time.Hour)
		task := createTestTask(uint(i), "Test Task", startTime)
		setupTaskWithConfig(taskSvc, task)
		err = scheduler.AddTask(task)
		assert.NoError(t, err)
	}

	// 获取已调度任务列表
	scheduledTasks := scheduler.GetScheduledTasks()
	assert.Len(t, scheduledTasks, 3)
}

// TestGetStats 测试获取统计信息
func TestGetStats(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, nil, nil, newMockVideoFileService(), logger, cfg)

	// 启动调度器
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()

	stats := scheduler.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "scheduled_tasks")
	assert.Contains(t, stats, "executing_tasks")
	assert.Contains(t, stats, "uptime")
}

// ErrTaskNotFound 任务未找到错误
var ErrTaskNotFound = fmt.Errorf("task not found")

// stubConversionService 验证 STYLE-003 修复：
// 一个空 stub 满足 ConversionService 接口，编译期即可断言接口契约。
// 真正的 *services.FFmpegConversionService 在 internal/services 包内通过
// 同名 _test 形式覆盖（避免 import cycle：services → scheduler → services）。
type stubConversionService struct{}

func (stubConversionService) SubmitConversion(context.Context, uint) error { return nil }
func (stubConversionService) GetConversionStatus(context.Context, uint) (models.ConversionStatus, error) {
	return "", nil
}
func (stubConversionService) RetryConversion(context.Context, uint) error { return nil }
func (stubConversionService) Start() error                                { return nil }
func (stubConversionService) Stop()                                       {}

// TestConversionService_InterfaceCompilationCheck 验证接口契约。
func TestConversionService_InterfaceCompilationCheck(t *testing.T) {
	var _ ConversionService = stubConversionService{}

	// 通过 reflect 验证接口方法数（防止误删方法）
	// 注意：reflect.TypeOf((*Interface)(nil)).NumMethod() == 0（ptr 化的 iface type
	// 是 *interface — 必须用 reflect.TypeOf((*Interface)(nil)).Elem()）。
	ifaceType := reflect.TypeOf((*ConversionService)(nil)).Elem()
	if ifaceType.NumMethod() != 5 {
		t.Errorf("ConversionService 方法数 = %d，期望 5", ifaceType.NumMethod())
	}
}
