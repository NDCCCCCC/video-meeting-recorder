# Phase 3: PPT Management - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-17
**Phase:** 03-ppt-management
**Areas discussed:** PPT Preview, Multi-result display, Slide merge UX, Result page layout

---

## PPT Preview

| Option | Description | Selected |
|--------|-------------|----------|
| 服务端提取图片（推荐） | Backend extracts each slide as JPEG via Python-pptx, frontend displays in carousel/lightbox | ✓ |
| PDF转换+查看器 | Convert PPTX to PDF via LibreOffice headless, render with react-pdf | |
| 客户端渲染 PPTX | Upload PPTX to frontend and parse with JS library | |

**User's choice:** 服务端提取图片
**Notes:** Python-pptx already integrated in PPTXGenerator, natural extension.

| Option | Description | Selected |
|--------|-------------|----------|
| 左右翻页+缩略图导航 | Arrow buttons + bottom thumbnail strip | |
| 上下滚动列表 | Vertical scroll through all pages | |
| 主视图+侧边缩略图 | Main view + sidebar thumbnail panel | ✓ |

**User's choice:** 主视图+侧边缩略图
**Notes:** Similar to PowerPoint's thumbnail sidebar pattern.

| Option | Description | Selected |
|--------|-------------|----------|
| 按需生成+缓存（推荐） | Extract on first preview, cache results | ✓ |
| 转录时立即生成 | Extract immediately after transcription completes | |
| Claude决定 | | |

**User's choice:** 按需生成+缓存

| Option | Description | Selected |
|--------|-------------|----------|
| 双分辨率（推荐） | Thumbnails 200x112px + main view 1920x1080px | ✓ |
| 单一分辨率 | One size 1280x720px, CSS-scaled thumbnails | |
| Claude决定 | | |

**User's choice:** 双分辨率：缩略图+高清图

| Option | Description | Selected |
|--------|-------------|----------|
| 支持全屏模式（推荐） | Fullscreen button hides sidebar, slides fill screen | ✓ |
| 不需要全屏 | Preview only within result page | |

**User's choice:** 支持全屏模式

| Option | Description | Selected |
|--------|-------------|----------|
| 页码指示+跳转（推荐） | Show "第 3/25 页" with click-to-jump input | ✓ |
| 仅缩略图导航 | No page numbers, navigate only via thumbnails | |

**User's choice:** 页码指示+跳转

| Option | Description | Selected |
|--------|-------------|----------|
| 单页下载 | Download individual slide as image | ✓ |
| 复制图片到剪贴板 | Copy slide image to clipboard for pasting | ✓ |

**User's choice:** Both selected (multiSelect)

| Option | Description | Selected |
|--------|-------------|----------|
| JPEG 质量90%（推荐） | JPEG quality 90%, good balance of size and clarity | ✓ |
| PNG 无损 | Lossless PNG, larger files | |

**User's choice:** JPEG 质量90%

---

## Multi-result Display

| Option | Description | Selected |
|--------|-------------|----------|
| 卡片列表（推荐） | Vertical list of transcription result cards | |
| Tab标签切换 | Tab-based switching between results | |
| 画廊式切换 | Horizontal thumbnail strip, prominent current result | ✓ |

**User's choice:** 画廊式切换
**Notes:** User preferred visual switching over text-heavy lists.

| Option | Description | Selected |
|--------|-------------|----------|
| 结果页内按钮触发（推荐） | "重新转录" button in result page action panel | ✓ |
| 返回文件列表触发 | Navigate back to file list to trigger | |

**User's choice:** 结果页内按钮触发

| Option | Description | Selected |
|--------|-------------|----------|
| 最新结果优先（推荐） | Default to newest transcription result | ✓ |
| 页数最多优先 | Default to result with most pages | |

**User's choice:** 最新结果优先

---

## Slide Merge UX

| Option | Description | Selected |
|--------|-------------|----------|
| 结果页内进入合并模式（推荐） | "合并幻灯片" button enters inline merge mode | ✓ |
| 独立合并页面 | Navigate to dedicated merge page | |

**User's choice:** 结果页内进入合并模式

| Option | Description | Selected |
|--------|-------------|----------|
| 勾选+拖拽排序 | Checkbox selection with drag reorder | |
| 点击选中+底部排列（推荐） | Click thumbnails to select, bottom bar with drag reorder | ✓ |
| 逐个PPT添加模式 | Add slides one PPT at a time | |

**User's choice:** 点击选中+底部排列

| Option | Description | Selected |
|--------|-------------|----------|
| 生成新PPTX文件（推荐） | Save merged result as new PPTX on server | ✓ |
| 仅下载不保存 | Download only, no server-side persistence | |

**User's choice:** 生成新PPTX文件

| Option | Description | Selected |
|--------|-------------|----------|
| 关联原始视频（推荐） | Associate merged PPT with original video | ✓ |
| 独立文件不关联 | Standalone file, no video association | |

**User's choice:** 关联原始视频

| Option | Description | Selected |
|--------|-------------|----------|
| 不限制 | No upper limit on merged slides | |
| 上限200页（推荐） | Max 200 slides per merge (aligned with PPTXGenerator limits) | ✓ |
| Claude决定 | | |

**User's choice:** 上限200页

| Option | Description | Selected |
|--------|-------------|----------|
| 简单Loading（推荐） | Loading spinner + completion toast | ✓ |
| 详细进度跟踪 | Detailed progress "已合并 5/20 页" | |

**User's choice:** 简单Loading

---

## Result Page Layout

| Option | Description | Selected |
|--------|-------------|----------|
| 左右分栏：预览+信息面板（推荐） | Left = PPT preview, Right = info/action panel | ✓ |
| 上下布局：预览+信息 | Top = preview, Bottom = info | |
| Claude决定 | | |

**User's choice:** 左右分栏：预览+信息面板

| Option | Description | Selected |
|--------|-------------|----------|
| 文件列表按钮跳转（推荐） | "预览PPT" button in file list jumps to result page | ✓ |
| 独立菜单入口 | Separate "转录结果" nav menu item | |

**User's choice:** 文件列表按钮跳转

| Option | Description | Selected |
|--------|-------------|----------|
| 基本信息区 | Video name, transcription time, sampling rate, page count, file size | ✓ |
| 操作按钮区 | Download, re-transcribe, merge, delete buttons | ✓ |
| 多结果切换区 | Gallery strip showing all results with time + page count | ✓ |

**User's choice:** All three sections selected (multiSelect)

---

## Claude's Discretion

- Exact Python-pptx slide extraction implementation
- Slide image caching strategy
- Merge PPTX generation approach
- PPTFile model extensions
- API endpoint paths and structures
- Thumbnail strip component details
- Drag-to-reorder library choice
- Merge mode UI state management
- Error handling for slide extraction failures
- Handling deleted source PPTs in merge results

## Deferred Ideas

None — discussion stayed within phase scope.
