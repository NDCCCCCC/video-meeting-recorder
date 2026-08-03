package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// TestHandleError_RecognizedSentinel 验证已知 sentinel 被映射并写入响应，返回 true。
func TestHandleError_RecognizedSentinel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	recognized := HandleError(c, errors.ErrNotFound)
	assert.True(t, recognized, "ErrNotFound 是已知 sentinel，应返回 true")
	assert.Equal(t, http.StatusNotFound, w.Code, "应写入 404")

	var body Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, CodeNotFound, body.Code, "响应体 code 应为 CodeNotFound")
}

// TestHandleError_BusinessError 按 Code 映射并写入。
func TestHandleError_BusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	be := errors.NewBusinessError(errors.CodeAlreadyExists, "已存在", nil)
	recognized := HandleError(c, be)
	assert.True(t, recognized)
	// 409 Conflict（GinError 的 switch 不识别 CodeDuplicateRecord，HandleError 用 GinErrorWithStatus 显式指定）。
	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestHandleError_ConflictStatusExplicit 验证 ErrAlreadyExists → 409（非 GinError 默认 500）。
// 这是 GinError switch 的已知缺口，HandleError 通过 GinErrorWithStatus 修复。
func TestHandleError_ConflictStatusExplicit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	HandleError(c, errors.ErrAlreadyExists)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestHandleError_UnknownError 验证未知错误 → 500 但返回 false（调用方可选择自行处理）。
func TestHandleError_UnknownError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	recognized := HandleError(c, errUnknown("something weird"))
	assert.False(t, recognized, "未知错误应返回 false")
	assert.Equal(t, http.StatusInternalServerError, w.Code, "保守策略：未知 → 500")
}

// TestHandleError_NoOpWhenNil 验证 err==nil 时不写入，返回 false。
func TestHandleError_NoOpWhenNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	recognized := HandleError(c, nil)
	assert.False(t, recognized)
	assert.False(t, c.Writer.Written(), "err==nil 时不应写入响应")
}

// TestHandleError_NoOpWhenAlreadyWritten 验证防双写：响应已写入时 no-op。
func TestHandleError_NoOpWhenAlreadyWritten(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	// 先写一个响应。
	GinSuccess(c, "already done")
	firstBody := w.Body.String()

	recognized := HandleError(c, errors.ErrNotFound)
	assert.False(t, recognized, "已写入时应返回 false")
	// body 不应被覆盖。
	assert.Equal(t, firstBody, w.Body.String())
}

// errUnknown 是一个非 sentinel、非 BusinessError 的测试错误。
type errUnknown string //nolint:errname // 测试 helper error，无需 xxxError 后缀

func (e errUnknown) Error() string { return string(e) }
