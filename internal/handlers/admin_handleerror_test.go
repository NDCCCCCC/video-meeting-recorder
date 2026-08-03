package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
)

// TestAdminHandler_HandleError_ClassifyReplacement verifies that the converged
// admin_handler error paths route through response.HandleError correctly across
// the 4 error classes (Phase 20 D-02.5):
//
//	(i)   sentinel direct
//	(ii)  sentinel wrapped via %w
//	(iii) *apperrors.BusinessError
//	(iv)  unknown ad-hoc error
//
// The handler now calls `if response.HandleError(c, err) { return }` after each
// service call (SaveAuthConfig / LookupUser / MigrateInputConfigs Count+Fetch+
// Commit). ShouldBindJSON parse-error sites are preserved as the canonical Gin
// pattern and are NOT counted as scatter per D-02.4.
//
// Status-code expectations mirror internal/errors/mapping.go (Phase 19 D4):
//
//	ErrInvalidInput              → 400 / 1001 (CodeInvalidRequest)
//	%w apperrors.ErrServiceUnavailable → 503 / 1005
//	BusinessError(CodeInvalidInput) → 400 / 1001
//	unknown                      → 500 / 1005 (CodeInternalError)
//
// Note: existing admin_ad_test.go is preserved (it contains only stub tests
// with no assertions).
func TestAdminHandler_HandleError_ClassifyReplacement(t *testing.T) {
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
			name:       "(i) sentinel direct: ErrInvalidInput → 400 InvalidRequest",
			err:        apperrors.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(ii) sentinel wrapped: ErrServiceUnavailable → 503",
			err:        fmt.Errorf("AD config validator: %w", apperrors.ErrServiceUnavailable),
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   response.CodeInternalError,
		},
		{
			name:       "(iii) BusinessError(CodeInvalidInput) → 400 InvalidRequest",
			err:        apperrors.NewBusinessError(apperrors.CodeInvalidInput, "Auth config mode invalid", nil),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(iv) unknown ad-hoc error → 500 InternalError",
			err:        errors.New("huawei_configs table missing migration"),
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
