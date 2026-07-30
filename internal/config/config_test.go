package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestValidateProductionSecrets 覆盖 SEC-001 启动校验：生产环境缺失/过短/相同密钥 → Fatal；
// 非生产环境仅 Warn。通过覆盖包级 fatalFunc 为 panic 实现可测试性。
func TestValidateProductionSecrets(t *testing.T) {
	original := fatalFunc
	defer func() { fatalFunc = original }()
	fatalFunc = func(l *zap.Logger, msg string, fields ...zap.Field) { panic(msg) }

	cases := []struct {
		name      string
		env       string
		sm4       string
		hls       string
		wantFatal bool
	}{
		{"prod 缺失 sm4", "production", "", strings.Repeat("h", 32), true},
		{"prod sm4 过短(31)", "production", strings.Repeat("a", 31), strings.Repeat("h", 32), true},
		{"prod 缺失 hls", "production", strings.Repeat("a", 32), "", true},
		{"prod sm4==hls", "production", strings.Repeat("a", 32), strings.Repeat("a", 32), true},
		{"prod 合法且互异", "production", strings.Repeat("a", 32), strings.Repeat("b", 32), false},
		{"dev 缺失不 Fatal", "development", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			core, recorded := observer.New(zapcore.WarnLevel)
			logger := zap.New(core)
			cfg := &Config{}
			cfg.Server.Environment = tc.env
			cfg.Auth.SM4Secret = tc.sm4
			cfg.Auth.HLSTokenSecret = tc.hls

			if tc.wantFatal {
				assert.Panics(t, func() { cfg.ValidateProductionSecrets(logger) },
					"生产环境不合规密钥应触发 Fatal(panic)")
			} else {
				assert.NotPanics(t, func() { cfg.ValidateProductionSecrets(logger) },
					"合法或非生产环境不应 Fatal")
			}

			// 非生产环境 + 短密钥应至少打印一条 Warn
			if tc.env != "production" && (len(tc.sm4) < 32 || len(tc.hls) < 32) {
				assert.Greater(t, recorded.Len(), 0, "非生产环境短密钥应打印 Warn")
			}
		})
	}
}

// TestValidateProductionSecrets_NilLogger 确保 nil logger 不会 panic（防御性）。
func TestValidateProductionSecrets_NilLogger(t *testing.T) {
	cfg := &Config{}
	cfg.Server.Environment = "production"
	assert.NotPanics(t, func() { cfg.ValidateProductionSecrets(nil) })
}

// TestLoad_BindEnvSM4Secret 验证 SEC-001 BindEnv 修复：os.Getenv("SM4_SECRET") 能真正
// 加载到 cfg.Auth.SM4Secret（在隔离的临时目录内运行，避免污染仓库）。
func TestLoad_BindEnvSM4Secret(t *testing.T) {
	t.Chdir(t.TempDir())
	secret := strings.Repeat("a", 40)
	t.Setenv("SM4_SECRET", secret)

	cfg, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, secret, cfg.Auth.SM4Secret, "SM4_SECRET 环境变量应通过 BindEnv/展开注入配置")
}

// TestLoad_NoHardcodedSecretDefault 验证 SEC-001/D-03.4：未提供密钥时不再回退到
// "change-me-in-production" 硬编码默认值。
func TestLoad_NoHardcodedSecretDefault(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("SM4_SECRET", "")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.NotEqual(t, "change-me-in-production", cfg.Auth.SM4Secret,
		"不应再出现硬编码默认值")
	assert.NotEqual(t, "change-me-in-production", cfg.Auth.HLSTokenSecret,
		"HLS 密钥不应回退到硬编码默认值")
}

// TestLoad_BindEnvHuaweiTLS 验证 SEC-003a：HUAWEI_INSECURE_SKIP_VERIFY 与
// HUAWEI_MIN_TLS_VERSION 环境变量通过显式 BindEnv 加载到配置。
func TestLoad_BindEnvHuaweiTLS(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("HUAWEI_INSECURE_SKIP_VERIFY", "true")
	t.Setenv("HUAWEI_MIN_TLS_VERSION", "1.3")

	cfg, err := Load()
	assert.NoError(t, err)
	// viper 把字符串 "true" 反序列化为 bool
	assert.True(t, cfg.Huawei.InsecureSkipVerify, "HUAWEI_INSECURE_SKIP_VERIFY 应加载为 true")
	assert.Equal(t, "1.3", cfg.Huawei.MinTLSVersion, "HUAWEI_MIN_TLS_VERSION 应加载为 1.3")
}
