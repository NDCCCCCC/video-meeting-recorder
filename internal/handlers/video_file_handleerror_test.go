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

// TestVideoFileHandler_HandleError_ClassifyReplacement verifies that the
// converged video_file_handler error paths route through response.HandleError
// correctly across the 4 error classes (Phase 20 D-02.5):
//
//	(i)   sentinel direct
//	(ii)  sentinel wrapped via %w
//	(iii) *apperrors.BusinessError
//	(iv)  unknown ad-hoc error
//
// The handler now calls `if response.HandleError(c, err) { return }` after
// each service call (ListFiles / BatchDeleteFiles / GetFileStats / ScanFiles /
// RenameVideoFile / BatchDownloadFiles). DeleteFile and RenameVideoFile were
// already partially converted in Phase 19 Wave 6; this plan widens the
// coverage to the remaining endpoints.
//
// Status-code expectations mirror internal/errors/mapping.go (Phase 19 D4):
//
//	ErrVideoFileNotFound      → 404 / 1004 (CodeNotFound)
//	%w apperrors.ErrInvalidInput → 400 / 1001 (CodeInvalidRequest)
//	BusinessError(CodeInvalidInput) → 400 / 1001
//	unknown                   → 500 / 1005 (CodeInternalError)
func TestVideoFileHandler_HandleError_ClassifyReplacement(t *testing.T) {
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
			err:        fmt.Errorf("rename: invalid name (rule violation): %w", apperrors.ErrInvalidInput),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(iii) BusinessError(CodeServiceUnavailable) → 503",
			err:        apperrors.NewBusinessError(apperrors.CodeServiceUnavailable, "FFmpeg backend offline", nil),
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   response.CodeInternalError,
		},
		{
			name:       "(iv) unknown ad-hoc error → 500 InternalError",
			err:        errors.New("FFmpeg binary not found in PATH"),
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
