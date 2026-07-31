package services

import (
	"encoding/json"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestConfigService(t *testing.T, withEncryptor bool) (*ConfigService, *config.Config) {
	t.Helper()
	db := openInMemoryDB(t)
	var enc *CredentialEncryptor
	if withEncryptor {
		enc = newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	}
	cfg := &config.Config{}
	svc := NewConfigService(db, zap.NewNop(), cfg, enc)
	return svc, cfg
}

func TestConfigService_SaveAuthConfig_StripsPasswordFromADJSON(t *testing.T) {
	svc, _ := newTestConfigService(t, true)

	ad := &auth.ADAuthConfig{
		Server:   "ad.example.com",
		BindDN:   "cn=admin",
		Password: "ad-password-123",
		BaseDN:   "dc=ex",
		UseTLS:   true,
	}
	require.NoError(t, svc.SaveAuthConfig("ad", ad))

	// 读回 auth.ad：JSON 不含 password 字段
	var adSetting models.SystemSetting
	require.NoError(t, svc.db.Where("`key` = ?", "auth.ad").First(&adSetting).Error)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(adSetting.Value), &m))
	_, hasPassword := m["password"]
	assert.False(t, hasPassword, "system_settings[auth.ad] JSON 不应包含 password 字段")

	// 验证其他字段保留
	assert.Equal(t, "ad.example.com", m["server"])
	assert.Equal(t, "cn=admin", m["bind_dn"])
}

func TestConfigService_SaveAuthConfig_PasswordStoredAsEnvelope(t *testing.T) {
	svc, _ := newTestConfigService(t, true)

	ad := &auth.ADAuthConfig{
		Server:   "ad.example.com",
		Password: "secret-pw-123",
	}
	require.NoError(t, svc.SaveAuthConfig("ad", ad))

	var pwdSetting models.SystemSetting
	require.NoError(t, svc.db.Where("`key` = ?", "auth.ad.password").First(&pwdSetting).Error)
	assert.True(t, svc.encryptor.LooksLikeEnvelope(pwdSetting.Value),
		"auth.ad.password 必须是 SM4-GCM envelope")
	pt, err := svc.encryptor.Decrypt(pwdSetting.Value)
	require.NoError(t, err)
	assert.Equal(t, "secret-pw-123", pt)
}

func TestConfigService_SaveAuthConfig_NilEncryptor_Fails(t *testing.T) {
	svc, _ := newTestConfigService(t, false)

	ad := &auth.ADAuthConfig{Server: "x", Password: "pw"}
	err := svc.SaveAuthConfig("local", ad)
	assert.Error(t, err, "无 encryptor 时应拒绝 SaveAuthConfig")
	assert.Contains(t, err.Error(), "CredentialEncryptor")
}

func TestConfigService_LoadAuthConfig_DecryptsPassword(t *testing.T) {
	svc, cfg := newTestConfigService(t, true)

	// 先写入
	ad := &auth.ADAuthConfig{
		Server:   "ad.example.com",
		BindDN:   "cn=admin",
		Password: "stored-password",
		BaseDN:   "dc=test",
		UseTLS:   true,
	}
	require.NoError(t, svc.SaveAuthConfig("ad", ad))

	// 重置 cfg（模拟重启）
	cfg.Auth.AD = config.ADAuthConfig{}

	require.NoError(t, svc.LoadAuthConfig())
	assert.Equal(t, "ad", cfg.Auth.Mode)
	assert.Equal(t, "ad.example.com", cfg.Auth.AD.Server)
	assert.Equal(t, "cn=admin", cfg.Auth.AD.BindDN)
	assert.Equal(t, "dc=test", cfg.Auth.AD.BaseDN)
	assert.True(t, cfg.Auth.AD.UseTLS)
	assert.Equal(t, "stored-password", cfg.Auth.AD.Password, "LoadAuthConfig 应解密出明文")
}

func TestConfigService_LoadAuthConfig_NilEncryptor_Fails(t *testing.T) {
	svc, _ := newTestConfigService(t, false)
	err := svc.LoadAuthConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CredentialEncryptor")
}

func TestConfigService_SaveAuthConfig_EmptyPassword_NoEnvelopeWritten(t *testing.T) {
	svc, _ := newTestConfigService(t, true)

	ad := &auth.ADAuthConfig{Server: "x"} // 无密码
	require.NoError(t, svc.SaveAuthConfig("ad", ad))

	// auth.ad.password 行不应被写入（或保留空）
	var pwdSetting models.SystemSetting
	err := svc.db.Where("`key` = ?", "auth.ad.password").First(&pwdSetting).Error
	if err == nil {
		// 如果存在行，内容应为空
		assert.Empty(t, pwdSetting.Value)
	}
	// ErrRecordNotFound 也是合法结果
}
