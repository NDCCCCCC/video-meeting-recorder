package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestMapToHTTPStatus_Sentinels 穷举验证每个 sentinel → (httpStatus, respCode)。
// STYLE-001 (Phase 19) 决策 3 组件 A 的映射表契约。
func TestMapToHTTPStatus_Sentinels(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantResp   int
	}{
		// 404 / CodeNotFound
		{"ErrNotFound", ErrNotFound, http.StatusNotFound, respCodeNotFound},
		{"ErrTaskNotFound", ErrTaskNotFound, http.StatusNotFound, respCodeNotFound},
		{"ErrVideoFileNotFound", ErrVideoFileNotFound, http.StatusNotFound, respCodeNotFound},
		// 401 / CodeUnauthorized
		{"ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized, respCodeUnauthorized},
		// 403 / CodeForbidden
		{"ErrForbidden", ErrForbidden, http.StatusForbidden, respCodeForbidden},
		// 400 / CodeInvalidRequest
		{"ErrInvalidInput", ErrInvalidInput, http.StatusBadRequest, respCodeInvalidRequest},
		{"ErrInvalidFileType", ErrInvalidFileType, http.StatusBadRequest, respCodeInvalidRequest},
		// 409 / CodeDuplicateRecord
		{"ErrAlreadyExists", ErrAlreadyExists, http.StatusConflict, respCodeDuplicateRecord},
		{"ErrTaskInProgress", ErrTaskInProgress, http.StatusConflict, respCodeDuplicateRecord},
		// 429 / CodeTooManyRequests
		{"ErrInsufficientQuota", ErrInsufficientQuota, http.StatusTooManyRequests, respCodeTooManyRequests},
		// 503 / CodeInternalError
		{"ErrServiceUnavailable", ErrServiceUnavailable, http.StatusServiceUnavailable, respCodeInternalError},
		// 500 / CodeInternalError
		{"ErrFFmpegFailed", ErrFFmpegFailed, http.StatusInternalServerError, respCodeInternalError},
		{"ErrTranscriptionFailed", ErrTranscriptionFailed, http.StatusInternalServerError, respCodeInternalError},
		{"ErrSplitFailed", ErrSplitFailed, http.StatusInternalServerError, respCodeInternalError},
		{"ErrInternal", ErrInternal, http.StatusInternalServerError, respCodeInternalError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, resp, msg := MapToHTTPStatus(tc.err)
			assert.Equal(t, tc.wantStatus, status, "httpStatus")
			assert.Equal(t, tc.wantResp, resp, "respCode")
			assert.NotEmpty(t, msg, "message 不应为空")
		})
	}
}

// TestMapToHTTPStatus_WrappedSentinel 验证 %w 包装后的 sentinel 仍能被 errors.Is 匹配。
func TestMapToHTTPStatus_WrappedSentinel(t *testing.T) {
	wrapped := fmt.Errorf("查询任务失败: %w", ErrTaskNotFound)
	status, resp, _ := MapToHTTPStatus(wrapped)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, respCodeNotFound, resp)
}

// TestMapToHTTPStatus_BusinessError 按 Code 字段穷举 BusinessError 映射。
func TestMapToHTTPStatus_BusinessError(t *testing.T) {
	cases := []struct {
		code       string
		wantStatus int
		wantResp   int
	}{
		{CodeNotFound, http.StatusNotFound, respCodeNotFound},
		{CodeAlreadyExists, http.StatusConflict, respCodeDuplicateRecord},
		{CodeTaskInProgress, http.StatusConflict, respCodeDuplicateRecord},
		{CodeInvalidInput, http.StatusBadRequest, respCodeInvalidRequest},
		{CodeUnauthorized, http.StatusUnauthorized, respCodeUnauthorized},
		{CodeForbidden, http.StatusForbidden, respCodeForbidden},
		{CodeServiceUnavailable, http.StatusServiceUnavailable, respCodeInternalError},
		{CodeFFmpegError, http.StatusInternalServerError, respCodeInternalError},
		{CodeInternalError, http.StatusInternalServerError, respCodeInternalError},
		{"UNKNOWN_CODE", http.StatusInternalServerError, respCodeInternalError},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			be := NewBusinessError(tc.code, "业务错误: "+tc.code, nil)
			status, resp, msg := MapToHTTPStatus(be)
			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantResp, resp)
			assert.NotEmpty(t, msg)
		})
	}
}

// TestMapToHTTPStatus_BusinessErrorWithInnerErr 验证 BusinessError 包装内部 error 时
// 仍按 Code 映射（errors.As 优先于 sentinel）。
func TestMapToHTTPStatus_BusinessErrorWithInnerErr(t *testing.T) {
	inner := fmt.Errorf("db connection refused")
	be := NewBusinessError(CodeServiceUnavailable, "下游不可用", inner)
	status, resp, msg := MapToHTTPStatus(be)
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.Equal(t, respCodeInternalError, resp)
	assert.Contains(t, msg, "下游不可用")
}

// TestMapToHTTPStatus_UnknownError 验证未识别错误 → 500（保守，永不 200）。
func TestMapToHTTPStatus_UnknownError(t *testing.T) {
	err := fmt.Errorf("something weird")
	status, resp, msg := MapToHTTPStatus(err)
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, respCodeInternalError, resp)
	assert.NotEmpty(t, msg)
}

// TestMapToHTTPStatus_Nil 验证 nil → (200, 0, "")。
func TestMapToHTTPStatus_Nil(t *testing.T) {
	status, resp, msg := MapToHTTPStatus(nil)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, resp)
	assert.Empty(t, msg)
}

// TestIsKnownError 覆盖 sentinel / BusinessError / 未知 / nil。
func TestIsKnownError(t *testing.T) {
	assert.False(t, IsKnownError(nil))
	assert.True(t, IsKnownError(ErrNotFound))
	assert.True(t, IsKnownError(fmt.Errorf("wrap: %w", ErrForbidden)))
	assert.True(t, IsKnownError(NewBusinessError(CodeInvalidInput, "x", nil)))
	assert.False(t, IsKnownError(fmt.Errorf("unknown")))
	assert.False(t, IsKnownError(errors.New("std error")))
}

// TestFromGORM 覆盖 gorm.ErrRecordNotFound / 其他错误 / nil / fallback。
func TestFromGORM(t *testing.T) {
	t.Run("RecordNotFound→ErrNotFound", func(t *testing.T) {
		err := FromGORM(gorm.ErrRecordNotFound, ErrInternal)
		assert.True(t, errors.Is(err, ErrNotFound))
	})
	t.Run("其他 gorm 错误→fallback", func(t *testing.T) {
		other := fmt.Errorf("connection reset")
		err := FromGORM(other, ErrInternal)
		assert.True(t, errors.Is(err, ErrInternal))
	})
	t.Run("其他错误无 fallback→原样返回", func(t *testing.T) {
		other := fmt.Errorf("connection reset")
		err := FromGORM(other, nil)
		assert.Equal(t, other, err)
	})
	t.Run("nil→nil", func(t *testing.T) {
		assert.Nil(t, FromGORM(nil, ErrInternal))
	})
}

// TestNotFound 验证 NotFound 包装 ErrNotFound 且 errors.Is 成立。
func TestNotFound(t *testing.T) {
	err := NotFound("task", 42)
	assert.True(t, errors.Is(err, ErrNotFound), "NotFound 应包装 ErrNotFound")
	assert.Contains(t, err.Error(), "task")
	assert.Contains(t, err.Error(), "42")
	// 映射到 404。
	status, _, _ := MapToHTTPStatus(err)
	assert.Equal(t, http.StatusNotFound, status)
}

// TestIsKnownError_ForeignKey_Sentinel (Phase 19 D1) 验证 ErrForeignKeyConstraint
// 被 IsKnownError 识别 + 可被 errors.Is 链匹配。
func TestIsKnownError_ForeignKey_Sentinel(t *testing.T) {
	assert.True(t, IsKnownError(ErrForeignKeyConstraint))
	wrapped := fmt.Errorf("创建文件记录失败: %w", ErrForeignKeyConstraint)
	assert.True(t, IsKnownError(wrapped))
	assert.True(t, errors.Is(wrapped, ErrForeignKeyConstraint))
}

// TestDoubleWrap_ForeignKey_StillDetectable (Phase 19 D1) 验证 createWithDuplicateCheck
// 的 `fmt.Errorf("%w: %w", ErrForeignKeyConstraint, err)` 双 %w wrap 仍可被 errors.Is
// 检测到 ErrForeignKeyConstraint。
//（Go 1.20+ errors 支持 multi-%w，"双 %w" 链中两个目标都可被 errors.Is 匹配。）
func TestDoubleWrap_ForeignKey_StillDetectable(t *testing.T) {
	inner := errors.New("FOREIGN KEY constraint failed: SQLite/driver message")
	doubleWrapped := fmt.Errorf("%w: %w", ErrForeignKeyConstraint, inner)
	assert.True(t, errors.Is(doubleWrapped, ErrForeignKeyConstraint), "双 %w wrap 必须仍可 errors.Is 检测 ErrForeignKeyConstraint")
	assert.True(t, errors.Is(doubleWrapped, inner), "双 %w wrap 必须仍可 errors.Is 检测 inner err")
}
