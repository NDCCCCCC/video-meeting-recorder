---
phase: 17
reviewers: [opencode]
reviewer_models: [github-copilot-via-opencode-1.14.20]
reviewed_at: 2026-07-30T16:27:00Z
plans_reviewed:
  - .planning/phases/17-56-p0-p1-p2/17-01-PLAN.md
  - .planning/phases/17-56-p0-p1-p2/17-02-PLAN.md
  - .planning/phases/17-56-p0-p1-p2/17-03-PLAN.md
  - .planning/phases/17-56-p0-p1-p2/17-04-PLAN.md
summaries_reviewed:
  - .planning/phases/17-56-p0-p1-p2/17-01-SUMMARY.md
  - .planning/phases/17-56-p0-p1-p2/17-02-SUMMARY.md
  - .planning/phases/17-56-p0-p1-p2/17-03-SUMMARY.md
  - .planning/phases/17-56-p0-p1-p2/17-04-SUMMARY.md
audit_source_of_truth: docs/audits/2026-07-30-backend-code-review.md
execution_base: cf2d248 (planning) .. 3b10afa (phase 17 complete HEAD)
execution_commits: 45 (41 code/test + 3 docs/state + 1 housekeeping)
runtime: claude-code (CLAUDE_CODE_ENTRYPOINT=cli — self skipped)
unavailable_reviewers: [gemini, codex, coderabbit, qwen, cursor, ollama, lm_studio, llama_cpp]
context: post-execution retrospective (phase 17 fully executed and verified on main)
reviewer_count: 1
risk_assessment: LOW (per opencode review §7)
---

# Cross-AI Plan Review — Phase 17 (POST-EXECUTION)

## Reviewers Invoked

- **opencode** (v1.14.20, via GitHub Copilot models) — sole external reviewer
- **claude** (current runtime, `CLAUDE_CODE_ENTRYPOINT=cli`) — skipped for independence (per GSD review workflow's environment detection)
- **gemini / codex / coderabbit / qwen / cursor / ollama / lm-studio / llama.cpp** — not installed/available on host

Reviewer availability was limited to a single external system (opencode), reducing cross-AI consensus strength. The opencode review (below) stands as the sole independent assessment.

**Phase:** 17 — 56 Go backend code review findings (P0/P1a/P1b/P2)
**Executed:** 45 commits (`cf2d248..3b10afa`), 80 files, ~4053 insertions
**Status:** Fully verified on `main` — `go build ./...` OK, `go vet ./...` quiet, 12 packages `go test -race` all PASS

---

## 1. Summary

Phase 17 plans are **exceptionally thorough** — likely the highest-quality GSD plans in this project so far. Each plan includes task-level `read_first` guidance, grep-verify acceptance criteria, threat models, no-go lists, and cross-wave dependency tracking. The four-wave P0→P1a→P1b→P2 decomposition was appropriate: P0 eliminated the 5 worst attack chains, P1a covered error-handling + security hardening, P1b handled performance + interface cleanup, and P2 swept remaining LOW items. Execution fidelity was high: all 56 audit findings were correctly triaged, with 50 remediated in code, 3 deferred (SEC-003b, STYLE-001 full, STYLE-009), 1 benign deferral (PERF-003 full sweep), 1 false positive (STYLE-002), and 1 judgment-based skip (P2 STYLE-009). However, the plans contained several **systematic defects** — notably assuming field types/signatures without code verification, assuming GORM call presence without confirmation, and under-estimating import cycles for interface moves — that drove most of the 13 documented deviations.

---

## 2. Plan Quality Strengths

- **Acceptance criteria as grep assertions** — Every task has `grep -c "..." file` style checks that are machine-verifiable, tightening the feedback loop between plan and execution (e.g., 17-01-PLAN.md:214-225 has 11 grep checks for SEC-001/002/004). This pattern should be standard across all GSD phases.

- **Threat model per plan** — Each plan includes a STRIDE threat register specific to its findings (e.g., 17-01-PLAN.md:521-549 maps T-17-01 through T-17-11 with mitigations). This forces the executor to reason about attack surfaces, not just mechanically apply diffs.

- **Wave dependency tracking** — `depends_on: [17-01]` / `[17-02]` / `[17-03]` ensures P0 must complete before P1a, etc. The wave structure prevented cascading conflicts (e.g., STYLE-004 GetUserID signature change in P1a depended on SEC-001 config changes in P0 being stable).

- **No-go lists** — Every task has an explicit "do not" section (e.g., 17-01-PLAN.md:205-208, 282-285, 419-424). This prevents scope creep that would have been catastrophic for a 56-finding phase.

- **Deferral documentation** — The `<deferred>` items are called out both in CONTEXT.md and in every downstream plan (e.g., SEC-003b, STYLE-001 full, STYLE-009, PERF-003 full sweep). The deferral traceability prevents ambiguity about what's intentionally out of scope.

- **Pathological case coverage in P0** — 17-01-PLAN.md SEC-001 task explicitly tests the `SM4_SECRET == HLS_TOKEN_SECRET` same-secret rejection case (acceptance criterion line 193), which is a subtle multi-factor security failure the audit flagged.

- **Testing discipline tiering** (D-04.1/4.2/4.3) — P0 must-have tests, P1 at-least-one test, P2 no tests. This risk-proportional test investment is sane.

- **Commit message hygiene** — Every commit body references the finding IDs it addresses (D-02.2), enabling automated traceability from git log to audit document.

---

## 3. Plan Quality Concerns

### HIGH

#### 1. Pre-plan code assumptions without verification (3 instances)

The plans consistently assumed field types, method signatures, and GORM call presence without reading the actual codebase at plan-writing time. This is a **systematic planning defect**:

- **17-01-PLAN.md:265-266** — `MinTLSVersion uint16` field type specified → actual code had `MinTLSVersion string` (`internal/config/config.go:140`). The plan's acceptance criterion `grep -c "MinTLSVersion.*cfg\\.Huawei"` was still satisfiable, but the type mismatch forced a `ParseMinTLSVersion` wrapper that the plan didn't anticipate.

- **17-01-PLAN.md:388-393** — PERF-003 requires `grep -c ".WithContext(ctx)" ≥ 10` in `video_recording_task_service.go` → actual file has **zero** context.Context parameters on its ~30 service methods. The full 403-site GORM `WithContext` sweep is a method-signature-level problem, not a per-site patch problem. The plan should have flagged this during pre-plan code inspection.

- **17-02-PLAN.md:254-259** — BUG-005 specifies 5 files needing `WithContext(ctx)` → actual audit shows only `audit_log_service.go` has both GORM calls AND ctx on signatures. The other 4 (`middleware/audit.go`, `middleware/auth.go`, `video_recording_task_service.go`, `usb_device_scanner.go`) either have no GORM calls (`auth.go`, `usb_device_scanner.go`) or have no ctx on method signatures (`video_recording_task_service.go`). The plan's `grep -c ".WithContext(ctx)" ≥ 1` acceptance for all 5 files was **impossible to satisfy** for 4 of them.

**Recommendation:** Pre-plan code spot-checks should verify: (a) field types exist and match expectations, (b) method signatures accept the parameters the fix requires, (c) GORM calls actually exist in targeted files. This can be a 30-second `grep -n` step during plan authoring.

#### 2. SEC-004 encoding fallback plan had wrong old encoding

**17-01-PLAN.md:183** specifies fallback to `base64.StdEncoding`. Actual old code used `base64.URLEncoding` (with padding). If the plan's StdEncoding-only fallback had been followed literally, **all existing tokens would break on restart** — violating D-03.3's backward-compat commitment. The executor correctly discovered this during implementation and added a 3-encoding chain (RawURLEncoding → URLEncoding → StdEncoding per SUMMARY.md D2). But the plan's own acceptance criterion (`grep -c "base64.StdEncoding" ≥ 1`) was still satisfied, meaning the plan-automation would have green-lit a correctness bug.

**Recommendation:** Plan acceptance criteria should include a **semantic verification step** beyond grep — e.g., "generate a token with the old code, restart, verify the token still validates." Or at minimum, the plan should read the signing code before specifying the fallback encoding.

#### 3. STYLE-003 import cycle not analyzed in plan

**17-03-PLAN.md:226-231** — The plan specifies compile-level assertions `var _ scheduler.ConversionService = (*services.FFmpegConversionService)(nil)` in consumer packages. But `scheduler` importing `services` creates a cycle: `services → scheduler → services`. The executor discovered this during implementation and switched to stub-based assertions (SUMMARY.md D2).

**Recommendation:** Interface relocation plans should include a `go mod graph` or import-dependency sketch during planning to verify the target consumer package can legally import the implementation. GSD plans should have a standard pre-check step for `import cycle` before committing to a compile-level assertion strategy.

### MEDIUM

#### 4. PERF-003 deferral decision opaque in plan text

**17-01-PLAN.md:386-393** — PERF-003 says "focused on functions touched in this task" but doesn't acknowledge that `video_recording_task_service.go` has no `ctx` parameters. The acceptance criterion `grep -c ".WithContext(ctx)" ≥ 10` would fail at code review unless the executor independently discovers the signature problem. The SUMMARY.md (D1) correctly deferred PERF-003 after discovery, but the plan text suggests P0 will satisfy this criterion — it couldn't.

**Recommendation:** If a PERF/STYLE finding requires method signature changes that cascade beyond the current phase scope, the plan should either (a) flag this during design or (b) explicitly defer with transparent reasoning (like the SEC-003b deferral is documented), not present an acceptance criterion that's impossible to meet.

#### 5. P2 SEC-013/014 and PERF-013/014 shared commits violate D-02.1

**17-04-SUMMARY.md D1/D2** — SEC-013 and SEC-014 were bundled into a single commit (`3719c67`), as were PERF-013/014 (`f776c2d`). The plan's mega-commit structure (D-02.1) specified separate atomic commits per finding. The rationale (same file changes) is reasonable, but it deviates from plan without plan-level acknowledgment.

**Recommendation:** Plans should anticipate same-file bundling and explicitly identify "these findings share a file and should be committed together" to avoid post-hoc explanations.

#### 6. PERF-015 "already existed" — plan didn't read current state

**17-04-PLAN.md:417-426** specifies adding DB pool configuration (`SetMaxOpenConns` etc.) in `app.go`. The SUMMARY.md D4 notes "app.go 187-189 行 **已存在** 池配置" — the code already had these calls from a prior wave. The plan acceptance criterion `grep -c "SetMaxOpenConns\|SetMaxIdleConns\|SetConnMaxLifetime" ≥ 3` was satisfiable from pre-existing code, making this a no-op plan specification.

**Recommendation:** P2 (LOW sweep) plans should verify what already exists before specifying changes. A simple `grep` during plan writing would have shown the pool config was already present.

### LOW

#### 7. P0 plan lists `cmd/server/app_test.go` but actual test is minimal

**17-01-PLAN.md:197** describes a mock-based `app_test.go` verifying `SetAuditService` wiring. The actual commit `4d3de0b` adds `cmd/server/app_test.go` (45 lines) — which is a simple smoke test, not the described mock assertion. The test is sufficient for D-04.1 (P0 needs unit tests), but the plan over-specified the testing approach.

#### 8. PERF-002 18 vs 15 find sites — auditing error not reconciled in plan

**17-01-SUMMARY.md ($Pseudo-finding "误报")** — The audit listed 18 `.Find` without Limit, but 3 were replaced by Pluck+IN in PERF-001. The plan's acceptance criterion still says "18 sites" when only 15 needed Limit. The SUMMARY correctly notes the discrepancy, but the plan should have flagged this during design.

---

## 4. Execution Deviations Worth Feeding Back

### Systematic Planning Defects (would re-occur)

| Deviation | Finding IDs | Root Cause | Future Prevention |
|-----------|-------------|------------|-------------------|
| PERF-003 deferred mid-P0 (method sigs lack ctx) | PERF-003 | Plan assumed `WithContext(ctx)` is a per-site patch; actual is method-signature cascade | Pre-plan `grep -n "ctx context.Context"` on target file signatures before scoping |
| SEC-004 fallback chain = 3 encodings, not plan's 2 | SEC-004 | Plan didn't read actual signing code (`URLEncoding` vs `StdEncoding`) | Plan must read the exact lines of code that compute the value before writing the fix spec |
| MinTLSVersion is `string`, plan assumed `uint16` | SEC-003a | Plan inferred type name from Go stdlib constant without checking config struct | Pre-plan `grep -n "MinTLSVersion" internal/config/config.go` to verify field type |
| STYLE-003 compile-assert cycles (services↔scheduler) | STYLE-003 | Plan didn't analyze import dependency graph | Pre-plan import-cycle check for any interface relocation plan |
| BUG-005: 4 of 5 files have no GORM or no ctx | BUG-005 | Plan assumed GORM calls based on file location, not verification | Pre-plan `grep -c "\.\(Find\|First\|Create\|Update\|Delete\)"` per targeted file |
| STYLE-004: 8 handlers not 10 | STYLE-004 | Plan counted handlers by file, didn't verify GetUserID usage | Pre-plan `grep -rn "middleware.GetUserID" internal/handlers/` for accurate count |
| PERF-015 already existed | PERF-015 | Plan didn't check if config calls were already present | Pre-plan acceptance grep on existing code base before specifying changes |

### Acceptable In-Flight Judgment Calls (specific to context)

| Deviation | Rationale | Correct Decision? |
|-----------|-----------|-------------------|
| `common/interfaces.go` Service interface NOT moved (>5 consumers) | Import cycle risk + blast radius > 5 consumers per W15 rule | Yes — marker comment not added (plan called for comment but SUMMARY D4 argues no need); acceptable judgment |
| SEC-013/014 shared commit (same file) | Both modify `cmd/server/app.go` adjacent lines | Yes — splitting into 2 commits would create an intermediate broken state |
| PERF-013/014 shared commit | Cross-file but same tier, small changes | Acceptable — bisect impact is negligible |
| BUG-005 only `audit_log_service.go` modified | Other 4 files have no ctx or no GORM | Yes — forcing `context.Background()` would be wrong per D-03.7 |
| STYLE-002 false positive | Audit §5.1 confirms; no code change needed | Yes |
| STYLE-009 skipped (133 getter renames) | Blast radius too large for P2 tier | Yes — correct per Claude's Discretion (CONTEXT.md §Claude's Discretion) |
| STYLE-003: stub pattern instead of real-type assertion | Import cycle prevented real-type assignment | Yes — stub is cleaner and equally effective |

---

## 5. Gaps Missed by Plans

### 5.1 SEC-003b: Huawei password DB plaintext (HIGH deferred)

**Risk:** Remains HIGH. The audit lists password-in-plaintext as a HIGH finding. The plans defer it to an "independent phase" with a single marker comment. The mitigating factor is that Huawei cloud credential access already requires `HUAWEI_INSECURE_SKIP_VERIFY=false` (enforced by SEC-003a), so the path to this credential is: compromise DB → read plaintext → authenticate to Huawei → exfiltrate recordings. This is a two-hop chain, but the DB compromise hop is realistic (SQL injection, backup exposure, DBA access).

**Assessment:** Acceptable deferral only if a dedicated phase is scheduled within 1-2 milestones. If no schedule exists, this should be promoted back to P0.

### 5.2 PERF-003 full sweep (403 missing `WithContext`) — MEDIUM risk

**Deferred risk:** GORM operations without `WithContext` continue to use `context.Background()`, meaning:
1. HTTP request cancellation doesn't cascade to SQL queries
2. Scheduler background tasks can't be cancelled during graceful shutdown
3. Connection pool is shared across all requests (no per-request timeout)

The plan's mitigation (adding `WithContext` to touched functions) covers only ~30 of 403 sites (~7%). The remaining 93% continue with `context.Background()`. This is arguably a **latent correctness bug**, not just a performance issue, because it affects graceful shutdown behavior (clients hang on DB during shutdown).

**Assessment:** Should be given a concrete phase slot, not left as an indefinite `<deferred>`. The blast radius grows with every new feature that adds GORM calls.

### 5.3 STYLE-001 full migration (168 `errors.New` + 474 `fmt.Errorf`)

**Deferred risk:** Error handling is inconsistent — some sites use sentinel errors (`ErrNotFound`), others use string comparison (`err.Error() == "用户不存在"`), and many use `%v` instead of `%w` (preventing `errors.Is`/`errors.As`). This is not a runtime correctness issue but it **prevents callers from programmatically inspecting errors**. The partial migration (3 sites in P2) is token compliance.

**Assessment:** LOW priority for production correctness, but MEDIUM priority for code maintainability. The plan's deferral decision is correct given the 642-site blast radius, but it should be tracked in a visible backlog.

### 5.4 HMAC jti: in-memory only vs. audit's server-side table

**Audit intent:** The audit (§2.2 SEC-004) suggests server-side `used_jtis` table (Redis or DB) for cross-restart replay protection. **Plan 17-01** implemented an in-memory `map[string]struct{}` that resets on restart, and correctly documents the limitation.

**Risk:** Low for most use cases — HLS tokens have a 5-min TTL, so an attacker who steals a token needs to:
1. Replay it within 5 minutes
2. Target the same process (not a different replica)
3. Do so before the in-memory map is reset by restart

For single-instance deployments with daily restarts, the window is 5 minutes per day. Acceptable for P0, but if multi-replica or long-lived processes are planned, this should be promoted.

### 5.5 No plan specified what happens to generated HLS URLs after tokenization (PERF-005)

**Gap:** The plan adds bounded-concurrency tokenization to `rewriteM3U8WithToken` but doesn't address the **leaked origin URLs** that may exist in the file system from before the fix. Old `.m3u8` files stored on disk still contain non-tokenized URLs. This is a minor operational concern, not a plan gap per se, but could be documented.

### 5.6 Testing gap: No integration test for auth → audit DI chain

**Gap:** SEC-002 adds `authService.SetAuditService(auditService)` but there is no end-to-end test that constructs the real auth+audit services and verifies a login failure produces an audit log entry. The `app_test.go` (45 lines) is a smoke test with mocks. The real wiring is only tested by `go build ./...`.

**Assessment:** Acceptable for P0 urgency (the fix is one line), but should be flagged as a gap if the audit system's correctness is critical.

---

## 6. Suggestions for Future Plan Templates

### 6.1 Pre-plan code verification step

Every plan should include a mandatory pre-plan task that reads the **exact lines** targeted by each finding and validates:
- Field types match expectations (`grep -n "fieldName" struct_file.go`)
- Method signatures accept required params (`grep -n "func.*MethodName(" target.go`)
- GORM calls actually exist in target files (`grep -c "\.\(Find\|First\|Create\|Update\)" target.go`)
- Old encoding/behavior is verified from source, not assumed (`grep -n "EncodeToString\|Encode" signing_code.go`)

This would have caught all 7 systematic defects listed in §4.

### 6.2 Import-cycle analysis for interface relocation plans

STYLE-003 (interface moves) should trigger a standard pre-check: render the import dependency graph for both source and target packages. If the target consumer already imports the source impl package (or vice versa), compile-time assertions may not be possible without stub patterns.

**Template addition for STYLE tasks:**
```
## Import Cycle Check
- Source package imports: `go list -f '{{.Imports}}' ./internal/services/`
- Target package imports: `go list -f '{{.Imports}}' ./internal/scheduler/`
- Cycle exists? YES/NO → If YES, use stub pattern instead of real-type assertion
```

### 6.3 Acceptance criteria with semantic validation

Grep-based acceptance criteria are necessary but not sufficient. For encoding changes (SEC-004), config field changes (SEC-003a), and signature changes (STYLE-004), include a semantic check:
```
## Semantic Acceptance
- Generate token with old encoding → restart → token validates (SEC-004)
- `cfg.Huawei.MinTLSVersion` after `Load()` is type string, not uint16 (SEC-003a)
- Build a test binary with the changes, run `go vet` on the old unused-import checker (STYLE-003)
```

### 6.4 Tier granularity is correct for Phase 17

The 4-wave decomposition (P0=11, P1a=12, P1b=7, P2=20) was appropriate. P1 splitting into a=error-handling+security and b=performance+interface was necessary because 19 MEDIUM findings in a single plan would have been too large. The dependency ordering (P0→P1a→P1b→P2) was correct — P0 changed config fields and function signatures that P1a's GetUserID change consumed.

### 6.5 Cross-wave gofmt drift prevention

The post-wave gofmt cleanup commit `c04f805` (P2 wave, fixing P0-touched files) reveals that gofmt drift can accumulate across waves when different tools/formatters touch the same file. Suggestion: **include a `gofmt -w` step at the end of every plan's verification**, not just during final acceptance. Alternatively, make pre-commit hooks run `gofmt` automatically.

### 6.6 Commit granularity flex in plans

P2 deviations show that same-file changes between findings force shared commits. Plans should pre-identify findings that share files and either:
- Explicitly marker them as "SAME FILE → combine into one commit"
- Specify the commit boundary as "at finding granularity, except where stated"

---

## 7. Risk Assessment

**Overall: LOW**

Despite the 13 documented deviations and the systematic planning defects, the Phase 17 outcome risk is LOW because:

1. **No deviation caused a correctness regression** — Every deviation was caught by the executor before merge (triple-encoding, MinTLSVersion string, BUG-005 scope reduction, STYLE-003 stubs). The verification gates (`go build`, `go vet`, `gofmt`, tests) caught all issues.

2. **All 56 findings correctly triaged** — 50 fixed, 3 explicitly deferred (SEC-003b, STYLE-001 full, STYLE-009), 1 benign (PERF-003 full sweep — acknowledged but deferred with transparent reasoning), 1 false positive (STYLE-002), 1 judgment-skip (STYLE-009 in P2 — per Claude's Discretion).

3. **The 5 worst attack chains are closed** — SEC-001+002+003a+004 fix the complete config→auth→TLS→token chain. The remaining gaps (SEC-003b DB password encryption, PERF-003 full sweep) are two-hop attacks requiring prior DB compromise.

4. **80 files changed with zero regressions** — `go test -race` on 12 packages all PASS. No existing test broke. No new dependencies introduced.

5. **Audit trail is complete** — Every commit references finding IDs. The SUMMARY files document all deviations with rationale. The deployment docs were synced.

The only note of caution: if the project does not schedule the 3 deferred items (SEC-003b, PERF-003 full sweep, STYLE-001 full) within the next 2 milestones, these should be re-escalated. SEC-003b in particular remains a HIGH finding with no timeline.

---

## Appendix: Deviation Count Summary

| Source | Systematic Planning Defects | Acceptable In-Flight | Notes |
|--------|----------------------------|---------------------|-------|
| Wave 1 (P0) | 3 (PERF-003 scope, SEC-004 encoding, MinTLSVersion type) | 2 (HuaweiConfigRow inline, SEC-003b defer) | 3 of 4 deviations were preventable by pre-plan code reading |
| Wave 2 (P1a) | 2 (BUG-005 file count, STYLE-004 handler count) | 1 (HMAC encoding not re-committed) | Both systematic defects = assumed GORM/usage without verification |
| Wave 3 (P1b) | 1 (STYLE-003 import cycle) | 3 (PERF-009 partial, PERF-008 sanitizer skip, common/interfaces.go) | Import cycle was predictable |
| Wave 4 (P2) | 1 (PERF-015 already existed) | 5 (SEC-013/014 bundled, PERF-013/014 bundled, SEC-013 default empty, STYLE-001 3-sites, STYLE-009 skip) | 4 of 5 were judgment calls, not plan errors |

**Total:** 7 systematic defects, 11 in-flight judgments. Systematic defect rate: ~17% of plans' acceptance criteria required executor correction — indicating the plans were thorough but the code-reading pre-check step was structurally missing.

---

## Consensus Summary

**Reviewers invoked:** 1 (opencode only — all other CLIs unavailable, claude skipped for independence per `CLAUDE_CODE_ENTRYPOINT=cli`)

With only one reviewer, **no cross-AI consensus is possible**. The opencode review (above) stands as the single independent assessment. The summary below distills its key findings for downstream consumption:

### Single-Reviewer Verdict (opencode)

- **Overall risk:** **LOW** — no correctness regressions; all 5 worst audit attack chains (SEC-001/002/003a/004 + BUG-001) closed
- **Plan quality:** "Highest-quality GSD plans in this project" — grep-verify acceptance criteria, STRIDE threat models per plan, wave dependency tracking, no-go lists, deferral traceability
- **Execution fidelity:** 50/56 findings remediated in code, 3 explicitly deferred (SEC-003b DB encryption, STYLE-001 full migration, STYLE-009 rename), 1 false positive (STYLE-002), 1 judgment-skip (STYLE-009 in P2 per Claude's Discretion), 1 benign deferral (PERF-003 full sweep with transparent reasoning)
- **Systematic planning defects (7):** Plans consistently assumed field types / method signatures / GORM call presence / old encoding format **without code verification** — drove 7 of 13 documented deviations. All preventable by a **30-second pre-plan grep verification step**.
- **Acceptable in-flight judgment calls (11):** Same-file commit bundling (SEC-013/14, PERF-13/14), `common/interfaces.go` skip (>5 consumers), STYLE-003 stub pattern (import cycle), STYLE-002 false positive — all correct decisions.

### Deferred Items Requiring Explicit Scheduling (per opencode §5)

| Item | Risk | Recommendation |
|------|------|----------------|
| SEC-003b (Huawei password DB plaintext) | **HIGH** | Dedicated phase within 1-2 milestones; if no schedule, re-escalate to P0 |
| PERF-003 full sweep (93% of 403 GORM sites still `context.Background()`) | **MEDIUM** (latent shutdown-correctness bug) | Concrete phase slot — blast radius grows with each new feature |
| STYLE-001 full migration (168 + 474 sites) | **MEDIUM** (maintainability) | Tracked backlog item; not blocking production |
| HMAC jti server-side `used_jtis` table | **LOW** (5-min TTL window) | Promote if multi-replica or long-lived processes planned |

### Future Plan Template Improvements (per opencode §6)

1. **Pre-plan code verification** — mandatory grep step to validate field types, method signatures, GORM call presence, old encoding format before writing acceptance criteria (would have caught all 7 systematic defects)
2. **Import-cycle analysis** for interface relocation plans — `go list -f '{{.Imports}}'` on source + target packages; if cycle exists, use stub assertions not real-type
3. **Semantic acceptance criteria** beyond grep — e.g., "generate token with old encoding → restart → token validates" for SEC-004
4. **Cross-wave gofmt prevention** — `gofmt -w` at end of every plan's verification, not just final acceptance (the `c04f805` cross-wave housekeeping commit proves the gap)
5. **Commit granularity flex** — pre-identify findings sharing files; explicit "SAME FILE → combine into one commit" markers

### To Incorporate Feedback

- For similar 50+ finding phases, apply the 5 template improvements before writing plans
- Schedule the 3 deferred items (SEC-003b, PERF-003 full, STYLE-001 full) explicitly in the next milestone roadmap
- The cross-wave gofmt drift issue (§6.5) is best addressed via a pre-commit hook (`gofmt` check), not per-plan cleanup
- Re-run cross-AI review for the next major phase (e.g., a Phase 18 fixing the deferred items) — hopefully with multiple reviewers available

---

*Review generated by opencode v1.14.20 via GSD `gsd-review` workflow. Single-reviewer post-execution retrospective; orchestrator wrapped with frontmatter + consensus section per workflow template.*
