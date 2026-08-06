---
phase: 24-activitywatcher-silencedetect
plan: 02
subsystem: recorder
tags: [activity-watcher, goroutine, sync.once, detct-03, watch-05, huawei, or-judgement]
tech_stack:
  added: []
  patterns: [triple-source-or-merge, sync.Once-close-once, rolling-window-bps-gate, huawei-disabled-fast-return]
key_files:
  created:
    - internal/recorder/activity_watcher.go
    - internal/recorder/activity_watcher_test.go
  modified: []
decisions:
  - "huaweiPoller 入口检查 cfg.SmartEnd.HuaweiEnabled=false 时直接 return (Open Question 2 推荐 — 节省 goroutine 资源且与 cfg 级开关一致);HuaweiEnabled=true 但 huaweiCli=nil 视为参数缺失立即降级 + 日志,避免运行期 panic"
  - "fileTicker 严格按 REQUIREMENTS.md DETECT-03 行 16 实现:growthBps = deltaBytes*8/CheckIntervalS ≥ FileMinGrowthBPS (默认 1024 B/s) 才更新 lastFileGrowthAt;未达标仅更新 lastFileSize 不动 lastFileGrowthAt,让 decisionTicker 自然累计 stall 时间 (24-RESEARCH.md Pitfall 6);StatFailureThreshold 连续 3 次失败 closeEnded(\"file_stat_failed\") 但**不**清零 lastFileGrowthAt"
  - "close-once 用 sync.Once 物理保证 (24-RESEARCH.md Pitfall 2 推荐);endReason 写入字段后被 Snapshot.EndedReason 暴露,无需额外 channel"
  - "OnReconnect 锁内仅清 silenceSince,不动 lastFileGrowthAt / huaweiEmptySince / huaweiDegraded / silenceDegraded (WATCH-05 + 24-VALIDATION.md '重连保持' 场景) — 录制重启期间已有 silenceSince 应被遗忘,但文件增长与华为状态应保留"
  - "IsActive 走 select-default 不持锁 (与 close-once 语义一致);IsActive 末尾不读 endedReason,直接探测 taskEndedCh 关闭信号,避免锁内细读"
  - "silenceFailureThreshold = 5 硬编码常量 (非 cfg 字段);24-VALIDATION.md §ActivityWatcher scenario matrix 'A 降级' 行明确写'5 次连续',与 cfg 阈值 (3/3) 区分以反映 ffmpeg stderr 文本噪声更高"
  - "Duration Step 1 — 24-01 Plan 字面 'use applySmartEndDefaults in test' 引用了 config 包私有函数;Step 2 替换为 SmartEndConfig 字面量 (覆盖 11 数字字段默认值),避免引用未导出符号 (Rule 3 - Blocking)"
metrics:
  duration: "~8 min"
  tasks: 1
  files_created: 2
  files_modified: 0
  commits: 3
  completed_date: 2026-08-06
---

# Phase 24 Plan 02 Summary: ActivityWatcher H+A+B OR 整合 + close-once + 6 公开 API (DETECT-03 + WATCH-01..05)

## 概览

落地 Phase 24 的核心编排模块 — `ActivityWatcher` 把 H (Huawei conference state) + A (silencedetect stderr) + B (文件增长速率) 三类异源信号按 OR 关系收敛到单一 `taskEndedCh` 通道，并实现多级降级、close-once 关闭、重连保持计时不重置。

**Phase 25 scheduler 集成面**：
- `<-watcher.EndedCh()` 阻塞收结束信号
- `watcher.IsActive()` 过滤"已结束"task
- `watcher.Snapshot()` 拿状态副本做 extend/early-end 决策
- `watcher.ExtendStepMin()` 拿单次延长步长
- `watcher.HuaweiEnabled()` 判定是否注入 huaweiCli
- `watcher.OnReconnect()` 在 ffmpeg 重连回调时同步调用清静音计时

**单文件、单依赖**：`internal/recorder/activity_watcher.go` 496 行（含中文 doc-comment），0 新增外部依赖；所有 import 走现有 go.mod（bufio / context / os / sync / time / go.uber.org/zap / config）。

---

## 完成的 Task

### Task 1 — ActivityWatcher struct + 4 goroutine + close-once + 6 公开 API

| Item | Value |
|------|-------|
| Commit (RED) | `88b20f0` `test(24-02): add failing test for ActivityWatcher contract (RED gate)` |
| Commit (GREEN) | `5ff35d9` `feat(24-02): implement ActivityWatcher (H+A+B OR + close-once)` |
| Commit (fix) | `0657cb3` `fix(24-02): use explicit SmartEnd fields in test fixture (no applySmartEndDefaults)` |
| Files | `internal/recorder/activity_watcher.go` (496 行) / `internal/recorder/activity_watcher_test.go` (195 行) |
| Verify | `go build ./internal/recorder` 0 退出 / `go vet ./internal/recorder` 无告警 / `go test ./internal/recorder -count=1` PASS（7 新单测 + 既有 recorder 单测全绿）|

**类型定义**（对齐 24-RESEARCH.md §ActivityWatcher 设计 §163-220）：

- `ActivityWatcher`:
  - 注入字段：`cfg *config.Config` / `huaweiCli HuaweiStateClient` / `filePath string` / `logFile *os.File` / `logger *zap.Logger` / `now func() time.Time`
  - 状态字段（锁内）：`mu sync.Mutex` / `silenceSince` / `lastFileSize` / `lastFileGrowthAt` / `huaweiEmptySince` / `huaweiLastState` / `huaweiLastJoinSum` / `huaweiConsecFailures` / `silenceParseFailures` / `statConsecFailures` / `huaweiDegraded` / `silenceDegraded` / `endedReason`
  - 关闭字段：`closeOnce sync.Once` / `taskEndedCh chan struct{}` (buffered 1)
  - 运行时字段：`cancel context.CancelFunc` / `wg sync.WaitGroup`
- `ActivitySnapshot` (10 字段): `SilenceSince` / `LastFileGrowthAt` / `HuaWeiEmptySince` / `LastHuaWeiStateEmpty` / `HuaWeiDegraded` / `SilenceDegraded` / `Ended` / `EndedReason` / `LastSilenceStart` / `TotalSilenceDuration`
- `fakeClock` (mu + t + Now/Advance): 供 24-04 集成测试驱动确定性时间

**4 个采样 goroutine**（各自监听 ctx.Done 退出）：

1. **silenceScanner(ctx)** — `bufio.NewScanner(w.logFile)` 逐行 `Parse(line)`：
   - 非目标行 → 静默丢弃,不计 failure
   - Parse error → `silenceParseFailures++`；达 `silenceFailureThreshold=5` 触发 `silenceDegraded=true` + log
   - 成功解析 → 重置 failure 计数
   - `Kind == Start` → `silenceSince = now()`
   - `Kind == End` → `silenceSince = time.Time{}`
   - `Kind == None` (含 duration-only 行) → 不动 silenceSince
   - logFile=nil 时入口直接 `<-ctx.Done()` 退出（与单测传 nil 兼容）

2. **fileTicker(ctx)** — `time.NewTicker(CheckIntervalS)` 周期 `os.Stat(filePath)`：
   - **REQUIREMENTS.md DETECT-03 行 16 严格实现**: `growthBps = deltaBytes * 8 / CheckIntervalS` (deltaBytes<0 兜底 0；CheckIntervalS<=0 兜底 1 避免除零)
   - `growthBps >= fileMinGrowthBPS` (默认 1024 B/s) → `lastFileGrowthAt = now()` + `lastFileSize = size` + `statConsecFailures = 0`
   - `growthBps < fileMinGrowthBPS` → 仅更新 `lastFileSize = size`,**不**更新 `lastFileGrowthAt` (Pitfall 6)
   - Stat error → `statConsecFailures++`；达 `StatFailureThreshold=3` 触发 `closeEnded("file_stat_failed")` 但**不**清零 lastFileGrowthAt (Pitfall 6)
   - filePath="" 时入口直接 `<-ctx.Done()` 退出（单测兼容）

3. **huaweiPoller(ctx)** — `time.NewTicker(HuaweiPollIntervalS)` 周期 `GetConferenceState`：
   - **`cfg.SmartEnd.HuaweiEnabled=false` 时入口直接 return** (Open Question 2 推荐)
   - `HuaweiEnabled=true` 但 `huaweiCli=nil` → 立即 `huaweiDegraded=true` + log 警告
   - 成功: 按 `state.HasConferenceFields` 选判据；`ConfState=="" && JoinSum==0` (新设备) 或 `IsInConf==0` (老设备 fallback) → 设置 `huaweiEmptySince = now()`；否则清零
   - 失败: `huaweiConsecFailures++`；达 `HuaweiFailureThreshold=3` 触发 `huaweiDegraded=true` + log

4. **decisionTicker(ctx)** — `time.NewTicker(CheckIntervalS)` 评估 OR 关闭:
   - **H 路径**: `!huaweiDegraded && !huaweiEmptySince.IsZero() && now-huaweiEmptySince >= HuaweiPersistS` → `closeEnded("huawei_state_empty")`
   - **A+B 路径**: `!silenceDegraded && !silenceSince.IsZero() && now-silenceSince >= SilenceDurationS && !lastFileGrowthAt.IsZero() && now-lastFileGrowthAt >= FileStallS` → `closeEnded("both_silence_and_stall")`

**6 个公开 API**：

| API | 行为 | 用途 |
|------|------|------|
| `EndedCh() <-chan struct{}` | 返回只读 `taskEndedCh` | Phase 25 scheduler `<-EndedCh()` 阻塞收结束信号 |
| `IsActive() bool` | `select { case <-taskEndedCh: return false; default: return true }` | 调度侧过滤"已结束" task |
| `Snapshot() ActivitySnapshot` | 锁内值拷贝 10 字段 | 调度侧 extend/early-end 决策 |
| `ExtendStepMin() time.Duration` | `time.Duration(cfg.SmartEnd.ExtendStepMin) * time.Minute` | 单次延长步长 |
| `OnReconnect()` | 锁内仅清 `silenceSince = time.Time{}` | ffmpeg 重连回调时调用 (WATCH-05) |
| `HuaweiEnabled() bool` | `cfg.SmartEnd.HuaweiEnabled` | coordinator 按需注入 huaweiCli |

**关闭与回调**：

- `closeEnded(reason)` 用 `sync.Once` 物理保证多次 close 不 panic；`endReason` 写入字段后被 Snapshot 暴露；首次 close 记 `logger.Info("smart_early_end (watcher)", reason=...)`
- `OnReconnect()` 锁内仅清 `silenceSince`,不动 `lastFileGrowthAt` / `huaweiEmptySince` / `huaweiDegraded` / `silenceDegraded` (WATCH-05 + 24-VALIDATION.md "重连保持" 场景)

**单测覆盖**（7 个用例,1 个含 2 子测,1 个含 2 子测,共 9 个断言）：

| Test | 覆盖点 | 24-VALIDATION.md 场景行 |
|------|--------|--------------------------|
| `TestNewActivityWatcher` | 返回非 nil + 初始 Snapshot + EndedCh 未关闭 | (基础契约) |
| `TestExtendStepMin` | explicit 60 + default 30 | "ExtendStepMin" |
| `TestHuaweiEnabled` | false by default + true after set | "HuaweiEnabled getter" |
| `TestIsActive` | true → Start → Stop → false | "close-once" |
| `TestOnReconnect` | silenceSince 清零, lastFileGrowthAt / huaweiEmptySince / degraded 保留 | "重连保持" |
| `TestSnapshot` | 零值 → 状态变更 → 拷贝语义无残留 | (Snapshot 字段读取) |
| `TestFakeClock` | Now + Advance 确定性时间 | (24-04 集成测试驱动) |

**未覆盖**（已在 24-04 集成测试计划）：
- `silenceScanner` 真实 bufio 流驱动（24-01 SilenceParser 已测, 24-04 测与 watcher 联动）
- `fileTicker` 真实 stat 路径 + growthBps 跨越 1024 阈值 (24-04 测)
- `huaweiPoller` 真实 `GetConferenceState` fake 驱动 (24-04 测)
- `decisionTicker` OR 触发（24-04 fake clock 驱动）
- `closeEnded` 多 close 不 panic (24-04 验)

---

## Deviations from Plan

### D1: silenceScanner 解析失败分支里用单独的 `silenceDegradedFlag()` 辅助读 degraded

**Rule 应用**: Rule 1 (auto-fix correctness) — 避免长时间持锁读 `w.silenceDegraded` 小字段；保持 `silenceScanner` 主路径少持锁。

**影响**: 0 行为变化，仅微调锁粒度。

### D2: huaweiPoller 增加 "HuaweiEnabled=true 但 huaweiCli=nil" 立即降级分支

**Plan 字面**: 未明确处理 `huaweiCli=nil` 且 `HuaweiEnabled=true` 这种参数组合。
**Rule 应用**: Rule 2 (auto-add missing critical functionality) — 缺此判断则运行期 `w.huaweiCli.GetConferenceState(ctx)` 在 nil receiver 调用上 panic（华为 nil 接收者调用是非 nil 检查绕过）。
**Fix**: 入口检查 `cli == nil` 时 `huaweiDegraded = true` + `logger.Warn("activity_watcher_degraded", reason="huawei_client_nil")` 后 `<-ctx.Done()` 退出。
**影响**: 0 行为变化 (Phase 25 注入时不应触发)，但运行期健壮性提升。

### D3: test fixture 不引用 `config.applySmartEndDefaults` (Step 2 修正)

**Plan 字面 action**（24-02 §Task 1 read_first 列的 `internal/config/smart_end.go`）：未直接命令测试使用 `applySmartEndDefaults`，但初始编写的 RED gate 测试文件引用了。
**冲突**: `applySmartEndDefaults` 是 `config` 包私有函数，`recorder` 包测试无法调用。
**Rule 应用**: Rule 3 (blocking) — 编译失败会阻塞所有后续步骤。
**Fix**: 把测试 fixture 改为 `cfg.SmartEnd = config.SmartEndConfig{...}` 字面量,显式覆盖 11 数字字段默认值（match `applySmartEndDefaults` 行为），不动 3 bool 字段。
**Commit**: `0657cb3` (fix, 独立 commit 切断 GREEN 提交污染).
**影响**: 0 行为变化，测试断言不变。

---

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED (`test(...)`) | `88b20f0` | ✅ |
| GREEN (`feat(...)`) | `5ff35d9` | ✅ |
| REFACTOR (`refactor(...)`) | — | N/A（gofmt 已并入 fix commit `0657cb3`） |

Gate 序列在 `git log --oneline` 中可验证：`test(24-02)` 在 `feat(24-02)` 之前，符合 plan-level TDD Gate Enforcement 要求。

注：fix commit `0657cb3` 没有 `refactor(...)` prefix 因为它是"test fixture 修正" 而非"实现重构"。

---

## 验证摘要

| 检查 | 命令 | 结果 |
|------|------|------|
| Build | `go build ./internal/recorder` | 退出 0 |
| Vet | `go vet ./internal/recorder` | 无告警 |
| Test (新) | `go test ./internal/recorder -run 'TestNewActivityWatcher\|TestExtendStepMin\|TestHuaweiEnabled\|TestIsActive\|TestOnReconnect\|TestSnapshot\|TestFakeClock' -count=1 -v` | PASS（9 断言）|
| Test (全) | `go test ./internal/recorder -count=1` | PASS（既有 silence_parser + coordinator + 新 7 项）|
| 行数 | `wc -l internal/recorder/activity_watcher.go` | 496 < 500 |
| FileMinGrowthBPS | `grep -c 'FileMinGrowthBPS' internal/recorder/activity_watcher.go` | 3 (DETECT-03 + 注释 + 1) |
| HuaweiEnabled() | `grep -c 'HuaweiEnabled()' internal/recorder/activity_watcher.go` | 1 |

---

## 关键文件清单

### 新增

- `D:/CODE/ClaudeCode/record_V2/internal/recorder/activity_watcher.go` — ActivityWatcher / ActivitySnapshot / fakeClock / NewActivityWatcher / Start / Stop / IsActive / Snapshot / EndedCh / ExtendStepMin / OnReconnect / HuaweiEnabled / silenceScanner / fileTicker / huaweiPoller / decisionTicker / closeEnded (496 行)
- `D:/CODE/ClaudeCode/record_V2/internal/recorder/activity_watcher_test.go` — TestNewActivityWatcher + TestExtendStepMin + TestHuaweiEnabled + TestIsActive + TestOnReconnect + TestSnapshot + TestFakeClock (195 行)

### 修改

- 无（本计划为单文件 + 配套单测；`coordinator.go` 的 ActivityWatcher 注入接线由 24-03 / 24-04 处理）

---

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_surface_extended:HuaweiClient | `internal/recorder/activity_watcher.go` | `huaweiPoller` 调 `HuaweiStateClient.GetConferenceState` 周期 `<=HuaweiPollIntervalS` (默认 30s);无新增网络/认证面 (interface 仅声明,实现仍在 `internal/huawei/client.go`) |
| threat_surface_extended:FS | `internal/recorder/activity_watcher.go` | `fileTicker` 周期 `os.Stat(filePath)`;无新增 FS 跨越 (filePath 来自 `task.MKVFilePath`,与既有 ffmpeg 输出同路径) |

新攻击面均已记录于 24-02-PLAN.md §threat_model T-24-02..T-24-06 + T-24-SC,所有已有 mitigation 沿用；24-02 实施未引入新 STRIDE 类别。

---

## Known Stubs

**有意保留**（按 plan 设计）：

- `huaweiPoller` H 路径的真实集成延迟到 24-04 (届时有 fake `HuaweiStateClient` 驱动)；24-02 仅保留 `HuaweiEnabled=true` 但 `huaweiCli=nil` 的"立即降级"安全路径。
- `silenceScanner` 的 bufio 流驱动延迟到 24-04 (届时配合 `tempfile + ffmpeg` 集成或 fake 写入)；24-02 单测仅断言 `silenceScanner` 在 `logFile=nil` 时不 panic。
- `fileTicker` 的真实 stat 跨越 1024 B/s 阈值延迟到 24-04。
- `decisionTicker` OR 触发的 fake clock 驱动延迟到 24-04。
- `closeEnded` 多 close 不 panic 验证延迟到 24-04。

这些 stub 符合 plan 字面要求（"24-04 will add the integration tests"）,不阻碍 24-02 阶段目标（单文件可编译 + 公开 API 契约正确）。

---

## Self-Check

- [x] `internal/recorder/activity_watcher.go` 存在（496 行）
- [x] `internal/recorder/activity_watcher_test.go` 存在（195 行）
- [x] `go build ./internal/recorder` 退出 0
- [x] `go vet ./internal/recorder` 无告警
- [x] `go test ./internal/recorder -count=1` PASS（7 项新单测 + 既有全测）
- [x] TDD gates: test(88b20f0) → feat(5ff35d9) → fix(0657cb3) 顺序正确
- [x] FileMinGrowthBPS 出现 3 次（cfg 字段读取 + 注释 + 1 逻辑）
- [x] HuaweiEnabled() 出现 1 次（公开 getter）
- [x] 496 < 500 lines
- [x] 0 新增外部依赖（threat model T-24-SC 满足）
- [x] 未触碰 `STATE.md` / `ROADMAP.md`（worktree mode 由 orchestrator 集中更新）
- [x] close-once 用 sync.Once (RESEARCH.md Pitfall 2 + plan 字面 action)
- [x] OnReconnect 仅清 silenceSince (WATCH-05 + 24-VALIDATION.md "重连保持")
- [x] huaweiPoller 入口直接 return on HuaweiEnabled=false (Open Question 2)

---

## 下游消费者（24-03 / 24-04 接线清单）

- **Plan 24-03 (WATCH-01 OR scheduler)** — `watcher.EndedCh()` + `watcher.IsActive()` + `watcher.ExtendStepMin()` 接入到 `internal/recorder/coordinator.go` 的调度循环
- **Plan 24-03 (WATCH-02 StatFailureThreshold)** — `fileTicker` 行为已 OK;24-03 验证集成层
- **Plan 24-03 (WATCH-03 DegradeOnSilenceLoss)** — `silenceScanner` 行为已 OK;24-03 按 `cfg.SmartEnd.DegradeOnSilenceLoss=false` 在决策侧短路
- **Plan 24-04 (WATCH-04 huawei 路径集成)** — 注入 fake `HuaweiStateClient`;启动时按 `watcher.HuaweiEnabled()` 决定注入与否
- **Plan 24-04 (集成测试)** — 用 `fakeClock` 驱动 `decisionTicker` 的 OR 触发;用 `tempfile + bufio.NewWriter` 驱动 `silenceScanner`;用 `os.Chtimes` 驱动 `fileTicker` 的 stat 跨越 1024 阈值

---

## Self-Check: PASSED

- [x] SUMMARY.md 存在 (257 lines)
- [x] 4 commits 全部存在 in git log (88b20f0, 5ff35d9, 0657cb3, 0535813)
- [x] STATE.md / ROADMAP.md 未被 diff (empty stat output)
- [x] 最终 git diff vs base: 仅 3 文件 +948 行 (activity_watcher.go / activity_watcher_test.go / 24-02-SUMMARY.md)
- [x] go build + go vet + go test 全 PASS

---

*Plan completed: 2026-08-06 — 4 commits (88b20f0, 5ff35d9, 0657cb3, 0535813) on `worktree-agent-ade23e375bac7d025`.*
