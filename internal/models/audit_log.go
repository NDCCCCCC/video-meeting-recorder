package models

import (
	"encoding/json"
	"time"
)

// AuditLog 审计日志
type AuditLog struct {
	ID uint `gorm:"primarykey" json:"id"`

	// 用户信息
	UserID   uint   `gorm:"index" json:"user_id,omitempty"`
	Username string `gorm:"index" json:"username,omitempty"`
	RoleID   uint   `json:"role_id,omitempty"`
	RoleName string `json:"role_name,omitempty"`

	// 操作信息
	Action     string `gorm:"type:varchar(50);index;not null" json:"action"`
	Module     string `gorm:"type:varchar(50);index;not null" json:"module"`
	Resource   string `gorm:"type:varchar(100)" json:"resource"`
	ResourceID *uint  `json:"resource_id,omitempty"`

	// 请求上下文
	RequestID string `gorm:"type:varchar(64);index" json:"request_id,omitempty"`
	TraceID   string `gorm:"type:varchar(64)" json:"trace_id,omitempty"`
	Method    string `gorm:"type:varchar(10)" json:"method,omitempty"`
	Path      string `gorm:"type:varchar(500)" json:"path,omitempty"`

	// 变更内容
	OldData  string `gorm:"type:text" json:"old_data,omitempty"`
	NewData  string `gorm:"type:text" json:"new_data,omitempty"`
	DiffData string `gorm:"type:text" json:"diff_data,omitempty"`

	// 执行结果
	Status    string `gorm:"type:varchar(20);index" json:"status"`
	ErrorMsg  string `gorm:"type:text" json:"error_msg,omitempty"`
	ErrorCode string `gorm:"type:varchar(20)" json:"error_code,omitempty"`

	// 环境信息
	IPAddress string `gorm:"type:varchar(50)" json:"ip_address,omitempty"`
	UserAgent string `gorm:"type:varchar(500)" json:"user_agent,omitempty"`

	// 时间信息
	Duration  int64     `gorm:"default:0" json:"duration"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`

	Base
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}

// 操作类型常量
const (
	// 认证操作
	ActionLogin               = "login"
	ActionLogout              = "logout"
	ActionLogoutAll           = "logout_all"
	ActionRefresh             = "refresh"
	ActionPasswordChange      = "password_change"
	ActionIPRestrictionFailed = "ip_restriction_failed"

	// CRUD操作
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionQuery  = "query"

	// 业务操作
	ActionExport  = "export"
	ActionImport  = "import"
	ActionExecute = "execute"
	ActionCancel  = "cancel"
	ActionApprove = "approve"
	ActionReject  = "reject"
)

// 模块类型常量
const (
	ModuleUser       = "user"
	ModuleRole       = "role"
	ModuleTask       = "task"
	ModuleConference = "conference"
	ModuleRecording  = "recording"
	ModuleFile       = "file"
	ModuleSystem     = "system"
	ModuleConfig     = "config"

	// 扩展模块常量（与 cmd/server/app.go 中 auditOp 调用对齐）
	ModuleAPIKey        = "apikey"
	ModuleInputConfig   = "input_config"
	ModulePPT           = "ppt"
	ModuleStorage       = "storage"
	ModuleNotification = "notification"
	ModuleVideo         = "video"
	ModuleTranscription = "transcription"
	ModuleAdmin         = "auth"
)

// 状态常量
const (
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusPartial = "partial"
)

// AuditLogData 审计日志数据
type AuditLogData struct {
	OldData interface{} `json:"old_data,omitempty"`
	NewData interface{} `json:"new_data,omitempty"`
	Diff    interface{} `json:"diff,omitempty"`
}

// MarshalJSON 序列化
func (a *AuditLogData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		OldData interface{} `json:"old_data,omitempty"`
		NewData interface{} `json:"new_data,omitempty"`
		Diff    interface{} `json:"diff,omitempty"`
	}{
		OldData: a.OldData,
		NewData: a.NewData,
		Diff:    a.Diff,
	})
}

// GetOldData 获取旧数据
func (a *AuditLog) GetOldData() map[string]interface{} {
	var data map[string]interface{}
	if a.OldData != "" {
		json.Unmarshal([]byte(a.OldData), &data)
	}
	return data
}

// GetNewData 获取新数据
func (a *AuditLog) GetNewData() map[string]interface{} {
	var data map[string]interface{}
	if a.NewData != "" {
		json.Unmarshal([]byte(a.NewData), &data)
	}
	return data
}

// GetDiffData 获取差异数据
func (a *AuditLog) GetDiffData() map[string]interface{} {
	var data map[string]interface{}
	if a.DiffData != "" {
		json.Unmarshal([]byte(a.DiffData), &data)
	}
	return data
}
