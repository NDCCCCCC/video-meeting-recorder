# 视频录制系统改进设计

**日期**: 2026-02-10
**作者**: Claude Code
**状态**: 实施中

## 实施状态

- [x] **Phase 1**: 数据库迁移 + MKV 录制 (已完成)
- [x] **Phase 2**: 转换服务 (已完成)
- [ ] **Phase 3**: HLS 预览服务 (进行中)
- [ ] **Phase 4**: 前端预览组件

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
GET /api/v1/tasks/:id/preview
  - 返回 HLS m3u8 文件
  - 需要认证 (JWT token)
  - 仅任务创建者可访问

GET /api/v1/tasks/:id/preview/segments/:segment
  - 返回 TS 分片文件
  - 需要认证
  - 仅任务创建者可访问
```

### 5.2 权限验证

```go
func (h *TaskHandler) GetHLSPreview(c *gin.Context) {
    taskID := c.Param("id")

    // 获取任务
    task, err := h.service.GetTask(taskID)

    // 验证权限
    claims := jwt.GetClaims(c)
    if task.CreatedBy != claims.UserID {
        c.JSON(403, gin.H{"error": "无权限访问此预览"})
        return
    }

    // 返回 m3u8 内容
    c.File(task.HLSPreviewPath)
}
```

## 6. 前端预览组件

### 6.1 依赖

```bash
npm install react-hls-player
```

### 6.2 组件代码

```tsx
import React, { useState, useEffect } from 'react'
import HlsPlayer from 'react-hls-player'
import { Button, Modal, Alert } from 'antd'
import { PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import axios from 'axios'

interface HLSPreviewProps {
  taskId: number
  taskName: string
}

export const HLSPreview: React.FC<HLSPreviewProps> = ({ taskId, taskName }) => {
  const [visible, setVisible] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>()
  const [hlsUrl, setHlsUrl] = useState<string>()

  const openPreview = async () => {
    setVisible(true)
    setLoading(true)
    setError(undefined)

    try {
      const res = await axios.get(`/api/v1/tasks/${taskId}/preview`)
      setHlsUrl(res.data.playback_url)
    } catch (err: any) {
      setError(err.response?.data?.error || '加载预览失败')
    } finally {
      setLoading(false)
    }
  }

  const handlePlayerError = () => {
    setError('直播连接中断，请刷新重试')
  }

  return (
    <>
      <Button
        icon={<PlayCircleOutlined />}
        onClick={openPreview}
        size="small"
      >
        实时预览
      </Button>

      <Modal
        title={`${taskName} - 实时预览`}
        open={visible}
        onCancel={() => setVisible(false)}
        footer={null}
        width={800}
      >
        {loading && <div className="text-center py-8">加载中...</div>}

        {error && (
          <Alert
            type="error"
            message={error}
            action={
              <Button size="small" onClick={openPreview}>
                <ReloadOutlined /> 刷新
              </Button>
            }
          />
        )}

        {hlsUrl && !error && (
          <HlsPlayer
            src={hlsUrl}
            autoPlay
            controls
            width="100%"
            height={450}
            onError={handlePlayerError}
          />
        )}

        <div className="text-gray-500 text-sm mt-2">
          预览延迟约 3-10 秒
        </div>
      </Modal>
    </>
  )
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
    "react-hls-player": "^3.0.7"
  }
}
```

### 配置更新

```yaml
# config.yaml
recording:
  format: "mkv"
  enable_hls: true
  hls_segment_duration: 10

conversion:
  max_workers: 3
  max_retries: 3

storage:
  recordings_path: "./data/recordings"
  hls_path: "./data/hls"
  cleanup_days: 30
```

### 数据库迁移

在 `internal/migrations/` 创建新迁移文件，执行上述 SQL 变更。

## 10. 实施顺序

1. **Phase 1**: 数据库迁移 + MKV 录制 ✅
2. **Phase 2**: 转换服务 (进行中)
3. **Phase 3**: HLS 预览服务
4. **Phase 4**: 前端预览组件

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

## 附录：相关文件

- `internal/recorder/coordinator.go` - 录制协调器
- `internal/scheduler/video_scheduler.go` - 任务调度器
- `internal/services/conversion_service.go` - 转换服务（新建）
- `internal/handlers/task_handler.go` - 任务处理器
- `frontend/src/pages/tasks/index.tsx` - 任务列表页面
