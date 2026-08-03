---
phase: 20
slug: handleerror-classify-convergence
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-01
---

# Phase 20 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
>
> Derived from `20-RESEARCH.md` §Validation Architecture. Phase 20 is a backend
> Go refactor (error-handling convergence + zap logger + docs generation) — no
> UI, no new external deps. All verification is `go test` + `go build` + smoke.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/stretchr/testify/assert` (already in go.mod) |
| **Config file** | none — Go std convention (`*_test.go` co-located, auto-discovered) |
| **Quick run command** | `go test ./internal/handlers/... ./internal/errors/... ./pkg/response/... -count=1` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | quick ~5s · full ~30s |

---

## Sampling Rate

- **After every task commit (atomic-commit level):** `go build ./... && go test ./internal/handlers/ -count=1`
- **After every plan wave:** `go build ./... && go vet ./... && go test -race ./internal/handlers/... ./internal/errors/... ./pkg/response/...`
- **Before `/gsd:verify-work`:** `go build ./... && go vet ./... && go test -race ./...` all green
- **Max feedback latency:** 30s

---

## Per-Task Verification Map

> Task IDs finalized once the planner emits PLAN.md frontmatter. Rows are keyed
> by requirement until then; planner must map each `REQ-20x` into ≥1 task and
> carry the matching `Automated Command` into that task's `<acceptance_criteria>`.

| Req ID | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|--------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| REQ-20a-classify | classify scatter points (12 files, ~70 sites) replaced by `response.HandleError` | — | N/A (refactor; no authz change) | unit (table-driven) | `go test ./internal/handlers/ -run TestClassifyReplacement -count=1` | ❌ W0 (12 new `_test.go`) | ⬜ pending |
| REQ-20a-formal | `classifyAuthLoginError` deleted; Login routes through `HandleError` | — | N/A | unit (rewrite existing) | `go test ./internal/handlers/ -run TestLogin_HandleError_ClassifyDrop -count=1` | ❌ W0 (rewrite `auth_handler_test.go`) | ⬜ pending |
| REQ-20a-ad-user-not-registered | `auth.ErrADUserNotRegistered` migrated to `internal/errors` (D-02.2 例外), preserves 403 | — | Login rejection stays 403 (not 500) | unit | `go test ./internal/handlers/ -run TestLogin_HandleError_ADUserNotRegistered -count=1` + `go test ./internal/errors/ -count=1` | ❌ W0 | ⬜ pending |
| REQ-20b-sentinel-field | `response.SentinelField(err)` returns correct `zap.Field` for 4 input classes (sentinel / BusinessError / unknown / nil) | — | N/A | unit | `go test ./pkg/response/ -run TestSentinelField -count=1` | ❌ W0 | ⬜ pending |
| REQ-20b-priority | `SentinelField` priority mirrors `IsKnownError` slice order | — | N/A | unit | `go test ./pkg/response/ -run TestSentinelField_Priority -count=1` | ❌ W0 | ⬜ pending |
| REQ-20b-upgrade | `zap.Error(err)` call-sites (~208) zero-intrusion upgrade to add `response.SentinelField(err)` | — | N/A | regression (compile) | `go build ./... && go vet ./...` | ✅ existing | ⬜ pending |
| REQ-20c-generator | `cmd/error-doc-gen` outputs full sentinel table + BusinessError table + missing-filepath error + diff consistency | — | N/A | unit + smoke | `go test ./cmd/error-doc-gen/... -count=1` | ❌ W0 | ⬜ pending |
| REQ-20c-doc-sync | `docs/errors.md` regenerated; `go:generate` + CI step enforces sync (no Makefile) | — | N/A | smoke | `go generate ./internal/errors/... && git diff --quiet docs/errors.md` | ❌ W0 | ⬜ pending |
| REQ-20-regression | 12 phase-17-touched packages pass `go test -race` | — | N/A | regression | `go test -race ./...` | ✅ Phase 17 baseline | ⬜ pending |
| REQ-20-build | `go build ./...` exits 0 after every commit | — | N/A | smoke | `go build ./...` | ✅ baseline green (HEAD 570a2bc) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/handlers/{ppt,input_config,file,video_file,admin,user,transcription,split,role,auth,video_recording_task,apikey}_handleerror_test.go` — one table-driven test per file covering 4 error classes (sentinel / sentinel-wrap / BusinessError / unknown)
- [ ] Rewrite `internal/handlers/auth_handler_test.go::TestClassifyAuthLoginError` → `TestLogin_HandleError_ClassifyDrop` (sync AD config 500→503 + ADUserNotRegistered 403 semantics)
- [ ] `pkg/response/sentinel_field_test.go` — 4 input-class assertions + slice-order priority
- [ ] `cmd/error-doc-gen/main_test.go` — generator 4-case verification
- [ ] `docs/errors.md` — produced by generator (checked in, CI-enforced sync)
- [ ] CI step invoking `go generate ./internal/errors/... && git diff --quiet docs/errors.md` (no Makefile — decision R-2)
- [ ] No new framework dep — `testify/assert` already in go.mod (`auth_handler_test.go:14`)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| AD login end-to-end (real AD server) returns 403 for unregistered user | REQ-20a-ad-user-not-registered | Requires live AD infra + real credentials; unit test mocks the error path | After deploy: attempt login with an AD account that exists in AD but not in local DB; confirm HTTP 403 (not 500) and `sentinel_type=ErrADUserNotRegistered` in logs |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved (post-execution honest verification per Phase 22 plan 22-06; all 6 sign-off items pass against 20-VERIFICATION.md status: passed 10/10 + 20-0[1-5]-SUMMARY.md Task Commits + 12 handleerror_test.go shipped + SentinelField tests shipped + docs/errors.md regen + CI sync-check live)
