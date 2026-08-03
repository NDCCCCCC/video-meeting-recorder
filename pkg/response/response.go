package response

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 错误码定义
const (
	CodeSuccess           = 0
	CodeInvalidRequest    = 1001
	CodeUnauthorized      = 1002
	CodeForbidden         = 1003
	CodeNotFound          = 1004
	CodeInternalError     = 1005
	CodeDuplicateRecord   = 1006
	CodeTooManyRequests   = 1007
	CodeInvalidCredential = 2001
	CodeUserNotFound      = 2002
	CodeUserExists        = 2003
	CodeInvalidPassword   = 2004
	CodeInvalidToken      = 2005
	CodeTokenExpired      = 2006
)

// Success 成功响应 (http.ResponseWriter)
func Success(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := Response{
		Code:    CodeSuccess,
		Message: "操作成功",
		Data:    data,
	}

	json.NewEncoder(w).Encode(resp)
}

// Error 错误响应 (http.ResponseWriter)
func Error(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")

	httpStatus := http.StatusOK
	switch code {
	case CodeUnauthorized:
		httpStatus = http.StatusUnauthorized
	case CodeForbidden:
		httpStatus = http.StatusForbidden
	case CodeNotFound:
		httpStatus = http.StatusNotFound
	case CodeInvalidRequest:
		httpStatus = http.StatusBadRequest
	default:
		httpStatus = http.StatusInternalServerError
	}

	w.WriteHeader(httpStatus)

	resp := Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}

	json.NewEncoder(w).Encode(resp)
}

// ErrorWithStatus 带自定义状态码的错误响应 (http.ResponseWriter)
func ErrorWithStatus(w http.ResponseWriter, httpStatus, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	resp := Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}

	json.NewEncoder(w).Encode(resp)
}

// Created 创建成功响应 (http.ResponseWriter)
func Created(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := Response{
		Code:    CodeSuccess,
		Message: "创建成功",
		Data:    data,
	}

	json.NewEncoder(w).Encode(resp)
}

// ========== Gin框架兼容函数 ==========

// GinSuccess Gin框架成功响应
func GinSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "操作成功",
		Data:    data,
	})
}

// GinError Gin框架错误响应
func GinError(c *gin.Context, code int, message string) {
	httpStatus := http.StatusOK
	switch code {
	case CodeUnauthorized:
		httpStatus = http.StatusUnauthorized
	case CodeForbidden:
		httpStatus = http.StatusForbidden
	case CodeNotFound:
		httpStatus = http.StatusNotFound
	case CodeInvalidRequest:
		httpStatus = http.StatusBadRequest
	case CodeTooManyRequests:
		httpStatus = http.StatusTooManyRequests
	default:
		httpStatus = http.StatusInternalServerError
	}

	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// GinErrorWithStatus Gin框架带自定义状态码的错误响应
func GinErrorWithStatus(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// GinCreated Gin框架创建成功响应
func GinCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    CodeSuccess,
		Message: "创建成功",
		Data:    data,
	})
}

// HandleError 把 err 通过 errors.MapToHTTPStatus 映射为 HTTP 响应并写入。
//
// STYLE-001 (Phase 19) 决策 3 组件 B：handler 把内联 switch 换成
// `if response.HandleError(c, err) { return }`——命中已知 sentinel 即写入响应并返回 true；
// 未识别的错误也写入（保守 500，永不 200）但返回 false，调用方可选择继续自行处理。
//
// 守卫：
//   - err==nil → no-op，返回 false。
//   - c.Writer.Written() 已为 true（handler 已写响应）→ no-op，返回 false（防双写）。
//
// 用 GinErrorWithStatus 显式指定 httpStatus，因为 GinError 的 switch 不识别
// CodeDuplicateRecord（会落到默认 500，而 409 Conflict 需要显式状态码）。
func HandleError(c *gin.Context, err error) bool {
	if err == nil || c.Writer.Written() {
		return false
	}
	httpStatus, respCode, message := errors.MapToHTTPStatus(err)
	GinErrorWithStatus(c, httpStatus, respCode, message)
	return errors.IsKnownError(err)
}
