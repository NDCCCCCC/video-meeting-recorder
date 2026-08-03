---
phase: 21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem
slug: close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-03
---

# Phase 21 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
>
> **Retro-fitted** post-execution per phase 21 retro-verify methodology. Original phase 21 execution predates Nyquist adoption; this file declares the sampling contract as it WOULD have been specified given the 5 plans (21-01/02/03/04/05) + 5 plan-commits (`2c679f2` / `d76d47d` / `4b52463` / `695b4fe` / `4959e9c`) + 8 metadata/state commits now on disk. Status remains `nyquist_compliant: false` to preserve honest provenance: this VALIDATION.md is documentation, not pre-execution sampling.

Phase 21 is a process-closure phase (per 21-VERIFICATION.md status: **passed 4/4 must-haves D-05.1..D-05.4**, retro-verified 2026-08-03): 4 plans are pure documentation (21-01/02/03 retro-verify VERIFICATION.md for phases 17/18/19 + 21-04 REQUIREMENTS.md creation), and only 21-05 modifies code (1-line canonical HandleError pattern fix in `auth_handler.go:57`). Phase 21 has the most complete documentation of all v1.1 phases (CONTEXT + RESEARCH + REVIEW + 5 PLANs + 5 SUMMARIES + VERIFICATION), so the Per-Task Verification Map can enumerate each plan's single task with its shipped commit hash directly.

Companion to `21-VERIFICATION.md` (status: **passed 4/4 must-haves**, retro-verified 2026-08-03 by Claude/gsd-verifier). VERIFICATION reports outcome; VALIDATION declares the pre-execution sampling contract retro-fitted to what shipped.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/stretchr/testify/assert` (already in go.mod) |
| **Config file** | none — Go std convention (`*_test.go` co-located, auto-discovered) |
| **Quick run command** | `go test -race ./internal/handlers/... -count=1` (phase 21 仅 21-05 修改代码, 仅触达 handlers 包; 其他 4 plans 是纯文档) |
| **Full suite command** | `go build ./... && go test -race ./...` (验证 21-05 改动不破坏全库) |
| **Estimated runtime** | quick ~3.5s (handlers race test per 21-VERIFICATION D-05.3) · full ~30s |

---

## Sampling Rate

- **After every atomic commit (atomic-commit level):**
  - docs plans (21-01/02/03/04): 文件存在 + frontmatter 字段 + 内容完整性 (`grep -E 'status: passed'`)
  - code plan (21-05): `go build ./... && go test -race ./internal/handlers/...`
- **After every plan wave:**
  - 21-01/02/03 (Wave 1 — three retro-verify docs plans): `go build ./... && go vet ./...` (sanity, 不触达代码改动)
  - 21-04 (Wave 2 — REQUIREMENTS.md): `go build ./... && go vet ./...` (sanity)
  - 21-05 (Wave 1 — single auth:57 fix): `go build ./... && go test -race ./internal/handlers/...`
- **Before `/gsd:verify-work`:** `go build ./... && go vet ./... && go test -race ./...`
- **Cross-AI review trigger:** 21-REVIEW.md post-execution review (per 21-CONTEXT D-06 cross-AI review for auth:57 fix 论据正确性)
- **Max feedback latency:** 30s

---

## Per-Task Verification Map

> Each row maps a phase-21 plan's single task to its shipped commit hash + automated verify command. All commit hashes 抄录 from `21-0[1-5]-SUMMARY.md` §"Task Commits" tables — no fabrication. Phase 21 has 5 plans, each with a single task per their respective SUMMARY.

| Plan | Task | Commit | Test Command | File Exists | Status |
|------|------|--------|--------------|-------------|--------|
| 21-01 | Task 1: phase 17 retro-verify — 重建 17-VERIFICATION.md (180 行, status: passed 7/7, 31 VERIFIED) | `2c679f2` | `grep -E 'status: passed\|score: 7/7' .planning/phases/17-56-p0-p1-p2/17-VERIFICATION.md` + `grep -c 'VERIFIED' .planning/phases/17-56-p0-p1-p2/17-VERIFICATION.md` (= 31) | ✅ (.planning/phases/17-56-p0-p1-p2/17-VERIFICATION.md, 180 行) | ✅ retro-verified (D-05.1) |
| 21-02 | Task 1: phase 18 retro-verify — 重建目录 (.gitkeep + 18-SUMMARY.md 副本 + 18-VERIFICATION.md 174 行) | `d76d47d` | `ls .planning/phases/18-credential-static-encryption-sec-003b/` (= 3 files) + `grep -E 'status: passed\|score: 11/11' 18-VERIFICATION.md` + `diff -q root/副本 18-SUMMARY.md` IDENTICAL | ✅ (.planning/phases/18-credential-static-encryption-sec-003b/, 3 文件) | ✅ retro-verified (D-05.1) |
| 21-03 | Task 1: phase 19 retro-verify — 重建目录 (.gitkeep + 19-SUMMARY.md 副本 + phase-19-D5-D21-summary.md 副本 + 19-VERIFICATION.md 220 行) | `4b52463` | `ls .planning/phases/19-ctx-cascade-sec-004-style-001-error/` (= 4 files) + `grep -E 'status: passed\|score: 10/10' 19-VERIFICATION.md` + `diff -q root/副本 19-SUMMARY.md + D5-D21` IDENTICAL | ✅ (.planning/phases/19-ctx-cascade-sec-004-style-001-error/, 4 文件) | ✅ retro-verified (D-05.1) |
| 21-04 | Task 1: REQUIREMENTS.md 创建 (251 行, ~80 REQ-IDs, 5 列结构) | `695b4fe` (task) + `a00a1a3` (SUMMARY) | `wc -l .planning/REQUIREMENTS.md` (= 251) + `grep -q 'REQ-17-' .planning/REQUIREMENTS.md` 等 11 REQ-20 IDs 全覆盖 + `grep -E 'TBD\|FIXME\|XXX' .planning/REQUIREMENTS.md` (= 0) | ✅ (.planning/REQUIREMENTS.md, 251 行) | ✅ retro-verified (D-05.2) |
| 21-05 | Task 1: auth_handler.go:57 canonical HandleError pattern fix (5 行 → 2 行, line 60 "兜底" comment 删, line 53-56 mapping.go comment 保留) | `4959e9c` | `go build ./...` (exit 0) + `go test -race ./internal/handlers/...` (ok 3.356s) + `go test -run TestLogin_HandleError_ClassifyDrop -v` (≥ 10 PASS) + `grep -c 'if response.HandleError' Login range (= 0)` + `! grep -q '兜底' auth_handler.go` + `git log -1 --format=%B \| grep -q 'Behavior-equivalent'` + `! grep -qi 'always returns true' auth_handler.go` | ✅ (internal/handlers/auth_handler.go, 1 insertion / 4 deletions) | ✅ retro-verified (D-05.3) |

*Status legend: ⬜ pending · ✅ retro-verified (post-execution, this VALIDATION.md is retroactive) · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

> Each shipped artifact from phase 21 plans is enumerated with its source plan. All artifacts marked `[x]` are covered by their respective phase-21 plans (per 21-VERIFICATION.md Required Artifacts + 21-0[1-5]-SUMMARY.md). No new files exist beyond what is listed.

- [x] `.planning/phases/17-56-p0-p1-p2/17-VERIFICATION.md` — phase 17 retro-verify (status: passed 7/7, 31 VERIFIED, 180 行) — covered by 21-01
- [x] `.planning/phases/18-credential-static-encryption-sec-003b/18-VERIFICATION.md` — phase 18 retro-verify (status: passed 11/11, 39 VERIFIED, 174 行) + dir reconstruction — covered by 21-02
- [x] `.planning/phases/18-credential-static-encryption-sec-003b/18-SUMMARY.md` — phase 18 SUMMARY 副本 (root + dir) IDENTICAL — covered by 21-02 (diff -q pass)
- [x] `.planning/phases/19-ctx-cascade-sec-004-style-001-error/19-VERIFICATION.md` — phase 19 retro-verify (status: passed 10/10, 41 VERIFIED, 220 行) + dir reconstruction — covered by 21-03
- [x] `.planning/phases/19-ctx-cascade-sec-004-style-001-error/19-SUMMARY.md` — phase 19 SUMMARY 副本 (root + dir) IDENTICAL — covered by 21-03 (diff -q pass)
- [x] `.planning/phases/19-ctx-cascade-sec-004-style-001-error/phase-19-D5-D21-summary.md` — D5-D21 增量 summary 副本 (docs/audits/ + dir) IDENTICAL — covered by 21-03 (diff -q pass)
- [x] `.planning/REQUIREMENTS.md` — v1.1 milestone REQ-ID 追溯表 (251 行, ~80 REQ-IDs, 5 列结构, orphan 0) — covered by 21-04
- [x] `internal/handlers/auth_handler.go` — line 57-61 → 57-58 canonical pattern fix + line 60 "兜底" 注释删 + line 53-56 mapping.go 注释保留 — covered by 21-05 (TestLogin_HandleError_ClassifyDrop 10 sub-tests 回归网 + go build/test 全绿)
- [x] 跨 phase 契约 intact (phase 17→18/19 cross-phase deferred 兑现): 见 17/18/19-VERIFICATION.md Deferred 段 + REQUIREMENTS.md REQ-17-SEC-003b / REQ-17-PERF-003 / REQ-17-BUG-005 / REQ-17-HMAC-jti 行 (4 项跨 phase 兑现显式标注)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Phase 21 不需要核心人工验证 — 全部 must-haves 程序化验证 | n/a | Per 21-VERIFICATION §Human Verification Required = 无 — 本阶段为过程产物补齐 + 1 行代码规范化, 全部行为程序化验证 (go build / go test -race / grep patterns 全 PASS) | n/a |
| auth:57 行为等价性 — 由 cross-AI review 21-REVIEW.md 独立验证 | D-05.3 控制流论据正确性 | commit body 论据需 cross-AI reviewer 复核 | 21-REVIEW.md status: clean 0 findings (per 21-VERIFICATION §D-05.3 + Anti-Patterns Found) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (5 plan commits 全部 mapping 到 Per-Task Verification Map: 21-01/02/03 docs, 21-04 docs, 21-05 code+test)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (每个 plan 都有文件存在 + 内容完整性 或 build/test 验证 per 21-VERIFICATION Behavioral Spot-Checks)
- [x] Wave 0 covers all MISSING references (4 retro-VERIFICATION.md + 1 REQUIREMENTS.md + 1 code fix = 6 主 artifact, 加 3 cross-phase 契约 + 2 SUMMARY 副本 = 9 checkbox)
- [x] No watch-mode flags (per 21-VERIFICATION Behavioral Spot-Checks — 使用 `go test -count=1 -race`, 无 `-watch`)
- [x] Feedback latency < 30s (handlers race test 3.356s per 21-VERIFICATION D-05.3; full ~30s)
- [x] `nyquist_compliant: false` set in frontmatter (draft 状态保留 — phase 21 retro-verify 已 passed, 但 nyquist_compliant 仅在 pre-execution 启用 sampling 时置 `true`; 本 VALIDATION.md 是 retro-fitted, 故保留 `false` — 见 file header > 注)

**Approval:** pending (retro-fitted post-execution; see `21-VERIFICATION.md` status: passed 4/4 must-haves D-05.1..D-05.4)
