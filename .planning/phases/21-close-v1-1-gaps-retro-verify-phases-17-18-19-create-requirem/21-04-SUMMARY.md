---
phase: 21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem
plan: 04
subsystem: docs
tags: [requirements, traceability, milestone, audit, retro-verify]

# Dependency graph
requires:
  - phase: 21-01 (retro-verify phase 17)
    provides: 17-VERIFICATION.md (status: passed, 7/7 must-haves) — REQ-17-* evidence source
  - phase: 21-02 (retro-verify phase 18)
    provides: 18-VERIFICATION.md (status: passed, 11/11 must-haves) — REQ-18-* + REQ-17-SEC-003b cross-phase evidence source
  - phase: 21-03 (retro-verify phase 19)
    provides: 19-VERIFICATION.md (status: passed, 10/10 must-haves) — REQ-19-* + REQ-17-PERF-003/BUG-005 cross-phase evidence source
provides:
  - ".planning/REQUIREMENTS.md — v1.1 milestone (phase 17/18/19/20) REQ-ID traceability table, 5 columns per D-03.3, ~80 REQ-IDs fully covered"
  - "Closes the 'REQUIREMENTS.md missing / orphan detection impossible' gap from v1.1-MILESTONE-AUDIT.md gaps_found"
  - "Coverage section with 4 grep-based orphan detection rules"
  - "Cross-phase兑现 explicit annotation pattern (REQ-17-SEC-003b→18 / REQ-17-PERF-003→19 / REQ-17-BUG-005→19)"
affects: [v1.1-MILESTONE-AUDIT, milestone-completion, future-phase-22-if-phase-16-included]

# Tech tracking
tech-stack:
  added: []  # pure docs — no new libraries/tools
  patterns:
    - "REQ-ID backfill pattern: when SUMMARY frontmatter requirements_completed is empty, derive IDs from deliverables (commits + VERIFICATION.md + live code) and label 'backfilled from deliverables, SUMMARY frontmatter was empty'"
    - "Cross-phase兑现 pattern: keep original phase ID (e.g., REQ-17-SEC-003b) with Phase column '17→18' and 状态 'done (delivered by Phase NN)' — preserves audit finding identity while making cross-phase flow visible"
    - "Dual canonical paths for lost PLAN/CONTEXT: register both root SUMMARY (git-tracked history) and .planning/phases/ copy (self-contained for VERIFICATION co-location)"

key-files:
  created:
    - ".planning/REQUIREMENTS.md — 251 lines, ~80 REQ-IDs, 4 phase sub-sections + Coverage + Out-of-scope observation + Canonical References"
  modified: []

key-decisions:
  - "List every audit finding as its own REQ-17-* row (52 rows total) instead of using range shorthand — gives cleanest audit trail and exceeds the >=40 row floor by 30%"
  - "Record BUG-007/008/009/010/012/013/014 as a single combined N/A row since audit reported 0 occurrences — preserves orphan-detection completeness without bloating the table"
  - "Cross-phase兑现 IDs (REQ-17-SEC-003b / REQ-17-PERF-003 / REQ-17-BUG-005) keep their original phase-17 ID with Phase='17→18' or '17→19' — preserves audit finding identity"
  - "Phase 16 recorded as Out-of-scope observation only, NOT adjudicated (per CONTEXT D-03.4) — left for milestone decision"
  - "Honest backfill labeling: every REQ-18-*/REQ-19-* row marked 'backfilled from deliverables, SUMMARY frontmatter was empty' per CONTEXT D-03.5 (no fabricated traceability)"

patterns-established:
  - "5-column traceability table per CONTEXT D-03.3: REQ-ID | Phase | 来源 | 状态 | 验证证据 — supports both orphan detection and 3-source cross-reference (REQUIREMENTS ↔ ROADMAP ↔ PLAN)"
  - "Orphan detection via 4 grep rules in Coverage section — runnable by any future auditor to verify coverage intact"

requirements-completed: [P21-R4]

# Metrics
duration: 3 min
completed: 2026-08-03
---

# Phase 21 Plan 04: Create REQUIREMENTS.md Summary

**v1.1 milestone REQ-ID traceability table (5 columns, ~80 REQ-IDs across phase 17/18/19/20) with explicit cross-phase兑现 annotations, Coverage/orphan-detection grep rules, and Out-of-scope phase-16 observation — closes the 'REQUIREMENTS.md missing' gap from v1.1-MILESTONE-AUDIT.md**

## Performance

- **Duration:** 3 min
- **Started:** 2026-08-03T01:57:21Z
- **Completed:** 2026-08-03T02:00:12Z
- **Tasks:** 1
- **Files modified:** 1 (created: `.planning/REQUIREMENTS.md`, 251 lines)

## Accomplishments

- Created `.planning/REQUIREMENTS.md` (251 lines) — the v1.1 milestone REQ-ID traceability table closing the audit's "REQUIREMENTS.md missing / orphan detection impossible" gap
- Covered ~80 REQ-IDs across 4 v1.1 phases: 52 REQ-17-* rows (incl. SEC-003a/b split + combined N/A row for BUG-007..014 zero-occurrence findings) + 5 REQ-18-* + 4 REQ-19-* + 11 REQ-20-*
- All 4 cross-phase兑现 flows explicitly annotated in 状态 column: REQ-17-SEC-003b→Phase 18, REQ-17-PERF-003→Phase 19, REQ-17-BUG-005→Phase 19, REQ-17-HMAC-jti→Phase 19 D3 (tracked as REQ-19-jti-db)
- Coverage section with 4 grep-based orphan detection rules — runnable by any future auditor
- Out-of-scope observation section records phase 16 attribution ambiguity without adjudicating (per CONTEXT D-03.4)
- Canonical References section registers all 4 VERIFICATION.md paths + root 18/19-SUMMARY.md (dual-path: git-tracked root + phase-dir copy) + audit reports + PLAN frontmatter + STATE/ROADMAP

## Task Commits

Each task was committed atomically:

1. **Task 1: 写 .planning/REQUIREMENTS.md** - `695b4fe` (docs)

**Plan metadata:** pending (STATE/ROADMAP commit follows this SUMMARY commit)

## Files Created/Modified

- `.planning/REQUIREMENTS.md` — v1.1 milestone REQ-ID traceability table. 5-column rows per CONTEXT D-03.3, grouped by phase (REQ-17-* / REQ-18-* / REQ-19-* / REQ-20-*), with header explaining REQ-ID convention + 3-source cross-reference model, Coverage section with orphan detection rules, Out-of-scope observation for phase 16, and Canonical References catalog. Backfill labeling per D-03.5; cross-phase兑现 annotations per RESEARCH §5 Pitfall 3.

## Decisions Made

- **Listed every audit finding as its own REQ-17-* row (52 total)** instead of using range shorthand (`REQ-17-SEC-005..015` etc.). The plan allowed range shorthand, but individual rows give the cleanest audit trail and exceed the >=40 row floor by 30%, making orphan detection trivial.
- **Recorded BUG-007/008/009/010/012/013/014 as a single combined N/A row** (audit's "0 处" findings). Preserves orphan-detection completeness (every audit finding ID still appears in the table) without bloating it with 7 identical N/A rows.
- **Cross-phase兑现 IDs keep their original phase-17 identity** with `Phase` column showing the flow (e.g., `17→18`) and `状态` column explicitly noting "done (delivered by Phase 18)". Alternative considered: re-issuing as new REQ-18-* IDs — rejected because it breaks audit-finding traceability (the audit said "SEC-003b" not "REQ-18-006").
- **Phase 16 recorded as Out-of-scope observation only, NOT adjudicated** per CONTEXT D-03.4. Decision left for milestone review — if phase 16 is included, a separate phase 22 should backfill REQ-16-* rows (mirroring phase 21's pattern for 17/18/19).
- **Honest backfill labeling** on every REQ-18-* / REQ-19-* row: "backfilled from deliverables, SUMMARY frontmatter was empty" per CONTEXT D-03.5. The alternative — silently filling in — would have violated the no-fake-traceability rule.
- **Dual canonical paths registered** for root `18-SUMMARY.md` / `19-SUMMARY.md` (git-tracked history, kept in place per D-02.6) AND the `.planning/phases/18-*/` / `.planning/phases/19-*/` copies (self-contained phase dirs for VERIFICATION co-location). The 来源 column in REQ-18-* / REQ-19-* rows points to the phase-dir copies so VERIFICATION and SUMMARY sit in the same directory.

## Deviations from Plan

None - plan executed exactly as written. The plan's `<action>` block was prescriptive enough (skeleton + per-row specifications + verify checks) that no creative interpretation was needed. One minor formatting adjustment: the meta block was written with plain key-value lines (`Defined: 2026-08-03`) rather than bold-markdown (`**Defined:**`) because the plan's own `<verify>` step 2 greps for the plain form.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. This is a pure-documentation plan; no runtime, no env vars, no dashboards.

## Next Phase Readiness

- **Phase 21 now 5/5 plans complete** (21-01/02/03/04/05 all landed on main). The phase is fully delivered.
- **All 5 gaps from v1.1-MILESTONE-AUDIT.md can now be marked closed** (when re-running `/gsd:audit-milestone v1.1`):
  1. ✅ REQUIREMENTS.md missing → closed by this plan (commit `695b4fe`)
  2. ✅ Phase 17 unverified → closed by 21-01 (commit `2c679f2`)
  3. ✅ Phase 18 directory missing → closed by 21-02 (commit `d76d47d`)
  4. ✅ Phase 19 directory missing → closed by 21-03 (commit `4b52463`)
  5. ✅ auth_handler.go:57 WARNING → closed by 21-05 (commit `4959e9c`)
- **Re-running `/gsd:audit-milestone v1.1`** is the explicit next step but is OUTSIDE this plan's scope (per CONTEXT `<deferred>` — it's the post-phase-21 acceptance action, not a phase-21 deliverable).
- **Phase 16 attribution decision** remains open (Out-of-scope observation in REQUIREMENTS.md) — milestone-level call, not a phase-21 deliverable.

## Verification Evidence

All 15 `<verify>` checks from 21-04-PLAN.md passed:

| # | Check | Result |
|---|-------|--------|
| 1 | `.planning/REQUIREMENTS.md` exists | PASS |
| 2 | H1 + meta (Defined / Milestone) | PASS |
| 3 | 5-column header `\| REQ-ID \| Phase \| 来源 \| 状态 \| 验证证据 \|` | PASS |
| 4 | REQ-17-* >= 40 rows (actual: 52) + key IDs present (SEC-001/003a/003b/BUG-001/PERF-003/STYLE-001/STYLE-009) | PASS |
| 5 | REQ-18-* >= 5 rows (actual: 5) | PASS |
| 6 | REQ-19-* >= 4 rows (actual: 4) | PASS |
| 7 | REQ-20-* = 11 rows + all 11 IDs present (REQ-20a-classify..REQ-20-typed-kind) | PASS |
| 8 | Cross-phase兑现 explicit (SEC-003b→18, PERF-003→19) | PASS |
| 9 | REQ-20-typed-kind marked deferred | PASS |
| 10 | Coverage + Out-of-scope + Canonical References sections present | PASS |
| 11 | Phase 16 + "不强行裁定" present | PASS |
| 12 | 4 VERIFICATION paths referenced (17/18/19/20) | PASS |
| 13 | Canonical root 18/19-SUMMARY.md paths registered | PASS |
| 14 | >= 120 lines (actual: 251) | PASS |
| 15 | Commit subject = `docs(21): create REQUIREMENTS.md — v1.1 REQ-ID traceability` + no business code in commit | PASS |

## REQ-ID Counts (by phase)

| Phase | Rows | Notes |
|-------|------|-------|
| REQ-17-* | 52 | 16 SEC (with SEC-003a/b split) + 10 BUG (9 individual + 1 combined N/A for 7 zero-occurrence findings) + 16 PERF + 10 STYLE |
| REQ-18-* | 5 | All backfilled from deliverables (SUMMARY frontmatter was empty) |
| REQ-19-* | 4 | High-level IDs (ctx / SEC-004 / STYLE-001 / jti-db) — 21 dN commits provide finer-grained evidence under each |
| REQ-20-* | 11 | 10 done + 1 deferred (REQ-20-typed-kind per 20-CONTEXT D-01.1) |
| **Total** | **72 rows** | ~80 REQ-IDs counting the 7 N/A findings combined into 1 row |

## Cross-phase兑现 Catalog

| REQ-ID | Origin | Delivered By | Evidence |
|--------|--------|--------------|----------|
| REQ-17-SEC-003b | Phase 17 (deferred per CONTEXT `<deferred>`) | Phase 18 | 18-VERIFICATION D18-1..D18-5 + commits e6315ce/5d536ec |
| REQ-17-PERF-003 | Phase 17 (partial — only modified/new sites) | Phase 19 (full ~190 methods) | 19-VERIFICATION D19-1 + 42 WithContext sites |
| REQ-17-BUG-005 | Phase 17 (same root cause as PERF-003) | Phase 19 | 19-VERIFICATION D19-1 |
| REQ-17-HMAC-jti (in-memory map) | Phase 17 (architectural decision) | Phase 19 D3 (DB-backed) | REQ-19-jti-db + commit 1f0ec35 |

## Orphan Detection Result

**Orphans: 0 ✓**

- Every audit finding in `docs/audits/2026-07-30-backend-code-review.md` has a matching REQ-17-* row (57 distinct finding IDs + SEC-003 split = 58 covered, with 7 zero-occurrence BUG IDs combined into 1 N/A row = 52 visible REQ-17-* rows)
- Every 18-VERIFICATION must_have maps to a REQ-18-* ID (5/5)
- Every 19-SUMMARY high-level deliverable maps to a REQ-19-* ID (4/4)
- Every 20-01/20-05 PLAN frontmatter REQ-20-* ID is present (11/11, incl. deferred REQ-20-typed-kind from 20-CONTEXT D-01.1)

## Self-Check: PASSED

- `.planning/REQUIREMENTS.md` exists on disk (251 lines, commit `695b4fe`) — FOUND
- `.planning/phases/21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem/21-04-SUMMARY.md` exists (commit `a00a1a3`) — FOUND
- Plan task commit `695b4fe` (`docs(21): create REQUIREMENTS.md — v1.1 REQ-ID traceability`) on main — FOUND
- SUMMARY metadata commit `a00a1a3` (`docs(21-04): complete REQUIREMENTS.md plan`) on main — FOUND
- STATE/ROADMAP metadata commit `69720a0` on main — FOUND
- All 15 plan `<verify>` automated checks PASS (see "Verification Evidence" section above)
- All 13 plan `<acceptance_criteria>` PASS (covered by the 15 verify checks; 1:1 mapping)
- Reverse assertion PASS: commit `695b4fe` contains ONLY `.planning/REQUIREMENTS.md` — no business code, no `docs/audits/` mutation, no other `.planning/` files affected

---

*Phase: 21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem*
*Plan: 04 (Wave 2)*
*Completed: 2026-08-03*
