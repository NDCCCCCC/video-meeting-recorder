package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

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

	scheduler := NewVideoSimpleScheduler(taskSvc, coord, logger, cfg)

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
	scheduler := NewVideoSimpleScheduler(newMockTaskService(), newMockCoordinator(), logger, cfg)

	startTime := time.Date(2026, 2, 10, 14, 30, 0, 0, time.UTC)
	triggerTime := scheduler.calculateTriggerTime(startTime, 5)

	expected := time.Date(2026, 2, 10, 14, 25, 0, 0, time.UTC)
	assert.Equal(t, expected, triggerTime)
}

// TestGenerateCronExpression 测试Cron表达式生成
func TestGenerateCronExpression(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	scheduler := NewVideoSimpleScheduler(newMockTaskService(), newMockCoordinator(), logger, cfg)

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
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, logger, cfg)

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
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, logger, cfg)

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
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, logger, cfg)

	// 启动调度器
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()

	// 创建已过期的任务
	startTime := time.Now().Add(-2 * time.Hour)
	task := createTestTask(1, "Expired Task", startTime)
	taskSvc.tasks[1] = task

	// 添加过期任务
	err = scheduler.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "触发时间已过期")
}

// TestRemoveTask 测试移除任务
func TestRemoveTask(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	taskSvc := newMockTaskService()
	coord := newMockCoordinator()
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, logger, cfg)

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
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, logger, cfg)

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
	scheduler := NewVideoSimpleScheduler(taskSvc, coord, logger, cfg)

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
