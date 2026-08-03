---
phase: 17-56-p0-p1-p2
slug: 56-p0-p1-p2
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-03
---

# Phase 17 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
>
> **Retro-fitted** post-execution per phase 21 retro-verify methodology (21-CONTEXT D-02.1/D-02.2). Original phase 17 execution predates Nyquist adoption; this file declares the sampling contract as it WOULD have been specified given the 56 findings + 4 plans + 45 atomic commits (`cf2d248..c04f805`) + 12 shipped test files now on disk. Status remains `nyquist_compliant: false` to preserve honest provenance: this VALIDATION.md is documentation, not pre-execution sampling.

Phase 17 is a backend Go refactor (56 findings — 13 HIGH + 18 MEDIUM + 25 LOW — from `docs/audits/2026-07-30-backend-code-review.md`) across 4 waves (P0/P1a/P1b/P2). Per 17-CONTEXT D-04.1/D-04.2: P0 + P1a ship with unit tests (D-04.1: 6 test files; D-04.2: 10 test files); P1b ships with unit tests + compile-level interface assertions (3 STUBs); P2 skips new unit tests (D-04.3) and only updates one existing test (`TestQuotaExceeded`) for SEC-012 MIME compatibility. All verification is `go test` + `go build` + `go vet` + `gofmt -l` + targeted `grep` checks.

Companion to `17-VERIFICATION.md` (status: **passed 7/7 must-haves**, retro-verify 2026-08-03 by phase 21 plan 21-01, commit `2c679f2`). VERIFICATION reports outcome; VALIDATION declares the pre-execution sampling contract retro-fitted to what shipped.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/stretchr/testify/assert` (already in go.mod) |
| **Config file** | none — Go std convention (`*_test.go` co-located, auto-discovered) |
| **Quick run command** | `go test ./internal/config/... ./internal/huawei/... ./internal/auth/hlstoken/... ./internal/services/... ./internal/handlers/... ./internal/middleware/... ./internal/migrations/... ./internal/recorder/... ./internal/models/... ./cmd/server/... -count=1` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | quick ~10s · full ~30s (12 phase-17-touched packages with race detection) |

---

## Sampling Rate

- **After every atomic commit (atomic-commit level):** `go build ./... && go test -count=1 ./internal/<touched-package>/...`
- **After every plan wave (17-01 / 17-02 / 17-03 / 17-04):** `go build ./... && go vet ./... && go test -race ./internal/config/... ./internal/huawei/... ./internal/auth/hlstoken/... ./internal/services/... ./internal/handlers/...`
- **Before `/gsd:verify-work`:** `go build ./... && go vet ./... && go test -race ./...`
- **Cross-AI review trigger:** 17-REVIEWS.md post-execution review (per 17-VERIFICATION Evidence Limitations; opencode single reviewer via GitHub Copilot, risk_assessment: LOW, 13 deviations all accepted)
- **Max feedback latency:** 30s

---

## Per-Task Verification Map

> Each row maps a plan's atomic commits to the corresponding automated test command.
> All commit hashes抄录 from `17-0[1-4]-SUMMARY.md` §"Commits" tables — no fabrication.

| Plan | Task | Commit | Test Command | File Exists | Status |
|------|------|--------|--------------|-------------|--------|
| 17-01 | Task 1 SEC-001/002/004 + Task 2 SEC-003a + Task 3a BUG-001/002 + PERF-001/002 + Task 3b PERF-004/005 | `4d3de0b` / `2bcee29` / `47ef805` / `4fc1d3c` | `go test -race ./internal/config/... ./internal/huawei/... ./internal/auth/hlstoken/...` | ✅ | ✅ retro-verified |
| 17-02 | 12 atomic commits (BUG-003..006 + SEC-005..010 + STYLE-004/005 + regression `b53cc8c`) | `d27903f` / `71ec912` / `a815354` / `75e4c91` / `47b4cc3` / `e4b8716` / `39abbc6` / `c12ac8b` / `93ea6ef` / `48cce67` / `e040d2d` / `b53cc8c` | `go test -race ./internal/middleware/... ./internal/models/... ./internal/migrations/... ./internal/services/storage/... ./internal/handlers/... ./internal/huawei/... ./internal/recorder/...` | ✅ | ✅ retro-verified |
| 17-03 | 7 atomic commits (PERF-006..011 + STYLE-003 interface re-placement) | `9150e95` / `679f916` / `43e550e` / `0dcb97a` / `bbce57b` / `c2ada57` / `0190f83` | `go test -race ./internal/services/...` + compile-level interface assertions in `video_scheduler_test.go` / `file_service_test.go` / `service_test.go` | ✅ | ✅ retro-verified |
| 17-04 | 17 atomic commits (BUG-011/015/016 + SEC-011..015 + PERF-012..016 + STYLE-001 partial/006/007/008/010 + gofmt + TestQuotaExceeded compatibility) | `4f5579a` / `3babd0f` / `0c5d6d3` / `3240d09` / `6687815` / `4b60f3a` / `3719c67` / `374e38f` / `0bcc8a4` / `b8df186` / `e11f392` / `f776c2d` / `c0c756a` / `1cbb6b6` / `b1493a9` / `778c357` / `72e2027` | `go test -race ./internal/services/storage/...` + touched-package regression (P2 跳过单测 per D-04.3, 仅 `TestQuotaExceeded` 兼容性更新) | ✅ | ✅ retro-verified |
| housekeeping | gofmt cleanup of 2 Wave-1 cross-wave leftovers | `c04f805` | `gofmt -l ./internal/... ./cmd/...` | ✅ | ✅ retro-verified |

*Status legend: ⬜ pending · ✅ retro-verified (post-execution, this VALIDATION.md is retroactive) · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

> Each shipped test file from phase 17 plans is enumerated with its finding ID and source plan. Files marked `[x]` are covered by downstream phases (cross-phase fulfilment evidence).

- [ ] `internal/config/config_test.go` — SEC-001/008 (P0 + P1a extension)
- [ ] `internal/auth/hlstoken/hls_token_test.go` — SEC-004 jti + RawURLEncoding
- [ ] `internal/huawei/manager_test.go` — SEC-003a TLS policy + ParseMinTLSVersion
- [ ] `internal/services/video_recording_task_service_test.go` — BUG-001 RetryTask + PERF-001 N+1
- [ ] `internal/middleware/auth_test.go` — STYLE-004/005 + SEC-010
- [ ] `internal/models/audit_log_test.go` — BUG-003 JSON parser
- [ ] `internal/models/api_key_test.go` — BUG-004 IP whitelist fail-closed
- [ ] `internal/migrations/013_add_ad_fields_test.go` — SEC-005 idempotent migration
- [ ] `internal/services/storage/file_service_test.go` — SEC-006 SHA-256 + STYLE-003 W9 StorageDriver compile assertion + `TestQuotaExceeded` P2 compatibility
- [ ] `cmd/server/cors_test.go` — SEC-007 exact-allowlist + default-deny
- [ ] `internal/handlers/file_handler_test.go` — SEC-009 token masking
- [ ] `internal/huawei/client_test.go` — SEC-009 response sanitization + PERF-006 Stop+WaitGroup (extended from P0/P1a)
- [ ] `internal/recorder/coordinator_test.go` — BUG-006 ctx-cancel reconnect + PERF-010 atomic ReconnectCount + PERF-011 HLSListSize fallback (extended from P0)
- [x] Cross-phase fulfilment (deferred → downstream): `internal/services/credential_encryptor_test.go::TestRepeatedRotation_V1ToV2ToV3` (phase 18 W4b, path: `internal/services/credential_encryptor_test.go:660`) — closes phase 17 deferred SEC-003b
- [x] Cross-phase fulfilment (deferred → downstream): `internal/auth/hlstoken/hls_token.go::usedJTIs` map → `hls_jti_records` table (phase 19 D3, path: `cmd/server/app.go:340` AutoMigrate registration) — closes phase 17 deferred HMAC jti replay
- [x] Cross-phase fulfilment (deferred → downstream): `internal/services/video_recording_task_service.go` 42 `.WithContext(ctx)` sites (phase 19 W3-W5 + 21 dN commits) — closes phase 17 deferred PERF-003 partial

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Phase 17 不需要人工验证 — 全部 must-haves 程序化验证 | n/a | Per 17-VERIFICATION §Human Verification Required = None required; retro-verify 基于 live code, 无需人工 | n/a |
| Cross-phase: phase 17 HMAC jti → phase 19 D3 `hls_jti_records` 表已落地 | 跨 phase 契约 intact | 需要 DB 起来 + 跨实例重启验证 | Deploy 后用两实例并发签发同 jti, 仅一个应 verify 通过 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (45 commits 全部 mapping 到 Per-Task Verification Map; 4 plans + 1 housekeeping row)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (每个 commit 都有 build/test 验证 per 17-0[1-4]-SUMMARY 验证摘要; commit-level `go build ./... && go test -count=1 ./internal/<touched-package>/...` per 17-CONTEXT 节奏)
- [x] Wave 0 covers all MISSING references (12 phase-17-touched package test files + 3 cross-phase fulfilment evidence checkboxes)
- [x] No watch-mode flags (per 17-0[1-4]-SUMMARY 验证摘要 — 使用 `go test -count=1` 与 `go test -race`, 无 `-watch`)
- [x] Feedback latency < 30s (full suite ~30s per Estimated runtime table)
- [x] `nyquist_compliant: false` set in frontmatter (draft 状态保留 — phase 21 retro-verify 已 passed, 但 `nyquist_compliant` 仅在 pre-execution 启用 sampling 时置 `true`; 本 VALIDATION.md 是 retro-fitted, 故保留 `false` — 见 file header > 注)

**Approval:** pending (retro-fitted post-execution; see `17-VERIFICATION.md` status: passed 7/7)