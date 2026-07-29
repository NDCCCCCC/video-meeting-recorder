package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestClassifyAuthLoginError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantStatus int
	}{
		{
			name:       "sentinel maps to forbidden",
			err:        auth.ErrADUserNotRegistered,
			wantCode:   response.CodeForbidden,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "other mapping error remains invalid credential",
			err:        errors.New("用户映射失败"),
			wantCode:   response.CodeInvalidCredential,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "wrapped sentinel maps to forbidden",
			err:        fmt.Errorf("用户映射失败: %w", auth.ErrADUserNotRegistered),
			wantCode:   response.CodeForbidden,
			wantStatus: http.StatusForbidden,
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, status := classifyAuthLoginError(tt.err)
			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantStatus, status)

			// Exercise the exact response helper used by AuthHandler.Login so the
			// classification and code-to-HTTP-status wiring are verified together.
			testRecorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(testRecorder)
			request, err := http.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
			assert.NoError(t, err)
			ctx.Request = request
			response.GinError(ctx, code, tt.err.Error())
			assert.Equal(t, tt.wantStatus, testRecorder.Code)
		})
	}
}
