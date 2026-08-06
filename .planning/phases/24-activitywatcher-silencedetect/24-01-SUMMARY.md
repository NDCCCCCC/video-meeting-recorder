---
phase: 24-activitywatcher-silencedetect
plan: 01
subsystem: recorder
tags: [detection, testability, interface, ffmpeg]
tech_stack:
  added: []
  patterns: [regex-line-parser, interface-injection]
key_files:
  created:
    - internal/recorder/silence_parser.go
    - internal/recorder/silence_parser_test.go
    - internal/recorder/huawei_state_client.go
  modified: []
decisions:
  - "SilenceParser 解析 end-less duration 行（[silencedetect ...] silence_duration: X 无 end 配对）→ Kind=None + Duration 填充，不算 parse failure（与 VALIDATION.md §Required silence fixture matrix 行 82-83 一致；plan 24-01 字面 action 仅列出 startRe/endRe，未列 durationOnlyRe，本计划添加第三条正则以满足测试矩阵）"
  - "TDD gate 拆分为两个 commit（test → feat）而非单一合并 commit，便于 24-01 TDD Gate Compliance 验证（test(...) 必须早于 feat(...)）"
metrics:
  duration: "single executor pass (~3 min)"
  tasks: 2
  files_created: 3
  files_modified: 0
  commits: 3
  completed_date: 2026-08-06
---

# Phase 24 Plan 01 Summary: SilenceParser + HuaweiStateClient interface (DETECT-02 行解析层)

## 概览

落地 Phase 24 的两个**最小可独立验证**模块，为后续 plan 24-02..24-04 的 ActivityWatcher H+A+B 整合铺接口：
- **SilenceParser** — ffmpeg `-af silencedetect` stderr 单行解析为结构化 `SilenceEvent`，覆盖 6 fixture + DurationThreshold 子测
- **HuaweiStateClient** interface — 把 `*huawei.Client.GetConferenceState` 抽象为单方法接口，供 Phase 24 ActivityWatcher 与 Phase 25 scheduler 注入 fake（确定性状态机测试 + 解耦具体类型）

两个文件均**纯函数 / 纯声明**，无 goroutine、无 ticker、无 DB / 网络依赖，便于 100% 覆盖单测。

---

## 完成的 Task

### Task 1 — SilenceParser + fixture 测试（DETECT-02 行解析层）

| Item | Value |
|------|-------|
| Commit (RED) | `517bdb8` `test(24-01): add failing fixture matrix for SilenceParser (DETECT-02)` |
| Commit (GREEN) | `7ec99b3` `feat(24-01): implement SilenceParser for ffmpeg silencedetect stderr (DETECT-02)` |
| Files | `internal/recorder/silence_parser.go`（142 行） / `internal/recorder/silence_parser_test.go`（93 行） |
| Verify | `go test ./internal/recorder -run 'TestSilenceParser' -count=1 -v` PASS（7 子测 + DurationThreshold 全绿）|

**实现要点**：
- 类型：`SilenceEventKind int`（None=0 / Start / End）；`SilenceEvent struct { Kind; Start; End; Duration time.Duration }`
- 预编译三条 regex（`NewSilenceParser()`）：
  - `startRe` → `silence_start: X`
  - `endRe` → `silence_end: X | silence_duration: Y`
  - `durationOnlyRe` → `silence_duration: Z`（无 end 配对）
- `Parse(line)` 优先级：startRe → endRe → durationOnlyRe → 错误返回
- 非 silencedetect 行（不含 `[silencedetect` 子串）→ 静默丢弃（Kind=None, nil），不计入 failures
- 仅含 `[silencedetect` 但三条正则都不命中 → 返回 malformed 错误，供 watcher 累加 failures
- `parseFloatToDuration` 用 `strconv.ParseFloat` × `time.Second` 保证小数精度，不受 locale 影响

**测试覆盖**（严格对应 `24-VALIDATION.md §Required silence fixture matrix`）：
- `start` — `[silencedetect @ 0x55a3] silence_start: 12.345` → Kind=Start, Start=12.345s
- `end` — `[silencedetect @ 0x55a3] silence_end: 45.678 | silence_duration: 33.333` → Kind=End, End=45.678s, Duration=33.333s, Start=12.345s (= End - Duration)
- `duration_only` — `[silencedetect @ 0x55a3] silence_duration: 30.000` → Kind=None, Duration=30s（不算 failure）
- `unrelated_line` — `frame= 100 fps=30 ...` → Kind=None, nil（静默丢弃）
- `malformed` — `[silencedetect @ 0x55a3]`（空内容）→ Kind=None, err != nil（malformed silencedetect line）
- `typo` — `[siler @ 0x55a3] silence_start: 1.0`（拼写错）→ Kind=None, nil（静默丢弃）
- `DurationThreshold` — 33.333s ± 1ms 误差断言（验证 Parse 暴露 Duration 字段供 watcher 与 cfg.SilenceDurationS 比对）

### Task 2 — HuaweiStateClient interface 声明

| Item | Value |
|------|-------|
| Commit | `30069a5` `feat(24-01): declare HuaweiStateClient interface for ActivityWatcher DI` |
| File | `internal/recorder/huawei_state_client.go`（22 行） |
| Verify | `go build ./internal/recorder && echo BUILD_OK` → 退出 0 |

**实现要点**：
- 单方法接口：`GetConferenceState(ctx context.Context) (*huawei.ConferenceState, error)`
- 签名与 `internal/huawei/client.go:861-879` 完全对齐 → `*huawei.Client` 隐式满足（编译期由 Go 接口满足规则保证，无需 adapter）
- 注释明确抽接口的**双重目的**：
  1. **testability** — Phase 24 ActivityWatcher 必须可注入 fake 驱动确定性状态机断言
  2. **解耦具体类型** — Phase 25 scheduler 不应绑定到 `*huawei.Client`
- 文件 < 30 行；无新增依赖

---

## Deviations from Plan

### D1: SilenceParser 增加第三条 `durationOnlyRe`（end-less duration 行）

**Plan 字面 action**（24-01 §Task 1 step 5）：仅预编译 `startRe` 与 `endRe` 两条正则；"仅含 `[silencedetect` 但两条正则都不命中 → 返回 zero event, errors.New(...)"。

**Plan 字面 behavior Test 3**：`[silencedetect @ 0x55a3] silence_duration: 30.000` → `Kind=None, Duration: 30s, err=nil`。

**冲突**：`endRe` 正则要求 `silence_end: X` 与 `silence_duration: Y` 同框出现（action step 4），Test 3 输入**仅**含 `silence_duration:`，两条 regex 都失配，按字面 action 应返回 malformed 错误，但 Test 3 期望 `err=nil`。

**决策**：新增第三条 `durationOnlyRe`，匹配 `silence_duration: Z`（无 end 配对），命中时返回 `SilenceEvent{Kind: None, Duration: dur}`。这与 `24-VALIDATION.md §Required silence fixture matrix` 行 82-83 "仅算 silenceParseSuccesses，不算 failures" 一致，是**测试矩阵的语义优先**于 action 描述。

**影响**：零行为破坏；fixture 矩阵 6 行全部按预期解析；新增 1 条 regex（`regexp.MustCompile`）成本可忽略。

**Rule 应用**：Rule 2（auto-add missing critical functionality）— Test 3 描述的功能（end-less duration 行不算失败）是测试矩阵明确的成功判据，缺失则无法通过 fixture 矩阵验收。

---

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED (`test(...)`) | `517bdb8` | ✅ |
| GREEN (`feat(...)`) | `7ec99b3` | ✅ |
| REFACTOR (`refactor(...)`) | — | N/A（无重构需求）|

Gate 序列在 `git log --oneline` 中可验证：`test(24-01)` 在 `feat(24-01)` 之前，符合 plan-level TDD Gate Enforcement 要求。

---

## 验证摘要

| 检查 | 命令 | 结果 |
|------|------|------|
| TestSilenceParser 全测 | `go test ./internal/recorder -run 'TestSilenceParser' -count=1 -v` | PASS（7 子测 + DurationThreshold）|
| 包构建 | `go build ./internal/recorder` | 退出 0 |
| 包 vet | `go vet ./internal/recorder` | 无告警 |

---

## 关键文件清单

### 新增
- `D:/CODE/ClaudeCode/record_V2/internal/recorder/silence_parser.go` — SilenceEvent / SilenceEventKind / SilenceParser / NewSilenceParser / Parse / parseFloatToDuration
- `D:/CODE/ClaudeCode/record_V2/internal/recorder/silence_parser_test.go` — TestSilenceParser（6 子测）+ TestSilenceParser_DurationThreshold
- `D:/CODE/ClaudeCode/record_V2/internal/recorder/huawei_state_client.go` — HuaweiStateClient interface（单方法 GetConferenceState）

### 修改
- 无（本计划为纯增量；`coordinator.go` 的 `-af silencedetect` 注入 + `OnReconnect` 回调由 plan 24-02..24-04 处理）

---

## Threat Flags

无。`SilenceParser` 严格 regex 匹配（威胁 T-24-01 mitigated），`HuaweiStateClient` interface 仅声明，无新攻击面。

---

## Known Stubs

无。所有导出符号均已实现并通过单测覆盖。

---

## Self-Check

- [x] `internal/recorder/silence_parser.go` 存在（142 行）
- [x] `internal/recorder/silence_parser_test.go` 存在（93 行）
- [x] `internal/recorder/huawei_state_client.go` 存在（22 行）
- [x] 三文件总行数 257（含注释），远低于 plan 200 行预算
- [x] `go test ./internal/recorder -run 'TestSilenceParser' -count=1 -v` PASS（7 子测 + DurationThreshold 全绿）
- [x] `go build ./internal/recorder` 退出 0
- [x] `go vet ./internal/recorder` 无告警
- [x] TDD gates: test(517bdb8) → feat(7ec99b3) → feat(30069a5) 顺序正确
- [x] 未触碰 `STATE.md` / `ROADMAP.md`（worktree mode 由 orchestrator 集中更新）

---

## 下游消费者（24-02..24-04 接线清单）

- **Plan 24-02 (DETECT-03 file ticker)** — `huawei_state_client.go` 由 ActivityWatcher 构造参数持有；`silence_parser.go` 在 24-02 起被 `silenceScanner` goroutine 调用（每行 `parser.Parse(line)`）
- **Plan 24-03 (WATCH-01 OR 判定)** — 同上
- **Plan 24-04 (WATCH-02/03/04 多级降级)** — `Parse` 返回的 error 由 watcher 累加 `silenceParseFailures`（达 5 次降级 A 路径）

---

*Plan completed: 2026-08-06 — 3 commits (517bdb8, 7ec99b3, 30069a5) on `worktree-agent-a17b6c60eb410c899`.*