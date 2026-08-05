# Milestone v2.0 — 智能录制收尾 (Smart Recording End)

> **来源**：`/gsd:new-milestone v2.0 --prd docs/plans/2026-08-05-smart-meeting-recording-end-design.md`
> **状态**：已收齐（2026-08-06），下游 `/gsd:new-milestone` 应消费此文件并跳过 §2 内联提问。

---

## Goal

让华为会议录制时长智能贴合会议真实时长——到点未结束自动延时（30min × 4 = 2h 上限），提前结束由 TE40 `WEB_GetMailboxDataAPI`（`confState=="" && joinSum==0`）主信号 + silencedetect + 文件停滞双兜底任一触发即收尾转码，无需人工干预。

## Target features

1. **主信号 H（业务权威）**：在 `recorder/coordinator.go` 启 `HuaweiStatePoller goroutine`，每 30s 调 `HuaweiClient.GetConferenceState()`（底层调 `WEB_GetMailboxDataAPI`），判据 `confState=="" && joinSum==0` 持续 `huawei_persist_s`（默认 30s）→ 视为会议已结束。
2. **兜底信号 A + B（双 AND）**：在 `recorder/coordinator.go` 给 ffmpeg 加 `silencedetect` 过滤器 + 输出文件大小采样（ticker 5s）。silencedetect 持续 ≥ 30s **且** 文件大小停滞 ≥ 120s → 也判定为结束（华为 API 不可达时仍可工作）。
3. **scheduler 多信号驱动**：`monitorTask` 改为 `select` on EndTime timer + `taskEndedCh` + `taskUpdateChans`。EndTime 到点先问 watcher 决定延时/结束；若到点但任一判定信号活跃 → `EndTime += 30min`，累计 ≤ 4 次（2h 上限）。
4. **可降级**：silencedetect 解析失败 → 切纯文件停滞 + 华为信号；`HuaweiClient.GetConferenceState()` 连续失败 → 降级关闭 H 信号只用 A+B；全部信号失效 → 不影响原有 EndTime 兜底。
5. **可观测**：每次"延时"和"提前结束"写 audit log（含 `snapshot: silence_since / last_file_growth / file_size_bytes / file_growth_bps`）；新增 `smart_extend` / `smart_early_end` / `max_extend_reached` / `activity_watcher_degraded` 日志字段；可选 Prometheus counter。
6. **可配置**：阈值/开关放 `app config`（`smart_end:` 段，含 14 项配置：`silence_db / silence_duration_s / file_stall_s / file_min_growth_bps / huawei_enabled / huawei_poll_interval_s / huawei_persist_s / huawei_failure_threshold / check_interval_s / extend_step_min / max_extend_count / stat_failure_threshold / degrade_on_silence_loss`）。
7. **审计字段**：GORM 加 `ExtensionCount / LastExtensionReason / EndedEarly / EndedEarlyReason / EndedByHuaWeAPI`（AutoMigrate 列表同步；走 AutoMigrate 不进 dormant `runCustomMigrations`）。

## Non-goals

- 不改 `VideoRecordingTaskStatus` 枚举（沿用 `completed / failed / canceled`）
- 不做"会议提前结束预测"（无 ML/ASR）
- 不重写 `monitorProcessWithKey` 现有的断流重连逻辑
- 不动前端 UI（仅后端 + config；后续 Phase 2.4 单独做）
- 不接 `MSG_CONF_STATE_CHANGE` 推送通道（项目零 websocket 基础设施）

## Candidate future phases (本期不做)

- 2.1 `MSG_CONF_STATE_CHANGE` 推送接入
- 2.2 T.140 字幕信号
- 2.3 跨 input 一致性
- 2.4 前端可视化
- 2.5 机器学习预测

## 用户已确认

- 摘要看起来对，按此继续 ✓
- 跳过 research（PRD 已是调研交付）✓
- 35 REQ-IDs / 8 类 DETECT/WATCH/SCHED/EXTEND/EARLY/AUDIT/CFG/OBS ✓

## 已落盘产物（下次 /gsd:new-milestone 应识别并复用）

- `PROJECT.md` 已加 `Current Milestone: v2.0` 段（commit `cfdeb37`）
- `STATE.md` 已 `state.milestone-switch v2.0 智能录制收尾`（commit `cfdeb37`）
- `REQUIREMENTS.md` 已新建（commit `506904a`，114 行，35 REQ-IDs / 8 类 / 5 future / 7 out-of-scope）
- **本文件 MILESTONE-CONTEXT.md** 应在下次 `/gsd:new-milestone` §6 被消费并删除

## 下游期望

- `/gsd:new-milestone v2.0` 续跑时：
  - §2 应直接用本文件摘要，跳过内联提问
  - §4/§5 应检测到 PROJECT.md / STATE.md 已是 v2.0，跳过重写
  - §8 应跳过 research（已决定）
  - §9 应检测到 REQUIREMENTS.md 已存在，复用并跳过重 scoping
  - §10 应启动 `gsd-roadmapper`，phase 编号从 23 起（v1.1 终点 22），预期拆 3 phase（A: 华为 API + GORM 字段 + 错误码 / B: watcher + 兜底 + 录制层 / C: scheduler + service + E2E + CI）

## PRD 源

`docs/plans/2026-08-05-smart-meeting-recording-end-design.md`（525 行，v2 2026-08-05）