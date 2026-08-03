package hlstoken

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// testSecret 返回长度 ≥ 32 的测试密钥。
func testSecret() string { return strings.Repeat("k", 32) }

// makeLegacyToken 手工构造一个"旧编码"token：数据用 URLEncoding，签名用指定编码，
// 模拟本修复前签发的 token（无 jti）。用于验证 Verify 的向后兼容承诺（D-03.3）。
func makeLegacyToken(secret string, sigEncoding *base64.Encoding, jti string) string {
	claims := HLSTokenClaims{
		TaskID:    1,
		UserID:    2,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
		Jti:       jti,
	}
	data, _ := json.Marshal(claims)
	encodedData := base64.URLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encodedData))
	sig := sigEncoding.EncodeToString(mac.Sum(nil))
	return encodedData + "." + sig
}

// TestNewHLSToken_ShortSecretPanics 验证 SEC-004：构造期密钥 < 32 字符 → panic（防御性兜底）。
func TestNewHLSToken_ShortSecretPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewHLSToken("too-short", time.Minute)
	})
}

// TestNewHLSToken_ValidSecretOK 合法长度（≥32）不应 panic。
func TestNewHLSToken_ValidSecretOK(t *testing.T) {
	h := NewHLSToken(testSecret(), time.Minute)
	assert.NotNil(t, h)
	assert.NotNil(t, h.usedJTIs, "usedJTIs 防重放集合应初始化")
}

// TestHLSVerify_BackwardCompat 验证 SEC-004/D-03.3：新代码同时接受新旧三种 base64 编码签名。
func TestHLSVerify_BackwardCompat(t *testing.T) {
	secret := testSecret()
	h := NewHLSToken(secret, time.Minute)

	cases := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "旧 URLEncoding 签名（无 jti）→ 验证通过",
			token:   makeLegacyToken(secret, base64.URLEncoding, ""),
			wantErr: false,
		},
		{
			name:    "旧 StdEncoding 签名 → 验证通过（D-03.3 兼容承诺）",
			token:   makeLegacyToken(secret, base64.StdEncoding, ""),
			wantErr: false,
		},
		{
			name:    "新 RawURLEncoding 签名（Generate）→ 验证通过",
			token:   h.Generate(1, 2),
			wantErr: false,
		},
		{
			name:    "篡改签名 → 拒绝",
			token:   h.Generate(1, 2) + "x",
			wantErr: true,
		},
	}

	// 注意：每个用例使用独立的 jti，避免相互触发防重放；旧 token 无 jti 不受影响。
	for i, tc := range cases {
		// 为新 token 用例生成独立 token，旧 token 用例已固定。
		if tc.name == "新 RawURLEncoding 签名（Generate）→ 验证通过" {
			tc.token = h.Generate(100+uint(i), 200+uint(i))
		}
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.Verify(tc.token)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVerify_MultiSegmentSameToken 验证 SEC-004 (Phase 19) 核心修复：同一 token
// 在其 TTL 窗口内可多次 Verify 全部通过。这模拟 HLS 播放的真实场景——m3u8 请求
// 与所有 .ts 分片共用 rewriteM3U8WithToken 注入的同一个 token。Phase 17 引入的
// 一次性 jti 拒绝导致首个 .ts 分片即 ErrTokenReplayed，多分片播放完全损坏。
func TestVerify_MultiSegmentSameToken(t *testing.T) {
	h := NewHLSToken(testSecret(), 5*time.Minute)
	tok := h.Generate(5, 6)

	var lastClaims *HLSTokenClaims
	for i := 0; i < 4; i++ {
		claims, err := h.Verify(tok)
		assert.NoError(t, err, "第 %d 次 Verify 应成功（多分片共用 token）", i+1)
		assert.NotNil(t, claims)
		lastClaims = claims
	}
	// jti 非空且被幂等记录（覆盖，非一次性）。
	assert.NotEmpty(t, lastClaims.Jti)
	h.mu.Lock()
	exp, ok := h.usedJTIs[lastClaims.Jti]
	h.mu.Unlock()
	assert.True(t, ok, "jti 应被记录到 usedJTIs")
	assert.Equal(t, lastClaims.ExpiresAt, exp, "记录的 ExpiresAt 应与 claims 一致")
}

// TestVerify_ExpiredStillRejected 验证 SEC-004 (Phase 19)：post-TTL 重放仍被
// Verify 内的 time.Now() > ExpiresAt 检查拦截（这才是真正的重放防线，不变）。
func TestVerify_ExpiredStillRejected(t *testing.T) {
	h := NewHLSToken(testSecret(), -time.Minute) // 负 duration → 立即过期
	tok := h.Generate(7, 8)
	_, err := h.Verify(tok)
	assert.Error(t, err, "过期 token 必须被拒绝")
	assert.NotContains(t, err.Error(), "已被使用", "拒绝原因应是过期而非一次性防重放")
}

// TestEvictExpired 验证 SEC-004 (Phase 19)：evictExpired 删除过期项、保留未过期项。
func TestEvictExpired(t *testing.T) {
	h := NewHLSToken(testSecret(), time.Minute)
	now := time.Now().Unix()

	h.mu.Lock()
	h.usedJTIs["past-1"] = now - 60
	h.usedJTIs["past-2"] = now - 1
	h.usedJTIs["future-1"] = now + 60
	h.usedJTIs["future-2"] = now + 3600
	h.mu.Unlock()

	h.evictExpired()

	h.mu.Lock()
	_, past1 := h.usedJTIs["past-1"]
	_, past2 := h.usedJTIs["past-2"]
	_, future1 := h.usedJTIs["future-1"]
	_, future2 := h.usedJTIs["future-2"]
	h.mu.Unlock()

	assert.False(t, past1, "远期过期项应被删除")
	assert.False(t, past2, "刚过期项应被删除")
	assert.True(t, future1, "未过期近期项应保留")
	assert.True(t, future2, "未过期远期项应保留")
}

// TestSweepLoop_StopsOnCtxCancel 验证 SEC-004 (Phase 19)：sweepLoop 在 ctx 取消时退出，
// 不泄漏 goroutine（遵循 BUG-006 约定）。
func TestSweepLoop_StopsOnCtxCancel(t *testing.T) {
	h := NewHLSToken(testSecret(), time.Minute)
	ctx, cancel := context.WithCancel(context.Background())

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.sweepLoop(ctx)
	}()

	// 取消并等待 goroutine 退出。若 sweepLoop 不响应 ctx.Done，wg.Wait 会挂起。
	cancel()
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// 通过：goroutine 在 ctx 取消后及时退出。
	case <-time.After(2 * time.Second):
		t.Fatal("sweepLoop 在 ctx 取消后未退出（goroutine 泄漏）")
	}
}

// TestEnforceCapHardLimit 验证 SEC-004 (Phase 19)：usedJTIs 超过 maxUsedJTIs 时
// 强制驱逐过期项 + 最早过期项，防止 sweeper 死亡时无限增长。
func TestEnforceCapHardLimit(t *testing.T) {
	// 用临时实例 + 手动注入 > maxUsedJTIs 项验证逻辑（不真正构造 100k+ 项，
	// 而是直接调用 enforceCapLocked 并断言收缩行为）。
	h := NewHLSToken(testSecret(), time.Minute)
	now := time.Now().Unix()

	h.mu.Lock()
	// 注入 maxUsedJTIs + 5 项：一半过期、一半未过期。
	for i := 0; i < maxUsedJTIs+5; i++ {
		if i%2 == 0 {
			h.usedJTIs[string(rune('a'+i%26))+strconv.Itoa(i)] = now - 60 // 过期
		} else {
			h.usedJTIs[string(rune('a'+i%26))+strconv.Itoa(i)] = now + 3600 // 未过期
		}
	}
	h.enforceCapLocked()
	afterLen := len(h.usedJTIs)
	h.mu.Unlock()

	assert.LessOrEqual(t, afterLen, maxUsedJTIs, "enforce 后 usedJTIs 不应超过 maxUsedJTIs")
}

// TestHLSVerify_Expired 验证过期 token 被拒绝（回归路径）。
func TestHLSVerify_Expired(t *testing.T) {
	h := NewHLSToken(testSecret(), -time.Minute) // 负 duration → 立即过期
	tok := h.Generate(7, 8)
	_, err := h.Verify(tok)
	assert.Error(t, err)
}

// ============================================================================
// Phase 19 D3: HLSJtiRecord DB 持久化测试
// ============================================================================

// TestHLSVerify_DB_PersistsJti 验证 db!=nil 模式下 Verify 把 jti 写入 hls_jti_records 表。
func TestHLSVerify_DB_PersistsJti(t *testing.T) {
	h := setupDBBackedHLS(t)
	tok := h.Generate(99, 100)

	claims, err := h.Verify(tok)
	require.NoError(t, err)
	assert.NotEmpty(t, claims.Jti, "新 Generate 的 token 必须有 jti")

	// 直接查 DB 验证记录存在
	var rec models.HLSJtiRecord
	err = h.db.Where("jti = ?", claims.Jti).First(&rec).Error
	require.NoError(t, err, "Verify 后应能在 hls_jti_records 查到 jti 记录")
	assert.Equal(t, claims.Jti, rec.Jti)
	assert.Equal(t, claims.ExpiresAt, rec.ExpiresAt)
}

// TestHLSVerify_DB_Idempotent 验证同 jti 重复 Verify 不报错（OnConflict DoNothing）。
func TestHLSVerify_DB_Idempotent(t *testing.T) {
	h := setupDBBackedHLS(t)
	tok := h.Generate(99, 100)

	for i := 0; i < 5; i++ {
		_, err := h.Verify(tok)
		require.NoError(t, err, "第 %d 次 Verify 不应报错", i+1)
	}

	// DB 中仍只有一行（不重复插入）
	var count int64
	h.db.Model(&models.HLSJtiRecord{}).Count(&count)
	assert.Equal(t, int64(1), count, "同 jti 多次 Verify 应只产生一行记录")
}

// TestHLSVerify_DB_SurvivesRecreate 验证 DB 模式跨实例化保留 jti 记录
// （与 in-memory 模式的核心区别——后者进程重启清空）。
func TestHLSVerify_DB_SurvivesRecreate(t *testing.T) {
	h1 := setupDBBackedHLS(t)
	tok := h1.Generate(99, 100)

	claims1, err := h1.Verify(tok)
	require.NoError(t, err)
	jti := claims1.Jti

	// 模拟"进程重启"——重新构造 HLSToken 共享同一 db
	h2 := NewHLSTokenWithDB(testSecret(), time.Minute, h1.db, nil)
	claims2, err := h2.Verify(tok)
	require.NoError(t, err)
	assert.Equal(t, jti, claims2.Jti, "新实例化后 jti 应仍可验证（HLS 多分片共用同一 token 场景）")

	// DB 中确实有记录
	var rec models.HLSJtiRecord
	h1.db.Where("jti = ?", jti).First(&rec)
	assert.Equal(t, jti, rec.Jti)
}

// setupDBBackedHLS 是 DB-backed HLS 测试夹具：in-memory sqlite + AutoMigrate + 5min TTL。
// 返回的 *HLSToken 的 db 字段对测试代码可见以便直接 query。
func setupDBBackedHLS(t *testing.T) *HLSToken {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.HLSJtiRecord{}))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})
	h := NewHLSTokenWithDB(testSecret(), 5*time.Minute, db, zap.NewNop())
	// h.db 字段未导出（package 内可见——test 同包 OK；外部无法直接访问）
	return h
}
