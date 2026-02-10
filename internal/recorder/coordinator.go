package recorder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
)

// 录制编码参数常量
const (
	videoCodec       = "libx264"
	videoPreset      = "medium"
	videoBitrate     = "5M"
	videoPixelFormat = "yuv420p"
	audioCodec       = "aac"
	audioBitrate     = "128k"
	outputFormat     = "mkv" // 使用 MKV 格式，防止中断时文件损坏
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
	logFile    *os.File
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
	CameraName    string          `json:"camera_name,omitempty"`    // 实际设备名称
	AudioName     string          `json:"audio_name,omitempty"`     // 实际设备名称
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

	// 使用 MKV 格式（MKV 在中断时不易损坏）
	outputPath := c.getOutputPath(task, outputFormat)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	input := c.buildRecordingInput(task, huaweiConfig)
	args, err := c.buildRecordingCommand(input, outputPath, task.EndTime.Sub(task.StartTime))
	if err != nil {
		return fmt.Errorf("构建录制命令失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, c.config.FFmpeg.Path, args...)

	logFile, err := c.startFFmpegProcess(cmd, outputPath)
	if err != nil {
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
		logFile:    logFile,
	}
	c.cancelFuncs[task.ID] = cancel
	task.RecordingFile = outputPath  // 兼容旧字段
	task.MKVFilePath = outputPath    // 新字段指向 MKV 文件

	c.logger.Info("录制已启动",
		zap.Uint("task_id", task.ID),
		zap.String("input_type", string(input.Type)),
		zap.String("output_path", outputPath),
	)

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
	// 替换不允许在文件名中出现的字符
	replacements := map[rune]string{
		' ':  "_",
		'/':  "_",
		'\\': "_",
		':':  "_",
		'*':  "_",
		'?':  "_",
		'"':  "_",
		'<':  "_",
		'>':  "_",
		'|':  "_",
		'\n': "",
		'\r': "",
		'\t': "",
	}

	result := make([]rune, 0, len(name))
	for _, r := range name {
		if repl, ok := replacements[r]; ok {
			result = append(result, []rune(repl)...)
		} else if r < 32 {
			// 跳过控制字符
			continue
		} else {
			result = append(result, r)
		}
	}

	// 限制文件名长度
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
		return nil, fmt.Errorf("创建日志文件失败: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("启动FFmpeg进程失败: %w", err)
	}

	return logFile, nil
}

// buildRecordingInput 构建录制输入配置
func (c *SimpleRecordingCoordinator) buildRecordingInput(task *models.VideoRecordingTask, huaweiConfig *models.HuaweiConfig) RecordingInput {
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
	c.waitForProcess(process, taskID, true) // true 表示是主动停止

	// 关闭日志文件
	if process.logFile != nil {
		process.logFile.Close()
	}

	delete(c.processes, taskID)
	delete(c.cancelFuncs, taskID)

	c.logger.Info("录制已停止", zap.Uint("task_id", taskID))
	return nil
}

// waitForProcess 等待进程结束（带超时）
// expectedStop: 是否是预期的停止（主动停止录制），如果是则不会记录错误日志
func (c *SimpleRecordingCoordinator) waitForProcess(process *RecordingProcess, taskID uint, expectedStop bool) {
	done := make(chan error, 1)
	go func() { done <- process.Cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			if expectedStop {
				// 主动停止时，FFmpeg进程被终止是正常行为
				c.logger.Debug("FFmpeg进程已停止", zap.Uint("task_id", taskID), zap.Error(err))
			} else {
				// 非预期退出才记录为错误
				c.logger.Error("FFmpeg进程退出异常", zap.Uint("task_id", taskID), zap.Error(err))
			}
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
	validBackends := map[string]bool{
		"dshow":        true,
		"v4l2":         true,
		"avfoundation":  true,
	}

	if !validBackends[input.CameraBackend] {
		return nil, fmt.Errorf("不支持的摄像头后端: %s", input.CameraBackend)
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

	return []string{
		"-f", input.CameraBackend,
		"-video_size", "1920x1080",
		"-framerate", "30",
		"-i", deviceParam,
	}, nil
}

// buildUSBAudioArgs 构建USB音频输入参数
func (c *SimpleRecordingCoordinator) buildUSBAudioArgs(input RecordingInput) ([]string, error) {
	validBackends := map[string]bool{
		"dshow":      true,
		"alsa":       true,
		"coreaudio":  true,
		"wasapi":     true,
	}

	if !validBackends[input.AudioBackend] {
		return nil, fmt.Errorf("不支持的音频后端: %s", input.AudioBackend)
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

	// 检查是否有僵尸进程（状态为running但进程已退出）
	now := time.Now()
	zombieCount := 0
	longRunningCount := 0

	for taskID, process := range c.processes {
		if process.Status == "running" {
			// 检查进程是否还在运行（通过检查进程状态）
			if process.Cmd.Process != nil {
				// 检查进程是否已退出
				if process.Cmd.ProcessState != nil && process.Cmd.ProcessState.Exited() {
					c.logger.Warn("发现僵尸录制进程",
						zap.Uint("task_id", taskID),
					)
					zombieCount++
				}
			}
			// 检查是否有运行时间过长的任务（超过24小时）
			if now.Sub(process.StartTime) > 24*time.Hour {
				c.logger.Warn("发现运行时间过长的录制进程",
					zap.Uint("task_id", taskID),
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
