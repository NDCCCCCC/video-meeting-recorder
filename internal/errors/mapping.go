package errors

import (
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

// 这些 respCode 值镜像 pkg/response 中的常量（CodeInvalidRequest=1001,
// CodeUnauthorized=1002, ...）。在此本地定义而非 import pkg/response，
// 以避免 internal/errors ↔ pkg/response 循环依赖（response.HandleError 需要
// 调用 errors.MapToHTTPStatus）。数值保持同步，新增时务必同步两边。
const (
	respCodeInvalidRequest  = 1001
	respCodeUnauthorized    = 1002
	respCodeForbidden       = 1003
	respCodeNotFound        = 1004
	respCodeInternalError   = 1005
	respCodeDuplicateRecord = 1006
	respCodeTooManyRequests = 1007
)

// MapToHTTPStatus 将 sentinel 或 BusinessError 错误映射到 HTTP 状态码 + 响应码 + 消息。
//
// 保守策略：未识别的错误一律映射到 500/respCodeInternalError（永不返回 200），
// 避免把真实故障误判为成功。err==nil 返回 (200, 0, "")。
//
// 映射顺序：BusinessError（typed，按 Code 字段）优先，其次 sentinel（errors.Is 链匹配，
// 兼容 Wrap/Wrapf 的 %w 包装）。
func MapToHTTPStatus(err error) (httpStatus int, respCode int, message string) {
	if err == nil {
		return http.StatusOK, 0, ""
	}

	// BusinessError（typed）—— 优先按 Code 字段映射。
	var be *BusinessError
	if errors.As(err, &be) {
		return mapBusinessError(be)
	}

	// Sentinel 错误（errors.Is 链匹配，兼容 Wrap/Wrapf 包装）。
	switch {
	case errors.Is(err, ErrNotFound),
		errors.Is(err, ErrTaskNotFound),
		errors.Is(err, ErrVideoFileNotFound),
		errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrRoleNotFound),
		errors.Is(err, ErrADAccountNotFound),
		errors.Is(err, ErrPermissionNotFound),
		errors.Is(err, ErrAPIKeyNotFound),
		errors.Is(err, ErrPPTFileNotFound):
		return http.StatusNotFound, respCodeNotFound, "资源不存在"
	case errors.Is(err, ErrUnauthorized),
		errors.Is(err, ErrTokenInvalid),
		errors.Is(err, ErrTokenExpired),
		errors.Is(err, ErrTokenNotYetValid),
		errors.Is(err, ErrTokenReplayed),
		errors.Is(err, ErrAPIKeyInvalid),
		errors.Is(err, ErrAPIKeyExpired):
		return http.StatusUnauthorized, respCodeUnauthorized, "未授权"
	case errors.Is(err, ErrForbidden),
		errors.Is(err, ErrSystemAdminProtected),
		errors.Is(err, ErrSystemRoleProtected),
		errors.Is(err, ErrUserDisabled),
		errors.Is(err, ErrAPIKeyDisabled),
		errors.Is(err, ErrAPIKeyIPNotAllowed):
		return http.StatusForbidden, respCodeForbidden, "禁止访问"
	case errors.Is(err, ErrInvalidInput),
		errors.Is(err, ErrInvalidFileType):
		return http.StatusBadRequest, respCodeInvalidRequest, "请求参数无效"
	case errors.Is(err, ErrAlreadyExists),
		errors.Is(err, ErrTaskInProgress),
		errors.Is(err, ErrUsernameExists),
		errors.Is(err, ErrEmailExists),
		errors.Is(err, ErrRoleNameExists),
		errors.Is(err, ErrRoleInUse):
		// 409 Conflict——GinError 的 switch 不识别 CodeDuplicateRecord，会落到默认 500，
		// 故调用方必须用 GinErrorWithStatus 显式指定 409（HandleError 已如此）。
		return http.StatusConflict, respCodeDuplicateRecord, "资源已存在或状态冲突"
	case errors.Is(err, ErrInsufficientQuota):
		return http.StatusTooManyRequests, respCodeTooManyRequests, "配额不足"
	case errors.Is(err, ErrServiceUnavailable),
		errors.Is(err, ErrADConfigError),
		errors.Is(err, ErrADUnreachable),
		errors.Is(err, ErrTranscriptionUnavailable):
		return http.StatusServiceUnavailable, respCodeInternalError, "服务暂不可用"
	case errors.Is(err, ErrFFmpegFailed),
		errors.Is(err, ErrTranscriptionFailed),
		errors.Is(err, ErrSplitFailed):
		return http.StatusInternalServerError, respCodeInternalError, "内部处理失败"
	default:
		return http.StatusInternalServerError, respCodeInternalError, "内部服务器错误"
	}
}

// mapBusinessError 按 BusinessError.Code 映射到 HTTP 状态 + respCode。
// 未识别的 Code → 500（保守）。消息优先用 be.Message，空则回退 be.Error()。
func mapBusinessError(be *BusinessError) (int, int, string) {
	msg := be.Message
	if msg == "" {
		msg = be.Error()
	}
	switch be.Code {
	case CodeNotFound:
		return http.StatusNotFound, respCodeNotFound, msg
	case CodeAlreadyExists, CodeTaskInProgress:
		return http.StatusConflict, respCodeDuplicateRecord, msg
	case CodeInvalidInput:
		return http.StatusBadRequest, respCodeInvalidRequest, msg
	case CodeUnauthorized:
		return http.StatusUnauthorized, respCodeUnauthorized, msg
	case CodeForbidden:
		return http.StatusForbidden, respCodeForbidden, msg
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable, respCodeInternalError, msg
	case CodeFFmpegError, CodeInternalError:
		return http.StatusInternalServerError, respCodeInternalError, msg
	default:
		return http.StatusInternalServerError, respCodeInternalError, msg
	}
}

// IsKnownError 报告 err 是否匹配任一预定义 sentinel 或 BusinessError。
// 供 response.HandleError 区分"已识别 sentinel"与"未知错误"（决定返回值）。
func IsKnownError(err error) bool {
	if err == nil {
		return false
	}
	var be *BusinessError
	if errors.As(err, &be) {
		return true
	}
	for _, sentinel := range []error{
		ErrNotFound, ErrTaskNotFound, ErrVideoFileNotFound,
		ErrUserNotFound, ErrRoleNotFound, ErrADAccountNotFound, ErrPermissionNotFound,
		ErrAPIKeyNotFound, ErrPPTFileNotFound,
		ErrUnauthorized, ErrForbidden, ErrInvalidInput, ErrInvalidFileType,
		ErrAlreadyExists, ErrTaskInProgress, ErrUsernameExists, ErrEmailExists,
		ErrRoleNameExists, ErrRoleInUse,
		ErrSystemAdminProtected, ErrSystemRoleProtected, ErrUserDisabled,
		ErrAPIKeyDisabled, ErrAPIKeyIPNotAllowed,
		ErrTokenInvalid, ErrTokenExpired, ErrTokenNotYetValid, ErrTokenReplayed,
		ErrAPIKeyInvalid, ErrAPIKeyExpired,
		ErrInsufficientQuota,
		ErrServiceUnavailable, ErrADConfigError, ErrADUnreachable,
		ErrTranscriptionUnavailable,
		ErrFFmpegFailed, ErrTranscriptionFailed,
		ErrSplitFailed, ErrInternal,
		ErrDuplicateRecord, ErrForeignKeyConstraint,
	} {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}

// FromGORM 把 gorm 错误转换为 sentinel：gorm.ErrRecordNotFound → ErrNotFound，
// 其他错误返回 fallback（通常为原 err 或包装后的内部错误）。fallback 为 nil 时返回原 err。
//
// 用于 service 边界：防止 gorm 错误类型泄漏出 service 层（STYLE-001 决策 3）。
func FromGORM(err error, fallback error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if fallback != nil {
		return fallback
	}
	return err
}

// NotFound 构造一个带上下文的"资源不存在"错误（用 %w 包装 ErrNotFound，
// 使 errors.Is(err, ErrNotFound) 成立）。
// 示例：NotFound("task", id) → "task <id> not found: resource not found"。
func NotFound(what string, id any) error {
	return fmt.Errorf("%s %v not found: %w", what, id, ErrNotFound)
}
