package auth

import (
	"context"
	"reflect"
	"testing"
)

// stubAuthenticator 验证 STYLE-003 修复：Authenticator 接口从 ad_config.go
// 迁移到 service.go（consumer package 定义接口）。stub 满足接口即验证 move 编译。
type stubAuthenticator struct{}

func (stubAuthenticator) Login(ctx context.Context, req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	return &LoginResponse{}, nil
}
func (stubAuthenticator) Logout(token string) error                       { return nil }
func (stubAuthenticator) ValidateToken(ctx context.Context, token string) (*UserDTO, error) {
	return &UserDTO{}, nil
}
func (stubAuthenticator) Name() string { return "stub" }

// TestAuthenticator_InterfaceCompilationCheck 验证接口契约。
func TestAuthenticator_InterfaceCompilationCheck(t *testing.T) {
	var _ Authenticator = stubAuthenticator{}

	// 通过 reflect 验证接口方法数
	ifaceType := reflect.TypeOf((*Authenticator)(nil)).Elem()
	if ifaceType.NumMethod() != 4 {
		t.Errorf("Authenticator 方法数 = %d，期望 4", ifaceType.NumMethod())
	}
}
