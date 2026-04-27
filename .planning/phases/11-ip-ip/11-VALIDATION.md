---
phase: 11
slug: ip-ip
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-27
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (`go test`) + Vitest (frontend) |
| **Config file** | Existing: `go.mod`, `frontend/vitest.config.ts` |
| **Quick run command** | `go test ./internal/auth/... -v -run TestIP` |
| **Full suite command** | `go test ./... && cd frontend && npm test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/auth/... -v -run TestIP`
- **After every plan wave:** Run `go test ./... && cd frontend && npm test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | D-01..D-05 | T-11-01 | IP restriction enforced before token generation | unit | `go test ./internal/auth/... -run TestCheckIPRestriction` | ❌ W0 | ⬜ pending |
| 11-01-02 | 01 | 1 | D-06..D-09 | T-11-02 | Invalid IP formats rejected | unit | `go test ./internal/auth/... -run TestValidateIPFormat` | ❌ W0 | ⬜ pending |
| 11-02-01 | 02 | 1 | D-02..D-03 | T-11-03 | OR logic correctly merges user+role IPs | unit | `go test ./internal/auth/... -run TestIPMergeORLogic` | ❌ W0 | ⬜ pending |
| 11-03-01 | 03 | 2 | D-13..D-15 | T-11-04 | IP restriction failure logged to audit | integration | `go test ./internal/auth/... -run TestIPRestrictionAuditLog` | ❌ W0 | ⬜ pending |
| 11-04-01 | 04 | 2 | Frontend UI | — | IP input validation on save | e2e | `cd frontend && npm test -- IPInput.test.ts` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/auth/iprestriction_test.go` — stubs for IP restriction tests
- [ ] `internal/models/iprule.go` — IP validation and matching functions with tests
- [ ] `frontend/src/components/auth/IPInput.test.ts` — IP input component tests

*Existing infrastructure:* Go testing and Vitest are already configured.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Admin lockout warning | D-13 | Requires UI interaction with admin's current IP | 1. Login as admin from IP 1.1.1.1<br>2. Try to set IP restriction to 2.2.2.2 only<br>3. Verify warning appears<br>4. Confirm lockout is prevented |
| Real IP detection | D-10 | Requires actual network requests | 1. Login from different network<br>2. Verify ClientIP() returns correct address<br>3. Check audit log records correct IP |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
