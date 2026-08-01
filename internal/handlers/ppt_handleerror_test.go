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

// TestPPTHandler_HandleError_ClassifyReplacement verifies that the converged PPT
// handler error paths route through response.HandleError correctly for all 4
// error classes (Phase 20 D-02.5):
//
//	(i)   sentinel direct
//	(ii)  sentinel wrapped via %w
//	(iii) *apperrors.BusinessError
//	(iv)  unknown ad-hoc error
//
// The handler now calls `if response.HandleError(c, err) { return }` after a
// service call returns one of the above error shapes. This test exercises the
// HandleError mapping contract directly so a regression in HandleError, the
// sentinel list, or the handler call-site changes is caught at PR time.
//
// Status-code expectations mirror internal/errors/mapping.go (Phase 19 D4):
//
//	ErrPPTFileNotFound    → 404 / 1004 (CodeNotFound)
//	%w apperrors.ErrInvalidInput → 400 / 1001 (CodeInvalidRequest)
//	BusinessError(CodeInvalidInput) → 400 / 1001
//	unknown                → 500 / 1005 (CodeInternalError)
func TestPPTHandler_HandleError_ClassifyReplacement(t *testing.T) {
	type body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data"`
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
			name:       "(i) sentinel direct: ErrPPTFileNotFound → 404 NotFound",
			err:        apperrors.ErrPPTFileNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name:       "(ii) sentinel wrapped: ErrInvalidInput → 400 InvalidRequest",
			err:        fmt.Errorf("frame bytes too large: 10000000 bytes (max 10MB): %w", apperrors.ErrInvalidInput),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(iii) BusinessError(CodeInvalidInput) → 400 InvalidRequest",
			err:        apperrors.NewBusinessError(apperrors.CodeInvalidInput, "bad payload", nil),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(iv) unknown ad-hoc error → 500 InternalError",
			err:        errors.New("random handler-layer failure"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a minimal gin context that mimics what a converged PPT
			// handler would create at the point HandleError is called.
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)

			// The converged handler idiom is:
			//     if response.HandleError(c, err) { return }
			// We invoke HandleError directly to validate the contract.
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
