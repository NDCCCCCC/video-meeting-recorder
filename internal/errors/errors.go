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
	ErrUserNotFound          = errors.New("user not found")
	ErrUsernameExists        = errors.New("username already taken")
	ErrEmailExists           = errors.New("email already in use")
	ErrRoleNotFound          = errors.New("role not found")
	ErrSystemAdminProtected  = errors.New("system admin protected from modification")
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
