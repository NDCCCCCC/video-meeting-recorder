package common

import (
	"context"
)

// Service 服务接口
// 所有服务必须实现此接口以实现统一的生命周期管理
type Service interface {
	// Initialize 初始化服务
	Initialize() error

	// Start 启动服务
	Start() error

	// Stop 停止服务
	Stop() error

	// Restart 重启服务
	Restart() error

	// HealthCheck 健康检查
	HealthCheck() error

	// GetName 获取服务名称
	GetName() string

	// GetType 获取服务类型
	GetType() string

	// GetStatus 获取服务状态
	GetStatus() ServiceStatus

	// GetContext 获取服务上下文
	GetContext() context.Context

	// SetContext 设置服务上下文
	SetContext(ctx context.Context)
}

// ServiceStatus 服务状态
type ServiceStatus int

const (
	StatusStopped   ServiceStatus = iota // 已停止
	StatusStarting                       // 启动中
	StatusRunning                        // 运行中
	StatusStopping                       // 停止中
	StatusError                          // 错误状态
)

// String 返回状态的字符串表示
func (s ServiceStatus) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusStopping:
		return "stopping"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// BaseService 基础服务实现
type BaseService struct {
	Name   string
	Type   string
	Status ServiceStatus
	Ctx    context.Context
	cancel context.CancelFunc
}

// NewBaseService 创建基础服务
func NewBaseService(name, serviceType string) *BaseService {
	ctx, cancel := context.WithCancel(context.Background())
	return &BaseService{
		Name:   name,
		Type:   serviceType,
		Status: StatusStopped,
		Ctx:    ctx,
		cancel: cancel,
	}
}

// GetName 获取服务名称
func (s *BaseService) GetName() string {
	return s.Name
}

// GetType 获取服务类型
func (s *BaseService) GetType() string {
	return s.Type
}

// GetStatus 获取服务状态
func (s *BaseService) GetStatus() ServiceStatus {
	return s.Status
}

// GetContext 获取服务上下文
func (s *BaseService) GetContext() context.Context {
	return s.Ctx
}

// SetContext 设置服务上下文
func (s *BaseService) SetContext(ctx context.Context) {
	s.Ctx = ctx
}

// Initialize 初始化服务（默认实现）
func (s *BaseService) Initialize() error {
	s.Status = StatusStarting
	return nil
}

// Start 启动服务（默认实现）
func (s *BaseService) Start() error {
	s.Status = StatusRunning
	return nil
}

// Stop 停止服务（默认实现）
func (s *BaseService) Stop() error {
	s.Status = StatusStopping
	if s.cancel != nil {
		s.cancel()
	}
	s.Status = StatusStopped
	return nil
}

// Restart 重启服务（默认实现）
func (s *BaseService) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

// HealthCheck 健康检查（默认实现）
func (s *BaseService) HealthCheck() error {
	if s.Status != StatusRunning {
		return ErrServiceNotRunning
	}
	return nil
}

// Repository 仓储接口
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uint) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, options *ListOptions) ([]*T, int64, error)
}

// ListOptions 列表查询选项
type ListOptions struct {
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	OrderBy  string              `json:"order_by"`
	Order    string              `json:"order"` // "asc" or "desc"
	Filters  map[string]interface{} `json:"filters"`
}

// 错误定义
var (
	ErrServiceNotRunning = &BusinessError{
		Code:    "SERVICE_NOT_RUNNING",
		Message: "service is not running",
	}
	ErrNotFound        = &BusinessError{
		Code:    "NOT_FOUND",
		Message: "resource not found",
	}
	ErrAlreadyExists   = &BusinessError{
		Code:    "ALREADY_EXISTS",
		Message: "resource already exists",
	}
	ErrUnauthorized    = &BusinessError{
		Code:    "UNAUTHORIZED",
		Message: "unauthorized access",
	}
	ErrForbidden       = &BusinessError{
		Code:    "FORBIDDEN",
		Message: "forbidden access",
	}
	ErrInvalidInput    = &BusinessError{
		Code:    "INVALID_INPUT",
		Message: "invalid input parameters",
	}
)

// BusinessError 业务错误
type BusinessError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Error 实现error接口
func (e *BusinessError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

// WithDetails 添加错误详情
func (e *BusinessError) WithDetails(details any) *BusinessError {
	return &BusinessError{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
	}
}
