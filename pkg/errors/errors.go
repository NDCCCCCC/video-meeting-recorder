package errors

import (
	"fmt"
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

// New 创建新的业务错误
func New(code, message string) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装已有错误
func Wrap(err error, code, message string) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: fmt.Sprintf("%s: %v", message, err),
	}
}
