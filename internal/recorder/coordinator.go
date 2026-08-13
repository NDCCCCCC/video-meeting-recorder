package recorder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// SimpleRecordingCoordinator 简单的录制协调器
type SimpleRecordingCoordinator struct {
	logger    *zap.Logger
	config    *config.Config
	processes map[string]*RecordingProcess // 使用字符串键支持多配置 (taskID_configType)
	mu        sync.RWMutex
	// taskIDProcessKeyIndex taskID -> 当前活跃的过程 key (per-task 只允许 1 个
	// in-flight 录制;IsProcessAlive(taskID) 通过该索引定位 cmd 读 ProcessState)。
	taskIDProcessKeyIndex map[uint]string
}

// RecordingProcess 录制进程
type RecordingProcess struct {
	TaskID     uint
	Cmd        *exec.Cmd // 主录制进程 (使用 tee muxer 同时输出 MKV 和 HLS)
	StartTime  time.Time
	OutputPath string // MKV文件路径
	HLSPath    string // HLS预览路径 (m3u8文件)
	Status     string
	CancelFunc context.CancelFunc
	logFile    *os.File
	// 自动重连支持
	ConfigType      string                     // 配置类型: usb 或 stream
	Task            *models.VideoRecordingTask // 任务信息（用于重连）
	HuaweiConfig    *models.InputConfig        // 华为配置（用于重连）
	ReconnectCount  atomic.Int32               // 当前重连次数（PERF-010：原子读，监控 goroutine 增 1 不会与 reconnect 比对产生 race）
	MaxReconnects   int                        // 最大重连次数
	ReconnectDelay  time.Duration              // 重连间隔
	ShouldReconnect bool                       // 是否应该重连（仅对流媒体有效）
}

// InputSourceType 输入源类型
type InputSourceType string

const (
	InputSourceUSB    InputSourceType = "usb"    // USB设备
	InputSourceRTSP   InputSourceType = "rtsp"   // RTSP流
	InputSourceRTMP   InputSourceType = "rtmp"   // RTMP流
	InputSourceStream InputSourceType = "stream" // 通用流媒体
	InputSourceMixed  InputSourceType = "mixed"  // 混合输入
)

// RecordingInput 录制输入配置
type RecordingInput struct {
	Type           InputSourceType `json:"type"`
	RTSPURL        string          `json:"rtsp_url,omitempty"`
	StreamProtocol string          `json:"stream_protocol,omitempty"` // rtmp, rtsp, srt, hls
	StreamURL      string          `json:"stream_url,omitempty"`
	StreamUsername string          `json:"stream_username,omitempty"`
	StreamPassword string          `json:"stream_password,omitempty"`
	CameraBackend  string          `json:"camera_backend"`
	AudioBackend   string          `json:"audio_backend"`
	CameraDevice   string          `json:"camera_device,omitempty"`
	AudioDevice    string          `json:"audio_device,omitempty"`
	CameraName     string          `json:"camera_name,omitempty"` // 实际设备名称
	AudioName      string          `json:"audio_name,omitempty"`  // 实际设备名称
	hasAudio       bool            // 内部标记是否有音频
}

// NewSimpleRecordingCoordinator 创建录制协调器
func NewSimpleRecordingCoordinator(logger *zap.Logger, cfg *config.Config) *SimpleRecordingCoordinator {
	return &SimpleRecordingCoordinator{
		logger:                 logger,
		config:                 cfg,
		processes:              make(map[string]*RecordingProcess),
		taskIDProcessKeyIndex:  make(map[uint]string),
	}
}

// IsProcessAlive 检查 taskID 对应的 ffmpeg 进程是否仍在运行。
//
// 返回 true 表示 ffmpeg 进程处于活跃状态 (cmd.ProcessState == nil),
// scheduler 据此在 endtime 到达时决定:
//   - true  → 走 UpdateTaskExtension 延长会议
//   - false → 走 completeTask 走收尾转码
//
// taskID 找不到活跃进程时返回 false (scheduler 把它当 ffmpeg 已退出处理)。
func (c *SimpleRecordingCoordinator) IsProcessAlive(taskID uint) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	processKey, ok := c.taskIDProcessKeyIndex[taskID]
	if !ok {
		return false
	}
	proc, ok := c.processes[processKey]
	if !ok || proc == nil || proc.Cmd == nil {
		return false
	}
	return proc.Cmd.ProcessState == nil
}

// StartRecording 启动录制
func (c *SimpleRecordingCoordinator) StartRecording(task *models.VideoRecordingTask, huaweiConfig *models.InputConfig) error {
	// 默认使用 USB 类型
	return c.StartRecordingWithConfig(task, huaweiConfig, "usb")
}

// StartRecordingWithConfig 启动指定配置类型的录制
func (c *SimpleRecordingCoordinator) StartRecordingWithConfig(task *models.VideoRecordingTask, huaweiConfig *models.InputConfig, configType string) error {
	// PERF-004: 把昂贵的资源创建（dir + ffmpeg 进程启动）移出锁外，仅保留最小临界区。

	// 根据配置类型生成输出路径
	mkvPath := c.getOutputPathWithType(task, configType, "mkv")
	if err := os.MkdirAll(filepath.Dir(mkvPath), 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w: %w", apperrors.ErrInternal, err)
	}

	// 生成 HLS 输出路径
	hlsPath := c.getHLSPathWithType(task, configType)
	if err := os.MkdirAll(hlsPath, 0o755); err != nil {
		return fmt.Errorf("创建HLS目录失败: %w: %w", apperrors.ErrInternal, err)
	}

	input := c.buildRecordingInput(task, huaweiConfig)

	// 使用 tee muxer 同时生成 MKV 和 HLS
	args, err := c.buildRecordingCommand(input, mkvPath, hlsPath, task.EndTime.Sub(task.StartTime), task.ID)
	if err != nil {
		return fmt.Errorf("构建录制命令失败: %w: %w", apperrors.ErrInternal, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, c.config.FFmpeg.Path, args...)

	logFile, err := c.startFFmpegProcess(cmd, mkvPath)
	if err != nil {
		cancel()
		return err
	}

	// HLS m3u8 文件路径
	m3u8Path := filepath.Join(hlsPath, "index.m3u8")

	// 使用带配置类型的键存储进程
	processKey := c.getProcessKey(task.ID, configType)

	// 判断是否为流媒体录制（仅对流媒体启用自动重连）
	isStreamRecording := configType == "stream" && input.Type != InputSourceUSB
	shouldReconnect := isStreamRecording

	rec := &RecordingProcess{
		TaskID:          task.ID,
		Cmd:             cmd,
		StartTime:       time.Now(),
		OutputPath:      mkvPath,
		HLSPath:         m3u8Path,
		Status:          "running",
		CancelFunc:      cancel,
		logFile:         logFile,
		ConfigType:      configType,
		Task:            task,
		HuaweiConfig:    huaweiConfig,
		MaxReconnects:   3,                // 最多重连3次
		ReconnectDelay:  10 * time.Second, // 重连间隔10秒
		ShouldReconnect: shouldReconnect,
	}

	// 最小临界区：仅做 map 注册（PERF-004）
	c.mu.Lock()
	c.processes[processKey] = rec
	if c.taskIDProcessKeyIndex == nil {
		c.taskIDProcessKeyIndex = make(map[uint]string)
	}
	c.taskIDProcessKeyIndex[task.ID] = processKey
	c.mu.Unlock()

	// 注: Phase 25 智能退出撤回后,ActivityWatcher 整文件删除,不再构造 watcher。
	// scheduler 改为调 IsProcessAlive(taskID) 直接查 ffmpeg cmd.ProcessState。

	// 总是用本次启动的 mkvPath 刷新 task.MKVFilePath / RecordingFile / HLSPreviewPath,
	// 无论 configType 是 usb / huawei_auto / stream。Phase 18 (bb3dc93e) 引入的条件
	// `configType == "usb" || task.MKVFilePath == ""` 不覆盖 stream/huawei_auto,
	// 导致流式录制完成后 completeTask 写出的 MKVFilePath 时间戳沿用上次残留(用户
	// 实测 115759 → 本次启动 120108),web 端报"提交转换任务失败"(MKV 不存在)。
	// 详见 .planning/debug/huawei-auto-smart-end.md Bug C。
	if configType == "usb" || configType == "huawei_auto" || configType == "stream" || task.MKVFilePath == "" {
		task.RecordingFile = mkvPath
		task.MKVFilePath = mkvPath
		task.HLSPreviewPath = m3u8Path
	}

	c.logger.Info("录制已启动（同时生成 MKV 和 HLS）",
		zap.Uint("task_id", task.ID),
		zap.String("config_type", configType),
		zap.String("input_type", string(input.Type)),
		zap.String("mkv_path", mkvPath),
		zap.String("hls_path", m3u8Path),
	)

	// 启动监控 goroutine，等待进程结束
	go c.monitorProcessWithKey(processKey, cmd, ctx)

	return nil
}

// getProcessKey 获取进程键
func (c *SimpleRecordingCoordinator) getProcessKey(taskID uint, configType string) string {
	return fmt.Sprintf("%d_%s", taskID, configType)
}

// getOutputPathWithType 生成带配置类型的输出文件路径
func (c *SimpleRecordingCoordinator) getOutputPathWithType(task *models.VideoRecordingTask, configType, format string) string {
	safeName := sanitizeFilename(task.Name)
	conferenceNumber := task.ConferenceNumber
	timestamp := time.Now().Format("20060102150405")

	// 流媒体类型添加 stream 后缀
	suffix := ""
	if configType == "stream" {
		suffix = "_stream"
	}

	filename := fmt.Sprintf("%s_%s_%s%s.%s", safeName, conferenceNumber, timestamp, suffix, format)
	outputDir := filepath.Join(c.config.Storage.RecordingsPath, fmt.Sprintf("task_%d", task.ID))
	return filepath.Join(outputDir, filename)
}

// getHLSPathWithType 生成带配置类型的 HLS 输出目录路径
func (c *SimpleRecordingCoordinator) getHLSPathWithType(task *models.VideoRecordingTask, configType string) string {
	safeName := sanitizeFilename(task.Name)
	conferenceNumber := task.ConferenceNumber
	timestamp := time.Now().Format("20060102150405")

	// 流媒体类型添加 stream 后缀
	suffix := ""
	if configType == "stream" {
		suffix = "_stream"
	}

	dirname := fmt.Sprintf("%s_%s_%s%s", safeName, conferenceNumber, timestamp, suffix)
	hlsDir := filepath.Join(c.config.Storage.HLSPath, fmt.Sprintf("task_%d", task.ID), dirname)
	return hlsDir
}

// monitorProcessWithKey 监控录制进程状态（带配置类型键）
// 支持自动重连：当流媒体录制异常退出时，自动尝试重新连接
func (c *SimpleRecordingCoordinator) monitorProcessWithKey(processKey string, cmd *exec.Cmd, ctx context.Context) {
	// 先等待进程结束，不持有锁
	err := cmd.Wait()

	// 获取进程信息;PERF-010：把后续用到的字段在锁内复制到本地变量，
	// 避免解锁后另一个 goroutine 把 entry 删除导致 pointer dereference race。
	c.mu.Lock()
	process, exists := c.processes[processKey]
	var (
		shouldReconnect bool
		taskCopy        *models.VideoRecordingTask
		huaweiCopy      *models.InputConfig
	)
	if exists && process != nil {
		process.Status = "stopped"
		shouldReconnect = process.ShouldReconnect
		taskCopy = process.Task
		huaweiCopy = process.HuaweiConfig
	}
	c.mu.Unlock()

	if err != nil {
		if ctx.Err() != nil {
			// Context 被取消，是主动停止
			c.logger.Debug("录制进程已停止", zap.String("process_key", processKey))
			return
		}

		// 非预期退出，检查是否需要重连
		c.logger.Error("录制进程异常退出",
			zap.String("process_key", processKey),
			zap.Error(err),
		)

		// 尝试自动重连（仅对启用了重连的流媒体录制）。
		// 使用本地变量（已 lock-safe 复制），即使 c.processes[processKey]
		// 被另一个 goroutine 删除，也不会触发 race detector。
		if process != nil && shouldReconnect && taskCopy != nil && huaweiCopy != nil {
			// 用重新查到的 process 再传给 attemptReconnect,避免 process map 已被删除。
			// 这里我们传入 process（已上锁的指针）即可, attemptReconnect 只读
			// 标志位/计数/路径字段,这些都是 lock-free(atomic) 或局部拷贝。
			c.attemptReconnect(ctx, processKey, process)
		}
	} else {
		c.logger.Info("录制进程正常结束", zap.String("process_key", processKey))
	}
}

// attemptReconnect 尝试重新连接流媒体
func (c *SimpleRecordingCoordinator) attemptReconnect(ctx context.Context, processKey string, process *RecordingProcess) {
	// PERF-010：原子读 ReconnectCount（避免与监控 goroutine 的状态切换竞争）
	currentCount := process.ReconnectCount.Load()
	// 检查是否超过最大重连次数
	if currentCount >= int32(process.MaxReconnects) {
		c.logger.Error("已达到最大重连次数，停止重连",
			zap.String("process_key", processKey),
			zap.Int("max_reconnects", process.MaxReconnects),
		)
		return
	}

	// 等待重连间隔
	c.logger.Info("等待重连间隔...",
		zap.String("process_key", processKey),
		zap.Duration("delay", process.ReconnectDelay),
		zap.Int32("attempt", currentCount+1),
	)
	timer := time.NewTimer(process.ReconnectDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	// 增加重连计数（PERF-010：原子 +1）
	process.ReconnectCount.Add(1)
	newCount := process.ReconnectCount.Load()

	c.logger.Info("尝试重新连接流媒体",
		zap.String("process_key", processKey),
		zap.Int("attempt", int(newCount)),
		zap.Int("max_attempts", process.MaxReconnects),
	)

	// 重新启动录制
	if err := c.restartRecording(processKey, process); err != nil {
		c.logger.Error("重连失败",
			zap.String("process_key", processKey),
			zap.Error(err),
		)
		// 递归尝试下一次重连
		if process.ReconnectCount.Load() < int32(process.MaxReconnects) {
			go c.attemptReconnect(ctx, processKey, process)
		}
	} else {
		c.logger.Info("重连成功，录制已恢复",
			zap.String("process_key", processKey),
		)
	}
}

// restartRecording 重新启动录制
func (c *SimpleRecordingCoordinator) restartRecording(processKey string, process *RecordingProcess) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 获取配置类型
	configType := process.ConfigType
	task := process.Task
	huaweiConfig := process.HuaweiConfig

	// 生成新的输出路径（使用新的时间戳）
	mkvPath := c.getOutputPathWithType(task, configType, "mkv")
	if err := os.MkdirAll(filepath.Dir(mkvPath), 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w: %w", apperrors.ErrInternal, err)
	}

	hlsPath := c.getHLSPathWithType(task, configType)
	if err := os.MkdirAll(hlsPath, 0o755); err != nil {
		return fmt.Errorf("创建HLS目录失败: %w: %w", apperrors.ErrInternal, err)
	}

	input := c.buildRecordingInput(task, huaweiConfig)

	// 计算剩余录制时长
	remainingDuration := time.Until(task.EndTime)
	if remainingDuration <= 0 {
		return fmt.Errorf("任务已结束，无法继续录制: %w", apperrors.ErrTaskNotFound)
	}

	// 使用 tee muxer 同时生成 MKV 和 HLS
	args, err := c.buildRecordingCommand(input, mkvPath, hlsPath, remainingDuration, task.ID)
	if err != nil {
		return fmt.Errorf("构建录制命令失败: %w: %w", apperrors.ErrInternal, err)
	}

	// 注: Phase 25 智能退出撤回后,不再有 ActivityWatcher 需要 Stop / OnReconnect。
	// 旧代码(OnReconnect / ActivityWatcher.Stop)随 activity_watcher.go 整文件删除。

	if process.logFile != nil {
		_ = process.logFile.Close()
		process.logFile = nil
	}

	// 创建新的录制进程
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, c.config.FFmpeg.Path, args...)

	logFile, err := c.startFFmpegProcess(cmd, mkvPath)
	if err != nil {
		cancel()
		return err
	}

	// HLS m3u8 文件路径
	m3u8Path := filepath.Join(hlsPath, "index.m3u8")

	// 更新进程信息
	process.Cmd = cmd
	process.StartTime = time.Now()
	process.OutputPath = mkvPath
	process.HLSPath = m3u8Path
	process.Status = "running"
	process.CancelFunc = cancel
	process.logFile = logFile

	// 注: 不再重建 ActivityWatcher(IsProcessAlive 直接查 cmd.ProcessState)。

	c.logger.Info("重连录制已启动",
		zap.String("process_key", processKey),
		zap.String("mkv_path", mkvPath),
		zap.String("hls_path", m3u8Path),
	)

	// 重新启动监控 goroutine
	go c.monitorProcessWithKey(processKey, cmd, ctx)

	return nil
}

// getOutputPath 生成输出文件路径
// 格式: {任务名称}_{会议号}_{时间戳}.mkv
func (c *SimpleRecordingCoordinator) getOutputPath(task *models.VideoRecordingTask, format string) string {
	// 清理任务名称中的特殊字符，用于文件名
	safeName := sanitizeFilename(task.Name)
	conferenceNumber := task.ConferenceNumber
	timestamp := time.Now().Format("20060102150405")

	filename := fmt.Sprintf("%s_%s_%s.%s", safeName, conferenceNumber, timestamp, format)
	outputDir := filepath.Join(c.config.Storage.RecordingsPath, fmt.Sprintf("task_%d", task.ID))
	return filepath.Join(outputDir, filename)
}

// sanitizeFilename 清理文件名中的特殊字符
func sanitizeFilename(name string) string {
	replacements := map[rune]string{
		' ': "_", '/': "_", '\\': "_", ':': "_",
		'*': "_", '?': "_", '"': "_", '<': "_",
		'>': "_", '|': "_",
		'\n': "", '\r': "", '\t': "",
	}

	result := make([]rune, 0, len(name))
	for _, r := range name {
		if repl, ok := replacements[r]; ok {
			result = append(result, []rune(repl)...)
		} else if r >= 32 {
			result = append(result, r)
		}
	}

	if len(result) > 100 {
		result = result[:100]
	}
	return string(result)
}

// startFFmpegProcess 启动FFmpeg进程并配置日志
func (c *SimpleRecordingCoordinator) startFFmpegProcess(cmd *exec.Cmd, outputPath string) (*os.File, error) {
	logPath := filepath.Join(filepath.Dir(outputPath), "ffmpeg.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %w: %w", apperrors.ErrInternal, err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// 打印完整的 FFmpeg 命令行用于调试
	// PERF-012: 用 strings.Builder 替换循环字符串拼接（避免重复分配）
	// Args[0] 是程序名，Args[1:] 是参数
	var commandLine strings.Builder
	commandLine.Grow(len(cmd.Path) + len(cmd.Args)*16)
	commandLine.WriteString(cmd.Path)
	if len(cmd.Args) > 0 {
		for _, arg := range cmd.Args[1:] {
			if strings.Contains(arg, " ") || strings.Contains(arg, "\t") {
				commandLine.WriteString(" \"")
				commandLine.WriteString(arg)
				commandLine.WriteString("\"")
			} else {
				commandLine.WriteString(" ")
				commandLine.WriteString(arg)
			}
		}
	}
	c.logger.Info("FFmpeg 命令行（可手动运行测试）",
		zap.String("command", commandLine.String()),
		zap.String("log_file", logPath),
	)

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("启动FFmpeg进程失败: %w: %w", apperrors.ErrFFmpegFailed, err)
	}

	return logFile, nil
}

// buildRecordingInput 构建录制输入配置
func (c *SimpleRecordingCoordinator) buildRecordingInput(task *models.VideoRecordingTask, huaweiConfig *models.InputConfig) RecordingInput {
	input := RecordingInput{
		CameraBackend: huaweiConfig.CameraBackend,
		AudioBackend:  huaweiConfig.AudioBackend,
		CameraDevice:  huaweiConfig.USBCameraDevice,
		AudioDevice:   huaweiConfig.USBAudioDevice,
		CameraName:    huaweiConfig.USBCameraName,
		AudioName:     huaweiConfig.USBAudioName,
	}

	// 调试日志：显示设备配置
	c.logger.Info("录制设备配置",
		zap.Uint("task_id", task.ID),
		zap.String("camera_backend", input.CameraBackend),
		zap.String("camera_name", input.CameraName),
		zap.String("camera_device", input.CameraDevice),
		zap.String("audio_backend", input.AudioBackend),
		zap.String("audio_name", input.AudioName),
		zap.String("audio_device", input.AudioDevice),
	)

	// 检查任务配置的RTSP流（向后兼容）
	if task.RTSPStreamURL != "" {
		input.Type = InputSourceRTSP
		input.RTSPURL = task.RTSPStreamURL
		input.hasAudio = true
	}

	// 检查华为配置的流媒体设置
	hasStream := huaweiConfig.StreamEnabled && huaweiConfig.StreamURL != ""
	if hasStream {
		input.StreamProtocol = huaweiConfig.StreamProtocol
		input.StreamURL = huaweiConfig.StreamURL
		input.StreamUsername = huaweiConfig.StreamUsername
		input.StreamPassword = huaweiConfig.StreamPassword

		switch huaweiConfig.StreamProtocol {
		case "rtmp":
			if input.Type == InputSourceRTSP {
				input.Type = InputSourceMixed
			} else {
				input.Type = InputSourceRTMP
			}
		default: // rtsp, srt, hls
			if input.Type == InputSourceRTSP || input.Type == InputSourceRTMP {
				input.Type = InputSourceMixed
			} else {
				input.Type = InputSourceStream
			}
		}
		input.hasAudio = true
	}

	// 检查USB设备配置
	hasUSB := huaweiConfig.USBCameraDevice != "" || huaweiConfig.USBAudioDevice != ""
	if hasUSB {
		input.hasAudio = input.hasAudio || huaweiConfig.USBAudioDevice != ""
		if input.Type == "" {
			input.Type = InputSourceUSB
		} else {
			input.Type = InputSourceMixed
		}
	}

	return input
}

// StopRecording 停止录制
func (c *SimpleRecordingCoordinator) StopRecording(taskID uint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 停止所有相关配置的录制进程
	// 查找所有以 taskID_ 开头的进程键
	stoppedCount := 0
	for key := range c.processes {
		if strings.HasPrefix(key, fmt.Sprintf("%d_", taskID)) {
			process := c.processes[key]
			if process != nil && process.Status == "running" {
				// 注: Phase 25 智能退出撤回后,不再有 ActivityWatcher 需要 Stop。
				process.CancelFunc()
				if process.logFile != nil {
					_ = process.logFile.Close()
				}
				stoppedCount++
			}
		}
	}

	// 也检查旧的单键格式（向后兼容）
	if process, ok := c.processes[fmt.Sprintf("%d", taskID)]; ok && process.Status == "running" {
		process.CancelFunc()
		if process.logFile != nil {
			_ = process.logFile.Close()
		}
		stoppedCount++
	}

	c.logger.Info("录制已停止", zap.Uint("task_id", taskID), zap.Int("stopped_count", stoppedCount))
	return nil
}

// buildRecordingCommand 构建录制命令（使用 tee muxer 同时输出 MKV 和 HLS）
func (c *SimpleRecordingCoordinator) buildRecordingCommand(input RecordingInput, mkvPath, hlsPath string, duration time.Duration, taskID uint) ([]string, error) {
	args := []string{"-y"}

	// 添加输入源
	inputArgs, err := c.buildInputArgs(input)
	if err != nil {
		return nil, err
	}
	args = append(args, inputArgs...)

	// 添加视频编码参数（使用CRF质量模式）
	args = append(args,
		"-c:v", "libx264",
		"-preset", c.config.FFmpeg.Preset,
		"-crf", fmt.Sprintf("%d", c.config.FFmpeg.CRF),
		"-maxrate", c.config.FFmpeg.MaxVideoBitrate,
		"-bufsize", c.config.FFmpeg.VideoBufSize,
		"-pix_fmt", "yuv420p",
	)

	// 添加音频编码参数（如果有音频）
	if input.hasAudio {
		args = append(args, "-c:a", "aac", "-b:a", c.config.FFmpeg.DefaultAudioBitrate+"k")
	}

	// 添加时长限制
	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.0f", duration.Seconds()))
	}

	// 添加 global_header 标志，这是 tee muxer 正常工作的关键
	// MKV 和 HLS 容器需要全局存储的比特流参数
	args = append(args, "-flags", "+global_header")

	// 注: Phase 25 智能退出撤回后,不再注入 silencedetect 过滤器 (A 信号源随
	// activity_watcher.go 整文件删除)。

	// 映射流：需要根据输入源类型确定正确的输入索引
	args = append(args, "-map", "0:v")
	if input.hasAudio {
		// 检查是否有独立的音频输入源（USB或Mixed类型）
		if input.Type == InputSourceUSB || input.Type == InputSourceMixed {
			// 分离的音频设备，使用第二个输入索引
			args = append(args, "-map", "1:a")
		} else {
			// RTSP等单一输入源，使用第一个输入索引
			args = append(args, "-map", "0:a")
		}
	}

	// 构建 tee muxer 输出规范
	hlsSegmentPath := filepath.Join(hlsPath, "segment_%03d.ts")
	hlsM3U8Path := filepath.Join(hlsPath, "index.m3u8")

	// 从配置读取 HLS 参数
	hlsSegmentDuration := c.config.FFmpeg.HLSSegmentDuration
	hlsListSize := c.config.FFmpeg.HLSListSize
	// PERF-011：HLSListSize 缺配置校验。
	// 配置为 0（未设置） → fallback 6；负数 → fallback 6 并记录警告。
	// 避免未校验的 hlsDeleteThreshold (= hlsListSize + 1) 传给 FFmpeg 导致未定义行为。
	hlsDeleteThreshold := hlsListSize + 1
	if hlsListSize <= 0 {
		c.logger.Warn("无效的 HLSListSize，使用默认值 6",
			zap.Int("got", hlsListSize))
		hlsListSize = 6
		hlsDeleteThreshold = hlsListSize + 1
	}

	// 使用相对路径，避免 Windows 盘符转义问题
	// 配置中的路径本身就是相对路径（如 ./data/recordings），直接使用即可
	// 标准化路径：将反斜杠转换为正斜杠，去除开头的 ./
	normalizePath := func(p string) string {
		// 转换为正斜杠（FFmpeg 需要）
		normalized := filepath.ToSlash(p)
		// 去除开头的 ./
		for strings.HasPrefix(normalized, "./") {
			normalized = normalized[2:]
		}
		// 去除开头的 /（如果转换后产生）
		normalized = strings.TrimPrefix(normalized, "/")
		return normalized
	}

	mkvRelPath := normalizePath(mkvPath)
	hlsSegmentRelPath := normalizePath(hlsSegmentPath)
	hlsM3U8RelPath := normalizePath(hlsM3U8Path)

	// 简单转义：空格、单引号、方括号
	escapeSimple := func(p string) string {
		p = strings.ReplaceAll(p, " ", "\\ ")
		p = strings.ReplaceAll(p, "'", "\\'")
		p = strings.ReplaceAll(p, "[", "\\[")
		p = strings.ReplaceAll(p, "]", "\\]")
		return p
	}

	mkvPathEscaped := escapeSimple(mkvRelPath)
	hlsSegmentPathEscaped := escapeSimple(hlsSegmentRelPath)
	hlsM3U8PathEscaped := escapeSimple(hlsM3U8RelPath)

	// 使用相对路径构建 tee spec
	// 添加 hls_flags=delete_segments+hls_delete_threshold 确保自动删除旧分段，防止长时间录制占用过多磁盘
	teeSpec := fmt.Sprintf("%s|[f=hls:hls_time=%d:hls_list_size=%d:hls_flags=delete_segments:hls_delete_threshold=%d:hls_segment_filename=%s]%s",
		mkvPathEscaped,
		hlsSegmentDuration,
		hlsListSize,
		hlsDeleteThreshold,
		hlsSegmentPathEscaped,
		hlsM3U8PathEscaped,
	)

	args = append(args, "-f", "tee", teeSpec)

	c.logger.Info("FFmpeg tee muxer 命令已构建",
		zap.String("mkv_output", mkvPath),
		zap.String("hls_output", hlsM3U8Path),
		zap.String("hls_segment_path", hlsSegmentPath),
		zap.Int("hls_segment_duration", hlsSegmentDuration),
		zap.Int("hls_list_size", hlsListSize),
		zap.Int("hls_delete_threshold", hlsDeleteThreshold),
		zap.String("tee_spec", teeSpec), // 打印完整的 tee spec 以便调试
	)

	return args, nil
}

// buildInputArgs 构建输入参数
func (c *SimpleRecordingCoordinator) buildInputArgs(input RecordingInput) ([]string, error) {
	var args []string
	var err error

	switch input.Type {
	case InputSourceUSB:
		if input.CameraDevice != "" {
			args, err = c.buildUSBVideoArgs(input)
		}
		if input.AudioDevice != "" {
			audioArgs, e := c.buildUSBAudioArgs(input)
			if e != nil {
				return nil, e
			}
			// 如果音频参数为空（无效设备），跳过音频
			if len(audioArgs) == 0 {
				c.logger.Warn("音频设备无效，跳过音频输入", zap.String("audio_device", input.AudioDevice))
			} else {
				args = append(args, audioArgs...)
			}
		}

	case InputSourceRTSP:
		args, err = c.buildRTSPArgs(input)

	case InputSourceRTMP:
		args, err = c.buildStreamArgs(input)

	case InputSourceStream:
		args, err = c.buildStreamArgs(input)

	case InputSourceMixed:
		// 混合输入：优先处理流媒体，然后添加USB音频
		if input.RTSPURL != "" {
			args, err = c.buildRTSPArgs(input)
		} else if input.StreamURL != "" {
			args, err = c.buildStreamArgs(input)
		}
		if err == nil && input.AudioDevice != "" {
			audioArgs, e := c.buildUSBAudioArgs(input)
			if e != nil {
				return nil, e
			}
			// 如果音频参数为空（无效设备），跳过音频
			if len(audioArgs) == 0 {
				c.logger.Warn("音频设备无效，跳过音频输入", zap.String("audio_device", input.AudioDevice))
			} else {
				args = append(args, audioArgs...)
			}
		}

	default:
		return nil, fmt.Errorf("不支持的输入源类型: %s: %w", input.Type, apperrors.ErrInvalidInput)
	}

	return args, err
}

// buildUSBVideoArgs 构建USB视频输入参数
func (c *SimpleRecordingCoordinator) buildUSBVideoArgs(input RecordingInput) ([]string, error) {
	validBackends := map[string]bool{
		"dshow":        true,
		"v4l2":         true,
		"avfoundation": true,
	}

	if !validBackends[input.CameraBackend] {
		return nil, fmt.Errorf("不支持的摄像头后端: %s: %w", input.CameraBackend, apperrors.ErrInvalidInput)
	}

	// 检查视频设备是否有效
	if input.CameraDevice == "" {
		c.logger.Warn("摄像头设备为空，跳过视频输入")
		return nil, fmt.Errorf("摄像头设备不能为空: %w", apperrors.ErrInvalidInput)
	}

	// 如果设备名称为空但设备值看起来像数字索引，警告但继续（某些系统可能使用数字索引）
	if input.CameraName == "" && isNumericString(input.CameraDevice) {
		c.logger.Warn("摄像头设备名称为空且设备值是数字，可能无法正常工作", zap.String("device", input.CameraDevice))
	}

	deviceParam := input.CameraDevice
	if input.CameraBackend == "dshow" {
		// dshow 需要使用实际设备名称
		// 如果有设备名称，使用实际设备名称（如 "OBS Virtual Camera"）
		if input.CameraName != "" {
			deviceParam = fmt.Sprintf("video=%s", input.CameraName)
		} else if !strings.HasPrefix(input.CameraDevice, "video=") {
			deviceParam = fmt.Sprintf("video=%s", input.CameraDevice)
		}
	}

	// 构建 dshow 参数
	args := []string{
		"-f", input.CameraBackend,
		"-video_size", "1920x1080",
		"-framerate", "30",
	}

	// 为 dshow 增加实时缓冲区和线程队列大小，防止丢帧和阻塞
	if input.CameraBackend == "dshow" {
		// rtbufsize 单位是字节，从配置读取
		args = append(args, "-rtbufsize", fmt.Sprintf("%d", c.config.FFmpeg.DShowBufferSize))
		// thread_queue_size 增加线程队列大小，从配置读取
		args = append(args, "-thread_queue_size", fmt.Sprintf("%d", c.config.FFmpeg.DShowThreadQueueSize))
	}

	args = append(args, "-i", deviceParam)
	return args, nil
}

// buildUSBAudioArgs 构建USB音频输入参数
func (c *SimpleRecordingCoordinator) buildUSBAudioArgs(input RecordingInput) ([]string, error) {
	validBackends := map[string]bool{
		"dshow":     true,
		"alsa":      true,
		"coreaudio": true,
		"wasapi":    true,
	}

	if !validBackends[input.AudioBackend] {
		return nil, fmt.Errorf("不支持的音频后端: %s: %w", input.AudioBackend, apperrors.ErrInvalidInput)
	}

	// 检查音频设备是否有效
	if input.AudioDevice == "" {
		c.logger.Warn("音频设备为空，跳过音频输入")
		return []string{}, nil
	}

	// 如果设备名称为空但设备值看起来像数字索引，也跳过
	if input.AudioName == "" && isNumericString(input.AudioDevice) {
		c.logger.Warn("音频设备名称为空且设备值是数字，跳过音频输入", zap.String("device", input.AudioDevice))
		return []string{}, nil
	}

	deviceParam := input.AudioDevice
	if input.AudioBackend == "dshow" {
		// dshow 需要使用实际设备名称
		// 如果有设备名称，使用实际设备名称（如 "麦克风"）
		if input.AudioName != "" {
			deviceParam = fmt.Sprintf("audio=%s", input.AudioName)
		} else if !strings.HasPrefix(input.AudioDevice, "audio=") {
			deviceParam = fmt.Sprintf("audio=%s", input.AudioDevice)
		}
	}

	// 为 dshow 增加线程队列大小，防止阻塞
	args := []string{"-f", input.AudioBackend}
	if input.AudioBackend == "dshow" {
		args = append(args, "-thread_queue_size", fmt.Sprintf("%d", c.config.FFmpeg.DShowThreadQueueSize))
	}
	args = append(args, "-i", deviceParam)
	return args, nil
}

// isNumericString 检查字符串是否只包含数字
func isNumericString(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// buildRTSPArgs 构建RTSP输入参数
func (c *SimpleRecordingCoordinator) buildRTSPArgs(input RecordingInput) ([]string, error) {
	if input.RTSPURL == "" {
		return nil, fmt.Errorf("RTSP URL不能为空: %w", apperrors.ErrInvalidInput)
	}
	return []string{"-rtsp_transport", "tcp", "-i", input.RTSPURL}, nil
}

// buildStreamArgs 构建流媒体输入参数
func (c *SimpleRecordingCoordinator) buildStreamArgs(input RecordingInput) ([]string, error) {
	if input.StreamURL == "" {
		return nil, fmt.Errorf("流媒体URL不能为空: %w", apperrors.ErrInvalidInput)
	}

	protocol := input.StreamProtocol
	if protocol == "" {
		// 默认使用 RTSP
		protocol = "rtsp"
	}

	var args []string
	switch protocol {
	case "rtmp":
		args = []string{"-i", input.StreamURL}
	case "rtsp":
		args = []string{"-rtsp_transport", "tcp", "-i", input.StreamURL}
	case "srt":
		args = []string{"-i", input.StreamURL}
	case "hls":
		args = []string{"-i", input.StreamURL}
	default:
		return nil, fmt.Errorf("不支持的流媒体协议: %s: %w", protocol, apperrors.ErrInvalidInput)
	}

	return args, nil
}

// HealthCheck 健康检查
func (c *SimpleRecordingCoordinator) HealthCheck() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 检查是否有僵尸进程（状态为running但进程已退出）
	now := time.Now()
	zombieCount := 0
	longRunningCount := 0

	for taskKey, process := range c.processes {
		if process.Status == "running" {
			// 从键中解析实际的 taskID
			actualTaskID := process.TaskID
			// 检查进程是否还在运行（通过检查进程状态）
			if process.Cmd.Process != nil {
				// 检查进程是否已退出
				if process.Cmd.ProcessState != nil && process.Cmd.ProcessState.Exited() {
					c.logger.Warn("发现僵尸录制进程",
						zap.Uint("task_id", actualTaskID),
						zap.String("process_key", taskKey),
					)
					zombieCount++
				}
			}
			// 检查是否有运行时间过长的任务（超过配置的最长录制时长）
			maxDuration := c.config.FFmpeg.MaxRecordingDuration
			if now.Sub(process.StartTime) > maxDuration {
				c.logger.Warn("发现运行时间过长的录制进程",
					zap.Uint("task_id", actualTaskID),
					zap.String("process_key", taskKey),
					zap.Duration("runtime", now.Sub(process.StartTime)),
				)
				longRunningCount++
			}
		}
	}

	if zombieCount > 0 {
		return fmt.Errorf("发现%d个僵尸录制进程", zombieCount)
	}

	return nil
}

// WatcherChannels returns the close-only watcher channels for all inputs of a task.
// It snapshots the matching channels under the coordinator read lock so the scheduler
// can fan them in without accessing recorder-owned process state directly.
func (c *SimpleRecordingCoordinator) WatcherChannels(taskID uint) []<-chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	prefix := fmt.Sprintf("%d_", taskID)
	out := make([]<-chan struct{}, 0, 2)
	for key, proc := range c.processes {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		// 注: Phase 25 智能退出撤回后,proc.taskEndedCh 字段整体删除;此函数保留
		// 仅为 RecorderCoordinatorInterface 接口兼容 (scheduler 端整段 mergeWatchers
		// 取消,但 video_scheduler 的接口契约保留)。
		_ = proc
	}
	return out
}

// GetRecordingStatus 获取录制状态
func (c *SimpleRecordingCoordinator) GetRecordingStatus(taskID uint) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 检查所有相关配置的录制进程
	runningCount := 0
	totalCount := 0

	// 检查带配置类型的键
	for key := range c.processes {
		if strings.HasPrefix(key, fmt.Sprintf("%d_", taskID)) {
			totalCount++
			if c.processes[key].Status == "running" {
				runningCount++
			}
		}
	}

	// 向后兼容：检查旧的单键格式
	if process, ok := c.processes[fmt.Sprintf("%d", taskID)]; ok {
		totalCount++
		if process.Status == "running" {
			runningCount++
		}
	}

	if totalCount == 0 {
		return "", fmt.Errorf("未找到录制任务: %d: %w", taskID, apperrors.ErrTaskNotFound)
	}

	if runningCount > 0 {
		return "running", nil
	}
	return "stopped", nil
}
