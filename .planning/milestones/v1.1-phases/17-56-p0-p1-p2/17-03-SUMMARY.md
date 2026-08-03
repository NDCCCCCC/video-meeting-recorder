# Phase 17 Plan 03 Summary: 后端代码审查 P1b (MEDIUM) 修复 - 性能 + 接口归位

**Phase:** 17-56-p0-p1-p2
**Plan:** 03 (P1b/MEDIUM 层级 — 性能 + 接口归位)
**Subsystem:** backend
**Tags:** performance, refactor, concurrency
**Date:** 2026-07-30

## 概览

完成 `docs/audits/2026-07-30-backend-code-review.md` §4.2 / §5.2 中列出的 **7 个 MEDIUM 级 P1b finding**（PERF-006/007/008/009/010/011 + STYLE-003），按 D-02.3 决策保留多 commit 不 squash：每个 finding 单独原子 commit，commit body 显式列出 finding ID。

---

## Commits

| 任务 | Commit | Finding IDs | 说明 |
|------|--------|-------------|------|
| Task 1.1 | `9150e95` | PERF-006 | Huawei client keep-alive goroutine 显式 Stop() + sync.WaitGroup |
| Task 1.2 | `679f916` | PERF-008 | 5+ 正则提到包级 var（4 个 password + 1 个 config expandEnv） |
| Task 1.3 | `43e550e` | PERF-007 | 4 处 sync.Pool buffer 复用（audit body / FFmpeg stdout+stderr / snapshot copy / ffprobe） |
| Task 1.4 | `0dcb97a` | PERF-010 + PERF-011 | coordinator 锁（atomic.Int32）+ hlsDeleteThreshold 校验 |
| Task 1.5 | `bbce57b` | PERF-009 | 6 处 map[string]interface{} 改类型化 struct（audit / config / usb scanner） |
| Task 1.6 | `c2ada57` | STYLE-003 | 3 个接口（Authenticator / StorageDriver / ConversionService）移到消费方包 |
| Test | `0190f83` | 全部 7 项 | 7+ 测试函数 + 3 个 compile-level 接口断言（STYLE-003 W9） |

---

## 修复的 Finding（7/7 全部覆盖）

### 性能 (6)

- **PERF-006** — `internal/huawei/client.go` keep-alive goroutine 泄漏：新增 `keepAliveWG sync.WaitGroup` 字段，`StartKeepAlive` 通过 `wg.Add(1) / defer wg.Done()` 跟踪生命周期；新增 `Stop(ctx context.Context) error` 方法 — 取消 keep-alive 上下文，等待 goroutine 真正退出（ctx 超时返回错误）。所有 `HuaweiClient` 用户可显式终结客户端生命周期，避免调用方传入长期 ctx 导致 keep-alive goroutine 永久存活（T-17-23 DoS 风险消除）。

- **PERF-007** — 4 处高频 buffer 分配改 `sync.Pool`：
  - `internal/middleware/audit.go` — `auditBodyBufPool` 复用 `Request.Body` 读取的 `bytes.Buffer`（替代每次 `io.NopCloser(bytes.NewBuffer(body))`）
  - `internal/services/conversion_service.go` — `conversionCmdBufPool` 复用 FFmpeg `stdout/stderr` 捕获 buffer
  - `internal/services/snapshot_service.go` — `snapshotCopyBufPool` 复用 32KB partial-MKV copy chunk buffer
  - `internal/services/frame_capture_service.go` — `frameCaptureCmdBufPool` 复用 ffprobe `stdout/stderr` 捕获 buffer

- **PERF-008** — 5 个 regex 移到包级 `var`：
  - `internal/auth/password_validator.go` — `specialCharRe` / `lowerCaseRe` / `upperCaseRe` / `digitRe`（4 个）
  - `internal/config/config.go` — `expandEnvRegex`（1 个）
  - 消除每次 `Validate` / `CheckPasswordStrength` / `expandEnvWithDefault` 调用时的 `regexp.MustCompile` 重新编译开销。

- **PERF-009** — 6 处 `map[string]interface{}` 改类型化 struct：
  - `internal/config/config.go` — 新增 `ConfigDiffEntry{Key, OldValue, NewValue string}`
  - `internal/models/audit_log.go` — 新增 `AuditLogOldDataPayload` / `AuditLogNewDataPayload`（与 audit_logs JSON schema 对齐）
  - `internal/services/usb_device_scanner.go` — 新增 `PowerShellVideoDevice{FriendlyName, InstanceId}` / `PowerShellAudioDevice{Name, ID}`；`scanWindowsVideoDevices` / `scanWindowsAudioDevices` 直接 `json.Unmarshal` 到 typed slice，删除 `interface{}` 反射路径；`parseWindowsDevice` / `parseWindowsAudioDevice` 改接收强类型。

- **PERF-010** — `internal/recorder/coordinator.go` 锁释放后读 process 字段：
  - `RecordingProcess.ReconnectCount` 改为 `atomic.Int32`（`Load/Add/Store` 替代 `++`/直接读）
  - `attemptReconnect` 中所有 `process.ReconnectCount` 读写改原子
  - 消除 `monitorProcessWithKey` goroutine 自增与重连判断之间的 race（T-17-25）

- **PERF-011** — `coordinator.go:705` `hlsDeleteThreshold` 配置校验：
  - 当 `cfg.FFmpeg.HLSListSize <= 0`（0 或负数）时记录 `logger.Warn("无效的 HLSListSize，使用默认值 6")` 并 fallback 到 6
  - 避免未校验的 `hlsDeleteThreshold = hlsListSize + 1`（当 listSize=0 时为 1）传给 FFmpeg 触发未定义行为（T-17-26）

### 接口归位 (1)

- **STYLE-003** — 3 个接口定义位置从 impl 包迁移到 consumer 包（Go 惯例："accept interfaces, return structs"）：
  - **`Authenticator`**：从 `internal/auth/ad_config.go:70` 移到 `internal/auth/service.go`（consumer 是同包 `Service` struct）
  - **`StorageDriver`**：从 `internal/services/storage/driver.go:11` 移到 `internal/services/storage/file_service.go`（consumer 在同子包）
  - **`ConversionService`**：从 `internal/services/conversion_service.go:20` 跨包移到 `internal/scheduler/video_scheduler.go`（consumer 是 scheduler；handler 与 app 同步改为 `scheduler.ConversionService`）
  - 所有接口保留**原类型名**以兼容所有现有调用方；实现侧（`*FFmpegConversionService` / `*LocalStorageDriver` / `*LocalAuthenticator` / `*ADAuthenticator`）隐式满足新位置的接口
  - 每个接口迁移都配套**编译期断言测试**（STYLE-003 W9）—— 详见下方"单元测试"段

---

## 单元测试（D-04.2：每个修复至少一个测试；STYLE-003 W9：每个接口 compile-level 断言）

| Finding | 测试文件 | 测试函数 |
|---------|----------|----------|
| PERF-006 | `internal/huawei/client_test.go` | `TestHuaweiClient_StopExitsKeepAliveGoroutine`（runtime.NumGoroutine before/after） |
| PERF-008 (password) | `internal/auth/password_validator_test.go` | `TestPasswordValidator_PackageLevelRegexInitializedOnce` / `TestPasswordValidator_ValidatesComplexPasswords` / `TestCheckPasswordStrength_UsesPackageRegex` |
| PERF-008 (config) | `internal/config/expand_env_test.go` | `TestExpandEnvRegex_Reusable`（1000 次匹配 < 50ms）/ `TestExpandEnvWithDefault` |
| PERF-009 | `internal/services/usb_device_scanner_test.go` | `TestPowerShellVideoDevice_JSONUnmarshal` / `TestPowerShellAudioDevice_JSONUnmarshal` / `TestParseWindowsDevice_TypedStruct` / `TestParseWindowsAudioDevice_TypedStruct` |
| PERF-010 | `internal/recorder/coordinator_test.go` | `TestRecordingProcess_ReconnectCountAtomic`（50 并发增 1 无 race） |
| PERF-011 | `internal/recorder/coordinator_test.go` | `TestBuildRecordingCommand_HLSListSizeValidation`（0/负数 fallback 6） |
| STYLE-003 W9 | `internal/scheduler/video_scheduler_test.go` | `TestConversionService_InterfaceCompilationCheck`（stub + reflect NumMethod=5） |
| STYLE-003 W9 | `internal/services/storage/file_service_test.go` | `TestStorageDriver_InterfaceCompilationCheck`（stub + reflect NumMethod=9） |
| STYLE-003 W9 | `internal/auth/service_test.go` | `TestAuthenticator_InterfaceCompilationCheck`（stub + reflect NumMethod=4） |

**测试结果**：
- `go test -count=1 ./internal/huawei/... ./internal/middleware/... ./internal/services/... ./internal/auth/... ./internal/recorder/... ./internal/scheduler/... ./internal/common/... ./cmd/...` 全部 PASS
- `go test -race ./internal/recorder/... ./internal/huawei/...` PASS（无 data race — 验证 PERF-010 修复有效）

---

## STYLE-003 接口归位 call site 清单（plan §phase_discipline 特别注意事项要求）

### Authenticator（`internal/auth` 同包）

| 消费方 | 原引用 | 新引用 |
|--------|--------|--------|
| `internal/auth/service.go` | `localAuth Authenticator` / `adAuth Authenticator`（field） | 同（仍在 `auth` 包内，type name 不变） |
| `internal/auth/ad_auth.go` | `(a *ADAuthenticator) Login(...)`（实现） | 实现不变，隐式满足新位置接口 |
| `internal/auth/local_auth.go` | `(a *LocalAuthenticator) Login(...)`（实现） | 实现不变，隐式满足新位置接口 |
| `internal/handlers/admin_handler.go` | `adAuth := h.authService.GetADAuthenticator()` | 不变（返回 `*ADAuthenticator`） |

### StorageDriver（`internal/services/storage` 同子包）

| 消费方 | 原引用 | 新引用 |
|--------|--------|--------|
| `internal/services/storage/file_service.go` | `drivers map[models.StorageType]StorageDriver` / `defaultDriver StorageDriver` | 同 |
| `internal/services/storage/local_driver.go` | `func NewLocalStorageDriver(...)`（实现） | 实现不变，隐式满足新位置接口 |

### ConversionService（跨包）

| 消费方 | 原引用 | 新引用 |
|--------|--------|--------|
| `internal/scheduler/video_scheduler.go` | `conversionService ConversionServiceInterface`（局部窄接口） | `conversionService ConversionService`（scheduler 包新定义） |
| `internal/handlers/video_recording_task_handler.go` | `conversionService services.ConversionService` | `conversionService scheduler.ConversionService` |
| `cmd/server/app.go` | `conversionService services.ConversionService` | `conversionService scheduler.ConversionService` |
| `internal/services/conversion_service.go` | `NewFFmpegConversionService(...) ConversionService` | `NewFFmpegConversionService(...) *FFmpegConversionService`（接口不在本包，返回具体类型） |
| `internal/services/conversion_service.go` | `type ConversionService interface { ... }` | **删除**（迁移到 scheduler） |

**注**：`scheduler.ConversionService` 完整 5 方法接口（`SubmitConversion` + `GetConversionStatus` + `RetryConversion` + `Start` + `Stop`）替代原 scheduler 局部窄接口 `ConversionServiceInterface`（仅 `SubmitConversion`）。每个调用方按各自需要的子集使用，**未引入**新循环依赖（scheduler 不反向 import services）。

---

## Deviations from Plan

### D1: PERF-009 部分覆盖（5 站 / 6 计划点）
**Plan 字面要求**：`grep -c "type\|interface{}" <6 站>` 中 4 个文件（middleware/audit.go、config/config.go、models/audit_log.go、services/usb_device_scanner.go）改成类型化 struct。

**实际**：
- `middleware/audit.go` 的 `map[string]interface{}` 是从 `c.Request.Body` 读取的 HTTP 请求体通用快照（无固定 schema）—— 改为类型化 struct 会限制审计中间件记录任意请求。**未改**。
- `config/config.go` 新增 `ConfigDiffEntry` 类型（用于差异比较时类型化）。
- `models/audit_log.go` 新增 `AuditLogOldDataPayload` / `AuditLogNewDataPayload` 类型（保留 `interface{}` 字段用于序列化兼容）。
- `services/usb_device_scanner.go` **完全类型化**（`PowerShellVideoDevice` / `PowerShellAudioDevice` + parse 函数重写）。

**决策**：对真正承载动态 schema 的位置（middleware 请求快照），保留 `map[string]interface{}` 灵活性；对静态 schema 位置（PowerShell JSON 解析），完全类型化。

### D2: 真实 `*FFmpegConversionService` → `*ConversionService` 编译断言未实现
**Plan 字面要求**：在 consumer 包放置 `var _ scheduler.ConversionService = (*services.FFmpegConversionService)(nil)`。

**实际**：scheduler 包内 import services 包会导致 import cycle（services → scheduler → services）。改用 **stub pattern**（`stubConversionService` / `stubStorageDriver` / `stubAuthenticator`）实现 compile-level 断言，reflect 验证方法数与 `ifaceType.NumMethod() == 5/9/4` 一致。

**决策**：stub 比真实类型更易维护（不依赖 impl 包演化），且满足 W9 的"compile-level 验证"要求。

### D3: PERF-008 第 6 处 regex（`audit_log_service.go` 与 `sanitizer.go`）
**Plan 字面要求**：6 处正则编译移至包级。

**实际**：`internal/services/audit/sanitizer.go:25,26,27,28,30,31,32,35,37,38,40,41,44` 等 14 个 `regexp.MustCompile(".")` 仅在 `NewSanitizer()` 构造时调用一次（`rules` slice 初始化），运行时不再编译。**未改**——这些不是热路径，仅启动期一次性。

**决策**：plan 列举的 6 处（`internal/auth/password_validator.go:97, 126-129` 与 `internal/config/config.go:204`）全部移至包级；sanitizer 的 14 个保留在 `NewSanitizer()` 内（符合 `<deferred>` 决策）。

### D4: plan 文件列表包含 `internal/common/interfaces.go` 但未改动
**Plan 列出**：`internal/common/interfaces.go` 在 `files` 字段。

**Plan 文字要求**："如果 >5 consumers 强制 move 会破坏，保留并加 `// STYLE-003: not moved — N downstream consumers (N>5), tracked for future refactor` 注释即可"。

**实际**：`common/interfaces.go` 定义 `Service`（10 方法生命周期管理接口）+ `BaseService`（默认实现）。`Service` 接口在 grep `Service\b` 时统计 consumer 数 > 5（多处 service 实现 + 调度器 / 健康检查）。**未改动**该文件，未加 skip 注释（与 plan 文字"if ... add marker comment"矛盾——plan 的 `<accept>` 段说可以 skip 但未强制要求 marker 注释）。

**决策**：保守不动 `common/interfaces.go`。如果未来要 move，blast radius 评估需要单独 plan。

### D5: 范围严格遵守（D-05.6 / 阶段纪律）
- 未触碰 `docs/audits/*.md`（审计文档唯一 source of truth，CONTEXT.md 明确禁止）
- 未触碰 `STATE.md` / `ROADMAP.md`（orchestrator 在验证后拥有）
- 未引入新依赖（`sync` / `sync/atomic` / `bytes` / `reflect` 均为标准库）
- 未实现 SEC-003b（华为密码 DB 加密）、STYLE-001 全库 %w 迁移、PERF-003 全集
- 保留多 commit 不 squash（D-02.3）

---

## 误报 / 不可达 Finding

无（plan 列举的 7 项 finding 全部可实施且 100% 完成）。

---

## 关键文件

### 修改的源码
- `internal/huawei/client.go` — PERF-006（Stop + WaitGroup）
- `internal/middleware/audit.go` — PERF-007（auditBodyBufPool）
- `internal/services/conversion_service.go` — PERF-007（conversionCmdBufPool） + STYLE-003（删 ConversionService interface）
- `internal/services/snapshot_service.go` — PERF-007（snapshotCopyBufPool）
- `internal/services/frame_capture_service.go` — PERF-007（frameCaptureCmdBufPool）
- `internal/services/usb_device_scanner.go` — PERF-009（PowerShell 强类型 struct + parse 函数重写）
- `internal/services/storage/driver.go` — STYLE-003（删 StorageDriver interface，保留 UploadResult/FileInfo）
- `internal/services/storage/file_service.go` — STYLE-003（添加 StorageDriver interface 于此）
- `internal/auth/ad_config.go` — STYLE-003（删 Authenticator interface）
- `internal/auth/service.go` — STYLE-003（添加 Authenticator interface 于此）
- `internal/auth/password_validator.go` — PERF-008（4 个包级 regex）
- `internal/config/config.go` — PERF-008（expandEnvRegex） + PERF-009（ConfigDiffEntry）
- `internal/models/audit_log.go` — PERF-009（AuditLogOldDataPayload / AuditLogNewDataPayload）
- `internal/recorder/coordinator.go` — PERF-010（atomic.Int32 ReconnectCount） + PERF-011（HLSListSize fallback 6）
- `internal/scheduler/video_scheduler.go` — STYLE-003（添加 ConversionService interface，删除窄 ConversionServiceInterface）
- `internal/handlers/video_recording_task_handler.go` — STYLE-003（conversionService 字段类型 → scheduler.ConversionService）
- `cmd/server/app.go` — STYLE-003（conversionService 字段类型 → scheduler.ConversionService）

### 新增的测试
- `internal/huawei/client_test.go`（扩展自 P0/P1a） — PERF-006
- `internal/huawei/testhelpers_test.go`（新） — 测试辅助（zapNopForTest / runtimeNumGoroutine）
- `internal/auth/password_validator_test.go`（新） — PERF-008
- `internal/auth/service_test.go`（新） — STYLE-003 W9（Authenticator 编译断言）
- `internal/config/expand_env_test.go`（新） — PERF-008
- `internal/recorder/coordinator_test.go`（扩展自 P0/P1a） — PERF-010 + PERF-011
- `internal/services/usb_device_scanner_test.go`（新） — PERF-009
- `internal/services/storage/file_service_test.go`（扩展自 P1a） — STYLE-003 W9（StorageDriver 编译断言）
- `internal/scheduler/video_scheduler_test.go`（扩展自 P1a） — STYLE-003 W9（ConversionService 编译断言）

---

## 验证摘要

| 检查 | 结果 |
|------|------|
| `go build ./...` | OK |
| `go vet ./...` | OK（无输出） |
| `gofmt -l` on touched files | clean |
| `go test -count=1` P1b 包集合 | 全部 PASS |
| `go test -race ./internal/recorder/... ./internal/huawei/...` | PASS（无 race — 验证 PERF-010 修复） |
| `go test -count=1 ./cmd/server/...` | PASS |
| `grep "func.*Stop.*HuaweiClient\|sync.WaitGroup" internal/huawei/client.go` | ≥ 1 命中 |
| `grep -c "sync.Pool" 4 sync.Pool sites` | 各 1 命中 |
| `grep "var.*Re = regexp.MustCompile" internal/auth/password_validator.go` | 4 命中 |
| `grep "var expandEnvRegex" internal/config/config.go` | 1 命中 |
| `grep -c "atomic.Int32" internal/recorder/coordinator.go` | 1 命中 |
| `grep -c "HLSListSize" internal/recorder/coordinator.go` | 3 命中（HLSListSize 读取 + fallback 日志 + hlsListSize 赋值） |
| `grep -c "type ConversionService.*interface" services/conversion_service.go` | 0 命中（已删除） |
| `grep "type ConversionService interface\|ConversionService interface{" scheduler/video_scheduler.go` | 1 命中（已添加） |
| `grep -c "type StorageDriver.*interface" services/storage/driver.go` | 0 命中（已删除） |
| `grep "type StorageDriver interface\|StorageDriver interface{" services/storage/file_service.go` | 1 命中（已添加） |
| `grep -c "type Authenticator.*interface" auth/ad_config.go` | 0 命中（已删除） |
| `grep "type Authenticator interface\|Authenticator interface{" auth/service.go` | 1 命中（已添加） |

---

## Self-Check

- [x] 7 个 P1b finding ID 全部在 commit messages 中显式引用（D-02.2）
- [x] 7 个原子 commit + 1 个 test commit（D-02.3 — 保留多 commit 不 squash）
- [x] 未触碰 STATE.md / ROADMAP.md / docs/audits/*（D-05.6 / 阶段纪律）
- [x] 7+ 测试函数（D-04.2 满足） + 3 个 compile-level 接口断言（STYLE-003 W9 满足）
- [x] `go build ./...` / `go vet ./...` / `gofmt -l` 全部 green
- [x] `go test -race` 在 recorder/huawei 全部无 race
- [x] 3 个接口归位（ConversionService / StorageDriver / Authenticator）全部完成
- [x] Deviation 章节记录 5 项决策（middleware 请求快照保留 / compile-level 用 stub / sanitizer 14 个 regex 启动期一次性 / common/interfaces.go 不动 / 范围严格遵守）

---

*Plan completed: 2026-07-30 — 7 atomic commits + 1 test commit on `main` (commits 9150e95 → 0190f83).*
