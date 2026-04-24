# Phase 10: Admin Dashboard, Audit Logs, and UI Enhancements - Pattern Map

**Mapped:** 2026-04-24
**Files analyzed:** 12
**Analogs found:** 10 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/pages/dashboard/index.tsx` | component | request-response | `frontend/src/pages/system/users/index.tsx` | exact |
| `frontend/src/pages/audit/index.tsx` | component | request-response | `frontend/src/pages/system/users/index.tsx` | exact |
| `frontend/src/pages/dashboard/components/StatCards.tsx` | component | request-response | `frontend/src/pages/tasks/index.tsx` (status display) | role-match |
| `frontend/src/pages/dashboard/components/ChartsSection.tsx` | component | transform | `frontend/src/pages/tasks/index.tsx` (data display) | role-match |
| `frontend/src/pages/dashboard/components/RecentActivity.tsx` | component | request-response | `frontend/src/pages/system/users/index.tsx` (table) | exact |
| `frontend/src/pages/dashboard/components/QuickActions.tsx` | component | event-driven | `frontend/src/pages/tasks/index.tsx` (action buttons) | exact |
| `frontend/src/pages/audit/components/AuditTable.tsx` | component | request-response | `frontend/src/pages/system/users/index.tsx` (table) | exact |
| `frontend/src/pages/audit/components/FilterBar.tsx` | component | event-driven | `frontend/src/pages/system/users/index.tsx` (filter bar) | exact |
| `frontend/src/pages/audit/components/DiffModal.tsx` | component | transform | `frontend/src/pages/system/users/index.tsx` (modal pattern) | role-match |
| `frontend/src/pages/audit/components/ExportButton.tsx` | component | file-I/O | `frontend/src/pages/tasks/index.tsx` (action buttons) | role-match |
| `frontend/src/styles/theme.ts` | config | request-response | `frontend/src/main.tsx` (ConfigProvider) | role-match |
| `internal/handlers/dashboard_handler.go` | handler | request-response | `internal/handlers/user_handler.go` | exact |
| `internal/services/dashboard_service.go` | service | CRUD | `internal/services/user_service.go` | exact |
| `internal/handlers/audit_handler.go` | handler | request-response | `internal/handlers/audit_handler.go` (existing) | exact |

## Pattern Assignments

### `frontend/src/pages/dashboard/index.tsx` (component, request-response)

**Analog:** `frontend/src/pages/system/users/index.tsx`

**Imports pattern** (lines 1-28):
```typescript
import { useState, useEffect } from 'react'
import {
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  Form,
  message,
  Popconfirm,
  Switch,
  Tag
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import * as userApi from '../../../api/user'
import type { UserInfo, UserListParams } from '../../../types/user'
```

**State management pattern** (lines 30-44):
```typescript
export default function UserManagement() {
  const [users, setUsers] = useState<UserInfo[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingUser, setEditingUser] = useState<UserInfo | null>(null)
  const [form] = Form.useForm()

  // 查询参数
  const [params, setParams] = useState<UserListParams>({
    page: 1,
    page_size: 20,
  })
```

**Data fetching pattern** (lines 47-60):
```typescript
// 加载用户列表
const loadUsers = async () => {
  setLoading(true)
  try {
    const response = await userApi.getUserList(params)
    if (response.data) {
      setUsers(response.data.items)
      setTotal(response.data.total)
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载用户列表失败')
  } finally {
    setLoading(false)
  }
}

useEffect(() => {
  loadUsers()
}, [params])
```

**Error handling pattern** (lines 55-57):
```typescript
} catch (error) {
  message.error(error instanceof Error ? error.message : '加载用户列表失败')
}
```

---

### `frontend/src/pages/audit/index.tsx` (component, request-response)

**Analog:** `frontend/src/pages/system/users/index.tsx`

**Imports pattern** (lines 1-28):
```typescript
import { useState, useEffect } from 'react'
import {
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  Form,
  message,
  Popconfirm,
  Tag
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import * as userApi from '../../../api/user'
import type { UserInfo, UserListParams } from '../../../types/user'
```

**Table columns pattern** (lines 204-304):
```typescript
const columns: ColumnsType<UserInfo> = [
  {
    title: 'ID',
    dataIndex: 'id',
    width: 80,
  },
  {
    title: '用户名',
    dataIndex: 'username',
    width: 150,
  },
  {
    title: '角色',
    dataIndex: 'roles',
    width: 200,
    render: (roles: Array<{ id: number; name: string; description: string }>) => (
      <>
        {roles?.map((role: { id: number; name: string; description: string }) => (
          <Tag
            key={role.id}
            color={role.name === 'shared_viewer' ? 'purple' : 'blue'}
            style={{ marginBottom: 4 }}
          >
            {role.description || role.name}
          </Tag>
        ))}
      </>
    ),
  },
  {
    title: '操作',
    key: 'action',
    width: 200,
    fixed: 'right' as const,
    render: (_, record) => (
      <Space size="small">
        <Button
          type="link"
          size="small"
          onClick={() => openModal(record)}
        >
          编辑
        </Button>
      </Space>
    ),
  },
]
```

**Pagination pattern** (lines 353-367):
```typescript
<Table
  columns={columns}
  dataSource={users}
  rowKey="id"
  loading={loading}
  scroll={{ x: 1200 }}
  pagination={{
    current: params.page,
    pageSize: params.page_size,
    total,
    showSizeChanger: true,
    showTotal: (t) => `共 ${t} 条`,
  }}
  onChange={handleTableChange}
/>
```

---

### `frontend/src/pages/audit/components/DiffModal.tsx` (component, transform)

**Analog:** `frontend/src/pages/system/users/index.tsx` (Modal pattern)

**Modal pattern** (lines 370-454):
```typescript
<Modal
  title={editingUser ? '编辑用户' : '新建用户'}
  open={modalVisible}
  onOk={handleSubmit}
  onCancel={closeModal}
  width={600}
  destroyOnClose
>
  <Form form={form} layout="vertical">
    <Form.Item
      name="username"
      label="用户名"
      rules={[
        { required: true, message: '请输入用户名' },
        { min: 3, max: 50, message: '用户名长度为3-50个字符' },
      ]}
    >
      <Input placeholder="请输入用户名" disabled={!!editingUser} />
    </Form.Item>
  </Form>
</Modal>
```

---

### `frontend/src/api/dashboard.ts` (api client, request-response)

**Analog:** `frontend/src/api/user.ts`

**API client pattern** (lines 1-26):
```typescript
import type {
  UserListParams,
  UserListApiResponse,
} from '../types/user'
import type { ApiResponse } from '../types/auth'
import { apiRequest } from './apiClient'

// 获取用户列表
export async function getUserList(params: UserListParams): Promise<UserListApiResponse> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  if (params.role_id) queryParams.append('role_id', params.role_id.toString())
  if (params.is_active !== undefined) queryParams.append('is_active', params.is_active.toString())

  const query = queryParams.toString()
  return apiRequest(`/api/v1/users${query ? `?${query}` : ''}`)
}
```

---

### `internal/handlers/dashboard_handler.go` (handler, request-response)

**Analog:** `internal/handlers/user_handler.go`

**Handler structure pattern** (lines 11-23):
```go
// UserHandler 用户处理器
type UserHandler struct {
	userService *services.UserService
	logger      *zap.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *services.UserService, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}
```

**List handler pattern** (lines 26-60):
```go
// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 分页获取用户列表，支持关键词搜索和筛选
// @Tags 用户管理
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键词"
// @Param role_id query int false "角色ID"
// @Param is_active query bool false "是否激活"
// @Success 200 {object} response.Response{data=services.ListUsersResponse}
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req services.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	result, err := h.userService.ListUsers(&req)
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取用户列表失败")
		return
	}

	response.GinSuccess(c, result)
}
```

---

### `internal/services/dashboard_service.go` (service, CRUD)

**Analog:** `internal/services/user_service.go`

**Service structure pattern** (lines 13-27):
```go
// UserService 用户服务
type UserService struct {
	db           *gorm.DB
	logger       *zap.Logger
	auditService *audit.AuditLogService
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB, logger *zap.Logger, auditService *audit.AuditLogService) *UserService {
	return &UserService{
		db:           db,
		logger:       logger,
		auditService: auditService,
	}
}
```

**Request/Response pattern** (lines 29-41):
```go
// ListRequest 用户列表请求
type ListUsersRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size" binding:"max=100"`
	Keyword  string `form:"keyword"`
	IsActive *bool  `form:"is_active"`
}

// ListResponse 用户列表响应
type ListUsersResponse struct {
	Total int64         `json:"total"`
	Items []models.User `json:"items"`
}
```

**List service method pattern** (lines 68-104):
```go
// ListUsers 获取用户列表
func (s *UserService) ListUsers(req *ListUsersRequest) (*ListUsersResponse, error) {
	var users []models.User
	var total int64

	query := s.db.Model(&models.User{})

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR full_name LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 状态筛选
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("Roles").
		Offset(offset).
		Limit(req.PageSize).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return &ListUsersResponse{
		Total: total,
		Items: users,
	}, nil
}
```

---

### `internal/handlers/audit_handler.go` (handler, request-response) - ENHANCEMENT

**Analog:** `internal/handlers/audit_handler.go` (existing)

**Existing Query pattern** (lines 58-98):
```go
// Query 查询审计日志
// @Summary 查询审计日志
// @Tags 审计日志
// @Produce json
// @Param module query string false "模块"
// @Param action query string false "操作"
// @Param status query string false "状态"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "页大小" default(20)
// @Success 200 {object} response.Response
// @Router /api/v1/audit/logs [get]
func (h *AuditHandler) Query(c *gin.Context) {
	var req audit.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 解析时间
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			req.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			req.EndTime = t
		}
	}

	result, err := h.auditService.Query(c.Request.Context(), &req, h.getUserID(c), h.getDataScope(c))
	if err != nil {
		h.logger.Warn("查询审计日志失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "查询失败")
		return
	}

	response.GinSuccess(c, result)
}
```

**Add Export handler pattern** (NEW - based on user_handler.go):
```go
// Export 导出审计日志
// @Summary 导出审计日志
// @Tags 审计日志
// @Produce json
// @Param format query string false "导出格式" Enums(csv, json)
// @Param module query string false "模块"
// @Param action query string false "操作"
// @Success 200 {file} file
// @Router /api/v1/audit/logs/export [get]
func (h *AuditHandler) Export(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")

	var req audit.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 导出限制：最多10000条
	req.Page = 1
	req.PageSize = 10000

	result, err := h.auditService.Query(c.Request.Context(), &req, h.getUserID(c), h.getDataScope(c))
	if err != nil {
		h.logger.Warn("导出审计日志失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "导出失败")
		return
	}

	if format == "csv" {
		h.exportCSV(c, result.Items)
	} else if format == "json" {
		h.exportJSON(c, result.Items)
	} else {
		response.GinError(c, response.CodeInvalidRequest, "不支持的导出格式")
	}
}
```

---

### `frontend/src/styles/theme.ts` (config, request-response)

**Analog:** `frontend/src/main.tsx` (ConfigProvider)

**Existing ConfigProvider pattern** (lines 12-20):
```typescript
<ConfigProvider
  locale={zhCN}
  theme={{
    token: {
      colorPrimary: '#1890ff',
      borderRadius: 6,
    },
  }}
>
```

**Theme extension pattern** (NEW):
```typescript
// 设计令牌系统
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

export type ThemeTokens = typeof designTokens
```

---

### `frontend/src/pages/dashboard/components/ChartsSection.tsx` (component, transform)

**Analog:** `frontend/src/pages/tasks/index.tsx` (data display with memo)

**Memo pattern** (lines 98-183):
```typescript
interface TaskActionsProps {
  record: VideoRecordingTask
  onStart: (id: number) => void
  onStop: (id: number) => void
  // ...
}

const TaskActions = memo(function TaskActions({
  record,
  onStart,
  onStop,
  // ...
}: TaskActionsProps) {
  return (
    <Space size="small">
      {/* action buttons */}
    </Space>
  )
})
```

---

## Shared Patterns

### Authentication & Authorization

**Source:** `frontend/src/pages/system/users/index.tsx` (permission checks via backend)
**Apply to:** All dashboard and audit log components
```typescript
// Backend enforces permissions via middleware
// Frontend shows/hides based on user role from context
// Example from tasks page:
import { PermissionGuard } from '../../components/PermissionGuard'
import { PERMISSIONS } from '../../utils/permissions'

<PermissionGuard permission={PERMISSIONS.TASK_VIEW}>
  {/* protected content */}
</PermissionGuard>
```

### API Client Pattern

**Source:** `frontend/src/api/apiClient.ts`
**Apply to:** All API client files (`dashboard.ts`, `audit.ts`)
```typescript
import { apiRequest } from './apiClient'

export async function getDashboardStats(): Promise<ApiResponse<DashboardStatsResponse>> {
  return apiRequest('/api/v1/dashboard/stats')
}

// Query params pattern
export async function getAuditLogs(params: AuditLogListParams): Promise<AuditLogListApiResponse> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.module) queryParams.append('module', params.module)

  const query = queryParams.toString()
  return apiRequest(`/api/v1/audit/logs${query ? `?${query}` : ''}`)
}
```

### Error Handling

**Source:** `frontend/src/api/apiClient.ts` (lines 230-240)
**Apply to:** All service and controller files
```typescript
// API errors handled centrally in apiRequest interceptor
// Components use try-catch with message.error
try {
  const response = await userApi.getUserList(params)
  if (response.data) {
    setUsers(response.data.items)
    setTotal(response.data.total)
  }
} catch (error) {
  message.error(error instanceof Error ? error.message : '加载用户列表失败')
}
```

### Backend Response Pattern

**Source:** `pkg/response/response.go` (lines 108-140)
**Apply to:** All backend handlers
```go
// Success response
response.GinSuccess(c, result)

// Error response
response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
response.GinError(c, response.CodeInternalError, "获取统计失败")
response.GinError(c, response.CodeNotFound, "日志不存在")
```

### Service Layer Pattern

**Source:** `internal/services/user_service.go` (lines 68-104)
**Apply to:** Dashboard service
```go
// Service structure with DB, logger, auditService
type DashboardService struct {
	db           *gorm.DB
	logger       *zap.Logger
	auditService *audit.AuditLogService
}

// List method with query building, pagination
func (s *DashboardService) GetStats(req *StatsRequest) (*StatsResponse, error) {
	// Build query
	query := s.db.Model(&models.Task{})

	// Apply filters
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Execute query with pagination
	// ...
}
```

### Table Component Pattern

**Source:** `frontend/src/pages/system/users/index.tsx` (lines 353-367)
**Apply to:** Audit log table, recent activity table
```typescript
<Table
  columns={columns}
  dataSource={items}
  rowKey="id"
  loading={loading}
  scroll={{ x: 1200 }}
  pagination={{
    current: params.page,
    pageSize: params.page_size,
    total,
    showSizeChanger: true,
    showTotal: (t) => `共 ${t} 条`,
  }}
  onChange={handleTableChange}
/>
```

### Filter Bar Pattern

**Source:** `frontend/src/pages/system/users/index.tsx` (lines 315-351)
**Apply to:** Audit log filter bar
```typescript
<div style={{ marginBottom: '16px' }}>
  <Space size="middle">
    <Input.Search
      placeholder="搜索用户名、邮箱或姓名"
      allowClear
      style={{ width: 300 }}
      onSearch={handleSearch}
      enterButton={<SearchOutlined />}
    />
    <Select
      placeholder="选择角色"
      allowClear
      style={{ width: 150 }}
      onChange={handleRoleFilter}
      options={[...]}
    />
    <Button icon={<ReloadOutlined />} onClick={loadUsers}>
      刷新
    </Button>
  </Space>
</div>
```

### Modal Form Pattern

**Source:** `frontend/src/pages/system/users/index.tsx` (lines 370-454)
**Apply to:** Diff modal, export modal
```typescript
<Modal
  title={editingUser ? '编辑用户' : '新建用户'}
  open={modalVisible}
  onOk={handleSubmit}
  onCancel={closeModal}
  width={600}
  destroyOnClose
>
  <Form form={form} layout="vertical">
    <Form.Item
      name="username"
      label="用户名"
      rules={[
        { required: true, message: '请输入用户名' },
        { min: 3, max: 50, message: '用户名长度为3-50个字符' },
      ]}
    >
      <Input placeholder="请输入用户名" />
    </Form.Item>
  </Form>
</Modal>
```

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `frontend/src/pages/dashboard/components/ChartsSection.tsx` | component | transform | No chart components exist yet; use @ant-design/charts from RESEARCH.md |
| `frontend/src/pages/audit/components/DiffModal.tsx` | component | transform | No diff visualization exists; use `diff` package from RESEARCH.md |
| `frontend/src/pages/audit/components/ExportButton.tsx` | component | file-I/O | No export button pattern exists; follow Dropdown.Button pattern from Ant Design docs |
| `frontend/src/hooks/useLoadingState.ts` | hook | request-response | No custom hooks exist for loading state; use TanStack Query's built-in loading state |

---

## Metadata

**Analog search scope:**
- `frontend/src/pages/` (React components)
- `frontend/src/api/` (API clients)
- `frontend/src/styles/` (styling)
- `internal/handlers/` (Go handlers)
- `internal/services/` (Go services)
- `pkg/response/` (response utilities)

**Files scanned:** 18
**Pattern extraction date:** 2026-04-24

**Key findings:**
1. **Strong table pattern**: User management page provides excellent analog for audit log table with filters, pagination, sorting
2. **API client consistency**: All API clients follow same pattern with `apiRequest` wrapper and query params building
3. **Backend service pattern**: Dashboard service can follow user service structure with DB, logger, auditService
4. **Modal pattern**: User edit modal provides template for diff modal with Form layout
5. **ConfigProvider exists**: Main.tsx already has ConfigProvider for theme tokens
6. **No chart precedent**: ChartsSection will be first chart component, follow @ant-design/charts from RESEARCH.md
7. **No diff precedent**: DiffModal will be first diff visualization, use `diff` package from RESEARCH.md
