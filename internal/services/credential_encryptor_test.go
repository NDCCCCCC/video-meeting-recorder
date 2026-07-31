package services

import (
	"context"
	"database/sql"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // pure-Go sqlite driver (no CGO)
)

// newTestEncryptor 构造一个仅 current 版本、密钥为 cur 的 encryptor。
func newTestEncryptor(t *testing.T, cur string) *CredentialEncryptor {
	t.Helper()
	enc, err := NewCredentialEncryptor("v1", cur, "", "", zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, enc)
	return enc
}

// newTestEncryptorWithPrevious 构造带 previous 版本的 encryptor。
func newTestEncryptorWithPrevious(t *testing.T, prevVer, prevSec, curVer, curSec string) *CredentialEncryptor {
	t.Helper()
	enc, err := NewCredentialEncryptor(curVer, curSec, prevVer, prevSec, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, enc)
	return enc
}

// openInMemoryDB 打开一个 SQLite 内存数据库 + 自动迁移 InputConfig + SystemSetting + VideoRecordingTask。
// 使用 modernc.org/sqlite 纯 Go driver，DSN 走 :memory: 加 foreign_keys。
// VideoRecordingTask 是 InputConfigService.GetConfigByID Preload 依赖的关联表。
func openInMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{
		Logger: gormLoggerSilent(),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.InputConfig{},
		&models.SystemSetting{},
		&models.VideoRecordingTask{},
	))
	return db
}

// gormLoggerSilent 返回一个静默 GORM logger，避免 ErrRecordNotFound 触发的 log 输出污染测试输出。
func gormLoggerSilent() logger.Interface {
	return logger.Default.LogMode(logger.Silent)
}

// ============================================================================
// 构造 + 基础加密 / 解密
// ============================================================================

func TestNewCredentialEncryptor(t *testing.T) {
	cur := "0123456789abcdef0123456789abcdef" // 32 chars
	prev := "fedcba9876543210fedcba9876543210"

	t.Run("仅 current", func(t *testing.T) {
		enc, err := NewCredentialEncryptor("v1", cur, "", "", zap.NewNop())
		require.NoError(t, err)
		assert.Equal(t, "v1", enc.CurrentVersion())
		assert.False(t, enc.HasPrevious())
		assert.Empty(t, enc.PreviousVersion())
	})

	t.Run("current + previous", func(t *testing.T) {
		enc, err := NewCredentialEncryptor("v2", cur, "v1", prev, zap.NewNop())
		require.NoError(t, err)
		assert.True(t, enc.HasPrevious())
		assert.Equal(t, "v1", enc.PreviousVersion())
	})

	t.Run("缺失 current version 应失败", func(t *testing.T) {
		_, err := NewCredentialEncryptor("", cur, "", "", zap.NewNop())
		assert.Error(t, err)
	})

	t.Run("current secret 过短应失败", func(t *testing.T) {
		_, err := NewCredentialEncryptor("v1", "short", "", "", zap.NewNop())
		assert.Error(t, err)
	})

	t.Run("Previous 配对缺失应失败", func(t *testing.T) {
		_, err := NewCredentialEncryptor("v1", cur, "v0", "", zap.NewNop())
		assert.Error(t, err)
		_, err = NewCredentialEncryptor("v1", cur, "", prev, zap.NewNop())
		assert.Error(t, err)
	})

	t.Run("Previous version == current 应失败", func(t *testing.T) {
		_, err := NewCredentialEncryptor("v1", cur, "v1", prev, zap.NewNop())
		assert.Error(t, err)
	})
}

func TestCredentialEncryptor_EncryptDecrypt(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")

	t.Run("明文 → envelope → 明文", func(t *testing.T) {
		env, err := enc.Encrypt("admin123")
		require.NoError(t, err)
		assert.True(t, utils.IsEncryptedPassword(env))
		assert.True(t, enc.LooksLikeEnvelope(env))

		pt, err := enc.Decrypt(env)
		require.NoError(t, err)
		assert.Equal(t, "admin123", pt)
	})

	t.Run("空明文 → 空 envelope", func(t *testing.T) {
		env, err := enc.Encrypt("")
		require.NoError(t, err)
		assert.Empty(t, env)

		pt, err := enc.Decrypt("")
		require.NoError(t, err)
		assert.Empty(t, pt)
	})

	t.Run("解密未知 version 失败", func(t *testing.T) {
		ct, err := utils.EncryptGCM(enc.currentKey, []byte("hello"))
		require.NoError(t, err)
		env, err := utils.EncodeCredentialEnvelope("v999", ct)
		require.NoError(t, err)

		_, err = enc.Decrypt(env)
		assert.Error(t, err, "未知 version envelope 应被拒绝")
		assert.Contains(t, err.Error(), "v999")
	})

	t.Run("LooksLikeEnvelope 区分", func(t *testing.T) {
		assert.True(t, enc.LooksLikeEnvelope("SM4:v1:abcd"))
		assert.False(t, enc.LooksLikeEnvelope("plain"))
		assert.False(t, enc.LooksLikeEnvelope("SM4-only"))
	})
}

// ============================================================================
// MigratePlaintextToGCM
// ============================================================================

func TestMigratePlaintextToGCM_PlaintextRows(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	rows := []*models.InputConfig{
		{Name: "A", ConfigType: "stream", StreamURL: "rtmp://x", Password: "plain-text-password", StreamPassword: "stream-plain", IsActive: true},
		{Name: "B", ConfigType: "stream", StreamURL: "rtmp://y", IsActive: true},
		{Name: "C", ConfigType: "stream", StreamURL: "rtmp://z", StreamPassword: "stream-plain-c", IsActive: true},
	}
	for _, r := range rows {
		require.NoError(t, db.Create(r).Error)
	}

	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))

	var got []models.InputConfig
	require.NoError(t, db.Unscoped().Find(&got).Error)
	require.Len(t, got, 3)

	for _, r := range got {
		switch r.Name {
		case "A":
			assert.True(t, enc.LooksLikeEnvelope(r.Password))
			assert.True(t, enc.LooksLikeEnvelope(r.StreamPassword))
			pt, _ := enc.Decrypt(r.Password)
			assert.Equal(t, "plain-text-password", pt)
			pt2, _ := enc.Decrypt(r.StreamPassword)
			assert.Equal(t, "stream-plain", pt2)
		case "B":
			assert.Empty(t, r.Password)
			assert.Empty(t, r.StreamPassword)
		case "C":
			assert.Empty(t, r.Password)
			assert.True(t, enc.LooksLikeEnvelope(r.StreamPassword))
			pt, _ := enc.Decrypt(r.StreamPassword)
			assert.Equal(t, "stream-plain-c", pt)
		}
	}
}

func TestMigratePlaintextToGCM_Base64Legacy(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	plaintext := "legacy-secret-password"
	b64stub := base64.StdEncoding.EncodeToString([]byte(plaintext))

	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Legacy", ConfigType: "stream", StreamURL: "rtmp://l",
		Password: b64stub, IsActive: true,
	}).Error)

	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))

	var got models.InputConfig
	require.NoError(t, db.Unscoped().First(&got, "name = ?", "Legacy").Error)
	assert.True(t, enc.LooksLikeEnvelope(got.Password))
	pt, err := enc.Decrypt(got.Password)
	require.NoError(t, err)
	assert.Equal(t, plaintext, pt)
}

func TestMigratePlaintextToGCM_Idempotent(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.InputConfig{
		Name: "X", ConfigType: "stream", StreamURL: "rtmp://x",
		Password: "plain-pw", IsActive: true,
	}).Error)

	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))
	var after1 models.InputConfig
	require.NoError(t, db.Unscoped().First(&after1, "name = ?", "X").Error)
	env1 := after1.Password

	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))
	var after2 models.InputConfig
	require.NoError(t, db.Unscoped().First(&after2, "name = ?", "X").Error)
	assert.Equal(t, env1, after2.Password, "已迁移行不应被二次加密")
}

func TestMigratePlaintextToGCM_SoftDeleted(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	row := &models.InputConfig{
		Name: "Deleted", ConfigType: "stream", StreamURL: "rtmp://d",
		Password: "soft-deleted-pw", IsActive: true,
	}
	require.NoError(t, db.Create(row).Error)
	require.NoError(t, db.Delete(row).Error)

	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))

	var got models.InputConfig
	require.NoError(t, db.Unscoped().First(&got, "name = ?", "Deleted").Error)
	assert.True(t, enc.LooksLikeEnvelope(got.Password), "软删除行也应被迁移")
}

func TestMigratePlaintextToGCM_AuthADPassword(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	plaintext := "ad-server-password"
	b64stub := base64.StdEncoding.EncodeToString([]byte(plaintext))
	require.NoError(t, db.Create(&models.SystemSetting{Key: "auth.ad.password", Value: b64stub}).Error)

	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))

	var got models.SystemSetting
	require.NoError(t, db.Unscoped().First(&got, "key = ?", "auth.ad.password").Error)
	assert.True(t, enc.LooksLikeEnvelope(got.Value))
	pt, err := enc.Decrypt(got.Value)
	require.NoError(t, err)
	assert.Equal(t, plaintext, pt)
}

func TestMigratePlaintextToGCM_AuthADStripPassword(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	jsonRaw := `{"server":"ad.example.com","bind_dn":"cn=admin","password":"plain-ad-pw","base_dn":"dc=ex","use_tls":true}`
	require.NoError(t, db.Create(&models.SystemSetting{Key: "auth.ad", Value: jsonRaw}).Error)

	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))

	var got models.SystemSetting
	require.NoError(t, db.Unscoped().First(&got, "key = ?", "auth.ad").Error)
	assert.False(t, strings.Contains(got.Value, `"password"`),
		"迁移后 auth.ad JSON 必须不含 password 字段")
	assert.Contains(t, got.Value, "ad.example.com", "其他字段应保留")
}

// ============================================================================
// InvariantScan
// ============================================================================

func TestInvariantScan_AllEnvelopePass(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	env1, _ := enc.Encrypt("pw-a")
	env2, _ := enc.Encrypt("pw-b")
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "A", ConfigType: "stream", StreamURL: "rtmp://a",
		Password: env1, StreamPassword: env2, IsActive: true,
	}).Error)

	require.NoError(t, enc.InvariantScan(ctx, db))
}

func TestInvariantScan_PlaintextRowFails(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Bad", ConfigType: "stream", StreamURL: "rtmp://x",
		Password: "still-plain", IsActive: true,
	}).Error)

	err := enc.InvariantScan(ctx, db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input_configs")
	assert.Contains(t, err.Error(), "password")
}

func TestInvariantScan_UnknownVersionFails(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.InputConfig{
		Name: "WrongVer", ConfigType: "stream", StreamURL: "rtmp://x",
		Password: "SM4:v999:YWJjZGVmZ2hpamtsbW5vcA==", IsActive: true,
	}).Error)

	err := enc.InvariantScan(ctx, db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "v999")
}

func TestInvariantScan_TamperedCiphertextFails(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	env, _ := enc.Encrypt("plain-pw")
	last := env[len(env)-1]
	var flipped byte
	if last == 'A' {
		flipped = 'B'
	} else {
		flipped = 'A'
	}
	tampered := env[:len(env)-1] + string(flipped)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Tampered", ConfigType: "stream", StreamURL: "rtmp://x",
		Password: tampered, IsActive: true,
	}).Error)

	err := enc.InvariantScan(ctx, db)
	assert.Error(t, err, "篡改的密文 tag 校验应失败")
}

func TestInvariantScan_ADJSONPasswordFails(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.SystemSetting{
		Key:   "auth.ad",
		Value: `{"server":"x","password":"leaked"}`,
	}).Error)

	err := enc.InvariantScan(ctx, db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

// ============================================================================
// RotateIfNeeded
// ============================================================================

func TestRotateIfNeeded_NoPrevious_NoOp(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)

	rotated, err := enc.RotateIfNeeded(context.Background(), db)
	assert.NoError(t, err)
	assert.Equal(t, 0, rotated)
}

func TestRotateIfNeeded_RewritesPreviousVersion(t *testing.T) {
	cur := "0123456789abcdef0123456789abcdef"
	prev := "fedcba9876543210fedcba9876543210"

	encPrev := newTestEncryptorWithPrevious(t, "", "", "v1", prev)
	env1, _ := encPrev.Encrypt("secret-1")
	env2, _ := encPrev.Encrypt("secret-2")
	env3, _ := encPrev.Encrypt("secret-3")

	db := openInMemoryDB(t)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "R1", ConfigType: "stream", StreamURL: "rtmp://a",
		Password: env1, StreamPassword: env2, IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "R2", ConfigType: "stream", StreamURL: "rtmp://b",
		StreamPassword: env3, IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.SystemSetting{Key: "auth.ad.password", Value: env1}).Error)

	encCur := newTestEncryptorWithPrevious(t, "v1", prev, "v2", cur)
	rotated, err := encCur.RotateIfNeeded(context.Background(), db)
	require.NoError(t, err)
	assert.Equal(t, 4, rotated, "3 个 input_configs envelope + 1 个 system_settings")

	var got []models.InputConfig
	require.NoError(t, db.Unscoped().Find(&got).Error)

	for _, r := range got {
		switch r.Name {
		case "R1":
			pt, err := encCur.Decrypt(r.Password)
			require.NoError(t, err)
			assert.Equal(t, "secret-1", pt)
			pt2, err := encCur.Decrypt(r.StreamPassword)
			require.NoError(t, err)
			assert.Equal(t, "secret-2", pt2)
			ver, _, _ := utils.ParseCredentialEnvelope(r.Password)
			assert.Equal(t, "v2", ver)
		case "R2":
			pt, err := encCur.Decrypt(r.StreamPassword)
			require.NoError(t, err)
			assert.Equal(t, "secret-3", pt)
			assert.Empty(t, r.Password)
		}
	}

	var adPwd models.SystemSetting
	require.NoError(t, db.Unscoped().First(&adPwd, "key = ?", "auth.ad.password").Error)
	pt, err := encCur.Decrypt(adPwd.Value)
	require.NoError(t, err)
	assert.Equal(t, "secret-1", pt)
}

func TestRotateIfNeeded_Idempotent(t *testing.T) {
	cur := "0123456789abcdef0123456789abcdef"
	prev := "fedcba9876543210fedcba9876543210"

	encPrev := newTestEncryptorWithPrevious(t, "", "", "v1", prev)
	env1, _ := encPrev.Encrypt("secret-1")
	db := openInMemoryDB(t)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "X", ConfigType: "stream", StreamURL: "rtmp://a",
		Password: env1, IsActive: true,
	}).Error)

	encCur := newTestEncryptorWithPrevious(t, "v1", prev, "v2", cur)
	rotated1, err := encCur.RotateIfNeeded(context.Background(), db)
	require.NoError(t, err)
	assert.Equal(t, 1, rotated1)

	rotated2, err := encCur.RotateIfNeeded(context.Background(), db)
	require.NoError(t, err)
	assert.Equal(t, 0, rotated2, "二轮轮换应为 no-op")
}
