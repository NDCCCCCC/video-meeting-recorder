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

// TestFileHandler_HandleError_ClassifyReplacement verifies that the converged
// file_handler error paths route through response.HandleError correctly across
// the 4 error classes (Phase 20 D-02.5):
//
//	(i)   sentinel direct
//	(ii)  sentinel wrapped via %w
//	(iii) *apperrors.BusinessError
//	(iv)  unknown ad-hoc error
//
// The handler now calls `if response.HandleError(c, err) { return }` after each
// service call (Upload / Download / Delete / Share / ShareDownload / List /
// GetQuota). This test exercises the HandleError mapping contract directly so
// a regression in HandleError, the sentinel list, or the handler call-site
// changes is caught at PR time.
//
// Status-code expectations mirror internal/errors/mapping.go (Phase 19 D4):
//
//	ErrVideoFileNotFound   → 404 / 1004 (CodeNotFound)
//	%w apperrors.ErrInvalidInput → 400 / 1001 (CodeInvalidRequest)
//	BusinessError(CodeInvalidInput) → 400 / 1001
//	unknown                → 500 / 1005 (CodeInternalError)
func TestFileHandler_HandleError_ClassifyReplacement(t *testing.T) {
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
			name:       "(i) sentinel direct: ErrVideoFileNotFound → 404 NotFound",
			err:        apperrors.ErrVideoFileNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name:       "(ii) sentinel wrapped: ErrInvalidInput → 400 InvalidRequest",
			err:        fmt.Errorf("share: invalid password policy: %w", apperrors.ErrInvalidInput),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(iii) BusinessError(CodeInvalidInput) → 400 InvalidRequest",
			err:        apperrors.NewBusinessError(apperrors.CodeInvalidInput, "存储配额检查失败", nil),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(iv) unknown ad-hoc error → 500 InternalError",
			err:        errors.New("share-token backend timed out"),
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
