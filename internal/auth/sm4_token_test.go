package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // pure-Go sqlite driver（CGO_ENABLED=0 下注册 "sqlite" driver name）

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// ============================================================================
// 宽限/重放回归测试（quick 260828-j2a，Task 3）
//
// 覆盖两个关键路径：
//  1. GracePeriod 宽限期内同一 Refresh Token 重复刷新命中 tokenCache 快速路径，
//     返回相同 token 对，且 session 仍 IsActive（防止前端迟到 401 的良性并发
//     刷新被误判为重放攻击）
//  2. 超过 GracePeriod 后再用旧 RT 刷新：返回 apperrors.ErrTokenReplayed 并
//     调用 RevokeUserSessions 撤销该用户全部会话（重放防御语义锁定）
//
// 实施要点：GracePeriod 的所有使用点已收敛为 SM4TokenService.gracePeriod 实例
// 字段，保留导出常量作为默认值；测试注入短窗口（如 50ms）避免 sleep 30s。
// ============================================================================

// 32 字符 hex SM4 secret，满足 deriveSM4Key 的 SEC-001 启动校验
const testSM4Secret = "0123456789abcdef0123456789abcdef"

// openSM4TokenTestDB 打开 modernc.org/sqlite 内存 DB + AutoMigrate 必要表，
// 与项目内纯 Go SQLite 测试基建保持一致。
func openSM4TokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.Session{},
	))
	return db
}

// newTestSM4TokenService 构造一个 gracePeriod 可注入的 SM4TokenService。
func newTestSM4TokenService(t *testing.T, db *gorm.DB, grace time.Duration) *SM4TokenService {
	t.Helper()
	cfg := newTestAuthConfig()
	svc := NewSM4TokenService(cfg, db, zap.NewNop())
	svc.gracePeriod = grace
	return svc
}

// newTestAuthConfig 构造最小可用 *config.Config，只填 SM4TokenService 用到的字段。
func newTestAuthConfig() *config.Config {
	return &config.Config{
		Auth: config.AuthConfig{
			SM4Secret:            testSM4Secret,
			AccessTokenDuration:  2 * time.Hour,
			RefreshTokenDuration: 7 * 24 * time.Hour,
			MaxSessionDuration:   30 * 24 * time.Hour,
		},
	}
}

// createActiveUser 创建最小可用用户（不绑定 role / permission，token claims 留空即可）。
func createActiveUser(t *testing.T, db *gorm.DB, username string) *models.User {
	t.Helper()
	u := &models.User{
		Username: username,
		Email:    username + "@test.local",
		IsActive: true,
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

// seedSession 创建一条 IsActive=true 的 session（与 local_auth.Login 行为一致，
// session.Token = refresh token）。
func seedSession(t *testing.T, db *gorm.DB, user *models.User, refreshToken string) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&models.Session{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: now.Add(24 * time.Hour),
		IsActive:  true,
	}).Error)
}

// ----------------------------------------------------------------------------
// 回归测试
// ----------------------------------------------------------------------------

// TestRefreshAccessToken_GracePeriodIdempotent：宽限期内同一 Refresh Token
// 重复刷新命中 tokenCache 快路径（sm4_token.go:296-304），两次返回相同 token 对，
// 且 session 仍 IsActive（防止前端迟到 401 的良性并发刷新被误判为重放攻击）。
func TestRefreshAccessToken_GracePeriodIdempotent(t *testing.T) {
	db := openSM4TokenTestDB(t)
	svc := newTestSM4TokenService(t, db, 50*time.Millisecond)
	user := createActiveUser(t, db, "alice")

	// 生成第一个 refresh token 并落地 session
	first, err := svc.GenerateTokenPair(user)
	require.NoError(t, err)
	seedSession(t, db, user, first.RefreshToken)

	// 第一次刷新：正常流程 → 新 token 对 + 缓存 + 启动后台撤销 goroutine
	refreshed1, err := svc.RefreshAccessTokenWithContext(context.Background(), first.RefreshToken)
	require.NoError(t, err)
	assert.NotEqual(t, first.AccessToken, refreshed1.AccessToken, "首次刷新应生成新 access token")
	assert.NotEqual(t, first.RefreshToken, refreshed1.RefreshToken, "首次刷新应轮换 refresh token")

	// 宽限期内第二次刷新同一（旧的）refresh token：命中 tokenCache 快路径
	// 锁定幂等：返回的 token 对必须与首次刷新结果完全一致
	refreshed2, err := svc.RefreshAccessTokenWithContext(context.Background(), first.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, refreshed1.AccessToken, refreshed2.AccessToken,
		"宽限期内应命中缓存，返回首次刷新的 access token")
	assert.Equal(t, refreshed1.RefreshToken, refreshed2.RefreshToken,
		"宽限期内应命中缓存，返回首次刷新的 refresh token")

	// 锁定 session 仍 IsActive（goroutine 未到期）：防止误触发 RevokeUserSessions
	var session models.Session
	require.NoError(t, db.Where("token = ?", first.RefreshToken).First(&session).Error)
	assert.True(t, session.IsActive, "宽限期内 session 必须保持 IsActive")
}

// TestRefreshAccessToken_ReplayAfterGrace：超过宽限期后再用旧 RT 刷新，
// 必须返回 apperrors.ErrTokenReplayed 并调用 RevokeUserSessions 撤销该用户全部
// 会话（重放防御语义锁定）。宽限期放大到 30s 后此路径必须仍生效。
func TestRefreshAccessToken_ReplayAfterGrace(t *testing.T) {
	db := openSM4TokenTestDB(t)
	svc := newTestSM4TokenService(t, db, 50*time.Millisecond)
	user := createActiveUser(t, db, "bob")

	first, err := svc.GenerateTokenPair(user)
	require.NoError(t, err)
	seedSession(t, db, user, first.RefreshToken)

	// 第一次刷新：更新 last_used_at + 启动后台撤销 goroutine（50ms 后将 IsActive 置 false）
	_, err = svc.RefreshAccessTokenWithContext(context.Background(), first.RefreshToken)
	require.NoError(t, err)

	// 等待宽限期 + 一点 buffer：确保后台 goroutine 已把 session 标为 inactive
	time.Sleep(100 * time.Millisecond)

	// 旧 RT 再次刷新：cache 已过期 → DB session exists, IsActive=false → 超过宽限期
	// → 触发重放攻击分支：返回 ErrTokenReplayed + RevokeUserSessions
	_, err = svc.RefreshAccessTokenWithContext(context.Background(), first.RefreshToken)
	require.Error(t, err)
	assert.True(t, apperrors.Is(err, apperrors.ErrTokenReplayed),
		"超窗后旧 RT 刷新必须返回 ErrTokenReplayed，实际 err=%v", err)

	// session 已 IsActive=false（goroutine + RevokeUserSessions 共同置 false）
	var session models.Session
	require.NoError(t, db.Where("token = ?", first.RefreshToken).First(&session).Error)
	assert.False(t, session.IsActive, "重放攻击检测后 session 必须 IsActive=false")
}

// TestRefreshAccessToken_GracePeriodDefault：导出常量 GracePeriod 默认等于 30s，
// 锁定放宽后的常量值（quick 260828-j2a 把 5s 放宽到 30s 对齐前端 REFRESH_GRACE_MS）。
func TestRefreshAccessToken_GracePeriodDefault(t *testing.T) {
	assert.Equal(t, 30*time.Second, GracePeriod,
		"GracePeriod 默认值必须为 30s（与前端 REFRESH_GRACE_MS 对齐）")
}
