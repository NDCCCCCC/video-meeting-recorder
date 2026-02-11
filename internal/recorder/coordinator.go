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

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
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
	Cmd        *exec.Cmd // 主录制进程 (使用 tee muxer 同时输出 MKV 和 HLS)
	StartTime  time.Time
	OutputPath string // MKV文件路径
	HLSPath    string // HLS预览路径 (m3u8文件)
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
	mkvPath := c.getOutputPath(task, "mkv")
	if err := os.MkdirAll(filepath.Dir(mkvPath), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 生成 HLS 输出路径
	hlsPath := c.getHLSPath(task)
	if err := os.MkdirAll(hlsPath, 0755); err != nil {
		return fmt.Errorf("创建HLS目录失败: %w", err)
	}

	input := c.buildRecordingInput(task, huaweiConfig)

	// 使用 tee muxer 同时生成 MKV 和 HLS
	args, err := c.buildRecordingCommand(input, mkvPath, hlsPath, task.EndTime.Sub(task.StartTime))
	if err != nil {
		return fmt.Errorf("构建录制命令失败: %w", err)
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

	c.processes[task.ID] = &RecordingProcess{
		TaskID:     task.ID,
		Cmd:        cmd,
		StartTime:  time.Now(),
		OutputPath: mkvPath,
		HLSPath:    m3u8Path,
		Status:     "running",
		CancelFunc: cancel,
		logFile:    logFile,
	}
	c.cancelFuncs[task.ID] = cancel
	task.RecordingFile = mkvPath     // 兼容旧字段
	task.MKVFilePath = mkvPath       // 新字段指向 MKV 文件
	task.HLSPreviewPath = m3u8Path   // HLS 预览路径

	c.logger.Info("录制已启动（同时生成 MKV 和 HLS）",
		zap.Uint("task_id", task.ID),
		zap.String("input_type", string(input.Type)),
		zap.String("mkv_path", mkvPath),
		zap.String("hls_path", m3u8Path),
	)

	// 启动监控 goroutine，等待进程结束
	go c.monitorProcess(task.ID, cmd, ctx)

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

// getHLSPath 生成 HLS 输出目录路径
// 格式: data/hls/task_{id}/{name}_{conf}_{timestamp}/
func (c *SimpleRecordingCoordinator) getHLSPath(task *models.VideoRecordingTask) string {
	safeName := sanitizeFilename(task.Name)
	conferenceNumber := task.ConferenceNumber
	timestamp := time.Now().Format("20060102150405")

	dirname := fmt.Sprintf("%s_%s_%s", safeName, conferenceNumber, timestamp)
	hlsDir := filepath.Join(c.config.Storage.HLSPath, fmt.Sprintf("task_%d", task.ID), dirname)
	return hlsDir
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

// normalizePathForFFmpeg 将 Windows 路径转换为 FFmpeg 兼容格式（使用正斜杠）
func normalizePathForFFmpeg(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
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

	// 停止主录制进程（tee muxer 会自动停止所有输出）
	process.CancelFunc()
	// 注意：不再调用 waitForProcess，因为 monitorProcess goroutine 已经在等待进程结束
	// 当进程退出时，monitorProcess 会更新进程状态为 "stopped"

	// 关闭日志文件
	if process.logFile != nil {
		process.logFile.Close()
	}

	delete(c.processes, taskID)
	delete(c.cancelFuncs, taskID)

	c.logger.Info("录制已停止", zap.Uint("task_id", taskID))
	return nil
}

// monitorProcess 监控录制进程状态
func (c *SimpleRecordingCoordinator) monitorProcess(taskID uint, cmd *exec.Cmd, ctx context.Context) {
	// 先等待进程结束，不持有锁
	err := cmd.Wait()

	// 然后获取锁更新状态
	c.mu.Lock()
	process, exists := c.processes[taskID]
	if exists {
		process.Status = "stopped"
	}
	c.mu.Unlock()

	if err != nil {
		if ctx.Err() != nil {
			// Context 被取消，是主动停止
			c.logger.Debug("录制进程已停止", zap.Uint("task_id", taskID))
		} else {
			// 非预期退出
			c.logger.Error("录制进程异常退出",
				zap.Uint("task_id", taskID),
				zap.Error(err),
			)
		}
	} else {
		c.logger.Info("录制进程正常结束", zap.Uint("task_id", taskID))
	}
}

// buildRecordingCommand 构建录制命令（使用 tee muxer 同时输出 MKV 和 HLS）
func (c *SimpleRecordingCoordinator) buildRecordingCommand(input RecordingInput, mkvPath string, hlsPath string, duration time.Duration) ([]string, error) {
	args := []string{"-y"}

	// 添加输入源
	inputArgs, err := c.buildInputArgs(input)
	if err != nil {
		return nil, err
	}
	args = append(args, inputArgs...)

	// 添加视频编码参数
	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-b:v", "5M",
		"-pix_fmt", "yuv420p",
	)

	// 添加音频编码参数（如果有音频）
	if input.hasAudio {
		args = append(args, "-c:a", "aac", "-b:a", "128k")
	}

	// 添加时长限制
	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.0f", duration.Seconds()))
	}

	// 添加 global_header 标志，这是 tee muxer 正常工作的关键
	// MKV 和 HLS 容器需要全局存储的比特流参数
	args = append(args, "-flags", "+global_header")

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

	// Tee muxer 格式: "mkv_path|[f=hls:hls_time=X:hls_list_size=Y:hls_segment_filename=Z]m3u8_path"
	teeSpec := fmt.Sprintf("%s|[f=hls:hls_time=%d:hls_list_size=%d:hls_segment_filename=%s]%s",
		normalizePathForFFmpeg(mkvPath),
		hlsSegmentDuration,
		hlsListSize,
		normalizePathForFFmpeg(hlsSegmentPath),
		normalizePathForFFmpeg(hlsM3U8Path),
	)

	args = append(args, "-f", "tee", teeSpec)

	c.logger.Debug("FFmpeg tee muxer 命令已构建",
		zap.String("mkv_output", mkvPath),
		zap.String("hls_output", hlsM3U8Path),
		zap.Int("hls_segment_duration", hlsSegmentDuration),
		zap.Int("hls_list_size", hlsListSize),
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

	// 为 dshow 增加线程队列大小，防止阻塞
	args := []string{"-f", input.AudioBackend}
	if input.AudioBackend == "dshow" {
		args = append(args, "-thread_queue_size", fmt.Sprintf("%d", c.config.FFmpeg.DShowThreadQueueSize))
	}
	args = append(args, "-i", deviceParam)
	return args, nil
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
			// 检查是否有运行时间过长的任务（超过配置的最长录制时长）
			maxDuration := c.config.FFmpeg.MaxRecordingDuration
			if now.Sub(process.StartTime) > maxDuration {
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
