# Requirements: Record V2 - v1.1 Milestone Traceability

- Defined: 2026-08-03 (retro-active)
- Milestone: v1.1 — 文件管理与编辑增强
- Scope: Phases 17, 18, 19, 20 (per ROADMAP Progress table; phase 16 ambiguity noted in `## Out-of-scope observation`)
- Method: REQ-ID backfilled from deliverables. SUMMARY frontmatter `requirements_completed` was empty for most phase-17/18/19/20 plans — per CONTEXT D-03.5 rows are backfilled from actual deliverables (git commits + live code + VERIFICATION.md cross-references) and labeled `"backfilled from deliverables, SUMMARY frontmatter was empty"` where applicable. No fabricated traceability.

### REQ-ID Convention

- Format: `REQ-<phase>-<source-id>` (e.g., `REQ-17-SEC-001`) — mirrors the audit's native finding IDs so any audit finding without a matching REQ-ID is detectable as an orphan.
- Cross-phase兑现 (e.g., SEC-003b deferred in phase 17, delivered in phase 18) keeps its original phase-17 ID with `Phase` column = `17→18` and `状态` column explicitly noting `"delivered by Phase 18"`. This preserves the audit's finding identity while making the cross-phase flow visible.
- Status vocabulary: `done` / `deferred` (explicit out-of-scope) / `partial` (some sites done, rest outstanding) / `N/A` (audit false positive or zero-occurrence finding) / `done (delivered by Phase NN)` (cross-phase兑现).

### 3-Source Cross-Reference Model

Each REQ-ID row is designed to be cross-checkable across three sources:
1. **REQUIREMENTS.md (this file)** — canonical REQ-ID + status + evidence pointer.
2. **ROADMAP.md Progress table** — phase-level completion status (e.g., "17. … 4/4 Complete 2026-07-30").
3. **`<phase>-PLAN.md` frontmatter `requirements:` field** — the planner's inbound REQ-ID contract (where populated; phase 17/18/19 plans pre-date this convention so most are empty, which is why backfilling is required).

A REQ-ID that appears here with status `done` but has no matching PLAN frontmatter entry AND no commit AND no VERIFICATION.md evidence is an **orphan** and fails the orphan-detection rules in `## Coverage`.

---

## v1.1 Requirements

### REQ-17-*: 后端代码审查 56 项修复 (Phase 17)

Source of truth: `docs/audits/2026-07-30-backend-code-review.md` (immutable). Each finding's native ID (SEC-001, BUG-001, PERF-001, STYLE-001) becomes a REQ-17-* row. SEC-003 is split into SEC-003a (TLS三项, done in phase 17) and SEC-003b (DB加密, deferred phase 17 → delivered phase 18) per 17-CONTEXT `<deferred>`. BUG-007/008/009/010/012/013/014 are listed in the audit as `0 处` (zero occurrences) — they are recorded as `N/A` so the orphan checker can verify no audit finding is silently dropped.

#### REQ-17-SEC-* (16 rows, SEC-003 split into a/b)

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-17-SEC-001 | 17 | audit §6.2 (SEC-001) | done | 17-VERIFICATION M1 + commit `4d3de0b` (`internal/config/config.go::ValidateProductionSecrets`) |
| REQ-17-SEC-002 | 17 | audit §6.2 (SEC-002) | done | 17-VERIFICATION M1 + commit `4d3de0b` (`cmd/server/app.go::authService.SetAuditService`) |
| REQ-17-SEC-003a | 17 | audit §6.2 (SEC-003a — TLS 三项: MinTLSVersion / cipher / cert verify) | done | 17-VERIFICATION M1 + commit `2bcee29` (`internal/huawei/manager.go::SetTLSPolicy`) |
| REQ-17-SEC-003b | 17→18 | audit §6.2 (SEC-003b — 华为密码 DB 加密) | done (delivered by Phase 18) | 18-VERIFICATION D18-1..D18-5 + commits `e6315ce` (SM4-GCM envelope) + `5d536ec` (production-decrypt invariant test); phase-17 live marker at `internal/huawei/manager.go:132-134` updated "SEC-003b/Phase 21: done" |
| REQ-17-SEC-004 | 17 | audit §6.2 (SEC-004) | done | 17-VERIFICATION M1 + commit `4d3de0b` (`internal/auth/hlstoken/hls_token.go` jti + RawURLEncoding) — full replay defense upgraded in phase 19 (see REQ-19-SEC-004) |
| REQ-17-SEC-005 | 17 | audit §6.2 (SEC-005 — SQL 字符串拼接) | done | 17-VERIFICATION M2 (P1a) + commit in W2 12-commit range `d27903f..b53cc8c` |
| REQ-17-SEC-006 | 17 | audit §6.2 (SEC-006 — MD5 文件指纹) | done | 17-VERIFICATION M2 + W2 commits |
| REQ-17-SEC-007 | 17 | audit §6.2 (SEC-007 — CORS 通配符 4 处) | done | 17-VERIFICATION M2 + W2 commits |
| REQ-17-SEC-008 | 17 | audit §6.2 (SEC-008 — CSRF 全局缺失) | done | 17-VERIFICATION M2 + W2 commits (Phase 21 Plan 05 normalized auth:57 as the last handler to use canonical `HandleError` pattern; see commit `4959e9c`) |
| REQ-17-SEC-009 | 17 | audit §6.2 (SEC-009 — 敏感信息日志 2 处) | done | 17-VERIFICATION M2 + W2 commits |
| REQ-17-SEC-010 | 17 | audit §6.2 (SEC-010 — Token URL query 泄露) | done | 17-VERIFICATION M2 + W2 commits |
| REQ-17-SEC-011 | 17 | audit §6.2 (SEC-011 — SHA256 截断派生 SM4 密钥) | done | 17-VERIFICATION M3 (P2) + W4 commits `4f5579a..72e2027` |
| REQ-17-SEC-012 | 17 | audit §6.2 (SEC-012 — 文件上传 MIME magic bytes) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-SEC-013 | 17 | audit §6.2 (SEC-013 — SSRF 4 个出站 URL) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-SEC-014 | 17 | audit §6.2 (SEC-014 — 公开路由依赖 handler 内校验) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-SEC-015 | 17 | audit §6.2 (SEC-015 — `gin.Context.GetString` 无 ok 检查) | done | 17-VERIFICATION M3 + W4 commits |

#### REQ-17-BUG-* (10 rows; BUG-007/008/009/010/012/013/014 are combined as N/A — audit reported 0 occurrences)

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-17-BUG-001 | 17 | audit §BUG-001 (RetryTask EndTime 真 bug) | done | 17-VERIFICATION M1 + commit `4d3de0b` |
| REQ-17-BUG-002 | 17 | audit §BUG-002 (8 个 fire-and-forget goroutine 缺 recover) | done | 17-VERIFICATION M1 + commit `4d3de0b` |
| REQ-17-BUG-003 | 17 | audit §BUG-003 (json.Unmarshal 错误被吞 6 处) | done | 17-VERIFICATION M2 (P1a) + W2 commits |
| REQ-17-BUG-004 | 17 | audit §BUG-004 (`_ =` 显式忽略 error 9 处) | done | 17-VERIFICATION M2 + W2 commits |
| REQ-17-BUG-005 | 17→19 | audit §BUG-005 (GORM 全部 403 处缺 `.WithContext(ctx)`) | done (delivered by Phase 19 — same root cause as PERF-003) | 17-VERIFICATION M2 + 19-VERIFICATION D19-1 (42 WithContext sites in `video_recording_task_service.go`) |
| REQ-17-BUG-006 | 17 | audit §BUG-006 (`time.Sleep` 不可被 ctx 取消 4 处) | done | 17-VERIFICATION M2 + W2 commits |
| REQ-17-BUG-011 | 17 | audit §BUG-011 (`int64(time.Now().UnixNano())` 拼接 5 处) | done | 17-VERIFICATION M3 (P2) + W4 commits |
| REQ-17-BUG-015 | 17 | audit §BUG-015 (注释与代码错位) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-BUG-016 | 17 | audit §BUG-016 (`IsIPAllowed` 不处理 IPv6-mapped IPv4) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-BUG-007..010/012..014 | 17 | audit §BUG-007..010/012..014 | N/A (audit 报告 `0 处` — these anti-patterns were enumerated as checklist items but reported as zero occurrences in the codebase at audit time) | 17-VERIFICATION §"N/A findings" — no code change required |

#### REQ-17-PERF-* (16 rows; PERF-003 split — phase 17 partial → phase 19 全量兑现)

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-17-PERF-001 | 17 | audit §PERF-001 (N+1 查询 + 3 重 Preload) | done | 17-VERIFICATION M1 (P0) + commit `4d3de0b` |
| REQ-17-PERF-002 | 17 | audit §PERF-002 (`.Find` 无 Limit 18 处) | done | 17-VERIFICATION M1 + W1 commits |
| REQ-17-PERF-003 | 17→19 | audit §PERF-003 (27 个长 DB 操作缺 WithContext) | partial (phase 17: 仅修改/新增处加 ctx, per 17-CONTEXT `<deferred>`) → done (phase 19: 全量 ~190 service 方法 ctx-first) | 17-VERIFICATION M1 (P0 note: "deferred to phase 19") + 19-VERIFICATION D19-1 (42 WithContext sites in `video_recording_task_service.go`, ~190 methods cascade-converted W3-W5) |
| REQ-17-PERF-004 | 17 | audit §PERF-004 (锁粒度过粗 3 处) | done | 17-VERIFICATION M1 + W1 commits |
| REQ-17-PERF-005 | 17 | audit §PERF-005 (Gin handler 同步重 IO 3 处) | done | 17-VERIFICATION M1 + W1 commits |
| REQ-17-PERF-006 | 17 | audit §PERF-006 (goroutine 泄漏 1 处) | done | 17-VERIFICATION M2 (P1b) + W3 commits `9150e95..0190f83` |
| REQ-17-PERF-007 | 17 | audit §PERF-007 (高频分配缺 `sync.Pool` 4 处) | done | 17-VERIFICATION M2 + W3 commits |
| REQ-17-PERF-008 | 17 | audit §PERF-008 (函数体内正则重复编译 6 处) | done | 17-VERIFICATION M2 + W3 commits |
| REQ-17-PERF-009 | 17 | audit §PERF-009 (`interface{}` 反序列化 6 处) | done | 17-VERIFICATION M2 + W3 commits |
| REQ-17-PERF-010 | 17 | audit §PERF-010 (`coordinator.go:218-247` 锁释放后继续读 process) | done | 17-VERIFICATION M2 + W3 commits |
| REQ-17-PERF-011 | 17 | audit §PERF-011 (`coordinator.go:705` hlsDeleteThreshold 缺配置校验) | done | 17-VERIFICATION M2 + W3 commits |
| REQ-17-PERF-012 | 17 | audit §PERF-012 (循环字符串拼接 1 处) | done | 17-VERIFICATION M3 (P2) + W4 commits |
| REQ-17-PERF-013 | 17 | audit §PERF-013 (同函数重复 `time.Now()` 1 处) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-PERF-014 | 17 | audit §PERF-014 (无缓冲 channel 误用 3 处) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-PERF-015 | 17 | audit §PERF-015 (数据库连接池配置缺失) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-PERF-016 | 17 | audit §PERF-016 (`time.After` 在 select 中 2 处) | done | 17-VERIFICATION M3 + W4 commits |

#### REQ-17-STYLE-* (10 rows)

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-17-STYLE-001 | 17→19/20 | audit §STYLE-001 (错误处理风格不统一) | partial (phase 17: 仅新修改/接触处用 `%w`; phase 19/20: handler 层与 high-frequency services 收敛; 全库 ~642 处 `errors.New`/`fmt.Errorf` 仍 deferred) | 17-VERIFICATION M3 + 19-VERIFICATION D19-3 (mapping.go + HandleError + error_mapper.go) + 19-VERIFICATION D19-8 (24 sentinels + ~356 散点) + 20-VERIFICATION (classify 全收敛) |
| REQ-17-STYLE-002 | 17 | audit §STYLE-002 (`auth → services/audit` 依赖方向) | N/A (false positive — audit §STYLE-002 自标 "误报'循环'"; 真正关联问题 SEC-002 已独立处理) | 17-04-SUMMARY §"误报" + audit §STYLE-002 自身说明 |
| REQ-17-STYLE-003 | 17 | audit §STYLE-003 (接口定义在实现方 3-4 处) | done | 17-VERIFICATION M2 (P1b) + W3 commits |
| REQ-17-STYLE-004 | 17 | audit §STYLE-004 (中间件 GetUserID 零值语义) | done | 17-VERIFICATION M2 (P1a) + W2 commits |
| REQ-17-STYLE-005 | 17 | audit §STYLE-005 (类型断言未守 ok) | done | 17-VERIFICATION M2 (P1a) + W2 commits |
| REQ-17-STYLE-006 | 17 | audit §STYLE-006 (`panic` 替代 error) | done | 17-VERIFICATION M3 (P2) + W4 commits |
| REQ-17-STYLE-007 | 17 | audit §STYLE-007 (`switch` 缺 default) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-STYLE-008 | 17 | audit §STYLE-008 (`defer conn.Close()` 缺 nil 防御) | done | 17-VERIFICATION M3 + W4 commits |
| REQ-17-STYLE-009 | 17 | audit §STYLE-009 (包名冗余 133 处 Get* rename) | deferred (per 17-CONTEXT Claude's Discretion 默认跳过: blast radius 大, API 破坏性, 不影响功能) | 17-VERIFICATION §Deferred — 133 `Get*` rename 不在本里程碑范围 |
| REQ-17-STYLE-010 | 17 | audit §STYLE-010 (godoc 缺失 8 处) | done | 17-VERIFICATION M3 + W4 commits |

### REQ-18-*: 凭据静态加密 + 密钥轮换 (Phase 18)

Phase 18's original PLAN.md / CONTEXT.md / DISCUSSION-LOG were never git-tracked and are permanently lost (data limitation, not deferred). REQ-18-* IDs are backfilled from deliverables (commits + 18-SUMMARY.md + STATE.md §Phase 18 + 18-VERIFICATION.md). Source paths in the `来源` column reference the in-phase-dir copy `.planning/phases/18-credential-static-encryption-sec-003b/18-SUMMARY.md` (root `18-SUMMARY.md` also tracked as canonical — see `## Canonical References`).

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-18-001 | 18 | 18-SUMMARY §"算法" | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 18-VERIFICATION D18-1 + commit `e6315ce` — SM4-GCM envelope format `SM4:<version>:<base64(nonce_12B|ciphertext|tag_16B)>` @ `internal/utils/sm4_password.go:19` (EncryptGCM/DecodeCredentialEnvelope/ParseCredentialEnvelope exports) + PKCS#7 padding for gmsm v1.4.1 GCMDecrypt block alignment |
| REQ-18-002 | 18 | 18-SUMMARY §"密钥族分离" | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 18-VERIFICATION D18-3 + commit `1dbb3b0` — `AuthConfig.CredentialSM4*` fields + `ValidateCredentialSM4Config` @ `internal/config/config.go`; credential-family isolation (`CREDENTIAL_SM4_*` vs `SM4_SECRET` vs `HLS_TOKEN_SECRET`) |
| REQ-18-003 | 18 | 18-SUMMARY §"启动期 fail-closed (10 步)" | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 18-VERIFICATION D18-6 + commit `bd84fe2` — `cmd/server/app.go::Initialize()` runs 10-step `MigratePlaintextToGCM` + double `InvariantScan` + `RotateIfNeeded`; fail-closed on any invariant violation |
| REQ-18-004 | 18 | 18-SUMMARY §Wave 4 §W4d | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 18-VERIFICATION D18-8 + commit `0c018f2` — `DEPLOYMENT.md` operator runbook (rotation procedure) + 物理残留章节 (WAL / vacuum / backup boundaries) |
| REQ-18-005 | 18 | post-audit (commit `5d536ec`, 2026-08-02) | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 18-VERIFICATION D18-7 + commit `5d536ec` — `cmd/server/app_test.go::TestHuaweiDBAdapter_ProductionDecrypts` (61 行 production-decrypt invariant: huaweiDBAdapter must decrypt SM4 envelopes under production config); `internal/huawei/manager.go:132-134` marker comment updated from "deferred" to "SEC-003b/Phase 21: done" |

### REQ-19-*: ctx 级联 + SEC-004 replay + STYLE-001 error (Phase 19)

Phase 19's original PLAN.md / CONTEXT.md / DISCUSSION-LOG were never git-tracked and are permanently lost (data limitation). REQ-19-* IDs are high-level (4 IDs covering the 4 deliverable families from 19-SUMMARY); the 21 `refactor(19/dN)` commits (D1-D21) provide finer-grained evidence under each. Source paths reference `.planning/phases/19-ctx-cascade-sec-004-style-001-error/19-SUMMARY.md` (root `19-SUMMARY.md` also canonical — see `## Canonical References`).

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-19-ctx | 19 | 19-SUMMARY §Wave 3-5 | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 19-VERIFICATION D19-1 + Wave 3-5 commits `213710c..a6c21b6` (W3 13 leaf/mid services) + `9a00cbe`+`2281927` (W4 TaskServiceInterface atomic triple) + `34b07f7`+`e2b0b6b`+`7828fc3`+`7a5a1cc`+`1ae6be0`+`b08255d` (W5 VideoRecordingTaskService 22 methods ctx-first + VideoFileService + callers + cancellation contract test) — live grep `grep -c ".WithContext(ctx)" internal/services/video_recording_task_service.go` = **42** |
| REQ-19-SEC-004 | 19 | 19-SUMMARY §Wave 1 | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 19-VERIFICATION D19-2 + commit `6fbdad4` — multi-segment HLS jti replay model rewrite; `TestVerify_MultiSegmentSameToken` (in-memory TTL sweeper, single-instance 5-min window risk accepted per user decision "不加 DB 表"); later hardened to DB-backed persistence via REQ-19-jti-db (D3) |
| REQ-19-STYLE-001 | 19 | 19-SUMMARY §Wave 2/6 + D5-D21 | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 19-VERIFICATION D19-3..D19-5 + commits `cacc294` (W2 STYLE-001 三组件 mapping.go + HandleError + error_mapper.go) + `3d171de` (W6 gorm wrap → BusinessError + HandleError on notification/ppt_file/video_file services and ppt/video rename handlers) + 21 dN commits D5-D21 (24 new sentinels in `knownSentinels` slice + ~356 散点 sentinel 化 across 13 services + 9 handlers + middleware + utilities) — see `docs/audits/phase-19-D5-D21-summary.md` |
| REQ-19-jti-db | 19 | 19-SUMMARY §D3 | done (backfilled from deliverables, SUMMARY frontmatter was empty) | 19-VERIFICATION D19-6 + commit `1f0ec35` — `internal/models/hls_jti_record.go::HLSJtiRecord` (table name `hls_jti_records`) + `cmd/server/app.go:340` `AutoMigrate(&models.HLSJtiRecord{})` registration + `internal/auth/hlstoken/hls_token.go:55,92,95` persistence callsites — upgrades REQ-17-HMAC-jti (phase 17 in-memory map) to cross-instance/restart persistence |

### REQ-20-*: HandleError 收敛 + sentinel 增强 (Phase 20)

Phase 20 is the only v1.1 phase with `requirements:` arrays in its PLAN frontmatter (20-01/20-05). These 11 REQ-20-* IDs are reused verbatim from `.planning/phases/20-handleerror-classify-convergence/20-01-PLAN.md` and `20-05-PLAN.md` frontmatter + 20-CONTEXT §D-22 (R-3/R-4/R-5/R-7 user-locked decisions). Verification evidence uniformly references `.planning/phases/20-handleerror-classify-convergence/20-VERIFICATION.md` (passed 10/10).

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-20a-classify | 20 | 20-01-PLAN frontmatter | done | 20-VERIFICATION (10/10) — 9 文件 27 处 ad-hoc classify 全替换为 `if response.HandleError(c, err) { return }` 规范模式 |
| REQ-20a-formal | 20 | 20-01-PLAN | done | 20-VERIFICATION — `classifyAuthLoginError` formal helper function deleted (replaced by sentinel-driven `errors.Is` chain in `mapping.go`) |
| REQ-20a-ad-user-not-registered | 20 | 20-01-PLAN (R-3 user-locked) | done | 20-VERIFICATION — `auth.ErrADUserNotRegistered` local var migrated into `internal/errors.ErrADUserNotRegistered`; 403 Forbidden semantics preserved (no regression to 500) |
| REQ-20b-sentinel-field | 20 | 20-01-PLAN frontmatter | done | 20-VERIFICATION — `pkg/response/sentinel.go::SentinelField(err) zap.Field` 4-state contract (sentinel → `sentinel_type="ErrXxx"` / BusinessError → `BusinessError(code=yyy)` / unknown → `ad-hoc` / nil → `zap.Skip()`) |
| REQ-20b-priority | 20 | 20-01-PLAN frontmatter | done | 20-VERIFICATION — `internal/errors.FirstKnownSentinelName` mirrors `IsKnownError` slice order (first `errors.Is` hit wins) |
| REQ-20b-upgrade | 20 | 20-01-PLAN | done | 20-VERIFICATION — 160+ `zap.Error(err)` sites upgraded to `zap.Error(err).Add(zap.Any("sentinel", response.SentinelField(err)))` pattern (logger now emits structured sentinel_type) |
| REQ-20c-generator | 20 | 20-05-PLAN frontmatter | done | 20-VERIFICATION — `cmd/error-doc-gen/main.go` standalone binary + `cmd/error-doc-gen/main_test.go` 4-case test (sentinel table / BusinessError table / missing filepath / diff stability); deterministic output |
| REQ-20c-doc-sync | 20 | 20-05-PLAN frontmatter | done | 20-VERIFICATION — `//go:generate go run ./cmd/error-doc-gen` directive @ `internal/errors/errors.go` + `.github/workflows/ci.yml` CI step fails if `git diff --quiet docs/errors.md` returns non-zero (no Makefile per user-locked R-2) |
| REQ-20-regression | 20 | 20-01-PLAN | done | 20-VERIFICATION — all pre-existing tests pass (`go test -race ./...` green); TestLogin_HandleError_ClassifyDrop 10 sub-tests continue to validate HandleError write contract |
| REQ-20-build | 20 | 20-01-PLAN | done | 20-VERIFICATION — `go build ./...` + `go vet ./...` green; no gofmt drift on touched files |
| REQ-20-typed-kind | 20 | 20-CONTEXT §D-01.1 | **deferred** | Explicitly out-of-scope per 20-CONTEXT D-01.1: typed error kind enum (Sentinel vs BusinessError vs ad-hoc) deferred. `SentinelField` returns `zap.Field` (string-valued, not typed enum) to preserve future extensibility per R-6 — when this deferred item lands, a separate REQ-ID (e.g., REQ-20d-typed-kind) should be created in a follow-up phase rather than reusing this row |

---

## Coverage

- v1.1 requirements: **~80 total** (58 REQ-17-* counting SEC-003a/b split + 5 REQ-18-* + 4 REQ-19-* + 11 REQ-20-* + cross-phase兑现 annotations)
- Mapped to phases: ~80 (100%)
- **Orphans: 0 ✓** (every audit finding has a matching REQ-ID row; every REQ-ID has at least one evidence pointer)
- **Deferred (explicit, not orphans): 4**
  - REQ-17-STYLE-001 全库 %w 迁移 (~642 处 `errors.New` / `fmt.Errorf` 散点)
  - REQ-17-STYLE-009 Get* rename (133 处, blast radius 大)
  - REQ-17-PERF-003 全库 ctx 级联 — **partial** in phase 17, **done** in phase 19 (REQ-19-ctx) — full-library residual sites may exist outside the 11 high-frequency service files but the audit's 27 long-DB-op list is fully covered
  - REQ-20-typed-kind (typed error kind enum — future extensible)
- **Delivered by other phase (cross-phase兑现, not orphans): 3**
  - REQ-17-SEC-003b → Phase 18 (REQ-18-001..005)
  - REQ-17-PERF-003 → Phase 19 (REQ-19-ctx)
  - REQ-17-BUG-005 (GORM WithContext 403 处) → Phase 19 (REQ-19-ctx, same root cause as PERF-003)
  - (HMAC jti in-memory map from phase 17 → `hls_jti_records` DB table in phase 19 D3, tracked as REQ-19-jti-db rather than a REQ-17-* row)
- **N/A (audit false positive or zero-occurrence): 8**
  - REQ-17-STYLE-002 (audit §STYLE-002 self-marked 误报)
  - REQ-17-BUG-007/008/009/010/012/013/014 (audit reported `0 处` — these anti-patterns were enumerated as checklist items but reported as zero occurrences in the codebase at audit time)

### Orphan 检测规则

The following 4 grep rules let any future auditor verify coverage is intact (run from repo root):

1. **Audit findings → REQ-17-* rows:** count distinct finding IDs in `docs/audits/2026-07-30-backend-code-review.md` and confirm each has a row above.
   ```bash
   grep -oE '(SEC|BUG|PERF|STYLE)-[0-9]{3}' docs/audits/2026-07-30-backend-code-review.md | sort -u | wc -l
   # Expected: 57 distinct finding IDs (SEC-001..015 + BUG-001..016 + PERF-001..016 + STYLE-001..010)
   # Plus SEC-003 split = 58 REQ-17-* coverage rows (matches this file's REQ-17-* section)
   ```

2. **Phase 18 deliverables → REQ-18-* rows:** the 18-VERIFICATION `must_haves` (D18-1..D18-11) and 18-SUMMARY deliverables must each map to a REQ-18-* ID. Minimum 5 rows (this file has 5).
   ```bash
   grep -cE '^\| REQ-18-' .planning/REQUIREMENTS.md  # Expected: >= 5
   ```

3. **Phase 19 deliverables → REQ-19-* rows:** the 4 high-level deliverables in 19-SUMMARY (ctx cascade / SEC-004 / STYLE-001 / jti-db) must each map to a REQ-19-* ID. Minimum 4 rows (this file has 4).
   ```bash
   grep -cE '^\| REQ-19-' .planning/REQUIREMENTS.md  # Expected: >= 4
   ```

4. **Phase 20 PLAN frontmatter → REQ-20-* rows:** every REQ-ID appearing in `.planning/phases/20-handleerror-classify-convergence/20-01-PLAN.md` or `20-05-PLAN.md` frontmatter `requirements:` field must appear in this file. Minimum 11 rows (this file has 11, including the deferred REQ-20-typed-kind from 20-CONTEXT D-01.1).
   ```bash
   grep -cE '^\| REQ-20' .planning/REQUIREMENTS.md  # Expected: = 11
   for id in REQ-20a-classify REQ-20a-formal REQ-20a-ad-user-not-registered REQ-20b-sentinel-field REQ-20b-priority REQ-20b-upgrade REQ-20c-generator REQ-20c-doc-sync REQ-20-regression REQ-20-build REQ-20-typed-kind; do
     grep -q "$id" .planning/REQUIREMENTS.md || echo "ORPHAN: $id"
   done
   ```

---

## Out-of-scope observation

**Phase 16 (visual-reshape) 归属歧义 — 本表不强行裁定.**

Phase 16 目录存在 (`.planning/phases/16-visual-reshape/16-01-SUMMARY.md`) 但**不在 v1.1 ROADMAP Progress 表** (表中 v1.1 部分只列 phase 17/18/19/20 — see `.planning/ROADMAP.md` lines 52-55). 然而 `gsd-sdk query init.milestone-op` 报告的 `phase_count: 5` 可能含 phase 16, 形成归属歧义.

Per CONTEXT D-03.4, 本表**不强行裁定** phase 16 是否属于 v1.1 milestone, 留待 milestone 决策:
- 如 milestone 决策**纳入** phase 16: 应另开 phase (e.g., phase 22) 补 phase 16 REQ-ID 追溯 (类似 phase 21 为 17/18/19 补 retro-verify + REQUIREMENTS.md 的模式).
- 如 milestone 决策**排除** phase 16: ROADMAP Progress 表保持现状 (phase 16 不在 v1.1 行), `init.milestone-op` 的计数偏差应作为 GSD 工具改进项独立追踪.

本表当前**仅覆盖 ROADMAP Progress 表中显式列出的 4 个 v1.1 phase (17/18/19/20)**, 不为 phase 16 创建任何 REQ-16-* 行, 也不删除 phase 16 目录. 这是观察记录, 不是裁定.

---

## Canonical References

This table draws evidence from the following sources. All paths are the canonical records of truth (immutable unless noted).

### Audit reports (immutable — DO NOT MODIFY)
- `docs/audits/2026-07-30-backend-code-review.md` — phase 17's 56-finding source of truth for all REQ-17-* IDs (SEC/BUG/PERF/STYLE series)
- `docs/audits/phase-19-D5-D21-summary.md` — phase 19 D5-D21 incremental sentinel-ization summary (8.5 KB, 21 commits + 24 sentinels + ~356 散点)

### SUMMARY records (two canonical paths per phase 18/19 — root = git-tracked history, `.planning/phases/` copy = self-contained phase dir for VERIFICATION co-location)
- Root (git-tracked, do not move): `18-SUMMARY.md` (21.7 KB) — phase 18 historical record
- Root (git-tracked, do not move): `19-SUMMARY.md` (33 KB) — phase 19 historical record
- Phase-dir copy: `.planning/phases/18-credential-static-encryption-sec-003b/18-SUMMARY.md` (identical content to root)
- Phase-dir copy: `.planning/phases/19-ctx-cascade-sec-004-style-001-error/19-SUMMARY.md` (identical content to root)
- Phase 17 SUMMARYs (in-place, git-tracked): `.planning/phases/17-56-p0-p1-p2/17-{01,02,03,04}-SUMMARY.md`
- Phase 20 SUMMARYs (in-place, git-tracked): `.planning/phases/20-handleerror-classify-convergence/20-{01,02,03,04,05}-SUMMARY.md`

### VERIFICATION reports (4 sources — Wave 1 outputs of phase 21 + pre-existing phase 20)
- `.planning/phases/17-56-p0-p1-p2/17-VERIFICATION.md` — phase 17 retro-verify, status: passed (7/7 must-haves), reconstructed by phase 21 plan 21-01 (commit `2c679f2`)
- `.planning/phases/18-credential-static-encryption-sec-003b/18-VERIFICATION.md` — phase 18 retro-verify, status: passed (11/11 must-haves), reconstructed by phase 21 plan 21-02 (commit `d76d47d`)
- `.planning/phases/19-ctx-cascade-sec-004-style-001-error/19-VERIFICATION.md` — phase 19 retro-verify, status: passed (10/10 must-haves), reconstructed by phase 21 plan 21-03 (commit `4b52463`)
- `.planning/phases/20-handleerror-classify-convergence/20-VERIFICATION.md` — phase 20 in-phase verification, status: passed (10/10 must-haves), pre-existing (commit predates phase 21)

### PLAN frontmatter (REQ-20-* source-of-truth)
- `.planning/phases/20-handleerror-classify-convergence/20-01-PLAN.md` — frontmatter `requirements:` defines REQ-20a-classify / REQ-20a-formal / REQ-20a-ad-user-not-registered / REQ-20b-sentinel-field / REQ-20b-priority / REQ-20b-upgrade / REQ-20-regression / REQ-20-build
- `.planning/phases/20-handleerror-classify-convergence/20-05-PLAN.md` — frontmatter `requirements:` defines REQ-20c-generator / REQ-20c-doc-sync
- `.planning/phases/20-handleerror-classify-convergence/20-CONTEXT.md` §D-22 + §D-01.1 — REQ-20-typed-kind (deferred) source

### Phase CONTEXT records (must_haves derivation)
- `.planning/phases/17-56-p0-p1-p2/17-CONTEXT.md` §D-01.4 (P0 11 项 + P1 18 项 + P2 25 项 分级) + `<deferred>` 段
- `.planning/phases/20-handleerror-classify-convergence/20-CONTEXT.md` §D-22 candidates + R-3/R-4/R-5/R-7 user-locked decisions

### Project state records
- `.planning/STATE.md` §Phase 17 / §Phase 18 / §Phase 19 / §Phase 19 Final Status / Phase 20 Context Captured — authoritative state log for each phase's scope / base HEAD / deviations / final status
- `.planning/ROADMAP.md` Progress table (lines 52-55) + per-phase `**Goal:**` text — retro-verify goal-backward starting point + phase completion record

### Audit reports driving this REQUIREMENTS.md
- `.planning/v1.1-MILESTONE-AUDIT.md` — original gaps_found audit whose "REQUIREMENTS.md missing" + "orphan detection impossible" gap is closed by this file (status: gaps_found → expected passed when re-audited)

---

*Requirements defined: 2026-08-03 (retro-active from audit gaps_found)*
*Last updated: 2026-08-03 after phase 21 retro-verify (21-01/02/03 complete + this file created by 21-04)*
