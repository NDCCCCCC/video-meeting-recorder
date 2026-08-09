// Package errors defines the project's sentinel / typed-error vocabulary.
//
//go:generate go run ../../cmd/error-doc-gen -errors-file errors.go -mapping-file mapping.go -output ../../docs/errors.md -repo-root ../..
package errors

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	// 通用错误
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInternal      = errors.New("internal server error")

	// 凭据/安全 sentinels：凭据加载/解密/鉴权失败时返回这些常量，而非 %w 包裹可能携带
	// 敏感信息（Password/SM4 key）的底层错误 —— 避免敏感数据经日志或错误链泄露（CodeQL
	// clear-text-logging 告警）。调用方仍可经 errors.Is 判定错误类别。
	ErrCredentialAccess = errors.New("credential access failure")
	ErrHuaweiAuthFailed = errors.New("huawei terminal authentication failed")

	// 业务错误
	ErrVideoFileNotFound   = errors.New("video file not found")
	ErrTaskNotFound        = errors.New("task not found")
	ErrTaskInProgress      = errors.New("task already in progress")
	ErrInvalidFileType     = errors.New("invalid file type")
	ErrFFmpegFailed        = errors.New("ffmpeg operation failed")
	ErrTranscriptionFailed = errors.New("transcription failed")
	ErrSplitFailed         = errors.New("split operation failed")
	ErrInsufficientQuota   = errors.New("insufficient quota")
	ErrServiceUnavailable  = errors.New("service temporarily unavailable")
	// ErrDuplicateRecord 唯一约束失败（如 UNIQUE 冲突）
	ErrDuplicateRecord = errors.New("duplicate record")
	// ErrForeignKeyConstraint 外键约束失败（如 taskID 不存在于 VideoRecordingTask 表）。
	// 用于 diagnostic-only 路径：service 检测到外键违反时区分 duplicate 与 FK 失败做差异化日志。
	ErrForeignKeyConstraint = errors.New("foreign key constraint failed")

	// 用户/角色管理 sentinels（Phase 19 D5）：user_service 高频错误路径统一化。
	// handler 改用 response.HandleError 后，这些 sentinel 决定 HTTP 状态码（404/409/403）。
	ErrUserNotFound         = errors.New("user not found")
	ErrUsernameExists       = errors.New("username already taken")
	ErrEmailExists          = errors.New("email already in use")
	ErrRoleNotFound         = errors.New("role not found")
	ErrSystemAdminProtected = errors.New("system admin protected from modification")

	// 认证 sentinels（Phase 19 D6, Phase 20 R-3 migration）：ad_auth + local_auth Login 路径统一化。
	// Login handler 已迁移到 response.HandleError (Phase 20 20-02)；mapping 后端以 sentinel 形式
	// 覆盖所有 Login 错误模式 (errors.Is 链)。
	// Auth 路径的密码错统一映射 401 Unauthorized，已存在 ErrUnauthorized sentinel 直接复用，
	// AD 账号不存在映射 404，被禁用映射 403，未注册白名单映射 403。
	ErrADAccountNotFound = errors.New("AD account not found")
	ErrUserDisabled      = errors.New("user disabled")
	ErrADConfigError     = errors.New("AD configuration error")
	ErrADUnreachable     = errors.New("AD server unreachable")
	// ErrADUserNotRegistered is returned when an AD-authenticated user is not
	// present in the local user table and auto-create is disabled. Mapped to
	// 403 (Phase 20 R-3 migration) so the auth login path can route through
	// HandleError without falling through to a 500 ad-hoc response.
	ErrADUserNotRegistered = errors.New("账号未在系统中注册，请联系管理员添加")

	// Token sentinels（Phase 19 D7）：sm4_token.go 高频 token 校验路径统一化。
	// middleware/auth.go 调 SM4TokenService.ValidateToken 时返回 sentinel；
	// mapping 全部走 401（与 ErrUnauthorized 同 status），但 message 区分供 frontend UX。
	// Invalid 覆盖 base64 解码、SM4-GCM Open 失败、JSON 反序列化失败、Token 类型不匹配；
	// Expired / NotYetValid 区分时间窗口；Replayed 检测 Refresh Token 宽限期外的重用。
	ErrTokenInvalid     = errors.New("token invalid")
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenNotYetValid = errors.New("token not yet valid")
	ErrTokenReplayed    = errors.New("token reuse detected")

	// Role sentinels（Phase 19 D9）：role_service admin 路径统一化。
	// 与 D5/D6/D7 sentinel 设计对齐——ErrRoleNotFound (D5) 已在 404 case，
	// 在此补充 role 特有状态。role_service 9 散点全部 sentinel 化。
	ErrRoleNameExists      = errors.New("role name already exists")
	ErrSystemRoleProtected = errors.New("system role protected from deletion")
	ErrRoleInUse           = errors.New("role still in use by users")
	ErrPermissionNotFound  = errors.New("permission not found")

	// APIKey sentinels（Phase 19 D10）：apikey_service admin 路径统一化。
	// apikey 验证路径（ValidateAPIKey）与 D7 token 校验类似——区分 NotFound / Invalid /
	// Expired / Disabled / IP-Not-Allowed。错误经 IsKnownError 链调用方分支。
	ErrAPIKeyNotFound     = errors.New("API key not found")
	ErrAPIKeyInvalid      = errors.New("API key invalid")
	ErrAPIKeyExpired      = errors.New("API key expired")
	ErrAPIKeyDisabled     = errors.New("API key disabled")
	ErrAPIKeyIPNotAllowed = errors.New("API key IP not allowed")

	// PPT file sentinels（Phase 19 D13）：ppt_file_service.go admin 与 handler 路径统一化。
	// ppt_file_service.go 仅 1 个 "PPT文件不存在" 适合 sentinel；其他内部错已用 %w
	// 或 BusinessError（D4 已迁移）；ppt_handler.go 的 10+ 处 "PPT文件不存在"
	// GinError 切到 HandleError 后经 sentinel 链识别为 404。
	ErrPPTFileNotFound = errors.New("PPT file not found")

	// Transcription sentinels（Phase 19 D14）：tingwu_client.go 外部 API 错误隔离。
	// ErrTranscriptionFailed (D5, 500 兜底) 复用覆盖 HTTP/parse 类错误;
	// ErrTranscriptionUnavailable (503) 用于 service 未启用 / URL 白名单错
	// ——这些是配置/环境问题而非传输问题。
	ErrTranscriptionUnavailable = errors.New("transcription service unavailable")

	// Smart-end recording sentinels (Phase 23 AUDIT-05):
	// 录制智能延长失败（录像时长已到，自动延长失败）
	ErrRecordingSmartExtend = errors.New("recording smart extend failed")
	// 录制智能早停失败（提前停止条件触发，停止失败）
	ErrRecordingSmartEarlyEnd = errors.New("recording smart early end failed")
	// 华为会议状态获取失败（拉取实时会议状态/参会人失败）
	ErrRecordingHuaWeiStateFetchFailed = errors.New("Huawei conference state fetch failed")
)

// Wrap 包装错误，添加上下文信息
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf 包装错误，支持格式化
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// Is 检查错误链中是否包含目标错误
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As 获取错误链中指定类型的错误
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// BusinessError 业务错误，包含错误码
type BusinessError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error 实现 error 接口
func (e *BusinessError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap 实现 errors.Unwrap 接口
func (e *BusinessError) Unwrap() error {
	return e.Err
}

// NewBusinessError 创建业务错误
func NewBusinessError(code, message string, err error) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 常用错误码常量
const (
	CodeNotFound             = "NOT_FOUND"
	CodeAlreadyExists        = "ALREADY_EXISTS"
	CodeInvalidInput         = "INVALID_INPUT"
	CodeUnauthorized         = "UNAUTHORIZED"
	CodeForbidden            = "FORBIDDEN"
	CodeInternalError        = "INTERNAL_ERROR"
	CodeServiceUnavailable   = "SERVICE_UNAVAILABLE"
	CodeTaskInProgress       = "TASK_IN_PROGRESS"
	CodeFFmpegError          = "FFMPEG_ERROR"
	CodeForeignKeyConstraint = "FOREIGN_KEY_CONSTRAINT"
)
