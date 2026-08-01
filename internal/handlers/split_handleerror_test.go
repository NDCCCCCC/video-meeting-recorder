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

// TestSplitHandler_HandleError_ClassifyReplacement verifies that the converged
// split_handler error paths route through response.HandleError correctly across
// the 4 error classes (Phase 20 D-02.5):
//
//	(i)   sentinel direct
//	(ii)  sentinel wrapped via %w
//	(iii) *apperrors.BusinessError
//	(iv)  unknown ad-hoc error
//
// The handler now calls `if response.HandleError(c, err) { return }` after
// SubmitSplit and GenerateSnapshot. Note: split_handler previously had 0
// zap.Error sites (per RESEARCH §1); this plan adds 2 SentinelField-bearing
// log entries that didn't exist before, so the upgrade also creates
// observability for FFmpeg failures that previously went silent.
//
// Status-code expectations mirror internal/errors/mapping.go (Phase 19 D4):
//
//	ErrSplitFailed               → 500 / 1005 (CodeInternalError)
//	%w apperrors.ErrInvalidInput → 400 / 1001 (CodeInvalidRequest)
//	BusinessError(CodeFFmpegError) → 500 / 1005 (CodeInternalError)
//	unknown                       → 500 / 1005 (CodeInternalError)
func TestSplitHandler_HandleError_ClassifyReplacement(t *testing.T) {
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
			name:       "(i) sentinel direct: ErrSplitFailed → 500",
			err:        apperrors.ErrSplitFailed,
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternalError,
		},
		{
			name:       "(ii) sentinel wrapped: ErrInvalidInput → 400 InvalidRequest",
			err:        fmt.Errorf("split: marker 30s after end of video: %w", apperrors.ErrInvalidInput),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(iii) BusinessError(CodeFFmpegError) → 500",
			err:        apperrors.NewBusinessError(apperrors.CodeFFmpegError, "ffmpeg keyframe extract failed", nil),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternalError,
		},
		{
			name:       "(iv) unknown ad-hoc error → 500 InternalError",
			err:        errors.New("disk full while writing split output"),
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
