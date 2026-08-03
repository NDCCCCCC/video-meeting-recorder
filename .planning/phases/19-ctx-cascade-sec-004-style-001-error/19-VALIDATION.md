---
phase: 19-ctx-cascade-sec-004-style-001-error
slug: ctx-cascade-sec-004-style-001-error
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-03
---

# Phase 19 — Validation Strategy

> Phase 19 已由 `19-VERIFICATION.md` retro-verify 为 passed（10/10 must-haves）。本文件是执行后回填的 Nyquist sampling contract，不声称原执行期间已启用该方法。
>
> 原 `PLAN.md` / `CONTEXT.md` 已永久丢失（见 `19-VERIFICATION.md` Evidence Limitations #1），因此 Per-Task Verification Map 以实际 wave、D1-D4 收尾和 D5-D21 commits 为索引。证据覆盖 32+ commits：11 个 wave/docs commits、4 个 D1-D4 收尾 commits、17 个 D5-D21 commits。`19-SUMMARY.md` frontmatter 的 `cacc294` 仅为 W2 终点；body 真正终点为 `6edb772`，body 为准（Pitfall 2）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/stretchr/testify/assert` (already in go.mod) |
| **Config file** | none — Go std convention (`*_test.go` co-located, auto-discovered) |
| **Quick run command** | `go test ./internal/services/... ./internal/handlers/... ./internal/scheduler/... ./internal/utils/... ./internal/middleware/... ./internal/auth/hlstoken/... ./internal/errors/... ./cmd/server/... -count=1` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | quick ~10s (services + handlers + scheduler + utils + middleware + auth + errors) · full ~30s |

---

## Sampling Rate

- **After every atomic commit:** `go build ./... && go test -count=1 ./internal/<touched-package>/...`
- **After every wave (W0..W6) + D1-D4 + D5-D21:** `go build ./... && go vet ./... && go test -race ./internal/services/... ./internal/handlers/... ./internal/scheduler/...`
- **Before `/gsd:verify-work`:** `go build ./... && go vet ./... && go test -race ./...`
- **Cross-package ctx 取消传播契约:** W5e `b08255d` 验证 5 个接口方法在 pre-cancelled ctx 下传播取消。
- **Max feedback latency:** 30s

---

## Per-Task Verification Map

> Phase 19 没有可恢复的 PLAN tasks；以下 rows 对应 `19-VERIFICATION.md` §Phase 19 Commit Reference / D5-D21 Cross-Reference 的实际交付序列。

| Wave/D# | Task | Commit | Test Command | File Exists | Status |
|----------|------|--------|--------------|-------------|--------|
| W0 | ctx 残留清理：dashboard_service 11 处 + audit_log_service + sm4_token 5 处 | `ad7d0a8` | `go test -count=1 ./internal/services/... ./internal/auth/hlstoken/...` | ✅ | ✅ retro-verified (D19-1) |
| W1 | SEC-004 jti replay：多分片 HLS 修复 (+245/-19) | `6fbdad4` | `go test -run TestVerify_MultiSegmentSameToken ./internal/auth/hlstoken/...` | ✅ `hls_token_test.go:110` | ✅ retro-verified (D19-2) |
| W2 | STYLE-001 error-mapping 三组件：mapping.go + HandleError + error_mapper.go | `cacc294` | `grep -c 'knownSentinels' internal/errors/mapping.go` (=3) + verify `cmd/server/app.go:650` ErrorMapper registration | ✅ | ✅ retro-verified (D19-3) |
| W3 | 13 leaf/mid 服务 ctx 级联，8 atomic commits（含 docs） | `213710c` / `24df855` / `557ffcd` / `a165981` / `3494e61` / `bb2b414` / `a6c21b6` / docs `a3197fa` | `go test -race ./internal/services/... ./internal/middleware/...` | ✅ | ✅ retro-verified (D19-1) |
| W4 | TaskServiceInterface 原子三元组 + docs | `9a00cbe` + `2281927` | `grep 'TaskServiceInterface' internal/scheduler/video_scheduler.go` (≥11 call sites) | ✅ | ✅ retro-verified (D19-1, D19-9) |
| W5 | VideoRecordingTaskService 22 方法 + VideoFileService callers + W5e contract test | `34b07f7` + `e2b0b6b` / `7828fc3` / `7a5a1cc` / `1ae6be0` / `b08255d` | `grep -c '.WithContext(ctx)' internal/services/video_recording_task_service.go` (=42) | ✅ | ✅ retro-verified (D19-1, D19-9) |
| W6 | STYLE-001 error 迁移：gorm wrap → BusinessError + HandleError | `3d171de` | `go test -race ./internal/services/... ./internal/handlers/... ./internal/scheduler/...` | ✅ | ✅ retro-verified (D19-8) |
| Final docs | Wave 6 summary + 范围对账（body 真正终点，非 frontmatter `cacc294`） | `6edb772` | `git log --oneline` and locate `6edb772` | ✅ | ✅ retro-verified (D19-1, EL) |
| D1 | FOREIGN KEY strings.Contains → ErrForeignKeyConstraint sentinel（dual-%w wrap） | `20ee289` | `grep -n 'fmt.Errorf.*%w.*%w' internal/services/video_file_service.go` | ✅ | ✅ retro-verified (D19-10) |
| D2 | taskServiceAdapter 与 VideoRecordingTaskService 合并 (+118/-120) | `3b2d41f` | `grep -c 'taskServiceAdapter' cmd/server/app.go` (=1, comment only) | ✅ | ✅ retro-verified (D19-7) |
| D3 | HLS jti 升级为 hls_jti_records 表 (+198/-8) | `1f0ec35` | `grep -n 'HLSJtiRecord' cmd/server/app.go` (AutoMigrate) | ✅ | ✅ retro-verified (D19-6) |
| D4 | errors 包增量迁移 + DeleteFile/NYCodecryptor 包装 | `f4291f5` | `grep -R 'apperrors.Err\|response.HandleError' internal/services internal/handlers` | ✅ | ✅ retro-verified (D19-5) |
| d5 | user_service + handler (+5 sentinels, 14 散点) | `7a0a7af` | `go test -count=1 ./internal/services/... ./internal/handlers/...` | ✅ | ✅ retro-verified (D19-4, D19-5) |
| d6 | ad_auth + local_auth Login (+4 sentinels, 33 散点) | `da8aaf9` | `go test -count=1 ./internal/auth/... ./internal/handlers/...` | ✅ | ✅ retro-verified (D19-4, D19-5) |
| d7 | sm4_token + middleware (+4 sentinels, 11 散点) | `964fb8f` | `go test -count=1 ./internal/utils/... ./internal/middleware/...` | ✅ | ✅ retro-verified (D19-4, D19-5) |
| d8 | ip_validator utility (0 sentinels, 9 散点) | `c8ed97f` | `go test -count=1 ./internal/utils/...` | ✅ | ✅ retro-verified (D19-5) |
| d9 | role_service + handler (+4 sentinels, 9 散点) | `2a807a6` | `go test -count=1 ./internal/services/... ./internal/handlers/...` | ✅ | ✅ retro-verified (D19-4, D19-5) |
| d10 | apikey_service + handler (+5 sentinels, 11 散点) | `d277f37` | `go test -count=1 ./internal/services/... ./internal/handlers/...` | ✅ | ✅ retro-verified (D19-4, D19-5) |
| d11 | hls_token 复用 D7 (0 sentinels, 6 散点) | `7b0d817` | `go test -count=1 ./internal/auth/hlstoken/...` | ✅ | ✅ retro-verified (D19-5) |
| d12 | auth/service.go (0 sentinels, 4 散点) | `00df988` | `go test -count=1 ./internal/auth/...` | ✅ | ✅ retro-verified (D19-5) |
| d13 | ppt_file_service + handler (+1 sentinel, 1 散点) | `e1dd2dd` | `go test -count=1 ./internal/services/... ./internal/handlers/...` | ✅ | ✅ retro-verified (D19-4, D19-5) |
| d14 | tingwu_client (+1 sentinel, 23 散点) | `1fa66d8` | `go test -count=1 ./internal/services/tingwu/...` | ✅ | ✅ retro-verified (D19-4, D19-5) |
| d15 | storage/file_service (0 sentinels, 22 散点) | `98e14ca` | `go test -count=1 ./internal/services/storage/...` | ✅ | ✅ retro-verified (D19-5) |
| d16 | scheduler + recorder (0 sentinels, 33 散点) | `8eec84a` | `go test -race ./internal/scheduler/... ./internal/recorder/...` | ✅ | ✅ retro-verified (D19-5) |
| d17 | Huawei SDK adapter (0 sentinels, 25 散点) | `cc49867` | `go build ./...` + inspect Huawei SDK adapter error wrapping | ✅ | ✅ retro-verified (D19-5) |
| d18 | utils/sm4_password (0 sentinels, 28 散点) | `71e5be0` | `go test -count=1 ./internal/utils/...` | ✅ | ✅ retro-verified (D19-5) |
| d19 | migrations + models + input_config (0 sentinels, 47 散点) | `ddf047c` | `go test -count=1 ./internal/migrations/... ./internal/models/... ./internal/services/...` | ✅ | ✅ retro-verified (D19-5) |
| d20 | transcription/oss/notification/local_driver/config (0 sentinels, 53 散点) | `ffdc0c6` | `go test -count=1 ./internal/services/transcription/... ./internal/services/oss/... ./internal/services/notification/...` | ✅ | ✅ retro-verified (D19-5) |
| d21 | video_recording_task_service (0 sentinels, 24 散点) | `f358602` | `go test -count=1 ./internal/services/...` | ✅ | ✅ retro-verified (D19-5) |

*Status: ⬜ pending · ✅ retro-verified (post-execution, this VALIDATION.md is retroactive)*

### Commit Coverage Summary

- **Wave evidence:** W0-W6 plus W3/W4/W5 sub-commits and final docs `6edb772` are explicitly enumerated above.
- **Close-out evidence:** D1-D4 commits `20ee289`, `3b2d41f`, `1f0ec35`, and `f4291f5` are each independently mapped.
- **Sentinel migration evidence:** all 17 D5-D21 commits are mapped one-to-one to `19-VERIFICATION.md` Cross-Reference rows.
- **No fabricated hashes:** every prefix above is copied from `19-VERIFICATION.md` §Phase 19 Commit Reference or its D5-D21 table.
- **Authority rule:** where `19-SUMMARY.md` metadata and body differ, the body ending at `6edb772` is authoritative per Pitfall 2.
- **Retroactive status:** green marks historical evidence, not contemporaneous Nyquist execution sampling.
- **Scope note:** phase 19 includes backend Go changes only; no UI sampling or browser automation is required by this contract.
- **Deferred note:** `video_file_service.go:891` diagnostic `strings.Contains` remains intentionally outside this validation goal.

---

## Wave 0 Requirements

- [ ] `internal/services/ctx_cancellation_test.go` — 4 测试函数：`TestRoleService_GetAllPermissions_PreCancelledCtx`、`TestRoleService_ListRoles_PreCancelledCtx`、`TestUserService_GetUserByID_PreCancelledCtx`、`TestUserService_ListUsers_PreCancelledCtx`，全部断言 `errors.Is(err, context.Canceled)`。
- [ ] `cmd/server/taskservice_adapter_ctx_test.go` — `TestVideoRecordingTaskService_CancellationPropagation`，覆盖 5 个接口方法的 pre-cancelled ctx；D2 合并前原名 `TestTaskServiceAdapter_CancellationPropagation`。
- [ ] `internal/auth/hlstoken/hls_token_test.go` — `TestVerify_MultiSegmentSameToken` 核心 SEC-004 修复断言 + 5 子测试。
- [x] `internal/errors/mapping.go::knownSentinels` slice（42 rows，单源）— already covered by phase 20 SentinelField + IsKnownError tests (`20-VERIFICATION.md` Observable Truths #1/#3/#4)。
- [x] `pkg/response/response.go::HandleError` — already covered by 102 call sites / 12 handler files and phase 20 handler tests。
- [x] `internal/middleware/error_mapper.go::ErrorMapper` — already covered by `cmd/server/app.go:650` global registration and middleware tests。
- [x] `internal/models/hls_jti_record.go::HLSJtiRecord` — already covered by AutoMigrate registration (`cmd/server/app.go:340`) + hls_token DB-mode tests。
- [x] `internal/services/video_recording_task_service.go` 42 `.WithContext(ctx)` sites — already covered by grep + cancellation contract test。
- [x] `internal/services/video_file_service.go::createWithDuplicateCheck` dual-%w wrap — already covered by caller `errors.Is` unwrapping and mapping tests。
- [x] 跨 phase 契约 intact：Phase 17 PERF-003/HMAC jti deferred → Phase 19 兑现，见 `19-VERIFICATION.md` D19-1（42 sites）与 D19-6（hls_jti_records）。

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Phase 19 不需要核心人工验证 — 全部 must-haves 程序化验证 | n/a | `19-VERIFICATION.md` §Human Verification Required = None required；可选 HLS 浏览器烟测已由单元测试覆盖 | n/a |
| Phase 19 D3 `hls_jti_records` 表跨实例/重启验证（可选） | 跨 phase 契约 intact | 需要多实例 + live DB | deploy 后两实例并发签发同 jti，仅一个应 verify 通过；重启后 jti 仍标记为 used |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies（32+ commits 全部映射：11 wave/docs + 4 D1-D4 + 17 D5-D21）。
- [x] Sampling continuity: no 3 consecutive tasks without automated verify（每个 commit 都有 build/test 或 grep verification evidence）。
- [x] Wave 0 covers all MISSING references（3 test files + indirect coverage + cross-phase contract）。
- [x] No watch-mode flags（使用 `go test -count=1` / `-race`，无 `-watch`）。
- [x] Feedback latency < 30s（quick scope ~10s）。
- [x] `nyquist_compliant: false` set in frontmatter（retro-fitted post-execution；不虚假声明 pre-execution sampling 已启用）。

**Approval:** pending (retro-fitted post-execution; see `19-VERIFICATION.md` status: passed 10/10)
