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

// TestAPIKeyHandler_HandleError_ClassifyReplacement verifies that the converged
// apikey_handler error paths route through response.HandleError correctly across
// the 4 error classes (Phase 20 D-02.5):
//
//	(i)   sentinel direct
//	(ii)  sentinel wrapped via %w
//	(iii) *apperrors.BusinessError
//	(iv)  unknown ad-hoc error
//
// The handler already used HandleError for service-error paths (Phase 19 D10),
// so the work for this plan is purely SentinelField logging upgrade on the
// 7 service-error log lines + 3 audit-record sites. The HandleError mapping
// is exercised here as a regression check that future refactors don't break
// the contract.
//
// Status-code expectations mirror internal/errors/mapping.go (Phase 19 D4):
//
//	ErrAPIKeyNotFound               → 404 / 1004 (CodeNotFound)
//	%w apperrors.ErrAPIKeyExpired   → 401 / 1002 (CodeUnauthorized)
//	BusinessError(CodeForbidden)    → 403 / 1003 (CodeForbidden)
//	unknown                         → 500 / 1005 (CodeInternalError)
func TestAPIKeyHandler_HandleError_ClassifyReplacement(t *testing.T) {
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
			name:       "(i) sentinel direct: ErrAPIKeyNotFound → 404 NotFound",
			err:        apperrors.ErrAPIKeyNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name:       "(ii) sentinel wrapped: ErrAPIKeyExpired → 401 Unauthorized",
			err:        fmt.Errorf("validate: key expired 7 days ago: %w", apperrors.ErrAPIKeyExpired),
			wantStatus: http.StatusUnauthorized,
			wantCode:   response.CodeUnauthorized,
		},
		{
			name:       "(iii) BusinessError(CodeForbidden) → 403",
			err:        apperrors.NewBusinessError(apperrors.CodeForbidden, "IP not in whitelist", nil),
			wantStatus: http.StatusForbidden,
			wantCode:   response.CodeForbidden,
		},
		{
			name:       "(iv) unknown ad-hoc error → 500 InternalError",
			err:        errors.New("usage log table corrupted"),
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
