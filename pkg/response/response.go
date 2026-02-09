package response

import (
	"encoding/json"
	"net/http"
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
	CodeUnauthorized       = 1002
	CodeForbidden         = 1003
	CodeNotFound          = 1004
	CodeInternalError     = 1005
	CodeDuplicateRecord   = 1006
	CodeInvalidCredential = 2001
	CodeUserNotFound      = 2002
	CodeUserExists        = 2003
	CodeInvalidPassword   = 2004
	CodeInvalidToken      = 2005
	CodeTokenExpired      = 2006
)

// Success 成功响应
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

// Error 错误响应
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

// ErrorWithStatus 带自定义状态码的错误响应
func ErrorWithStatus(w http.ResponseWriter, httpStatus int, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	resp := Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}

	json.NewEncoder(w).Encode(resp)
}

// Created 创建成功响应
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
