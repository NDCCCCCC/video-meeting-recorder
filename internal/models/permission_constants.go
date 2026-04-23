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
	ResourceFileView      = "files:view"
	ResourceFileEdit      = "files:edit"
	ResourceFileDelete    = "files:delete"
	ResourceFileScan      = "files:scan"
	ResourceFileSplit     = "files:split"
	ResourceFileTranscribe = "files:transcribe"
	ResourceFilePPTView   = "files:ppt_view" // 匹配前端 FILE_PPT_VIEW

	// PPT 文件（保留旧的权限字符串以兼容，但新增前端使用的权限）
	ResourcePPTView     = "ppts:view"
	ResourcePPTDelete   = "ppts:delete"
	ResourcePPTEdit     = "ppts:edit"
	ResourcePPTDownload = "ppts:download"

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
	ResourceFileEdit,
	ResourceFileDelete,
	ResourceFileScan,
	ResourceFileSplit,
	ResourceFileTranscribe,
	ResourceFilePPTView,
	ResourcePPTView,
	ResourcePPTDelete,
	ResourcePPTEdit,
	ResourcePPTDownload,
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
