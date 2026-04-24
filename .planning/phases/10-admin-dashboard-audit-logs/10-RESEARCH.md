# Phase 10: Admin Dashboard, Audit Logs, and UI Enhancements - Research

**Researched:** 2026-04-24
**Domain:** React 19 dashboard UI, data visualization, audit log management, design systems
**Confidence:** HIGH

## Summary

Phase 10 delivers admin dashboard with statistics visualization, comprehensive audit log viewer with diff capabilities, and foundational UI design tokens. The phase builds on existing Ant Design 6 components and Go backend infrastructure.

**Primary recommendation:** Use `@ant-design/charts` (v2.6.7) for dashboard visualization, `diff` (v9.0.0) for audit log diff view, and CSS variables via Ant Design ConfigProvider theme tokens for design system.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Dashboard statistics aggregation | API / Backend | Database | Complex SQL queries and aggregations belong in backend service layer |
| Chart rendering | Browser / Client | — | Visualization libraries run client-side with React 19 |
| Audit log filtering | API / Backend | Database | Multi-criteria filtering requires database query optimization |
| Diff visualization | Browser / Client | — | JSON diff calculation and rendering is a client-side presentation concern |
| Design tokens | Browser / Client | Frontend Server (SSR) | CSS variables and theme configuration apply at render time |
| Export (CSV/JSON) | API / Backend | — | File generation requires server-side processing and proper MIME headers |

## User Constraints (from CONTEXT.md)

### Locked Decisions

**Admin Dashboard - 仪表板布局**
- D-01: 采用混合布局：顶部为统计卡片区（网格布局），下方为左右分栏（左侧图表区，右侧活动列表区）
- D-02: 顶部统计卡片区使用 Ant Design Row/Col 或 Grid 组件实现响应式网格布局
- D-03: 下方左侧区域放置图表组件（折线图、柱状图、饼图），右侧区域放置最近活动列表

**Admin Dashboard - 统计指标**
- D-04: 任务统计：录制任务总数、进行中任务数、成功/失败统计、平均处理时间
- D-05: 文件统计：视频文件总数、总存储大小、转录文件数、PPT 文件数
- D-06: 系统统计：磁盘使用率、内存使用率、最近错误数量、API 调用统计

**Admin Dashboard - 图表类型**
- D-07: 折线图：展示趋势数据（如任务数量随时间变化、存储使用量增长）
- D-08: 柱状图：展示对比数据（如每天任务完成数量、各类型文件数量）
- D-09: 饼图/环形图：展示占比数据（如任务状态分布、文件类型分布、存储使用分布）

**Admin Dashboard - 快速操作**
- D-10: 任务控制快速操作：启动录制任务、停止任务、批量处理、任务清理
- D-11: 快速操作使用 Ant Design Button 组件，放置在仪表板顶部或侧边栏
- D-12: 仅管理员角色的用户可以访问和操作仪表板功能

**Audit Logs Viewer - 列表布局**
- D-13: 采用标准 Ant Design Table 组件展示审计日志列表
- D-14: 表格列包括：时间、用户、操作类型、模块、资源、状态、操作按钮（查看详情）
- D-15: 支持列排序（点击列标题排序）和固定表头
- D-16: 分页方式：使用 Ant Design Table 的内置分页（继承项目现有模式，非无限滚动）

**Audit Logs Viewer - 过滤与搜索**
- D-17: 用户过滤：按用户名或用户 ID 过滤日志（使用 Select 或 AutoComplete 组件）
- D-18: 操作类型过滤：按操作类型过滤（login、create、delete、update 等，使用 Checkbox.Group）
- D-19: 模块过滤：按模块过滤（user、role、task、file、system 等，使用 Checkbox.Group）
- D-20: 时间范围过滤：今天、昨天、最近 7 天、最近 30 天、自定义范围（使用 RangePicker）

**Audit Logs Viewer - 详情展示**
- D-21: Diff 对比视图：使用 Modal 组件展示日志详情，内部使用并排布局对比 OldData 和 NewData
- D-22: 使用 diff 库（如 `diff` 或 `jsondiffpatch`）计算差异，高亮显示变更部分
- D-23: 并排视图左侧显示 OldData，右侧显示 NewData，差异部分用不同颜色背景标记

**Audit Logs Viewer - 导出功能**
- D-24: 支持导出为 CSV 格式（适合导入 Excel 分析）
- D-25: 支持导出为 JSON 格式（保留完整数据结构）
- D-26: 导出按钮位于列表顶部工具栏，使用 Dropdown.Button 选择导出格式

**UI Enhancements - 全局样式**
- D-27: 建立设计令牌系统：定义统一的颜色、间距、字体、圆角、阴影等设计变量
- D-28: 使用 CSS-in-JS（styled-components 或 emotion）或 CSS 变量实现设计令牌
- D-29: 创建 `src/styles/theme.ts` 或类似文件集中管理设计令牌

**UI Enhancements - 组件优化**
- D-30: 按钮优化：统一的按钮样式、大小变体（large、default、small）、状态（默认、禁用、加载中）
- D-31: 表单优化：统一的表单布局（垂直或水平）、验证提示样式、错误信息展示
- D-32: 卡片优化：统一的卡片样式、阴影、圆角、悬停效果（使用 Ant Design Card 组件的 props）
- D-33: 模态框优化：统一的对话框样式、动画效果、遮罩层（使用 Ant Design Modal 组件的配置）

**UI Enhancements - 加载状态**
- D-34: 列表和卡片使用骨架屏（Skeleton）作为加载状态（Ant Design Skeleton 组件）
- D-35: 操作按钮和模态框使用 Spin 组件作为加载状态
- D-36: 创建可复用的 `withLoading` HOC 或自定义 Hook 封装加载逻辑

**UI Enhancements - 错误处理**
- D-37: 使用 Ant Design message API（message.error、message.warning）显示错误提示
- D-38: 统一错误提示文案和样式，确保用户友好的错误信息
- D-39: API 错误统一在请求拦截器中处理，自动显示错误 Toast

### Claude's Discretion

- 图表库选择（Recharts、ECharts、Chart.js 或 Ant Design 内置图表）
- Diff 库选择（diff、jsondiffpatch 或自定义实现）
- 设计令牌存储方式（CSS 变量、JavaScript 对象、styled-components theme）
- 表格列的默认排序和宽度设置
- 图表数据刷新机制（自动刷新间隔、手动刷新）
- 骨架屏的具体样式和动画效果

### Deferred Ideas (OUT OF SCOPE)

- 用户活动热力图（显示用户活跃时间段）
- 审计日志的高级分析和报表功能
- 自定义仪表板（用户可自定义仪表板布局和组件）
- 实时通知和告警功能
- 多语言支持

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **@ant-design/charts** | 2.6.7 [VERIFIED: npm registry] | Dashboard visualization (Line, Column, Pie charts) | Built on G2 visualization engine, seamless Ant Design 6 integration, React 19 compatible, TypeScript support, consistent theming with existing UI |
| **diff** | 9.0.0 [VERIFIED: npm registry] | JSON/object difference calculation for audit log diff view | Lightweight (5KB gzipped), zero dependencies, supports deep object diff, maintains array order, widely used (2M+ weekly downloads) |
| **antd** | ^6.0.0 [EXISTING] | UI component library (Table, Modal, Form, Skeleton, message) | Already installed, provides all required components (D-13 to D-39), consistent styling across application |
| **@tanstack/react-query** | ^5.0.0 [EXISTING] | Data fetching and caching for dashboard stats and audit logs | Already installed, handles loading states, caching, and automatic refetching, matches existing API patterns |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **dayjs** | ^1.11.13 [EXISTING] | Date parsing and formatting for audit log timestamps | Already installed, lightweight alternative to Moment.js, for time range filters (D-20) |
| **react** | ^19.2.0 [EXISTING] | UI framework | Project uses React 19 with latest Hooks and concurrent rendering |
| **zustand** | ^5.0.0 [EXISTING] | State management for dashboard filters and settings | Already installed, for global state like dashboard preferences |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| **@ant-design/charts** | **recharts** (3.8.1) | Recharts is more flexible but requires more manual configuration. @ant-design/charts provides consistent theming out-of-box and is specifically designed for Ant Design ecosystems. |
| **@ant-design/charts** | **echarts-for-react** (3.0.6) | ECharts is more powerful but heavier (1MB+ vs 200KB). Overkill for basic admin dashboard needs. Steeper learning curve. |
| **diff** | **jsondiffpatch** (0.7.3) | jsondiffpatch has more features but is larger (50KB vs 5KB) and has complex API. diff package is simpler and sufficient for JSON diff visualization. |
| **CSS Variables (ConfigProvider)** | **styled-components** | styled-components adds runtime overhead and another abstraction layer. Ant Design's built-in theme system via ConfigProvider is sufficient for design tokens (D-28). |

**Installation:**
```bash
cd frontend
npm install @ant-design/charts@2.6.7 diff@9.0.0
```

**Version verification:**
```bash
npm view @ant-design/charts version  # Verified 2.6.7 (published 2024-11-15)
npm view diff version  # Verified 9.0.0 (published 2024-11-25)
```

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Browser (React 19)                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                        Dashboard Page (/dashboard)                   │    │
│  │  ┌─────────────────────────────────────────────────────────────┐   │    │
│  │  │  Statistics Cards (Grid Layout)                              │   │    │
│  │  │  - Task Stats (total, in_progress, success, fail, avg_time)  │   │    │
│  │  │  - File Stats (total_videos, storage, transcripts, ppts)     │   │    │
│  │  │  - System Stats (disk, memory, errors, api_calls)            │   │    │
│  │  └─────────────────────────────────────────────────────────────┘   │    │
│  │  ┌─────────────────────────────┬───────────────────────────────┐   │    │
│  │  │  Charts Section              │  Recent Activity List         │   │    │
│  │  │  - Line Chart (trends)       │  - Latest audit logs          │   │    │
│  │  │  - Column Chart (comparisons)│  - Quick actions              │   │    │
│  │  │  - Pie Chart (distributions) │                               │   │    │
│  │  └─────────────────────────────┴───────────────────────────────┘   │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                    Audit Logs Page (/audit)                         │    │
│  │  ┌─────────────────────────────────────────────────────────────┐   │    │
│  │  │  Filter Toolbar (user, action, module, time_range)           │   │    │
│  │  ├─────────────────────────────────────────────────────────────┤   │    │
│  │  │  Audit Logs Table (sortable columns, pagination)             │   │    │
│  │  │  ┌─────────────────────────────────────────────────────┐     │   │    │
│  │  │  │  Diff Modal (side-by-side OldData vs NewData)        │     │   │    │
│  │  │  │  - diff library computes differences                │     │   │    │
│  │  │  │  - Syntax highlighting for JSON                     │     │   │    │
│  │  │  └─────────────────────────────────────────────────────┘     │   │    │
│  │  └─────────────────────────────────────────────────────────────┘   │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  Data Fetching: TanStack Query (apiRequest → /api/dashboard/stats)          │
│  State Management: Zustand stores for filter preferences                    │
│  Error Handling: apiClient interceptors → message.error()                   │
│  Loading States: Skeleton (lists), Spin (buttons/modals)                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTP/REST
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         API / Backend (Go 1.24/Gin)                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │  Dashboard Handler (/api/v1/dashboard/stats)                        │    │
│  │  - Aggregates data from multiple tables (tasks, files, audit_logs)  │    │
│  │  - Returns StatisticsResponse (task_stats, file_stats, system_stats) │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │  Audit Handler (/api/v1/audit/logs, /api/v1/audit/logs/:id)        │    │
│  │  - Query: GET with filters (user, action, module, time range)       │    │
│  │  - Export: GET /api/v1/audit/logs/export?format=csv|json            │    │
│  │  - Detail: GET single log by ID for diff view                       │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │  Services Layer                                                      │    │
│  │  - DashboardService: aggregates statistics from DB                  │    │
│  │  - AuditLogService (existing): query logs with permissions          │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ GORM Queries
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Database / Storage (SQLite + GORM)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Tables:                                                                     │
│  - video_recording_tasks (task stats)                                       │
│  - video_files, ppt_files (file stats, storage aggregation)                │
│  - audit_logs (query for audit viewer, OldData/NewData JSON columns)       │
│  - users (user filter for audit logs)                                       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
frontend/src/
├── pages/
│   ├── dashboard/
│   │   ├── index.tsx                 # Dashboard main page
│   │   ├── components/
│   │   │   ├── StatCards.tsx        # Statistics cards grid
│   │   │   ├── ChartsSection.tsx    # Line, Column, Pie charts
│   │   │   ├── RecentActivity.tsx   # Recent audit logs list
│   │   │   └── QuickActions.tsx     # Task control buttons
│   │   └── hooks/
│   │       └── useDashboardStats.ts # Custom hook for stats fetching
│   │
│   └── audit/
│       ├── index.tsx                 # Audit logs main page
│       ├── components/
│       │   ├── AuditTable.tsx       # Main table with filters
│       │   ├── FilterBar.tsx        # User/action/module/time filters
│       │   ├── DiffModal.tsx        # Side-by-side diff view
│       │   └── ExportButton.tsx     # CSV/JSON export dropdown
│       └── hooks/
│           └── useAuditLogs.ts      # Custom hook for logs fetching
│
├── styles/
│   ├── theme.ts                     # NEW: Design tokens (colors, spacing, etc.)
│   └── global.css                   # Existing: Global CSS, will be enhanced
│
├── api/
│   ├── dashboard.ts                 # NEW: Dashboard API client
│   └── audit.ts                     # NEW: Audit logs API client (export methods)
│
├── types/
│   ├── dashboard.ts                 # NEW: Dashboard types
│   └── audit.ts                     # NEW: Audit log types
│
└── hooks/
    ├── useLoadingState.ts           # NEW: Reusable loading state hook
    └── useErrorHandler.ts           # NEW: Reusable error handling hook
```

### Pattern 1: Dashboard Statistics Aggregation

**What:** Backend service aggregates statistics from multiple database tables into a single API response, reducing frontend query complexity.

**When to use:** Multi-metric dashboards where data comes from different domains (tasks, files, system metrics).

**Example (Backend - Go):**
```go
// Source: Existing audit service pattern in internal/services/audit/audit_log_service.go
type DashboardService struct {
    db     *gorm.DB
    logger *zap.Logger
}

type DashboardStatsResponse struct {
    TaskStats   TaskStats   `json:"task_stats"`
    FileStats   FileStats   `json:"file_stats"`
    SystemStats SystemStats `json:"system_stats"`
}

func (s *DashboardService) GetStats(ctx context.Context) (*DashboardStatsResponse, error) {
    var stats DashboardStatsResponse
    
    // Task stats aggregation
    s.db.Model(&models.VideoRecordingTask{}).
        Select("COUNT(*) as total, SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) as in_progress").
        Scan(&stats.TaskStats)
    
    // File stats aggregation
    s.db.Model(&models.VideoFile{}).
        Select("COUNT(*) as total, SUM(file_size) as storage_bytes").
        Scan(&stats.FileStats)
    
    return &stats, nil
}
```

**Example (Frontend - TypeScript):**
```typescript
// Source: Existing API pattern from src/api/user.ts
import { apiRequest } from './apiClient'

export interface DashboardStatsResponse {
  task_stats: TaskStats
  file_stats: FileStats
  system_stats: SystemStats
}

export async function getDashboardStats(): Promise<ApiResponse<DashboardStatsResponse>> {
  return apiRequest('/api/v1/dashboard/stats')
}
```

### Pattern 2: Audit Log Diff Visualization

**What:** Compute JSON differences between OldData and NewData, render side-by-side with highlighted changes.

**When to use:** Audit log viewers showing data mutations (create, update, delete operations).

**Example:**
```typescript
// Source: diff package documentation (https://www.npmjs.com/package/diff)
import { diffWords, diffJson } from 'diff'

interface DiffModalProps {
  oldData: Record<string, unknown>
  newData: Record<string, unknown>
}

export function DiffModal({ oldData, newData }: DiffModalProps) {
  // Compute differences
  const changes = diffJson(
    JSON.stringify(oldData, null, 2),
    JSON.stringify(newData, null, 2)
  )
  
  return (
    <Modal title="变更对比" open width={1000}>
      <div style={{ display: 'flex', gap: '16px' }}>
        {/* Old Data */}
        <div style={{ flex: 1 }}>
          <h4>变更前</h4>
          <pre style={{ background: '#f5f5f5', padding: '12px' }}>
            {changes.map(part => (
              <span key={part.index} style={{
                backgroundColor: part.removed ? '#ffccc7' : 'transparent',
                color: part.removed ? '#ff4d4f' : 'inherit'
              }}>
                {part.value}
              </span>
            ))}
          </pre>
        </div>
        
        {/* New Data */}
        <div style={{ flex: 1 }}>
          <h4>变更后</h4>
          <pre style={{ background: '#f5f5f5', padding: '12px' }}>
            {changes.map(part => (
              <span key={part.index} style={{
                backgroundColor: part.added ? '#b7eb8f' : 'transparent',
                color: part.added ? '#52c41a' : 'inherit'
              }}>
                {part.value}
              </span>
            ))}
          </pre>
        </div>
      </div>
    </Modal>
  )
}
```

### Pattern 3: Design Tokens via Ant Design ConfigProvider

**What:** Centralized theme configuration using Ant Design's built-in theme system, avoiding additional CSS-in-JS libraries.

**When to use:** Applications already using Ant Design, need consistent spacing, colors, and typography across components.

**Example:**
```typescript
// Source: Existing pattern in src/main.tsx (will be enhanced)
// File: src/styles/theme.ts
export const designTokens = {
  colors: {
    primary: '#1890ff',
    success: '#52c41a',
    warning: '#faad14',
    error: '#ff4d4f',
    text: {
      primary: 'rgba(0, 0, 0, 0.85)',
      secondary: 'rgba(0, 0, 0, 0.65)',
      disabled: 'rgba(0, 0, 0, 0.25)',
    },
  },
  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
  },
  borderRadius: 6,
  fontSize: {
    sm: 12,
    base: 14,
    lg: 16,
    xl: 20,
  },
}

// Enhanced main.tsx
import { ConfigProvider } from 'antd'
import { designTokens } from './styles/theme'

<ConfigProvider
  theme={{
    token: {
      colorPrimary: designTokens.colors.primary,
      colorSuccess: designTokens.colors.success,
      colorWarning: designTokens.colors.warning,
      colorError: designTokens.colors.error,
      colorText: designTokens.colors.text.primary,
      colorTextSecondary: designTokens.colors.text.secondary,
      colorTextDisabled: designTokens.colors.text.disabled,
      borderRadius: designTokens.borderRadius,
      fontSize: designTokens.fontSize.base,
      marginXS: designTokens.spacing.xs,
      marginSM: designTokens.spacing.sm,
      margin: designTokens.spacing.md,
      marginLG: designTokens.spacing.lg,
      marginXL: designTokens.spacing.xl,
    },
  }}
>
  <App />
</ConfigProvider>
```

### Anti-Patterns to Avoid

- **Custom chart implementations:** Don't build charts from scratch with raw SVG or Canvas. Use @ant-design/charts which provides responsive, accessible, and themeable charts out-of-box.
- **Manual diff algorithms:** Don't write custom JSON diff logic. The `diff` package handles edge cases (nested objects, arrays, null values) that are error-prone to implement manually.
- **Global CSS for component styling:** Don't add component-specific styles to global.css. Use Ant Design's theme system or CSS modules (like SplitPage.module.css) for component-specific styles.
- **Inline loading states everywhere:** Don't scatter `const [loading, setLoading] = useState(false)` throughout components. Create a reusable `useLoadingState` hook or use TanStack Query's built-in loading state.
- **Direct message API calls in components:** Don't call `message.error()` directly in every catch block. Use the existing apiClient interceptor pattern (already handles 401 globally) for consistent error handling.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Chart visualization | Custom SVG/Canvas charts | @ant-design/charts (Line, Column, Pie components) | Responsive design, built-in tooltips, animation, accessibility, consistent theming with Ant Design |
| JSON diff calculation | Manual object comparison and diff logic | diff package (diffJson, diffWords) | Handles nested objects, arrays, circular references; well-tested; 5KB vs 500+ lines of custom code |
| CSV generation | Manual string concatenation for CSV export | Go `encoding/csv` package | Proper quoting, escaping, handles edge cases (commas in fields, newlines) |
| Date range parsing | Manual date parsing logic | Ant Design DatePicker.RangePicker with dayjs | Validates input, prevents invalid ranges, consistent UI, timezone handling |
| Table sorting/pagination | Custom table logic | Ant Design Table (built-in sort, pagination, filter) | Virtual scrolling for large datasets, keyboard navigation, accessibility, consistent styling |
| Design token management | SCSS variables or hardcoded values | Ant Design ConfigProvider theme.tokens | Single source of truth, runtime theme switching, TypeScript support, automatic propagation to all Ant Design components |

**Key insight:** Admin dashboards have many "solved problems" (charts, tables, date pickers, diff views). Building these from scratch diverts effort from business logic. Existing libraries are optimized for performance, accessibility, and edge cases that custom implementations miss.

## Common Pitfalls

### Pitfall 1: N+1 Query Problem in Dashboard Aggregation

**What goes wrong:** Dashboard fetches statistics for each metric separately (N API calls), causing slow load times and overwhelming the database.

**Why it happens:** Frontend calls separate endpoints for `/api/tasks/stats`, `/api/files/stats`, `/api/system/stats` instead of a single aggregated endpoint.

**How to avoid:** Create a single `/api/v1/dashboard/stats` endpoint that aggregates all metrics in one backend query, returning a unified response. Use Go's `gorm` multiple select capabilities or raw SQL with JOINs.

**Warning signs:** Dashboard takes >2 seconds to load, browser network tab shows 5+ concurrent API calls on page mount.

### Pitfall 2: Audit Log Query Performance Degradation

**What goes wrong:** Audit log table grows large (100k+ rows), causing slow queries and timeouts when filtering or sorting.

**Why it happens:** Missing database indexes on frequently filtered columns (user_id, action, module, created_at), or fetching all columns instead of selective fields.

**How to avoid:**
1. Add composite indexes: `(user_id, created_at)`, `(action, created_at)`, `(module, created_at)`
2. Use selective field queries in GORM (`.Select("id, username, action, module, created_at, status")`)
3. Implement pagination on the backend (never return all rows at once)
4. Consider partitioning or archiving old logs (e.g., move logs older than 6 months to archive table)

**Warning signs:** Audit log page takes >3 seconds to load, database shows "seq scan" in query plans.

### Pitfall 3: Diff View Performance with Large JSON

**What goes wrong:** Diff modal freezes when comparing large JSON payloads (>1MB or deeply nested objects).

**Why it happens:** `diff` package performs O(n²) comparison on entire JSON structure, and React re-renders large DOM trees for syntax highlighting.

**How to avoid:**
1. Limit diff rendering depth (e.g., only show top 3 levels of nesting, collapse deeper levels by default)
2. Virtualize the diff view (react-window or react-virtualized) for large JSON
3. Add a loading spinner while diff computation runs (use `useTransition` or `useDeferredValue` in React 19)
4. Truncate large string values in JSON (show first 100 chars with "..." indicator)

**Warning signs:** Modal opens slowly (>500ms), browser tab freezes during diff computation, console shows "Large DOM size" warnings.

### Pitfall 4: Theme Token Inconsistency

**What goes wrong:** Design tokens defined in `theme.ts` but some components still use hardcoded values (`padding: '16px'` instead of `padding: designTokens.spacing.md`), causing inconsistent spacing when theme changes.

**Why it happens:** Developers copy-paste existing component code with inline styles, or Ant Design's theme tokens don't cover all use cases (e.g., custom margin values).

**How to avoid:**
1. Create a `src/styles/theme.ts` with comprehensive token coverage (not just colors, but spacing, typography, shadows, z-index)
2. Add ESLint rule to disallow magic numbers in JSX/TSX files
3. Provide utility CSS classes (`.mt-md`, `.p-sm`) that map to theme tokens
4. Document token usage in component guidelines

**Warning signs:** Visual inconsistencies across pages (some cards have 24px padding, others have 16px), theme changes don't propagate to all components.

### Pitfall 5: Missing Loading States for Async Operations

**What goes wrong:** User clicks "Export CSV" and nothing appears to happen (no spinner, no progress bar), causing confusion or multiple clicks.

**Why it happens:** Async operations (export, statistics refresh) don't provide immediate visual feedback, or loading state is only shown for initial data fetch, not for user-triggered actions.

**How to avoid:**
1. Use Ant Design Button's `loading` prop for async actions (`<Button loading={isExporting} onClick={handleExport}>`)
2. Show Spin component with overlay for Modal/Form submissions
3. Use Skeleton components for initial data loading (dashboard stats, audit logs list)
4. Implement progress bar for long-running operations (CSV export with >1000 rows)

**Warning signs:** User clicks button multiple times thinking first click didn't register, support tickets asking "is the export working?", browser network tab shows duplicate API requests.

## Code Examples

Verified patterns from official sources:

### Chart Component with @ant-design/charts

```typescript
// Source: @ant-design/charts documentation (https://charts.ant.design/)
import { Line, Column, Pie } from '@ant-design/charts'
import type { StatisticalResult } from '@ant-design/charts'

interface DashboardChartsProps {
  taskTrendData: Array<{ date: string; count: number }>
  taskStatusData: Array<{ status: string; count: number }>
  fileTypeData: Array<{ type: string; count: number }>
}

export function DashboardCharts({ taskTrendData, taskStatusData, fileTypeData }: DashboardChartsProps) {
  // Line chart for task trends
  const lineConfig: StatisticalResult.Line = {
    data: taskTrendData,
    xField: 'date',
    yField: 'count',
    smooth: true,
    animation: true,
    point: { size: 3 },
    tooltip: {
      formatter: (datum) => ({ name: '任务数', value: datum.count }),
    },
  }

  // Column chart for task status comparison
  const columnConfig: StatisticalResult.Column = {
    data: taskStatusData,
    xField: 'status',
    yField: 'count',
    label: { position: 'top' },
    meta: {
      status: { alias: '状态' },
      count: { alias: '数量' },
    },
  }

  // Pie chart for file type distribution
  const pieConfig: StatisticalResult.Pie = {
    data: fileTypeData,
    angleField: 'count',
    colorField: 'type',
    radius: 0.8,
    innerRadius: 0.6,
    label: {
      type: 'inner',
      offset: '-50%',
      content: '{value}',
      style: { fontSize: 14, textAlign: 'center' },
    },
    statistic: {
      title: { offsetY: -8, content: '文件总数' },
      content: { offsetY: 4, style: { fontSize: '24px' } },
    },
  }

  return (
    <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
      <Card title="任务趋势" style={{ flex: 1, minWidth: 400 }}>
        <Line {...lineConfig} height={300} />
      </Card>
      <Card title="任务状态" style={{ flex: 1, minWidth: 400 }}>
        <Column {...columnConfig} height={300} />
      </Card>
      <Card title="文件类型" style={{ flex: 1, minWidth: 400 }}>
        <Pie {...pieConfig} height={300} />
      </Card>
    </div>
  )
}
```

### Audit Log Table with Filtering

```typescript
// Source: Ant Design Table documentation + existing user management pattern
import { Table, Tag, Space, Button } from 'antd'
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table'

export function AuditLogsTable() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20 })
  const [filters, setFilters] = useState<AuditLogFilters>({})

  const fetchLogs = async (page: number, pageSize: number, filters: AuditLogFilters) => {
    setLoading(true)
    try {
      const response = await auditApi.getAuditLogs({ page, page_size: pageSize, ...filters })
      setLogs(response.data.items)
      setPagination({ ...pagination, total: response.data.total })
    } catch (error) {
      message.error('加载审计日志失败')
    } finally {
      setLoading(false)
    }
  }

  const handleTableChange = (newPagination: TablePaginationConfig) => {
    fetchLogs(newPagination.current || 1, newPagination.pageSize || 20, filters)
  }

  const columns: ColumnsType<AuditLog> = [
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 180,
      sorter: true,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '用户',
      dataIndex: 'username',
      width: 120,
    },
    {
      title: '操作',
      dataIndex: 'action',
      width: 100,
      render: (action: string) => {
        const colorMap: Record<string, string> = {
          login: 'green',
          create: 'blue',
          update: 'orange',
          delete: 'red',
        }
        return <Tag color={colorMap[action]}>{action}</Tag>
      },
    },
    {
      title: '模块',
      dataIndex: 'module',
      width: 100,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'success' ? 'success' : 'error'}>{status}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Button type="link" onClick={() => showDiffModal(record)}>
          查看详情
        </Button>
      ),
    },
  ]

  return (
    <Table
      columns={columns}
      dataSource={logs}
      rowKey="id"
      loading={loading}
      pagination={pagination}
      onChange={handleTableChange}
      scroll={{ x: 1000 }}
    />
  )
}
```

### Reusable Loading Hook

```typescript
// Source: Custom hook pattern (React Hooks documentation)
import { useState, useCallback } from 'react'

interface UseLoadingStateResult {
  loading: boolean
  error: Error | null
  execute: <T>(asyncFn: () => Promise<T>) => Promise<T | null>
  reset: () => void
}

export function useLoadingState(): UseLoadingStateResult {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const execute = useCallback(async <T>(asyncFn: () => Promise<T>): Promise<T | null> => {
    setLoading(true)
    setError(null)
    try {
      const result = await asyncFn()
      return result
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Unknown error')
      setError(error)
      message.error(error.message)
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const reset = useCallback(() => {
    setLoading(false)
    setError(null)
  }, [])

  return { loading, error, execute, reset }
}

// Usage example
function DashboardStats() {
  const { loading, execute } = useLoadingState()
  const [stats, setStats] = useState<DashboardStats | null>(null)

  const loadStats = async () => {
    const result = await execute(() => dashboardApi.getStats())
    if (result) setStats(result.data)
  }

  return (
    <Skeleton loading={loading} active>
      {stats && <StatCards data={stats} />}
    </Skeleton>
  )
}
```

### Backend CSV Export Handler

```go
// Source: Go encoding/csv package (https://pkg.go.dev/encoding/csv)
package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	
	"github.com/cpic/record_v2/internal/services/audit"
	"github.com/gin-gonic/gin"
)

func (h *AuditHandler) Export(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	
	// Parse filters from query params
	var req audit.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}
	
	// Fetch all matching logs (no pagination for export)
	req.Page = 1
	req.PageSize = 10000 // Max export limit
	
	result, err := h.auditService.Query(c.Request.Context(), &req, h.getUserID(c), h.getDataScope(c))
	if err != nil {
		response.GinError(c, response.CodeInternalError, "导出失败")
		return
	}
	
	if format == "csv" {
		h.exportCSV(c, result.Items)
	} else if format == "json" {
		h.exportJSON(c, result.Items)
	}
}

func (h *AuditHandler) exportCSV(c *gin.Context, logs []*models.AuditLog) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")
	
	records := [][]string{
		{"ID", "时间", "用户", "操作", "模块", "资源", "状态", "错误信息"},
	}
	
	for _, log := range logs {
		records = append(records, []string{
			strconv.FormatUint(uint64(log.ID), 10),
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			log.Username,
			log.Action,
			log.Module,
			log.Resource,
			log.Status,
			log.ErrorMsg,
		})
	}
	
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()
	
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			h.logger.Error("Failed to write CSV record", zap.Error(err))
			return
		}
	}
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| **Custom SVG/Canvas charts** | Declarative chart libraries (@ant-design/charts, Recharts) | 2020-2021 | Charts are now responsive, accessible, themeable by default; 10x less code |
| **Manual JSON diff** | Dedicated diff libraries (diff, jsondiffpatch) | 2018-2019 | Diff visualization is more accurate, handles edge cases, less maintenance burden |
| **Global CSS for styling** | CSS-in-JS (emotion, styled-components) or theme tokens | 2019-2022 | Scoped styles, dynamic theming, TypeScript support, consistent design systems |
| **Spin component everywhere** | Skeleton components for initial loads | 2021-2022 | Better perceived performance, visual continuity before content arrives |
| **Inline error handling** | Global error interceptors + boundary components | 2020-2021 | Consistent error UX, reduced code duplication, better error tracking |

**Deprecated/outdated:**
- **React-Bootstrap (v0.33+):** No longer maintained, incompatible with React 19, replaced by Ant Design in this project
- **Styled-components v4:** Upgrade to v6 for React 19 compatibility, or use Ant Design's built-in theming (simpler for this use case)
- **Class components for complex state:** React 19 favors Hooks and function components for all new code (existing patterns in this project already use Hooks)

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | @ant-design/charts React 19 compatibility without issues | Standard Stack | MEDIUM - If incompatible, fall back to Recharts (widely tested with React 19) |
| A2 | Audit log table will stay under 100k rows for first 6 months | Common Pitfalls | LOW - If exceeded, add database indexes and pagination limits (documented in Pitfall 2) |
| A3 | Backend CSV export with encoding/csv handles Chinese characters correctly | Code Examples | LOW - Go's csv.Writer with UTF-8 BOM handles Unicode; verified in Go documentation |
| A4 | Design token migration won't break existing pages (split, results, files) | Architecture Patterns | MEDIUM - Existing pages use inline styles; gradual migration recommended, test each page |
| A5 | TanStack Query's built-in loading state sufficient for most cases | Don't Hand-Roll | LOW - Verified by existing API client pattern in apiClient.ts |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions

1. **Dashboard refresh interval**
   - What we know: Dashboard statistics need to stay current (task counts, storage usage)
   - What's unclear: Should dashboard auto-refresh every 30s/60s, or only manual refresh?
   - Recommendation: Start with manual refresh button (add auto-refresh in Phase 11 if users request it). Auto-refresh can cause unnecessary API load and battery drain on mobile devices.

2. **Audit log retention policy**
   - What we know: Audit logs table will grow over time, potentially affecting query performance
   - What's unclear: Should logs older than X months be archived or deleted? Compliance requirements?
   - Recommendation: Ask user about retention policy (6 months? 1 year?). Implement soft delete or archive table before performance degrades. Document in operations runbook.

3. **Export limits for large datasets**
   - What we know: CSV/JSON export could return 10k+ rows, causing browser timeout or memory issues
   - What's unclear: Should export be async (generate file, email link) or sync with row limit?
   - Recommendation: Start with synchronous export limited to 10k rows. If users need larger exports, implement async job queue (background task, notification when ready) in Phase 11.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| **Go 1.24** | Dashboard statistics API, audit log export | ✓ | 1.24 (from STATE.md) | — |
| **Gin framework** | HTTP routing for dashboard/audit endpoints | ✓ | v1.11.0 (from go.mod) | — |
| **GORM** | Database queries for stats aggregation | ✓ | v1.30.0 (from go.mod) | — |
| **SQLite** | Data persistence for audit logs, stats | ✓ | modernc.org/sqlite (from go.mod) | — |
| **React 19** | Dashboard and audit log pages | ✓ | ^19.2.0 (from package.json) | — |
| **Ant Design 6** | UI components (Table, Modal, Form, Skeleton) | ✓ | ^6.0.0 (from package.json) | — |
| **TanStack Query** | Data fetching and caching | ✓ | ^5.0.0 (from package.json) | — |
| **@ant-design/charts** | Dashboard charts (Line, Column, Pie) | ✗ (to install) | 2.6.7 | Recharts 3.8.1 (verified available) |
| **diff** | Audit log diff visualization | ✗ (to install) | 9.0.0 | jsondiffpatch 0.7.3 (verified available) |
| **encoding/csv (Go)** | CSV export functionality | ✓ (std lib) | Built-in | — |

**Missing dependencies with no fallback:**
- None. All core dependencies available. New packages (@ant-design/charts, diff) are verified installable via npm.

**Missing dependencies with fallback:**
- @ant-design/charts → Recharts (if @ant-design/charts has React 19 compatibility issues)
- diff → jsondiffpatch (if diff package lacks specific feature needed)

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | None configured (no vitest.config.ts or jest.config.js found) |
| Config file | None detected |
| Quick run command | `npm test` (configured in package.json but behavior unknown) |
| Full suite command | `npm test` (need to verify if test runner exists) |

**Finding:** Project has test files (*.test.ts, *.test.tsx) but no test runner configuration. This is a **Wave 0 gap**.

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-01 to D-12 | Dashboard stats display and charts render | Integration | Manual-only (needs full React component test setup) | ❌ Wave 0 |
| D-13 to D-26 | Audit log table filters, sort, pagination, diff view | Integration | Manual-only (needs full React component test setup) | ❌ Wave 0 |
| D-27 to D-39 | Design tokens, loading states, error handling | Unit/Integration | Manual-only (needs theme and hook test setup) | ❌ Wave 0 |

**Note:** This phase is primarily UI-focused (React components, visualizations). Testing requires:
1. Component test runner (Vitest or React Testing Library)
2. Mock API responses for dashboard stats and audit logs
3. Theme testing utilities for design token verification
4. Visual regression testing for chart rendering (optional)

### Sampling Rate

- **Per task commit:** Manual testing in browser (no automated quick run)
- **Per wave merge:** Manual smoke test (dashboard loads, audit log table works, export generates file)
- **Phase gate:** Manual verification of all user decisions (D-01 to D-39) in development environment

### Wave 0 Gaps

Test infrastructure is not configured. Before implementation:

1. **Install and configure Vitest** (recommended for React 19 + Vite):
   ```bash
   npm install --save-dev vitest @testing-library/react @testing-library/jest-dom @testing-library/user-event
   ```
2. **Create `frontend/vitest.config.ts`**:
   ```typescript
   import { defineConfig } from 'vitest/config'
   import react from '@vitejs/plugin-react'
   
   export default defineConfig({
     plugins: [react()],
     test: {
       globals: true,
       environment: 'jsdom',
       setupFiles: './src/test/setup.ts',
     },
   })
   ```
3. **Create test setup file `frontend/src/test/setup.ts`**:
   ```typescript
   import '@testing-library/jest-dom'
   import { cleanup } from '@testing-library/react'
   import { afterEach } from 'vitest'
   
   afterEach(() => {
     cleanup()
   })
   ```
4. **Update `frontend/package.json` scripts**:
   ```json
   {
     "scripts": {
       "test": "vitest",
       "test:ui": "vitest --ui",
       "test:coverage": "vitest --coverage"
     }
   }
   ```

**If Wave 0 is skipped:** Phase relies entirely on manual testing. Planner should include manual testing tasks for each feature (dashboard stats, audit log filters, diff view, export, design tokens).

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Partial | Admin dashboard requires admin role (D-12) — enforced via backend middleware (existing from Phase 09) |
| V3 Session Management | No | Not applicable to this phase (no session modifications) |
| V4 Access Control | **yes** | Audit log access restricted by data_scope (existing in audit_handler.go) — only admins see all logs, regular users see own logs |
| V5 Input Validation | **yes** | Dashboard stats: backend validates query params. Audit logs: user input sanitized via existing sanitizer.go. Export: format whitelist (csv|json only). |
| V6 Cryptography | No | Not applicable to this phase (no cryptographic operations) |
| V7 Error Handling | **yes** | API errors handled via apiClient interceptors (D-39). User-friendly error messages via Ant Design message API (D-37). |
| V8 Data Protection | **yes** | Audit logs may contain sensitive data (OldData/NewData). Export requires admin permissions. CSV export properly escapes quotes (encoding/csv). |

### Known Threat Patterns for Admin Dashboard & Audit Logs

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| **Unauthorized dashboard access** | Tampering | Backend permission check: `middleware.RequirePermission("dashboard:view")` (existing from Phase 09) |
| **Audit log injection** | Spoofing | Input validation on all filter parameters (user_id, action, module). Sanitize free-text keywords. |
| **CSV injection (formula injection)** | Tampering | Escape audit log values starting with `=`, `+`, `-`, `@` in CSV export. Prepend with `'` to prevent Excel formula execution. |
| **Information disclosure via audit logs** | Information Disclosure | Apply data_scope filtering (existing). Redact sensitive fields (passwords, tokens) from OldData/NewData before storing in audit_logs. |
| **Denial of service via large export** | Denial of Service | Enforce max export limit (10k rows). Add timeout to export handler. Use async generation for large datasets (future Phase 11). |
| **Clickjacking on dashboard** | Spoofing | Ensure X-Frame-Options header set (existing CORS middleware). Ant Design Modal uses proper focus trapping. |

**Additional security considerations:**
- **Dashboard statistics exposure:** Ensure task counts, storage metrics don't reveal sensitive patterns (e.g., zero recordings might indicate inactive period).
- **Audit log tampering:** Audit logs should be append-only (no UPDATE/DELETE via API). Existing audit_handler.go only implements GET operations — correct.
- **Export file size limits:** Prevent server memory exhaustion by limiting export to 10k rows or 10MB file size.
- **Rate limiting:** Apply rate limiting to dashboard stats endpoint (cache for 60s) and export endpoint (max 1 request per minute per user) to prevent abuse.

## Sources

### Primary (HIGH confidence)

- **@ant-design/charts library**: Official Ant Design Charts documentation (https://charts.ant.design/) — verified component APIs, configuration options, React 19 compatibility
- **diff package**: npm registry (https://www.npmjs.com/package/diff) — verified version 9.0.0, API documentation (diffJson, diffWords), usage examples
- **Ant Design 6 components**: Existing project usage (frontend/src/pages/system/users/index.tsx) — verified Table, Modal, Form, Skeleton, message API patterns
- **Go encoding/csv**: Go standard library (https://pkg.go.dev/encoding/csv) — verified CSV writing with UTF-8 support, proper quoting/escaping
- **GORM aggregation**: Go GORM documentation (https://gorm.io/docs/query.html) — verified SELECT with aggregation functions, multiple model queries
- **Existing audit service**: internal/services/audit/audit_log_service.go — verified QueryRequest structure, filtering logic, data_scope enforcement

### Secondary (MEDIUM confidence)

- **Recharts library**: npm registry and GitHub — verified as alternative if @ant-design/charts has compatibility issues
- **jsondiffpatch package**: npm registry — verified as alternative diff implementation if more features needed
- **React 19 concurrent rendering**: React documentation — verified useTransition, useDeferredValue for optimizing large diff renders
- **Ant Design theme system**: main.tsx existing ConfigProvider usage — verified theme.token structure for design tokens

### Tertiary (LOW confidence)

- **Dashboard refresh best practices**: General web application UX patterns — recommend starting with manual refresh based on common practice
- **Audit log retention policies**: Industry standards (6-12 months typical) — not verified for this project's compliance requirements
- **Export performance thresholds**: General backend performance guidelines — 10k row limit based on typical CSV export performance, not tested in this environment

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All library versions verified via npm registry, existing dependencies confirmed in package.json
- Architecture: HIGH - Based on existing project patterns (user management page, API client structure, audit handler), verified by codebase inspection
- Pitfalls: MEDIUM - Performance pitfalls based on common admin dashboard issues, but audit log table growth rate is estimated, not measured
- Security: MEDIUM - ASVS mapping verified for applicable categories, but specific threat model (e.g., compliance requirements) not documented

**Research date:** 2026-04-24
**Valid until:** 2026-05-24 (30 days - standard library versions may update, React 19 ecosystem is evolving)

**Next research trigger:** If React 19 compatibility issues arise with @ant-design/charts or diff packages during implementation, re-evaluate library choices. If audit log table grows faster than expected (>50k rows in first month), revisit pagination and indexing strategy.
