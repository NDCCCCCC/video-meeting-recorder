---
phase: 12
slug: windows-ad
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-28
---

# Phase 12 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — Wave 0 installs |
| **Quick run command** | `go test ./internal/auth/... -run TestAD -v` |
| **Full suite command** | `go test ./... -v` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/auth/... -run TestAD -v`
- **After every plan wave:** Run `go test ./... -v`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 12-00-01 | 00 | 0 | D-01 to D-05 | T-12-01 to T-12-03 | Strategy pattern enforces mode isolation | unit | `go test ./internal/auth/... -run TestAuthStrategy -v` | ✅ W0 | ⬜ pending |
| 12-01-01 | 01 | 1 | D-21 to D-23 | T-12-04 | AD fields nullable, no auth_source | unit | `go test ./internal/models/... -run TestUserADFields -v` | ✅ W0 | ⬜ pending |
| 12-01-02 | 01 | 1 | D-04 to D-05 | T-12-05 | Local mode bypasses AD entirely | integration | `go test ./internal/auth/... -run TestLocalAuthOnly -v` | ✅ W0 | ⬜ pending |
| 12-02-01 | 02 | 1 | D-15 to D-17 | T-12-06 | Config validation blocks invalid AD | unit | `go test ./internal/config/... -run TestADValidation -v` | ✅ W0 | ⬜ pending |
| 12-02-02 | 02 | 1 | D-12 to D-14 | T-12-07 | Port 389 shows warning | unit | `go test ./internal/config/... -run TestPortWarning -v` | ✅ W0 | ⬜ pending |
| 12-03-01 | 03 | 2 | D-01 to D-03 | T-12-08 | AD mode routes to LDAP only | integration | `go test ./internal/auth/... -run TestADAuthFlow -v` | ✅ W0 | ⬜ pending |
| 12-03-02 | 03 | 2 | D-06 to D-08 | T-12-09 | User transparent to source | integration | `go test ./internal/services/... -run TestADUserMapping -v` | ✅ W0 | ⬜ pending |
| 12-04-01 | 04 | 2 | D-18 to D-20 | T-12-10 | Error messages sanitized | unit | `go test ./internal/auth/... -run TestADErrorMessages -v` | ✅ W0 | ⬜ pending |
| 12-05-01 | 05 | 3 | D-09 to D-11 | T-12-11 | Test connection validates real AD | integration | `go test ./internal/handlers/... -run TestADTestAPI -v` | ✅ W0 | ⬜ pending |
| 12-05-02 | 05 | 3 | D-19 | T-12-12 | Detailed errors logged only | unit | `go test ./internal/auth/... -run TestADErrorLogging -v` | ✅ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/auth/ad_authenticator_test.go` — stubs for D-01 to D-05 (strategy pattern)
- [ ] `internal/auth/local_authenticator_test.go` — stubs for local mode isolation
- [ ] `internal/models/user_ad_test.go` — stubs for AD field validation (D-21 to D-23)
- [ ] `internal/config/ad_config_test.go` — stubs for AD config validation (D-15 to D-17)
- [ ] `internal/handlers/admin_ad_test.go` — stubs for test API (D-09 to D-11)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| LDAPS with self-signed cert | D-15 | Internal AD may use self-signed certs | Configure AD with test server using self-signed cert, verify connection succeeds with InsecureSkipVerify option |
| AD port 389 warning display | D-12 to D-14 | UI warning verification | Set AD server port to 389, confirm warning icon and message appear inline |
| AD user first login flow | D-06 to D-08 | End-to-end user creation | Login with new AD user, verify local user record created automatically |
| Mode switch audit log | D-01 to D-03 | Admin operation tracking | Switch auth mode local→ad→local, check audit logs record each change |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
