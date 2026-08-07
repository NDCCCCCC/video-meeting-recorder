// Package recorder: ActivityWatcher 整合 H + A + B 三类信号,按 OR 关系
// 收敛到单一 taskEndedCh 通道。Phase 25 scheduler 仅需读 <-EndedCh + IsActive
// + Snapshot,无须感知三源细节。
//
// 四个采样 goroutine (silenceScanner / fileTicker / huaweiPoller / decisionTicker)
// 各自监听 ctx.Done 退出;close-once 由 sync.Once 物理保证 (RESEARCH.md Pitfall 2);
// OnReconnect 仅清 silenceSince 不动文件 ticker / H 状态 (WATCH-05)。
package recorder

import (
	"bufio"
	"context"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/observability"
)

// silenceFailureThreshold 是 SilenceParser 连续解析失败的硬编码阈值,触发
// silenceDegraded 关闭 A 路径。固定 5 而非 cfg 字段,理由:24-VALIDATION.md
// §ActivityWatcher scenario matrix "A 降级" 行明确写"5 次连续",与 cfg 其他
// 阈值 (3/3) 区分以反映 ffmpeg stderr 文本噪声更高的现实。
const silenceFailureThreshold = 5

// ActivityWatcher 整合 H + A + B 三类信号 + 多级降级状态机 + close-once。
type ActivityWatcher struct {
	// 注入字段 (构造时一次性赋值,运行期不变)。
	cfg       *config.Config
	huaweiCli HuaweiStateClient
	filePath  string
	logFile   *os.File
	logger    *zap.Logger

	// 时间源 (测试可替换为 fakeClock)。
	now func() time.Time

	// 状态字段 — 全部读写过 mu 锁。
	mu               sync.Mutex
	silenceSince     time.Time
	lastFileSize     int64
	lastFileGrowthAt time.Time
	// lastFileGrowthBps 最近一次"达标"的速率缓存 (Phase 25 AUDIT-02）。
	// 仅在 fileTicker 走 growthBps >= FileMinGrowthBPS 分支时刷新,
	// 未达标不刷新 — 与 lastFileGrowthAt 同一周期语义。
	lastFileGrowthBps    int64
	huaweiEmptySince     time.Time
	huaweiLastState      string
	huaweiLastJoinSum    int
	huaweiConsecFailures int
	silenceParseFailures int
	statConsecFailures   int
	huaweiDegraded       bool
	silenceDegraded      bool
	endedReason          string

	// 关闭字段 — close-once 由 sync.Once 保证。
	closeOnce   sync.Once
	taskEndedCh chan struct{}

	// 运行时字段 — Start 初始化,Stop 清理。
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ActivitySnapshot 是 watcher 状态的可移植值拷贝,Phase 25 scheduler 可持有
// 长期观察 watcher 状态变化。每次 Snapshot 调用都过 w.mu 锁并复制字段。
// FileSizeBytes 与 FileGrowthBps 由 fileTicker 维护 (Phase 25 AUDIT-02),
// 供 service 层 audit log 序列化使用 — 与 LastFileGrowthAt 一起构成
// "文件侧 telemetry" 的完整快照。
type ActivitySnapshot struct {
	SilenceSince         time.Time
	LastFileGrowthAt     time.Time
	HuaWeiEmptySince     time.Time
	LastHuaWeiStateEmpty bool
	HuaWeiDegraded       bool
	SilenceDegraded      bool
	Ended                bool
	EndedReason          string
	LastSilenceStart     time.Time
	TotalSilenceDuration time.Duration
	// FileSizeBytes 最近一次 os.Stat 读到的文件大小 (bytes);fileTicker 每次
	// 循环都更新,与 lastFileSize 同步,供 Snapshot 暴露。Phase 25 AUDIT-02。
	FileSizeBytes int64 `json:"file_size_bytes"`
	// FileGrowthBps 最近一次"达标"(growthBps >= FileMinGrowthBPS)的速率缓存;
	// 未达标时不刷新 — 与 LastFileGrowthAt 同一周期语义,scheduler 读 Snapshot
	// 时永远是"最后一次达标"的速率,适合 audit log 与配置阈值对比。Phase 25 AUDIT-02。
	FileGrowthBps int64 `json:"file_growth_bps"`
}

// fakeClock 提供确定性时间源,ActivityWatcher.now 字段可替换为
// fakeClock.Now 驱动测试。24-04 集成测试也会使用本类型驱动 silent/deadline
// 判定路径。
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

// Now 返回当前 fake 时间,加锁保证并发可见。
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

// Advance 把 fake 时间向前推进 d。常用于测试 long-running 状态机。
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// NewActivityWatcher 构造 ActivityWatcher。
//
// 参数:
//   - cfg       : 全局配置,本 watcher 仅读 cfg.SmartEnd.* 字段
//   - huaweiCli : 状态拉取客户端;HuaweiEnabled=false 时可传 nil
//   - filePath  : MKV 录制路径,供 fileTicker 周期 os.Stat
//   - logFile   : ffmpeg stderr 句柄,供 silenceScanner 周期 Scan;可传 nil
//     (比如外部单元测试不启 scanner),silenceScanner 入口判断
//   - logger    : zap logger;nil 时降级为 zap.NewNop() 避免 nil 指针
func NewActivityWatcher(cfg *config.Config, huaweiCli HuaweiStateClient, filePath string, logFile *os.File, logger *zap.Logger) *ActivityWatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ActivityWatcher{
		cfg:         cfg,
		huaweiCli:   huaweiCli,
		filePath:    filePath,
		logFile:     logFile,
		logger:      logger,
		now:         time.Now,
		taskEndedCh: make(chan struct{}, 1),
	}
}

// Start 启动 4 个采样 goroutine (silenceScanner / fileTicker / huaweiPoller /
// decisionTicker)。多次 Start 是未定义行为(预期仅调用一次)。
func (w *ActivityWatcher) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.wg.Add(4)
	go w.silenceScanner(ctx)
	go w.fileTicker(ctx)
	go w.huaweiPoller(ctx)
	go w.decisionTicker(ctx)
}

// Stop 取消 ctx,等待 4 个 goroutine 退出,最后 close(taskEndedCh) (若未关闭)。
// 多次 Stop 是安全的:closeOnce 保证 channel 仅关闭一次。
func (w *ActivityWatcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	w.closeEnded("stopped")
}

// EndedCh 返回只读 taskEndedCh 通道,Phase 25 scheduler 用 `<-w.EndedCh()`
// 阻塞收结束信号。close-once 由 sync.Once 物理保证多次 close 不 panic。
func (w *ActivityWatcher) EndedCh() <-chan struct{} {
	return w.taskEndedCh
}

// IsActive 报告 watcher 是否仍在"未结束"状态。由 close-once 语义保证
// (close 后 taskEndedCh 立即可读,select 进入 default 失败)。Phase 25
// scheduler 用 IsActive 把"已结束"task 排除出活跃集。
func (w *ActivityWatcher) IsActive() bool {
	select {
	case <-w.taskEndedCh:
		return false
	default:
		return true
	}
}

// Snapshot 复制当前状态。锁内复制保证读到一致的字段组合。Phase 25 调度器
// 在每个调度 tick 读 Snapshot 决策是否触发 extend/early-end。
func (w *ActivityWatcher) Snapshot() ActivitySnapshot {
	w.mu.Lock()
	defer w.mu.Unlock()
	return ActivitySnapshot{
		SilenceSince:         w.silenceSince,
		LastFileGrowthAt:     w.lastFileGrowthAt,
		HuaWeiEmptySince:     w.huaweiEmptySince,
		LastHuaWeiStateEmpty: !w.huaweiEmptySince.IsZero(),
		HuaWeiDegraded:       w.huaweiDegraded,
		SilenceDegraded:      w.silenceDegraded,
		Ended:                w.endedReason != "",
		EndedReason:          w.endedReason,
		LastSilenceStart:     w.silenceSince, // 简化:此处为最后已知静音起点
		TotalSilenceDuration: w.totalSilenceDurationLocked(),
		FileSizeBytes:        w.lastFileSize,
		FileGrowthBps:        w.lastFileGrowthBps,
	}
}

// totalSilenceDurationLocked 计算从 silenceSince 起到 now 的持续时长,
// 供 Snapshot 暴露 TotalSilenceDuration 字段。调用方需持有 w.mu。
func (w *ActivityWatcher) totalSilenceDurationLocked() time.Duration {
	if w.silenceSince.IsZero() {
		return 0
	}
	return w.now().Sub(w.silenceSince)
}

// ExtendStepMin 把 cfg.SmartEnd.ExtendStepMin (分钟) 换算为 time.Duration。
// Phase 25 scheduler 用 ExtendStepMin 作为单次延长步长,与 cfg 联动。
func (w *ActivityWatcher) ExtendStepMin() time.Duration {
	return time.Duration(w.cfg.SmartEnd.ExtendStepMin) * time.Minute
}

// HuaweiEnabled 直接读 cfg.SmartEnd.HuaweiEnabled。Phase 25 coordinator 用
// HuaweiEnabled 判定是否注入 huaweiCli (避免在 huawei_enabled=false 时占用
// *huawei.Client 资源)。暴露本 getter 让 coordinator 无须直接读 cfg。
func (w *ActivityWatcher) HuaweiEnabled() bool {
	return w.cfg.SmartEnd.HuaweiEnabled
}

// OnReconnect 是 ffmpeg 重连回调,由 coordinator.restartRecording 同步触发。
// 仅清 silenceSince;不动 lastFileGrowthAt / huaweiEmptySince / huaweiDegraded
// / silenceDegraded — 录制重启期间已有 silenceSince 应被遗忘,但文件增长与
// 华为状态应保留(WATCH-05 + 24-VALIDATION.md "重连保持" 场景)。
func (w *ActivityWatcher) OnReconnect() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.silenceSince = time.Time{}
}

// closeEnded 关闭 taskEndedCh,reason 写入 endedReason (供 Snapshot 暴露)。
// sync.Once 物理保证多次 close 不 panic;后续调用仅记 Debug。
func (w *ActivityWatcher) closeEnded(reason string) {
	w.closeOnce.Do(func() {
		w.mu.Lock()
		w.endedReason = reason
		w.mu.Unlock()
		close(w.taskEndedCh)
		w.logger.Info("smart_early_end (watcher)",
			zap.String("reason", reason),
		)
	})
}

// ---------------------------------------------------------------------------
// 采样 goroutine
// ---------------------------------------------------------------------------

// silenceScanner 读 ffmpeg stderr 流,逐行 Parse silencedetect 行。
// 行为契约:
//   - 非目标行 (不含 "[silencedetect" 前缀) → 静默丢弃,不计 failure
//   - Parse error (含前缀但三条正则都没命中) → silenceParseFailures++;达
//     silenceFailureThreshold 触发 silenceDegraded
//   - 成功解析 (Start/End/None) → silenceParseFailures 回 0
//   - Kind == Start → silenceSince = now
//   - Kind == End   → silenceSince = zero
//   - Kind == None (含 duration-only 行) → 仅供 Snapshot 不动 silenceSince
//
// logFile 为 nil 时直接 return (与 newTestWatcher 传 nil 兼容)。
func (w *ActivityWatcher) silenceScanner(ctx context.Context) {
	defer w.wg.Done()
	if w.logFile == nil {
		<-ctx.Done()
		return
	}
	parser := NewSilenceParser()
	scanner := bufio.NewScanner(w.logFile)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := scanner.Text()
		ev, err := parser.Parse(line)
		if err != nil {
			w.mu.Lock()
			w.silenceParseFailures++
			failures := w.silenceParseFailures
			w.mu.Unlock()
			if failures >= silenceFailureThreshold && !w.silenceDegradedFlag() {
				w.mu.Lock()
				w.silenceDegraded = true
				w.mu.Unlock()
				// 降级开关关闭时仍记 "degrade_on_silence_loss=false" 但不关闭路径
				// 按 plan: 仅设 silenceDegraded=true;cfg.DegradeOnSilenceLoss=false
				// 的语义在 Phase 25 决策侧处理,本阶段保持简单。
				_ = w.cfg.SmartEnd.DegradeOnSilenceLoss
				w.logger.Warn("activity_watcher_degraded",
					zap.String("reason", "silence_parser_failed"),
					zap.Int("consecutive_failures", failures),
				)
				// OBS-05: atomic counter +1 (Phase 25 OBS-04 接入点)。
				// 放在 WARN 之后,确保 log fire 与 counter increment 1:1 同步。
				observability.RecordWatcherDegraded()
			}
			continue
		}
		// 成功解析 → 重置 failure 计数
		w.mu.Lock()
		w.silenceParseFailures = 0
		w.mu.Unlock()
		if ev.Kind == SilenceEventStart {
			w.mu.Lock()
			w.silenceSince = w.now()
			w.mu.Unlock()
		} else if ev.Kind == SilenceEventEnd {
			w.mu.Lock()
			w.silenceSince = time.Time{}
			w.mu.Unlock()
		}
	}
	// Scan 退出 (EOF or error) — 等 ctx.Done 后清理。
	<-ctx.Done()
}

// silenceDegradedFlag 锁内读 silenceDegraded 字段,避免 silenceScanner 主体
// 长时间持锁读小字段。
func (w *ActivityWatcher) silenceDegradedFlag() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.silenceDegraded
}

// fileTicker 按 CheckIntervalS 周期 os.Stat filePath,计算 growthBps:
//   - growthBps = deltaBytes * 8 / CheckIntervalS
//     (deltaBytes < 0 时按 0 视作文件 truncate/替换; CheckIntervalS <= 0
//     兜底为 1 避免除零 — Validate 已限定 > 0,这里是保险)
//   - growthBps >= FileMinGrowthBPS → lastFileGrowthAt = now 且 lastFileSize = size
//   - growthBps <  FileMinGrowthBPS → 仅 lastFileSize = size,不更新 lastFileGrowthAt
//     (让 decisionTicker 自然累计 now-lastFileGrowthAt 触发 stall)
//   - Stat error → statConsecFailures++;>= StatFailureThreshold 触发
//     closeEnded("file_stat_failed"),但**不**清零 lastFileGrowthAt (Pitfall 6)
func (w *ActivityWatcher) fileTicker(ctx context.Context) {
	defer w.wg.Done()
	if w.filePath == "" {
		// 单元测试场景:filePath 留空 → ticker 只走 ctx.Done 退出
		<-ctx.Done()
		return
	}
	interval := time.Duration(w.cfg.SmartEnd.CheckIntervalS) * time.Second
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(w.filePath)
			if err != nil {
				w.mu.Lock()
				w.statConsecFailures++
				consec := w.statConsecFailures
				w.mu.Unlock()
				if consec >= w.cfg.SmartEnd.StatFailureThreshold {
					// file_stat_failed 是早期结束信号 (INFO),不是 watcher 降级事件 (ERROR);
					// 刻意不计为 RecordWatcherDegraded — 参见 25-03 plan §OBS-04 设计备注。
					// closeEnded 内已发 smart_early_end (watcher) INFO 日志 + close taskEndedCh,
					// scheduler handleTaskEnded 据此走 MarkTaskEndedEarly(snap.EndedReason,
					// byHuaWeiAPI=false) → OBS-02 计数;此处不再额外 OBS-04 计数避免双重信号。
					w.closeEnded("file_stat_failed")
					return
				}
				continue
			}
			size := info.Size()
			checkInterval := int64(w.cfg.SmartEnd.CheckIntervalS)
			if checkInterval <= 0 {
				checkInterval = 1
			}
			w.mu.Lock()
			deltaBytes := size - w.lastFileSize
			if deltaBytes < 0 {
				deltaBytes = 0
			}
			growthBps := deltaBytes * 8 / checkInterval
			if growthBps >= w.cfg.SmartEnd.FileMinGrowthBPS {
				w.lastFileGrowthAt = w.now()
				w.lastFileSize = size
				w.lastFileGrowthBps = growthBps
				w.statConsecFailures = 0
			} else {
				// 未达标:仅更新 lastFileSize,不更新 lastFileGrowthAt / lastFileGrowthBps
				w.lastFileSize = size
			}
			w.mu.Unlock()
		}
	}
}

// huaweiPoller 按 HuaweiPollIntervalS 周期 GetConferenceState。
//   - HuaweiEnabled=false 时入口直接 return (Open Question 2 推荐 — 跳过
//     goroutine 节省资源;Phase 25 调度不启用 watcher 时 cfg.SmartEnd.Enabled
//     也走该判断路径)
//   - 成功:根据 ConferenceState.HasConferenceFields 选判据;ConfState=="" &&
//     JoinSum==0 (新设备) 或 IsInConf==0 (老设备 fallback) → huaweiEmptySince =
//     now;否则 huaweiEmptySince = zero
//   - 失败:huaweiConsecFailures++;>= HuaweiFailureThreshold → huaweiDegraded=true
//   - close H 路径
func (w *ActivityWatcher) huaweiPoller(ctx context.Context) {
	defer w.wg.Done()
	if !w.cfg.SmartEnd.HuaweiEnabled {
		<-ctx.Done()
		return
	}
	if w.huaweiCli == nil {
		// HuaweEnabled=true 但 cli 未注入 → 视为不可用,直接降级
		w.mu.Lock()
		w.huaweiDegraded = true
		w.mu.Unlock()
		w.logger.Warn("activity_watcher_degraded",
			zap.String("reason", "huawei_client_nil"),
		)
		// OBS-05: atomic counter +1 (Phase 25 OBS-04 接入点)。
		observability.RecordWatcherDegraded()
		<-ctx.Done()
		return
	}
	interval := time.Duration(w.cfg.SmartEnd.HuaweiPollIntervalS) * time.Second
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := w.huaweiCli.GetConferenceState(ctx)
			if err != nil {
				w.mu.Lock()
				w.huaweiConsecFailures++
				consec := w.huaweiConsecFailures
				w.mu.Unlock()
				if consec >= w.cfg.SmartEnd.HuaweiFailureThreshold {
					w.mu.Lock()
					w.huaweiDegraded = true
					w.mu.Unlock()
					w.logger.Warn("activity_watcher_degraded",
						zap.String("reason", "huawei_api_unreachable"),
						zap.Int("consecutive_failures", consec),
					)
					// OBS-05: atomic counter +1 (Phase 25 OBS-04 接入点)。
					observability.RecordWatcherDegraded()
				}
				// 失败时保留 huaweiEmptySince (不重置) — 失败不代表"会议恢复"
				continue
			}
			w.mu.Lock()
			w.huaweiConsecFailures = 0
			w.huaweiLastState = state.ConfState
			w.huaweiLastJoinSum = state.JoinSum
			w.mu.Unlock()
			// 判据: HasConferenceFields → ConfState+JoinSum;否则 IsInConf fallback
			empty := false
			if state.HasConferenceFields {
				empty = state.ConfState == "" && state.JoinSum == 0
			} else {
				empty = state.IsInConf == 0
			}
			w.mu.Lock()
			if empty {
				if w.huaweiEmptySince.IsZero() {
					w.huaweiEmptySince = w.now()
				}
			} else {
				w.huaweiEmptySince = time.Time{}
			}
			w.mu.Unlock()
		}
	}
}

// decisionTicker 每 CheckIntervalS 评估 OR 关闭条件:
//   - H 路径: !huaweiDegraded && !huaweiEmptySince.IsZero() && now-huaweiEmptySince
//     >= HuaweiPersistS → closeEnded("huawei_state_empty")
//   - A+B 路径: !silenceDegraded && !silenceSince.IsZero() &&
//     now-silenceSince >= SilenceDurationS && !lastFileGrowthAt.IsZero() &&
//     now-lastFileGrowthAt >= FileStallS → closeEnded("both_silence_and_stall")
func (w *ActivityWatcher) decisionTicker(ctx context.Context) {
	defer w.wg.Done()
	interval := time.Duration(w.cfg.SmartEnd.CheckIntervalS) * time.Second
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			now := w.now()
			hwDegraded := w.huaweiDegraded
			hwEmptySince := w.huaweiEmptySince
			silDegraded := w.silenceDegraded
			silSince := w.silenceSince
			lastGrowth := w.lastFileGrowthAt
			w.mu.Unlock()

			// H 路径
			if !hwDegraded && !hwEmptySince.IsZero() &&
				now.Sub(hwEmptySince) >= time.Duration(w.cfg.SmartEnd.HuaweiPersistS)*time.Second {
				w.closeEnded("huawei_state_empty")
				return
			}
			// A+B 路径
			if !silDegraded && !silSince.IsZero() &&
				now.Sub(silSince) >= time.Duration(w.cfg.SmartEnd.SilenceDurationS)*time.Second &&
				!lastGrowth.IsZero() &&
				now.Sub(lastGrowth) >= time.Duration(w.cfg.SmartEnd.FileStallS)*time.Second {
				w.closeEnded("both_silence_and_stall")
				return
			}
		}
	}
}
