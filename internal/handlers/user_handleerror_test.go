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

// TestUserHandler_HandleError_ClassifyReplacement verifies that the converged
// user_handler error paths route through response.HandleError correctly across
// the 4 error classes (Phase 20 D-02.5):
//
//	(i)   sentinel direct
//	(ii)  sentinel wrapped via %w
//	(iii) *apperrors.BusinessError
//	(iv)  unknown ad-hoc error
//
// The handler already used HandleError for service-error paths (Phase 19 D5),
// so the work for this plan is mostly SentinelField logging upgrade on the
// audit-record sites. The HandleError mapping is exercised here as a
// regression check that future refactors don't break the contract.
//
// Status-code expectations mirror internal/errors/mapping.go (Phase 19 D4):
//
//	ErrUserNotFound                → 404 / 1004 (CodeNotFound)
//	%w apperrors.ErrInvalidInput   → 400 / 1001 (CodeInvalidRequest)
//	BusinessError(CodeAlreadyExists) → 409 / 1006 (CodeDuplicateRecord)
//	unknown                        → 500 / 1005 (CodeInternalError)
func TestUserHandler_HandleError_ClassifyReplacement(t *testing.T) {
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
			name:       "(i) sentinel direct: ErrUserNotFound → 404 NotFound",
			err:        apperrors.ErrUserNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name:       "(ii) sentinel wrapped: ErrInvalidInput → 400 InvalidRequest",
			err:        fmt.Errorf("update user: invalid email format: %w", apperrors.ErrInvalidInput),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "(iii) BusinessError(CodeAlreadyExists) → 409",
			err:        apperrors.NewBusinessError(apperrors.CodeAlreadyExists, "username already taken", nil),
			wantStatus: http.StatusConflict,
			wantCode:   response.CodeDuplicateRecord,
		},
		{
			name:       "(iv) unknown ad-hoc error → 500 InternalError",
			err:        errors.New("DB transaction deadlock"),
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
