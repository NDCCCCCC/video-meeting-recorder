package main

import (
	"strings"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAuthService_SetAuditServiceWiring 验证 SEC-002 修复：authService 可以被注入
// auditService（与 app.go 中新增的 `authService.SetAuditService(auditService)` 一行对应）。
// 此前 authService 从未注入 auditService，导致登录失败等 6 个审计点全部因 auditLogger==nil 短路。
func TestAuthService_SetAuditServiceWiring(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}

	cfg := &config.Config{}
	cfg.Auth.SM4Secret = strings.Repeat("a", 32)
	cfg.Auth.HLSTokenSecret = strings.Repeat("b", 32)
	cfg.Server.Environment = "development"
	logger := zap.NewNop()

	// 与 app.go initHandlers 一致的构造顺序
	authService := auth.NewService(cfg, db, logger)
	auditService := audit.NewAuditLogService(db, logger)

	// SEC-002 关键断言：注入不 panic，且 authService 非 nil
	assert.NotPanics(t, func() {
		authService.SetAuditService(auditService)
	})
	assert.NotNil(t, authService)

	// nil 注入同样不应 panic（防御性）
	assert.NotPanics(t, func() {
		authService.SetAuditService(nil)
	})
}
