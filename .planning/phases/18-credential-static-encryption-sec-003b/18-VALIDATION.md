---
phase: 18-credential-static-encryption-sec-003b
slug: credential-static-encryption-sec-003b
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-03
---

# Phase 18 — Validation Strategy

> Per-phase validation contract retro-fitted after execution from shipped evidence.
>
> Companion to `18-VERIFICATION.md` (status: **passed, 11/11 must-haves**) and
> `18-SUMMARY.md`. VERIFICATION reports the outcome; this VALIDATION document
> reconstructs the feedback-sampling contract that would have governed execution.

Phase 18 has no surviving PLAN.md, CONTEXT.md, or DISCUSSION-LOG; per
`18-VERIFICATION.md` Evidence Limitations #1, this contract is reconstructed from
SUMMARY + git history + live code. Commit enumeration follows the `18-SUMMARY.md`
§Wave 4 body, not its stale frontmatter `Final HEAD: bd84fe2`: the body is authoritative
and records 9 atomic wave commits through W4d `0c018f2`, plus post-audit commit
`5d536ec` as pre-stored evidence dated before phase 21 kickoff.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/stretchr/testify/assert` (already in go.mod) |
| **Config file** | none — Go std convention (`*_test.go` co-located, auto-discovered) |
| **Quick run command** | `go test ./internal/services/... ./internal/utils/... ./cmd/server/... -count=1` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | quick ~8s (`internal/services` 8.010s + `cmd/server` 2.057s recorded by 18-VERIFICATION behavioral spot-checks, with package overlap/concurrency) · full ~30s |

---

## Sampling Rate

- **After every atomic commit:** `go build ./... && go test -count=1 ./internal/utils/... ./internal/services/... ./cmd/server/...`
- **After every wave (W1a..W4d):** `go build ./... && go vet ./... && go test -race ./internal/utils/... ./internal/services/... ./cmd/server/...`
- **Before `/gsd:verify-work`:** `go build ./... && go vet ./... && go test -race ./...`
- **Operator runbook dry-run:** follow W4d `DEPLOYMENT.md` operator runbook out-of-band; staging infrastructure makes it non-blocking for automated verification.
- **Max feedback latency:** 30s

---

## Per-Task Verification Map

> Phase 18 has no surviving PLAN tasks. Rows therefore map shipped waves from the
> `18-VERIFICATION.md` Cross-Reference evidence and `18-SUMMARY.md` §Wave 4 body.

| Wave | Task | Commit | Test Command | File Exists | Status |
|------|------|--------|--------------|-------------|--------|
| W1a | SM4-GCM envelope + PKCS7 padding + `ParseCredentialEnvelope` / `EncodeCredentialEnvelope` | `e6315ce` | `go test -race ./internal/utils/...` | ✅ (`internal/utils/sm4_password.go:254/365/389`) | ✅ retro-verified |
| W1b | `AuthConfig.CredentialSM4*` + `ValidateCredentialSM4Config` | `1dbb3b0` | `go test -race ./internal/config/... && go build ./...` | ✅ (`internal/config/config.go:111-128`) | ✅ retro-verified |
| W1c | CredentialEncryptor encrypt/decrypt/rotate/invariant scan + four-field widening | `edaa4ae` | `go test -race ./internal/services/...` | ✅ (`internal/services/credential_encryptor.go`) | ✅ retro-verified |
| W2 | encrypt-on-write/decrypt-on-read in input config service + config service base64-stub removal | `558f723` | `go test -race ./internal/services/...` + verify exactly two `s.encryptor.Encrypt/Decrypt` call sites | ✅ | ✅ retro-verified |
| W3 | fail-closed Initialize 10-step sequence + `widenPasswordColumns` | `bd84fe2` | `go test -race ./cmd/server/...` + verify Migrate / dual InvariantScan / Rotate wiring | ✅ (`cmd/server/app.go:143-196/361`) | ✅ retro-verified |
| W4a | per-site/version counts and three-stage logging (`after_migrate` / `after_rotate` / `after_invariant`) | `8796ca3` | `go test -run 'Test(CountByVersion|VersionCounts|LogVersionCounts)' ./internal/services/...` | ✅ | ✅ retro-verified |
| W4b | `TestRepeatedRotation_V1ToV2ToV3` four-stage end-to-end rotation | `3822497` | `go test -run TestRepeatedRotation_V1ToV2ToV3 ./internal/services/...` | ✅ (`credential_encryptor_test.go:660`) | ✅ retro-verified |
| W4c | `TestRepeatedRotation_IntermediateInvariantScan` and cmd/server invariant extensions | `a182cd6` | `go test -run 'TestRepeatedRotation_IntermediateInvariantScan|TestPhase18_' ./internal/services/... ./cmd/server/...` | ✅ | ✅ retro-verified |
| W4d | `DEPLOYMENT.md` operator runbook + physical-remanence guidance | `0c018f2` | verify headings `凭据密钥配置与轮换` and `凭据存储的物理残留` in `DEPLOYMENT.md` | ✅ (`DEPLOYMENT.md:150/321`) | ✅ retro-verified |
| post-audit | `TestHuaweiDBAdapter_ProductionDecrypts` + SEC-003b marker update | `5d536ec` — ⚠ pre-stored evidence (2026-08-02, before phase 21 kickoff; not a phase 21 deliverable) | `go test -run TestHuaweiDBAdapter_ProductionDecrypts ./cmd/server/...` | ✅ (`cmd/server/app_test.go:53`, `internal/huawei/manager.go:132`) | ✅ retro-verified (pre-stored) |

*Status: ⬜ pending · ✅ retro-verified (post-execution; this VALIDATION.md is retroactive) · ⚠ pre-stored evidence (date predates phase 21 kickoff)*

---

## Wave 0 Requirements

> Existing phase 18 tests are listed as the test inventory that a pre-execution Wave 0
> would have required. The live repository is authoritative where the historical
> planning note conflicts with it: `internal/utils/sm4_password_test.go` exists and
> directly covers the SM4-GCM primitives.

- [ ] `internal/utils/sm4_password_test.go` — SM4-GCM round-trip, random nonce, wrong key, tamper/truncation, envelope parser/encoder, and version constant tests (existing shipped coverage)
- [ ] `internal/config/config_test.go` — `ValidateCredentialSM4Config` valid/invalid matrices + `BindEnvCredentialSM4` (existing shipped coverage)
- [ ] `internal/services/credential_encryptor_test.go` — `TestRepeatedRotation_V1ToV2ToV3` (line 660), `TestRepeatedRotation_IntermediateInvariantScan`, migration, invariant, rotation, and count/log tests (existing shipped coverage)
- [ ] `internal/services/input_config_service_test.go` — create/update encrypt-on-write and get decrypt-on-read paths (existing shipped coverage)
- [ ] `internal/services/config_service_test.go` — AD password strip/store/load encryption contract (existing shipped coverage)
- [ ] `cmd/server/phase18_integration_test.go` — startup migration, fail-closed invariant, repeated rotation, and version-log integration tests (existing shipped coverage)
- [ ] `cmd/server/app_test.go` — `TestHuaweiDBAdapter_ProductionDecrypts` production decrypt invariant (`5d536ec` pre-stored evidence)
- [x] `internal/utils/sm4_password.go` — exported `EncryptGCM` / `DecryptGCM` / envelope primitives already directly covered by `sm4_password_test.go`
- [x] `internal/services/input_config_service.go` — encrypt-on-write + decrypt-on-read wiring already covered by service tests and the two live encryptor call sites
- [x] `internal/config/config.go` — `CredentialSM4*` field family and startup validation already covered by config tests and startup compile path
- [x] `cmd/server/app.go` — fail-closed startup wiring + widening already covered by cmd/server integration tests and production-decrypt invariant
- [x] `internal/huawei/manager.go` — SEC-003b marker and plaintext boundary already covered by `TestHuaweiDBAdapter_ProductionDecrypts` and 18-VERIFICATION D18-7/D18-11
- [x] `DEPLOYMENT.md` — operator runbook + physical-remanence sections already covered by content review and heading checks (18-VERIFICATION D18-8)
- [x] Cross-phase contract intact: phase 17 SEC-003b deferred → phase 18 delivered; phase 19 D2 adapter merge preserves SM4-GCM decrypt (`3b2d41f`)
- [x] Real production-data post-audit is **out of v1.1 scope** per `18-SUMMARY.md` §“不在本 phase 范围”; staging verification remains future operator work

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Operator rotation runbook dry-run (A/B/C) | W4d `DEPLOYMENT.md` runbook | Requires a real multi-instance environment and production-like secret injection/KMS | In staging configure v2 + previous=v1, restart and require `by_version__v1=0`; configure v3 + previous=v2 and require `by_version__v2=0`; remove previous and verify clean restart |
| Physical-remanence controls for WAL/VACUUM/snapshots/backup media/memory/swap/monitoring | W4d physical-remanence section | Requires OS-level and backup-chain tools | After staging rotation checkpoint/truncate WAL, VACUUM as directed, inspect backup artifacts for plaintext credentials, and require all searches to be negative |
| Phase 18 core must-haves | n/a | No manual verification required for retro-verify | All 11 must-haves were programmatically verified; see `18-VERIFICATION.md` §Human Verification Required |

---

## Validation Sign-Off

- [x] All tasks have automated verification or Wave 0 dependencies (9 wave commits + 1 post-audit pre-stored evidence row)
- [x] Sampling continuity: no 3 consecutive tasks without automated verification (each wave has build/test or deterministic content checks)
- [x] Wave 0 covers all missing references (7 shipped test files + indirect implementation evidence + cross-phase contract + out-of-scope note)
- [x] No watch-mode flags (`go test -count=1` / `go test -race`; no `-watch`)
- [x] Feedback latency < 30s (recorded package spot-checks fit the 30s bound)
- [x] `nyquist_compliant: false` set in frontmatter (honest retro-fitted provenance; pre-execution Nyquist sampling was not active)

**Approval:** pending (retro-fitted post-execution; see `18-VERIFICATION.md` status: passed 11/11)
