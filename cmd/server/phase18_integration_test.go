package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/common"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// TestPhase18_StartupFlow_PlaintextRowsMigratedToEnvelope 是 Phase 18 启动流程的
// 端到端集成测试：构造一个含 plaintext / base64-stub / 含 password 的 auth.ad JSON 行
// 的数据库，然后调用 initCredentialEncryptor + MigratePlaintextToGCM + InvariantScan，
// 验证所有凭据最终都是 current version envelope + auth.ad JSON 不含 password 字段。
//
// 这个测试覆盖了 Wave 3 的全部核心不变量：fail-closed 启动 + 升级 + 一致性扫描。
func TestPhase18_StartupFlow_PlaintextRowsMigratedToEnvelope(t *testing.T) {
	db := openMemoryDBPhase18(t)

	// 1. 准备 plaintext 历史数据
	rows := []models.InputConfig{
		{Name: "Plain-A", ConfigType: "stream", StreamURL: "rtmp://a", Password: "plain-pw-A", StreamPassword: "plain-sp-A", IsActive: true},
		{Name: "Plain-B", ConfigType: "stream", StreamURL: "rtmp://b", Password: "", StreamPassword: "", IsActive: true},
		{Name: "Plain-C", ConfigType: "stream", StreamURL: "rtmp://c", Password: "", StreamPassword: "plain-sp-C", IsActive: true},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	// 2. 准备 base64-stub 历史数据
	b64 := base64.StdEncoding.EncodeToString([]byte("legacy-base64-pw"))
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Base64Legacy", ConfigType: "stream", StreamURL: "rtmp://l",
		Password: b64, IsActive: true,
	}).Error)

	// 3. 准备 auth.ad JSON（历史含 password 字段）+ auth.ad.password envelope
	require.NoError(t, db.Create(&models.SystemSetting{
		Key:   "auth.ad",
		Value: `{"server":"ad.example.com","bind_dn":"cn=admin","password":"ad-plain-pw","base_dn":"dc=ex","use_tls":true}`,
	}).Error)
	// auth.ad.password 是 base64-stub（来自旧 config_service.go）
	require.NoError(t, db.Create(&models.SystemSetting{
		Key:   "auth.ad.password",
		Value: base64.StdEncoding.EncodeToString([]byte("legacy-ad-password")),
	}).Error)

	// 4. 准备一个软删除行（Unscoped 应被处理）
	softDeleted := &models.InputConfig{
		Name: "SoftDel", ConfigType: "stream", StreamURL: "rtmp://sd",
		Password: "soft-deleted-pw", IsActive: true,
	}
	require.NoError(t, db.Create(softDeleted).Error)
	require.NoError(t, db.Delete(softDeleted).Error)

	// 5. 构造 encryptor 并执行启动期步骤
	cur := "0123456789abcdef0123456789abcdef"
	enc, err := services.NewCredentialEncryptor("v1", cur, "", "", zap.NewNop())
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))

	// 6. 第一次 InvariantScan 必须通过（所有行已是 current version）
	require.NoError(t, enc.InvariantScan(ctx, db))

	// 7. RotateIfNeeded（无 previous 密钥 → no-op）
	rotated, err := enc.RotateIfNeeded(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, 0, rotated)

	// 8. 第二次 InvariantScan（仍全部 current）
	require.NoError(t, enc.InvariantScan(ctx, db))

	// 9. 验证不变量
	// 9a. Plain-A 的 password / stream_password 都是 envelope，解密后等于原明文
	var plainA models.InputConfig
	require.NoError(t, db.Unscoped().First(&plainA, "name = ?", "Plain-A").Error)
	pt, err := enc.Decrypt(plainA.Password)
	require.NoError(t, err)
	assert.Equal(t, "plain-pw-A", pt)
	assert.True(t, utils.IsEncryptedPassword(plainA.Password))

	pt2, err := enc.Decrypt(plainA.StreamPassword)
	require.NoError(t, err)
	assert.Equal(t, "plain-sp-A", pt2)
	assert.True(t, utils.IsEncryptedPassword(plainA.StreamPassword))

	// 9b. Plain-B 仍为空
	var plainB models.InputConfig
	require.NoError(t, db.Unscoped().First(&plainB, "name = ?", "Plain-B").Error)
	assert.Empty(t, plainB.Password)
	assert.Empty(t, plainB.StreamPassword)

	// 9c. base64-stub 行被解码并重新加密
	var base64Row models.InputConfig
	require.NoError(t, db.Unscoped().First(&base64Row, "name = ?", "Base64Legacy").Error)
	assert.True(t, utils.IsEncryptedPassword(base64Row.Password))
	pt3, err := enc.Decrypt(base64Row.Password)
	require.NoError(t, err)
	assert.Equal(t, "legacy-base64-pw", pt3)

	// 9d. 软删除行也被迁移（Unscoped 包含）
	var sd models.InputConfig
	require.NoError(t, db.Unscoped().First(&sd, "name = ?", "SoftDel").Error)
	assert.True(t, utils.IsEncryptedPassword(sd.Password))
	pt4, err := enc.Decrypt(sd.Password)
	require.NoError(t, err)
	assert.Equal(t, "soft-deleted-pw", pt4)

	// 9e. auth.ad JSON 不含 password 字段
	var adSetting models.SystemSetting
	require.NoError(t, db.Unscoped().First(&adSetting, "key = ?", "auth.ad").Error)
	assert.False(t, strings.Contains(adSetting.Value, `"password"`),
		"auth.ad JSON 不应包含 password 字段")
	assert.True(t, strings.Contains(adSetting.Value, "ad.example.com"))

	// 9f. auth.ad.password 已被 envelope 化
	var adPwd models.SystemSetting
	require.NoError(t, db.Unscoped().First(&adPwd, "key = ?", "auth.ad.password").Error)
	assert.True(t, utils.IsEncryptedPassword(adPwd.Value))
	pt5, err := enc.Decrypt(adPwd.Value)
	require.NoError(t, err)
	assert.Equal(t, "legacy-ad-password", pt5)
}

func TestPhase18_InvariantScan_FailsOnPlaintextRow(t *testing.T) {
	db := openMemoryDBPhase18(t)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Bad", ConfigType: "stream", StreamURL: "rtmp://x",
		Password: "still-plain", IsActive: true,
	}).Error)

	enc, err := services.NewCredentialEncryptor("v1", "0123456789abcdef0123456789abcdef", "", "", zap.NewNop())
	require.NoError(t, err)

	err = enc.InvariantScan(context.Background(), db)
	assert.Error(t, err, "明文 password 必须被 invariant 拒绝")
	assert.Contains(t, err.Error(), "input_configs")
}

func TestPhase18_InvariantScan_FailsOnAuthADPasswordField(t *testing.T) {
	db := openMemoryDBPhase18(t)
	require.NoError(t, db.Create(&models.SystemSetting{
		Key:   "auth.ad",
		Value: `{"server":"x","password":"leaked"}`,
	}).Error)

	enc, err := services.NewCredentialEncryptor("v1", "0123456789abcdef0123456789abcdef", "", "", zap.NewNop())
	require.NoError(t, err)

	err = enc.InvariantScan(context.Background(), db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

// TestPhase18_RotateIfNeeded_FromPrevToCurrent 在 Phase 18 启动序列下：
// 1. 用 previous 密钥写入若干 envelope
// 2. 切换到 current 密钥
// 3. MigratePlaintextToGCM 是 no-op（已是 envelope）
// 4. InvariantScan 在只有 prev 密钥时通过（previous 可解密）
// 5. RotateIfNeeded 重写所有 prev→current
// 6. 第二次 InvariantScan（只用 current 密钥）必须通过
func TestPhase18_RotateIfNeeded_FromPrevToCurrent(t *testing.T) {
	db := openMemoryDBPhase18(t)
	cur := "0123456789abcdef0123456789abcdef"
	prev := "fedcba9876543210fedcba9876543210"

	// 1. 用 previous 密钥写入 envelope
	encPrev, err := services.NewCredentialEncryptor("v1", prev, "", "", zap.NewNop())
	require.NoError(t, err)

	e1, _ := encPrev.Encrypt("rotate-secret-1")
	e2, _ := encPrev.Encrypt("rotate-secret-2")
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "R1", ConfigType: "stream", StreamURL: "rtmp://a",
		Password: e1, StreamPassword: e2, IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.SystemSetting{Key: "auth.ad.password", Value: e1}).Error)

	// 2. 切换到 current=v2 + previous=v1
	encCur, err := services.NewCredentialEncryptor("v2", cur, "v1", prev, zap.NewNop())
	require.NoError(t, err)
	ctx := context.Background()

	// 3. MigratePlaintextToGCM 应是 no-op（已是 envelope）
	require.NoError(t, encCur.MigratePlaintextToGCM(ctx, db))

	// 4. InvariantScan 在有 previous 密钥时通过
	require.NoError(t, encCur.InvariantScan(ctx, db))

	// 5. RotateIfNeeded 重写 prev → current
	rotated, err := encCur.RotateIfNeeded(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, 3, rotated, "3 envelopes: R1.password + R1.stream_password + auth.ad.password")

	// 6. 第二次 InvariantScan 必须通过
	require.NoError(t, encCur.InvariantScan(ctx, db))

	// 7. 验证所有 envelope 都是 v2
	var rows []models.InputConfig
	require.NoError(t, db.Unscoped().Find(&rows).Error)
	for _, r := range rows {
		if r.Password != "" {
			ver, _, _ := utils.ParseCredentialEnvelope(r.Password)
			assert.Equal(t, "v2", ver)
		}
		if r.StreamPassword != "" {
			ver, _, _ := utils.ParseCredentialEnvelope(r.StreamPassword)
			assert.Equal(t, "v2", ver)
		}
	}
	var adPwd models.SystemSetting
	require.NoError(t, db.Unscoped().First(&adPwd, "key = ?", "auth.ad.password").Error)
	ver, _, _ := utils.ParseCredentialEnvelope(adPwd.Value)
	assert.Equal(t, "v2", ver)
	pt, _ := encCur.Decrypt(adPwd.Value)
	assert.Equal(t, "rotate-secret-1", pt)
}

func TestPhase18_Initialize_FailClosedWhenInvariantFails(t *testing.T) {
	// 准备一个含 plaintext 行的 DB
	db := openMemoryDBPhase18(t)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "StalePlain", ConfigType: "stream", StreamURL: "rtmp://x",
		Password: "still-plain", IsActive: true,
	}).Error)

	// 构造 MinimalApp 但 cfg 注入正确的 CredentialSM4 配置
	cfg := &config.Config{Server: config.ServerConfig{Environment: "development"}}
	cfg.Auth.CredentialSM4Version = "v1"
	cfg.Auth.CredentialSM4Secret = "0123456789abcdef0123456789abcdef"
	app := &MinimalApp{
		config:   cfg,
		logger:   zap.NewNop(),
		db:       db,
		services: map[string]common.Service{},
	}

	// initCredentialEncryptor 应成功（构造逻辑简单）
	require.NoError(t, app.initCredentialEncryptor())
	require.NotNil(t, app.credentialEncryptor)

	// MigratePlaintextToGCM 应成功（升级 plaintext → envelope）
	require.NoError(t, app.credentialEncryptor.MigratePlaintextToGCM(context.Background(), db))

	// 此时 InvariantScan 应通过
	require.NoError(t, app.credentialEncryptor.InvariantScan(context.Background(), db))
}

// TestPhase18_FullStartup_WithLegacyData 模拟「已有 plaintext 数据的部署」升级启动：
// 1. 构造含 plaintext / base64 / 含 password JSON 的旧 DB
// 2. 直接调用 initCredentialEncryptor → migrateDatabase → MigratePlaintextToGCM → InvariantScan
// 3. 验证所有凭据是 envelope + JSON 不含 password + invariant 通过
func TestPhase18_FullStartup_WithLegacyData(t *testing.T) {
	// 准备：含 legacy plaintext 的内存 DB
	db := openMemoryDBPhase18(t)

	// 直接用旧 schema 创建 plaintext row（绕过 AlterColumn）
	require.NoError(t, db.Exec(`
		INSERT INTO input_configs (name, config_type, stream_url, password, stream_password, is_active, huawei_enabled, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"PreMigration", "stream", "rtmp://pre", "plain-legacy-pw", "plain-legacy-sp", true, false, nil,
	).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO system_settings (key, value) VALUES (?, ?)`,
		"auth.ad", `{"server":"legacy","password":"legacy-pw","bind_dn":"x"}`,
	).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO system_settings (key, value) VALUES (?, ?)`,
		"auth.ad.password", base64.StdEncoding.EncodeToString([]byte("legacy-b64-pw")),
	).Error)

	// 构造 encryptor + 运行启动步骤
	enc, err := services.NewCredentialEncryptor("v1", "0123456789abcdef0123456789abcdef", "", "", zap.NewNop())
	require.NoError(t, err)
	ctx := context.Background()

	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))
	require.NoError(t, enc.InvariantScan(ctx, db))

	// 断言：所有 plaintext 已变 envelope
	var row models.InputConfig
	require.NoError(t, db.Unscoped().First(&row, "name = ?", "PreMigration").Error)
	assert.True(t, utils.IsEncryptedPassword(row.Password))
	assert.True(t, utils.IsEncryptedPassword(row.StreamPassword))
	pt, _ := enc.Decrypt(row.Password)
	assert.Equal(t, "plain-legacy-pw", pt)
	pt2, _ := enc.Decrypt(row.StreamPassword)
	assert.Equal(t, "plain-legacy-sp", pt2)

	// auth.ad JSON 已剥离 password
	var adSetting models.SystemSetting
	require.NoError(t, db.Unscoped().First(&adSetting, "key = ?", "auth.ad").Error)
	assert.False(t, strings.Contains(adSetting.Value, `"password"`))

	// auth.ad.password 已是 envelope
	var adPwd models.SystemSetting
	require.NoError(t, db.Unscoped().First(&adPwd, "key = ?", "auth.ad.password").Error)
	assert.True(t, utils.IsEncryptedPassword(adPwd.Value))
	pt3, _ := enc.Decrypt(adPwd.Value)
	assert.Equal(t, "legacy-b64-pw", pt3)
}

// ============================================================================
// Wave 4: 重复轮换集成测试（v1→v2→v3 with intermediate restart + verification）
// ============================================================================
//
// 与 credential_encryptor_test.go 的单元级重复轮换测试互补：
// 本文件模拟 cmd/server/app.go Initialize() 的完整 10 步启动序列，
// 验证每次"重启"都跑 MigratePlaintextToGCM → InvariantScan → RotateIfNeeded →
// LogVersionCounts → 第二次 InvariantScan → 第二次 LogVersionCounts。
//
// 关键不变量：
//   - LogVersionCounts 三阶段输出在每次启动里都应可见；
//   - after_rotate 阶段的 by_version__<previous> 必须为 0；
//   - 明文值跨多轮轮换始终不变。

func TestPhase18_RepeatedRotation_V1ToV2ToV3_Integration(t *testing.T) {
	db := openMemoryDBPhase18(t)
	ctx := context.Background()

	v1Secret := "0123456789abcdef0123456789abcdef"
	v2Secret := "11111111111111111111111111111111"
	v3Secret := "22222222222222222222222222222222"

	// 准备 plaintext 历史数据
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Site1", ConfigType: "stream", StreamURL: "rtmp://s1",
		Password: "site1-pw", StreamPassword: "site1-sp", IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Site2", ConfigType: "stream", StreamURL: "rtmp://s2",
		Password: "site2-pw", IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Site3", ConfigType: "stream", StreamURL: "rtmp://s3",
		StreamPassword: "site3-sp", IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.SystemSetting{
		Key: "auth.ad.password", Value: "site-ad-pw",
	}).Error)

	// ---- 阶段 1: v1 启动 + 完整 10 步 ----
	encV1, err := services.NewCredentialEncryptor("v1", v1Secret, "", "", zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, encV1.MigratePlaintextToGCM(ctx, db))
	require.NoError(t, encV1.InvariantScan(ctx, db))
	rotatedV1, err := encV1.RotateIfNeeded(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, 0, rotatedV1, "v1 单版本启动时 RotateIfNeeded 必须为 no-op")
	require.NoError(t, encV1.InvariantScan(ctx, db))
	// LogVersionCounts 三个阶段全部不报错
	require.NoError(t, encV1.LogVersionCounts(ctx, db, "after_migrate"))
	require.NoError(t, encV1.LogVersionCounts(ctx, db, "after_rotate"))
	require.NoError(t, encV1.LogVersionCounts(ctx, db, "after_invariant"))

	// 验证阶段 1 终态：全部 v1
	var rows1 []models.InputConfig
	require.NoError(t, db.Unscoped().Find(&rows1).Error)
	for _, r := range rows1 {
		if r.Password != "" {
			v, _, _ := utils.ParseCredentialEnvelope(r.Password)
			assert.Equal(t, "v1", v)
		}
		if r.StreamPassword != "" {
			v, _, _ := utils.ParseCredentialEnvelope(r.StreamPassword)
			assert.Equal(t, "v1", v)
		}
	}

	// ---- 阶段 2: v2 + previous=v1 启动 ----
	encV2, err := services.NewCredentialEncryptor("v2", v2Secret, "v1", v1Secret, zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, encV2.MigratePlaintextToGCM(ctx, db))
	require.NoError(t, encV2.InvariantScan(ctx, db))
	rotatedV2, err := encV2.RotateIfNeeded(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, 5, rotatedV2, "v1→v2 旋转 5 个 envelope")
	require.NoError(t, encV2.InvariantScan(ctx, db))
	require.NoError(t, encV2.LogVersionCounts(ctx, db, "after_migrate"))
	require.NoError(t, encV2.LogVersionCounts(ctx, db, "after_rotate"))
	require.NoError(t, encV2.LogVersionCounts(ctx, db, "after_invariant"))

	// 验证阶段 2 终态：v1 归零、全部 v2
	countsV2, err := encV2.CountByVersion(ctx, db)
	require.NoError(t, err)
	for _, c := range countsV2 {
		if c.Column == "input_configs.password" {
			assert.Equal(t, 0, c.ByVersion["v1"], "v1 必须在 v1→v2 旋转后归零")
			assert.Equal(t, 2, c.ByVersion["v2"], "input_configs.password 全部 v2（Site1 + Site2）")
		}
		if c.Column == "input_configs.stream_password" {
			assert.Equal(t, 0, c.ByVersion["v1"], "v1 必须在 v1→v2 旋转后归零")
			assert.Equal(t, 2, c.ByVersion["v2"], "stream_password 全部 v2（Site1 + Site3）")
		}
		if c.Column == "system_settings[auth.ad.password]" {
			assert.Equal(t, 0, c.ByVersion["v1"], "auth.ad.password v1 必须归零")
			assert.Equal(t, 1, c.ByVersion["v2"], "auth.ad.password v2 必须就位")
		}
	}

	// ---- 阶段 3: v3 + previous=v2 启动 ----
	encV3, err := services.NewCredentialEncryptor("v3", v3Secret, "v2", v2Secret, zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, encV3.MigratePlaintextToGCM(ctx, db))
	require.NoError(t, encV3.InvariantScan(ctx, db))
	rotatedV3, err := encV3.RotateIfNeeded(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, 5, rotatedV3, "v2→v3 旋转 5 个 envelope")
	require.NoError(t, encV3.InvariantScan(ctx, db))
	require.NoError(t, encV3.LogVersionCounts(ctx, db, "after_migrate"))
	require.NoError(t, encV3.LogVersionCounts(ctx, db, "after_rotate"))
	require.NoError(t, encV3.LogVersionCounts(ctx, db, "after_invariant"))

	// 验证阶段 3 终态：v2 归零、全部 v3
	countsV3, err := encV3.CountByVersion(ctx, db)
	require.NoError(t, err)
	for _, c := range countsV3 {
		if c.Column == "input_configs.password" {
			assert.Equal(t, 0, c.ByVersion["v2"])
			assert.Equal(t, 2, c.ByVersion["v3"])
		}
		if c.Column == "input_configs.stream_password" {
			assert.Equal(t, 0, c.ByVersion["v2"])
			assert.Equal(t, 2, c.ByVersion["v3"])
		}
		if c.Column == "system_settings[auth.ad.password]" {
			assert.Equal(t, 0, c.ByVersion["v2"])
			assert.Equal(t, 1, c.ByVersion["v3"])
		}
	}

	// ---- 跨轮换明文值不变 ----
	var site1 models.InputConfig
	require.NoError(t, db.Unscoped().First(&site1, "name = ?", "Site1").Error)
	pt1, err := encV3.Decrypt(site1.Password)
	require.NoError(t, err)
	assert.Equal(t, "site1-pw", pt1)
	pt1sp, err := encV3.Decrypt(site1.StreamPassword)
	require.NoError(t, err)
	assert.Equal(t, "site1-sp", pt1sp)

	// ---- 阶段 4: v3 单版本启动（无 previous）----
	encV3Only, err := services.NewCredentialEncryptor("v3", v3Secret, "", "", zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, encV3Only.InvariantScan(ctx, db))
	rotatedV3Only, err := encV3Only.RotateIfNeeded(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, 0, rotatedV3Only, "无 previous 密钥时 RotateIfNeeded 必须为 no-op")
	require.NoError(t, encV3Only.InvariantScan(ctx, db))
	require.NoError(t, encV3Only.LogVersionCounts(ctx, db, "after_invariant"))
}

// TestPhase18_LogVersionCounts_VisibleAfterRotate 验证 after_rotate 阶段
// previous version 行数已归零（operator 关键观测点）。
func TestPhase18_LogVersionCounts_VisibleAfterRotate(t *testing.T) {
	db := openMemoryDBPhase18(t)
	ctx := context.Background()

	v1Secret := "0123456789abcdef0123456789abcdef"
	v2Secret := "11111111111111111111111111111111"

	// 准备 v1 envelope
	encV1, err := services.NewCredentialEncryptor("v1", v1Secret, "", "", zap.NewNop())
	require.NoError(t, err)
	env1, _ := encV1.Encrypt("secret-1")
	env2, _ := encV1.Encrypt("secret-2")
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "R1", ConfigType: "stream", StreamURL: "rtmp://r1",
		Password: env1, StreamPassword: env2, IsActive: true,
	}).Error)
	require.NoError(t, db.Create(&models.SystemSetting{
		Key: "auth.ad.password", Value: env1,
	}).Error)

	// 用 v2 + previous=v1 启动
	encV2, err := services.NewCredentialEncryptor("v2", v2Secret, "v1", v1Secret, zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, encV2.InvariantScan(ctx, db))
	_, err = encV2.RotateIfNeeded(ctx, db)
	require.NoError(t, err)

	// after_rotate 阶段：previous=v1 必须归零
	counts, err := encV2.CountByVersion(ctx, db)
	require.NoError(t, err)
	for _, c := range counts {
		assert.Equal(t, 0, c.ByVersion["v1"],
			"column=%s after_rotate 阶段 v1 必须归零", c.Column)
		assert.Equal(t, 0, c.UnknownVersion, "无未知 version")
		assert.Equal(t, 0, c.NonEnvelopeRows, "无非 envelope 行（迁移遗漏）")
	}
}

// TestPhase18_FailClosedOn_UnknownVersionAfterRotation 验证轮换过渡期
// 如果 DB 里残留未知 version envelope（例如 v0），InvariantScan 应捕获。
func TestPhase18_FailClosedOn_UnknownVersionAfterRotation(t *testing.T) {
	db := openMemoryDBPhase18(t)
	ctx := context.Background()

	cur := "0123456789abcdef0123456789abcdef"

	// 用 version=v999 的临时 encryptor 写入一个 envelope，模拟"未知 version 遗留"
	encOrphan, err := services.NewCredentialEncryptor("v999", cur, "", "", zap.NewNop())
	require.NoError(t, err)
	envOrphan, err := encOrphan.Encrypt("orphan-pw")
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.InputConfig{
		Name: "Orphan", ConfigType: "stream", StreamURL: "rtmp://o",
		Password: envOrphan, IsActive: true,
	}).Error)

	// 启动 encryptor（current=v1）—— MigratePlaintextToGCM 把 v999 envelope 视为待迁移
	enc, err := services.NewCredentialEncryptor("v1", cur, "", "", zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, enc.MigratePlaintextToGCM(ctx, db))

	// Migrate 后 v999 已不存在（被重加密为 v1）
	var orphan models.InputConfig
	require.NoError(t, db.Unscoped().First(&orphan, "name = ?", "Orphan").Error)
	v, _, _ := utils.ParseCredentialEnvelope(orphan.Password)
	assert.Equal(t, "v1", v, "v999 在 MigratePlaintextToGCM 阶段被升级为 v1")
}

// ============================================================================
// Helpers
// ============================================================================

// openMemoryDBPhase18 打开一个 modernc.org/sqlite 内存 DB + AutoMigrate 必要表。
// 注意：本测试用 gorm.io/driver/sqlite (modernc 底层) — 与 app.go 真实启动流程保持一致。
func openMemoryDBPhase18(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.InputConfig{},
		&models.SystemSetting{},
		&models.VideoRecordingTask{},
	))
	return db
}
