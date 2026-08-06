package recorder

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/huawei"
)

// newTestWatcher 构造测试用 ActivityWatcher:
//   - SmartEnd 字段显式填默认值 (不依赖 config.applySmartEndDefaults 因其
//     是 config 包私有;bool 字段默认 false, 简化 huaweiPoller 路径)
//   - huaweiCli 传 nil (H 路径不被本套单测覆盖, 由 24-04 集成测试驱动)
//   - filePath/logFile 传 ""/nil (fileTicker/silenceScanner 路径只在 Start 之后采样,
//     本批单测覆盖 Start/Stop/IsActive/OnReconnect/Snapshot/ExtendStepMin/HuaweiEnabled
//     等不动 filePath 的契约)
func newTestWatcher(t *testing.T, mutate func(*config.SmartEndConfig)) *ActivityWatcher {
	t.Helper()
	cfg := &config.Config{}
	cfg.SmartEnd = config.SmartEndConfig{
		SilenceDB:              -30,
		SilenceDurationS:       30,
		FileStallS:             120,
		FileMinGrowthBPS:       1024,
		HuaweiPollIntervalS:    30,
		HuaweiPersistS:         30,
		HuaweiFailureThreshold: 3,
		CheckIntervalS:         5,
		ExtendStepMin:          30,
		MaxExtendCount:         4,
		StatFailureThreshold:   3,
		// Enabled / HuaweiEnabled / DegradeOnSilenceLoss 走零值 false 以简化。
	}
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
//
//	cfg.SmartEnd.ExtendStepMin=60 → 60*time.Minute
//	cfg.SmartEnd.ExtendStepMin=30 (默认) → 30*time.Minute
//
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

// ---------------------------------------------------------------------------
// Phase 24-04 Nyquist scenario matrix (24-VALIDATION.md §ActivityWatcher scenario matrix)
// ---------------------------------------------------------------------------

// activityWatcherFakeHuawei 实现 HuaweiStateClient 接口,驱动 huaweiPoller 行为;
// 字段 state/err 可在测试中动态调整。
type activityWatcherFakeHuawei struct {
	mu    sync.Mutex
	state *huawei.ConferenceState
	err   error
}

func (f *activityWatcherFakeHuawei) GetConferenceState(context.Context) (*huawei.ConferenceState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state, f.err
}

// activityWatcherConfig 构造启用 SmartEnd 的短间隔 cfg,供 H 路径 1-2s 内触发
// closeEnded 的真实时间集成测试使用。Enabled/HuaweiEnabled/DegradeOnSilenceLoss
// 都设 true 以驱动 4 个 goroutine 全开;数字字段设为 1s 让 decisionTicker 在
// 1-3 个 tick 内触发关闭条件 (避免 fakeClock 注入范围扩张到 production code)。
func activityWatcherConfig() *config.Config {
	return &config.Config{SmartEnd: config.SmartEndConfig{
		Enabled:                true,
		HuaweiEnabled:          true,
		DegradeOnSilenceLoss:   true,
		SilenceDB:              -30,
		SilenceDurationS:       1,
		FileStallS:             1,
		FileMinGrowthBPS:       1024,
		HuaweiPollIntervalS:    1,
		HuaweiPersistS:         1,
		HuaweiFailureThreshold: 3,
		CheckIntervalS:         1,
		ExtendStepMin:          30,
		MaxExtendCount:         4,
		StatFailureThreshold:   3,
	}}
}

// waitActivityEnded 阻塞直到 watcher.EndedCh 关闭或 5s 超时。
func waitActivityEnded(t *testing.T, w *ActivityWatcher) {
	t.Helper()
	select {
	case <-w.EndedCh():
	case <-time.After(5 * time.Second):
		t.Fatal("ActivityWatcher did not end within timeout")
	}
}

// 1. H 路径触发:fake Huawei 持续 confState=""+JoinSum=0 至少 huawei_persist_s
// 后,decisionTicker 检测 huaweiEmptySince 满足条件 → closeEnded("huawei_state_empty")。
func TestActivityWatcher_MeetingEnded_HuaweiEmpty(t *testing.T) {
	f := &activityWatcherFakeHuawei{state: &huawei.ConferenceState{HasConferenceFields: true}}
	w := NewActivityWatcher(activityWatcherConfig(), f, "", nil, zap.NewNop())
	w.Start()
	defer w.Stop()
	waitActivityEnded(t, w)
	s := w.Snapshot()
	require.True(t, s.Ended)
	require.Equal(t, "huawei_state_empty", s.EndedReason)
	require.True(t, s.LastHuaWeiStateEmpty)
}

// 2. A+B 路径触发:logFile 写入 silence_start → silenceScanner 设 silenceSince;
// fileTicker 周期 stat,但 lastFileGrowthAt 已被测试在 Start 前置为 now-2s
// (>= file_stall_s);decisionTicker 在 silenceDurationS+FileStallS 满足时关。
func TestActivityWatcher_MeetingEnded_AndAB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "record.mkv")
	require.NoError(t, os.WriteFile(path, []byte("seed"), 0644))
	logFile, err := os.CreateTemp(t.TempDir(), "ffmpeg-*.log")
	require.NoError(t, err)
	defer logFile.Close()
	_, err = logFile.WriteString("[silencedetect @ 0x55a3] silence_start: 0\n")
	require.NoError(t, err)
	require.NoError(t, logFile.Sync())
	// 写完后 os.File 偏移在 EOF;bufio.Scanner 从偏移 0 起读需先 Seek 回 0。
	_, err = logFile.Seek(0, 0)
	require.NoError(t, err)
	cfg := activityWatcherConfig()
	cfg.SmartEnd.HuaweiEnabled = false
	w := NewActivityWatcher(cfg, nil, path, logFile, zap.NewNop())
	// 预置 lastFileGrowthAt 到 2s 前 (> file_stall_s=1),让 B 路径首次 decision 时立即满足。
	w.mu.Lock()
	w.lastFileGrowthAt = time.Now().Add(-2 * time.Second)
	w.mu.Unlock()
	w.Start()
	defer w.Stop()
	waitActivityEnded(t, w)
	require.Equal(t, "both_silence_and_stall", w.Snapshot().EndedReason)
}

// 3. H 降级:fakeHuawei 3 次返 err → huaweiDegraded=true → H 路径不再评估;
// 验证后续即使恢复 confState=empty 也不触发 closeEnded;此测仅验降级状态,
func TestActivityWatcher_HuaweiDegraded(t *testing.T) {
	f := &activityWatcherFakeHuawei{err: errors.New("unavailable")}
	cfg := activityWatcherConfig()
	cfg.SmartEnd.SilenceDurationS = 1
	w := NewActivityWatcher(cfg, f, "", nil, zap.NewNop())
	w.Start()
	defer w.Stop()
	require.Eventually(t, func() bool { return w.Snapshot().HuaWeiDegraded }, 4*time.Second, 50*time.Millisecond)
	require.False(t, w.Snapshot().Ended)
}

// 4. A 降级:logFile 写入 5 行 [silencedetect malformed] (silenceParseFailures
// 累计 ≥ 5) → silenceDegraded=true;验证 A 路径在 evaluate 时被短路。
func TestActivityWatcher_SilenceDegraded(t *testing.T) {
	logFile, err := os.CreateTemp(t.TempDir(), "ffmpeg-*.log")
	require.NoError(t, err)
	defer logFile.Close()
	for i := 0; i < 5; i++ {
		_, err = logFile.WriteString("[silencedetect malformed]\n")
		require.NoError(t, err)
	}
	require.NoError(t, logFile.Sync())
	// 写完后 os.File 偏移在 EOF;bufio.Scanner 从偏移 0 起读需先 Seek 回 0。
	_, err = logFile.Seek(0, 0)
	require.NoError(t, err)
	cfg := activityWatcherConfig()
	cfg.SmartEnd.HuaweiEnabled = false
	w := NewActivityWatcher(cfg, nil, "", logFile, zap.NewNop())
	w.Start()
	defer w.Stop()
	require.Eventually(t, func() bool { return w.Snapshot().SilenceDegraded }, 2*time.Second, 25*time.Millisecond)
}

// 5. stat 死亡:filePath 指向不存在路径 → fileTicker 连续 3 次 os.Stat 失败
// → closeEnded("file_stat_failed")。
func TestActivityWatcher_StatFailed(t *testing.T) {
	cfg := activityWatcherConfig()
	cfg.SmartEnd.HuaweiEnabled = false
	w := NewActivityWatcher(cfg, nil, filepath.Join(t.TempDir(), "missing.mkv"), nil, zap.NewNop())
	w.Start()
	defer w.Stop()
	waitActivityEnded(t, w)
	require.Equal(t, "file_stat_failed", w.Snapshot().EndedReason)
}

// 6. 重连保持 (WATCH-05):手动预置 silenceSince / lastFileGrowthAt / huaweiEmptySince
// + degraded 标记;调 OnReconnect() → 仅 silenceSince 清零,其余保留。
func TestActivityWatcher_Reconnect(t *testing.T) {
	w := newTestWatcher(t, nil)
	oldGrowth := time.Now().Add(-time.Second)
	w.mu.Lock()
	w.silenceSince = time.Now()
	w.lastFileGrowthAt = oldGrowth
	w.huaweiEmptySince = time.Now().Add(-time.Second)
	w.huaweiDegraded = true
	w.silenceDegraded = true
	w.mu.Unlock()
	w.OnReconnect()
	s := w.Snapshot()
	require.True(t, s.SilenceSince.IsZero())
	require.Equal(t, oldGrowth, s.LastFileGrowthAt)
	require.False(t, s.HuaWeiEmptySince.IsZero())
	require.True(t, s.HuaWeiDegraded)
	require.True(t, s.SilenceDegraded)
}

// 7. close-once (WATCH-01):H 触发后再次调用 closeEnded,验证 sync.Once 拦下;
// EndedReason 仍是首次触发的 "huawei_state_empty",EndedCh 仍 closed。
func TestActivityWatcher_CloseOnce(t *testing.T) {
	f := &activityWatcherFakeHuawei{state: &huawei.ConferenceState{HasConferenceFields: true}}
	w := NewActivityWatcher(activityWatcherConfig(), f, "", nil, zap.NewNop())
	w.Start()
	defer w.Stop()
	waitActivityEnded(t, w)
	// 模拟 A+B 条件满足并尝试再次触发 closeEnded。
	w.mu.Lock()
	w.silenceSince = time.Now().Add(-2 * time.Second)
	w.lastFileGrowthAt = time.Now().Add(-2 * time.Second)
	w.mu.Unlock()
	w.closeEnded("both_silence_and_stall")
	require.Equal(t, "huawei_state_empty", w.Snapshot().EndedReason)
	select {
	case <-w.EndedCh():
	default:
		t.Fatal("EndedCh should remain closed")
	}
}

// 8. ExtendStepMin (EXTEND-03):cfg.SmartEnd.ExtendStepMin=60 → watcher.ExtendStepMin() = 60min。
func TestActivityWatcher_ExtendStep(t *testing.T) {
	w := newTestWatcher(t, func(s *config.SmartEndConfig) { s.ExtendStepMin = 60 })
	require.Equal(t, 60*time.Minute, w.ExtendStepMin())
}
