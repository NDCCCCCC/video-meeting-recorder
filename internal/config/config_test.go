package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestBindEnvHuaweiCABundle 验证 SEC-003a：HUAWEI_CA_BUNDLE_FILE 通过 bindSecretEnv
// 的 BindEnv 写入 v.BindEnv("huawei.ca_bundle_file", ...)，使用隔离 viper 实例 + Unmarshal
// 精确断言，避免被 Load() 自动产生默认 config.yaml 内容污染。
func TestBindEnvHuaweiCABundle(t *testing.T) {
	want := "/tmp/test-huawei-ca.pem"
	t.Setenv("HUAWEI_CA_BUNDLE_FILE", want)

	v := viper.New()
	bindSecretEnv(v)
	// 同时注册 YAML decoder（Unmarshal 需要知道类型才能反序列化）
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(`huawei: {}`)))

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))
	assert.Equal(t, want, cfg.Huawei.CABundleFile,
		"HUAWEI_CA_BUNDLE_FILE 应通过 BindEnv 写入 huawei.ca_bundle_file")
}

// TestHuaweiCABundle_EmptyPreserved 验证 SEC-003a：显式空字符串必须保留为空，
// system-CA opt-out 不能被任何默认值覆盖。在隔离 viper 中写入 YAML 明文
// `huawei.ca_bundle_file: ""`，Unmarshal 后字段值必须为空。
func TestHuaweiCABundle_EmptyPreserved(t *testing.T) {
	v := viper.New()
	bindSecretEnv(v)
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(`huawei:
  ca_bundle_file: ""
`)))

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))
	assert.Equal(t, "", cfg.Huawei.CABundleFile,
		"显式设置 ca_bundle_file=\"\" 必须保留为空字符（system-CA opt-out）")
}

func TestCSRFEnabledEnvBinding(t *testing.T) {
	if (&Config{}).Security.CSRFEnabled {
		t.Fatal("CSRF must default to false")
	}
	t.Setenv("CSRF_ENABLED", "true")
	v := viper.New()
	bindSecretEnv(v)
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Security.CSRFEnabled {
		t.Fatal("CSRF_ENABLED=true was not bound")
	}
}

func TestCSRFSafeOriginsEnvBinding(t *testing.T) {
	// SEC-008 (Phase 21): 验证 CSRF_SAFE_ORIGINS 通过 splitCommaSeparated 被加载为列表。
	t.Setenv("CSRF_SAFE_ORIGINS", "https://app.example.com,https://admin.example.com")
	t.Setenv("CSRF_ENABLED", "true")
	v := viper.New()
	bindSecretEnv(v)
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Security.CSRFEnabled {
		t.Fatal("CSRF_ENABLED=true was not bound")
	}
	want := []string{"https://app.example.com", "https://admin.example.com"}
	if len(cfg.Security.CSRFSafeOrigins) != len(want) {
		t.Fatalf("CSRFSafeOrigins length = %d, want %d", len(cfg.Security.CSRFSafeOrigins), len(want))
	}
	for i, w := range want {
		if cfg.Security.CSRFSafeOrigins[i] != w {
			t.Fatalf("CSRFSafeOrigins[%d] = %q, want %q", i, cfg.Security.CSRFSafeOrigins[i], w)
		}
	}
}

// ============================================================================
// Phase 18: 凭据静态加密密钥族校验
// ============================================================================

func TestValidateCredentialSM4Config_AcceptValid(t *testing.T) {
	cur := strings.Repeat("a", 32)
	prev := strings.Repeat("b", 32)
	cases := []struct {
		name string
		cfg  Config
	}{
		{
			name: "current only (最常见启动态)",
			cfg: Config{
				Auth: AuthConfig{
					CredentialSM4Version: "v1",
					CredentialSM4Secret:  cur,
				},
			},
		},
		{
			name: "current + previous 配对",
			cfg: Config{
				Auth: AuthConfig{
					CredentialSM4Version:         "v2",
					CredentialSM4Secret:          cur,
					CredentialSM4PreviousVersion: "v1",
					CredentialSM4PreviousSecret:  prev,
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, tc.cfg.ValidateCredentialSM4Config(), "合法配置应通过")
		})
	}
}

func TestValidateCredentialSM4Config_Reject(t *testing.T) {
	cur := strings.Repeat("a", 32)
	prev := strings.Repeat("b", 32)
	cases := []struct {
		name        string
		cfg         Config
		wantContain string
	}{
		{
			name:        "缺失 version",
			cfg:         Config{Auth: AuthConfig{CredentialSM4Secret: cur}},
			wantContain: "CREDENTIAL_SM4_VERSION 必须显式设置",
		},
		{
			name:        "version 格式非法（v0）",
			cfg:         Config{Auth: AuthConfig{CredentialSM4Version: "v0", CredentialSM4Secret: cur}},
			wantContain: "格式非法",
		},
		{
			name:        "version 格式非法（version）",
			cfg:         Config{Auth: AuthConfig{CredentialSM4Version: "version", CredentialSM4Secret: cur}},
			wantContain: "格式非法",
		},
		{
			name:        "version 格式非法（数字无 v 前缀）",
			cfg:         Config{Auth: AuthConfig{CredentialSM4Version: "1", CredentialSM4Secret: cur}},
			wantContain: "格式非法",
		},
		{
			name:        "secret 缺失",
			cfg:         Config{Auth: AuthConfig{CredentialSM4Version: "v1"}},
			wantContain: "≥ 32 字符",
		},
		{
			name:        "secret 过短 (31)",
			cfg:         Config{Auth: AuthConfig{CredentialSM4Version: "v1", CredentialSM4Secret: strings.Repeat("a", 31)}},
			wantContain: "≥ 32 字符",
		},
		{
			name: "Previous 配对缺失 (仅 version)",
			cfg: Config{Auth: AuthConfig{
				CredentialSM4Version:         "v1",
				CredentialSM4Secret:          cur,
				CredentialSM4PreviousVersion: "v0",
			}},
			wantContain: "同时设置或同时缺失",
		},
		{
			name: "Previous 配对缺失 (仅 secret)",
			cfg: Config{Auth: AuthConfig{
				CredentialSM4Version:        "v1",
				CredentialSM4Secret:         cur,
				CredentialSM4PreviousSecret: prev,
			}},
			wantContain: "同时设置或同时缺失",
		},
		{
			name: "Previous == Current version",
			cfg: Config{Auth: AuthConfig{
				CredentialSM4Version:         "v1",
				CredentialSM4Secret:          cur,
				CredentialSM4PreviousVersion: "v1",
				CredentialSM4PreviousSecret:  prev,
			}},
			wantContain: "必须不等于",
		},
		{
			name: "Previous == Current secret",
			cfg: Config{Auth: AuthConfig{
				CredentialSM4Version:         "v2",
				CredentialSM4Secret:          cur,
				CredentialSM4PreviousVersion: "v1",
				CredentialSM4PreviousSecret:  cur,
			}},
			wantContain: "必须不等于",
		},
		{
			name: "Previous secret 过短",
			cfg: Config{Auth: AuthConfig{
				CredentialSM4Version:         "v2",
				CredentialSM4Secret:          cur,
				CredentialSM4PreviousVersion: "v1",
				CredentialSM4PreviousSecret:  strings.Repeat("b", 31),
			}},
			wantContain: "≥ 32 字符",
		},
		{
			name: "Previous version 格式非法",
			cfg: Config{Auth: AuthConfig{
				CredentialSM4Version:         "v2",
				CredentialSM4Secret:          cur,
				CredentialSM4PreviousVersion: "v0",
				CredentialSM4PreviousSecret:  prev,
			}},
			wantContain: "格式非法",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.ValidateCredentialSM4Config()
			assert.Error(t, err, "非法配置应被拒绝")
			if err != nil {
				assert.Contains(t, err.Error(), tc.wantContain, "错误信息应指明失败原因")
			}
		})
	}
}

func TestBindEnvCredentialSM4(t *testing.T) {
	cur := strings.Repeat("a", 40)
	prev := strings.Repeat("b", 40)
	t.Setenv("CREDENTIAL_SM4_VERSION", "v2")
	t.Setenv("CREDENTIAL_SM4_SECRET", cur)
	t.Setenv("CREDENTIAL_SM4_PREVIOUS_VERSION", "v1")
	t.Setenv("CREDENTIAL_SM4_PREVIOUS_SECRET", prev)

	v := viper.New()
	bindSecretEnv(v)
	var cfg Config
	assert.NoError(t, v.Unmarshal(&cfg))

	assert.Equal(t, "v2", cfg.Auth.CredentialSM4Version, "CREDENTIAL_SM4_VERSION 应通过 BindEnv 加载")
	assert.Equal(t, cur, cfg.Auth.CredentialSM4Secret, "CREDENTIAL_SM4_SECRET 应通过 BindEnv 加载")
	assert.Equal(t, "v1", cfg.Auth.CredentialSM4PreviousVersion, "CREDENTIAL_SM4_PREVIOUS_VERSION 应通过 BindEnv 加载")
	assert.Equal(t, prev, cfg.Auth.CredentialSM4PreviousSecret, "CREDENTIAL_SM4_PREVIOUS_SECRET 应通过 BindEnv 加载")
}
