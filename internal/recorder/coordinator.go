package recorder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
)

// 录制编码参数常量
const (
	videoCodec     = "libx264"
	videoPreset    = "medium"
	videoBitrate   = "5M"
	videoPixelFormat = "yuv420p"
	audioCodec     = "aac"
	audioBitrate   = "128k"
	outputFormat   = "mp4"
)

// SimpleRecordingCoordinator 简单的录制协调器
type SimpleRecordingCoordinator struct {
	logger      *zap.Logger
	config      *config.Config
	processes   map[uint]*RecordingProcess
	cancelFuncs map[uint]context.CancelFunc
	mu          sync.RWMutex
}

// RecordingProcess 录制进程
type RecordingProcess struct {
	TaskID     uint
	Cmd        *exec.Cmd
	StartTime  time.Time
	OutputPath string
	Status     string
	CancelFunc context.CancelFunc
}

// InputSourceType 输入源类型
type InputSourceType string

const (
	InputSourceUSB   InputSourceType = "usb"   // USB设备
	InputSourceRTSP  InputSourceType = "rtsp"  // RTSP流
	InputSourceMixed InputSourceType = "mixed" // 混合输入
)

// RecordingInput 录制输入配置
type RecordingInput struct {
	Type         InputSourceType `json:"type"`
	RTSPURL      string          `json:"rtsp_url,omitempty"`
	CameraBackend string         `json:"camera_backend"`
	AudioBackend  string         `json:"audio_backend"`
	CameraDevice  string          `json:"camera_device,omitempty"`
	AudioDevice   string          `json:"audio_device,omitempty"`
	hasAudio      bool            // 内部标记是否有音频
}

// NewSimpleRecordingCoordinator 创建录制协调器
func NewSimpleRecordingCoordinator(logger *zap.Logger, cfg *config.Config) *SimpleRecordingCoordinator {
	return &SimpleRecordingCoordinator{
		logger:      logger,
		config:      cfg,
		processes:   make(map[uint]*RecordingProcess),
		cancelFuncs: make(map[uint]context.CancelFunc),
	}
}

// StartRecording 启动录制
func (c *SimpleRecordingCoordinator) StartRecording(task *models.VideoRecordingTask, huaweiConfig *models.HuaweiConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	outputPath := c.getOutputPath(task.ID, huaweiConfig.OutputFormat)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	input := c.buildRecordingInput(task, huaweiConfig)
	args, err := c.buildRecordingCommand(input, outputPath, task.EndTime.Sub(task.StartTime))
	if err != nil {
		return fmt.Errorf("构建录制命令失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	if err := c.startFFmpegProcess(cmd, outputPath); err != nil {
		cancel()
		return err
	}

	c.processes[task.ID] = &RecordingProcess{
		TaskID:     task.ID,
		Cmd:        cmd,
		StartTime:  time.Now(),
		OutputPath: outputPath,
		Status:     "running",
		CancelFunc: cancel,
	}
	c.cancelFuncs[task.ID] = cancel
	task.RecordingFile = outputPath

	c.logger.Info("录制已启动",
		zap.Uint("task_id", task.ID),
		zap.String("input_type", string(input.Type)),
		zap.String("output_path", outputPath),
	)

	return nil
}

// getOutputPath 生成输出文件路径
func (c *SimpleRecordingCoordinator) getOutputPath(taskID uint, format string) string {
	outputDir := filepath.Join(c.config.Storage.RecordingsPath, fmt.Sprintf("task_%d", taskID))
	timestamp := time.Now().Format("20060102150405")
	return filepath.Join(outputDir, fmt.Sprintf("recording_%s.%s", timestamp, format))
}

// startFFmpegProcess 启动FFmpeg进程并配置日志
func (c *SimpleRecordingCoordinator) startFFmpegProcess(cmd *exec.Cmd, outputPath string) error {
	logPath := filepath.Join(filepath.Dir(outputPath), "ffmpeg.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("启动FFmpeg进程失败: %w", err)
	}

	return nil
}

// buildRecordingInput 构建录制输入配置
func (c *SimpleRecordingCoordinator) buildRecordingInput(task *models.VideoRecordingTask, huaweiConfig *models.HuaweiConfig) RecordingInput {
	input := RecordingInput{
		CameraBackend: huaweiConfig.CameraBackend,
		AudioBackend:  huaweiConfig.AudioBackend,
		CameraDevice:  huaweiConfig.USBCameraDevice,
		AudioDevice:   huaweiConfig.USBAudioDevice,
	}

	// 检查任务配置的RTSP流
	if task.RTSPStreamURL != "" {
		input.Type = InputSourceRTSP
		input.RTSPURL = task.RTSPStreamURL
		input.hasAudio = true // RTSP流通常包含音频
	}

	// 检查USB设备配置
	hasUSB := huaweiConfig.USBCameraDevice != "" || huaweiConfig.USBAudioDevice != ""
	if hasUSB {
		input.hasAudio = input.hasAudio || huaweiConfig.USBAudioDevice != ""
		if input.Type == InputSourceRTSP {
			input.Type = InputSourceMixed
		} else {
			input.Type = InputSourceUSB
		}
	}

	return input
}

// StopRecording 停止录制
func (c *SimpleRecordingCoordinator) StopRecording(taskID uint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	process, ok := c.processes[taskID]
	if !ok {
		return fmt.Errorf("未找到录制任务: %d", taskID)
	}

	process.CancelFunc()
	c.waitForProcess(process, taskID)

	delete(c.processes, taskID)
	delete(c.cancelFuncs, taskID)

	c.logger.Info("录制已停止", zap.Uint("task_id", taskID))
	return nil
}

// waitForProcess 等待进程结束（带超时）
func (c *SimpleRecordingCoordinator) waitForProcess(process *RecordingProcess, taskID uint) {
	done := make(chan error, 1)
	go func() { done <- process.Cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			c.logger.Error("FFmpeg进程退出异常", zap.Uint("task_id", taskID), zap.Error(err))
		}
	case <-time.After(10 * time.Second):
		if process.Cmd.Process != nil {
			process.Cmd.Process.Kill()
		}
		c.logger.Warn("FFmpeg进程超时，强制终止", zap.Uint("task_id", taskID))
	}
}

// buildRecordingCommand 构建录制命令
func (c *SimpleRecordingCoordinator) buildRecordingCommand(input RecordingInput, outputPath string, duration time.Duration) ([]string, error) {
	args := []string{"-y"}

	// 添加输入源
	inputArgs, err := c.buildInputArgs(input)
	if err != nil {
		return nil, err
	}
	args = append(args, inputArgs...)

	// 添加视频编码参数
	args = append(args,
		"-c:v", videoCodec,
		"-preset", videoPreset,
		"-b:v", videoBitrate,
		"-pix_fmt", videoPixelFormat,
	)

	// 添加音频编码参数（如果有音频）
	if input.hasAudio {
		args = append(args, "-c:a", audioCodec, "-b:a", audioBitrate)
	}

	// 添加输出参数
	args = append(args, "-f", outputFormat, "-movflags", "+faststart")

	// 添加时长限制
	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.0f", duration.Seconds()))
	}

	args = append(args, outputPath)
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
			args = append(args, audioArgs...)
		}

	case InputSourceRTSP:
		args, err = c.buildRTSPArgs(input)

	case InputSourceMixed:
		args, err = c.buildRTSPArgs(input)
		if err == nil && input.AudioDevice != "" {
			audioArgs, e := c.buildUSBAudioArgs(input)
			if e != nil {
				return nil, e
			}
			args = append(args, audioArgs...)
		}

	default:
		return nil, fmt.Errorf("不支持的输入源类型: %s", input.Type)
	}

	return args, err
}

// buildUSBVideoArgs 构建USB视频输入参数
func (c *SimpleRecordingCoordinator) buildUSBVideoArgs(input RecordingInput) ([]string, error) {
	deviceParam := input.CameraDevice
	if input.CameraBackend == "dshow" {
		deviceParam = fmt.Sprintf("video=%s", input.CameraDevice)
	}

	return []string{
		"-f", input.CameraBackend,
		"-video_size", "1920x1080",
		"-framerate", "30",
		"-i", deviceParam,
	}, nil
}

// buildUSBAudioArgs 构建USB音频输入参数
func (c *SimpleRecordingCoordinator) buildUSBAudioArgs(input RecordingInput) ([]string, error) {
	deviceParam := input.AudioDevice
	if input.AudioBackend == "dshow" {
		deviceParam = fmt.Sprintf("audio=%s", input.AudioDevice)
	}

	return []string{"-f", input.AudioBackend, "-i", deviceParam}, nil
}

// buildRTSPArgs 构建RTSP输入参数
func (c *SimpleRecordingCoordinator) buildRTSPArgs(input RecordingInput) ([]string, error) {
	if input.RTSPURL == "" {
		return nil, fmt.Errorf("RTSP URL不能为空")
	}
	return []string{"-rtsp_transport", "tcp", "-i", input.RTSPURL}, nil
}

// HealthCheck 健康检查
func (c *SimpleRecordingCoordinator) HealthCheck() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return nil
}

// GetRecordingStatus 获取录制状态
func (c *SimpleRecordingCoordinator) GetRecordingStatus(taskID uint) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	process, ok := c.processes[taskID]
	if !ok {
		return "", fmt.Errorf("未找到录制任务: %d", taskID)
	}
	return process.Status, nil
}
