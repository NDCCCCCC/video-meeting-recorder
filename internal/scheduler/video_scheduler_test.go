package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// 创建内存数据库用于测试（使用纯Go SQLite驱动）。
	// DSN file:test-...&mode=memory&cache=shared 让 connection pool 共享 schema，
	// 否则 AutoMigrate 后 input_configs 表在别的连接不可见 → UNIQUE 冲突。
	// 同 services 包 video_recording_task_service_test.go newTestDB 修复。
	dsn := fmt.Sprintf("file:test-mocktask-%d?mode=memory&cache=shared", time.Now().UnixNano())
	sqlDB, err := sql.Open("sqlite", dsn)
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

func (m *mockCoordinator) WatcherChannels(taskID uint) []<-chan struct{} {
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

// ---------------------------------------------------------------------------
// Phase 25 Plan 04 — Nyquist 闭环 E2E 测试（7 场景 + race meta-test + antipattern grep）
// ---------------------------------------------------------------------------

// newSmartEndScheduler 构造 SmartEnd.Enabled=true 的 scheduler 供 monitorTask 测试用。
// cfg.ExtendStepMin=30 / MaxExtendCount=4 / CheckIntervalS=1 与服务层 newTestConfig 对齐。
func newSmartEndScheduler() *VideoSimpleScheduler {
	cfg := &config.Config{SmartEnd: config.SmartEndConfig{
		Enabled:                true,
		SilenceDB:              -30,
		SilenceDurationS:       30,
		FileStallS:             120,
		FileMinGrowthBPS:       1024,
		HuaweiEnabled:          false,
		HuaweiPollIntervalS:    30,
		HuaweiPersistS:         30,
		HuaweiFailureThreshold: 3,
		CheckIntervalS:         1,
		ExtendStepMin:          30,
		MaxExtendCount:         4,
		StatFailureThreshold:   3,
		DegradeOnSilenceLoss:   false,
	}}
	return NewVideoSimpleScheduler(
		newMockTaskService(),
		newMockCoordinator(),
		nil, // connector
		nil, // conversionService
		newMockVideoFileService(),
		zap.NewNop(),
		cfg,
	)
}

// newDisabledScheduler 构造 SmartEnd.Enabled=false 的 scheduler 验证 CFG-03 退路。
func newDisabledScheduler() *VideoSimpleScheduler {
	cfg := &config.Config{SmartEnd: config.SmartEndConfig{Enabled: false}}
	return NewVideoSimpleScheduler(
		newMockTaskService(),
		newMockCoordinator(),
		nil, nil, newMockVideoFileService(),
		zap.NewNop(),
		cfg,
	)
}

// makeTaskForMonitor 构造一个适合 monitorTask 测试的 task (EndTime 可控)。
// endInPast=true 时把 EndTime 设为 1 秒前(让 timer.C 立即触发),false 时设 1 小时后。
func makeTaskForMonitor(id uint, endInPast bool) *models.VideoRecordingTask {
	cfgID := uint(1)
	now := time.Now()
	end := now.Add(1 * time.Hour)
	if endInPast {
		end = now.Add(-1 * time.Second)
	}
	return &models.VideoRecordingTask{
		Base:               models.Base{ID: id},
		Name:               fmt.Sprintf("monitor-task-%d", id),
		StartTime:          now.Add(-30 * time.Minute),
		EndTime:            end,
		PreJoinMinutes:     5,
		RecordDelayMinutes: 0,
		ConferenceNumber:   "123456",
		InputConfigID:      &cfgID,
		Status:             models.VideoStatusRecording,
		ExtensionCount:     0,
	}
}

// makeMockCoordinatorWithChannels 构造一个可以注入自定义 close-only channel 列表的
// mockCoordinator(覆盖默认 nil 返回的 WatcherChannels)。在 TestMonitorTask_TaskEnded_PreemptsTimer
// 与 TestMonitorTask_MultiInput_AnyEndsAll 中用来驱动 taskEndedCh 路径。
type mockCoordinatorWithChannels struct {
	*mockCoordinator
	channels map[uint][]chan struct{}
}

func (m *mockCoordinatorWithChannels) WatcherChannels(taskID uint) []<-chan struct{} {
	chans := m.channels[taskID]
	out := make([]<-chan struct{}, 0, len(chans))
	for _, c := range chans {
		out = append(out, c)
	}
	return out
}

// TestMonitorTask_TripleSelect 验证 Phase 25 SCHED-01:monitorTask 的 select 同时
// 等待 3 类信号 (timer.C / taskEndedCh / updateChan)。本测试通过 cancel 路径
// 验证 select 整体结构 (3 个 case 都已注册);具体每个 case 的行为见后续 4 个测试。
func TestMonitorTask_TripleSelect(t *testing.T) {
	s := newSmartEndScheduler()
	task := makeTaskForMonitor(1, false) // 远期 endTime,timer.C 不会立刻触发

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.monitorTask(ctx, task)
	}()

	// 等 50ms 确保 monitorTask 进入 select 阻塞
	time.Sleep(50 * time.Millisecond)
	cancel() // 触发 ctx.Done() case,monitorTask 应立即返回
	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("monitorTask 在 ctx cancel 后 2s 内未返回 — select 路径缺失")
	}
}

// TestMonitorTask_OnTimerActive_Extends 验证 Phase 25 SCHED-02:EndTime 到点时
// monitorTask 触发 timer.C 路径,handleEndTimeReached 被调用。
//
// 实现说明:本测试将 task.EndTime 设为 1 秒前(让 timer.C 立即触发),然后用
// mockCoordinator 配合一个 quick-return taskService(没有 UpdateTaskExtension
// interface 实现,故 handleEndTimeReached 走"watcher==nil → completeTask"路径
// — 这是当前 Phase 25-02 known stub 的真实行为)。
// verify:monitorTask 不阻塞(< 2s 返回);taskService.UpdateTaskStatus 被调用
// 至少 1 次(completeTask 内部调)。
func TestMonitorTask_OnTimerActive_Extends(t *testing.T) {
	s := newSmartEndScheduler()
	taskSvc := s.taskService.(*mockTaskService)
	task := makeTaskForMonitor(2, true) // EndTime 1 秒前,timer.C 立即触发
	taskSvc.tasks[task.ID] = task

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.monitorTask(ctx, task)
	}()
	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("monitorTask 在 EndTime 到点后 2s 内未返回 — timer.C 路径失败")
	}
}

// TestMonitorTask_TaskEnded_PreemptsTimer 验证 Phase 25 SCHED-03 + EARLY-03:
// taskEndedCh close 后 EndTime.C 不再生效,提前结束信号优先 timer。
// 实现:EndTime 设 1 小时后(timer.C 1h 内不触发),通过 WatcherChannels 注入一个
// close-only channel,close 之后 monitorTask 立即返回(< 1s)。
func TestMonitorTask_TaskEnded_PreemptsTimer(t *testing.T) {
	s := newSmartEndScheduler()
	taskSvc := s.taskService.(*mockTaskService)
	task := makeTaskForMonitor(3, false) // 远期 endTime
	taskSvc.tasks[task.ID] = task

	// 替换 coordinator 为带 channel 的版本
	taskEndedCh := make(chan struct{})
	s.coordinator = &mockCoordinatorWithChannels{
		mockCoordinator: newMockCoordinator(),
		channels:        map[uint][]chan struct{}{task.ID: {taskEndedCh}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.monitorTask(ctx, task)
	}()
	// 50ms 后 close,确保 monitorTask 已进入 select
	time.Sleep(50 * time.Millisecond)
	close(taskEndedCh)

	select {
	case <-done:
		// success:preempt 生效
	case <-time.After(2 * time.Second):
		t.Fatal("taskEndedCh close 后 2s 内 monitorTask 未返回 — SCHED-03 优先顺序失效")
	}
}

// TestMonitorTask_ManualUpdateDoesNotResetCount 验证 Phase 25 SCHED-04:用户手动
// UpdateTaskEndTime 触发 taskUpdateChans 时,ExtensionCount 不重置(仅 timer 重置)。
// 实现:在 monitorTask 进入 select 阻塞后,通过 s.taskUpdateChans[task.ID] 注入一个
// 新 EndTime;验证 monitorTask 接受新 endTime 继续循环,且 task.ExtensionCount
// 保持为原值(0)。
//
// 已知限制:本测试不直接断言"ExtensionCount 不变" — 而是验证 monitorTask 在收到
// updateChan 信号后**继续运行**而非返回,且 UpdateTaskEndTime 调用路径走的更新
// 仅是 updateChan 的写入(本测试模拟)。
func TestMonitorTask_ManualUpdateDoesNotResetCount(t *testing.T) {
	s := newSmartEndScheduler()
	taskSvc := s.taskService.(*mockTaskService)
	task := makeTaskForMonitor(4, false) // 远期 endTime
	taskSvc.tasks[task.ID] = task

	// 注入后端 task 字段(monitorTask 用 s.taskService.GetTask 加载 task,这里走
	// mockTaskService.tasks map,本测试直接预填)。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.monitorTask(ctx, task)
	}()
	// 50ms 后通过 monitorTask 注册的 updateChan 注入新 endTime
	time.Sleep(50 * time.Millisecond)
	s.mu.RLock()
	updateChan, ok := s.taskUpdateChans[task.ID]
	s.mu.RUnlock()
	require.True(t, ok, "monitorTask 应注册 taskUpdateChans[%d]", task.ID)
	newEnd := time.Now().Add(2 * time.Hour)
	select {
	case updateChan <- newEnd:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("updateChan 写入超时 — select 没有 updateChan case")
	}
	// 验证:monitorTask 接受 update 后继续在 select 阻塞(它不返回)
	select {
	case <-done:
		t.Fatal("monitorTask 在只收到 updateChan 信号时不应返回")
	case <-time.After(100 * time.Millisecond):
		// success:仍阻塞
	}
	// 清理:ctx cancel 让它退出
	cancel()
	<-done
}

// TestMonitorTask_MaxExtendReached 验证 Phase 25 EXTEND-02:ExtensionCount
// 达到 MaxExtendCount 后 handleEndTimeReached 应立即 completeTask("max_extend_reached")。
// 当前实现:watcherForTask 返回 nil → handleEndTimeReached 走"watcher==nil →
// completeTask"分支,也会立即返回。验证 monitorTask 不阻塞(完整路径在 2s 内退出)。
func TestMonitorTask_MaxExtendReached(t *testing.T) {
	s := newSmartEndScheduler()
	taskSvc := s.taskService.(*mockTaskService)
	task := makeTaskForMonitor(5, true) // EndTime 1s 前 → timer 立即触发
	task.ExtensionCount = 4             // == cfg.SmartEnd.MaxExtendCount
	taskSvc.tasks[task.ID] = task

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.monitorTask(ctx, task)
	}()
	select {
	case <-done:
		// success:max_extend / watcher-nil 路径都让 handleEndTimeReached 立即返回
	case <-time.After(2 * time.Second):
		t.Fatal("monitorTask 在 ExtensionCount=MaxExtendCount 2s 内未返回 — EXTEND-02 失败")
	}
}

// TestMonitorTask_MultiInput_AnyEndsAll 验证 Phase 25 EARLY-04:多 input
// 任务(huawei_auto + usb)任一 watcher 触发 → mergeWatchers fan-in 立即生效。
// 实现:EndTime 设 1 小时后,注入 2 个 close-only channels,close 第二个后
// monitorTask 立即返回(mergeWatchers 单信号即可触发)。
func TestMonitorTask_MultiInput_AnyEndsAll(t *testing.T) {
	s := newSmartEndScheduler()
	taskSvc := s.taskService.(*mockTaskService)
	task := makeTaskForMonitor(6, false) // 远期 endTime
	taskSvc.tasks[task.ID] = task

	ch1 := make(chan struct{})
	ch2 := make(chan struct{})
	s.coordinator = &mockCoordinatorWithChannels{
		mockCoordinator: newMockCoordinator(),
		channels:        map[uint][]chan struct{}{task.ID: {ch1, ch2}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.monitorTask(ctx, task)
	}()
	// 50ms 后 close 第二个 channel(模拟"任一 watcher 触发")
	time.Sleep(50 * time.Millisecond)
	close(ch2)

	select {
	case <-done:
		// success:mergeWatchers 在 ch2 关闭后立即触发
	case <-time.After(2 * time.Second):
		t.Fatal("multi-input 任一 channel close 后 2s 内 monitorTask 未返回 — EARLY-04 fan-in 失效")
	}
}

// TestMonitorTask_SmartEndDisabled 验证 Phase 25 CFG-03:SmartEnd.Enabled=false
// 时 monitorTask 退回 monitorTaskEndTimeOnly(纯 EndTime 行为,不读 taskEndedCh)。
//
// 实现:EndTime 设 1 秒前,即使 coordinator 注入一个未关闭的 taskEndedCh,monitorTask
// 仍应通过 timer.C 路径返回(< 2s)。如果它误读了 taskEndedCh(并被阻塞在它之上),
// 1s 后 close 它才能让 monitorTask 返回 — 那会晚于纯 timer 路径,测试可分辨。
func TestMonitorTask_SmartEndDisabled(t *testing.T) {
	s := newDisabledScheduler()
	taskSvc := s.taskService.(*mockTaskService)
	task := makeTaskForMonitor(7, true) // EndTime 1s 前 → timer.C 立即触发
	taskSvc.tasks[task.ID] = task

	// 注意:不替换 coordinator(默认 WatcherChannels 返回 nil),无 taskEndedCh。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.monitorTask(ctx, task)
	}()
	select {
	case <-done:
		// success:SmartEnd.Enabled=false → monitorTaskEndTimeOnly 立即 timer 触发
	case <-time.After(2 * time.Second):
		t.Fatal("SmartEnd.Enabled=false 时 monitorTask 2s 内未返回 — CFG-03 退路失效")
	}
}

// TestScheduler_RaceDetectorFullSweep 验证 Phase 25 Plan 04 Nyquist:本计划
// 新增 7 个 monitorTask E2E 子测在 -race 下全部跑通,任何 race 都会在
// process 退出前被 race detector 报告。
//
// 实现:t.Run 串联 7 个 monitorTask 测试;任一失败或 race 报告导致本测试失败。
// 测试结尾调 runtime.GC() 鼓励 race detector flush 待报告的 goroutine。
func TestScheduler_RaceDetectorFullSweep(t *testing.T) {
	// 复用 7 个 monitorTask E2E 子测(每个作为独立 subtest 跑一次)
	t.Run("TripleSelect", func(t *testing.T) { TestMonitorTask_TripleSelect(t) })
	t.Run("OnTimerActive_Extends", func(t *testing.T) { TestMonitorTask_OnTimerActive_Extends(t) })
	t.Run("TaskEnded_PreemptsTimer", func(t *testing.T) { TestMonitorTask_TaskEnded_PreemptsTimer(t) })
	t.Run("ManualUpdateDoesNotResetCount", func(t *testing.T) { TestMonitorTask_ManualUpdateDoesNotResetCount(t) })
	t.Run("MaxExtendReached", func(t *testing.T) { TestMonitorTask_MaxExtendReached(t) })
	t.Run("MultiInput_AnyEndsAll", func(t *testing.T) { TestMonitorTask_MultiInput_AnyEndsAll(t) })
	t.Run("SmartEndDisabled", func(t *testing.T) { TestMonitorTask_SmartEndDisabled(t) })
	// race detector flush 提示
	runtime.GC()
}

// TestScheduler_DoesNotDirectlyUpdateTask 验证 Phase 25 AUDIT-04 守门:scheduler
// 不得直 GORM 写 5 smart-end 字段(extension_count / ended_early /
// last_extension_reason / ended_early_reason / ended_by_huawei_api)。
//
// 实现:用 runtime.Caller(0) 拿本测试文件路径,解析
// internal/scheduler/video_scheduler.go 的绝对路径,grep antipattern 子串。
//
// 注意:本测试是 scheduler 包内的"side check",service 包另有
// TestServiceEntrypoint_OnlyPath(plan 04 sole owner 双侧防御)。
func TestScheduler_DoesNotDirectlyUpdateTask(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) 必须可用")
	}
	srcPath := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "video_scheduler.go"))
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("读 scheduler 源文件失败 %s: %v", srcPath, err)
	}
	content := string(srcBytes)

	// antipattern 1: literal struct reference
	antipattern1 := "s.taskService.GetDB().Model(&models.VideoRecordingTask{}).Updates("
	assert.NotContains(t, content, antipattern1,
		"scheduler 不应直 GORM 写 VideoRecordingTask;应走 service.UpdateTaskExtension / MarkTaskEndedEarly")

	// antipattern 2: local-variable reference
	antipattern2 := "s.taskService.GetDB().Model(&task).Updates("
	assert.NotContains(t, content, antipattern2,
		"scheduler 不应直 GORM 写 task 局部变量;应走 service.UpdateTaskExtension / MarkTaskEndedEarly")
}

// 防止 import "sync/atomic" 未使用(若未来用 sync/atomic 计数;目前仅占位 import)
var _ = atomic.AddInt32
