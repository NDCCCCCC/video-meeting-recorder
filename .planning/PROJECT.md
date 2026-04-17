# Record V2 - 视频会议录制管理系统

## What This Is

视频会议录制管理系统 V2.0，专为华为会议终端设计的自动化录制、管理、转录和PPT生成平台。支持自动录制华为会议、USB设备录制、RTSP流录制，提供视频多点分割、阿里通义听悟AI转录、PPT自动提取等能力。面向企业内部用户，通过Web界面和API进行管理。

## Core Value

会议视频从录制到PPT的一站式处理，让会议内容可检索、可回顾、可分享。

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

- ✓ 华为会议终端自动录制 — 已实现，支持自动/手动录制任务
- ✓ 视频格式转换（MKV→MP4）— 已实现，基于FFmpeg
- ✓ HLS实时预览 — 已实现，录制中可观看实时流
- ✓ RBAC权限系统 — 已实现，支持角色/权限/API密钥
- ✓ SM4-GCM加密Token认证 — 已实现
- ✓ 审计日志 — 已实现，记录所有操作
- ✓ 文件管理（上传/下载/配额/分享）— 已实现
- ✓ 通知系统 — 已实现
- ✓ API密钥管理 — 已实现
- ✓ USB设备扫描 — 已实现

### Active

<!-- Current scope. Building toward these. -->

- [ ] 视频多点分割 — 用户可在视频上标记多个时间点，将视频拆分为多段
- [ ] 阿里通义听悟转录集成 — 手动触发视频转录，支持PPT画面提取
- [ ] 阿里云OSS文件中转 — 上传视频到OSS获取公网URL供通义听悟访问
- [ ] 转录结果管理 — 转录任务状态跟踪，PPT和文本结果存储
- [ ] PPT独立下载 — 转录完成后PPT和视频分别下载

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
| 全Go实现（非Python微服务） | 与现有代码库一致，单进程部署，运维简单 | — Pending |
| 阿里云OSS替代七牛云 | 七牛测试域名30天过期，OSS有永久域名且与通义听悟同生态 | — Pending |
| 通义听悟PPT提取（非文字转PPT） | 会议场景中PPT画面提取比文字生成更准确 | — Pending |
| 视频多点分割用FFmpeg | FFmpeg已集成，seek+copy模式快速无损分割 | — Pending |
| 转录仅手动触发 | 初期简化流程，避免自动转录的资源消耗和错误处理 | — Pending |

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

---
*Last updated: 2026-04-17 after milestone v1.0 initialization*
