---
phase: 26-tls-ca-sec-003a-hotfix
plan: 01
subsystem: security
tags: [tls, x509, ca-bundle, pem, huawei, sec-003a]

# Dependency graph
requires:
  - phase: 17-56-p0-p1-p2 (v1.1 Phase 17)
    provides: MinTLSVersion=tls.VersionTLS12 + InsecureSkipVerify=false baseline + SEC-003a acceptance criteria
  - phase: 24 (v2.0 ActivityWatcher; paused)
    provides: GetConferenceState / ConferenceConnector that depend on Huawei HTTPS sessions this plan unlocks
provides:
  - Manager.SetCABundle(path) parsing private CA bundle with atomic publish under m.mu
  - NewHTTPClient gains caCertPool *x509.CertPool parameter feeding tls.Config.RootCAs
  - HuaweiConfig.CABundleFile + BindEnv to HUAWEI_CA_BUNDLE_FILE + deployment templates
  - cmd/server fail-closed startup wiring that calls SetCABundle before SetTLSPolicy-side effects and before HuaweiConferenceConnector construction
  - 5-scenario regression test suite (ValidPEM / InvalidOrMissing / EmptyPath / CertPoolBranches / ServerAndRootChain)
affects:
  - phase 24-04 (Nyquist verify): ActivityWatcher tests now have working Huawei TLS path to spin up against
  - phase 25 (scheduler + E2E): conference end signal chain is no longer blocked by x509 unknown authority

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Atomic PEM bundle publish: parse every CERTIFICATE block into a fresh x509.NewCertPool, assign m.caCertPool only after full success so no partially trusted pool is observable"
    - "Pool snapshot under m.mu.RLock in createClient so already-cached clients aren't disturbed by subsequent SetCABundle calls"
    - "Fail-closed startup wiring: SetCABundle runs between SetTLSPolicy and HuaweiConferenceConnector construction; logger.Fatal path + non-Fatal returned error for test-friendly dual coverage"
    - "Typed config + env override via explicit BindEnv (`huawei.ca_bundle_file`/`HUAWEI_CA_BUNDLE_FILE`)"

key-files:
  created: []
  modified:
    - internal/huawei/client.go
    - internal/huawei/manager.go
    - internal/huawei/manager_test.go
    - internal/config/config.go
    - internal/config/config_test.go
    - cmd/server/app.go
    - config.yaml.example

key-decisions:
  - "Atomic-only publish: SetCABundle builds a brand-new x509.NewCertPool, parses every PEM block with x509.ParseCertificate, and only swaps m.caCertPool if every block succeeds — partial-trust is impossible"
  - "createClient snapshots m.caCertPool under m.mu.RLock into Config.caCertPool pointer; avoids mutation surprise on cached clients (matches SetTLSPolicy semantics)"
  - "Empty/whitespace path → SetCABundle(\"\") is a first-class branch that sets pool to nil (system-CA fallback) — preserving InsecureSkipVerify=false + full chain validation"
  - "setDefaults intentionally does NOT assign cfg.Huawei.CABundleFile, so explicit empty string remains empty (system-CA opt-out pinned by TestHuaweiCABundle_EmptyPreserved)"
  - "Fail-closed wiring: loadHuaweiCABundle sits between SetTLSPolicy and NewHuaweiConferenceConnector; logger.Fatal is the production exit, returned wrapped error keeps the function testable with swapped fatalFunc"

patterns-established:
  - "PEM chain test pattern: generate self-signed CA via crypto/x509.Certificate{IsCA:true}, sign server cert with same CA, drive httptest.NewUnstartedServer over StartTLS, then assert real handshake under InsecureSkipVerify=false with the loaded pool"
  - "Pointer-identity assertion for tls.Config.RootCAs via assert.Same to lock in the contract that tls.Config.RootCAs is literally the caller's pool — eliminates subtle copy shenanigans"

requirements-completed: [SEC-003a-01, SEC-003a-02, SEC-003a-03, SEC-003a-04, SEC-003a-05]

# Metrics
duration: 27min
completed: 2026-08-06
---

# Phase 26 Plan 01 Summary

**Huawei terminal TLS private CA loading via atomic SetCABundle + RootCAs injection, with fail-closed startup wiring and 5-scenario regression suite**

## Performance

- **Duration:** 27 min
- **Started:** 2026-08-06T04:36:55Z
- **Completed:** 2026-08-06T05:03:00Z
- **Tasks:** 3 implementation + 1 docs commit
- **Files modified:** 7 (3 huawei package files, 2 config files, cmd/server/app.go, config.yaml.example)

## Accomplishments
- SEC-003a-01/02: `Manager.SetCABundle(path string) error` parses every PEM block with `x509.ParseCertificate`, adds each to a fresh `x509.NewCertPool`, and only publishes `m.caCertPool` after full success — no partial-trust state is observable. Errors carry the path AND `%w`-wrap the underlying `os.ReadFile`/`x509.ParseCertificate` cause.
- SEC-003a-01/05: `NewHTTPClient(server, port, timeout, insecureSkipVerify, minTLSVersion, caCertPool, logger)` gains a `*x509.CertPool` argument wired to `tls.Config.RootCAs`. `nil` preserves system-CA fallback; non-nil is verified by a real httptest TLS handshake under `InsecureSkipVerify=false`.
- SEC-003a-03: `cmd/server/app.go` initializes the bundle in `loadHuaweiCABundle` between `SetTLSPolicy` and `HuaweiConferenceConnector` construction. Empty path → `SetCABundle("")` + INFO log. Non-empty error → `logger.Fatal("加载华为 TLS CA bundle 失败", zap.String("path", ...), zap.Error(err))` + returned wrapped error.
- SEC-003a-04: `HuaweiConfig.CABundleFile` with `mapstructure:"ca_bundle_file"` + explicit `BindEnv("huawei.ca_bundle_file", "HUAWEI_CA_BUNDLE_FILE")`. Default `./certs/huawei-10.62.10.3-ca.pem` in `createDefaultConfigFile` and `config.yaml.example`. `setDefaults` deliberately skips assignment so explicit empty configuration remains empty (system-CA opt-out).
- SEC-003a-05: Five named tests across 2 files:
  - `TestSetCABundle_ValidPEM` — real httptest self-signed server handshake under `InsecureSkipVerify=false`
  - `TestSetCABundle_InvalidOrMissing` — 6 subtests cover missing file (`errors.As → *os.PathError`), malformed PEM, zero-cert PEM, trailing garbage, wrong PEM type, broken DER
  - `TestSetCABundle_EmptyPath` — `""`, `"   "`, `"\t\n"` all reduce `caCertPool` to nil with no error
  - `TestNewHTTPClient_CertPoolBranches` — pointer-identity assertion (`assert.Same`) that `tls.Config.RootCAs` literally equals the caller's pool when non-nil
  - `TestCABundle_ServerAndRootChain` — server cert + CA concatenated in one PEM, real handshake verifies hostname + chain against the trust store derived from that single file
- Phase 17 TLS hardening invariants preserved: `MinVersion=tls.VersionTLS12`, no 3DES cipher in `CipherSuites`, `InsecureSkipVerify=false` remains caller's policy, `ForceAttemptHTTP2=false`, SEC-013 outbound URL allowlist still enforced.

## Task Commits

1. **Task 1: Parse and inject Huawei private CA bundles into every new HTTP client** — `c8ef568` (feat)
   - NewHTTPClient signature + RootCAs assignment
   - Manager.caCertPool + SetCABundle + createClient snapshot
   - 5-scenario regression test suite
   - Includes gofmt -w absorption for MailboxState struct column alignment (was pre-existing drift)

2. **Task 2: Expose CA bundle path through Huawei configuration + deployment templates** — `f311e86` (feat)
   - HuaweiConfig.CABundleFile with `mapstructure:"ca_bundle_file"`
   - `BindEnv("huawei.ca_bundle_file", "HUAWEI_CA_BUNDLE_FILE")` beside existing TLS bindings
   - `config.yaml.example` template gets `ca_bundle_file` line with explanatory comment
   - `createDefaultConfigFile` inline YAML adds the same key/value
   - TestBindEnvHuaweiCABundle + TestHuaweiCABundle_EmptyPreserved

3. **Task 3: Load CA bundle fail-closed during server startup + atomic regression gate** — `c2357d7` (feat)
   - `cmd/server/app.go` adds `loadHuaweiCABundle` invoked after SetTLSPolicy, before NewHuaweiConferenceConnector
   - empty path → SetCABundle("") + INFO "未配置，使用系统 CA"
   - non-empty error → logger.Fatal "加载华为 TLS CA bundle 失败" + returned wrapped error
   - success → INFO "加载成功" with the resolved path

**Plan metadata:** (this docs commit)

## Files Created/Modified
- `internal/huawei/client.go` — adds `crypto/x509` import, `Config.caCertPool` field, updates `NewHTTPClient` signature with `caCertPool *x509.CertPool`, assigns `tls.Config.RootCAs: caCertPool`; `NewHuaweiClient` forwards `config.caCertPool`. Also absorbs pre-existing MailboxState column-alignment gofmt drift.
- `internal/huawei/manager.go` — adds `crypto/x509`, `encoding/pem`, `os`, `strings` imports; adds `Manager.caCertPool` field; adds `SetCABundle(path string) error` (atomic publish under `m.mu`); `createClient` snapshots `m.caCertPool` under `m.mu.RLock` before constructing `Config`.
- `internal/huawei/manager_test.go` — adds 5 named tests with 6 subtests inside `InvalidOrMissing`, plus shared helpers (`generateSelfSignedCert`, `parseFirstCertDER`, `parsePKCS1PrivateKey`, `signServerCert`, `startTLSServer`, `fakeHuaweiAPIHandler`, `makeHuaweiClient`). Uses Go stdlib `crypto/rsa`, `crypto/x509`, `crypto/x509/pkix`, `encoding/pem`, `net/http/httptest` only — no live device, no tracked PEM.
- `internal/config/config.go` — adds `CABundleFile string` to `HuaweiConfig` with exact tags; adds `BindEnv`; inserts `ca_bundle_file:` line into the inline `createDefaultConfigFile` YAML.
- `internal/config/config_test.go` — adds `TestBindEnvHuaweiCABundle` (env var → typed field) and `TestHuaweiCABundle_EmptyPreserved` (explicit empty survives Unmarshal).
- `cmd/server/app.go` — adds `loadHuaweiCABundle` helper called between `SetTLSPolicy` and `SetOutboundURLAllowlist`/`NewHuaweiConferenceConnector`; emits INFO/Fatal logging without PEM bytes or private keys.
- `config.yaml.example` — adds `ca_bundle_file: ./certs/huawei-10.62.10.3-ca.pem` under `huawei:` with explanatory comment about `HUAWEI_CA_BUNDLE_FILE` override and empty-string system-CA semantics.

## Decisions Made
- **Pointer identity contract for `tls.Config.RootCAs`.** Test asserts `assert.Same(pool, tlsCfg.RootCAs)` so any future "defensive copy" doesn't silently break the trust chain. Documented via comment block above the test.
- **Atomic pool publish only after full PEM parse.** SetCABundle builds a `x509.NewCertPool()` locally, adds each parsed cert, then assigns `m.caCertPool` inside the write lock — refused to commit the pool if any block failed. This is a structural defense against the "half-loaded trust store" failure mode.
- **`createClient` snapshot semantics.** Snapshotting under `m.mu.RLock` makes "SetCABundle then GetClient" behave deterministically: existing cached clients stay on their old pool (consistent with `SetTLSPolicy`); new clients take the freshly published pool.
- **Test pool fed by pointer, not by re-parse.** `Config.caCertPool` is set directly from `Manager.caCertPool` rather than re-parsing the PEM file in each call site; tests verify this with `assert.Same`.
- **`logger.Fatal` + returned error dual coverage.** `loadHuaweiCABundle` calls `a.logger.Fatal(...)` for production exit AND returns `fmt.Errorf("加载华为 TLS CA bundle 失败: %s: %w", path, err)`. The dual path means a test that swaps `fatalFunc` to panic via `defer recover` still sees the failure surface — preserves SEC-001 fail-closed pattern.

## Deviations from Plan

None — plan executed exactly as written. The gofmt -w absorption on `internal/huawei/client.go` was pre-existing column-alignment drift from Phase 23 work (MailboxState struct) that surfaced only when the success-criteria gate ran `gofmt -l` on touched files; folding it into the Task 1 commit keeps the 4-commit boundary intact without semantic change.

## Issues Encountered

- **`git stash --keep-index` was run mid-session for unrelated diagnostic reasons, leaving an unexpected mid-session state.** Recovered without data loss by using a scratch branch (`scratch/save-26-task3-staging`) plus `git cherry-pick` + interactive `--autosquash` rebase to rebuild the intended commit topology. No audit or payload data was lost; the only fallout was a mid-iteration commit message recovery that produced temporary wrong-message commits, then dropped during rebase.

## Next Phase Readiness
- Phase 24-04 (Nyquist verify): the ActivityWatcher/Coordinator tests now have a working Huawei HTTPS path that the recorder can prove live. Recommended to run `/gsd:execute-phase 24` next.
- Phase 25 (scheduler + E2E): once `GetConferenceState` is reachable, the H signal chain (DETECT-01) becomes exercisable end-to-end.
- No new environment setup required for this hotfix beyond deploying `certs/huawei-10.62.10.3-ca.pem` to the configured path (already gitignored; same operator flow as `server.crt`/`server.key`).

---

*Phase: 26-tls-ca-sec-003a-hotfix*
*Completed: 2026-08-06*

## Self-Check: PASSED

- File `26-01-SUMMARY.md` exists at `.planning/phases/26-tls-ca-sec-003a-hotfix/26-01-SUMMARY.md`
- All 4 commit hashes verified in git log: `c8ef568`, `f311e86`, `c2357d7`, `7818db5`
- Working tree clean post-SUMMARY commit

