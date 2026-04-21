# Phase 9: Multi-Role Permissions & Shared Viewer - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning

<domain>
## Phase Boundary

创建"共享查看者"（shared_viewer）角色，允许用户查看所有用户的数据（无视数据所有权），同时支持用户拥有多个角色。数据所有权已通过 CreatedBy 字段存在，此阶段添加多对多用户-角色关系和共享查看者可见性控制。

</domain>

<decisions>
## Implementation Decisions

### 共享查看者角色的核心功能
- **D-01:** 共享查看者角色（shared_viewer）仅控制数据可见性范围，不影响操作权限判断
- **D-02:** 拥有共享查看者角色的用户可以查询所有用户创建的数据（VideoFile、RecordingTask 等），CreatedBy 过滤器被跳过
- **D-03:** 操作权限（files:delete、tasks:edit 等）仍由用户的其他角色决定，共享查看者角色本身不授予任何操作权限
- **D-04:** 角色存储名称：shared_viewer；显示名称：共享查看者

### 多角色系统架构
- **D-05:** 从单角色（User.RoleID）迁移到多对多关系：创建 users_roles 关联表
- **D-06:** User 模型变更：删除 RoleID 字段和 Role 外键，添加 Roles []Role 关联
- **D-07:** 权限检查逻辑：用户任一角色有该权限即通过（OR 逻辑）—— User.HasPermission() 遍历所有角色
- **D-08:** 数据库迁移：创建 users_roles 表，将现有单角色关系迁移到多对多表，确保数据不丢失

### 数据所有权与可见性控制
- **D-09:** 数据所有权已存在：VideoFile.CreatedBy 和 VideoRecordingTask.CreatedBy 字段
- **D-10:** 当前行为：用户只能看到自己创建的数据（查询时 WHERE created_by = ?）
- **D-11:** 共享查看者行为：查询时跳过 created_by 过滤，返回所有数据
- **D-12:** 可见性检查在权限中间件之前进行，决定查询范围；权限检查决定操作是否允许

### 共享查看者角色分配
- **D-13:** 仅系统管理员（admin 角色）可以将 shared_viewer 角色分配给用户
- **D-14:** 用户角色管理页面需要更新以支持多角色选择（从单选改为多选）
- **D-15:** 审计日志应记录共享查看者角色的分配和撤销操作（敏感操作）

### 数据库迁移策略
- **D-16:** 迁移步骤：1）创建 users_roles 表；2）将现有 users.role_id 数据复制到 users_roles；3）添加外键约束；4）删除 users.role_id 列
- **D-17:** 回滚计划：保留 role_id 列的备份或创建迁移回滚脚本
- **D-18:** 迁移验证：确认每个用户在迁移后至少保留原有角色

### Claude's Discretion
- 数据库迁移脚本的具体实现（Go 还是 SQL 脚本）
- users_roles 关联表是否需要 CreatedAt/UpdatedAt 软删除字段
- 权限中间件是否需要缓存用户的角色列表以优化性能
- 前端用户角色选择组件的具体实现（Ant Design Transfer 或 Select multiple）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project requirements
- `.planning/REQUIREMENTS.md` — RBAC 权限系统已验证，审计日志已实现
- `.planning/PROJECT.md` — 技术栈（Go 1.24/Gin, React 19/Ant Design 6, SQLite/GORM）

### Existing permission system
- `internal/models/user.go` — User 模型，HasPermission() 方法，单角色 RoleID 字段
- `internal/models/role.go` — Role 模型，预定义角色常量
- `internal/models/permission.go` — Permission 模型，权限资源常量，AllPermissions 列表
- `internal/models/permission_constants.go` — 权限资源常量定义，MenuPermissionMap
- `internal/services/role_service.go` — RoleService，角色 CRUD，权限分配逻辑
- `internal/middleware/permission.go` — 权限中间件，Required() 装饰器

### Existing data ownership
- `internal/models/video_file.go` — VideoFile.CreatedBy 字段（第 26 行）
- `internal/models/video_recording_task.go` — VideoRecordingTask.CreatedBy 字段（第 34 行）

### Prior phases context
- `.planning/phases/01-video-splitting/01-CONTEXT.md` — VideoFile 来源类型，文件列表模式
- `.planning/phases/04-cloud-services/04-CONTEXT.md` — 审计日志系统

No external specs — requirements are fully captured in existing codebase.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **RoleService** (`internal/services/role_service.go`): 角色 CRUD 逻辑，AssignPermissions() 方法展示多对多关联操作模式（角色-权限）
- **User.HasPermission()** (`internal/models/user.go`): 当前单角色权限检查，需要修改为遍历多个角色
- **Permission Middleware** (`internal/middleware/permission.go`): Required() 装饰器，从上下文获取用户并检查权限
- **GORM Migration Pattern**: 项目使用 GORM 自动迁移，但多对多表需要显式定义模型

### Established Patterns
- **GORM Many-to-Many**: Role ↔ Permission 已使用 `gorm:"many2many:role_permissions;"` 模式
- **权限常量**: permission_constants.go 定义所有资源:action 格式的权限字符串
- **预定义角色**: RoleAdmin, RoleOperator, RoleViewer, RoleAPIClient 常量
- **软删除**: Base 模型包含 DeletedAt，所有主表支持软删除

### Integration Points
- **User Model**: 删除 RoleID 字段，添加 Roles []Role 关联，更新 HasPermission() 方法
- **UsersRoles 模型**: 新建关联表模型，可能需要 UserRole struct 或仅依赖 GORM 自动创建
- **权限中间件**: 可能需要更新以处理多角色情况（虽然现有逻辑遍历即可工作）
- **用户管理前端**: 用户角色选择 UI 从单选改为多选
- **角色管理前端**: 添加 shared_viewer 到预定义角色列表
- **数据查询层**: VideoFileService, RecordingTaskService 等需要检查用户是否有 shared_viewer 角色

### Data Ownership Filter Locations
需要修改查询以支持共享查看者的服务：
- `internal/services/video_file_service.go` — ListFiles() 方法
- `internal/services/*_service.go` — 任何按 CreatedBy 过滤数据的列表查询

</code_context>

<specifics>
## Specific Ideas

- 共享查看者是一个"可见性提升器"角色，而不是"操作权限"角色
- 数据库迁移需要特别小心，确保现有用户不丢失原有角色
- 前端角色选择应该显示"共享查看者"角色的特殊性质（可能用不同颜色或图标标记）
- 考虑在用户列表中为拥有共享查看者角色的用户显示特殊标识

</specifics>

<deferred>
## Deferred Ideas

- 用户可以"申请"共享查看者角色的审批流程（未来增强）
- 更细粒度的数据共享（如按部门、项目共享，而非全系统）
- 临时共享查看者权限（有时限的访问提升）

</deferred>

---

*Phase: 09-multi-role-permissions*
*Context gathered: 2026-04-21*
