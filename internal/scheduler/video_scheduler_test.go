package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockVideoFileService 模拟视频文件服务
type mockVideoFileService struct{}

func newMockVideoFileService() *mockVideoFileService {
	return &mockVideoFileService{}
}

func (m *mockVideoFileService) CreateFileFromTask(task *models.VideoRecordingTask, format *string) (*models.VideoFile, error) {
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
}

func newMockTaskService() *mockTaskService {
	return &mockTaskService{
		tasks: make(map[uint]*models.VideoRecordingTask),
	}
}

func (m *mockTaskService) GetTask(id uint) (*models.VideoRecordingTask, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (m *mockTaskService) GetPendingTasks() ([]*models.VideoRecordingTask, error) {
	var result []*models.VideoRecordingTask
	for _, task := range m.tasks {
		if task.Status == models.VideoStatusPending {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockTaskService) UpdateTaskStatus(id uint, status models.VideoRecordingTaskStatus, errorMsg string) error {
	task, ok := m.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	task.Status = status
	task.ErrorMsg = errorMsg
	return nil
}

func (m *mockTaskService) GetHuaweiConfig(id uint) (*models.HuaweiConfig, error) {
	return &models.HuaweiConfig{
		Base:       models.Base{ID: id},
		Name:       "Test Config",
		Server:     "192.168.1.100",
		Port:       80,
		Username:   "admin",
		Password:   "password",
		OutputFormat: "mp4",
		CameraBackend: "dshow",
		AudioBackend:  "dshow",
	}, nil
}

func (m *mockTaskService) UpdateRecordingPaths(id uint, mkvPath, hlsPath string) error {
	task, ok := m.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	task.MKVFilePath = mkvPath
	task.HLSPreviewPath = hlsPath
	return nil
}

// mockCoordinator 模拟录制协调器
type mockCoordinator struct{}

func newMockCoordinator() *mockCoordinator {
	return &mockCoordinator{}
}

func (m *mockCoordinator) StartRecording(task *models.VideoRecordingTask, config *models.HuaweiConfig) error {
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
	return &models.VideoRecordingTask{
		Base:             models.Base{ID: id},
		Name:              name,
		StartTime:         startTime,
		EndTime:           startTime.Add(1 * time.Hour),
		PreJoinMinutes:    5,
		RecordDelayMinutes: 0,
		ConferenceNumber:  "123456",
		HuaweiConfigID:    1,
		Status:            models.VideoStatusPending,
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
	taskSvc.tasks[1] = task

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
	taskSvc.tasks[1] = task

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
	taskSvc.tasks[1] = task

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
	taskSvc.tasks[1] = task

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
		taskSvc.tasks[uint(i)] = task
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
