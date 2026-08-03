package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/utils"
)

// newTestInputConfigService 构造带 CredentialEncryptor 的 InputConfigService。
// 每个测试用 isolated in-memory DB。
func newTestInputConfigService(t *testing.T, withEncryptor bool) (*InputConfigService, *gormDBCloser) {
	t.Helper()
	db := openInMemoryDB(t)

	var enc *CredentialEncryptor
	if withEncryptor {
		enc = newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	}
	cfg := &config.Config{Admin: config.AdminConfig{MigrationConcurrency: 1}}
	svc := NewInputConfigService(db, zap.NewNop(), cfg, nil, enc)
	return svc, &gormDBCloser{db: db}
}

// gormDBCloser 是测试 helper，用于在测试结束时显式关闭 DB（避免 modernc.org/sqlite 报 busy）。
type gormDBCloser struct {
	db interface{}
}

// assertPasswordEqual 比对 DB 内的 envelope 是否对应特定明文（通过 decryptor 验证）。
// 由于 GCM nonce 随机，直接字符串比对 envelope 会失败；这里解密比对。
func assertPasswordEqual(t *testing.T, enc *CredentialEncryptor, dbEnvelope, wantPlaintext string) {
	t.Helper()
	if wantPlaintext == "" {
		assert.Empty(t, dbEnvelope)
		return
	}
	require.NotEmpty(t, dbEnvelope)
	pt, err := enc.Decrypt(dbEnvelope)
	require.NoError(t, err)
	assert.Equal(t, wantPlaintext, pt)
}

func TestInputConfigService_CreateConfig_EncryptsPasswords(t *testing.T) {
	svc, closer := newTestInputConfigService(t, true)
	_ = closer // 测试内自动回收

	req := &CreateInputConfigRequest{
		Name:           "Test-Stream",
		ConfigType:     "stream",
		StreamURL:      "rtmp://x",
		StreamUsername: "u",
		StreamPassword: "stream-pw-123",
		Password:       "huawei-pw-456",
	}

	got, err := svc.CreateConfig(context.Background(), req)
	require.NoError(t, err)
	assert.NotZero(t, got.ID)

	// DB 中存的是 envelope（不等于明文）
	var dbRow models.InputConfig
	require.NoError(t, svc.db.First(&dbRow, got.ID).Error)
	assert.NotEqual(t, "huawei-pw-456", dbRow.Password, "DB 存的应为 envelope 而非明文")
	assert.True(t, utils.IsEncryptedPassword(dbRow.Password), "Password 必须是 envelope")
	assert.NotEqual(t, "stream-pw-123", dbRow.StreamPassword)
	assert.True(t, utils.IsEncryptedPassword(dbRow.StreamPassword))

	// GetConfigByID 应解密回明文
	loaded, err := svc.GetConfigByID(context.Background(), got.ID)
	require.NoError(t, err)
	assert.Equal(t, "huawei-pw-456", loaded.Password, "GetConfigByID 应解密 Password")
	assert.Equal(t, "stream-pw-123", loaded.StreamPassword)
}

func TestInputConfigService_CreateConfig_NilEncryptor_PassesThrough(t *testing.T) {
	svc, _ := newTestInputConfigService(t, false)

	req := &CreateInputConfigRequest{
		Name:           "NoEnc",
		ConfigType:     "stream",
		StreamURL:      "rtmp://x",
		Password:       "plain-pw",
		StreamPassword: "plain-spw",
	}
	got, err := svc.CreateConfig(context.Background(), req)
	require.NoError(t, err)

	// encryptor=nil 时，DB 存明文（向后兼容）
	var dbRow models.InputConfig
	require.NoError(t, svc.db.First(&dbRow, got.ID).Error)
	assert.Equal(t, "plain-pw", dbRow.Password)
	assert.Equal(t, "plain-spw", dbRow.StreamPassword)
}

func TestInputConfigService_CreateConfig_EmptyPasswords_StaysEmpty(t *testing.T) {
	svc, _ := newTestInputConfigService(t, true)

	req := &CreateInputConfigRequest{
		Name:       "EmptyPw",
		ConfigType: "stream",
		StreamURL:  "rtmp://x",
	}
	got, err := svc.CreateConfig(context.Background(), req)
	require.NoError(t, err)

	assert.Empty(t, got.Password)
	assert.Empty(t, got.StreamPassword)

	var dbRow models.InputConfig
	require.NoError(t, svc.db.First(&dbRow, got.ID).Error)
	assert.Empty(t, dbRow.Password)
	assert.Empty(t, dbRow.StreamPassword)
}

func TestInputConfigService_UpdateConfig_EncryptsOnlyChanged(t *testing.T) {
	svc, _ := newTestInputConfigService(t, true)
	ctx := context.Background()

	// 先创建一条
	created, err := svc.CreateConfig(context.Background(), &CreateInputConfigRequest{
		Name:       "Update",
		ConfigType: "stream",
		StreamURL:  "rtmp://x",
		Password:   "original-pw",
	})
	require.NoError(t, err)

	// 不改 password，只改 name
	newName := "Update-v2"
	_, _, err = svc.UpdateConfig(context.Background(), created.ID, &UpdateInputConfigRequest{Name: &newName})
	require.NoError(t, err)

	var after models.InputConfig
	require.NoError(t, svc.db.Unscoped().First(&after, created.ID).Error)
	assertPasswordEqual(t, svc.encryptor, after.Password, "original-pw")

	// 改 password → 应重新加密
	newPw := "new-pw-v2"
	_, newConfig, err := svc.UpdateConfig(context.Background(), created.ID, &UpdateInputConfigRequest{Password: &newPw})
	require.NoError(t, err)
	assert.True(t, utils.IsEncryptedPassword(newConfig.Password))
	assert.NotEqual(t, newPw, newConfig.Password)

	loaded, err := svc.GetConfigByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, newPw, loaded.Password, "GetConfigByID 应解密出新 password")
	_ = ctx
}

func TestInputConfigService_GetConfigByID_DecryptionFailureFails(t *testing.T) {
	svc, _ := newTestInputConfigService(t, true)

	// 创建后用另一个 encryptor 重写 password（模拟数据被改）
	created, err := svc.CreateConfig(context.Background(), &CreateInputConfigRequest{
		Name: "DecFail", ConfigType: "stream", StreamURL: "rtmp://x",
		Password: "ok",
	})
	require.NoError(t, err)

	// 篡改 DB 中的 envelope（tag 翻转）
	var dbRow models.InputConfig
	require.NoError(t, svc.db.First(&dbRow, created.ID).Error)
	tampered := dbRow.Password[:len(dbRow.Password)-1] + "X"
	require.NoError(t, svc.db.Model(&dbRow).Update("password", tampered).Error)

	_, err = svc.GetConfigByID(context.Background(), created.ID)
	assert.Error(t, err, "篡改 envelope 后 GetConfigByID 应失败")
	assert.Contains(t, err.Error(), "解密")
}

func TestInputConfigService_UpdateConfig_StreamPassword_Encryption(t *testing.T) {
	svc, _ := newTestInputConfigService(t, true)

	created, err := svc.CreateConfig(context.Background(), &CreateInputConfigRequest{
		Name: "Sp", ConfigType: "stream", StreamURL: "rtmp://x",
	})
	require.NoError(t, err)

	newSp := "new-stream-pw"
	_, nc, err := svc.UpdateConfig(context.Background(), created.ID, &UpdateInputConfigRequest{StreamPassword: &newSp})
	require.NoError(t, err)
	assert.True(t, utils.IsEncryptedPassword(nc.StreamPassword))

	loaded, err := svc.GetConfigByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, newSp, loaded.StreamPassword)
}
