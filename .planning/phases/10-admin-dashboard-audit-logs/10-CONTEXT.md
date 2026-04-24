# Phase 10: Admin Dashboard, Audit Logs, and UI Enhancements - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 10 交付三个核心功能：管理员仪表板（Admin Dashboard）、审计日志查看器（Audit Logs Viewer）和用户界面增强（UI Enhancements）。

**管理员仪表板**提供系统概览和统计数据的可视化展示，包括任务统计、文件统计和系统统计，支持多种图表类型和任务控制快速操作。

**审计日志查看器**提供完整的审计日志查询和展示功能，包括表格布局、多维度过滤、Diff 对比视图和数据导出（CSV、JSON）。

**UI 增强**建立统一的设计令牌系统，优化核心组件（按钮、表单、卡片、模态框），改进加载状态（骨架屏）和错误处理（Toast 通知）。

</domain>

<decisions>
## Implementation Decisions

### Admin Dashboard - 仪表板布局
- **D-01:** 采用混合布局：顶部为统计卡片区（网格布局），下方为左右分栏（左侧图表区，右侧活动列表区）
- **D-02:** 顶部统计卡片区使用 Ant Design Row/Col 或 Grid 组件实现响应式网格布局
- **D-03:** 下方左侧区域放置图表组件（折线图、柱状图、饼图），右侧区域放置最近活动列表

### Admin Dashboard - 统计指标
- **D-04:** 任务统计：录制任务总数、进行中任务数、成功/失败统计、平均处理时间
- **D-05:** 文件统计：视频文件总数、总存储大小、转录文件数、PPT 文件数
- **D-06:** 系统统计：磁盘使用率、内存使用率、最近错误数量、API 调用统计

### Admin Dashboard - 图表类型
- **D-07:** 折线图：展示趋势数据（如任务数量随时间变化、存储使用量增长）
- **D-08:** 柱状图：展示对比数据（如每天任务完成数量、各类型文件数量）
- **D-09:** 饼图/环形图：展示占比数据（如任务状态分布、文件类型分布、存储使用分布）

### Admin Dashboard - 快速操作
- **D-10:** 任务控制快速操作：启动录制任务、停止任务、批量处理、任务清理
- **D-11:** 快速操作使用 Ant Design Button 组件，放置在仪表板顶部或侧边栏
- **D-12:** 仅管理员角色的用户可以访问和操作仪表板功能

### Audit Logs Viewer - 列表布局
- **D-13:** 采用标准 Ant Design Table 组件展示审计日志列表
- **D-14:** 表格列包括：时间、用户、操作类型、模块、资源、状态、操作按钮（查看详情）
- **D-15:** 支持列排序（点击列标题排序）和固定表头
- **D-16:** 分页方式：使用 Ant Design Table 的内置分页（继承项目现有模式，非无限滚动）

### Audit Logs Viewer - 过滤与搜索
- **D-17:** 用户过滤：按用户名或用户 ID 过滤日志（使用 Select 或 AutoComplete 组件）
- **D-18:** 操作类型过滤：按操作类型过滤（login、create、delete、update 等，使用 Checkbox.Group）
- **D-19:** 模块过滤：按模块过滤（user、role、task、file、system 等，使用 Checkbox.Group）
- **D-20:** 时间范围过滤：今天、昨天、最近 7 天、最近 30 天、自定义范围（使用 RangePicker）

### Audit Logs Viewer - 详情展示
- **D-21:** Diff 对比视图：使用 Modal 组件展示日志详情，内部使用并排布局对比 OldData 和 NewData
- **D-22:** 使用 diff 库（如 `diff` 或 `jsondiffpatch`）计算差异，高亮显示变更部分
- **D-23:** 并排视图左侧显示 OldData，右侧显示 NewData，差异部分用不同颜色背景标记

### Audit Logs Viewer - 导出功能
- **D-24:** 支持导出为 CSV 格式（适合导入 Excel 分析）
- **D-25:** 支持导出为 JSON 格式（保留完整数据结构）
- **D-26:** 导出按钮位于列表顶部工具栏，使用 Dropdown.Button 选择导出格式

### UI Enhancements - 全局样式
- **D-27:** 建立设计令牌系统：定义统一的颜色、间距、字体、圆角、阴影等设计变量
- **D-28:** 使用 CSS-in-JS（styled-components 或 emotion）或 CSS 变量实现设计令牌
- **D-29:** 创建 `src/styles/theme.ts` 或类似文件集中管理设计令牌

### UI Enhancements - 组件优化
- **D-30:** 按钮优化：统一的按钮样式、大小变体（large、default、small）、状态（默认、禁用、加载中）
- **D-31:** 表单优化：统一的表单布局（垂直或水平）、验证提示样式、错误信息展示
- **D-32:** 卡片优化：统一的卡片样式、阴影、圆角、悬停效果（使用 Ant Design Card 组件的 props）
- **D-33:** 模态框优化：统一的对话框样式、动画效果、遮罩层（使用 Ant Design Modal 组件的配置）

### UI Enhancements - 加载状态
- **D-34:** 列表和卡片使用骨架屏（Skeleton）作为加载状态（Ant Design Skeleton 组件）
- **D-35:** 操作按钮和模态框使用 Spin 组件作为加载状态
- **D-36:** 创建可复用的 `withLoading` HOC 或自定义 Hook 封装加载逻辑

### UI Enhancements - 错误处理
- **D-37:** 使用 Ant Design message API（message.error、message.warning）显示错误提示
- **D-38:** 统一错误提示文案和样式，确保用户友好的错误信息
- **D-39:** API 错误统一在请求拦截器中处理，自动显示错误 Toast

### Claude's Discretion
- 图表库选择（Recharts、ECharts、Chart.js 或 Ant Design 内置图表）
- Diff 库选择（diff、jsondiffpatch 或自定义实现）
- 设计令牌存储方式（CSS 变量、JavaScript 对象、styled-components theme）
- 表格列的默认排序和宽度设置
- 图表数据刷新机制（自动刷新间隔、手动刷新）
- 骨架屏的具体样式和动画效果

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project requirements
- `.planning/PROJECT.md` — 技术栈（Go 1.24/Gin, React 19/Ant Design 6, SQLite/GORM）
- `.planning/STATE.md` — 项目进度和决策历史

### Phase 4 context (audit log backend model)
- `.planning/phases/04-cloud-services/04-CONTEXT.md` — 审计日志后端模型已实现

### Phase 9 context (multi-role permissions)
- `.planning/phases/09-multi-role-permissions/09-CONTEXT.md` — 多角色权限系统，共享查看者角色

### Existing audit log model
- `internal/models/audit_log.go` — AuditLog 模型（包含所有字段和常量定义）

### Existing frontend pages
- `frontend/src/pages/dashboard/index.tsx` — 仪表板占位页面（待实现）
- `frontend/src/pages/audit/index.tsx` — 审计日志占位页面（待实现）

### Project frontend patterns
- Phase 1-9 的各 CONTEXT.md 文件 — 前端模式（Ant Design 组件使用、状态管理、API 调用）

### Ant Design documentation
- Ant Design 6 Table 组件文档 — 表格功能、分页、排序、过滤
- Ant Design 6 Modal 组件文档 — 对话框使用方式
- Ant Design 6 Skeleton 组件文档 — 骨架屏使用方式
- Ant Design 6 Form 组件文档 — 表单验证和布局

No external specs — requirements are fully captured in decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **AuditLog Model** (`internal/models/audit_log.go`): 完整的审计日志模型，包含用户、操作、模块、请求上下文、变更内容等字段
- **Ant Design 6**: 项目已使用 Ant Design 6，可直接使用 Table、Modal、Form、Button、Card、Skeleton、message 等组件
- **React 19**: 项目使用 React 19，可使用最新特性（如 Hooks、Suspense）
- **TanStack Query**: 项目已集成，可用于数据获取和缓存

### Established Patterns
- **API 调用**: 使用 `apiRequest<T>()` 函数调用后端 API
- **状态管理**: 使用 useState 和 useEffect 管理组件状态
- **路由**: 使用 React Router（从 `src/pages/` 结构推断）
- **权限检查**: 通过后端中间件验证权限，前端根据用户角色显示/隐藏功能

### Integration Points
- **Backend**: 创建统计 API（GET /api/dashboard/stats）、审计日志查询 API（GET /api/audit/logs）、导出 API（GET /api/audit/logs/export）
- **Frontend**: 实现仪表板页面、审计日志页面，更新全局样式和组件
- **设计令牌**: 创建 `src/styles/theme.ts` 或类似文件
- **权限**: 仪表板功能仅对管理员角色可见

</code_context>

<specifics>
## Specific Ideas

- 仪表板布局应响应式设计，在移动设备上自动调整为单列布局
- 图表应支持交互（如点击柱状图查看详细数据、鼠标悬停显示 tooltip）
- 审计日志的 Diff 视图应支持折叠/展开嵌套对象，以便查看复杂变更
- 设计令牌应包括暗色主题支持，为未来扩展做准备
- 组件优化应保持 Ant Design 的默认风格，仅在必要时进行定制
- 骨架屏应模拟实际内容结构，提供更好的用户体验

</specifics>

<deferred>
## Deferred Ideas

- 用户活动热力图（显示用户活跃时间段）
- 审计日志的高级分析和报表功能
- 自定义仪表板（用户可自定义仪表板布局和组件）
- 实时通知和告警功能
- 多语言支持

---

*Phase: 10-admin-dashboard-audit-logs*
*Context gathered: 2026-04-24*
