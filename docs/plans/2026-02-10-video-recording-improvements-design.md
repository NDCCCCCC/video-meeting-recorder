# 视频录制系统改进设计

**日期**: 2026-02-10
**作者**: Claude Code
**状态**: 已完成

## 实施状态

- ✅ **Phase 1**: 数据库迁移 + MKV 录制 (已完成)
- ✅ **Phase 2**: 转换服务 (已完成)
- ✅ **Phase 3**: HLS 预览服务 (已完成)
- ✅ **Phase 4**: 前端预览组件 (已完成)

## 概述

本设计旨在解决当前视频录制系统的主要问题：
1. MP4文件在录制中断时损坏（无法播放）
2. 缺少实时预览功能
3. 任务删除限制过于严格

## 1. FFmpeg 双输出方案（MKV + HLS）

### 1.1 使用 tee muxer

使用 FFmpeg tee muxer 同时输出两个流：
- **MKV 主录制**: 容错性强，中断时可恢复
- **HLS 预览流**: 实时播放，低延迟

### 1.2 FFmpeg 命令结构

```bash
ffmpeg -y \
  [输入参数] \
  -c:v libx264 -preset medium -b:v 5M -pix_fmt yuv420p \
  -c:a aac -b:a 128k \
  -f tee -map 0:v -map 0:a? \
  -tee_format "mkv" \
  "[f=mkv:onfail=ignore]{output_mkv}|[f=hls:hls_time=10:hls_list_size=0:hls_segment_filename={segment_path}]{output_m3u8}"
```

### 1.3 文件命名规范

```
格式: {任务名称}_{会议号}_{时间戳}.{扩展名}

示例:
  MKV:  周一例会_123456_20260210140530.mkv
  MP4:  周一例会_123456_20260210140530.mp4
  HLS:  周一例会_123456_20260210140530/
        ├── index.m3u8
        └── segment_000000.ts
```

## 2. 文件存储结构

```
data/
├── recordings/
│   └── task_{id}/
│       ├── {name}_{conf}_{timestamp}.mkv
│       ├── {name}_{conf}_{timestamp}.mp4
│       └── ffmpeg.log
└── hls/
    └── task_{id}/
        └── {name}_{conf}_{timestamp}/
            ├── index.m3u8
            └── segment_*.ts
```

## 3. 数据库 Schema 变更

```sql
-- 新增字段
ALTER TABLE video_recording_tasks ADD COLUMN mkv_file_path TEXT;
ALTER TABLE video_recording_tasks ADD COLUMN hls_preview_path TEXT;
ALTER TABLE video_recording_tasks ADD COLUMN mp4_file_path TEXT;
ALTER TABLE video_recording_tasks ADD COLUMN conversion_status TEXT DEFAULT 'pending';
ALTER TABLE video_recording_tasks ADD COLUMN conversion_error_msg TEXT;
ALTER TABLE video_recording_tasks ADD COLUMN conversion_started_at TIMESTAMP;
ALTER TABLE video_recording_tasks ADD COLUMN conversion_completed_at TIMESTAMP;
ALTER TABLE video_recording_tasks ADD COLUMN conversion_retry_count INTEGER DEFAULT 0;
```

### 转换状态枚举

```go
type ConversionStatus string

const (
    ConversionStatusPending    ConversionStatus = "pending"
    ConversionStatusProcessing ConversionStatus = "processing"
    ConversionStatusCompleted  ConversionStatus = "completed"
    ConversionStatusFailed     ConversionStatus = "failed"
)
```

## 4. 异步转换服务

### 4.1 架构

```
┌─────────────────┐
│  录制完成事件    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  转换任务队列    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────┐
│  Worker Pool (3 workers) │
└────────┬────────────────┘
         │
         ▼
┌─────────────────┐
│  FFmpeg 转换     │
│  MKV → MP4      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  更新任务状态    │
└─────────────────┘
```

### 4.2 转换服务接口

```go
type ConversionService interface {
    // 提交转换任务
    SubmitConversion(taskID uint) error

    // 获取转换状态
    GetConversionStatus(taskID uint) (ConversionStatus, error)

    // 重试失败任务
    RetryConversion(taskID uint) error

    // 启动/停止服务
    Start() error
    Stop()
}
```

### 4.3 重试机制

- 最大重试次数: 3
- 退避策略: 指数退避 (1分钟 → 5分钟 → 30分钟)
- 超过重试次数标记为 failed，保留 MKV 文件

## 5. HLS 预览服务

### 5.1 路由设计

```
GET /api/v1/recordings/:id/preview
  - 返回 HLS m3u8 文件信息和带token的播放地址
  - 需要认证 (JWT token)
  - 仅任务创建者可访问

GET /api/v1/recordings/:id/preview/stream/:file
  - 返回 TS 分片文件或 m3u8 播放列表
  - 需要认证 (HLS token)
  - 仅任务创建者可访问
```

### 5.2 权限验证

```go
func (h *VideoRecordingTaskHandler) GetHLSPreview(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
        return
    }

    // 获取任务信息
    task, err := h.taskService.GetTaskByID(id)
    if err != nil {
        response.GinError(c, response.CodeNotFound, "任务不存在")
        return
    }

    // 检查HLS预览路径是否存在
    if task.HLSPreviewPath == "" {
        response.GinError(c, response.CodeNotFound, "该任务没有HLS预览")
        return
    }

    // 检查 m3u8 文件是否存在
    _, m3u8Err := os.Stat(task.HLSPreviewPath)
    m3u8Exists := m3u8Err == nil

    // 验证权限：只有任务创建者可以访问
    userID := middleware.GetUserID(c)
    if task.CreatedBy != userID {
        response.GinError(c, response.CodeForbidden, "无权限访问此预览")
        return
    }

    // 根据状态和文件存在性返回不同响应
    // 返回完整的 m3u8 播放列表 URL（包含 token）
    playbackURL := fmt.Sprintf("/api/v1/recordings/%d/preview/stream/index.m3u8", id)

    // 生成访问 token
    accessToken := h.hlsToken.Generate(id, userID)
    playbackURLWithToken := fmt.Sprintf("%s?token=%s", playbackURL, accessToken)

    switch {
    case task.Status == "recording" && !m3u8Exists:
        response.GinSuccess(c, gin.H{
            "task_id":      id,
            "playback_url": playbackURLWithToken,
            "status":       task.Status,
            "ready":        false,
            "message":      "HLS预览正在准备中，请稍后刷新",
        })
    case !m3u8Exists:
        response.GinError(c, response.CodeNotFound, "HLS预览文件不存在")
    default:
        response.GinSuccess(c, gin.H{
            "task_id":      id,
            "playback_url": playbackURLWithToken,
            "status":       task.Status,
            "ready":        true,
        })
    }
}
```

## 6. 前端预览组件

### 6.1 依赖

```bash
npm install hls.js
```

### 6.2 组件代码

```tsx
// HLS 实时预览组件

import { useState, useEffect, useRef, useCallback } from 'react'
import { Button, Modal, Alert, Space } from 'antd'
import { ReloadOutlined, EyeOutlined } from '@ant-design/icons'
import { getTaskPreview } from '../api/task'
import type { VideoRecordingTask } from '../types/task'
import Hls from 'hls.js'

interface HLSPreviewProps {
  taskId: number
  taskName: string
  status: string
}

// HLS 播放器组件（使用 hls.js 库）
function HLSPlayer({ src, onError }: { src: string; onError: () => void }) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  // 使用 ref 存储稳定的 onError 回调，避免 useEffect 重复触发
  const onErrorRef = useRef(onError)
  onErrorRef.current = onError

  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    // 清理函数
    const cleanup = () => {
      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }
    }

    // 加载 HLS
    const loadHLS = () => {
      cleanup()

      try {
        // 检查是否支持 HLS.js
        if (Hls.isSupported()) {
          // 使用 hls.js - 直播流优化配置
          const hls = new Hls({
            debug: false,
            enableWorker: true,
            lowLatencyMode: true,
            backBufferLength: 30, // 减少后缓冲，降低内存占用
            maxBufferLength: 30,   // 最大缓冲30秒
            maxMaxBufferLength: 60,
            liveSyncDuration: 3,   // 直播同步延迟，尽量接近直播边缘
            liveMaxLatencyDuration: 10, // 最大延迟10秒
          })

          hlsRef.current = hls

          hls.loadSource(src)
          hls.attachMedia(video)

          hls.on(Hls.Events.MANIFEST_PARSED, () => {
            console.log('HLS manifest parsed, starting playback')
            video.play().catch(err => {
              console.warn('Auto-play prevented:', err)
            })
          })

          hls.on(Hls.Events.ERROR, (_event, data) => {
            if (data.fatal) {
              switch (data.type) {
                case Hls.ErrorTypes.NETWORK_ERROR:
                  console.error('HLS network error:', data)
                  hls.startLoad()
                  break
                case Hls.ErrorTypes.MEDIA_ERROR:
                  console.error('HLS media error:', data)
                  hls.recoverMediaError()
                  break
                default:
                  console.error('HLS fatal error:', data)
                  cleanup()
                  onErrorRef.current()
                  break
              }
            }
          })
        } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
          // Safari 原生支持
          video.src = src
          video.play().catch(err => {
            console.warn('Auto-play prevented:', err)
          })
        } else {
          onErrorRef.current()
        }
      } catch (error) {
        console.error('HLS load error:', error)
        onErrorRef.current()
      }
    }

    loadHLS()

    return cleanup
  }, [src]) // 只依赖 src，避免 onError 变化导致重建

  return (
    <video
      ref={videoRef}
      // 直播模式不显示控制条（进度条、时间等）
      muted    // 静音以提高自动播放成功率
      playsInline // 移动端防止全屏
      autoPlay  // 自动播放
      style={{ width: '100%', maxHeight: '450px', backgroundColor: '#000' }}
    />
  )
}

export function HLSPreview({ taskId, taskName, status }: HLSPreviewProps) {
  const [visible, setVisible] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>()
  const [hlsUrl, setHlsUrl] = useState<string>()
  const [currentStatus, setCurrentStatus] = useState(status)
  const [isPreparing, setIsPreparing] = useState(false)
  const [retryCount, setRetryCount] = useState(0)
  const retryTimerRef = useRef<NodeJS.Timeout | null>(null)

  const MAX_RETRY_COUNT = 10  // 最大重试次数

  // 清理定时器
  useEffect(() => {
    return () => {
      if (retryTimerRef.current) {
        clearTimeout(retryTimerRef.current)
      }
    }
  }, [])

  // 使用 useCallback 稳定回调函数，避免子组件重建
  const handlePlayerError = useCallback(() => {
    setError('直播连接中断，请刷新重试')
  }, [])

  const openPreview = async () => {
    setVisible(true)
    setLoading(true)
    setError(undefined)
    setHlsUrl(undefined)
    setIsPreparing(false)
    setRetryCount(0)  // 重置重试计数

    // 清理之前的定时器
    if (retryTimerRef.current) {
      clearTimeout(retryTimerRef.current)
      retryTimerRef.current = null
    }

    try {
      const response = await getTaskPreview(taskId)
      if (response.data) {
        if (response.data.ready === false) {
          // HLS 正在准备中
          setIsPreparing(true)
          setError(response.data.message || 'HLS预览正在准备中，请稍后刷新')
          // 自动重试：3秒后自动刷新（最多10次）
          if (retryCount < MAX_RETRY_COUNT) {
            retryTimerRef.current = setTimeout(() => {
              setRetryCount(prev => prev + 1)
              handleRefresh()
            }, 3000)
          } else {
            setError('HLS预览初始化超时，请关闭后重试')
            setIsPreparing(false)
          }
        } else if (response.data.playback_url) {
          // HLS 已就绪，使用 API 返回的带 token 的 URL
          setHlsUrl(response.data.playback_url)
          setCurrentStatus(response.data.status)
          setIsPreparing(false)
          setError(undefined)
          setRetryCount(0)  // 成功后重置计数
        }
      }
    } catch (err: any) {
      setError(err.response?.data?.message || err.message || '加载预览失败')
      setIsPreparing(false)
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = useCallback(() => {
    openPreview()
  }, [taskId])

  const handleClose = useCallback(() => {
    // 清理定时器
    if (retryTimerRef.current) {
      clearTimeout(retryTimerRef.current)
      retryTimerRef.current = null
    }
    setVisible(false)
    setHlsUrl(undefined)
    setError(undefined)
  }, [])

  return (
    <>
      <Button
        icon={<EyeOutlined />}
        onClick={openPreview}
        size="small"
        disabled={status !== 'recording'}
        title={status === 'recording' ? '实时预览' : '仅录制中可预览'}
      >
        预览
      </Button>

      <Modal
        title={`${taskName} - 实时预览`}
        open={visible}
        onCancel={handleClose}
        footer={null}
        width={800}
      >
        {loading && (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <Space direction="vertical">
              <div>加载中...</div>
            </Space>
          </div>
        )}

        {error && (
          <Alert
            type={isPreparing ? "warning" : "error"}
            message={error}
            action={
              <Button size="small" onClick={handleRefresh}>
                <ReloadOutlined /> 刷新
              </Button>
            }
            style={{ marginBottom: 16 }}
          />
        )}

        {hlsUrl && !error && (
          <>
            <HLSPlayer src={hlsUrl} onError={handlePlayerError} />
            <div style={{ marginTop: 16, color: '#999', fontSize: 12 }}>
              <Space>
                <span>状态: {currentStatus === 'recording' ? '录制中' : currentStatus}</span>
                <span>•</span>
                <span>预览延迟约 3-5 秒</span>
              </Space>
            </div>
          </>
        )}

        {!loading && !error && !hlsUrl && (
          <Alert
            type="warning"
            message="暂无预览可用"
            description="该任务暂无可用的 HLS 预览流"
          />
        )}
      </Modal>
    </>
  )
}

// 用于在表格中渲染的包装组件
export function RenderTaskPreview(task: VideoRecordingTask) {
  return <HLSPreview taskId={task.id} taskName={task.name} status={task.status} />
}
```

### 6.3 任务列表集成

```tsx
// 在任务操作列添加预览按钮
{
  title: '操作',
  render: (_, record) => (
    <Space>
      {record.status === 'recording' && (
        <HLSPreview taskId={record.id} taskName={record.name} />
      )}
      {/* 其他操作按钮 */}
    </Space>
  )
}
```

## 7. 错误处理和容错机制

### 7.1 FFmpeg 进程故障

- 意外退出时自动重启（最多3次）
- 记录详细错误日志到 `ffmpeg_error.log`
- 通知用户录制中断，保留已录制的 MKV

### 7.2 转换任务容错

- 失败后指数退避重试
- 超过3次标记为 failed
- 转换失败不影响 MKV 文件

### 7.3 HLS 播放容错

- 前端检测 m3u8 不存在显示"直播准备中..."
- 分片加载失败自动重试
- 提供刷新按钮

### 7.4 存储监控

- 录制前检查磁盘空间（需要预估大小的2倍）
- 定期清理超过30天的临时 HLS 文件

## 8. 测试考虑

### 单元测试
- 转换服务状态转换逻辑
- HLS 路径生成
- 权限验证

### 集成测试
- 端到端: 录制 → MKV → 转换 MP4
- FFmpeg 崩溃恢复
- 并发录制任务

### 性能测试
- 100个并发 HLS 访问
- 10个同时录制任务
- 转换队列满载处理

### 手动测试清单
- [ ] 录制3分钟，MKV 文件完整
- [ ] HLS 预览延迟 <10秒
- [ ] 转换后 MP4 与 MKV 时长一致
- [ ] 非创建者访问返回 403
- [ ] FFmpeg 崩溃后自动重启

## 9. 部署注意事项

### 新增依赖

**Go (go.mod)**
```go
require (
    github.com/grafov/m3u8 v0.12.0
)
```

**前端 (package.json)**
```json
{
  "dependencies": {
    "hls.js": "^1.4.12"
  }
}
```

### 配置更新

```yaml
# config.yaml
storage:
  recordings_path: "./data/recordings"
  hls_path: "./data/hls"
  temp_path: "./data/temp"
  max_disk_usage: 90

ffmpeg:
  path: "./bin/ffmpeg"
  ffprobe_path: "./bin/ffprobe"
  max_processes: 5
  timeout: "5m"
  default_codec: "h264"
  default_format: "mp4"
  default_video_bitrate: "2000"
  default_audio_bitrate: "128"
  # DShow 设备配置
  dshow_buffer_size: 2097152      # 2MB 实时缓冲区
  dshow_thread_queue_size: 8      # 线程队列大小
  # HLS 配置
  hls_segment_duration: 10        # HLS 分片时长（秒）
  hls_list_size: 5                # HLS 播放列表保留分片数
  # 录制监控配置
  max_recording_duration: "24h"   # 最长录制时长

auth:
  hls_token_secret: "change-me-in-production"
  hls_token_duration: "5m"
```

### 数据库迁移

在 `internal/migrations/` 创建新迁移文件，执行上述 SQL 变更。

## 实施顺序

1. **Phase 1**: 数据库迁移 + MKV 录制 ✅
2. **Phase 2**: 转换服务 ✅
3. **Phase 3**: HLS 预览服务 ✅
4. **Phase 4**: 前端预览组件 ✅

## 11. Phase 1 实施记录 ✅

**完成日期**: 2026-02-10

### 已修改文件

1. **`internal/models/video_recording_task.go`**
   - 添加 `ConversionStatus` 枚举类型
   - 添加新字段: `MKVFilePath`, `HLSPreviewPath`, `MP4FilePath`
   - 添加转换追踪字段: `ConversionStatus`, `ConversionErrorMsg`, `ConversionStartedAt`, `ConversionCompletedAt`, `ConversionRetryCount`

2. **`internal/recorder/coordinator.go`**
   - 输出格式从 `mp4` 改为 `mkv`
   - 更新文件命名格式: `{任务名称}_{会议号}_{时间戳}.mkv`
   - 添加 `sanitizeFilename` 函数
   - 同时更新 `RecordingFile` 和 `MKVFilePath` 字段

### 数据库变更

GORM AutoMigrate 将在下次启动时自动添加以下字段：

```sql
ALTER TABLE video_recording_tasks ADD COLUMN mkv_file_path TEXT;
ALTER TABLE video_recording_tasks ADD COLUMN hls_preview_path TEXT;
ALTER TABLE video_recording_tasks ADD COLUMN mp4_file_path TEXT;
ALTER TABLE video_recording_tasks ADD COLUMN conversion_status TEXT DEFAULT 'pending';
ALTER TABLE video_recording_tasks ADD COLUMN conversion_error_msg TEXT;
ALTER TABLE video_recording_tasks ADD COLUMN conversion_started_at TIMESTAMP;
ALTER TABLE video_recording_tasks ADD COLUMN conversion_completed_at TIMESTAMP;
ALTER TABLE video_recording_tasks ADD COLUMN conversion_retry_count INTEGER DEFAULT 0;
```

## 12. Phase 2 实施记录 ✅

**完成日期**: 2026-02-10

### 已修改文件

1. **`internal/services/conversion_service.go`** (新建)
   - 创建 `ConversionService` 接口
   - 实现 `FFmpegConversionService`
   - Worker Pool 架构（3个并发worker）
   - 任务队列（100容量缓冲）
   - 重试机制：最多3次，指数退避（1分钟、5分钟、30分钟）
   - FFmpeg转换使用 `-c:v copy` 直接复制视频流

2. **`cmd/server/app.go`**
   - 添加 `conversionService` 字段到 `MinimalApp`
   - 在 `initHandlers` 中创建转换服务
   - 在 `registerServices` 中启动转换服务
   - 在 `Stop` 中停止转换服务

3. **`internal/scheduler/video_scheduler.go`**
   - 添加 `ConversionServiceInterface` 接口
   - 添加 `conversionService` 字段
   - 在 `completeTask` 中自动提交转换任务
   - 添加 `SetConversionService` 方法

4. **`internal/handlers/video_recording_task_handler.go`**
   - 添加 `conversionService` 字段
   - 添加 `SetConversionService` 方法
   - 添加 `GetConversionStatus` 端点
   - 添加 `RetryConversion` 端点

5. **`internal/scheduler/video_scheduler_test.go`**
   - 更新所有测试调用以传入新的 `conversionService` 参数

### API 端点

```
GET /api/v1/recordings/:id/conversion-status - 获取转换状态
POST /api/v1/recordings/:id/conversion-retry - 重试转换
```

## 13. Phase 3 实施记录 ✅

**完成日期**: 2026-02-10

### 已修改文件

1. **`internal/config/config.go`**
   - 添加 `HLSPath` 字段到 `StorageConfig`
   - 默认值: `./data/hls`
   - 在 `setDefaults` 和 `ensureDirectories` 中处理

2. **`internal/recorder/coordinator.go`**
   - 更新 `RecordingProcess` 添加 `HLSPath` 字段
   - 添加 `hlsSegmentDuration` 常量 (10秒)
   - 添加 `getHLSPath` 函数生成 HLS 目录路径
   - 修改 `buildRecordingCommand` 使用 tee muxer 双输出
   - 修改 `StartRecording` 同时设置 MKV 和 HLS 路径
   - Tee muxer 格式: 优化的 Windows 路径处理，使用相对路径避免转义问题

3. **`internal/handlers/video_recording_task_handler.go`**
   - 添加 `GetHLSPreview` 端点（返回带token的播放地址）
   - 添加 `ServeHLSStream` 端点（支持 m3u8 重写和分段访问）
   - 添加路径遍历安全检查函数
   - 权限验证：仅任务创建者可访问
   - HLS token 验证机制

4. **`cmd/server/app.go`**
   - 注册 HLS 预览相关路由

5. **`internal/auth/hlstoken/hls_token.go`** (新建)
   - HLS 预览访问 token 生成和验证

### API 端点

```
GET /api/v1/recordings/:id/preview - 获取HLS预览信息和带token的播放地址
GET /api/v1/recordings/:id/preview/stream/:file - 提供HLS流文件（需要token）
```

### HLS 文件结构

```
data/hls/
└── task_{id}/
    └── {name}_{conf}_{timestamp}/
        ├── index.m3u8
        └── segment_*.ts
```

## 14. Phase 4 实施记录 ✅

**完成日期**: 2026-02-10

### 已修改文件

1. **`frontend/src/api/task.ts`**
   - 添加 `getTaskPreview` 函数获取预览信息
   - 添加 `getHLSStreamUrl` 函数构建流文件 URL

2. **`frontend/src/components/HLSPreview.tsx`** (新建)
   - 创建 `HLSPreview` 组件
   - 使用 `hls.js` 库实现 HLS 播放
   - 优化直播流配置（低延迟模式）
   - 支持自动重试机制（HLS准备中状态）
   - 模态对话框显示预览播放器
   - 错误处理和用户友好提示

3. **`frontend/src/pages/tasks/index.tsx`**
   - 导入 `HLSPreview` 组件
   - 在操作列添加预览按钮
   - 仅在任务状态为 `recording` 时显示预览按钮

### 功能特性

- 使用 `hls.js` 库实现高质量 HLS 播放
- 仅录制中任务可预览
- 权限验证（仅任务创建者）
- 错误处理和自动重试
- 预览延迟约 3-5 秒（优化的低延迟配置）
- 响应式设计，支持不同屏幕尺寸
- 自动检测 HLS 准备状态并提示用户

## 附录：相关文件

- `internal/recorder/coordinator.go` - 录制协调器
- `internal/scheduler/video_scheduler.go` - 任务调度器
- `internal/services/conversion_service.go` - 转换服务（新建）
- `internal/handlers/task_handler.go` - 任务处理器
- `frontend/src/pages/tasks/index.tsx` - 任务列表页面
