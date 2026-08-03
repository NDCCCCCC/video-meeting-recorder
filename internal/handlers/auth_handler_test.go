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

// TestLogin_HandleError_ClassifyDrop exercises the Login error path's
// re-routing through response.HandleError (Phase 20 R-3/R-4).
//
// Status-code expectations mirror internal/errors/mapping.go (Phase 19 D4 +
// Phase 20 20-01 additive changes):
//
//	ErrADUserNotRegistered     → 403 / 1003 (R-3: white-list 403 preserved)
//	ErrADAccountNotFound       → 404 / 1004
//	ErrUserDisabled            → 403 / 1003
//	ErrADConfigError           → 503 / 1005 (R-4: was 500, now 503)
//	ErrADUnreachable           → 503 / 1005 (R-4: was 500, now 503)
//	%w ErrUnauthorized         → 401 / 1002
//	BusinessError(InvalidInput)→ 400 / 1001
//	unknown                    → 500 / 1005 (HandleError returns false)
//
// The test verifies the full mapping path (errors.Is → mapping.go → HTTP
// status + response code), which is the contract governing Login's error
// response shape post convergence.
func TestLogin_HandleError_ClassifyDrop(t *testing.T) {
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
			name:       "ErrADUserNotRegistered (R-3) → 403 Forbidden",
			err:        apperrors.ErrADUserNotRegistered,
			wantStatus: http.StatusForbidden,
			wantCode:   response.CodeForbidden,
		},
		{
			name:       "wrapped ErrADUserNotRegistered → 403 Forbidden",
			err:        fmt.Errorf("用户映射失败: %w", apperrors.ErrADUserNotRegistered),
			wantStatus: http.StatusForbidden,
			wantCode:   response.CodeForbidden,
		},
		{
			name:       "ErrADAccountNotFound → 404 NotFound",
			err:        apperrors.ErrADAccountNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name:       "ErrUserDisabled → 403 Forbidden",
			err:        apperrors.ErrUserDisabled,
			wantStatus: http.StatusForbidden,
			wantCode:   response.CodeForbidden,
		},
		{
			name:       "ErrADConfigError (R-4) → 503 ServiceUnavailable",
			err:        apperrors.ErrADConfigError,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   response.CodeInternalError,
		},
		{
			name:       "ErrADUnreachable (R-4) → 503 ServiceUnavailable",
			err:        apperrors.ErrADUnreachable,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   response.CodeInternalError,
		},
		{
			name:       "ErrUnauthorized → 401 Unauthorized",
			err:        apperrors.ErrUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantCode:   response.CodeUnauthorized,
		},
		{
			name:       "wrapped ErrUnauthorized → 401 Unauthorized",
			err:        fmt.Errorf("登录失败: %w", apperrors.ErrUnauthorized),
			wantStatus: http.StatusUnauthorized,
			wantCode:   response.CodeUnauthorized,
		},
		{
			name:       "BusinessError(CodeInvalidInput) → 400 InvalidRequest",
			err:        apperrors.NewBusinessError(apperrors.CodeInvalidInput, "用户名或密码错误", nil),
			wantStatus: http.StatusBadRequest,
			wantCode:   response.CodeInvalidRequest,
		},
		{
			name:       "unknown ad-hoc error → 500 (HandleError returns false)",
			err:        errors.New("随机未识别错误"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)

			// Execute the converged Login error path: the handler now calls
			// `if response.HandleError(c, err) { return }` which delegates
			// to mapping.go for sentinel/BusinessError → HTTP status mapping.
			known := response.HandleError(ctx, tt.err)

			assert.Equal(t, tt.wantStatus, rec.Code,
				"HTTP status mismatch: HandleError(%v) → %d, want %d",
				tt.err, rec.Code, tt.wantStatus)

			got := parseBody(t, rec)
			assert.Equal(t, tt.wantCode, got.Code,
				"response code mismatch: HandleError(%v) → code %d, want %d",
				tt.err, got.Code, tt.wantCode)

			// HandleError must return true for known sentinels / typed
			// BusinessError so the caller can `return` safely (this is the
			// control-flow contract that lets Login exit after HandleError).
			var be *apperrors.BusinessError
			isKnownSentinelOrBE := errors.As(tt.err, &be) ||
				errors.Is(tt.err, apperrors.ErrADUserNotRegistered) ||
				errors.Is(tt.err, apperrors.ErrADAccountNotFound) ||
				errors.Is(tt.err, apperrors.ErrUserDisabled) ||
				errors.Is(tt.err, apperrors.ErrADConfigError) ||
				errors.Is(tt.err, apperrors.ErrADUnreachable) ||
				errors.Is(tt.err, apperrors.ErrUnauthorized)
			if isKnownSentinelOrBE {
				assert.True(t, known, "known sentinel should make HandleError return true")
			} else {
				assert.False(t, known, "unknown ad-hoc should make HandleError return false")
			}
		})
	}
}
