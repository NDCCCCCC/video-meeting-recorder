# Requirements: Record V2 — v2.0 智能录制收尾 (Smart Recording End)

**Defined:** 2026-08-06
**Milestone:** v2.0 — 智能录制收尾
**Core Value:** 会议视频从录制到PPT的一站式处理，让会议内容可检索、可回顾、可分享
**PRD Source:** `docs/plans/2026-08-05-smart-meeting-recording-end-design.md` (v2: 纳入 TE40 邮箱 API 主信号)

> 让华为会议录制时长智能贴合会议真实时长——到点未结束自动延时（30min × 4 = 2h 上限），提前结束由 TE40 `WEB_GetMailboxDataAPI`（`confState=="" && joinSum==0`）主信号 + silencedetect + 文件停滞双兜底任一触发即收尾转码，无需人工干预。

## v2.0 Requirements

### SEC-003a — 华为终端 TLS 私有 CA 加载 (hotfix, Phase 26)

> 来源：debug session `.planning/debug/huawei-tls-private-ca.md`（root-cause-confirmed, 2026-08-06）。华为终端 `https://10.62.10.3` 的服务器证书由私有自签 CA `huawei_ca` 颁发，Go `crypto/tls` 默认使用系统 CA bundle，无法信任。本组为 v2.0 三 phase（23/24/25）的前置硬阻塞——`HuaweiClient.GetConferenceState()` 与 `HuaweiConferenceConnector.ConnectToConference` 在此修复落地前完全无法工作。

- [ ] **SEC-003a-01**: 系统在 `HuaweiConfig.CABundleFile` 非空时从指定 PEM 文件读取所有证书（含 server cert + `huawei_ca` 自签根），注入 `tls.Config.RootCAs` 作为信任锚；`CABundleFile` 为空字符串时维持原行为（系统 CA bundle）
- [ ] **SEC-003a-02**: `HuaweiClientManager.SetCABundle(path string) error` 在 PEM 文件不存在 / 解析失败时返回 wrapped error（含文件路径与底层 cause），调用方负责 fail-closed 处理
- [ ] **SEC-003a-03**: `cmd/server` 启动时调用 `huaweiMgr.SetCABundle(cfg.Huawei.CABundleFile)`，错误时 `logger.Fatal` 退出；空字符串则跳过加载并记录 INFO 日志
- [ ] **SEC-003a-04**: `config.yaml` 默认值 `./certs/huawei-10.62.10.3-ca.pem`（相对 working dir），运维可通过 `config.yaml` / `HUAWEI_CA_BUNDLE_FILE` 环境变量在不重新编译二进制的条件下覆盖路径（服务进程重启生效即可）；PEM 文件未随仓库 tracked（`.gitignore` 已忽略），部署时由运维手动同步
- [ ] **SEC-003a-05**: 新增 unit 子测覆盖 5 场景：① 正常 PEM + httptest 自签 server 握手成功；② PEM 损坏 / 不存在 → SetCABundle 返回 wrapped error；③ 空路径 → RootCAs=nil（系统 CA 默认）；④ `ca_bundle_file` 同时含 server cert + 自签根时 chain verify 成功；⑤ `NewHTTPClient` 签名变化后 `caCertPool==nil` 与非 nil 两条分支均覆盖

### DETECT — 检测信号采集

- [ ] **DETECT-01**: 系统可在 `HuaweiClient.GetConferenceState()` 中识别 `confState=="" && joinSum==0` 持续 ≥ `huawei_persist_s`（默认 30s）作为会议已结束的业务级权威信号
- [ ] **DETECT-02**: 系统可通过 ffmpeg `-af silencedetect=noise=-30dB:d=30` 过滤器检测 ≥ `silence_duration_s`（默认 30s）持续静音
- [ ] **DETECT-03**: 系统可通过 ticker（`check_interval_s`，默认 5s）采样输出文件大小，识别 ≥ `file_stall_s`（默认 120s）无增长（rolling window，`file_min_growth_bps` 阈值，默认 1024 B/s）
- [ ] **DETECT-04**: 系统可在华为 API 字段缺失（老设备）时 fallback 到现有 `IsInConf==0` 判据（`huawei/client.go:212` 现有字段），保证 H 信号可用性

### WATCH — ActivityWatcher 整合与降级

- [ ] **WATCH-01**: 系统提供 `ActivityWatcher` 整合 H + A + B 三类信号，按 OR 关系判定（任一触发即发 `EventMeetingEnded`），H 需持续 `huawei_persist_s`，A + B 是 AND 关系
- [ ] **WATCH-02**: 系统可在 silencedetect stderr 解析连续 5 次无有效行时，自动降级关闭 silence 分支，仅用文件停滞 + 华为信号判定（`degrade_on_silence_loss=true`）
- [ ] **WATCH-03**: 系统可在 `HuaweiClient.GetConferenceState()` 连续失败 ≥ `huawei_failure_threshold`（默认 3）次时自动降级关闭 H 信号，只用 A + B
- [ ] **WATCH-04**: 系统可在文件采样连续 ≥ `stat_failure_threshold`（默认 3）次 `os.Stat` 失败时，视为流死亡，发 `EventMeetingEnded` reason=`file_stat_failed`
- [ ] **WATCH-05**: 系统可在 ffmpeg 重连时（`attemptReconnect`）保持 watcher 静音/文件计时不重置（仅清 `SilenceSince`，避免断流期间假静音），保证与现有断流重连逻辑不冲突

### SCHED — scheduler 多信号驱动

- [ ] **SCHED-01**: `scheduler.monitorTask` 改为 `select` on EndTime timer + `taskEndedCh` + `taskUpdateChans[task.ID]`，单次循环同时等待三类信号
- [ ] **SCHED-02**: EndTime 到点时，系统先问 `watcher.IsActive()`；活跃（任一信号未触发）则 `EndTime += extend_step_min`，否则 `completeTask("endtime_no_activity")`
- [ ] **SCHED-03**: `taskEndedCh` 信号永远优先于 EndTime timer —— channel close 后 EndTime.C 不再生效，提前结束信号不受 timer 竞争
- [ ] **SCHED-04**: 用户手动 `UpdateTaskEndTime` 触发 `taskUpdateChans` 时，`ExtensionCount` 不重置（仅 timer 重置），避免人为绕开上限

### EXTEND — 自动延时与上限

- [ ] **EXTEND-01**: 单任务 `ExtensionCount` 上限 = `max_extend_count`（默认 4），累计延时 = 4 × `extend_step_min`（默认 30min）= 2h 总上限
- [ ] **EXTEND-02**: 上限达 4 次后 EndTime 到点仍活跃 → 强制 `completeTask("max_extend_reached")`，任务状态保留 `completed`（**不**标 `failed`，便于运维不告警），audit log 写 `warn`
- [ ] **EXTEND-03**: 默认 `extend_step_min=30`（可配，运维可调到 60min 等大值减少延时抖动）

### EARLY — 提前结束触发

- [ ] **EARLY-01**: H 信号触发时 → `completeTask("smart_early_end")`，`task.EndedByHuaWeAPI=true`，`task.EndedEarlyReason="huawei_state_empty"`
- [ ] **EARLY-02**: A + B 双 AND 命中（silence ≥ 30s AND file_stall ≥ 120s）→ `completeTask("smart_early_end")`，`task.EndedByHuaWeAPI=false`，`task.EndedEarlyReason="both_silence_and_stall"`
- [ ] **EARLY-03**: 提前结束信号永远优先于 EndTime timer（`select` 内 case 顺序保证；channel close 后 EndTime.C 不再生效）
- [ ] **EARLY-04**: 多 input 任务（huawei_auto + usb 同时录制），任一 watcher 判定结束 → 整体结束（保守策略，避免部分 ffmpeg 仍在写但任务已收尾）

### AUDIT — GORM 字段 + audit log

- [ ] **AUDIT-01**: `video_recording_tasks` 表加字段（AutoMigrate 列表同步，不进 dormant `runCustomMigrations`）：`ExtensionCount int` / `LastExtensionReason string` / `EndedEarly bool` / `EndedEarlyReason string` / `EndedByHuaWeAPI bool`
- [ ] **AUDIT-02**: 每次"延时"事件写 audit log，含 snapshot：`silence_since` / `last_file_growth` / `file_size_bytes` / `file_growth_bps` + `extension_count` + `new_end_time`
- [ ] **AUDIT-03**: 每次"提前结束"事件写 audit log，含 `task_id` + `reason` + `ended_by_huawei_api` + `ended_early_reason` + final snapshot
- [ ] **AUDIT-04**: service 层封装 `UpdateTaskExtension(task, deltaMin, reason)` + `MarkTaskEndedEarly(task, reason, byHuaWeiAPI bool)`，统一写入 GORM 字段 + audit log（避免散落多个调用点）
- [ ] **AUDIT-05**: `internal/errors/` 加 `ErrRecordingSmartExtend` / `ErrRecordingSmartEarlyEnd` / `ErrRecordingHuaWeiStateFetchFailed` 三个 sentinel，同步更新 `docs/errors.md`（CI sync-check 门禁，`.github/workflows/ci.yml:44-51`）

### CFG — 配置项与开关

- [ ] **CFG-01**: `internal/config/smart_end.go` 新增 `SmartEndConfig` 结构体（含 14 项阈值/开关，PRD §6 完整列表），从 `config.yaml` 加载，提供合理默认值
- [ ] **CFG-02**: `config.yaml` 新增 `smart_end:` 段，含 14 项配置（`enabled / silence_db / silence_duration_s / file_stall_s / file_min_growth_bps / huawei_enabled / huawei_poll_interval_s / huawei_persist_s / huawei_failure_threshold / check_interval_s / extend_step_min / max_extend_count / stat_failure_threshold / degrade_on_silence_loss`）
- [ ] **CFG-03**: `smart_end.enabled=false` 时系统退回纯 EndTime 行为（scheduler 不读 `taskEndedCh`，watcher 不启），便于运维临时回退
- [ ] **CFG-04**: `smart_end.huawei_enabled=false` 时系统降级只用兜底 A + B（华为轮询 goroutine 不启），便于 TE40 设备下线/维护时回退

### OBS — 可观测性（日志）

- [ ] **OBS-01**: 系统输出 `INFO smart_extend task=<id> count=<n> new_end=<ts> reason=<text>` 日志（每次自动延时）
- [ ] **OBS-02**: 系统输出 `INFO smart_early_end task=<id> reason=<text> snapshot=<json>` 日志（每次提前结束）
- [ ] **OBS-03**: 系统输出 `WARN max_extend_reached task=<id> force_end=true` 日志（强制截断）
- [ ] **OBS-04**: 系统输出 `ERROR activity_watcher_degraded reason=<text>` 日志（watcher 降级事件）
- [ ] **OBS-05**: 可选 Prometheus counter（`record_v2_smart_extend_total` / `record_v2_smart_early_end_total` / `record_v2_watcher_degraded_total`）；项目当前无 prometheus 集成则仅做日志，预留 counter 接入点

---

## Future Requirements（本期不做，下个里程碑候选）

> 来源：PRD §13 后续 Phase 候选（本期明确不做）。

- **FUTURE-01**: 系统支持 `MSG_CONF_STATE_CHANGE` 推送接入，替代 30s 轮询，判定延迟降至 1-2s（需项目具备消息推送基础设施）
- **FUTURE-02**: 系统支持 TE40 T.140 字幕信号（`isSupportT140=1 && isOpenT140=1` 时作为辅助活动信号，避免纯静音会议被 A+B 误杀）
- **FUTURE-03**: 系统支持跨 input 一致性（多 input 任务下，若部分 input 已结束但其他仍活跃，给出"软结束"信号，仅停部分 ffmpeg，不 completeTask）
- **FUTURE-04**: 前端任务详情页显示 `ExtensionCount` / `EndedEarlyReason` / `EndedByHuaWeAPI`，让用户直观看到智能延时的发生（需前端 work，本期仅后端）
- **FUTURE-05**: 系统支持机器学习预测（用历史会议数据训练"提前结束"概率模型，超阈值提前结束）

---

## Out of Scope（明确不做）

> 来源：PRD §2.2 Non-Goals。

- 不改 `VideoRecordingTaskStatus` 枚举（沿用 `completed / failed / canceled`，不新增 `smart_extended` 等）
- 不做"会议提前结束预测"（无 ML/ASR；不做 FUTURE-05 范围内的概率模型）
- 不重写 `monitorProcessWithKey` 现有的断流重连逻辑（保留 `attemptReconnect`，仅在 watcher 侧处理重连期间状态，见 WATCH-05）
- 不动前端 UI（本期仅后端 + config；FUTURE-04 单独做）
- 不接 `MSG_CONF_STATE_CHANGE` 推送通道（FUTURE-01 范围内）
- 不做多语言/i18n（业务未要求）
- 不重写 scheduler 框架（沿用 `github.com/robfig/cron/v3` + `time.NewTimer`/`time.NewTicker` 模式）

---

## Traceability

> Phase-to-requirement mapping filled by `/gsd:roadmap` (2026-08-06, v2.0 roadmap created).

| REQ-ID | Phase | Status | Notes |
|--------|-------|--------|-------|
| DETECT-01 | Phase 23 | Pending | 华为 GetConferenceState confState=="" && joinSum==0 持续 30s 检测 |
| DETECT-02 | Phase 24 | Pending | ffmpeg silencedetect 过滤器 ≥ 30s 静音 |
| DETECT-03 | Phase 24 | Pending | ticker 5s 采样文件大小 ≥ 120s 无增长 |
| DETECT-04 | Phase 23 | Pending | 老设备 fallback 到 IsInConf==0 判据 |
| WATCH-01 | Phase 24 | Pending | ActivityWatcher 整合 H+A+B OR 判定 + EventMeetingEnded |
| WATCH-02 | Phase 24 | Pending | silencedetect 解析失败自动降级 |
| WATCH-03 | Phase 24 | Pending | 华为 API 连续失败 3 次降级关闭 H |
| WATCH-04 | Phase 24 | Pending | os.Stat 连续失败视为流死亡 |
| WATCH-05 | Phase 24 | Pending | ffmpeg 重连保持 watcher 计时不重置 |
| SCHED-01 | Phase 25 | Pending | monitorTask 改 select 驱动 |
| SCHED-02 | Phase 25 | Pending | EndTime 到点先问 watcher.IsActive() |
| SCHED-03 | Phase 25 | Pending | taskEndedCh 永远优先于 EndTime timer |
| SCHED-04 | Phase 25 | Pending | 用户手动 UpdateTaskEndTime 时 ExtensionCount 不重置 |
| EXTEND-01 | Phase 25 | Pending | ExtensionCount 上限 = 4 (2h 总上限) |
| EXTEND-02 | Phase 25 | Pending | 上限达 4 次后强制 completeTask("max_extend_reached") |
| EXTEND-03 | Phase 24 | Pending | 默认 extend_step_min=30（可配） |
| EARLY-01 | Phase 25 | Pending | H 信号触发 → smart_early_end + EndedByHuaWeAPI=true |
| EARLY-02 | Phase 25 | Pending | A+B 双 AND 命中 → smart_early_end |
| EARLY-03 | Phase 25 | Pending | 提前结束信号永远优先于 EndTime timer |
| EARLY-04 | Phase 25 | Pending | 多 input 任一 watcher 触发整体结束 |
| AUDIT-01 | Phase 23 | Pending | GORM 加 5 字段（AutoMigrate 列表同步） |
| AUDIT-02 | Phase 25 | Pending | 延时事件 audit log 含 snapshot |
| AUDIT-03 | Phase 25 | Pending | 提前结束事件 audit log 含 snapshot |
| AUDIT-04 | Phase 25 | Pending | service 层封装 UpdateTaskExtension + MarkTaskEndedEarly |
| AUDIT-05 | Phase 23 | Pending | 3 个 sentinel + docs/errors.md 同步 + CI sync-check |
| CFG-01 | Phase 23 | Pending | SmartEndConfig 结构体 + 默认值 |
| CFG-02 | Phase 23 | Pending | config.yaml smart_end 段（14 项） |
| CFG-03 | Phase 25 | Pending | smart_end.enabled=false 退回纯 EndTime |
| CFG-04 | Phase 25 | Pending | smart_end.huawei_enabled=false 只用 A+B |
| OBS-01 | Phase 25 | Pending | smart_extend 日志 |
| OBS-02 | Phase 25 | Pending | smart_early_end 日志 |
| OBS-03 | Phase 25 | Pending | max_extend_reached 日志 |
| OBS-04 | Phase 25 | Pending | activity_watcher_degraded 日志 |
| OBS-05 | Phase 25 | Pending | 可选 Prometheus counter（无 prometheus 集成则仅做日志） |

**Coverage:** 34/34 (100%) — 0 orphan

- **Phase 23**: 6 reqs (DETECT-01/04, AUDIT-01/05, CFG-01/02)
- **Phase 24**: 8 reqs (DETECT-02/03, WATCH-01..05, EXTEND-03)
- **Phase 25**: 20 reqs (SCHED-01..04, EXTEND-01/02, EARLY-01..04, AUDIT-02/03/04, CFG-03/04, OBS-01..05)

---

## Cross-Reference

- **PRD:** `docs/plans/2026-08-05-smart-meeting-recording-end-design.md` (525 lines, v2 2026-08-05)
- **PRD 实施步骤 §9**: 11 步任务，按 PRD §9 建议拆 3 phase（A: 华为 API + 字段 + 错误码 / B: watcher + 兜底 + 录制层 / C: scheduler + service + E2E + CI）
- **项目内已有基础设施**: `internal/huawei/client.go:702-744` `HuaweiClient.GetMailboxData()` 仅需扩展 8 个 struct 字段；零新增依赖
- **关联技术约束**: `commit-boundary-separation.md`（4 个 commit 拆分）/ `migration-automigrate-convention.md`（AutoMigrate 列表同步）/ `.planning/gitignored.md`（`git add -f`）/ `docs/ci-maintenance.md`（errors.md sync-check）