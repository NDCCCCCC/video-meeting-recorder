package models

// 权限资源常量
const (
	// 仪表盘
	ResourceDashboardView = "dashboard:view"

	// 录制任务
	ResourceTaskView   = "tasks:view"
	ResourceTaskCreate = "tasks:create"
	ResourceTaskEdit   = "tasks:edit"
	ResourceTaskDelete = "tasks:delete"
	ResourceTaskStart  = "tasks:start"
	ResourceTaskStop   = "tasks:stop"

	// 视频文件
	ResourceFileView   = "files:view"
	ResourceFileDelete = "files:delete"
	ResourceFileScan   = "files:scan"

	// 审计日志
	ResourceAuditView = "audit:view"

	// 系统管理
	ResourceUserView   = "users:view"
	ResourceUserCreate = "users:create"
	ResourceUserEdit   = "users:edit"
	ResourceUserDelete = "users:delete"

	ResourceRoleView   = "roles:view"
	ResourceRoleCreate = "roles:create"
	ResourceRoleEdit   = "roles:edit"
	ResourceRoleDelete = "roles:delete"

	ResourceConfigView = "configs:view"
	ResourceConfigEdit = "configs:edit"

	// 系统设置
	ResourceSystemSettings = "system:settings"
)

// MenuPermissionMap 菜单权限映射
var MenuPermissionMap = map[string]string{
	"/dashboard":             ResourceDashboardView,
	"/tasks":                 ResourceTaskView,
	"/files":                 ResourceFileView,
	"/audit":                 ResourceAuditView,
	"/system/users":          ResourceUserView,
	"/system/roles":          ResourceRoleView,
	"/system/huawei-configs": ResourceConfigView,
	"/system/settings":       ResourceSystemSettings,
}

// AllPermissions 所有权限列表（用于权限分配）
var AllPermissions = []string{
	ResourceDashboardView,
	ResourceTaskView,
	ResourceTaskCreate,
	ResourceTaskEdit,
	ResourceTaskDelete,
	ResourceTaskStart,
	ResourceTaskStop,
	ResourceFileView,
	ResourceFileDelete,
	ResourceFileScan,
	ResourceAuditView,
	ResourceUserView,
	ResourceUserCreate,
	ResourceUserEdit,
	ResourceUserDelete,
	ResourceRoleView,
	ResourceRoleCreate,
	ResourceRoleEdit,
	ResourceRoleDelete,
	ResourceConfigView,
	ResourceConfigEdit,
	ResourceSystemSettings,
}
