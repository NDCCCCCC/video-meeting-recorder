---
phase: 06-06
plan: 02
title: 并排16:9布局
one-liner: CSS Grid 并排布局，PPT 与视频同步 16:9 纵横比，1200px 断点垂直堆叠
subsystem: ppt-editor
tags: [ui, layout, responsive, frontend]
tech-stack:
  added: []
  patterns: [CSS Grid, aspect-ratio, responsive breakpoints]
key-files:
  created: []
  modified:
    - frontend/src/pages/results/index.tsx
    - frontend/src/components/PPTPreview.tsx
    - frontend/src/styles/global.css
decisions: []
metrics:
  duration_seconds: 202
  completed_date: "2026-04-20T09:05:10Z"
  tasks_completed: 2
  files_changed: 3
  deviations: 2
---

# Phase 06-06 Plan 02: 并排16:9布局 Summary

## Objective
实现 PPT 和视频预览的并排 16:9 布局，支持响应式堆叠，提供同步查看体验。

## Outcomes Delivered

### 核心功能
- **CSS Grid 并排布局**: 160px 缩略图 | 1fr PPT | 1fr 视频的三列布局
- **16:9 纵横比强制**: 两个预览框使用 `aspectRatio: '16 / 9'` CSS 属性保持宽高比
- **响应式断点**: 1200px 宽度以下自动垂直堆叠，无横向滚动
- **信息/操作栏重新定位**: 移至预览区域下方，无需滚动即可访问
- **缩略图侧边栏优化**: 宽度从 200px 减少至 160px，节省空间

### 技术实现
1. **PPTPreview 组件增强**
   - 添加 `containerStyle` prop 支持灵活布局组合
   - 添加 `hideThumbnailSidebar` prop 避免重复显示缩略图
   - 缩略图宽度优化为 160px（符合 UI-SPEC）

2. **Results 页面重构**
   - 移除 Ant Design Row/Col 布局系统
   - 实现 CSS Grid 三列布局
   - 外置缩略图侧边栏（160px 固定宽度）
   - PPT 和视频预览框共享 16:9 纵横比约束

3. **响应式布局**
   - 使用全局 CSS 类 `ppt-preview-grid` 处理媒体查询
   - @media (max-width: 1200px) 断点切换为单列堆叠
   - 避免了 CSS-in-JS 中不支持的 @media 对象语法

## Deviations from Plan

### 自动修复的问题

**1. [Rule 1 - Bug] 修复重复缩略图侧边栏显示**
- **发现时机**: Task 2 提交后的代码审查
- **问题**: PPTPreview 组件内置的缩略图侧边栏与 results 页面外置的缩略图侧边栏重复显示
- **修复**: 添加 `hideThumbnailSidebar` prop 到 PPTPreview 组件，results 页面传递 `true` 隐藏内置侧边栏
- **文件修改**: `frontend/src/components/PPTPreview.tsx`, `frontend/src/pages/results/index.tsx`
- **Commit**: 9821970

**2. [Rule 1 - Bug] 修复 CSS-in-JS @media 语法不支持**
- **发现时机**: Task 2 实现过程中的语法检查
- **问题**: CSS-in-JS 对象语法不支持 @media 媒体查询作为直接属性（例如 `{ '@media (max-width: 1200px)': {...} }`）
- **修复**: 将响应式布局逻辑移至全局 CSS 文件 `global.css`，使用 `.ppt-preview-grid` 类处理 @media 查询
- **文件修改**: `frontend/src/styles/global.css`, `frontend/src/pages/results/index.tsx`
- **Commit**: da82fa2

## Commits

| Hash | Type | Message |
|------|------|---------|
| 387dd17 | feat | optimize PPTPreview for side-by-side layout |
| 187875c | feat | implement side-by-side 16:9 preview layout |
| 9821970 | fix | hide PPTPreview built-in thumbnail sidebar in side-by-side layout |
| da82fa2 | fix | implement responsive breakpoint via CSS class |

## Verification Results

### 自动化验证
- ✅ 缩略图侧边栏宽度从 200px 更改为 160px
- ✅ PPTPreview 组件接受 `containerStyle` prop
- ✅ Results 页面包含 CSS Grid 布局 (`gridTemplateColumns: '160px 1fr 1fr'`)
- ✅ 两个预览框使用 `aspectRatio: '16 / 9'` CSS 属性
- ✅ 全局 CSS 包含 @media (max-width: 1200px) 断点

### 手动验证清单
- [ ] 在桌面浏览器（≥1200px）打开 PPT 预览页面，验证三列并排布局可见
- [ ] 检查 PPT 预览保持 16:9 形状（未被压缩或拉伸）
- [ ] 检查视频预览保持 16:9 形状
- [ ] 调整浏览器窗口大小，验证纵横比保持不变
- [ ] 调整浏览器窗口至 <1200px，验证布局垂直堆叠（缩略图 → PPT → 视频）
- [ ] 验证没有横向滚动条出现
- [ ] 滚动页面查看预览区域下方的操作栏
- [ ] 验证所有操作按钮可见且无需横向滚动
- [ ] 验证标签导航正常工作（基本信息 | 文字内容 | 操作）
- [ ] 验证缩略图侧边栏宽度为 160px
- [ ] 验证缩略图侧边栏支持垂直滚动（50+ 张缩略图）
- [ ] 点击缩略图验证导航到对应幻灯片

## Known Stubs
无。

## Threat Flags
无（本计划仅涉及客户端 CSS 布局，无安全相关威胁）。

## Performance Metrics

| Metric | Value |
|--------|-------|
| 执行时长 | 202 秒（3 分钟） |
| 任务完成数 | 2 |
| 文件修改数 | 3 |
| 自动修复数 | 2 |
| 提交数 | 4 |

## Next Steps
- 执行 Phase 06-06-03: 缩略图优化（图片懒加载、虚拟滚动）
- 执行 Phase 06-06-04: 键盘快捷键增强
- 执行 Phase 06-06-05: PPT 导出选项配置

## Notes
- CSS Grid 布局提供了比 Flexbox 更精确的三列控制
- `aspectRatio` CSS 属性在现代浏览器中支持良好，无需 JavaScript 计算
- 响应式断点使用全局 CSS 类，避免了 CSS-in-JS 的限制
- PPTPreview 组件的 `hideThumbnailSidebar` prop 提高了组件的可重用性
