# Record V2 - 视频会议录制管理系统

## What This Is

视频会议录制管理系统 V2.0，专为华为会议终端设计的自动化录制、管理、转录和PPT生成平台。支持自动录制华为会议、USB设备录制、RTSP流录制，提供视频多点分割、阿里通义听悟AI转录、PPT自动提取等能力。面向企业内部用户，通过Web界面和API进行管理。

## Core Value

会议视频从录制到PPT的一站式处理，让会议内容可检索、可回顾、可分享。

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

- ✓ 华为会议终端自动录制 — 已实现
- ✓ 视频格式转换（MKV→MP4）— 已实现
- ✓ HLS实时预览 — 已实现
- ✓ RBAC权限系统 — 已实现
- ✓ SM4-GCM加密Token认证 — 已实现
- ✓ 审计日志 — 已实现
- ✓ 文件管理（上传/下载/配额/分享）— 已实现
- ✓ 通知系统 — 已实现
- ✓ API密钥管理 — 已实现
- ✓ USB设备扫描 — 已实现
- ✓ 视频多点分割 — v1.0 (Phase 1)
- ✓ 录制MP4快照 — v1.0 (Phase 1)
- ✓ 自动文件扫描 — v1.0 (Phase 1)
- ✓ 本地转录（SSIM/pHash/边缘检测）— v1.0 (Phase 2)
- ✓ PPT预览/下载/合并 — v1.0 (Phase 3)
- ✓ 阿里云OSS文件中转 — v1.0 (Phase 4)
- ✓ 通义听悟云端转录 — v1.0 (Phase 4)
- ✓ 云端/本地自动降级 — v1.0 (Phase 4)
- ✓ 后端代码审查 56 项修复（13 HIGH + 18 MEDIUM + 25 LOW，零回归）— v1.1 (Phase 17)
- ✓ 凭据静态加密 SM4-GCM envelope + 自动密钥轮换（SEC-003b）— v1.1 (Phase 18)
- ✓ ctx 全量级联（403 GORM 调用 + ~190 service 方法）+ jti replay 防御（SEC-004/PERF-003/BUG-005）— v1.1 (Phase 19)
- ✓ HandleError 统一收敛 + sentinel 体系 + zap SentinelField + 自动生成 docs/errors.md + CI sync-check — v1.1 (Phase 20)
- ✓ v1.1 REQ-ID 追溯体系（REQUIREMENTS.md，~80 REQ-IDs，0 orphan）— v1.1 (Phase 21)

### Active

<!-- Next milestone scope. -->

#### Milestone v2.0 — 智能录制收尾（Smart Recording End）

**Goal:** 让华为会议录制时长智能贴合会议真实时长——到点未结束自动延时（30min × 4 = 2h 上限），提前结束由 TE40 `WEB_GetMailboxDataAPI`（`confState=="" && joinSum==0`）主信号 + silencedetect + 文件停滞双兜底任一触发即收尾转码，无需人工干预。

**Target features:**

- **主信号 H（业务权威）**：在 `recorder/coordinator.go` 启 `HuaweiStatePoller goroutine`，每 30s 调 `HuaweiClient.GetConferenceState()`（底层调 `WEB_GetMailboxDataAPI`），判据 `confState=="" && joinSum==0` 持续 `huawei_persist_s`（默认 30s）→ 视为会议已结束。
- **兜底信号 A + B（双 AND）**：在 `recorder/coordinator.go` 给 ffmpeg 加 `silencedetect` 过滤器 + 输出文件大小采样（ticker 5s）。silencedetect 持续 ≥ 30s **且** 文件大小停滞 ≥ 120s → 也判定为结束（华为 API 不可达时仍可工作）。
- **scheduler 多信号驱动**：`monitorTask` 改为 `select` on EndTime timer + `taskEndedCh` + `taskUpdateChans`。EndTime 到点先问 watcher 决定延时/结束；若到点但任一判定信号活跃 → `EndTime += 30min`，累计 ≤ 4 次（2h 上限）。
- **可降级**：silencedetect 解析失败 → 切纯文件停滞 + 华为信号；`HuaweiClient.GetConferenceState()` 连续失败 → 降级关闭 H 信号只用 A+B；全部信号失效 → 不影响原有 EndTime 兜底。
- **可观测**：每次"延时"和"提前结束"写 audit log（含 `snapshot: silence_since / last_file_growth / file_size_bytes / file_growth_bps`）；新增 `smart_extend` / `smart_early_end` / `max_extend_reached` / `activity_watcher_degraded` 日志字段；可选 Prometheus counter（如项目无 prometheus 集成则仅做日志）。
- **可配置**：阈值/开关放 `app config`（`smart_end:` 段，含 `silence_db / silence_duration_s / file_stall_s / file_min_growth_bps / huawei_enabled / huawei_poll_interval_s / huawei_persist_s / huawei_failure_threshold / check_interval_s / extend_step_min / max_extend_count / stat_failure_threshold / degrade_on_silence_loss`）。
- **审计字段**：GORM 加 `ExtensionCount / LastExtensionReason / EndedEarly / EndedEarlyReason / EndedByHuaWeAPI`（AutoMigrate 列表同步；走 AutoMigrate 不进 dormant `runCustomMigrations`）。

**非目标（明确不做）：**

- 不改 `VideoRecordingTaskStatus` 枚举（沿用 `completed / failed / canceled`）。
- 不做"会议提前结束预测"（无 ML/ASR）。
- 不重写 `monitorProcessWithKey` 现有的断流重连逻辑（watcher 在重连期间保持静音/文件计时不重置）。
- 不动前端 UI（仅后端 + config；后续 Phase 2.4 单独做前端可视化）。
- 不接 `MSG_CONF_STATE_CHANGE` 推送通道（文档示例是 HTTP 长轮询而非 WebSocket；项目零 websocket 基础设施；30s 轮询已能满足"提前结束"判定时效）。

**候选后续 Phase（本期不做）：** 2.1 MSG_CONF_STATE_CHANGE 推送接入 / 2.2 T.140 字幕信号 / 2.3 跨 input 一致性 / 2.4 前端可视化 / 2.5 机器学习预测。

**PRD 来源：** `docs/plans/2026-08-05-smart-meeting-recording-end-design.md`（v2：纳入 TE40 邮箱 API 主信号；2026-08-05 调研）。
**项目内已有基础设施（无需新增）：** `HuaweiClient.GetMailboxData()`（`huawei/client.go:702-744`）仅需扩展 8 个 struct 字段；零新增依赖、零 websocket/SSE/long-polling 基建。

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- 实时语音转写 — 仅支持离线转录，不做实时字幕
- 自动转录触发 — 仅手动触发，不做录制后自动转录
- 视频智能场景检测 — 仅支持手动指定时间点分割
- 多语言翻译 — 通义听悟翻译功能暂不启用
- 文字生成PPT — 仅使用通义听悟的PPT画面提取功能

## Context

- **现有架构**: Go 1.24 (Gin) + React 19 (Ant Design 6), SQLite数据库, FFmpeg视频处理
- **认证方式**: SM4-GCM加密Token + API Key双认证
- **视频处理**: FFmpeg已集成，用于录制、格式转换、HLS流
- **外部服务依赖**: 华为会议系统API、阿里通义听悟API（新增）、阿里云OSS（新增）
- **服务器环境**: 无公网IP，需要阿里云OSS作为文件中转到通义听悟
- **通义听悟API版本**: 2023-09-30, domain: tingwu.cn-beijing.aliyuncs.com
- **v1.1 安全加固状态**: 凭据静态加密 SM4-GCM envelope（SEC-003b，fail-closed 启动不变量）；ctx 全量级联（GORM + service）；HandleError 单一错误响应路径收敛（消除 CR-01 双写）；jti replay 防御（TTL sweeper，单实例）；sentinel 体系 + 自动文档生成（CI sync-check 门禁）
- **已知 tech debt（v1.1 deferred）**: STYLE-001 全库 %w 迁移（~117 errors.New + ~474 fmt.Errorf）；STYLE-009 Get* rename（124 处）；KMS/Vault 自动注入凭据；真实生产数据 post-audit；v1.0 UAT/verification gaps（24 项遗留元数据，见 STATE.md Deferred Items）

## Constraints

- **Tech stack**: 后端Go (Gin), 前端React (Ant Design), 数据库SQLite — 已有架构不变
- **Storage**: 阿里云OSS仅作为临时文件中转，转录完成后删除云端文件
- **Tingwu API**: 需要公网可访问的视频URL，这是使用OSS的根本原因
- **Cost**: 阿里云OSS新用户3个月免费，之后约¥1-2/次转录，可接受
- **No public IP**: 服务器无公网IP，不能直接本地服务文件

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 全Go实现（非Python微服务） | 与现有代码库一致，单进程部署，运维简单 | ✓ Good |
| 阿里云OSS替代七牛云 | 七牛测试域名30天过期，OSS有永久域名且与通义听悟同生态 | ✓ Good |
| 通义听悟PPT提取（非文字转PPT） | 会议场景中PPT画面提取比文字生成更准确 | ✓ Good |
| 视频多点分割用FFmpeg | FFmpeg已集成，seek+copy模式快速无损分割 | ✓ Good |
| 转录仅手动触发 | 初期简化流程，避免自动转录的资源消耗和错误处理 | ✓ Good |
| SM4-GCM envelope 凭据加密（SEC-003b） | 华为密码明文→AEAD加密，与浏览器传输密钥族隔离，fail-closed 启动 | ✓ Good |
| jti replay 不加 DB 表（TTL sweeper） | 单实例 5min 窗口风险可接受，避免 Redis/GORM 架构复杂度 | ⚠️ Revisit（多实例部署需 Redis 共享 used_jtis） |
| HandleError 全量收敛 classify 函数 | 单一错误响应路径，消除 CR-01 双写 reintroduce 风险 | ✓ Good |
| error-doc-gen + CI sync-check | docs/errors.md 与代码漂移有 CI 门禁（.github/workflows/ci.yml:44-51） | ✓ Good |
| ctx 全量级联（非 per-method 增量） | 一次性消除 403 处 GORM ctx 缺失，避免渐进式遗漏 | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

## Milestone Status

- **v1.0** — 已交付归档（phase 01-14，产品功能完整，shipped 2026-05-06）
- **v1.1 文件管理与编辑增强 / 后端安全加固** — 已交付归档（phase 17-22，shipped 2026-08-03）。6 phase / 25 plans：17 后端代码审查 56 项修复 / 18 凭据静态加密 SEC-003b SM4-GCM / 19 ctx 全量级联 + SEC-004 jti + STYLE-001 error / 20 HandleError 收敛 / 21 v1.1 过程缺口闭环（REQUIREMENTS.md + retro-verify 17/18/19）/ 22 审计 tech debt 收尾（errors.md regenerate + 4 VALIDATION.md 回填 + phase 20 Nyquist 签核翻转）。重审审计 `status: tech_debt`，gaps 全空（requirements 60/60、phases 5/5、integration 5/5、flows 4/4）。
- **v2.0 智能录制收尾（Smart Recording End）** — 规划中（2026-08-06 start）。PRD 源 `docs/plans/2026-08-05-smart-meeting-recording-end-design.md`：主信号 H（华为 `WEB_GetMailboxDataAPI` confState+joinSum）+ 兜底 A+B（silencedetect + 文件停滞）OR 判定；scheduler 多信号驱动（EndTime + taskEndedCh + taskUpdateChans）；自动延时 30min × 4（2h 上限）+ 提前结束 + 强制截断（max_extend_reached）；GORM 字段 + audit log + 配置项全可观测可降级。预期 phase 23+。

---
*Last updated: 2026-08-06 — v2.0 智能录制收尾 milestone started*
