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

// TestCacheControlFor 验证静态资源缓存策略分流。
// 回归背景：serveFile 曾对所有文件统一设 max-age=3600，含 index.html。
// 由于 Vite 产物是内容哈希的，index.html 是唯一的缓存破除入口，缓存它会让
// 浏览器在 TTL 内继续按旧哈希名加载旧 bundle，表现为"发了新版本但页面没变"。
func TestCacheControlFor(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{"SPA 入口必须每次校验", "index.html", "no-cache, must-revalidate"},
		{"带前导斜杠的入口同样不缓存", "/index.html", "no-cache, must-revalidate"},
		{"大写扩展名也识别为 html", "INDEX.HTML", "no-cache, must-revalidate"},
		{"内容哈希 JS 永久缓存", "assets/index-Dku1sN1D.js", "public, max-age=31536000, immutable"},
		{"内容哈希 CSS 永久缓存", "/assets/index-uAopjGxH.css", "public, max-age=31536000, immutable"},
		{"无哈希零散资源用短 TTL", "vite.svg", "public, max-age=3600"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cacheControlFor(tt.file))
		})
	}
}
