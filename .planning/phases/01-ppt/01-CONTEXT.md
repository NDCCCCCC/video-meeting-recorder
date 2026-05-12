# Phase 01: 在视频播放中添加外挂字幕支持 - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning
**Updated:** 2026-05-12 (本地字幕生成方案决策)

<domain>
## Phase Boundary

为三个页面的视频播放添加外挂字幕功能：
1. 预览视频页面（VideoPlayerModal组件）
2. 切割视频页面
3. 预览PPT页面（PPTPreview组件中的视频）

字幕显示在视频下方独立区域，不使用HTML5 TextTrack覆盖层。
新增本地字幕生成功能，使用 Sherpa-ONNX + Paraformer 模型实现中文语音识别。
</domain>

<decisions>
## Implementation Decisions

### 字幕格式
- **D-01:** 使用 WebVTT 格式（HTML5原生支持，无需额外解析库）
- **D-02:** 使用 Sherpa-ONNX + Paraformer-zh-small 本地模型进行中文语音识别（不使用云服务）

### 字幕生成（本地 ASR）
- **D-12:** Sherpa-ONNX 模型文件存储在服务器本地路径（默认 `/models/sherpa`，支持环境变量 `SHERPA_MODEL_PATH` 覆盖）
- **D-13:** 新视频创建后自动触发字幕生成（包括：上传视频完成、录制任务完成）
- **D-14:** 字幕生成采用异步任务处理，完成后通过通知系统告知用户
- **D-15:** 使用 FFmpeg 提取音频：MP4 → WAV 16kHz 单声道（Sherpa-ONNX 输入要求）
- **D-16:** 字幕生成任务队列机制，限制并发数量（建议最多 2 个并发任务）

### 字幕存储
- **D-03:** 字幕文件存储在视频文件同目录（命名：{视频文件名}.vtt）
- **D-04:** API返回字幕文件URL，前端通过HTTP获取并渲染到独立区域

### UI与交互
- **D-05:** 字幕显示在视频下方独立区域（非视频覆盖层）
- **D-06:** 提供字幕开关按钮（显示/隐藏字幕区域）
- **D-07:** 支持字号调整（小/中/大）
- **D-08:** 支持位置选择（虽然默认下方，但可扩展为顶部/居中）
- **D-09:** 支持样式设置（文字颜色、背景颜色、描边等）

### 同步与时间控制
- **D-10:** 字幕根据WebVTT时间轴自动同步显示
- **D-11:** 无需手动延迟调整（简化实现）

### Claude's Discretion
以下方面由Claude自行决定：
- Sherpa-ONNX Go 集成的具体实现细节
- 任务队列的实现方式（内存队列 vs 数据库队列）
- 字幕生成失败的重试策略（建议最多重试 1 次）
- 用户字幕样式配置的持久化方式（建议使用 localStorage）
- 字幕生成进度提示的文案和展示方式
- 默认字幕样式（建议白色文字+黑色半透明背景）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Sherpa-ONNX 官方文档
- https://k2-fsa.github.io/sherpa/onnx/go-api/ — Sherpa-ONNX Go API 官方文档
- https://k2-fsa.github.io/sherpa/onnx/pretrained_models/online-transducer/ — Paraformer 预训练模型下载
- https://github.com/k2-fsa/sherpa-onnx-go — Sherpa-ONNX Go 包源码

### Video Player Components
- `frontend/src/components/VideoPlayerModal.tsx` — 现有视频播放器组件（控制条、状态管理、快捷键）
- `frontend/src/components/PPTPreview.tsx` — PPT预览组件（包含视频播放）
- `frontend/src/components/VideoPlayerSimple.tsx` — 简单播放器组件

### Project Context
- `.planning/PROJECT.md` — 项目架构和约束（Go 1.24 + React 19 + SQLite + FFmpeg）

### External Specifications
- [WebVTT格式规范](https://developer.mozilla.org/en-US/docs/Web/API/WebVTT_API) — 字幕文件格式标准
- [FFmpeg 音频转换文档](https://ffmpeg.org/documentation.html) — 音频格式转换参考

### UI Specification
- `.planning/phases/01-ppt/01-UI-SPEC.md` — UI 设计契约（36项锁定需求，MUST READ）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **VideoPlayerModal组件**: 自定义控制条、播放状态管理、键盘快捷键系统
- **useKeyboardShortcuts hook**: 键盘事件处理模式
- **Ant Design组件**: Slider, Button, Space, ColorPicker 等UI组件
- **通知系统**: 现有的消息通知机制（`message` from Ant Design）

### Established Patterns
- 视频URL通过API获取（带Token认证）
- 状态使用useState + useCallback模式
- 样式使用内联style对象常量（STYLES常量）
- 模态框使用Ant Design Modal组件
- 异步任务处理模式（参考现有的转录任务）

### Integration Points
- **API客户端**: `frontend/src/api/apiClient.ts` — 添加字幕文件URL获取API、字幕生成触发API
- **视频播放组件**: 需要在VideoPlayerModal中添加字幕区域
- **PPT预览组件**: 需要在PPTPreview中添加字幕区域
- **文件服务**: 后端需要提供字幕文件的HTTP访问接口（带Token认证）
- **录制服务**: `internal/services/video_recording_task_service.go` — 录制完成后触发字幕生成
- **上传服务**: 文件上传完成后触发字幕生成
- **通知系统**: 字幕生成完成后通知用户

### Creative Options
- 字幕区域可以作为一个独立的SubtitlePanel组件，被VideoPlayerModal和PPTPreview复用
- 可以使用TanStack Query缓存字幕文件内容
- 字幕文件可以与视频文件一起打包下载
- Sherpa-ONNX 可以作为独立的 Go service 封装，便于测试和替换

</code_context>

<specifics>
## Specific Ideas

### 本地 ASR 集成
- **模型**: Paraformer-zh-small (~40MB)，专门优化中文识别
- **内存占用**: < 500MB，适合资源受限环境
- **处理速度**: 约 0.5x 实时（1分钟音频约需2分钟处理）
- **音频预处理**: FFmpeg 命令 `ffmpeg -i input.mp4 -ar 16000 -ac 1 output.wav`
- **输出格式**: Sherpa-ONNX 返回带时间戳的文本，需格式化为 WebVTT

### 字幕显示
- 字幕显示在视频下方，高度自适应，最大高度不超过视频容器的30%
- 字幕文件命名：与视频文件同名但扩展名为.vtt（如 video.mp4 → video.vtt）
- 后端提供字幕文件存在性检查API，避免前端404错误
- 字幕控制按钮集成到现有控制条中（在下载按钮旁边）

### 任务触发场景
1. **上传视频**: 文件上传成功、转码完成后自动触发
2. **录制视频**: 录制任务完成、视频文件保存后自动触发

</specifics>

<deferred>
## Deferred Ideas

- 实时字幕生成（边录边生成）— 需要 GPU 加速，资源消耗大
- 字幕编辑功能 — 属于独立功能模块，后续考虑
- 多语言字幕 — 当前仅支持中文，其他语言需要不同模型
- 字幕翻译功能 — 依赖外部 API 或额外模型

</deferred>

---

*Phase: 01-ppt*
*Context gathered: 2026-05-12*
*Context updated: 2026-05-12 (本地字幕生成方案)*
