---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: 文件管理与编辑增强
status: milestone_complete
last_updated: 2026-08-03T04:13:43.402Z
last_activity: 2026-08-03
progress:
  total_phases: 2
  completed_phases: 1
  total_plans: 11
  completed_plans: 98
  percent: 50
stopped_at: Milestone complete (Phase 22 was final phase)
---

# STATE.md - Project Memory

**Project:** Record V2
**Milestone:** v1.1 - 文件管理与编辑增强
**Last Updated:** 2026-04-28
**Last Activity:** 2026-08-03

---

## Project Reference

### Core Value

会议视频从录制到PPT的一站式处理，让会议内容可检索、可回顾、可分享。

### What This Is

视频会议录制管理系统 V2.0，专为华为会议终端设计的自动化录制、管理、转录和PPT生成平台。支持自动录制华为会议、USB设备录制、RTSP流录制，提供视频多点分割、阿里通义听悟AI转录、PPT自动提取等能力。

### Current Focus

Phase 1: Video Splitting - Multi-point video splitting, recording snapshot, and auto scan (all local, no external dependencies).

---

## Current Position

Phase: 22 (address-v1-1-audit-tech-debt-regenerate-errors-md-backfill-v) — EXECUTING
Plan: Not started
**Phase:** 22
**Status:** Milestone complete
**Progress:** [███████░░░] 73%

### Phase Summary

按 Phase 17 延后项处理：审计 SEC-003b 修复 + 用户加码"自动密钥轮换"。算法 SM4-GCM（AEAD），envelope 格式 `SM4:<version>:<base64(nonce|ciphertext|tag)>`，密钥族与浏览器传输密钥完全隔离（`CREDENTIAL_SM4_*` vs `SM4_SECRET`），启动 fail-closed 10 步不变量扫描，操作员轮换手册 + 物理残留（WAL/vacuum/备份）边界文档。

### Wave Status

| Wave | Status | 内容 |
|------|--------|------|
| W1a | ✅ | SM4-GCM envelope + PKCS7 padding + tamper 检测 |
| W1b | ✅ | AuthConfig.CredentialSM4* + BindEnv + ValidateCredentialSM4Config |
| W1c | ✅ | CredentialEncryptor service + 列宽扩 + 集成测试 |
| W2  | ✅ | encrypt-on-write + decrypt-on-read 接入业务层 + admin 端点修复 + 删除 base64 stubs 死代码 |
| W3  | ✅ | fail-closed 启动 + 列宽扩展 + 集成测试 + 文档 |
| W4  | ✅ | DEPLOYMENT.md 操作员手册 + 重复轮换测试（v1→v2→v3）+ per-site/version 计数日志 + 物理残留章节 |

### Base HEAD for Phase 18

- Base: `e294ae9` (Phase 17 cross-AI review)
- After W3: `bd84fe2`
- After W3 docs: `3b1cb79`
- After W1+2+3 state: `8c69e33`
- After W4: `7e9baaf`

### Notes — Phase 18 Deviations (all accepted)

- **gmsm v1.4.1 nonce mutation** — GCM 内部修改 nonce slice backing array → 实现层做 defensive copy（已注释）
- **gmsm v1.4.1 GCMDecrypt block alignment** — 需先 PKCS#7 补到 16B 边界（tag 覆盖完整 plaintext）
- **No `migrations/017_*` file** — 列宽扩展直接由 `cmd/server/app.go:widenPasswordColumns()` 在 Initialize() 内执行，避开 dormant registry
- **18-SUMMARY.md at project root** — 与 Phase 17 `.planning/` 模式略有偏离，但保留在 git 历史便于追溯
- **Wave 4 test expected 4 envelopes rotated, actually 5** — Live3.stream_password 初次 Migrate 时是 v1，也被覆盖；测试预期已修正
- **GORM `.Scan(&rows)` 不支持 → `db.Raw(...).Rows()` + manual scan** — 在 `CountByVersion` 中已 workaround

### Out-of-Scope 列表（与 Phase 17 review §5 一致，仍未处理）

- `models.FileShare.Password`（共享链接密码需 salted hash，独立 phase）
- `models.APIKey.Key` / `Session.Token` / `AccessToken` / `ShareToken`（按值查找）
- 遗留 `huawei_configs` 表清理
- `StreamURL` URL-embedded 凭据拒绝
- dormant migration registry 清理
- 前端 transport GCM 迁移
- KMS / 真实生产数据 post-audit

---

### Phase Summary

按 P0→P1a→P1b→P2 顺序修复 `docs/audits/2026-07-30-backend-code-review.md` 中 **56 个代码审查发现**（13 HIGH + 18 MEDIUM + 25 LOW），全部落地 `main`：45 个原子 commit（41 代码/测试 + 4 docs/state），`go build ./...` + `go vet ./...` 静默通过，所有 12 个 phase-17 触及包 `gofmt -l` clean，12 包 `go test -race` 全部 PASS，零回归。

### Wave Status

| Wave | Plan | Commits | Status |
|------|------|---------|--------|
| 1 | 17-01 (P0) | 4 | ✅ Verified (SEC-001/002/003a/004 + BUG-001/002 + PERF-001/002/004/005 + 文档) |
| 2 | 17-02 (P1a) | 12 | ✅ Verified (BUG-003..006 + SEC-005..010 + STYLE-004/005) |
| 3 | 17-03 (P1b) | 7 | ✅ Verified (PERF-006..011 + STYLE-003 接口归位) |
| 4 | 17-04 (P2) | 18 | ✅ Verified (BUG-011/015/016 + SEC-011..015 + PERF-012..016 + STYLE-001 partial/006/007/008/010) |
| + | housekeeping | 1 | ✅ gofmt 修复 2 个 Wave-1 跨 wave 遗漏 |
| + | docs checkpoints | 3 | ✅ STATE/ROADMAP x3 (W1/W2/W3) |

### Deferred (独立 phase 处理，per CONTEXT.md `<deferred>`)

- **STYLE-001 全库 %w 迁移**：168 处 `errors.New` + 474 处 `fmt.Errorf` → `%w`（本 phase 仅 2 service + 1 handler + 3 处 `%w` 包装 in touched files）
- **SEC-003b** 华为密码 DB 加密（`models.InputConfig.Password` 明文 → SM4-ECB），需独立迁移 + 前端/配置联动
- **PERF-003** `video_recording_task_service` 全面 `WithContext(ctx)` 透传（需为 30 个 service 方法加 ctx 参数，403 处签名级联）
- **STYLE-009** 133 处包名冗余 `Get*` rename（CONTEXT.md Claude's Discretion 默认跳过）
- **HMAC jti 服务端 `used_jtis` 表**（Redis 或 DB）—— 架构决策
- **`koanf` 替代 viper**、**audit 包迁移**、**golangci-lint + errcheck/gosec** 等工具链改进

### Wave 3 Recovery Note

Wave 3 (P1b) 执行器在写 SUMMARY.md 时遇上游 API 错误（`模型不存在`）崩溃；**全部 7 个代码+测试 commit 已落地 main**，build/tests 绿，仅 SUMMARY 叙述中断——由 orchestrator 通过文件系统核验确认恢复，无需重跑 plan。Wave 4 prompt 已加入"API 错误时优先保证 SUMMARY 写盘提交"的纪律。

### Phase Base HEAD

- 规划基线：`cf2d248` (docs(17): create phase plan)
- 最终 HEAD：`c04f805` (chore housekeeping + gofmt clean)
- 增量：45 commits (41 code/test + 4 docs/state)

### Next Steps (非阻塞)

Phase 17 完成。无即时 follow-up。可选：

1. 用真实凭据做生产环境审计验证（参考 `.planning/quick/260729-lr4-100/260729-lr4-SUMMARY.md` 验证步骤）——验证新加的 12 个 auth 审计点是否真的入 audit_logs
2. 处理 `<deferred>` 列表中的任何独立 phase
3. 整理前端 23 个未提交文件（与 phase 17 无关）

---

---

### Roadmap Evolution

- Phase 21 added: Close v1.1 gaps — retro-verify phases 17/18/19 + create REQUIREMENTS.md + fix auth_handler.go:57 WARNING (driven by /gsd-audit-milestone gaps_found, 2026-08-01)

---
- Phase 22 added: Address v1.1 audit tech debt: regenerate errors.md + backfill VALIDATION.md for 17/18/19/21

## Phase 19 — ctx 全量级联 + SEC-004 replay 修复 + STYLE-001 error 迁移

### Phase 19 Scope (用户确认)

纳入：

- **PERF-003/BUG-005**：ctx 全量级联（403 处 GORM 调用、~190 service 方法、11+ service ctx-less 文件）
- **SEC-004**：jti replay 模型修复（**不加 DB 表**，仅修复一次性问题 + TTL sweeper）
- **STYLE-001**：error-mapping 三组件（mapping.go + HandleError + error_mapper.go）

排除（用户确认不修）：

- PERF-001（Preload N+1 审计误判）
- STYLE-009（Get* rename 130 处，API 破坏性）
- PERF-009（audit map schemaless 内在）

### 最终交付（9 commits，main 落地）

| Wave | HEAD | 范围 |
|------|------|------|
| W4 | `9a00cbe` | TaskServiceInterface ctx 原子三元组（adapter + interface + mock 同步） |
| W4 | `2281927` | docs: Wave 4 SUMMARY section |
| W5* | `34b07f7` | VideoRecordingTaskService ctx-first (22 方法) |
| W5a | `e2b0b6b` | VideoFileService 内部 helpers (4 方法 + 3 caller) |
| W5b | `7828fc3` | ScanFiles chain (5 方法 + 1 caller) |
| W5c | `7a5a1cc` | batch ops (3 方法 + 1 caller) |
| W5d | `1ae6be0` | 全量 caller ctx 透传 (handlers + scheduler + tests) |
| W5e | `b08255d` | ctx 取消传播 contract test + Wave 5 总结 |
| W6  | `3d171de` | STYLE-001 error 迁移 (gorm wrap + HandleError) |
| W6  | `6edb772` | docs: Wave 6 summary + 范围对账 |

### 验证

- `go build ./...` 0 错误
- `go vet ./...` 0 错误
- `go test -race ./internal/services/... ./internal/handlers/... ./internal/scheduler/... ./internal/utils/... ./internal/middleware/...` 全绿

### Scope-vs-执行 对账

| 承诺元素 | 实际 |
|---------|------|
| ctx 级联 ~190 service 方法 | ✅ VideoRecordingTaskService 22 + VideoFileService 23 + 调用者全栈 |
| jti replay 修复（加 DB 表） | ❌ → ✅ **不加 DB 表** + TTL sweeper +400 行 |
| error-mapping 三组件 + middleware | ✅ mapping.go + HandleError + error_mapper.go + 全局注册 |
| 服务边界 gorm wrap | ✅ notification/ppt_file/timestamp_mapper/video_file 各 `==` → `errors.Is` |
| handler string-match → HandleError | ✅ ppt_handler.go RenamePPTFile + video_file_handler.go RenameVideoFile |
| 高频 handler 5-10 路径迁移 | ✅ 2 handler（重命名路径是最高频用户错误路径） |

### DEFERRED（保留给后续 phase）

| 项 | 原因 |
|----|------|
| `internal/services/video_file_service.go:891` strings.Contains "FOREIGN KEY" | 仅诊断日志增强，非用户错误路径 |
| `taskServiceAdapter` 与 `VideoFileService` 合并 | 含 Phase 18 SM4-GCM 解密逻辑，独立 phase |
| `internal/errors` 包被 0 service 文件 import（部分） | 大部分 service 仍用 `fmt.Errorf` + `errors.Is(gorm.ErrRecordNotFound)`，全库迁移增量在 Wave 6 范围外 |
| HMAC jti DB 表（Redis 或 GORM） | 用户"不加 DB 表"决策的 5min TTL 单实例窗口风险保留作架构 future work |

### Phase 19 Base HEAD

- 规划基线：`cf2d248` (Phase 17 plan)
- Phase 19 进入基线：`2281927` (W4 docs commit 前)
- 最终 HEAD：`6edb772` (Phase 19 docs 总结)
- 增量：11 commits（Wave 4-6 + docs）

### 用户 follow-up（不阻塞）

- 可选：处理 `<deferred>` 列表中的任何项为独立 phase
- 可选：用真实凭据做手工验证（新加 12 个 auth 审计点 + Wave 6 handler 错误路径）

---

---

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260428-pvs | 新建及编辑用户模态框添加检查按钮，可以通过用户名查找域控中的相关信息并自动填充，比如姓名和邮箱 | 2026-04-28 | 5d569fb | [260428-pvs-ad-user-lookup](./quick/260428-pvs-ad-user-lookup/) |
| 260428-ad | AD用户白名单 - 只允许已存在的AD用户登录 | 2026-04-28 | - | [260428-ad-whitelist](./quick/260428-ad-whitelist/) |
| 260428-n0k | AD配置持久化到数据库，服务器重启后恢复 | 2026-04-28 | - | [260428-n0k-ad](./quick/260428-n0k-ad/) |
| 260428-mlh | 前端域控账号登录使用SM4加密密码，后端解密后传给域控服务器 | 2026-04-28 | 1500094 | [260428-mlh-sm4](./quick/260428-mlh-sm4/) |
| 260428-m9t | 登录后右上角去掉个人信息按钮，为系统设置添加路由，创建认证管理菜单 | 2026-04-28 | 0d872a8 | [260428-m9t-sidebar](./quick/260428-m9t-sidebar/) |
| 260423-f7v | 文件管理页面添加视频上传功能 | 2026-04-23 | d4f78f7 | [260423-f7v-add-video-upload-feature](./quick/260423-f7v-add-video-upload-feature/) |
| 260729-kbf | 检查审计日志是否对所有操作进行了审计（覆盖率≈14%，audit.go 中间件 dead code） | 2026-07-29 | - | [260729-kbf-audit-log-coverage](./quick/260729-kbf-audit-log-coverage/) |
| 260729-lr4 | 补充写操作审计覆盖率到100%，处理凭据脱敏引入的新安全风险 | 2026-07-29 | d4c4fb7 | [260729-lr4-100](./quick/260729-lr4-100/) |
| 260729-m8l | 补 OldData 捕获支持 update/delete 差异对比（6 个代表性站点 + 21 个待接入清单） | 2026-07-29 | 2cef9f0 | [260729-m8l-olddata-update-delete](./quick/260729-m8l-olddata-update-delete/) |
| 260729-mwt | 补 input-config / system / file OldData 捕获（6 个高危站点） | 2026-07-29 | 20a7abe | [260729-mwt-input-config-system-file-olddata-6](./quick/260729-mwt-input-config-system-file-olddata-6/) |
| 260730-bc3 | 补 16 站点 OldData 捕获（recording 5 + storage 3 + ppts 4 + apikey 3 + notification 1，中危 P1） | 2026-07-30 | a494c77 | [260730-bc3-38-recording-5-storage-3-ppts-4-apikey-3](./quick/260730-bc3-38-recording-5-storage-3-ppts-4-apikey-3/) |
| 260730-dr8 | 补 48 个敏感 GET 端点审计（HIGH 13 + MEDIUM 35） | 2026-07-30 | e934df9 | [260730-dr8-42-get-high-14-medium-28](./quick/260730-dr8-42-get-high-14-medium-28/) |
| 260730-eis | 清理构建阻塞 + line-ending 修复（app.go CRLF + frontend/dist 占位 + ip_restriction_test.go ctx 未透传） | 2026-07-30 | 4d2e39f | [260730-eis-clean-build-blockers-line-ending-ctx](./quick/260730-eis-clean-build-blockers-line-ending-ctx/) |

---

## Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260428-pvs | 新建及编辑用户模态框添加检查按钮，可以通过用户名查找域控中的相关信息并自动填充，比如姓名和邮箱 | 2026-04-28 | 5d569fb | [260428-pvs-ad-user-lookup](./quick/260428-pvs-ad-user-lookup/) |
| 260428-ad | AD用户白名单 - 只允许已存在的AD用户登录 | 2026-04-28 | - | [260428-ad-whitelist](./quick/260428-ad-whitelist/) |
| 260428-n0k | AD配置持久化到数据库，服务器重启后恢复 | 2026-04-28 | - | [260428-n0k-ad](./quick/260428-n0k-ad/) |
| 260428-mlh | 前端域控账号登录使用SM4加密密码，后端解密后传给域控服务器 | 2026-04-28 | 1500094 | [260428-mlh-sm4](./quick/260428-mlh-sm4/) |
| 260428-m9t | 登录后右上角去掉个人信息按钮，为系统设置添加路由，创建认证管理菜单 | 2026-04-28 | 0d872a8 | [260428-m9t-sidebar](./quick/260428-m9t-sidebar/) |
| 260423-f7v | 文件管理页面添加视频上传功能 | 2026-04-23 | d4f78f7 | [260423-f7v-add-video-upload-feature](./quick/260423-f7v-add-video-upload-feature/) |
| 260729-kbf | 检查审计日志是否对所有操作进行了审计（覆盖率≈14%，audit.go 中间件 dead code） | 2026-07-29 | - | [260729-kbf-audit-log-coverage](./quick/260729-kbf-audit-log-coverage/) |
| 260729-lr4 | 补充写操作审计覆盖率到100%，处理凭据脱敏引入的新安全风险 | 2026-07-29 | d4c4fb7 | [260729-lr4-100](./quick/260729-lr4-100/) |
| 260729-m8l | 补 OldData 捕获支持 update/delete 差异对比（6 个代表性站点 + 21 个待接入清单） | 2026-07-29 | 2cef9f0 | [260729-m8l-olddata-update-delete](./quick/260729-m8l-olddata-update-delete/) |
| 260729-mwt | 补 input-config / system / file OldData 捕获（6 个高危站点） | 2026-07-29 | 20a7abe | [260729-mwt-input-config-system-file-olddata-6](./quick/260729-mwt-input-config-system-file-olddata-6/) |
| 260730-bc3 | 补 16 站点 OldData 捕获（recording 5 + storage 3 + ppts 4 + apikey 3 + notification 1，中危 P1） | 2026-07-30 | a494c77 | [260730-bc3-38-recording-5-storage-3-ppts-4-apikey-3](./quick/260730-bc3-38-recording-5-storage-3-ppts-4-apikey-3/) |
| 260730-dr8 | 补 48 个敏感 GET 端点审计（HIGH 13 + MEDIUM 35） | 2026-07-30 | e934df9 | [260730-dr8-42-get-high-14-medium-28](./quick/260730-dr8-42-get-high-14-medium-28/) |
| 260730-eis | 清理构建阻塞 + line-ending 修复（app.go CRLF + frontend/dist 占位 + ip_restriction_test.go ctx 未透传） | 2026-07-30 | 4d2e39f | [260730-eis-clean-build-blockers-line-ending-ctx](./quick/260730-eis-clean-build-blockers-line-ending-ctx/) |

---

## Performance Metrics

### Requirements Coverage

- Total v1 requirements: 30
- Mapped to phases: 30 (100%)
- Unmapped: 0

### Phase Breakdown

| Phase | Requirements | Status |
|-------|--------------|--------|
| Phase 1 - Video Splitting | 10 | Pending |
| Phase 2 - Local Transcription | 7 | Pending |
| Phase 3 - PPT Management | 7 | Pending |
| Phase 4 - Cloud Services | 6 | Pending |

---
| Phase 05-file-rename P02 | 18 | 4 tasks | 5 files |
| Phase 07 P04 | 3min | 3 tasks | 1 files |
| Phase 08 P03 | 64 | 2 tasks | 2 files |
| Phase 08 P04 | 4min | 3 tasks | 1 files |
| Phase 11 P01 | 9min | 4 tasks | 9 files |
| Phase 12 P04 | 71 | 2 tasks | 3 files |
| Phase 15 P01 | 7min | 2 tasks | 4 files |
| Phase 15 P02 | 540 | 2 tasks | 7 files |
| Phase 15-ai P04 | 6m | 2 tasks | 3 files |
| Phase 15 P5 | 6min | 2 tasks | 6 files |
| Phase 15-ai P06 | 12min | 2 tasks | 9 files |
| Phase 22 P03 | 4min | 1 tasks | 2 files |
| Phase 22 P04 | 6min | 1 tasks | 2 files |

## Roadmap Evolution

- Phase 7 added: Preview Page UI Improvements (2026-04-20)
- Phase 8 added: Video Snapshot & Player Enhancement (2026-04-20)
- Phase 9 added: Multi-Role Permissions & Shared Viewer (2026-04-21)
- Phase 10 added: Admin Dashboard, Audit Logs, and UI Enhancements (2026-04-24)
- Phase 11 added: IP地址登录限制 - 为用户和角色添加IP地址组 (2026-04-27)
- Phase 12 added: Windows AD域控认证 - 集成Windows Active Directory域控认证，支持LDAP/LDAPS双端口 (2026-04-28)
- Phase 13 added: 重构华为配置，支持USB设备和流媒体录制模式 (2026-04-29)
- Phase 14 added: 文件管理页面添加批量下载和批量转录功能 (2026-04-30)
- Phase 1 added: 新功能 - 在视频播放中添加外挂字幕支持（预览视频、切割视频、预览PPT页面） (2026-05-12)
- Phase 15 added: 前端去 AI 味 (2026-07-28)
- Phase 17 added: 后端代码审查 56 个发现修复 - P0/P1/P2 全量 (2026-07-30)
- Phase 20 added: 错误处理统一收敛 + sentinel 体系增强（HandleError 全量替换 classify / zap logger errors.Is / 自动文档生成 / typed error kind） (2026-08-01)

## Accumulated Context

### Key Architectural Decisions

1. **All Go Implementation** (No Python microservice)
   - Rationale: Consistency with existing codebase, single-process deployment, simpler operations
   - Status: Pending validation

2. **Aliyun OSS for File Relay**
   - Rationale: Server has no public IP; Tingwu API requires publicly accessible URLs
   - Status: Pending validation

3. **Tingwu REST API with Manual HMAC-SHA256 Signing**
   - Rationale: No official Go SDK exists for Tingwu API
   - Status: Pending validation

4. **FFmpeg for Splitting and Frame Extraction**
   - Rationale: FFmpeg already integrated for recording/conversion
   - Status: Pending validation

5. **Local PPT Generation with Go-pptx Library**
   - Rationale: No need to generate PPT for cloud paths (download directly)
   - Status: Pending validation

6. **Manual Transcription Trigger Only**
   - Rationale: Cost control, user choice, simplified error handling
   - Status: Pending validation

7. **Atomic File Rename with Transaction Rollback** (Phase 05)
   - Rationale: Ensure data consistency when updating both DB records and physical files
   - Pattern: Start DB transaction → os.Range physical file → update DB → commit on success, rollback on failure
   - Status: Implemented and tested (12 test cases passing)

8. **Original Recording Immutability** (Phase 05)
   - Rationale: Original recordings are source of truth; splits/snapshots are derived copies
   - Pattern: Check source_type='recording' && parent_id=NULL to reject rename operations
   - Status: Enforced at service layer

9. **File Extension Preservation at Service Layer** (Phase 05)
   - Rationale: Prevent malicious extension changes; maintain file type consistency
   - Pattern: Extract extension from current file path, append to user-provided name
   - Status: Implemented for both VideoFile (.mp4) and PPTFile (.pptx)

### Tech Stack Context

**Backend:**

- Go 1.24 (Gin framework)
- SQLite database with GORM
- SM4-GCM encryption for Token authentication
- FFmpeg for video processing (already integrated)

**Frontend:**

- React 19
- Ant Design 6
- Zustand for state management
- TanStack Query for API caching

**External Dependencies (New):**

- Aliyun OSS Go SDK v2 (alibabacloud-oss-go-sdk-v2)
- Aliyun Tingwu API (manual REST with HMAC-SHA256)
- Muprprpr/Go-pptx for PPTX generation

### Dependency Analysis

**Phase 1 (Video Splitting):**

- SPLIT-01 to SPLIT-05: Video multi-point splitting with FFmpeg
- SNAP-01, SNAP-02: Recording snapshot without interrupting
- SCAN-01, SCAN-02: Auto file scanning
- UI-01: Split page layout
- Depends on: Nothing (first phase, all local)

**Phase 2 (Local Transcription):**

- LCL-01 to LCL-04: Frame extraction + SSIM/pHash/edge detection + PPTX generation
- TRAN-01 (local), TRAN-04, TRAN-06: Local transcription trigger, status, segment
- Depends on: Phase 1 (split segments need transcription)

**Phase 3 (PPT Management):**

- PPT-01 to PPT-06: PPT preview, download, multi-result, merge
- UI-03: PPT result page layout
- Depends on: Phase 2 (transcription produces PPT results)

**Phase 4 (Cloud Services):**

- OSS-01, OSS-02: Aliyun OSS file relay
- TRAN-01 (cloud), TRAN-02, TRAN-03, TRAN-05: Cloud transcription + fallback
- UI-02: Transcription task page layout
- Depends on: Phase 2 (local fallback), Phase 3 (PPT management)

### Critical Pitfalls from Research

1. **OSS File Orphaning** - Temporary files never deleted, causing indefinite storage costs
   - Mitigation: Lifecycle rules, cleanup handlers, periodic cleanup job

2. **Tingwu Status Polling Thundering Herd** - Rate limiting from simultaneous status requests
   - Mitigation: Jittered exponential backoff, staggered polls, global rate limiter

3. **FFmpeg Keyframe Misalignment** - ±2s precision limitation with -c copy mode
   - Mitigation: Document limitation, offer re-encode option, smart split to nearest keyframe

4. **PPT Image URL Download Timeouts** - Sequential downloads take 5-20 minutes
   - Mitigation: Parallel downloads with worker pool, progress tracking, retry with backoff

5. **Database Transaction Mismatch with OSS** - DB rollback leaves orphaned OSS files
   - Mitigation: Two-phase commit pattern, idempotent operations, state machine

---

## Decisions Log

### 2026-08-03 - Phase 19 Nyquist validation contract retro-fitted from commit evidence (Phase 22 Plan 04)

**Decision:** Backfill `19-VALIDATION.md` with a wave-based map because the original Phase 19 `PLAN.md` / `CONTEXT.md` are unavailable. Enumerate every W0-W6/docs, D1-D4, and D5-D21 commit from `19-VERIFICATION.md`; retain `status: draft`, `nyquist_compliant: false`, and `wave_0_complete: false`; treat body endpoint `6edb772` as authoritative over stale frontmatter endpoint `cacc294`.
**Rationale:** A retro-fitted contract must preserve honest Nyquist provenance while closing the audit's missing-artifact gap without fabricating task IDs or commit hashes.
**Outcome:** Task commit `428491c` adds only the 124-line `.planning/phases/19-ctx-cascade-sec-004-style-001-error/19-VALIDATION.md`, with 42 expected commit prefixes, 6 required sections, and 6 checked sign-off items.

### 2026-08-03 - Phase 18 Nyquist validation contract retro-fitted from live wave evidence (Phase 22 Plan 03)

**Decision:** Backfill `18-VALIDATION.md` as a wave-based contract because Phase 18 has no surviving PLAN.md. Use `18-SUMMARY.md` §Wave 4 body as authoritative for all 9 wave commits through W4d `0c018f2`, retain post-audit `5d536ec` as explicitly pre-stored evidence, and keep `status: draft`, `nyquist_compliant: false`, and `wave_0_complete: false` because the contract was reconstructed after execution.
**Rationale:** The SUMMARY frontmatter ends at W3 `bd84fe2` and is stale; body + git history + live files provide the complete evidence chain. The live repository also contains `internal/utils/sm4_password_test.go`, overriding the plan's stale note that no independent SM4 password test existed. This preserves honest Nyquist provenance while ensuring Wave 0 names every shipped Phase 18 test artifact.
**Outcome:** Task commit `c3fb6fd` adds only the 115-line `.planning/phases/18-credential-static-encryption-sec-003b/18-VALIDATION.md`; quick tests across `internal/utils`, `internal/services`, and `cmd/server` pass. Phase 22 is now 3/6 plans executed; next is 22-04.

### 2026-08-03 - docs/errors.md regenerated by auto-generator; audit footer refresh (Phase 22 Plan 01)

**Decision:** Run `go generate ./internal/errors/...` from repo root (D:/CODE/ClaudeCode/record_V2); commit only the regenerated `docs/errors.md`. Post-Phase 20 convergence status: Sentinel Table 42 rows, BusinessError Table 10 rows, Ad-hoc Error Audit footer count = 16 (target 0). Per-sentinel call-site count deltas vs prior committed file: ErrInternal 105→108 (+3), BusinessError(INTERNAL_ERROR) 66→67 (+1), BusinessError(NOT_FOUND) 49→50 (+1); all other 48 cells unchanged. Footer total unchanged because the grep pattern (`internal/handlers/*.go` excluding `_test.go` and `ShouldBindJSON` blocks) still finds exactly 16 sites — i.e., no new handler regressed to inline `err.Error()` classify; the per-cell increments come from `+String` chains wrapping the same BusinessError at multiple call points.
**Rationale:** v1.1-MILESTONE-AUDIT.md §Executive Summary listed "audit footer drift (31 sites)" as tech_debt — the prior committed `docs/errors.md` predated Phase 20 HandleError convergence (which reduced inline classify from 31 → 16). Fresh regen closes the cleanup signal: CI sync-check at `.github/workflows/ci.yml` lines 44-51 (`go generate && git diff --quiet docs/errors.md`) now passes against the committed file (verified locally: `go generate && git diff --quiet docs/errors.md && echo SYNC_OK` → exit 0). Generator determinism confirmed by running it twice; both runs produced byte-identical output. **Pitfall 1 avoided (per repo rule: `.planning/` gitignored)**: `22-01-SUMMARY.md` committed with `git add -f` so the .planning/ tree is actually tracked. **Pitfall 2 (rule: don't manually edit auto-gen doc)**: no hand edits to docs/errors.md; all deltas are reproduced by the generator from current source state. **Pre-existing partial-regen sanity check**: the working tree already contained `ErrInternal 105→108, INTERNAL_ERROR 66→67, NOT_FOUND 49→50, audit 15→16` from a prior session — re-running the generator reproduced these exact values byte-for-byte, confirming the partial regen was authoritative and no manual revert was needed (per repo note "Do NOT blindly trust the existing working-tree diff").
**Outcome:** Two commits on main: `1829adc` (single-file docs commit: docs/errors.md, 4 insertions / 4 deletions) + `ebfe20c` (planning metadata: `.planning/phases/22-.../22-01-SUMMARY.md`, 129 lines). `go build ./...` exit 0 confirms no business-code side effects. `git show --stat HEAD` (post-doc-commit) lists only docs/errors.md per the "no business code touched" gate. Phase 22 now 1/6 plans done; remaining plans (22-02..22-06, VALIDATION.md backfill for phases 17/18/19/21) can proceed against the now-fresh docs/errors.md baseline + committed audit-footer reference value of 16.

### 2026-08-03 - REQUIREMENTS.md created — v1.1 milestone REQ-ID traceability (Phase 21 Plan 04)

**Decision:** 创建 `.planning/REQUIREMENTS.md` (251 行, 5 列 per CONTEXT D-03.3), 覆盖 ~80 REQ-IDs 跨 v1.1 四 phase (52 REQ-17-* 计 SEC-003a/b 拆分 + 5 REQ-18-* + 4 REQ-19-* + 11 REQ-20-*). 跨 phase 兑现项显式标注 (REQ-17-SEC-003b→Phase 18 / REQ-17-PERF-003→Phase 19 / REQ-17-BUG-005→Phase 19 / REQ-17-HMAC-jti→Phase 19 D3). Coverage 段含 4 条 orphan 检测 grep 规则; Out-of-scope observation 段记录 phase 16 归属歧义 (不强行裁定 per D-03.4); Canonical References 段登记 root 18/19-SUMMARY.md + docs/audits/* + 4 个 VERIFICATION.md 路径.
**Rationale:** v1.1-MILESTONE-AUDIT.md gaps_found 标记 "REQUIREMENTS.md missing, orphan 检测 impossible" — 本 plan 关闭该 gap. 关键设计选择: (1) 每个 audit finding 单独成行 (52 行而非 range 简写) 给出最干净审计轨迹, 超过 40 行门槛 30%; (2) BUG-007/008/009/010/012/013/014 审计报告 "0 处" 合并为单行 N/A 保留 orphan 检测完整性; (3) 跨 phase 兑现 IDs 保留原 phase-17 身份 (Phase column '17→18') 而非重新发 REQ-18-* IDs, 避免破坏 audit-finding 追溯; (4) 每 REQ-18-*/REQ-19-* 行显式标 "backfilled from deliverables, SUMMARY frontmatter was empty" per D-03.5 不伪造追溯; (5) phase 16 仅作 Out-of-scope 观察 (不裁定 per D-03.4), 留待 milestone 决策; (6) 双 canonical 路径登记 (root + phase-dir 副本) — root = git 历史原版, 副本 = VERIFICATION 同目录便于追溯. 全部 15 项 verify 检查 PASS (REQ-17 52 行 >= 40, REQ-18 5 行, REQ-19 4 行, REQ-20 11 行, 4 个 VERIFICATION 路径全引, 跨 phase 字样存在, "不强行裁定" 字样存在, 251 行 >= 120).
**Outcome:** commit `695b4fe` on main (仅含 .planning/REQUIREMENTS.md, 未触业务代码 / docs/audits); SUMMARY commit `a00a1a3`. Phase 21 现在 5/5 plans done (21-01/02/03/04/05); v1.1-MILESTONE-AUDIT.md 的 5 项 gap 全部可标记关闭. 后续动作 (out-of-scope): 重跑 `/gsd:audit-milestone v1.1` 验证 gaps_found → passed (本 phase 不含此重跑); phase 16 归属裁定 (milestone-level 决策); 审计 tech_debt 10+ 项 (各为后续独立 phase).

### 2026-08-03 - auth_handler.go:57 canonical HandleError pattern (Phase 21 Plan 05)

**Decision:** Collapse Login handler's 5-line pattern (`if response.HandleError(c, err) { return }; // 兜底：unknown error（response.HandleError 已写 500）。; return` at line 57-61) into the canonical 2-line form (`response.HandleError(c, err); return` at line 57-58). Delete the line-60 兜底 comment (fallback branch no longer exists). Retain the line 53-56 `// Phase 20 (20-02)...` mapping.go comment block (documents sentinel→HTTP-status mapping, separate concern from if-vs-canonical shape).
**Rationale:** v1.1-MILESTONE-AUDIT.md §Cross-Phase Integration Findings flagged auth:57 as the only handler family (1/9) NOT using the canonical `HandleError(c, err); return` pattern — a latent CR-01 reintroduction vector if a future contributor appended a GinError after line 59. Behavior-equivalent per 21-RESEARCH §6 + CONTEXT D-04.2 (corrected): both branches of the original pattern (known→return inside if / unknown→fall through to bare return) "write response then return", identical to canonical `HandleError; return`. Key invariant: at auth:57 call site `c.Writer.Written()=false` (ShouldBindJSON failure already returned at line 36-39 with GinError; Warn log line 48-52 doesn't write HTTP), so HandleError must reach its GinErrorWithStatus write branch. **Pitfall 1 avoided**: commit body uses the control-flow argument, NOT the factually-wrong "HandleError always returns true" (it returns `errors.IsKnownError(err)` = false for unknown errors per pkg/response/response.go:179). Regression net intact: TestLogin_HandleError_ClassifyDrop 10 sub-tests (5 error classes × wrapped/unwrapped + R-3/R-4) all PASS, go build/test -race green. Tech_debt scope guard: RefreshToken :93 / ChangePassword :182 / LogoutAll raw `GinError + err.Error()` leaks NOT touched (D-04.4 — deferred to follow-up phase).
**Outcome:** commit `4959e9c` on main (single-file: `internal/handlers/auth_handler.go`, 1 insertion, 4 deletions); v1.1-MILESTONE-AUDIT.md auth:57 WARNING can be marked closed. Phase 21 now 4/5 plans done (21-01/02/03/05); only 21-04 (REQUIREMENTS.md, Wave 2) remains — its depends_on [21-01,21-02,21-03] is satisfied.

### 2026-08-03 - Phase 19 retro-verify directory reconstructed + VERIFICATION.md (Phase 21 Plan 03)

**Decision:** 重建 `.planning/phases/19-ctx-cascade-sec-004-style-001-error/` (目录被早期清理/从未建但代码已全部落地 main); 复制 root `19-SUMMARY.md` + `docs/audits/phase-19-D5-D21-summary.md` 入目录 (原版均不动 per D-02.6); 新写 `19-VERIFICATION.md` via goal-backward retro-verify, status: passed (10/10 must-haves), 220 行, 41 VERIFIED 标记.
**Rationale:** v1.1-MILESTONE-AUDIT.md gaps_found 标记 phase 19 "missing-directory" — 实际交付物全部落地 main (11 wave commits W0-W6 + 21 D1-D21 sentinel 化 commits = 32+ commits 总计, 新增 24 sentinels, 迁移 ~356 散点). 证据基于 root 19-SUMMARY (33KB) + D5-D21 summary (8.5KB) + 32+ commits + STATE.md §Phase 19 + 实时代码 (42 WithContext sites in video_recording_task_service.go + HLSJtiRecord AutoMigrate at app.go:340 + 42 knownSentinels slice in mapping.go) 多源交叉 (per 21-CONTEXT D-02.5). body 为准 per Pitfall 2: 32+ commits 全列 (W0 ad7d0a8 / W1 6fbdad4 / W2 cacc294 / W3 213710c..a6c21b6 / W4 9a00cbe+2281927 / W5 34b07f7+e2b0b6b+7828fc3+7a5a1cc+1ae6be0+b08255d / W6 3d171de / D1 20ee289 / D2 3b2d41f / D3 1f0ec35 / D4 f4291f5 / docs 6edb772), 不只信 frontmatter 的 W2 终点 cacc294. 4 项 19-SUMMARY §DEFERRED 中 3 项实际在 D1-D4 收尾阶段交付 (D2 adapter 合并 / D3 hls_jti_records 表 / D4 errors 包增量), 仅 video_file_service:891 strings.Contains 真正 deferred (低价值诊断日志).
**Outcome:** commit `4b52463` on main (仅含 .planning/phases/19-*/ 内 4 文件: .gitkeep + 19-SUMMARY.md 副本 + phase-19-D5-D21-summary.md 副本 + 19-VERIFICATION.md, 未触业务代码); v1.1-MILESTONE-AUDIT.md 的 "phase-19 missing-directory" gap 可标记关闭. 后续 21-04 (REQUIREMENTS.md, wave 2 — depends_on [21-01,21-02,21-03] 已就绪) + 21-05 (auth:57 fix) 继续.

### 2026-08-03 - Phase 18 retro-verify directory reconstructed + VERIFICATION.md (Phase 21 Plan 02)

**Decision:** 重建 `.planning/phases/18-credential-static-encryption-sec-003b/` (目录被早期清理/从未建但代码已全部落地 main); 复制 root `18-SUMMARY.md` 入目录 (root 原版保持不动 per D-02.6); 新写 `18-VERIFICATION.md` via goal-backward retro-verify, status: passed (11/11 must-haves), 174 lines, 39 VERIFIED 标记.
**Rationale:** v1.1-MILESTONE-AUDIT.md gaps_found 标记 phase 18 "missing-directory" — 实际交付物全部落地 main (9 wave commits W1a..W4d 共 9 个原子 commit + 1 post-audit 5d536ec). 证据基于 root 18-SUMMARY (21.7KB) + 10 commits + STATE.md §Phase 18 + 实时代码四源交叉 (per 21-CONTEXT D-02.4). body 为准 per Pitfall 2: 9 wave commits 全列 (W1a e6315ce / W1b 1dbb3b0 / W1c edaa4ae / W2 558f723 / W3 bd84fe2 / W4a 8796ca3 / W4b 3822497 / W4c a182cd6 / W4d 0c018f2), 不只信 frontmatter 的 W3 终点 bd84fe2. 5d536ec 显式标预存证据 (2026-08-02 commit 早于 phase 21 启动 2026-08-03, 不是 phase 21 交付物 per Pitfall 6).
**Outcome:** commit `d76d47d` on main (仅含 .planning/phases/18-*/ 内 3 文件: .gitkeep + 18-SUMMARY.md 副本 + 18-VERIFICATION.md, 未触业务代码); v1.1-MILESTONE-AUDIT.md 的 "phase-18 missing-directory" gap 可标记关闭. 后续 21-03 (phase 19 retro-verify) + 21-04 (REQUIREMENTS.md) + 21-05 (auth:57 fix) 继续.

### 2026-08-03 - Phase 17 retro-verify VERIFICATION.md reconstructed (Phase 21 Plan 01)

**Decision:** 重建 `.planning/phases/17-56-p0-p1-p2/17-VERIFICATION.md` via goal-backward retro-verify; status: passed (7/7 must-haves); 31 VERIFIED 标记; 179 lines.
**Rationale:** v1.1-MILESTONE-AUDIT.md gaps_found 标记 phase 17 "未验证" — 实际目录齐全 (CONTEXT + 4 PLAN + 4 SUMMARY + REVIEWS) 仅缺 VERIFICATION.md. 证据基于已落地的 SUMMARY + 45 git commits (`cf2d248..c04f805`) + 实时代码三源交叉 (per 21-CONTEXT D-02.3). 跨 phase deferred 项去向显式标注: SEC-003b→Phase 18 (live marker at manager.go:132-134) / PERF-003→Phase 19 (42 WithContext sites in video_recording_task_service.go) / HMAC jti→Phase 19 D3 (HLSJtiRecord AutoMigrate at app.go:340).
**Outcome:** commit `2c679f2` on main (仅含 17-VERIFICATION.md, 未触其他 phase 17 文件); v1.1-MILESTONE-AUDIT.md 的 "phase-17 unverified" gap 可标记关闭. 后续 21-02/03 (phase 18/19 retro-verify) + 21-04 (REQUIREMENTS.md) + 21-05 (auth:57 fix) 继续.

### 2026-07-28 - Dashboard mock data removed entirely (Phase 15 Plan 04)

**Decision:** Delete the hardcoded `taskTrendData` mock array and the `任务趋势` Line chart card from the dashboard rather than rebuilding them with real data; ChartsSection now renders only the two real-stats charts (任务状态 Column + 文件类型 Pie).
**Rationale:** Research §4 confirmed the backend has no time-series endpoint for task counts; the only honest path is to remove the fabricated trend surface. Aggregate all-zero check (`taskStats.total + fileStats.total_videos + systemStats.error_count`) drives an empty-state block in StatCards, replacing 13 zero cards that would otherwise look like a failure. Per-card zero checks were rejected because `disk_usage_percent` and `memory_usage_percent` are hardcoded `0.0` with a backend TODO (`dashboard_service.go:199-200`) — per-card zero would always false-positive.
**Outcome:** Dashboard now renders only truthful fields from `/api/v1/dashboard/stats`; D-07.1, D-07.2, D-07.4 satisfied. D-07.3 (StatCards fields) was already verified in Plan 15-01.

### 2026-07-28 - framer-motion 12 /m subpath API correction (Phase 15 Plan 02)

**Decision:** Import `m` and `AnimatePresence` from the main `framer-motion` package, NOT from `framer-motion/m` as research §2 and PLAN 15-02 originally specified.
**Rationale:** framer-motion 12.34.0's `/m` subpath exports only element-named components (`div`, `span`, etc. — 165 exports total) for the strictest per-element tree-shaking. It does NOT export the `m` namespace, `AnimatePresence`, `LazyMotion`, or `MotionConfig`. TypeScript error TS2305 confirmed at runtime + in `node_modules/framer-motion/dist/m.d.ts`. The D-04.4 perf budget (≤6KB gz/position, tree-shake) is still met because: (1) `framer-motion` has `sideEffects: false`, (2) Vite `manualChunks.motion: ['framer-motion']` isolates it into its own chunk, (3) `<LazyMotion strict>` forces `m.*` usage and ensures only the `domAnimation` feature subset loads.
**Outcome:** Plan 15-02 executed with the corrected import path; downstream plans (03 illustrations, 05 NotFound) should use the same main-package import.

### 2026-07-28 - Phase 15 Learnings extracted (extract-learnings workflow)

**Decision:** Run `/gsd-extract-learnings 15-ai` to consolidate Phase 15's 13 commits / 28 files of changes into a structured `15-LEARNINGS.md` after user reported "感觉没有什么很大的变化". Extraction surfaced 8 decisions, 6 lessons, 5 patterns, 4 surprises. Key finding: 60% of changes are foundational (design tokens, motion infrastructure, Playwright config) or conditional (empty/error states, 404 page) — "去 AI 味" 通过删除 mock 数据 + 补空/错/加载态 + 单一品牌色 + 微交互实现，不是"换皮肤"式大改。
**Rationale:** User's "no big change" perception is accurate but misframed — visible deltas are: (1) product name unification across 4 surfaces (录制管理系统 → 录播服务系统), (2) brand color #1890ff → #0F766E teal, (3) deleted dashboard mock trend chart, (4) ~120ms route fade, (5) self-made SVG illustrations on empty/error states, (6) honest 404 page with NotFoundMascot.
**Outcome:** `15-LEARNINGS.md` written to `.planning/phases/15-ai/15-LEARNINGS.md`. Available for future phases to consult.

### 2026-07-28 - framer-motion Easing type does not accept CSS cubic-bezier strings (Phase 15 Plan 02)

**Decision:** Mirror `designTokens.motion.easing.*` CSS strings as `BezierDefinition` 4-tuples inside `motionConfig.ts`.
**Rationale:** framer-motion's `Easing` type (from `motion-utils`) is `EasingDefinition | EasingFunction` where `EasingDefinition = BezierDefinition | 'linear' | 'easeIn' | ...`. `BezierDefinition` is a `[number, number, number, number]` tuple, NOT a CSS `cubic-bezier(...)` string. theme.ts continues to store CSS strings for CSS consumers; motionConfig.ts has a local `easing` object with the corresponding tuples, documented as needing to stay in sync.
**Outcome:** TSC passes cleanly; durations still read from designTokens (single source of truth for ms values).

### 2026-04-20 - VideoPlayerModal Integration

**Decision:** Volume state split into volume (stored) and actualVolume (applied) for mute toggle
**Rationale:** Mute toggle needs to preserve pre-mute volume level for restoration when unmuting; split state allows independent control of UI slider and video element
**Outcome:** Mute toggle with volume preservation integrated into VideoPlayerModal

### 2026-04-20 - Frame-Level Navigation Implementation

**Decision:** Frame time calculated as 1/30 second for standard 30fps videos
**Rationale:** HTML5 video API doesn't provide frame-level seeking; using 1/30s increments provides sufficient precision for slide capture workflows
**Outcome:** Frame navigation hook and component created with browser compatibility detection

### 2026-04-17 - Roadmap Creation

**Decision:** 4-phase structure with external services last
**Rationale:** Build local features first (splitting, local transcription, PPT management) that work without external dependencies, then add cloud services as enhancement
**Outcome:** ROADMAP.md created with 30/30 requirements mapped (100% coverage)

---

## Todos

### Immediate

- [ ] Execute `/gsd-plan-phase 1` to create Phase 1 implementation plan
- [ ] Set up OSS integration development environment
- [ ] Review existing recording infrastructure for snapshot implementation

### Short-term

- [ ] Validate OSS SDK v2 integration patterns
- [ ] Test FFmpeg snapshot extraction without interrupting recording
- [ ] Design file scan trigger mechanism

---

## Session Continuity

### Last Session

- 2026-07-29T07:47Z — Quick task 260729-lr4 (审计 100% + Sanitizer) 6 commits landed
- Session paused via /gsd-pause-work at ~79% context
- Resumed via /gsd-resume-work 2026-07-29T07:50Z

### Stopped At

Quick task 260729-lr4 完成，等待人工生产验证（非阻塞）。
HANDOFF.json 待删除（一次性的）。

### Resume File

`.planning/.continue-here.md` + `.planning/HANDOFF.json`（待消费）

### Next Steps

1. 人工生产验证（参考 `.planning/quick/260729-lr4-100/260729-lr4-SUMMARY.md`）
2. 可选：补 OldData 捕获（service 层 hook）— `/gsd-quick 补 OldData 捕获支持 update/delete 差异对比`
3. 可选：前端 23 个未提交文件整理（与审计无关，独立任务）

---

*STATE.md initialized: 2026-04-17*

**Last Session:** 2026-08-03T03:42:34.869Z
**Last Resume:** 2026-07-29T07:50:11.179Z — /gsd-resume-work consumed HANDOFF.json
**Active context:** Quick task 260729-lr4 — 全部 6 commits on main，handoff 待删除，待人工生产验证

**Planned Phase:** 01 (ppt) — 3 plans — 2026-05-12T06:58:49.592Z

**Session Handoff:** Quick task 260729-lr4 (审计覆盖率 100% + Sanitizer) — 完成。Resume file: `.planning/quick/260729-lr4-100/`

---

## Phase 19 Final Status

**状态**: ✅ 完成 — 11 commits 落地 main（含 9 个代码/测试 commit + 2 个 docs commit）

**范围**:

- ✅ PERF-003/BUG-005 ctx 全量级联
- ✅ SEC-004 jti replay 模型修复（不加 DB 表，TTL sweeper）
- ✅ STYLE-001 error 迁移（mapping.go + HandleError + error_mapper.go）

**最终 HEAD**: `6edb772` (docs: Wave 6 summary + scope 对账)

**DEFERRED** (Phase 19 范围外):

- `taskServiceAdapter` 与 `VideoFileService` 合并
- HMAC jti DB 表（架构 future work）
- 全 `internal/errors` 包 import 迁移增量

**下一步** (用户): 手工验证可选；处理 `<deferred>` 列表中的任何项为独立 phase。

---

## Phase 20 — Context Captured (2026-08-01)

**状态**: 📋 CONTEXT 收齐，Ready for planning

**范围 (3 项目标聚焦)**:

- handler ad-hoc classify 全量清理: 9 文件 27 处 inline 分支 + `classifyAuthLoginError` formal 函数，全部走 `if response.HandleError(c, err) { return }`
- zap logger `sentinel_type` 字段接入: `SentinelField(err)` helper; sentinel → `sentinel_type="ErrXxx"`, BusinessError → `BusinessError(code=yyy)`, unknown → `ad-hoc`
- 自动生成 `docs/errors.md`: go:generate + Makefile check; 列 name | kind | HTTP status | call-site count

**最终决策**:

- Scope: 仅 3 项；typed error kind 字段 deferred
- Classify 替换: 一次性全量扫荡 + 表驱动单测验证状态码不回归
- Service 边界: BusinessError / sentinel `%w` wrap；handler 一律 HandleError
- 不主动迁 cross-package local error var (仅 survey)

**最终交付**:

- `.planning/phases/20-handleerror-classify-convergence/20-CONTEXT.md` (398 行)
- `.planning/phases/20-handleerror-classify-convergence/20-DISCUSSION-LOG.md` (122 行)
- Commit `e2eac56` on main

**下一步**: `/gsd-plan-phase 20` 生成 PLAN.md,或 `/gsd-plan-phase 20 --skip-research` 直接进入 plan
