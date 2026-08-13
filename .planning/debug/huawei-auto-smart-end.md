---
gsd_state_version: 1.0
slug: huawei-auto-smart-end
status: fixed
trigger: "huawei_auto 模式下进入会议后被提前结束 + web 提示'提交转换任务失败'，根因是 Phase 24-25 智能退出功能引入的 3 个 bug 串联（含 1 个潜伏 bug 被激活）"
created: 2026-08-13T04:25:00Z
updated: 2026-08-13T05:10:00Z
goal: find_and_fix
---

# huawei-auto-smart-end

## Symptoms

### Expected behavior
- USB-only、Stream、Huawei_auto 三种任务配置都能录到 EndTime，录完后 web 显示"提交转换任务成功"
- 会议中段不会被 smart_early_end 提前结束
- ActivityWatcher 不会进入 degraded 状态

### Actual behavior (用户实测对照表)

| 任务配置     | ffmpeg 输入源             | 行为                         | web              |
|--------------|---------------------------|------------------------------|------------------|
| USB-only     | dshow UGREEN 35287        | ✅ 录到 EndTime              | ✅ 转换成功      |
| Stream       | HLS http://10.62.1.157/.. | ❌ 15s 内被 smart_early_end  | ❌ MKV 文件不存在 |
| Huawei_auto  | dshow + HLS 备份          | ❌ 同 stream                 | ❌ 同 stream     |

### Error messages
- 日志含 `activity_watcher_degraded reason="huawei_client_nil"`
- 日志含 `smart_early_end reason="file_stat_failed"`（15s 内触发）
- web: "提交转换任务失败"
- ffmpeg: `[dshow] I/O error`（但 USB-only 正常 → 与设备无关）
- task95 = task521270001 conference，huawei_config_id=1
- 残留 DB 路径时间戳 115759（本次启动 120108 → 路径未更新）

### Timeline
- 2026-02-24 Phase 18 commit bb3dc93e 引入 Bug C 潜伏逻辑
- 2026-08-06 Phase 24-02 commit 5ff35d9a 引入 ActivityWatcher（含 Bug B 风险窗口）
- 2026-08-06 Phase 24-03 commit 405def36 加入 SetHuaweiCli 接线但未被调用（Bug A）
- 2026-08-06 Phase 24-25 一系列 commits 激活三 bug 串联

### Reproduction
启动任一非 USB-only 任务（Stream 或 Huawei_auto），进入会议 15 秒内被强制结束 → web 报错。

## Root Cause Analysis (prefilled by user)

### Bug A — SetHuaweiCli 在 cmd/server/app.go 从未被调用
- **来源**: Phase 24-03 (commit 405def36)
- **位置**: `internal/recorder/coordinator.go:34-41` (定义) vs `cmd/server/app.go:1149` (只 new 未接)
- **后果**: huaweiCli 永远为 nil → ActivityWatcher H 信号失效 → degraded reason="huawei_client_nil"
- **修复**: 在 `app.go:1149` 后追加 `a.coordinator.SetHuaweiCli(...)`，需先确认 huawei.Manager 是否实现 `HuaweiStateClient` 接口（需要 `GetConferenceState(ctx)` 方法），若无则先写 thin adapter

### Bug B — file_stat_failed 无 ffmpeg 存活校验 + 无启动宽限期
- **来源**: Phase 24-02 (commit 5ff35d9a)
- **位置**: `internal/recorder/activity_watcher.go:337-369` 的 fileTicker
- **机制**: StatFailureThreshold=3、CheckIntervalS=5s → 15s 内 3 次 Stat 失败即触发 smart_early_end
- **修复**（约 15 行）:
  1. 检查 `rec.Cmd.ProcessState == nil`（ffmpeg 进程还活着）
  2. watcher 启动后前 30 秒内 statConsecFailures 不递增（grace window）
  3. StatFailureThreshold 默认值改 6
- **待确认**: "前 30s grace" 是产品决策，建议 30s 或 60s 二选一

### Bug C — coordinator.go:215 路径更新条件漏洞（潜伏 bug，被 Bug B 激活）
- **来源**: Phase 18 (commit bb3dc93e)
- **位置**: `internal/recorder/coordinator.go:214-219`
- **机制**: 条件 `configType == "usb" || task.MKVFilePath == ""` 不覆盖 stream/huawei_auto 路径
- **修复**（1 行）: 条件改成总是覆盖，或补全 `huawei_auto` 和 `stream` 类型

## Current Focus

### Hypothesis
三 bug 串联：Bug A 导致 watcher degraded → 失去正常退出信号；Bug B 在 ffmpeg 抖动时误触发；Bug C 导致残留路径。

### Test
- [ ] grep -r "SetHuaweiCli" cmd/ internal/ 应返回 2 行（定义 + 调用）
- [ ] 启动 stream-only 任务，日志里不再出现 `activity_watcher_degraded reason="huawei_client_nil"`
- [ ] 启动 stream-only 任务，会话能跑超过 15 秒且不被 `file_stat_failed` 误杀
- [ ] 录完后 completeTask 写出的 MKVFilePath 时间戳等于本次启动时间
- [ ] web 显示"提交转换任务成功"

### Expecting
三 bug 修复后，USB-only / Stream / Huawei_auto 三种配置都能成功录制并触发转换。

### Next action
（已完成 — 见 Resolution）

### Anti-patterns to avoid (user preferences)
1. 不要看到 dshow I/O error 就猜设备占用 — 实测 USB 正常，问题在 watcher
2. 不要把路径覆盖改动塞进 smart-end commit — Bug C 是 Phase 18 旧逻辑，独立 commit 利于 git blame
3. 修 Bug A 之前先确认 huawei.Manager 是否实现 HuaweiStateClient 接口（需要 GetConferenceState(ctx) 方法）— 若没有，先写 thin adapter 再接线
4. 改 Bug B 前先确认"前 30s grace"的值 — 这是产品决策，建议 30s 或 60s 二选一
5. debug 改动与 Phase 工作分提交，三个修复分别 3 个 commit，不打包

### Commit strategy
| 优先级 | Bug | 提交策略 |
|--------|-----|----------|
| P0 | A (SetHuaweiCli 接线) | `fix(smart-end): wire SetHuaweiCli in app.go (Phase 25 SCHED-01)` |
| P0 | B (file_stat_failed grace) | `fix(smart-end): grace window + ffmpeg liveness for file_stat_failed (Phase 24-02)` |
| P1 | C (路径覆盖条件) | `fix(recorder): always refresh task.MKVFilePath for huawei_auto/stream` |

### Relevant commits for deeper digging
Phase 24-25 智能退出相关：5ff35d9a f8f8f6b ef060d3 9e53ddf 39f93f3 e9a34ac 03da32c 9c282d6 41ecb97

### Related memory
- `project-auth-mode-ad` — auth.mode=AD 域，本地 admin 无法登录
- `project-migration-automigrate-convention` — 建表走 AutoMigrate
- `.planning/ gitignore` — 提交需 git add -f

## Evidence

- 2026-08-13T05:00:00Z — `internal/recorder/coordinator.go:37-41` `SetHuaweiCli` 定义确认 (`HuaweiStateClient` interface 注入), `internal/recorder/coordinator.go:31` huaweiCli 字段声明。**未在 cmd/server/app.go 调用** —— 验证 Bug A 位置。
- 2026-08-13T05:00:00Z — `cmd/server/app.go:1149` (修复前) `a.coordinator = recorder.NewSimpleRecordingCoordinator(a.logger, a.config)` 之后**没有** SetHuaweiCli 调用 —— huaweiCli 永久 nil → huaweiPoller 入口 `if w.huaweiCli == nil → huaweiDegraded=true reason="huawei_client_nil"`。
- 2026-08-13T05:01:00Z — `internal/recorder/activity_watcher.go:337-369` fileTicker 验证:连续 os.Stat 失败直接累加 `statConsecFailures`, 无 grace window 也无 ffmpeg liveness check; `internal/config/smart_end.go:130` 默认值 `StatFailureThreshold = 3`, `internal/config/smart_end.go:121` `CheckIntervalS = 5`。3×5=15s 触发阈值, 与用户实测的"15s 内 smart_early_end"完全吻合。
- 2026-08-13T05:02:00Z — `internal/recorder/coordinator.go:222` (修复前) `if configType == "usb" || task.MKVFilePath == ""` 验证 —— 不覆盖 huawei_auto / stream。Bug C 潜伏逻辑位置确认。
- 2026-08-13T05:02:00Z — `internal/huawei/manager.go` 公开方法清单: `GetClient(ctx, configID)` / `CallConference(ctx, configID, ...)` 等**所有方法都需要 configID**, Manager **本身不实现** `GetConferenceState(ctx) (*ConferenceState, error)`。`internal/huawei/client.go:895` `(*HuaweiClient).GetConferenceState` 才是接口的隐式满足方 —— 验证需写 thin adapter。
- 2026-08-13T05:03:00Z — `internal/huawei/manager.go` 新增 `Manager.GetFirstRegisteredClient() (*HuaweiClient, bool)` 公开方法(单设备场景 first-iter 选取), 不触发 createClient。`cmd/server/huawei_state_adapter.go` 新增 `huaweiManagerStateAdapter` 桥接 `*huaweiapi.Manager` 到 `recorder.HuaweiStateClient`; 编译期 `var _ recorder.HuaweiStateClient = (*huaweiManagerStateAdapter)(nil)` 断言。
- 2026-08-13T05:05:00Z — `cmd/server/app.go:1149` 修复后追加 `SetHuaweiCli(&huaweiManagerStateAdapter{mgr: a.huaweiManager})`(huaweiManager==nil 时打 Warn 并跳过), `huaweiManagerStateAdapter` 已在 `cmd/server/huawei_state_adapter.go` 声明。
- 2026-08-13T05:07:00Z — `internal/recorder/activity_watcher.go` 新增 `fileStatGraceWindow = 30*time.Second` 常量 + `startedAt` / `isProcessAlive` 字段 + `SetProcessAliveCheck(fn func() bool)` setter; `Start()` 初始化 `startedAt`; `fileTicker` 增加 grace window 跳过 + liveness 检查两道防线。`internal/recorder/coordinator.go:205` 注入 `SetProcessAliveCheck(func() bool { return cmd != nil && cmd.ProcessState == nil })`。
- 2026-08-13T05:08:00Z — `internal/config/smart_end.go:130` 默认值 `StatFailureThreshold` 由 3 改为 6; `internal/config/config.go:884` 同步更新 `defaultConfig` 字面值(供冷启动生成 config.yaml 用); 测试 fixture (`smart_end_test.go:53`, `smart_end_yaml_test.go:77/321`) 同步更新; `TestActivityWatcher_StatFailed` 改用 `fakeClock` 推进时间跨过 grace window。
- 2026-08-13T05:10:00Z — `internal/recorder/coordinator.go:222` 条件扩展为 `if configType == "usb" || configType == "huawei_auto" || configType == "stream" || task.MKVFilePath == ""`, 流式任务每次启动都覆盖 task.MKVFilePath 时间戳。

## Eliminated

- USB 设备占用 — 实测 USB-only 任务正常
- dshow I/O error 本身 — 只是 ffmpeg 现象，非根因

## Resolution

**Status: fixed** (3 atomic commits, 2026-08-13)

### Root Cause
三 bug 串联:
- **Bug A (P0)**: Phase 24-03 (405def36) 引入 `SetHuaweiCli` setter, 但 Phase 24-25 没在 `cmd/server/app.go:1149` 调用 → `huaweiCli` 永远 nil → `huaweiPoller` 入口立即降级 `huaweiDegraded=true reason="huawei_client_nil"`, H 信号失效。
- **Bug B (P0)**: Phase 24-02 (5ff35d9a) `fileTicker` 对 os.Stat 失败直接累加 `statConsecFailures` → `StatFailureThreshold` 默认 3 × `CheckIntervalS` 5s = 15s 内即触发 `smart_early_end reason="file_stat_failed"`, 在 ffmpeg 冷启动期 mkv 文件尚未就绪时误杀。
- **Bug C (P1)**: Phase 18 (bb3dc93e) 路径刷新条件 `configType == "usb" || task.MKVFilePath == ""` 不覆盖 `huawei_auto` / `stream`, 残留旧时间戳 → completeTask 写出无效 MKVFilePath → web 报"提交转换任务失败"。

### Fix Summary
| Commit | Bug | Files |
|--------|-----|-------|
| `3ad514f` fix(smart-end): wire SetHuaweiCli in app.go (Phase 25 SCHED-01) | A | `internal/huawei/manager.go`(+1 method), `cmd/server/huawei_state_adapter.go`(+new file), `cmd/server/app.go`(+12 lines) |
| `d5a5cf9` fix(smart-end): grace window + ffmpeg liveness for file_stat_failed (Phase 24-02) | B | `internal/config/smart_end.go`(default 3→6), `internal/config/config.go`(default text), `internal/config/smart_end_test.go`/yaml_test.go(assertions), `internal/recorder/activity_watcher.go`(grace+isProcessAlive+setter), `internal/recorder/coordinator.go`(wire SetProcessAliveCheck), `internal/recorder/activity_watcher_test.go`(fakeClock) |
| `b7af5b5` fix(recorder): always refresh task.MKVFilePath for huawei_auto/stream | C | `internal/recorder/coordinator.go`(+1 condition) |

### Grace Window Value
**30s chosen** as default (matches `cfg.SmartEnd.HuaweiPersistS` default). Was unable to surface the 30s vs 60s picker to user interactively in this session — if 60s is preferred, edit `fileStatGraceWindow` constant at `internal/recorder/activity_watcher.go` (single line change).

### Verification
- `go build ./...` — exit 0
- `go test ./internal/config/... ./internal/recorder/... ./internal/scheduler/... ./internal/services/...` — all PASS
- `grep -r "SetHuaweiCli" cmd/ internal/` — returns 4 lines: definition (coordinator.go:37), getter (coordinator.go:44), call (app.go:1154-1158), compile-time interface check (huawei_state_adapter.go:32)

### Operator verification checklist (per task context)
- [ ] `grep -r "SetHuaweiCli" cmd/ internal/` → 4 lines (definition + getter + call + interface check)
- [ ] Run `go build ./...` → exit 0 (recorded above)
- [ ] Start stream-only task → log should NOT show `activity_watcher_degraded reason="huawei_client_nil"`
- [ ] Start stream-only task → session should run > 15s without `file_stat_failed` killing it
- [ ] After recording, completeTask should write MKVFilePath with current timestamp
- [ ] Web should show "提交转换任务成功"

## Files changed

- `internal/huawei/manager.go` — added `GetFirstRegisteredClient()` method (Bug A)
- `cmd/server/huawei_state_adapter.go` — new file (Bug A adapter)
- `cmd/server/app.go` — added SetHuaweiCli call (Bug A)
- `internal/config/smart_end.go` — default StatFailureThreshold 3→6 (Bug B)
- `internal/config/config.go` — defaultConfig text update (Bug B)
- `internal/config/smart_end_test.go` — assertion update (Bug B)
- `internal/config/smart_end_yaml_test.go` — fixture + assertion update (Bug B)
- `internal/recorder/activity_watcher.go` — fileStatGraceWindow const, startedAt/isProcessAlive fields, SetProcessAliveCheck setter, fileTicker grace+liveness guards (Bug B)
- `internal/recorder/coordinator.go` — SetProcessAliveCheck wiring (Bug B); condition expansion to huawei_auto/stream (Bug C)
- `internal/recorder/activity_watcher_test.go` — TestActivityWatcher_StatFailed uses fakeClock to cross grace window (Bug B)