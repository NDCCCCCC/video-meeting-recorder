// Package common 公共类型与工具。
//
// 历史遗留:interfaces.go 曾定义通用 Service 接口 (Initialize/Start/Stop/Restart/...)
// 和 BaseService 抽象基类 — STYLE-003 Phase 21 审计发现:
//   - common.Service 接口定义的 11 个方法 (含 GetContext/SetContext/ServiceStatus)
//     从未被任何生产服务实现; services map[string]common.Service 字段因此成为 dead store。
//   - BaseService 仅被 NewBaseService 自身引用, 无使用方。
//
// FU-7 决策:整个 Service + ServiceStatus + BaseService 块**删除** (Go 惯例 = 不用
// 接口不写接口; 日后真有通用 lifecycle 需求时按 "consumer defines interface"
// 在 cmd/server 包内联定义即可)。
//
// 保留: Repository[T] / ListOptions / BusinessError / Error helper,因为这些仍被
// 仓库层泛型使用 (后续如发现也未使用,再清理)。
package common

import (
	"context"
)

// Repository 仓储接口（保留）
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uint) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, options *ListOptions) ([]*T, int64, error)
}

// ListOptions 列表查询选项（保留）
type ListOptions struct {
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	OrderBy  string                 `json:"order_by"`
	Order    string                 `json:"order"` // "asc" or "desc"
	Filters  map[string]interface{} `json:"filters"`
}

// 错误定义（保留；apikey/admin 等仍在用）。
var (
	ErrNotFound = &BusinessError{
		Code:    "NOT_FOUND",
		Message: "resource not found",
	}
	ErrAlreadyExists = &BusinessError{
		Code:    "ALREADY_EXISTS",
		Message: "resource already exists",
	}
	ErrUnauthorized = &BusinessError{
		Code:    "UNAUTHORIZED",
		Message: "unauthorized access",
	}
	ErrForbidden = &BusinessError{
		Code:    "FORBIDDEN",
		Message: "forbidden access",
	}
	ErrInvalidInput = &BusinessError{
		Code:    "INVALID_INPUT",
		Message: "invalid input parameters",
	}
)

// BusinessError 业务错误（保留）。
type BusinessError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Error 实现 error 接口。
func (e *BusinessError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

// WithDetails 添加错误详情。
func (e *BusinessError) WithDetails(details any) *BusinessError {
	return &BusinessError{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
	}
}
