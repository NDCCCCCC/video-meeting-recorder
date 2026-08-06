package recorder

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
)

// newTestWatcher 构造测试用 ActivityWatcher:
//   - 走 applySmartEndDefaults 填默认数字 (bool 字段默认 false, 简化 huaweiPoller 路径)
//   - huaweiCli 传 nil (H 路径不被本套单测覆盖, 由 24-04 集成测试驱动)
//   - filePath/logFile 传 ""/nil (fileTicker/silenceScanner 路径只在 Start 之后采样,
//     本批单测覆盖 Start/Stop/IsActive/OnReconnect/Snapshot/ExtendStepMin/HuaweiEnabled
//     等不动 filePath 的契约)
func newTestWatcher(t *testing.T, mutate func(*config.SmartEndConfig)) *ActivityWatcher {
	t.Helper()
	cfg := &config.Config{}
	applySmartEndDefaults(cfg)
	if mutate != nil {
		mutate(&cfg.SmartEnd)
	}
	w := NewActivityWatcher(cfg, nil, "", nil, zap.NewNop())
	require.NotNil(t, w)
	return w
}

// TestNewActivityWatcher 验证 NewActivityWatcher 路径:
//   - 返回非 nil pointer
//   - 初始 Snapshot 应返回 Ended=false / EndedReason=""
//   - EndedCh 关闭前不应能接收 (默认分支)
func TestNewActivityWatcher(t *testing.T) {
	w := newTestWatcher(t, nil)
	require.NotNil(t, w)

	snap := w.Snapshot()
	require.False(t, snap.Ended)
	require.Equal(t, "", snap.EndedReason)

	select {
	case <-w.EndedCh():
		t.Fatal("EndedCh 在未结束时不应触发")
	default:
	}
}

// TestExtendStepMin 验证 ExtendStepMin():
//   cfg.SmartEnd.ExtendStepMin=60 → 60*time.Minute
//   cfg.SmartEnd.ExtendStepMin=30 (默认) → 30*time.Minute
// 对应 24-VALIDATION.md §ActivityWatcher scenario matrix "ExtendStepMin" 行。
func TestExtendStepMin(t *testing.T) {
	t.Run("explicit 60", func(t *testing.T) {
		w := newTestWatcher(t, func(se *config.SmartEndConfig) {
			se.ExtendStepMin = 60
		})
		require.Equal(t, 60*time.Minute, w.ExtendStepMin())
	})

	t.Run("default 30", func(t *testing.T) {
		w := newTestWatcher(t, nil)
		require.Equal(t, 30*time.Minute, w.ExtendStepMin())
	})
}

// TestHuaweiEnabled 验证 HuaweiEnabled() bool 直读 cfg.SmartEnd.HuaweiEnabled,
// Phase 25 coordinator 按此判定是否注入 huaweiCli (RESEARCH.md Pitfall 4)。
func TestHuaweiEnabled(t *testing.T) {
	t.Run("false by default", func(t *testing.T) {
		w := newTestWatcher(t, nil)
		require.False(t, w.HuaweiEnabled())
	})

	t.Run("true after set", func(t *testing.T) {
		w := newTestWatcher(t, func(se *config.SmartEndConfig) {
			se.HuaweiEnabled = true
		})
		require.True(t, w.HuaweiEnabled())
	})
}

// TestIsActive 验证 IsActive() 在 close-once 之前为 true,close-once 之后为 false;
// 通过 Start 然后 Stop 触发 stopped 路径,验证 close-once + IsActive 翻转。
func TestIsActive(t *testing.T) {
	w := newTestWatcher(t, nil)
	require.True(t, w.IsActive())

	w.Start()
	// 给 goroutine 一点点时间启动后被 cancel。
	time.Sleep(50 * time.Millisecond)
	w.Stop()

	require.False(t, w.IsActive())
}

// TestOnReconnect 验证 OnReconnect() 仅清 silenceSince,不动 lastFileGrowthAt /
// huaweiEmptySince / huaweiDegraded / silenceDegraded (24-VALIDATION.md "重连保持"
// 场景 + WATCH-05)。
//
// 手法:直接通过 unexported 字段赋值 (同包测试可达),触发 OnReconnect,然后断言
// silenceSince 被清零,其他字段保持原值。
func TestOnReconnect(t *testing.T) {
	w := newTestWatcher(t, nil)

	// 注入初始状态:模拟"已静音 30s + 文件已停滞 100s + 华为空状态 20s + 均未降级"
	w.mu.Lock()
	w.silenceSince = time.Now().Add(-30 * time.Second)
	w.lastFileGrowthAt = time.Now().Add(-100 * time.Second)
	w.huaweiEmptySince = time.Now().Add(-20 * time.Second)
	w.huaweiDegraded = false
	w.silenceDegraded = false
	w.mu.Unlock()

	w.OnReconnect()

	w.mu.Lock()
	defer w.mu.Unlock()
	require.True(t, w.silenceSince.IsZero(), "OnReconnect 必须清 silenceSince")
	require.False(t, w.lastFileGrowthAt.IsZero(), "OnReconnect 不应动 lastFileGrowthAt")
	require.False(t, w.huaweiEmptySince.IsZero(), "OnReconnect 不应动 huaweiEmptySince")
	require.False(t, w.huaweiDegraded, "OnReconnect 不应动 huaweiDegraded")
	require.False(t, w.silenceDegraded, "OnReconnect 不应动 silenceDegraded")
}

// TestSnapshot 验证 Snapshot() 返回各字段的拷贝,后续修改原状态不影响 snapshot;
// 对应 24-VALIDATION.md 场景矩阵中 Snapshot 字段读取 (LastHuaWeiStateEmpty /
// LastSilenceStart / TotalSilenceDuration)。
func TestSnapshot(t *testing.T) {
	w := newTestWatcher(t, nil)

	// 初始 Snapshot 字段全为 zero。
	snap := w.Snapshot()
	require.True(t, snap.SilenceSince.IsZero(), "initial SilenceSince should be zero")
	require.True(t, snap.LastFileGrowthAt.IsZero(), "initial LastFileGrowthAt should be zero")
	require.True(t, snap.HuaWeiEmptySince.IsZero(), "initial HuaWeiEmptySince should be zero")
	require.False(t, snap.LastHuaWeiStateEmpty)
	require.False(t, snap.HuaWeiDegraded)
	require.False(t, snap.SilenceDegraded)
	require.False(t, snap.Ended)
	require.Equal(t, "", snap.EndedReason)
	require.True(t, snap.LastSilenceStart.IsZero())
	require.Equal(t, time.Duration(0), snap.TotalSilenceDuration)

	// 写状态后再 Snapshot,字段应同步反映。
	w.mu.Lock()
	w.silenceSince = time.Now()
	w.lastFileGrowthAt = time.Now()
	w.huaweiEmptySince = time.Now()
	w.huaweiDegraded = true
	w.silenceDegraded = true
	w.mu.Unlock()

	snap2 := w.Snapshot()
	require.False(t, snap2.SilenceSince.IsZero())
	require.False(t, snap2.LastFileGrowthAt.IsZero())
	require.False(t, snap2.HuaWeiEmptySince.IsZero())
	require.True(t, snap2.HuaWeiDegraded)
	require.True(t, snap2.SilenceDegraded)

	// 拷贝语义:snap 是上一份拷贝,不应受后续状态变化影响。
	require.True(t, snap.SilenceSince.IsZero(), "Snapshot 应该是值拷贝,旧实例不被新状态污染")
}

// TestFakeClock 验证 fakeClock.Now() / Advance() 正确驱动时间,方便 24-04 集成测
// 试使用 (24-RESEARCH.md testability 章节要求导出该类型)。
func TestFakeClock(t *testing.T) {
	var fc fakeClock
	fc.t = time.Unix(1_700_000_000, 0).UTC()
	require.Equal(t, int64(1_700_000_000), fc.Now().Unix())

	fc.Advance(5 * time.Second)
	require.Equal(t, int64(1_700_000_005), fc.Now().Unix())

	// 边界: Advance(0) 不变。
	fc.Advance(0)
	require.Equal(t, int64(1_700_000_005), fc.Now().Unix())
}
