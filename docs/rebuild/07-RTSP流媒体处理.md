# RTSP流录制处理

## 一、RTSP录制概述

### 1.1 功能概述

RTSP流录制模块负责接收来自外部RTSP源的流媒体并进行录制存储，主要功能：

- **RTSP流接收** - 接受来自网络摄像头、华为终端等的RTSP流
- **流录制** - 将RTSP流录制为MP4/MKV文件
- **健康检查** - 监控RTSP流状态，检测断流
- **自动重连** - 流断开时自动重新连接
- **并发录制** - 支持多路流同时录制

### 1.2 与NVR模块的区别

| 特性 | RTSP流录制 | NVR模块 |
|------|------------|---------|
| **主要用途** | 基础流录制 | 完整硬盘录像系统 |
| **运动检测** | ❌ 不支持 | ✅ 支持 |
| **ONVIF协议** | ❌ 不支持 | ✅ 支持 |
| **PTZ控制** | ❌ 不支持 | ✅ 支持 |
| **设备管理** | 简单 | 完整（NVR设备、摄像头） |
| **存储管理** | 基础 | 智能（循环删除、保护机制） |
| **会议集成** | ✅ 原生支持 | ✅ 支持 |
| **架构设计** | 传统服务层 | DDD领域驱动 |
| **代码复杂度** | 简单 | 复杂（6个阶段，8000+行） |

**使用建议**：
- 如果只需要**基本的RTSP流录制功能**（如接收华为终端的RTSP流），使用 **RTSP流录制模块**
- 如果需要**完整的监控录像系统**（运动检测、ONVIF设备管理等），使用 **NVR模块**

### 1.3 架构设计

```
┌────────────────────────────────────────────────────────────────┐
│                      StreamManager                             │
│                    流管理器 (971行)                             │
│  - AddStream()            添加流                               │
│  - RemoveStream()         移除流                               │
│  - GetStream()            获取流                               │
│  - MonitorStreams()       监控所有流                           │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│                       RTSPRecorder                             │
│                     RTSP录制器 (355行)                          │
│  - StartRecording()       开始录制                             │
│  - StopRecording()        停止录制                             │
│  - GetRecordingStatus()   获取录制状态                         │
│  - MonitorProcess()       监控FFmpeg进程                       │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│                    FFmpegOrchestrator                          │
│                   FFmpeg编排器 (723行)                          │
│  - StartProcess()        启动FFmpeg进程                        │
│  - StopProcess()         停止进程                               │
│  - MonitorProcesses()    监控所有进程                           │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│                       FFmpeg进程                                │
│  - 录制进程             stream_<stream_id>                     │
│  - 转换进程             convert_<stream_id>                   │
└────────────────────────────────────────────────────────────────┘
```

### 1.4 核心组件

| 组件 | 文件 | 行数 | 职责 |
|------|------|------|------|
| StreamManager | stream_manager.go | 971 | 流管理、监控、重连 |
| RTSPRecorder | recorder.go | 355 | 流录制 |
| FFmpegOrchestrator | ffmpeg_orchestrator/orchestrator.go | 723 | FFmpeg进程编排 |

## 二、RTSP流录制实现

### 2.1 StreamManager 流管理器

```go
// internal/rtsp/stream_manager.go
type StreamManager struct {
    streams       map[string]*Stream
    recorders     map[string]*RTSPRecorder
    orchestrator  *ffmpeg_orchestrator.FFmpegOrchestrator
    logger        *zap.Logger
    config        *config.Config
    mu            sync.RWMutex
}

// Stream 流信息
type Stream struct {
    ID            string
    Name          string
    URL           string
    Type          StreamType
    Status        StreamStatus
    StartTime     time.Time
    LastActivity  time.Time
    BytesReceived int64
    ErrorCount    int
    MaxRetries    int
    Recording     *RecordingInfo
}

// StreamType 流类型
type StreamType string

const (
    StreamTypeLive    StreamType = "live"     // 直播流
    StreamTypeRecord  StreamType = "record"   // 录制流
)

// StreamStatus 流状态
type StreamStatus string

const (
    StreamStatusStarting  StreamStatus = "starting"
    StreamStatusRunning   StreamStatus = "running"
    StreamStatusStopping  StreamStatus = "stopping"
    StreamStatusStopped   StreamStatus = "stopped"
    StreamStatusError     StreamStatus = "error"
)

// RecordingInfo 录制信息
type RecordingInfo struct {
    IsRecording    bool
    OutputPath     string
    StartTime      time.Time
    Duration       time.Duration
    FileSize       int64
}

// AddStream 添加流
func (m *StreamManager) AddStream(stream *Stream) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 1. 检查流是否已存在
    if _, exists := m.streams[stream.ID]; exists {
        return fmt.Errorf("流已存在: %s", stream.ID)
    }

    // 2. 初始化流
    stream.Status = StreamStatusStarting
    stream.StartTime = time.Now()
    stream.LastActivity = time.Now()

    // 3. 添加到管理器
    m.streams[stream.ID] = stream

    // 4. 启动流连接
    go m.startStream(stream)

    m.logger.Info("流已添加",
        zap.String("stream_id", stream.ID),
        zap.String("url", stream.URL),
    )

    return nil
}

// startStream 启动流
func (m *StreamManager) startStream(stream *Stream) {
    m.logger.Info("启动流",
        zap.String("stream_id", stream.ID),
        zap.String("url", stream.URL),
    )

    // 1. 连接RTSP流（验证连接）
    if err := m.connectStream(stream); err != nil {
        m.logger.Error("连接流失败",
            zap.String("stream_id", stream.ID),
            zap.Error(err),
        )
        stream.Status = StreamStatusError
        return
    }

    // 2. 更新状态
    stream.Status = StreamStatusRunning

    // 3. 启动健康检查
    go m.monitorStream(stream)

    m.logger.Info("流启动成功",
        zap.String("stream_id", stream.ID),
    )
}

// connectStream 连接并验证流
func (m *StreamManager) connectStream(stream *Stream) error {
    // 使用FFmpeg验证流连接
    cmd := exec.Command(m.config.FFmpeg.Path,
        "-rtsp_transport", "tcp",
        "-i", stream.URL,
        "-t", "3",                 // 只读取3秒验证
        "-f", "null",
        "-",
    )

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("启动FFmpeg失败: %w", err)
    }

    // 等待连接确认
    done := make(chan error, 1)
    go func() {
        done <- cmd.Wait()
    }()

    select {
    case <-time.After(10 * time.Second):
        cmd.Process.Kill()
        return errors.New("连接超时")
    case err := <-done:
        if err != nil {
            return fmt.Errorf("连接失败: %w", err)
        }
    }

    return nil
}

// monitorStream 监控流
func (m *StreamManager) monitorStream(stream *Stream) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // 检查流是否活跃
            if time.Since(stream.LastActivity) > 60*time.Second {
                m.logger.Warn("流不活跃",
                    zap.String("stream_id", stream.ID),
                    zap.Duration("inactive_time", time.Since(stream.LastActivity)),
                )

                // 尝试重连
                go m.reconnectStream(stream)
                return
            }

        case <-stream.Done:
            m.logger.Info("流已停止",
                zap.String("stream_id", stream.ID),
            )
            return
        }
    }
}

// reconnectStream 重连流
func (m *StreamManager) reconnectStream(stream *Stream) {
    m.logger.Info("重连流",
        zap.String("stream_id", stream.ID),
        zap.Int("retry_count", stream.ErrorCount),
    )

    // 1. 停止当前录制
    if stream.Recording != nil && stream.Recording.IsRecording {
        m.StopRecording(stream.ID)
    }

    // 2. 检查重试次数
    if stream.ErrorCount >= stream.MaxRetries {
        m.logger.Error("重连次数超过限制",
            zap.String("stream_id", stream.ID),
            zap.Int("max_retries", stream.MaxRetries),
        )
        stream.Status = StreamStatusError
        return
    }

    // 3. 等待后重试
    time.Sleep(time.Duration(stream.ErrorCount*5) * time.Second)

    // 4. 重新启动
    stream.ErrorCount++
    go m.startStream(stream)
}
```

### 2.2 RTSPRecorder 录制器

```go
// internal/rtsp/recorder.go
type RTSPRecorder struct {
    streamID    string
    outputDir   string
    process     *os.Process
    status      RecordingStatus
    logger      *zap.Logger
    config      *config.Config
    mu          sync.Mutex
    cancelFunc  context.CancelFunc
}

// RecordingStatus 录制状态
type RecordingStatus string

const (
    RecordingStatusIdle      RecordingStatus = "idle"
    RecordingStatusRecording RecordingStatus = "recording"
    RecordingStatusStopping  RecordingStatus = "stopping"
    RecordingStatusError     RecordingStatus = "error"
)

// StartRecording 开始录制
func (r *RTSPRecorder) StartRecording(streamURL, outputDir string, duration time.Duration) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    // 1. 检查状态
    if r.status == RecordingStatusRecording {
        return errors.New("正在录制中")
    }

    // 2. 创建输出目录
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return fmt.Errorf("创建输出目录失败: %w", err)
    }

    // 3. 生成输出文件名
    outputPath := filepath.Join(outputDir,
        fmt.Sprintf("recording_%s.mp4", time.Now().Format("20060102150405")))

    // 4. 构建FFmpeg命令
    args := r.buildFFmpegCommand(streamURL, outputPath, duration)

    // 5. 创建上下文
    ctx, cancel := context.WithCancel(context.Background())
    r.cancelFunc = cancel

    // 6. 启动FFmpeg进程
    cmd := exec.CommandContext(ctx, r.config.FFmpeg.Path, args...)

    // 日志文件
    logPath := filepath.Join(outputDir, "ffmpeg.log")
    logFile, err := os.Create(logPath)
    if err != nil {
        return fmt.Errorf("创建日志文件失败: %w", err)
    }

    cmd.Stdout = logFile
    cmd.Stderr = logFile

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("启动FFmpeg失败: %w", err)
    }

    // 7. 保存进程信息
    r.process = cmd.Process
    r.status = RecordingStatusRecording

    // 8. 监控进程
    go r.monitorProcess(cmd, outputPath)

    r.logger.Info("开始录制",
        zap.String("stream_id", r.streamID),
        zap.String("output_path", outputPath),
    )

    return nil
}

// buildFFmpegCommand 构建FFmpeg命令
func (r *RTSPRecorder) buildFFmpegCommand(streamURL, outputPath string, duration time.Duration) []string {
    args := []string{
        "-y",                    // 覆盖输出文件
        "-rtsp_transport", "tcp", // 使用TCP(更稳定)
        "-i", streamURL,
        "-c", "copy",            // 复制流(不重新编码)
        "-f", "mp4",             // MP4格式
        "-movflags", "+faststart", // 快速启动
        outputPath,
    }

    if duration > 0 {
        args = append(args,
            "-t", fmt.Sprintf("%.0f", duration.Seconds()),
        )
    }

    return args
}

// StopRecording 停止录制
func (r *RTSPRecorder) StopRecording() error {
    r.mu.Lock()
    defer r.mu.Unlock()

    // 1. 检查状态
    if r.status != RecordingStatusRecording {
        return errors.New("未在录制中")
    }

    // 2. 更新状态
    r.status = RecordingStatusStopping

    // 3. 取消上下文
    if r.cancelFunc != nil {
        r.cancelFunc()
    }

    // 4. 等待进程退出
    if r.process != nil {
        done := make(chan error, 1)
        go func() {
            _, err := r.process.Wait()
            done <- err
        }()

        select {
        case <-time.After(10 * time.Second):
            // 超时，强制终止
            r.process.Kill()
        case <-done:
            // 正常退出
        }
    }

    // 5. 更新状态
    r.status = RecordingStatusIdle

    r.logger.Info("停止录制",
        zap.String("stream_id", r.streamID),
    )

    return nil
}

// monitorProcess 监控进程
func (r *RTSPRecorder) monitorProcess(cmd *exec.Cmd, outputPath string) {
    err := cmd.Wait()

    r.mu.Lock()
    defer r.mu.Unlock()

    if err != nil {
        r.status = RecordingStatusError
        r.logger.Error("录制进程异常退出",
            zap.String("stream_id", r.streamID),
            zap.Error(err),
        )
    } else {
        r.status = RecordingStatusIdle
        r.logger.Info("录制完成",
            zap.String("stream_id", r.streamID),
            zap.String("output_path", outputPath),
        )
    }

    // 获取文件信息
    if info, err := os.Stat(outputPath); err == nil {
        r.logger.Info("录制文件信息",
            zap.String("path", outputPath),
            zap.Int64("size", info.Size()),
        )
    }
}
```

### 2.3 使用示例

```go
// 从华为会议获取RTSP流并录制
func RecordConferenceStream(conferenceNumber string, duration time.Duration) error {
    // 1. 获取RTSP流地址
    streamURL, err := huaweiService.GetRTSPStreamURL(conferenceNumber)
    if err != nil {
        return fmt.Errorf("获取RTSP流地址失败: %w", err)
    }

    // 2. 创建流
    stream := &Stream{
        ID:   fmt.Sprintf("conf_%s", conferenceNumber),
        Name: fmt.Sprintf("会议 %s", conferenceNumber),
        URL:  streamURL,
        Type: StreamTypeRecord,
    }

    // 3. 添加流到管理器
    if err := streamManager.AddStream(stream); err != nil {
        return err
    }

    // 4. 开始录制
    recorder := NewRTSPRecorder(stream.ID, logger, config)
    outputDir := filepath.Join(config.Storage.RecordingsDir, stream.ID)

    if err := recorder.StartRecording(streamURL, outputDir, duration); err != nil {
        return err
    }

    // 5. 等待录制完成
    time.Sleep(duration)

    // 6. 停止录制
    return recorder.StopRecording()
}
```

## 三、FFmpeg命令详解

### 3.1 RTSP录制命令

```bash
# 基础RTSP录制命令
ffmpeg -y \
  -rtsp_transport tcp \
  -i "rtsp://10.62.10.3:554/stream1" \
  -c copy \
  -f mp4 \
  -movflags +faststart \
  output.mp4

# 带时长限制的录制
ffmpeg -y \
  -rtsp_transport tcp \
  -i "rtsp://10.62.10.3:554/stream1" \
  -c copy \
  -f mp4 \
  -t 3600 \
  -movflags +faststart \
  output.mp4

# 带重新编码的录制（降低码率）
ffmpeg -y \
  -rtsp_transport tcp \
  -i "rtsp://10.62.10.3:554/stream1" \
  -c:v libx264 \
  -b:v 2000k \
  -c:a aac \
  -b:a 128k \
  -f mp4 \
  -movflags +faststart \
  output.mp4
```

### 3.2 参数说明

| 参数 | 说明 |
|------|------|
| `-y` | 覆盖输出文件 |
| `-rtsp_transport tcp` | 使用TCP传输（更稳定） |
| `-rtsp_transport udp` | 使用UDP传输（低延迟） |
| `-i url` | 输入RTSP流地址 |
| `-c copy` | 复制流，不重新编码 |
| `-c:v libx264` | 视频重新编码为H.264 |
| `-c:a aac` | 音频重新编码为AAC |
| `-b:v 2000k` | 视频比特率2Mbps |
| `-b:a 128k` | 音频比特率128kbps |
| `-t 3600` | 录制3600秒（1小时） |
| `-f mp4` | 输出MP4格式 |
| `-movflags +faststart` | 快速启动（适合流媒体播放） |

## 四、配置管理

### 4.1 RTSP配置

```yaml
# config.yaml
rtsp:
  # 流配置
  transport: "tcp"           # tcp | udp
  connection_timeout: 10      # 连接超时(秒)
  read_timeout: 30            # 读超时(秒)

  # 重连配置
  max_retries: 3              # 最大重试次数
  retry_interval: 5           # 重试间隔(秒)
  reconnect_enabled: true     # 启用自动重连

  # 健康检查
  health_check:
    enabled: true
    interval: 30              # 检查间隔(秒)
    unhealthy_threshold: 3   # 不健康阈值
    healthy_threshold: 2     # 健康阈值

  # 录制配置
  recording:
    enabled: true
    output_dir: "./recordings/rtsp"
    format: "mp4"             # mp4 | mkv
    codec: "copy"             # copy | libx264
    auto_segment: false       # 自动分段
    segment_duration: 300     # 分段时长(秒)

  # 磁盘管理
  storage:
    min_free_space: 1073741824 # 最小剩余空间(1GB)
    auto_clean: true           # 自动清理
    clean_days: 7              # 保留天数
```

## 五、与华为系统集成

### 5.1 获取华为RTSP流

```go
// internal/huawei/service.go
func (s *HuaweiService) GetRTSPStreamURL(conferenceNumber string) (string, error) {
    // 1. 获取会议信息
    info, err := s.GetConferenceInfo(context.Background(), conferenceNumber)
    if err != nil {
        return "", err
    }

    // 2. 检查是否有RTSP流
    if len(info.RTSPStreams) == 0 {
        return "", errors.New("会议没有可用的RTSP流")
    }

    // 3. 返回主码流
    for _, stream := range info.RTSPStreams {
        if stream.Type == "main" {
            return stream.URL, nil
        }
    }

    // 如果没有主码流，返回第一个
    return info.RTSPStreams[0].URL, nil
}
```

### 5.2 会议录制流程

```go
// 会议录制完整流程
func RecordConference(conferenceNumber string, duration time.Duration) error {
    // 1. 获取RTSP流地址
    streamURL, err := huaweiService.GetRTSPStreamURL(conferenceNumber)
    if err != nil {
        return fmt.Errorf("获取RTSP流失败: %w", err)
    }

    // 2. 创建录制器
    recorder := rtsp.NewRTSPRecorder(
        fmt.Sprintf("conf_%s", conferenceNumber),
        logger,
        config,
    )

    // 3. 输出目录
    outputDir := filepath.Join(
        config.Storage.RecordingsDir,
        "conferences",
        conferenceNumber,
        time.Now().Format("2006-01-02"),
    )

    // 4. 开始录制
    if err := recorder.StartRecording(streamURL, outputDir, duration); err != nil {
        return fmt.Errorf("开始录制失败: %w", err)
    }

    // 5. 等待录制完成
    time.Sleep(duration)

    // 6. 停止录制
    if err := recorder.StopRecording(); err != nil {
        return fmt.Errorf("停止录制失败: %w", err)
    }

    return nil
}
```

## 六、相关文档

- [01-系统架构总览.md](./01-系统架构总览.md)
- [05-FFmpeg录制与直播.md](./05-FFmpeg录制与直播.md)
- [04-华为系统集成详解.md](./04-华为系统集成详解.md)

## 七、NVR模块说明

### 7.1 NVR模块已完成

根据项目文档，NVR硬盘录像模块已经**100%完成**，包含以下功能：

| 功能 | 状态 | 说明 |
|------|------|------|
| RTSP流接收 | ✅ | 接受来自网络摄像头的RTSP流输入 |
| ONVIF集成 | ✅ | 通过ONVIF协议管理支持PTZ控制的摄像头 |
| 运动检测录制 | ✅ | 检测画面运动时自动触发录制 |
| 持续录制 | ✅ | 支持全天候持续录制模式 |
| 智能存储管理 | ✅ | 磁盘空间不足时自动删除旧录像 |
| 会议集成 | ✅ | 可与会议系统关联，会议开始时同步录制 |

### 7.2 是否需要保留NVR模块？

**建议保留NVR模块**，原因如下：

1. **功能完整性** - NVR模块提供了运动检测、ONVIF协议、PTZ控制等高级功能
2. **已完成实现** - 模块已经100%完成，包含8000+行代码和完整文档
3. **DDD架构** - 采用领域驱动设计，代码质量高，易于维护
4. **独立使用场景** - 适合作为独立的监控系统使用

**但同时保留RTSP流录制模块**，原因如下：

1. **简单场景** - 对于只需要基础录制的场景（如接收华为终端流），RTSP模块更轻量
2. **快速集成** - RTSP模块与华为系统集成更紧密，无需额外配置
3. **低耦合** - 作为基础功能，可被其他模块复用

**最终建议**：
- **华为会议录制** → 使用 **RTSP流录制模块**（简单直接）
- **专业监控场景** → 使用 **NVR模块**（功能完整）
- 两个模块**共存**，各自服务于不同场景
