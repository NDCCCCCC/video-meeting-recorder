# Phase 01: 在视频播放中添加外挂字幕支持 - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning

<domain>
## Phase Boundary

为三个页面的视频播放添加外挂字幕功能：
1. 预览视频页面（VideoPlayerModal组件）
2. 切割视频页面
3. 预览PPT页面（PPTPreview组件中的视频）

字幕显示在视频下方独立区域，不使用HTML5 TextTrack覆盖层。
</domain>

<decisions>
## Implementation Decisions

### 字幕格式
- **D-01:** 使用 WebVTT 格式（HTML5原生支持，无需额外解析库）
- **D-02:** 字幕内容由单独的字幕生成服务创建（技术方案后续研究决定）

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
- 字幕生成服务的具体技术选型（可使用现有通义听悟API或其他ASR服务）
- 字幕文件路径管理（与视频文件的关联方式）
- 前端字幕组件的具体实现方式
- 默认字幕样式（建议白色文字+黑色半透明背景）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Video Player Components
- `frontend/src/components/VideoPlayerModal.tsx` — 现有视频播放器组件（控制条、状态管理、快捷键）
- `frontend/src/components/PPTPreview.tsx` — PPT预览组件（包含视频播放）
- `frontend/src/components/VideoPlayerSimple.tsx` — 简单播放器组件

### Project Context
- `.planning/PROJECT.md` — 项目架构和约束（Go 1.24 + React 19 + SQLite + FFmpeg）

### External Specifications
- [WebVTT格式规范](https://developer.mozilla.org/en-US/docs/Web/API/WebVTT_API) — 字幕文件格式标准
- [HTML5 Track元素](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/track) — 原生字幕支持（参考，但不使用覆盖层）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **VideoPlayerModal组件**: 自定义控制条、播放状态管理、键盘快捷键系统
- **useKeyboardShortcuts hook**: 键盘事件处理模式
- **Ant Design组件**: Slider, Button, Space等UI组件

### Established Patterns
- 视频URL通过API获取（带Token认证）
- 状态使用useState + useCallback模式
- 样式使用内联style对象常量（STYLES常量）
- 模态框使用Ant Design Modal组件

### Integration Points
- **API客户端**: `frontend/src/api/apiClient.ts` — 添加字幕文件URL获取API
- **视频播放组件**: 需要在VideoPlayerModal中添加字幕区域
- **PPT预览组件**: 需要在PPTPreview中添加字幕区域
- **文件服务**: 后端需要提供字幕文件的HTTP访问接口（带Token认证）

### Creative Options
- 字幕区域可以作为一个独立的SubtitlePanel组件，被VideoPlayerModal和PPTPreview复用
- 可以使用TanStack Query缓存字幕文件内容
- 字幕文件可以与视频文件一起打包下载

</code_context>

<specifics>
## Specific Ideas

- 字幕显示在视频下方，高度自适应，最大高度不超过视频容器的30%
- 字幕生成服务可以复用现有的通义听悟API（需确认是否返回时间轴数据）
- 字幕文件命名：与视频文件同名但扩展名为.vtt（如 video.mp4 → video.vtt）
- 后端提供字幕文件存在性检查API，避免前端404错误
- 字幕控制按钮集成到现有控制条中（在下载按钮旁边）

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-ppt*
*Context gathered: 2026-05-12*
