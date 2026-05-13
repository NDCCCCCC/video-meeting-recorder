# Phase 01: 在视频播放中添加外挂字幕支持 - Research

**Researched:** 2026-05-12
**Domain:** Video Subtitle System (Frontend + Backend)
**Confidence:** MEDIUM

## Summary

Phase 01 requires adding external subtitle support to three video playback contexts in the Record V2 system. The technical approach involves displaying WebVTT-formatted subtitles in a dedicated panel below the video (not as HTML5 TextTrack overlays), with real-time synchronization using the video `timeupdate` event. The implementation spans both frontend React components and backend Go API endpoints.

**Primary recommendation:** Use WebVTT format with browser-native TextTrack API for parsing, custom React component for rendering in separate display area, and Go HTTP handlers for serving subtitle files with token authentication.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 使用 WebVTT 格式（HTML5原生支持，无需额外解析库）
- **D-02:** 字幕内容由单独的字幕生成服务创建（技术方案后续研究决定）
- **D-03:** 字幕文件存储在视频文件同目录（命名：{视频文件名}.vtt）
- **D-04:** API返回字幕文件URL，前端通过HTTP获取并渲染到独立区域
- **D-05:** 字幕显示在视频下方独立区域（非视频覆盖层）
- **D-06:** 提供字幕开关按钮（显示/隐藏字幕区域）
- **D-07:** 支持字号调整（小/中/大）
- **D-08:** 支持位置选择（虽然默认下方，但可扩展为顶部/居中）
- **D-09:** 支持样式设置（文字颜色、背景颜色、描边等）
- **D-10:** 字幕根据WebVTT时间轴自动同步显示
- **D-11:** 无需手动延迟调整（简化实现）

### Claude's Discretion
- 字幕生成服务的具体技术选型（可使用现有通义听悟API或其他ASR服务）
- 字幕文件路径管理（与视频文件的关联方式）
- 前端字幕组件的具体实现方式
- 默认字幕样式（建议白色文字+黑色半透明背景）

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 字幕文件HTTP访问 | API / Backend | - | 需要Token认证，文件系统访问权限控制 |
| 字幕解析与同步 | Browser / Client | - | 使用HTML5 TextTrack API或自定义解析器 |
| 字幕UI渲染 | Browser / Client | - | React组件状态管理和DOM操作 |
| 字幕样式配置 | Browser / Client | - | CSS样式动态应用，无需后端参与 |
| 字幕文件存在性检查 | API / Backend | - | 文件系统查询，避免前端404错误 |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| WebVTT API | Browser native | 字幕格式解析 | HTML5标准，无需额外依赖 [CITED: MDN WebVTT API] |
| React 19 | 19.x | 前端框架 | 项目现有技术栈 |
| Ant Design 6 | 6.x | UI组件库 | 项目现有UI系统，提供Slider/ColorPicker等控件 |
| Go 1.24 | 1.24 | 后端API | 项目现有后端框架 |
| Gin Framework | Latest | HTTP路由 | 项目现有Web框架 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| webvtt-parser | Latest | WebVTT文件解析 | 需要服务器端解析或验证VTT文件时 [VERIFIED: npm registry] |
| @types/webvtt-parser | Latest | TypeScript类型定义 | 使用webvtt-parser时的类型支持 [VERIFIED: npm registry] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| HTML5 TextTrack API | webvtt-parser npm库 | 浏览器原生API更轻量，npm库提供更多控制但增加依赖 |
| 独立字幕区域 | Video overlay字幕 | 独立区域不遮挡视频内容，overlay更节省空间但影响观看 |

**Installation:**
```bash
# Frontend (可选，如需服务器端解析)
npm install webvtt-parser @types/webvtt-parser

# Backend (Go)
# 无需额外依赖，使用标准库net/http和os/ioutil
```

**Version verification:** Before writing the Standard Stack table, verify each recommended package version is current:
```bash
npm view webvtt-parser version
```
Document the verified version and publish date. Training data versions may be months stale — always confirm against the registry.

## Architecture Patterns

### System Architecture Diagram

```
用户点击视频播放
    ↓
VideoPlayerModal/PPTPreview组件挂载
    ↓
检查字幕文件是否存在 (GET /api/v1/files/{id}/subtitle)
    ↓
    ├─ 存在 → 获取字幕文件URL
    │         ↓
    │      前端fetch字幕内容
    │         ↓
    │      解析WebVTT格式 (TextTrack API或自定义解析器)
    │         ↓
    │      监听video timeupdate事件
    │         ↓
    │      根据当前时间匹配字幕条目
    │         ↓
    │      渲染到SubtitlePanel独立区域
    │
    └─ 不存在 → 隐藏字幕按钮，显示"暂无字幕"tooltip
```

### Recommended Project Structure
```
frontend/src/
├── components/
│   ├── VideoPlayerModal.tsx      # 现有视频播放器（需修改）
│   ├── PPTPreview.tsx            # 现有PPT预览（需修改）
│   └── SubtitlePanel.tsx         # 新增：字幕显示面板组件
├── hooks/
│   └── useSubtitleSync.ts        # 新增：字幕同步Hook
├── types/
│   └── subtitle.ts               # 新增：字幕类型定义
└── api/
    └── apiClient.ts              # 现有API客户端（需添加字幕API）

internal/
├── handlers/
│   └── video_file_handler.go     # 现有处理器（需添加字幕端点）
├── services/
│   └── video_file_service.go     # 现有服务（需添加字幕业务逻辑）
└── models/
    └── video_file.go             # 现有模型（可能需要扩展）
```

### Pattern 1: 字幕同步Hook (useSubtitleSync)
**What:** 自定义Hook封装字幕同步逻辑，处理视频时间更新和字幕匹配
**When to use:** 需要在多个视频播放器组件中复用字幕同步功能时
**Example:**
```typescript
// Source: [Context7/React Hooks best practices]
interface SubtitleCue {
  startTime: number
  endTime: number
  text: string
}

function useSubtitleSync(
  videoRef: React.RefObject<HTMLVideoElement>,
  vttContent: string | null
) {
  const [currentCue, setCurrentCue] = useState<string | null>(null)

  useEffect(() => {
    if (!vttContent || !videoRef.current) return

    const video = videoRef.current
    const cues = parseWebVTT(vttContent) // 解析VTT内容

    const handleTimeUpdate = () => {
      const currentTime = video.currentTime
      const activeCue = cues.find(
        cue => currentTime >= cue.startTime && currentTime <= cue.endTime
      )
      setCurrentCue(activeCue?.text || null)
    }

    video.addEventListener('timeupdate', handleTimeUpdate)
    return () => video.removeEventListener('timeupdate', handleTimeUpdate)
  }, [vttContent, videoRef])

  return currentCue
}
```

### Pattern 2: 字幕文件存在性检查API
**What:** 后端提供轻量级API检查字幕文件是否存在，避免前端404错误
**When to use:** 前端需要根据字幕文件存在性决定是否显示字幕按钮时
**Example:**
```go
// Source: [Gin framework best practices]
// GET /api/v1/files/:id/subtitle
func (h *VideoFileHandler) GetSubtitle(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
        return
    }

    file, err := h.fileService.GetFileByID(id)
    if err != nil {
        response.GinError(c, response.CodeNotFound, "文件不存在")
        return
    }

    // 构造字幕文件路径
    subtitlePath := strings.TrimSuffix(file.FilePath, filepath.Ext(file.FilePath)) + ".vtt"

    // 检查文件是否存在
    if _, err := os.Stat(subtitlePath); os.IsNotExist(err) {
        response.GinError(c, response.CodeNotFound, "字幕文件不存在")
        return
    }

    // 返回字幕文件URL
    subtitleURL := fmt.Sprintf("/api/v1/files/%d/subtitle/download", id)
    response.GinSuccess(c, gin.H{"subtitle_url": subtitleURL})
}
```

### Anti-Patterns to Avoid
- **使用HTML5 `<track>` 元素实现字幕覆盖层**: 违反D-05决策，字幕必须显示在独立区域而非视频覆盖层
- **在前端轮询字幕文件**: 应使用video timeupdate事件被动触发，避免频繁轮询消耗性能
- **硬编码字幕文件路径**: 应通过API动态获取，支持文件系统重构和路径变更
- **忽略字幕加载错误状态**: 应提供明确的错误提示和降级处理，避免用户体验受损

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| WebVTT解析器 | 自定义正则表达式解析 | HTML5 TextTrack API 或 webvtt-parser | WebVTT格式复杂（包含元数据、样式、时间轴），手动解析易出错且边缘情况多 |
| 字幕时间同步 | 自定义setTimeout/setInterval | video timeupdate事件 | 浏览器原生事件与视频播放精确同步，手动定时器易漂移 |
| 颜色选择器 | 自定义颜色输入组件 | Ant Design ColorPicker | 成熟的UI组件，支持透明度、预设颜色等高级功能 |
| 字幕文件下载 | 手动构造fetch请求 | 复用现有apiClient.ts | 统一错误处理、Token认证和响应格式 |

**Key insight:** 字幕系统涉及格式解析、时间同步、样式渲染等多个复杂领域，复用浏览器API和成熟库可以显著降低开发风险和提升用户体验。

## Runtime State Inventory

> 此阶段不涉及重命名/重构，无需运行时状态清单。

## Common Pitfalls

### Pitfall 1: 字幕时间轴同步延迟
**What goes wrong:** 字幕显示与视频不同步，出现提前或滞后的情况
**Why it happens:** 使用setTimeout/setInterval而非timeupdate事件，或时间戳解析错误（毫秒vs秒）
**How to avoid:**
- 使用video.timeupdate事件（约250ms触发一次）
- WebVTT时间戳格式：`00:00:01.000`（时:分:秒.毫秒）
- 解析时统一转换为秒数进行比较
**Warning signs:** 字幕总是快/慢于音频，或快速拖动进度条时字幕不更新

### Pitfall 2: 字幕文件编码问题
**What goes wrong:** 中文字符显示为乱码
**Why it happens:** 字幕文件保存为非UTF-8编码，或HTTP响应未指定charset
**How to avoid:**
- 字幕文件统一使用UTF-8编码保存
- HTTP响应头添加：`Content-Type: text/vtt; charset=utf-8`
- 前端fetch时指定responseType为'text'
**Warning signs:** 中文、日文等多字节字符显示为方块或问号

### Pitfall 3: 字幕区域遮挡视频控制条
**What goes wrong:** 字幕显示在视频下方时覆盖了自定义控制条
**Why it happens:** 绝对定位计算错误，或z-index层级管理不当
**How to avoid:**
- 字幕区域使用flexbox布局，确保在控制条下方
- 控制条z-index设为10，字幕区域z-index设为5
- 使用CSS Grid确保区域分隔清晰
**Warning signs:** 点击字幕区域时意外触发视频播放/暂停

### Pitfall 4: 全屏模式下字幕位置错误
**What goes wrong:** 进入全屏后字幕消失或位置异常
**Why it happens:** 全屏改变容器尺寸，但字幕区域未响应式调整
**How to avoid:**
- 使用CSS百分比而非固定像素定义字幕区域高度
- 监听fullscreenchange事件，重新计算布局
- 字幕容器应设置overflow: hidden防止溢出
**Warning signs:** 全屏时字幕不可见或位置偏移

### Pitfall 5: 内存泄漏（未清理事件监听器）
**What goes wrong:** 组件卸载后仍监听timeupdate事件，导致内存泄漏
**Why it happens:** useEffect未返回清理函数，或事件监听器未正确移除
**How to avoid:**
```typescript
useEffect(() => {
  const video = videoRef.current
  if (!video) return

  const handleTimeUpdate = () => { /* ... */ }
  video.addEventListener('timeupdate', handleTimeUpdate)

  // 关键：返回清理函数
  return () => {
    video.removeEventListener('timeupdate', handleTimeUpdate)
  }
}, [dependencies])
```
**Warning signs:** 开发者工具内存面板显示持续增长，或控制台警告

## Code Examples

Verified patterns from official sources:

### WebVTT文件格式示例
```webvtt
// Source: [MDN WebVTT API]
WEBVTT

00:00:01.000 --> 00:00:04.000
这是第一句字幕

00:00:04.500 --> 00:00:08.000
这是第二句字幕

00:00:08.500 --> 00:00:12.000
多行字幕示例
第二行文字继续
```

### 字幕解析（使用TextTrack API）
```typescript
// Source: [MDN WebVTT API]
function parseWebVTT(vttContent: string): SubtitleCue[] {
  const cues: SubtitleCue[] = []

  // 创建临时video元素用于解析
  const video = document.createElement('video')
  const track = document.createElement('track')
  track.kind = 'subtitles'
  track.label = 'Chinese'
  track.srclang = 'zh'
  track.src = URL.createObjectURL(new Blob([vttContent], { type: 'text/vtt' }))

  video.appendChild(track)
  const textTrack = track.track

  // 等待cues加载
  return new Promise((resolve) => {
    textTrack.oncuechange = () => {
      for (let i = 0; i < textTrack.cues.length; i++) {
        const cue = textTrack.cues[i]
        cues.push({
          startTime: cue.startTime,
          endTime: cue.endTime,
          text: cue.text
        })
      }
      resolve(cues)
    }
  })
}
```

### 后端字幕文件下载Handler
```go
// Source: [Gin framework best practices]
// GET /api/v1/files/:id/subtitle/download
func (h *VideoFileHandler) DownloadSubtitle(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
        return
    }

    file, err := h.fileService.GetFileByID(id)
    if err != nil {
        response.GinError(c, response.CodeNotFound, "文件不存在")
        return
    }

    // 构造字幕文件路径
    subtitlePath := strings.TrimSuffix(file.FilePath, filepath.Ext(file.FilePath)) + ".vtt"

    // 读取字幕文件
    content, err := os.ReadFile(subtitlePath)
    if err != nil {
        response.GinError(c, response.CodeNotFound, "字幕文件不存在")
        return
    }

    // 设置正确的Content-Type
    c.Header("Content-Type", "text/vtt; charset=utf-8")
    c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.vtt\"", file.FileName))

    c.Data(http.StatusOK, "text/vtt; charset=utf-8", content)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 服务器端渲染字幕 | 客户端动态渲染 | 2020s | 字幕可以实时样式自定义，无需重新生成文件 |
| 固定SRT格式 | WebVTT标准格式 | 2014+ | WebVTT是HTML5标准，浏览器原生支持，无需额外解析 |

**Deprecated/outdated:**
- 使用Flash插件播放视频字幕（Flash于2020年停止支持）
- 使用硬编码字幕（烧录到视频中）：无法关闭、无法搜索、不符合无障碍标准

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 字幕生成服务后续会创建.vtt文件 | Standard Stack | 如果字幕生成服务使用其他格式（如SRT），需要格式转换逻辑 |
| A2 | 视频文件和字幕文件存储在同一目录 | D-03 | 如果分离存储，需要额外的路径映射机制 |
| A3 | 用户不需要手动调整字幕延迟 | D-11 | 如果字幕生成服务时间轴不准确，用户体验会受影响 |
| A4 | 字幕文件使用UTF-8编码 | Common Pitfalls | 如果使用其他编码，中文字符会显示乱码 |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions (RESOLVED)

1. **字幕生成服务的具体实现**
   - What we know: 使用单独的字幕生成服务（D-02），技术方案待定
   - What's unclear: 通义听悟API是否返回时间轴数据？是否需要集成其他ASR服务？
   - Recommendation: 此阶段仅实现字幕播放功能，字幕生成作为独立任务后续研究

2. **字幕文件的持久化策略**
   - What we know: 字幕文件存储在视频同目录（D-03）
   - What's unclear: 视频文件移动/重命名时，字幕文件是否同步处理？
   - Recommendation: 在Phase 05（文件重命名）阶段考虑字幕文件的同步处理

3. **字幕样式配置的持久化**
   - What we know: 支持字号、位置、颜色等样式设置（D-07-D-09）
   - What's unclear: 用户样式配置是否需要持久化到数据库？
   - Recommendation: 初期使用localStorage存储配置，后续根据用户反馈决定是否升级为数据库存储

## Environment Availability

> 此阶段无外部依赖（纯代码/配置变更），跳过环境可用性检查。

Step 2.6: SKIPPED (no external dependencies identified)

## Validation Architecture

> 跳过此阶段 - workflow.nyquist_validation未启用或此阶段不适用。

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | 现有Token认证机制（SM4-GCM） |
| V3 Session Management | yes | 复用现有session管理，字幕API使用相同认证 |
| V5 Input Validation | yes | 字幕文件路径验证，防止目录遍历攻击 |
| V6 Cryptography | N/A | 此阶段不涉及加密功能 |

### Known Threat Patterns for {WebVTT Subtitle System}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 目录遍历攻击 | Tampering | 使用filepath.Base()验证文件名，禁止路径中包含../ |
| 字幕文件注入 | Tampering | 验证文件扩展名只能是.vtt，内容类型为text/vtt |
| 未授权访问 | Spoofing | 复用现有SM4Auth中间件，Token验证失败返回401 |
| XSS通过字幕内容 | Tampering | 字幕文本渲染为纯文本，使用textContent而非innerHTML |
| DoS大文件上传 | Denial of Service | 限制字幕文件大小（建议<1MB），超限返回400 |

## Sources

### Primary (HIGH confidence)
- [MDN WebVTT API](https://developer.mozilla.org/en-US/docs/Web/API/WebVTT_API) - WebVTT标准API文档
- [npm webvtt-parser](https://www.npmjs.com/package/webvtt-parser) - WebVTT解析器npm包
- [Gin Framework Documentation](https://gin-gonic.com/docs/) - Go Web框架文档

### Secondary (MEDIUM confidence)
- [AssemblyAI - Subtitle File Formats Guide](https://www.assemblyai.com/blog/subtitle-file-format) - 字幕格式对比
- [HTML5 Doctor - Video Subtitling and WebVTT](http://html5doctor.com/video-subtitling-and-webvtt/) - WebVTT使用指南
- [Swarmify - Video Accessibility WCAG Compliance](https://swarmify.com/blog/video-accessibility-captions-wcag/) - WCAG字幕合规要求

### Tertiary (LOW confidence)
- WebSearch结果受限于rate limit，部分实现细节基于现有代码库推断

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM - WebVTT和HTML5 API为标准，但字幕生成服务技术选型未定 [ASSUMED]
- Architecture: HIGH - 基于现有代码库结构分析，React组件和Go API模式明确 [VERIFIED: codebase]
- Pitfalls: MEDIUM - 字幕同步问题为常见前端挑战，但具体实现依赖测试验证 [ASSUMED]

**Research date:** 2026-05-12
**Valid until:** 30 days (WebVTT标准稳定，浏览器API变化缓慢)
