package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
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

// TestHuaweiDBAdapter_ProductionDecrypts 验证 SEC-003b 修复：
// 生产用 huaweiDBAdapter 必须经 CredentialEncryptor 解密后再返回明文密码。
// 直接返回密文会让 manager 把 "SM4:<version>:<base64>" 当明文提交给华为终端,
// 导致 401。这是 Phase 18 之后必须保证的不变式。
func TestHuaweiDBAdapter_ProductionDecrypts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	if err := db.AutoMigrate(&models.InputConfig{}); err != nil {
		t.Fatalf("migrate 失败: %v", err)
	}

	logger := zap.NewNop()

	encryptor, err := services.NewCredentialEncryptor(
		"v1",
		strings.Repeat("c", 32),
		"", "",
		logger,
	)
	if err != nil {
		t.Fatalf("构造 CredentialEncryptor 失败: %v", err)
	}

	const plaintextPwd = "HuaweiP@ss-2026"
	envelope, err := encryptor.Encrypt(plaintextPwd)
	if err != nil {
		t.Fatalf("Encrypt 失败: %v", err)
	}
	if !strings.HasPrefix(envelope, "SM4:v1:") {
		t.Fatalf("意外 envelope 格式: %q", envelope)
	}

	// 插入密文形态的 InputConfig
	cfg2 := models.InputConfig{
		Name:           "test-huawei",
		ConfigType:     "huawei",
		Server:         "10.0.0.1",
		Username:       "admin",
		Password:       envelope,
		TerminalNumber: "T01",
	}
	if err := db.Create(&cfg2).Error; err != nil {
		t.Fatalf("insert InputConfig 失败: %v", err)
	}

	adapter := &huaweiDBAdapter{db: db, encryptor: encryptor}
	got, err := adapter.GetHuaweiConfig(cfg2.ID)
	if err != nil {
		t.Fatalf("GetHuaweiConfig 失败: %v", err)
	}
	assert.Equal(t, plaintextPwd, got.Password,
		"生产适配器必须解密密文 envelope,不应直接返回 'SM4:v1:...'")
	assert.NotContains(t, got.Password, "SM4:",
		"返回值不应再含 envelope 前缀")
}
