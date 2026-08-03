package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// TestErrorMapper_MapsUnwrittenError 验证：handler 通过 c.Error 记录错误但未写响应时，
// 中间件兜底映射为 HTTP 响应（已知 sentinel）。
func TestErrorMapper_MapsUnwrittenError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := ErrorMapper(zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/999", nil)

	// 模拟 handler 记录错误但未写响应。
	_ = c.Error(errors.ErrNotFound)

	mw(c)

	assert.Equal(t, http.StatusNotFound, w.Code, "backstop 应映射 ErrNotFound → 404")
}

// TestErrorMapper_MapsUnknownErrorTo500 验证未知错误 → 500。
func TestErrorMapper_MapsUnknownErrorTo500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := ErrorMapper(zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	_ = c.Error(errStr("weird internal failure"))

	mw(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestErrorMapper_NoOpWhenWritten 验证防双写：handler 已写响应时中间件不干预。
func TestErrorMapper_NoOpWhenWritten(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := ErrorMapper(zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	// handler 已写入响应。
	c.String(http.StatusOK, "handled inline")
	originalBody := w.Body.String()
	_ = c.Error(errors.ErrInternal) // 即使有未处理错误，也不应覆盖已有响应。

	mw(c)

	assert.Equal(t, http.StatusOK, w.Code, "不应覆盖 handler 已写的状态码")
	assert.Equal(t, originalBody, w.Body.String(), "不应覆盖 handler 已写的响应体")
}

// TestErrorMapper_NoOpWhenNoErrors 验证 c.Errors 为空时 no-op。
func TestErrorMapper_NoOpWhenNoErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := ErrorMapper(zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	mw(c)

	assert.False(t, c.Writer.Written(), "无错误时不应写入响应")
}

// errStr 是一个测试用的非 sentinel 错误。
type errStr string

func (e errStr) Error() string { return string(e) }
