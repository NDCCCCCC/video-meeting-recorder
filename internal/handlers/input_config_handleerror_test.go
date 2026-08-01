package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestInputConfigHandler_HandleError_ClassifyReplacement verifies the converged
// input_config_handler error paths route through response.HandleError correctly
// for all 4 error classes (Phase 20 D-02.5).
//
// After convergence, service-call failures map via mapping.go instead of being
// hardcoded to CodeInvalidRequest. ShouldBindJSON parse errors remain
// gin-style GinError(CodeInvalidRequest, "请求参数错误: "+err.Error()) — those
// are the canonical Gin pattern, not classify scatter.
func TestInputConfigHandler_HandleError_ClassifyReplacement(t *testing.T) {
	type body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	parseBody := func(t *testing.T, rec *httptest.ResponseRecorder) body {
		t.Helper()
		var b body
		if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
			t.Fatalf("decode response body: %v (raw=%s)", err, rec.Body.String())
		}
		return b
	}

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{
			name:       "(i) sentinel direct: ErrInvalidInput → 400",
			err:        apperrors.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(ii) sentinel wrapped: ErrAlreadyExists → 409 Conflict",
			err:        fmt.Errorf("input config exists: %w", apperrors.ErrAlreadyExists),
			wantStatus: http.StatusConflict,
			wantCode:   response.CodeDuplicateRecord,
		},
		{
			name:       "(iii) BusinessError(CodeNotFound) → 404",
			err:        apperrors.NewBusinessError(apperrors.CodeNotFound, "配置不存在", nil),
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name:       "(iv) unknown ad-hoc error → 500",
			err:        errors.New("随机内部错误"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			response.HandleError(ctx, tt.err)
			assert.Equal(t, tt.wantStatus, rec.Code,
				"HTTP status mismatch: HandleError(%v) → %d, want %d",
				tt.err, rec.Code, tt.wantStatus)
			got := parseBody(t, rec)
			assert.Equal(t, tt.wantCode, got.Code,
				"response code mismatch: HandleError(%v) → code %d, want %d",
				tt.err, got.Code, tt.wantCode)
		})
	}
}
