---
phase: 23
slug: api-gorm-sentinel
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-06
---

# Phase 23 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1 |
| **Config file** | none (standard `go test`) |
| **Quick run command** | `go test ./internal/huawei ./internal/models ./internal/config ./internal/errors ./cmd/error-doc-gen` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** `go test ./<modified-package> -count=1`
- **After every plan wave:** `go test ./internal/huawei ./internal/models ./internal/config ./internal/errors ./cmd/error-doc-gen`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~60 seconds (per-package quick run)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 23-01-01 | 01 | 1 | DETECT-01 | T1 / Tampering | strict nested JSON parse; presence-aware fields | unit | `go test ./internal/huawei -run 'Test.*ConferenceState' -count=1` | ❌ W0 | ⬜ pending |
| 23-01-02 | 01 | 1 | DETECT-04 | T1 / Tampering | absent fields → IsInConf fallback | unit | `go test ./internal/huawei -run 'Test.*ConferenceState.*Fallback' -count=1` | ❌ W0 | ⬜ pending |
| 23-02-01 | 02 | 1 | AUDIT-01 | — / N/A | 5 columns migrate; defaults persist; R/W round-trip | integration (SQLite memory) | `go test ./internal/models -run 'TestVideoRecordingTaskSmartEndFields' -count=1` | ❌ W0 | ⬜ pending |
| 23-03-01 | 03 | 1 | AUDIT-05 | — / N/A | 3 sentinels map to 500, recognized, docs clean | unit + artifact | `go test ./internal/errors ./cmd/error-doc-gen && go generate ./internal/errors/... && git diff --exit-code -- docs/errors.md` | partial | ⬜ pending |
| 23-04-01 | 04 | 1 | CFG-01 | T3 / DoS | all defaults load; explicit `false` preserved; invalid thresholds rejected | unit | `go test ./internal/config -run 'TestSmartEnd' -count=1` | ❌ W0 | ⬜ pending |
| 23-05-01 | 05 | 1 | CFG-02 | T3 / DoS | YAML contains exactly 14 keys, loads expected values | unit / config artifact | `go test ./internal/config -run 'TestSmartEnd.*YAML' -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Extend `internal/huawei/client_test.go` with mailbox fixtures (matrix in RESEARCH.md §Huawei fixture matrix) and error tests for malformed JSON
- [ ] Create `internal/models/video_recording_task_test.go` for 5-field schema + read/write round-trip
- [ ] Create `internal/config/smart_end_test.go` for defaults, explicit `false`, exact 14 keys, threshold validation
- [ ] Extend `internal/errors/mapping_test.go` for 3 sentinel names + HTTP 500 mapping
- [ ] `cmd/error-doc-gen` regenerates `docs/errors.md` clean (CI sync-check passes)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real TE40 `WEB_GetMailboxDataAPI` payload parse | DETECT-01 | Hardware-in-the-loop; sanitized fixture unavailable | When TE40 device accessible: hit API, capture response, parse via `GetConferenceState`, confirm `confState`/`joinSum` presence detected |

*If no TE40 hardware: rely on synthetic JSON fixtures matching PRD-documented field shape.*

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s (per-package quick run)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending