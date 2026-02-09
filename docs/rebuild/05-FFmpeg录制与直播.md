# FFmpeg录制与直播

## 一、FFmpeg集成概述

### 1.1 功能概述

系统通过FFmpeg实现以下功能：

- **USB摄像头录制** - 跨平台摄像头视频采集
- **音频录制** - 支持麦克风和系统音频
- **RTSP流录制** - 录制华为会议直播流
- **直播推流** - 将视频推送到RTMP/RTSP服务器
- **格式转换** - MKV到MP4的异步转换
- **视频处理** - 剪辑、合并、转码等

### 1.2 架构设计

```
┌────────────────────────────────────────────────────────────────┐
│              SimpleRecordingCoordinator (474行)                │
│                    录制协调器                                   │
│  - StartRecording()      启动录制                              │
│  - StopRecording()       停止录制                               │
│  - MonitorRecording()    监控录制状态                          │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│              FFmpegOrchestrator (723行)                        │
│                    FFmpeg编排器                                 │
│  - StartProcess()        启动FFmpeg进程                        │
│  - StopProcess()         停止进程                               │
│  - MonitorProcesses()    监控所有进程                           │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│                    FFmpeg进程                                   │
│  - 录制进程             capture_<task_id>                      │
│  - 转换进程             convert_<task_id>                      │
│  - 推流进程             stream_<task_id>                       │
└────────────────────────────────────────────────────────────────┘
```

### 1.3 核心组件

| 组件 | 文件 | 行数 | 职责 |
|------|------|------|------|
| SimpleRecordingCoordinator | simple_recording_coordinator.go | 474 | 录制协调器 |
| FFmpegOrchestrator | ffmpeg_orchestrator/orchestrator.go | 723 | FFmpeg编排器 |
| Recorder | rtsp/recorder.go | 355 | RTSP录制器 |
| StreamManager | rtsp/stream_manager.go | 971 | 流管理器 |

## 二、FFmpeg命令生成

### 2.1 USB摄像头录制

```go
// internal/services/video_recording/simple_recording_coordinator.go
type RecordingConfig struct {
    TaskID             uint
    OutputPath         string
    Duration           time.Duration
    VideoDevice        string
    AudioDevice        string
    Resolution         string
    FrameRate          int
    VideoBitrate       string
    AudioBitrate       string
    Format             string
}

// BuildFFmpegCommand 构建FFmpeg命令
func (c *RecordingConfig) BuildFFmpegCommand() []string {
    args := []string{"-y"} // 覆盖输出文件

    // 跨平台设备输入
    switch runtime.GOOS {
    case "windows":
        // Windows: DirectShow
        args = append(args,
            "-f", "dshow",
            "-video_size", c.Resolution,
            "-framerate", fmt.Sprintf("%d", c.FrameRate),
            "-i", fmt.Sprintf("video=%s", c.VideoDevice),
        )
        if c.AudioDevice != "" {
            args = append(args,
                "-f", "dshow",
                "-i", fmt.Sprintf("audio=%s", c.AudioDevice),
            )
        }

    case "linux":
        // Linux: Video4Linux2
        args = append(args,
            "-f", "v4l2",
            "-video_size", c.Resolution,
            "-framerate", fmt.Sprintf("%d", c.FrameRate),
            "-i", c.VideoDevice,
        )
        if c.AudioDevice != "" {
            args = append(args,
                "-f", "alsa",
                "-i", c.AudioDevice,
            )
        }

    case "darwin":
        // macOS: AVFoundation
        args = append(args,
            "-f", "avfoundation",
            "-video_size", c.Resolution,
            "-framerate", fmt.Sprintf("%d", c.FrameRate),
            "-i", c.VideoDevice,
        )
        if c.AudioDevice != "" {
            args = append(args,
                "-f", "avfoundation",
                "-i", c.AudioDevice,
            )
        }
    }

    // 视频编码参数
    args = append(args,
        "-c:v", "libx264",
        "-preset", "medium",
        "-b:v", c.VideoBitrate,
        "-pix_fmt", "yuv420p",
    )

    // 音频编码参数
    if c.AudioDevice != "" {
        args = append(args,
            "-c:a", "aac",
            "-b:a", c.AudioBitrate,
        )
    }

    // 输出格式
    args = append(args,
        "-f", c.Format,
        "-movflags", "+faststart", // MP4快速启动
        c.OutputPath,
    )

    return args
}
```

**命令示例**：

```bash
# Windows (DirectShow)
ffmpeg -y \
  -f dshow -video_size 1920x1080 -framerate 30 -i "video=USB Camera" \
  -f dshow -i "audio=麦克风" \
  -c:v libx264 -preset medium -b:v 5M -pix_fmt yuv420p \
  -c:a aac -b:a 128k \
  -f mp4 -movflags +faststart \
  output.mp4

# Linux (Video4Linux2)
ffmpeg -y \
  -f v4l2 -video_size 1920x1080 -framerate 30 -i /dev/video0 \
  -f alsa -i hw:1,0 \
  -c:v libx264 -preset medium -b:v 5M -pix_fmt yuv420p \
  -c:a aac -b:a 128k \
  -f mp4 -movflags +faststart \
  output.mp4

# macOS (AVFoundation)
ffmpeg -y \
  -f avfoundation -video_size 1920x1080 -framerate 30 -i "0" \
  -f avfoundation -i ":0" \
  -c:v libx264 -preset medium -b:v 5M -pix_fmt yuv420p \
  -c:a aac -b:a 128k \
  -f mp4 -movflags +faststart \
  output.mp4
```

### 2.2 RTSP流录制

```go
// BuildRTSPRecordingCommand 构建RTSP录制命令
func BuildRTSPRecordingCommand(rtspURL, outputPath string, duration time.Duration) []string {
    args := []string{
        "-y",                    // 覆盖输出文件
        "-rtsp_transport", "tcp", // 使用TCP传输(更稳定)
        "-i", rtspURL,           // RTSP输入
        "-c", "copy",            // 复制流(不重新编码)
        "-f", "mp4",             // MP4格式
        "-movflags", "+faststart",
        outputPath,
    }

    if duration > 0 {
        args = append(args,
            "-t", fmt.Sprintf("%.0f", duration.Seconds()),
        )
    }

    return args
}
```

**命令示例**：

```bash
# RTSP流录制
ffmpeg -y \
  -rtsp_transport tcp \
  -i "rtsp://10.62.10.3:554/stream1" \
  -c copy \
  -f mp4 -movflags +faststart \
  -t 3600 \
  output.mp4
```

### 2.3 直播推流

```go
// BuildStreamingCommand 构建推流命令
func BuildStreamingCommand(inputPath, rtmpURL string) []string {
    args := []string{
        "-re",                   // 读取输入的本地帧率
        "-i", inputPath,         // 输入文件
        "-c", "copy",            // 复制流
        "-f", "flv",             // FLV格式
        rtmpURL,                 // RTMP服务器地址
    }

    return args
}
```

**命令示例**：

```bash
# 推流到RTMP服务器
ffmpeg -re \
  -i input.mp4 \
  -c copy \
  -f flv \
  rtmp://localhost/live/stream
```

## 三、FFmpegOrchestrator 实现

### 3.1 进程管理

```go
// internal/core/ffmpeg_orchestrator/orchestrator.go
type FFmpegOrchestrator struct {
    processes  map[string]*ManagedProcess
    byType     map[ProcessType]map[string]*ManagedProcess
    bySession  map[string]map[string]*ManagedProcess
    events     chan *ProcessEvent
    logger     *zap.Logger
    config     *config.Config
    mu         sync.RWMutex
}

// 进程类型
type ProcessType string

const (
    ProcessTypeCapture ProcessType = "capture" // 录制进程
    ProcessTypeConvert ProcessType = "convert" // 转换进程
    ProcessTypeStream  ProcessType = "stream"  // 推流进程
)

// ManagedProcess 托管进程
type ManagedProcess struct {
    ID            string
    Type          ProcessType
    SessionID     string
    Cmd           *exec.Cmd
    Process       *os.Process
    StartTime     time.Time
    Status        ProcessStatus
    OutputPath    string
    CancelFunc    context.CancelFunc
    RetryCount    int
    MaxRetries    int
}

// 进程状态
type ProcessStatus string

const (
    ProcessStatusStarting  ProcessStatus = "starting"
    ProcessStatusRunning   ProcessStatus = "running"
    ProcessStatusStopping  ProcessStatus = "stopping"
    ProcessStatusStopped   ProcessStatus = "stopped"
    ProcessStatusFailed    ProcessStatus = "failed"
    ProcessStatusCompleted ProcessStatus = "completed"
)

// StartProcess 启动FFmpeg进程
func (o *FFmpegOrchestrator) StartProcess(ctx context.Context, req *StartProcessRequest) (*ManagedProcess, error) {
    // 1. 创建进程ID
    processID := fmt.Sprintf("%s_%d_%s", req.Type, req.TaskID, time.Now().Format("20060102150405"))

    // 2. 构建命令
    cmd := exec.CommandContext(ctx, o.config.FFmpegPath, req.Args...)

    // 3. 设置输出
    logPath := filepath.Join(o.config.LogDir, fmt.Sprintf("ffmpeg_%s.log", processID))
    logFile, err := os.Create(logPath)
    if err != nil {
        return nil, fmt.Errorf("创建日志文件失败: %w", err)
    }

    cmd.Stdout = logFile
    cmd.Stderr = logFile

    // 4. 启动进程
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("启动FFmpeg进程失败: %w", err)
    }

    // 5. 创建托管进程
    process := &ManagedProcess{
        ID:         processID,
        Type:       req.Type,
        SessionID:  req.SessionID,
        Cmd:        cmd,
        Process:    cmd.Process,
        StartTime:  time.Now(),
        Status:     ProcessStatusRunning,
        OutputPath: req.OutputPath,
        MaxRetries: 3,
    }

    // 6. 注册进程
    o.mu.Lock()
    o.processes[processID] = process
    if o.byType[process.Type] == nil {
        o.byType[process.Type] = make(map[string]*ManagedProcess)
    }
    o.byType[process.Type][processID] = process
    if o.bySession[process.SessionID] == nil {
        o.bySession[process.SessionID] = make(map[string]*ManagedProcess)
    }
    o.bySession[process.SessionID][processID] = process
    o.mu.Unlock()

    // 7. 发布事件
    o.events <- &ProcessEvent{
        Type:      ProcessEventTypeStarted,
        ProcessID: processID,
        Timestamp: time.Now(),
    }

    // 8. 监控进程
    go o.monitorProcess(process)

    return process, nil
}

// StopProcess 停止进程
func (o *FFmpegOrchestrator) StopProcess(processID string, timeout time.Duration) error {
    o.mu.RLock()
    process, ok := o.processes[processID]
    o.mu.RUnlock()

    if !ok {
        return fmt.Errorf("进程不存在: %s", processID)
    }

    // 1. 更新状态
    process.Status = ProcessStatusStopping

    // 2. 尝试优雅停止
    if process.CancelFunc != nil {
        process.CancelFunc()
    }

    // 3. 等待进程退出
    done := make(chan error, 1)
    go func() {
        done <- process.Process.Wait()
    }()

    select {
    case <-time.After(timeout):
        // 超时，强制终止
        process.Process.Kill()
        o.logger.Warn("FFmpeg进程超时，强制终止",
            zap.String("process_id", processID),
        )
    case err := <-done:
        if err != nil {
            o.logger.Error("FFmpeg进程退出异常",
                zap.String("process_id", processID),
                zap.Error(err),
            )
        }
    }

    // 4. 更新状态
    process.Status = ProcessStatusStopped

    // 5. 发布事件
    o.events <- &ProcessEvent{
        Type:      ProcessEventTypeStopped,
        ProcessID: processID,
        Timestamp: time.Now(),
    }

    return nil
}

// monitorProcess 监控进程
func (o *FFmpegOrchestrator) monitorProcess(process *ManagedProcess) {
    err := process.Process.Wait()

    o.mu.Lock()
    if err != nil {
        process.Status = ProcessStatusFailed
        o.logger.Error("FFmpeg进程异常退出",
            zap.String("process_id", process.ID),
            zap.Error(err),
        )
    } else {
        process.Status = ProcessStatusCompleted
    }
    o.mu.Unlock()

    // 发布事件
    eventType := ProcessEventTypeCompleted
    if err != nil {
        eventType = ProcessEventTypeFailed
    }

    o.events <- &ProcessEvent{
        Type:      eventType,
        ProcessID: process.ID,
        Error:     err,
        Timestamp: time.Now(),
    }
}
```

### 3.2 事件处理

```go
// ProcessEventType 进程事件类型
type ProcessEventType string

const (
    ProcessEventTypeStarted   ProcessEventType = "started"
    ProcessEventTypeStopped   ProcessEventType = "stopped"
    ProcessEventTypeCompleted ProcessEventType = "completed"
    ProcessEventTypeFailed    ProcessEventType = "failed"
)

// ProcessEvent 进程事件
type ProcessEvent struct {
    Type      ProcessEventType
    ProcessID string
    Error     error
    Timestamp time.Time
}

// StartEventLoop 启动事件循环
func (o *FFmpegOrchestrator) StartEventLoop() {
    for event := range o.events {
        o.handleEvent(event)
    }
}

// handleEvent 处理事件
func (o *FFmpegOrchestrator) handleEvent(event *ProcessEvent) {
    switch event.Type {
    case ProcessEventTypeStarted:
        o.logger.Info("FFmpeg进程启动",
            zap.String("process_id", event.ProcessID),
        )

    case ProcessEventTypeCompleted:
        o.logger.Info("FFmpeg进程完成",
            zap.String("process_id", event.ProcessID),
        )
        // 清理进程
        o.cleanupProcess(event.ProcessID)

    case ProcessEventTypeFailed:
        o.logger.Error("FFmpeg进程失败",
            zap.String("process_id", event.ProcessID),
            zap.Error(event.Error),
        )
        // 清理进程
        o.cleanupProcess(event.ProcessID)

        // 尝试重试
        o.retryProcess(event.ProcessID)

    case ProcessEventTypeStopped:
        o.logger.Info("FFmpeg进程停止",
            zap.String("process_id", event.ProcessID),
        )
        // 清理进程
        o.cleanupProcess(event.ProcessID)
    }
}

// cleanupProcess 清理进程
func (o *FFmpegOrchestrator) cleanupProcess(processID string) {
    o.mu.Lock()
    defer o.mu.Unlock()

    process, ok := o.processes[processID]
    if !ok {
        return
    }

    // 从索引中删除
    delete(o.processes, processID)
    if o.byType[process.Type] != nil {
        delete(o.byType[process.Type], processID)
    }
    if o.bySession[process.SessionID] != nil {
        delete(o.bySession[process.SessionID], processID)
    }
}
```

## 四、SimpleRecordingCoordinator 实现

### 4.1 录制协调器

```go
// internal/services/video_recording/simple_recording_coordinator.go
type SimpleRecordingCoordinator struct {
    orchestrator *ffmpeg_orchestrator.FFmpegOrchestrator
    logger       *zap.Logger
    config       *config.Config
}

// StartRecording 启动录制
func (c *SimpleRecordingCoordinator) StartRecording(task *models.VideoRecordingTask, huaweiConfig *models.HuaweiConfig) error {
    // 1. 生成输出路径
    outputDir := filepath.Join(c.config.Storage.RecordingsDir, fmt.Sprintf("task_%d", task.ID))
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return fmt.Errorf("创建输出目录失败: %w", err)
    }

    outputPath := filepath.Join(outputDir, fmt.Sprintf("recording_%s.mkv",
        time.Now().Format("20060102150405")))

    // 2. 构建录制配置
    recordingConfig := &RecordingConfig{
        TaskID:       task.ID,
        OutputPath:   outputPath,
        Duration:     task.EndTime.Sub(task.StartTime),
        VideoDevice:  huaweiConfig.USBCameraDevice,
        AudioDevice:  huaweiConfig.USBAudioDevice,
        Resolution:   "1920x1080",
        FrameRate:    30,
        VideoBitrate: "5M",
        AudioBitrate: "128k",
        Format:       "mkv", // 先录制MKV(容错性好)
    }

    // 3. 构建FFmpeg命令
    args := recordingConfig.BuildFFmpegCommand()

    // 4. 启动进程
    ctx, cancel := context.WithCancel(context.Background())

    req := &ffmpeg_orchestrator.StartProcessRequest{
        Type:       ffmpeg_orchestrator.ProcessTypeCapture,
        TaskID:     task.ID,
        SessionID:  fmt.Sprintf("task_%d", task.ID),
        Args:       args,
        OutputPath: outputPath,
    }

    process, err := c.orchestrator.StartProcess(ctx, req)
    if err != nil {
        cancel()
        return fmt.Errorf("启动录制进程失败: %w", err)
    }

    // 5. 保存取消函数
    c.cancelFuncs.Store(task.ID, cancel)

    // 6. 更新任务信息
    task.RecordingFile = outputPath
    task.Status = models.VideoStatusRecording

    return nil
}

// StopRecording 停止录制
func (c *SimpleRecordingCoordinator) StopRecording(taskID uint) error {
    // 1. 获取取消函数
    cancel, ok := c.cancelFuncs.Load(taskID)
    if !ok {
        return errors.New("未找到录制任务")
    }

    // 2. 取消上下文
    cancel.(context.CancelFunc)()

    // 3. 停止FFmpeg进程
    processID := fmt.Sprintf("capture_%d_*", taskID)
    if err := c.orchestrator.StopProcessByPrefix(processID, 10*time.Second); err != nil {
        c.logger.Error("停止录制进程失败",
            zap.Uint("task_id", taskID),
            zap.Error(err),
        )
    }

    // 4. 启动格式转换
    go c.convertRecording(taskID)

    return nil
}

// convertRecording 转换录制格式
func (c *SimpleRecordingCoordinator) convertRecording(taskID uint) {
    // 1. 加载任务
    var task models.VideoRecordingTask
    if err := c.db.First(&task, taskID).Error; err != nil {
        c.logger.Error("加载任务失败", zap.Error(err))
        return
    }

    // 2. 构建输出路径
    dir := filepath.Dir(task.RecordingFile)
    mp4Path := filepath.Join(dir, fmt.Sprintf("recording_%s.mp4",
        time.Now().Format("20060102150405")))

    // 3. 构建转换命令
    args := []string{
        "-i", task.RecordingFile,  // 输入MKV
        "-c", "copy",               // 复制流(不重新编码)
        "-f", "mp4",                // MP4格式
        "-movflags", "+faststart",  // 快速启动
        mp4Path,
    }

    // 4. 启动转换进程
    ctx := context.Background()
    req := &ffmpeg_orchestrator.StartProcessRequest{
        Type:       ffmpeg_orchestrator.ProcessTypeConvert,
        TaskID:     taskID,
        SessionID:  fmt.Sprintf("task_%d", taskID),
        Args:       args,
        OutputPath: mp4Path,
    }

    _, err := c.orchestrator.StartProcess(ctx, req)
    if err != nil {
        c.logger.Error("启动转换进程失败",
            zap.Uint("task_id", taskID),
            zap.Error(err),
        )
        return
    }

    // 5. 等待转换完成
    c.orchestrator.WaitForProcessCompletion(fmt.Sprintf("convert_%d_*", taskID))

    // 6. 更新任务信息
    task.RecordingFile = mp4Path
    c.db.Save(&task)

    // 7. 删除MKV文件
    os.Remove(task.RecordingFile)
}
```

## 五、RTSP流录制

### 5.1 RTSP录制器

```go
// internal/rtsp/recorder.go
type RTSPRecorder struct {
    orchestrator *ffmpeg_orchestrator.FFmpegOrchestrator
    logger       *zap.Logger
    config       *config.Config
}

// StartRecording 启动RTSP录制
func (r *RTSPRecorder) StartRecording(streamURL, outputPath string, duration time.Duration) error {
    // 1. 构建命令
    args := []string{
        "-y",
        "-rtsp_transport", "tcp",  // 使用TCP(更稳定)
        "-i", streamURL,
        "-c", "copy",              // 复制流
        "-f", "mp4",
        "-movflags", "+faststart",
        outputPath,
    }

    if duration > 0 {
        args = append(args, "-t", fmt.Sprintf("%.0f", duration.Seconds()))
    }

    // 2. 启动进程
    ctx := context.Background()
    req := &ffmpeg_orchestrator.StartProcessRequest{
        Type:       ffmpeg_orchestrator.ProcessTypeCapture,
        Args:       args,
        OutputPath: outputPath,
    }

    _, err := r.orchestrator.StartProcess(ctx, req)
    if err != nil {
        return fmt.Errorf("启动RTSP录制失败: %w", err)
    }

    return nil
}
```

## 六、配置管理

### 6.1 FFmpeg配置

```yaml
# config.yaml
ffmpeg:
  # FFmpeg可执行文件路径
  path: "/usr/bin/ffmpeg"

  # 输出目录
  output_dir: "./recordings"

  # 日志目录
  log_dir: "./logs/ffmpeg"

  # 录制配置
  recording:
    # 默认分辨率
    resolution: "1920x1080"

    # 默认帧率
    frame_rate: 30

    # 视频比特率
    video_bitrate: "5M"

    # 音频比特率
    audio_bitrate: "128k"

    # 预设
    preset: "medium"  # ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow

    # 编解码器
    video_codec: "libx264"
    audio_codec: "aac"

    # 像素格式
    pixel_format: "yuv420p"

  # 转换配置
  conversion:
    # 并发转换数
    concurrent_conversions: 3

    # 超时时间
    timeout: 300

  # 进程管理
  process:
    # 最大进程数
    max_processes: 10

    # 停止超时
    stop_timeout: 10

    # 自动重试
    auto_retry: true
    max_retries: 3
```

## 七、相关文档

- [01-系统架构总览.md](./01-系统架构总览.md)
- [03-视频录制任务生命周期.md](./03-视频录制任务生命周期.md)
- [04-华为系统集成详解.md](./04-华为系统集成详解.md)
