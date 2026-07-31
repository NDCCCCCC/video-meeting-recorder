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

// ============================================================================
// Wave 4: 按 version 计数（CountByVersion / LogVersionCounts）
// ============================================================================

func TestCountByVersion_EmptyDB(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)
	ctx := context.Background()

	counts, err := enc.CountByVersion(ctx, db)
	require.NoError(t, err)
	require.Len(t, counts, 3, "3 个列（input_configs.password / stream_password / system_settings[auth.ad.password]）")

	for _, c := range counts {
		assert.Equal(t, 0, c.Total)
		assert.Equal(t, 0, c.EmptyRows)
		assert.Equal(t, 0, c.NonEnvelopeRows)
		assert.Equal(t, 0, c.UnknownVersion)
		assert.Empty(t, c.ByVersion)
	}
}

func TestCountByVersion_MixedVersions(t *testing.T) {
	cur := "0123456789abcdef0123456789abcdef"
	prev := "fedcba9876543210fedcba9876543210"

	// 用 prev 密钥写 v1 envelope
	encPrev := newTestEncryptorWithPrevious(t, "", "", "v1", prev)
	env1, _ := encPrev.Encrypt("secret-1")
	env2, _ := encPrev.Encrypt("secret-2")
	env3, _ := encPrev.Encrypt("secret-3")

	// 用 cur 密钥写 v2 envelope
	encCur := newTestEncryptorWithPrevious(t, "v1", prev, "v2", cur)
	envCur1, _ := encCur.Encrypt("new-1")

	db := openInMemoryDB(t)

	// input_configs.password: 2 个 v1 + 1 个 v2
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "R1", ConfigType: "stream", StreamURL: "rtmp://a",
		Password: env1, StreamPassword: env2, IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "R2", ConfigType: "stream", StreamURL: "rtmp://b",
		Password: envCur1, IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "R3", ConfigType: "stream", StreamURL: "rtmp://c",
		StreamPassword: env3, IsActive: true,
	}).Error)
	// 空行（合法"无凭据"）
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Empty", ConfigType: "stream", StreamURL: "rtmp://e", IsActive: true,
	}).Error)
	// 明文遗留（应被 NonEnvelopeRows 捕获）
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Stale", ConfigType: "stream", StreamURL: "rtmp://s",
		Password: "still-plain", IsActive: true,
	}).Error)
	// 未知 version envelope
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "UnknownVer", ConfigType: "stream", StreamURL: "rtmp://u",
		StreamPassword: "SM4:v999:YWJjZGVm", IsActive: true,
	}).Error)
	// system_settings[auth.ad.password] 一个 v1
	require.NoError(t, db.Create(&models.SystemSetting{
		Key: "auth.ad.password", Value: env1,
	}).Error)

	counts, err := encCur.CountByVersion(context.Background(), db)
	require.NoError(t, err)
	require.Len(t, counts, 3)

	// 找到 input_configs.password 这一列
	var pwCounts, spCounts, adCounts VersionCounts
	for _, c := range counts {
		switch c.Column {
		case "input_configs.password":
			pwCounts = c
		case "input_configs.stream_password":
			spCounts = c
		case "system_settings[auth.ad.password]":
			adCounts = c
		}
	}

	// input_configs.password: 1 v1 (R1.env1) + 1 v2 (R2.envCur1) = 2 envelopes,
	// 3 empty (R3/Empty/UnknownVer.Password=""), 1 non-envelope (Stale.Password="still-plain")
	assert.Equal(t, 2, pwCounts.Total)
	assert.Equal(t, 1, pwCounts.NonEnvelopeRows, "明文遗留应被 NonEnvelopeRows 捕获")
	assert.Equal(t, 3, pwCounts.EmptyRows)
	assert.Equal(t, 0, pwCounts.UnknownVersion)
	assert.Equal(t, 1, pwCounts.ByVersion["v1"])
	assert.Equal(t, 1, pwCounts.ByVersion["v2"])

	// input_configs.stream_password: 2 v1 (R1.env2 + R3.env3) + 1 v999 (UnknownVer) = 3 envelopes,
	// 3 empty (R2/Empty/Stale.StreamPassword=""), 0 non-envelope
	assert.Equal(t, 3, spCounts.Total)
	assert.Equal(t, 1, spCounts.UnknownVersion, "v999 应被 UnknownVersion 捕获")
	assert.Equal(t, 3, spCounts.EmptyRows)
	assert.Equal(t, 0, spCounts.NonEnvelopeRows)
	assert.Equal(t, 2, spCounts.ByVersion["v1"])
	assert.Equal(t, 0, spCounts.ByVersion["v2"])
	_, hasV999 := spCounts.ByVersion["v999"]
	assert.True(t, hasV999, "v999 应出现在 ByVersion map")

	// system_settings[auth.ad.password]: 1 v1
	assert.Equal(t, 1, adCounts.Total)
	assert.Equal(t, 1, adCounts.ByVersion["v1"])
}

func TestVersionCounts_FormatForLog_SortedKeys(t *testing.T) {
	// 不依赖 db，纯字段测试
	c := VersionCounts{
		Column:         "input_configs.password",
		Total:          5,
		EmptyRows:      1,
		ByVersion:      map[string]int{"v2": 2, "v1": 2, "v3": 1, "v999": 0},
		UnknownVersion: 0,
	}

	fields := c.FormatForLog()
	require.Greater(t, len(fields), 4, "至少含 column / total / empty_rows / non_envelope_rows / unknown_version_rows + by_version__*")

	// 检查 by_version__* 输出按字典序排列：v1, v2, v3, v999
	var prev string
	for _, f := range fields {
		key := f.Key
		if !strings.HasPrefix(key, "by_version__") {
			continue
		}
		v := strings.TrimPrefix(key, "by_version__")
		if prev != "" {
			assert.True(t, prev <= v, "by_version__ 字段必须按字典序排列：prev=%q v=%q", prev, v)
		}
		prev = v
	}
}

func TestLogVersionCounts_DoesNotError(t *testing.T) {
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)

	require.NoError(t, enc.LogVersionCounts(context.Background(), db, "after_migrate"))
}

func TestLogVersionCounts_NoPrevious(t *testing.T) {
	// 验证无 previous 密钥时（默认场景），unknown_version 仍正确报告
	enc := newTestEncryptor(t, "0123456789abcdef0123456789abcdef")
	db := openInMemoryDB(t)

	// 直接 INSERT 一个 v2 envelope（不是 current=v1）
	ct, _ := utils.EncryptGCM(enc.currentKey, []byte("pw"))
	envV2, _ := utils.EncodeCredentialEnvelope("v2", ct)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Mismatch", ConfigType: "stream", StreamURL: "rtmp://m",
		Password: envV2, IsActive: true,
	}).Error)

	counts, err := enc.CountByVersion(context.Background(), db)
	require.NoError(t, err)

	var pw VersionCounts
	for _, c := range counts {
		if c.Column == "input_configs.password" {
			pw = c
		}
	}
	assert.Equal(t, 1, pw.UnknownVersion, "无 previous 密钥时 v2 应算 unknown")
	assert.Equal(t, 1, pw.ByVersion["v2"])
}
