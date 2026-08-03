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

### Active

<!-- Next milestone scope. -->

(None — plan next milestone with /gsd-new-milestone)

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

- **v1.0** — 已交付归档（phase 01-14，产品功能完整）
- **v1.1 文件管理与编辑增强** — 全部 phase 完成（17 后端代码审查 56 项修复 / 18 凭据静态加密 SEC-003b SM4-GCM / 19 ctx 全量级联 + SEC-004 jti + STYLE-001 error / 20 HandleError 收敛 / 21 v1.1 过程缺口闭环 / 22 审计 tech debt 收尾）。Phase 22 (is_last_phase) 关闭 `v1.1-MILESTONE-AUDIT.md` 剩余 tech debt：重新生成 `docs/errors.md`（SYNC_OK，footer=16）+ 回填 phase 17/18/19/21 的 Nyquist VALIDATION.md（§Nyquist Compliance MISSING→present）+ 翻转 phase 20 VALIDATION.md 签核（nyquist_compliant/wave_0_complete→true，6/6 sign-off，honest verification 经 20-VERIFICATION 10/10 独立佐证）。里程碑可重审诚实归档为 `passed`。下一步：`/gsd:complete-milestone` 归档 v1.1，或 `/gsd:new-milestone` 规划下一里程碑。

---
*Last updated: 2026-08-03 after Phase 22 (v1.1 audit tech-debt closure, is_last_phase)*
