# Phase 23: 华为 API 扩展 + GORM 字段 + sentinel 错误码 - Research

**Researched:** 2026-08-06
**Domain:** Go 后端基础设施（华为 HTTP API、GORM/SQLite、Viper 配置、sentinel 错误文档门禁）
**Confidence:** HIGH（代码库事实与锁定 PRD 为主；TE40 字段语义仅由项目 PRD 转述，见风险）

## User Constraints

无独立 `23-CONTEXT.md`；以下约束来自 `STATE.md`、`ROADMAP.md`、`REQUIREMENTS.md` 与用户给定 phase context。

### Locked Decisions

- Phase 23 仅做基础设施：DETECT-01/04、AUDIT-01/05、CFG-01/02；不实现 Phase 24 watcher/轮询 goroutine，也不实现 Phase 25 scheduler/service/E2E。`STATE.md:105-128`、`REQUIREMENTS.md:83-94` [VERIFIED: codebase read]
- H 信号判据锁定为 `confState=="" && joinSum==0`，持续阈值默认 30 秒；老设备字段缺失时 fallback 到 `IsInConf==0`。`REQUIREMENTS.md:13-16` [VERIFIED: codebase read]
- GORM 变更走启动期 `AutoMigrate`，不进入 dormant `runCustomMigrations`。`STATE.md:92-94`、`cmd/server/app.go:317-342` [VERIFIED: codebase read]
- 新增 sentinel 必须同步生成 `docs/errors.md`；CI 通过 `go generate ./internal/errors/...` 后检查 diff。`.github/workflows/ci.yml:53-60` [VERIFIED: codebase read]
- 不改任务状态枚举、不改前端、不接 `MSG_CONF_STATE_CHANGE`、不重写 scheduler 框架。`REQUIREMENTS.md:83-94` [VERIFIED: codebase read]
- 默认 `extend_step_min=30`、`max_extend_count=4`，累计延时上限 2 小时。`STATE.md:107-114` [VERIFIED: codebase read]
- `.planning/` 被 gitignore；若后续提交研究/计划文档需 `git add -f`。`STATE.md:115-120` [VERIFIED: codebase read]

### Claude's Discretion

- 设计 `GetConferenceState` 的返回类型、字段存在性检测方式和可测试边界。
- 为 14 项配置确定字段类型、默认值落点和最小合理校验。
- 为新增模型字段确定 GORM 标签与 schema/read-write 测试组织。
- 为 3 个 sentinel 确定 HTTP 分类；本研究建议全部 500，且加入 recognition 列表。

### Deferred Ideas (OUT OF SCOPE)

- Phase 24：H 轮询、30 秒 persistence 状态机、silencedetect、文件停滞、降级状态机。
- Phase 25：scheduler 多信号 select、自动延时、提前结束 service/audit、E2E、开关运行时接线和可观测日志。
- `MSG_CONF_STATE_CHANGE`、T.140、跨 input 软结束、前端展示、ML 预测。`REQUIREMENTS.md:71-79` [VERIFIED: codebase read]

## Project Constraints (from CLAUDE.md)

- 项目是视频切割与会议转录 PPT 系统。`CLAUDE.md:1-4` [VERIFIED: codebase read]
- 后端主要目录为 `internal/`，其中 `auth/`、`models/`、`services/` 是既有分层。`CLAUDE.md:10-17` [VERIFIED: codebase read]
- 项目声明自动技能 `spike-findings-record-v2`，内容针对 Windows AD；与本阶段技术域无直接约束，未发现仓库内 skill 文件可进一步加载。`CLAUDE.md:6-8` [VERIFIED: codebase read]

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DETECT-01 | `GetConferenceState()` 识别空 `confState` + `joinSum==0`，持续阈值默认 30s | 明确 mailbox 双层 JSON、字段存在性、纯数据 API 与 Phase 24 persistence 边界 |
| DETECT-04 | 字段缺失时 fallback `IsInConf==0` | 建议用指针字段保留“缺失”语义，避免零值把新设备空状态与老设备缺字段混淆 |
| AUDIT-01 | `video_recording_tasks` 加 5 字段并走 AutoMigrate | 给出标签、startup migration 位置、SQLite schema/read-write 测试 |
| AUDIT-05 | 3 sentinel + `docs/errors.md` 同步 | 明确 errors.go、mapping.go、mapping_test.go、go generate、CI 门禁的完整同步面 |
| CFG-01 | `SmartEndConfig` 14 项 + 默认值 | 给出结构、挂载到根 Config、defaults 与 validation 建议 |
| CFG-02 | `config.yaml` 增加 `smart_end:` 14 项 | 指出根配置与发布配置 `bin/config.yaml` 双份模板风险及精确默认值 |
</phase_requirements>

## Summary

Phase 23 应拆成四个可独立验证的基础任务：华为 mailbox 数据契约、任务模型 schema、错误词汇表、智能收尾配置。当前项目已有所需全部库和基础设施，不需增加依赖：Huawei API 已通过 `GetMailboxData` 调用目标 ActionID，任务模型已经在 startup `AutoMigrate` 列表中，Viper 已提供嵌套 YAML 解码和集中默认值，错误文档已有 generator + CI sync-check。`go.mod:4-22` [VERIFIED: codebase read]

最关键的规划纠偏是：**不要把 30 秒 persistence 状态存进 `HuaweiClient.GetConferenceState()`**。客户端应是无状态的数据采集边界，返回一次采样及字段是否存在；持续时间由 Phase 24 的 watcher/poller 使用 timestamp 管理。否则并发任务共享同一 client 时会互相污染空状态起始时间，且 Phase 23 会越界实现轮询。另一个关键点是必须区分 JSON 字段“缺失”和“存在但值为零/空串”；当前 `int/string` 零值无法表达这种区别。

**Primary recommendation:** Phase 23 用“无状态、显式字段存在性”的 `ConferenceState` 契约打通 H 信号；同时完成 GORM schema、错误生成链、Viper defaults 的完整单元/集成测试，不启动 watcher 或 scheduler 行为。

## Phase 23 Goal (recap)

落地 H 信号（主业务权威）的数据通路与可观测基线：一次调用能解析 `confState/joinSum` 并暴露新字段是否可用；数据库可持久化五个审计字段；错误体系识别三个新 sentinel 并自动生成文档；应用配置可加载 `smart_end` 的十四项配置与默认值。`ROADMAP.md:64-73` [VERIFIED: codebase read]

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| TE40 mailbox 状态采集 | API / Backend（Huawei client） | 外部 Huawei TE40 | client 负责 HTTP 与 JSON 数据契约，不负责跨采样状态机 |
| 30s 持续判定契约 | API / Backend（Phase 24 watcher） | Huawei client | client 返回样本；watcher 持有每个录制实例的时间状态，Phase 23 只为其准备接口 |
| 智能收尾审计字段 | Database / Storage（GORM model） | API / Backend | 字段归任务聚合持久化，业务更新方法延后到 Phase 25 |
| sentinel 词汇表 | API / Backend（internal/errors） | CI/docs | 代码定义是 source of truth，generator 派生文档 |
| `smart_end` 配置 | API / Backend（internal/config） | 部署 YAML | 强类型结构、默认值与校验归 config 包；YAML 是运维输入 |

## Codebase Findings (what exists, where, current state)

### Huawei mailbox API

- `MailboxState.State` 当前只有 9 个具体值字段；`IsInConf` 位于 `internal/huawei/client.go:201-214`。[VERIFIED: codebase read]
- `GetMailboxData(ctx)` 已调用 `POST WEB_GetMailboxDataAPI`，携带 `SessionID` cookie。`internal/huawei/client.go:702-715` [VERIFIED: codebase read]
- 响应有两层 JSON：外层 `APIResponse.Data` 是字符串，字符串内有 `state`，再将 `state` raw message 解入 `mailboxState.State`。`internal/huawei/client.go:724-739` [VERIFIED: codebase read]
- 当前解析错误被静默吞掉并返回零值 `MailboxState,nil`。这对会议结束判定不可接受，因为 malformed/empty data 会伪装成 `confState="", joinSum=0, IsInConf=0`。`internal/huawei/client.go:724-743` [VERIFIED: codebase read]
- 现有 `IsInConference` 判定实际上是 `IsInConf==1 && Callstate==2`；需求所谓 fallback `IsInConf==0` 是结束判据，而不是复用 `IsInConference()` 的完整反逻辑。`internal/huawei/client.go:746-753` [VERIFIED: codebase read]
- `GetConferenceInfo()` 是老设备 stub，固定返回空会议信息，不应拿来实现 H 信号。`internal/huawei/client.go:855-869` [VERIFIED: codebase read]
- `HuaweiClient` 是 `Client` 的兼容 alias；新 API 可继续挂在 `*HuaweiClient` 以保持调用风格，但新接口定义应优先使用 `Client` 命名。`internal/huawei/client.go:411-428` [VERIFIED: codebase read]
- 现有 Huawei 测试只有 sanitizer 与 keepalive goroutine 退出测试，没有 mailbox JSON fixture。`internal/huawei/client_test.go:1-59` [VERIFIED: codebase read]

### GORM model and migration

- `VideoRecordingTask` 已映射到 `video_recording_tasks`；现有字段采用 PascalCase Go 名 + snake_case 自动列名，数值常显式 `gorm:"default:0"`，长原因文本采用 `type:text`。`internal/models/video_recording_task.go:9-42,132-135` [VERIFIED: codebase read]
- startup migration 在 `cmd/server/app.go:309-355`，`&models.VideoRecordingTask{}` 已在 AutoMigrate 列表中。因此此次只加 struct 字段即可让启动时补列，**无需**新增 migration registry 条目。`cmd/server/app.go:317-342` [VERIFIED: codebase read]
- 生产数据库配置是 SQLite，不是 PostgreSQL；验证不应写 `psql DESC`。根/发布配置均为 `driver: sqlite`。`config.yaml:17-25`、`bin/config.yaml:17-26` [VERIFIED: codebase read]
- 服务测试已有 SQLite `:memory:` + `db.AutoMigrate(&models.VideoRecordingTask{})` 模式，可直接作为 schema/read-write 测试 analog。`internal/services/video_recording_task_service_test.go:18-29` [VERIFIED: codebase read]

### Error sentinels and docs sync

- sentinel 定义集中在 `internal/errors/errors.go` 的单个 var block，模式为 `ErrXxx = errors.New("...")`。`internal/errors/errors.go:10-97` [VERIFIED: codebase read]
- 仅在 `errors.go` 定义不够：`IsKnownError`/logging 依赖 `mapping.go` 的 `knownSentinels` 列表。若不加入列表，`response.HandleError` 会将它们视为 unknown。`internal/errors/mapping.go:125-202` [VERIFIED: codebase read]
- `MapToHTTPStatus` 的 default 是 500；generator 也会把未显式 case 的 sentinel 记录为 default 500。`internal/errors/mapping.go:89-95`、`cmd/error-doc-gen/main.go:193-203` [VERIFIED: codebase read]
- `mapping_test.go` 手工枚举 recognition 名称和部分 HTTP mapping，因此新增项必须扩充测试。`internal/errors/mapping_test.go:12-53,130-220` [VERIFIED: codebase read]
- `errors.go` 上的 `go:generate` 调用 `cmd/error-doc-gen`，扫描 `errors.go`、`mapping.go` 并覆盖 `docs/errors.md`。`internal/errors/errors.go:0-3`、`cmd/error-doc-gen/main.go:140-219` [VERIFIED: codebase read]
- CI 执行 generator 后要求 `docs/errors.md` 无 diff；不能手工只编辑表格。`.github/workflows/ci.yml:53-60` [VERIFIED: codebase read]

### Config loading

- 根 `Config` 当前没有 `SmartEnd` 字段；每个子结构使用 `mapstructure/json/yaml` 三套一致 tag。`internal/config/config.go:25-42` [VERIFIED: codebase read]
- `Load()` 读取根目录 `config.yaml`，二次展开环境变量，再 `v.Unmarshal(&cfg)`，最后调用集中式 `setDefaults(&cfg)`。`internal/config/config.go:327-401` [VERIFIED: codebase read]
- 默认值不是通过 Viper `SetDefault` 在生产代码中设置，而是在 `setDefaults` 中按零值补齐。`internal/config/config.go:404-458,569-644` [VERIFIED: codebase read]
- 仓库存在两份部署模板：`config.yaml` 与 `bin/config.yaml`，当前关键段基本同步。只改根文件会导致打包部署配置缺项。`config.yaml:65-106`、`bin/config.yaml:66-107` [VERIFIED: codebase read]
- 当前 `Config` 没有通用 `Validate()`；只发现安全关键凭据的专用校验。Phase 23 最适合新增 `SmartEndConfig.Validate()` 并在 `Load()` defaults 后调用，或至少添加纯函数单测供 Phase 25 接线。`internal/config/config_test.go:153-294` [VERIFIED: codebase read]

## Technical Approach (how to implement each requirement)

### DETECT-01 / DETECT-04 — 无状态 conference sample API

**推荐数据结构**：[ASSUMED]

```go
// internal/huawei/client.go
type ConferenceState struct {
    ConfState       string
    JoinSum         int
    ConfLeftTime    int
    HasConferenceFields bool
    IsInConf        int
}

func (c *Client) GetConferenceState(ctx context.Context) (*ConferenceState, error)
```

实现要点：

1. 扩展 mailbox state 至 PRD 指定 8 个字段；对决定 fallback 的 `confState` 和 `joinSum` 使用 `*string`、`*int`，或自定义 `UnmarshalJSON` 记录 presence。不能仅看零值。[ASSUMED]
2. `GetConferenceState` 只进行一次 `GetMailboxData` 采样，并返回：
   - 新字段二者均存在：`HasConferenceFields=true`，后续 watcher 用 `ConfState=="" && JoinSum==0`；
   - 任一新字段缺失：`HasConferenceFields=false`，后续 watcher fallback `IsInConf==0`；
   - malformed/empty response：返回 error，且 `%w` 包装 `apperrors.ErrRecordingHuaWeiStateFetchFailed`，绝不能返回“已结束”零值。[ASSUMED]
3. 不在 client 内保留 `emptySince`、timer、mutex 或 goroutine。30 秒 persistence 必须留给 Phase 24 的每-watcher 状态；Phase 23 的测试仅证明 sample/fallback 元信息与错误语义。[ASSUMED]
4. 保守判据：新字段存在但 `confState` 非空、哪怕 `joinSum==0` 也不判结束；新字段存在时不再混用 `IsInConf`。`docs/plans/2026-08-05-smart-meeting-recording-end-design.md:340-346` [VERIFIED: codebase read]
5. 为可测性，优先把“解析 `APIResponse.Data`”抽成 package-private 纯函数，例如 `parseMailboxState(data string) (*MailboxState,error)`；不要为了测试真实 client 私有 session/http 细节而启动 TE40。[ASSUMED]

**错误边界：** `GetMailboxData` 的 transport/session 错误原样作为 cause 保留，`GetConferenceState` 统一加新 sentinel；同时修正 mailbox 解析失败的静默成功。这样 Phase 24 可通过 `errors.Is(err, ErrRecordingHuaWeiStateFetchFailed)` 计数降级，同时日志仍保留 root cause。[ASSUMED]

**30 秒 persistence 的规划接口（Phase 24 使用，不在本阶段实现）：** 每个 watcher 在第一次 empty sample 时记 `emptySince=now`；后续连续 empty 且 `now-emptySince >= HuaweiPersistS` 才触发；任意 active sample 清零；错误 sample 不应既清零也触发，失败次数另计。这避免轮询间隔 30s 时用“连续两次”等价替代时间持续，且便于 fake clock 测试。[ASSUMED]

### AUDIT-01 — 五字段 GORM schema

在 `VideoRecordingTask` 的录制/转换元数据附近加入：[ASSUMED]

```go
ExtensionCount       int    `gorm:"not null;default:0" json:"extension_count"`
LastExtensionReason string `gorm:"type:text;not null;default:''" json:"last_extension_reason,omitempty"`
EndedEarly           bool   `gorm:"not null;default:false" json:"ended_early"`
EndedEarlyReason     string `gorm:"type:text;not null;default:''" json:"ended_early_reason,omitempty"`
EndedByHuaWeAPI      bool   `gorm:"not null;default:false" json:"ended_by_huawei_api"`
```

- 列名由 GORM naming strategy 自动得到 `extension_count`、`last_extension_reason`、`ended_early`、`ended_early_reason`、`ended_by_hua_we_api`。注意 acronym `HuaWeAPI` 的实际 snake_case 需要测试确认；若产品/SQL 期望 `ended_by_huawei_api`，必须显式 `column:ended_by_huawei_api`，避免 acronym 分词差异。[ASSUMED]
- 不为五字段加 index：Phase 23/25 的主要访问路径仍按 task ID，需求未声明按这些字段过滤；额外 index 会增加写成本。[ASSUMED]
- 不改 `cmd/server/app.go` AutoMigrate 列表，因为 model 已包含；但应新增 regression test 证明它仍在列表或直接调用 `MinimalApp.migrateDatabase()` 后检查列。[ASSUMED]
- schema 测试使用 SQLite `:memory:`：`AutoMigrate` → `Migrator().HasColumn` 五次 → create/read/update round-trip → 断言默认值和非默认值。项目不需要外部 DB CLI。[VERIFIED: codebase pattern]

### AUDIT-05 — 三个 sentinel 与生成链

1. 在 `internal/errors/errors.go` 添加业务组：[ASSUMED]
   - `ErrRecordingSmartExtend = errors.New("recording smart extend failed")`
   - `ErrRecordingSmartEarlyEnd = errors.New("recording smart early end failed")`
   - `ErrRecordingHuaWeiStateFetchFailed = errors.New("Huawei conference state fetch failed")`
2. 在 `mapping.go`：三者加入 `knownSentinels`；建议三个均加入内部处理失败的 500 case，或至少依赖 default 500。显式 case 的优势是分类意图更清楚。[ASSUMED]
3. 更新 `mapping_test.go` 的 `TestMapToHTTPStatus_Sentinels` 和 `TestFirstKnownSentinelName`。[VERIFIED: codebase pattern]
4. 运行 `go generate ./internal/errors/...`，由 generator 更新 `docs/errors.md`；不要手工维护 call-site count。[VERIFIED: codebase read]
5. 验证 `git diff --exit-code -- docs/errors.md` 在第二次 generate 后为空，并跑 `go test ./internal/errors/... ./cmd/error-doc-gen/...`。[ASSUMED]

### CFG-01 / CFG-02 — SmartEndConfig

建议在新文件 `internal/config/smart_end.go` 定义结构，根 `Config` 加 `SmartEnd SmartEndConfig`，字段使用当前三 tag 约定：[ASSUMED]

| YAML key | Go field | Type | Default | 建议约束 |
|----------|----------|------|---------|----------|
| enabled | Enabled | bool | true | — |
| silence_db | SilenceDB | int | -30 | `[-100, 0)` |
| silence_duration_s | SilenceDurationS | int | 30 | `>0` |
| file_stall_s | FileStallS | int | 120 | `>0` |
| file_min_growth_bps | FileMinGrowthBPS | int64 | 1024 | `>=0` |
| huawei_enabled | HuaweiEnabled | bool | true | — |
| huawei_poll_interval_s | HuaweiPollIntervalS | int | 30 | `>0` |
| huawei_persist_s | HuaweiPersistS | int | 30 | `>0` |
| huawei_failure_threshold | HuaweiFailureThreshold | int | 3 | `>0` |
| check_interval_s | CheckIntervalS | int | 5 | `>0` |
| extend_step_min | ExtendStepMin | int | 30 | `>0` |
| max_extend_count | MaxExtendCount | int | 4 | `>=0`（0 是否禁用自动延时需 Phase 25 明确） |
| stat_failure_threshold | StatFailureThreshold | int | 3 | `>0` |
| degrade_on_silence_loss | DegradeOnSilenceLoss | bool | true | — |

**布尔默认值陷阱：** 当前 `setDefaults` 无法区分 YAML 中显式 `false` 与 bool 零值。`enabled` 和 `huawei_enabled` 要支持未来 CFG-03/04 显式关闭，因此不能简单写 `if !cfg.SmartEnd.Enabled { cfg...=true }`，否则配置 `false` 会被覆盖。[VERIFIED: config loading behavior + ASSUMED solution]

推荐用 Viper 在 unmarshal 前 `SetDefault("smart_end.enabled", true)`、`SetDefault("smart_end.huawei_enabled", true)`、`SetDefault("smart_end.degrade_on_silence_loss", true)`，其余数值也可统一在 Viper defaults 设置；或者把三个 bool 改成 `*bool` 并在 defaults 后解引用。前者更符合 Viper 能力且保持消费端 bool 简洁，但需要调整 `Load()` 的两阶段 Viper 重建，确保 defaults 在最终 `v.Unmarshal` 前设置。[ASSUMED]

**模板同步：** 根 `config.yaml` 与 `bin/config.yaml` 均加入 PRD §6 原值。若 `createDefaultConfigFile()` 内嵌模板另有一份，也必须搜索并同步；planner 应加一个 exact-key-set 测试防止 13/15 项漂移。[ASSUMED]

**校验：** Phase 23 应验证明显非法值并 fail fast，不做跨信号业务策略。建议 `SmartEndConfig.Validate()` 只做上表范围；不要强制 `huawei_persist_s >= huawei_poll_interval_s`，因为更短 persistence 仍可在下一次 sample 评估，业务可接受。[ASSUMED]

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library (`encoding/json`, `errors`, `time`) | Go 1.25.0 | mailbox 解码、sentinel wrapping、持续时间契约 | 项目 toolchain 已锁定 `go 1.25.0`。`go.mod:0-3` [VERIFIED: codebase read] |
| GORM | v1.30.0 | model 映射与 AutoMigrate | 项目既有 ORM；任务模型已在 startup migration。`go.mod:20-22` [VERIFIED: codebase read] |
| GORM SQLite driver | v1.6.0 | 生产/测试 schema 验证 | 项目配置与测试均采用 SQLite。`go.mod:20-22` [VERIFIED: codebase read] |
| Viper | v1.19.0 | `smart_end` 嵌套 YAML 解码与 defaults | 现有 `config.Load` 已使用。`go.mod:13`、`internal/config/config.go:327-383` [VERIFIED: codebase read] |
| testify | v1.11.1 | 单元/集成断言 | 当前测试标准依赖。`go.mod:14` [VERIFIED: codebase read] |

**Installation:** 无新增 package；本阶段禁止无必要新增依赖。[VERIFIED: phase PRD]

## Architecture Patterns

### System Architecture Diagram

```text
TE40 /action.cgi WEB_GetMailboxDataAPI
        |
        v
Huawei HTTPClient.Post
        |
        v
APIResponse.Data (JSON string)
        |
        v
parseMailboxState --parse error--> ErrRecordingHuaWeiStateFetchFailed
        |
        v
GetConferenceState: one immutable sample + field-presence/fallback metadata
        |
        +----> Phase 24 watcher (OUT OF SCOPE): per-recording emptySince >= 30s

config.yaml + bin/config.yaml --> Viper Load/defaults/validate --> Config.SmartEnd

VideoRecordingTask model --> startup AutoMigrate --> SQLite video_recording_tasks

errors.go --> mapping.go recognition/status --> go generate --> docs/errors.md --> CI diff gate
```

### Recommended Project Structure

```text
internal/
├── huawei/
│   ├── client.go                 # mailbox fields + stateless GetConferenceState
│   └── client_test.go            # JSON/presence/fallback/error fixtures
├── models/
│   ├── video_recording_task.go   # five persisted fields
│   └── video_recording_task_test.go # schema/default/read-write test
├── config/
│   ├── config.go                 # root Config + Load/default hook
│   ├── smart_end.go              # 14-field struct + validation/default helpers
│   └── smart_end_test.go         # defaults, explicit false, YAML, invalid bounds
└── errors/
    ├── errors.go                 # three sentinels
    ├── mapping.go                # status + recognition
    └── mapping_test.go           # mapping/name coverage
config.yaml                       # operator template
bin/config.yaml                   # packaged runtime template
docs/errors.md                    # generated only
```

### Pattern 1: Stateless integration client

**What:** external client performs one authenticated request and parses one response; caller owns polling state.
**When to use:** shared client can serve multiple recording tasks concurrently.
**Why:** avoids cross-task `emptySince` data races and keeps Phase 23 from absorbing Phase 24.
**Source:** existing context-aware API style at `internal/huawei/client.go:702-753`; state ownership boundary is a design recommendation. [VERIFIED: codebase read + ASSUMED]

### Pattern 2: Source-of-truth generation

**What:** define sentinels in Go, map/recognize them, regenerate docs.
**When to use:** every new `ErrXxx`.
**Source:** `internal/errors/errors.go:0-3`, `.github/workflows/ci.yml:53-60`. [VERIFIED: codebase read]

### Anti-Patterns to Avoid

- **Zero-value means field missing:** JSON absent `joinSum` and real `joinSum:0` both become `0`; track presence explicitly.
- **Parse failure means empty meeting:** current `GetMailboxData` behavior risks false early end; return sentinel error instead.
- **Persistence inside shared Huawei client:** introduces global mutable state and wrong semantics across simultaneous tasks.
- **Implementing watcher in Phase 23:** pollers/timers/failure thresholds belong to Phase 24.
- **Only defining errors.New:** without `knownSentinels`, logging/HandleError recognition remains incomplete.
- **Hand-editing errors.md:** generator call-site counts and sorting are canonical.
- **Setting true defaults after unmarshal with `if !bool`:** overrides operator's explicit `false`, breaking future rollback switches.
- **Updating only root config.yaml:** packaged `bin/config.yaml` would drift.
- **Adding custom migration:** task model is already in startup AutoMigrate; dormant registry is forbidden.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Schema migration | raw `ALTER TABLE` registry | existing GORM `AutoMigrate` | model already registered and SQLite compatibility handled centrally |
| YAML parser | custom map decoding | existing Viper + mapstructure tags | nested config path already standardized |
| Error documentation | manual sentinel table/call counts | existing `go generate ./internal/errors/...` | CI checks deterministic generated output |
| Persistence timing in Phase 23 | timer/goroutine in client | Phase 24 per-watcher timestamp + injectable clock | state belongs to recording instance, not transport client |
| HTTP mock framework | new package | package-private parser fixtures or `httptest` + existing client seams | zero new dependencies required |

## Files to Create/Modify (with paths and rationale)

### Create

| Path | Rationale |
|------|-----------|
| `internal/config/smart_end.go` | 14-field `SmartEndConfig`, defaults registration/helper, optional `Validate()` |
| `internal/config/smart_end_test.go` | defaults、explicit false、14-key YAML、invalid range regression |
| `internal/models/video_recording_task_test.go` | AutoMigrate schema/default/read-write coverage；也可放现有 service test，但 model test 更聚焦 |

### Modify

| Path | Rationale |
|------|-----------|
| `internal/huawei/client.go` | mailbox 字段 presence、严格解析、`ConferenceState`、`GetConferenceState` |
| `internal/huawei/client_test.go` | new/old/malformed/transport fixtures |
| `internal/models/video_recording_task.go` | 五个智能录制持久化字段 |
| `internal/config/config.go` | `Config.SmartEnd`、Viper defaults 注册、Load 后校验 |
| `config.yaml` | `smart_end:` 14 项 operator 配置 |
| `bin/config.yaml` | 打包运行时模板保持同步 |
| `internal/errors/errors.go` | 三个 sentinel 定义 |
| `internal/errors/mapping.go` | 三个 sentinel 的 recognition 与 HTTP 500 分类 |
| `internal/errors/mapping_test.go` | 映射和稳定名称覆盖 |
| `docs/errors.md` | 由 `go generate` 自动更新，不手工维护 |

### Explicitly do not modify in Phase 23

- `internal/recorder/activity_watcher.go`、`internal/recorder/coordinator.go`：Phase 24。
- `internal/scheduler/video_scheduler.go`、`internal/services/video_recording_task_service.go`：Phase 25。
- `.github/workflows/ci.yml`：现有 sync-check 已足够，本阶段无需改门禁。
- dormant custom migration registry：禁止。

## Patterns & Conventions (closest analogs in codebase)

| New work | Closest analog | Reuse |
|----------|----------------|-------|
| Huawei state API | `GetMailboxData` / `IsInConference`, `internal/huawei/client.go:702-753` | ctx、session cookie、HTTP action、double-unmarshal |
| Huawei tests | `internal/huawei/client_test.go:21-55` | package-local construction、zap nop、context lifecycle；新增 parser fixtures |
| Model fields | conversion fields, `internal/models/video_recording_task.go:28-38` | explicit GORM type/default + snake_case JSON |
| Migration | `cmd/server/app.go:309-355` | startup AutoMigrate, no registry |
| SQLite model tests | `internal/services/video_recording_task_service_test.go:18-29` | `:memory:` DB + AutoMigrate |
| Config struct | `HuaweiConfig`, `internal/config/config.go:187-197` | triple tags, typed fields |
| Config defaults | `setDefaults`, `internal/config/config.go:404-644` | centralized defaults; special handling needed for true-valued bool defaults |
| Config validation tests | credential config table tests, `internal/config/config_test.go:153-294` | table-driven valid/invalid cases |
| Sentinel definition | `ErrTranscriptionUnavailable`, `internal/errors/errors.go:92-97` | grouped comment + errors.New |
| Sentinel mapping | `internal/errors/mapping.go:84-95,154-202` | HTTP mapping + knownSentinels |
| Generated docs | `internal/errors/errors.go:2`, `.github/workflows/ci.yml:53-60` | go generate then clean-diff gate |

## Common Pitfalls

### Pitfall 1: 老设备 fallback 无法可靠触发
**What goes wrong:** 新字段不存在时 Go 零值恰好满足结束判据，代码误以为新字段存在。
**Why it happens:** `encoding/json` 对缺失字段不报错。
**How to avoid:** 指针字段或 presence-aware unmarshal；只有 `confState`、`joinSum` 均存在才采用新判据。
**Warning signs:** old-device fixture `{ "isInConf": 1 }` 被判定结束。

### Pitfall 2: malformed response false positive
**What goes wrong:** 当前双层 unmarshal 失败后返回空 state,nil，未来被当成会议结束。
**Why it happens:** `GetMailboxData` 静默吞 parse errors。
**How to avoid:** 空 Data、缺 state、state 非对象、类型错误均返回 wrapped fetch sentinel。
**Warning signs:** malformed fixture 的 `err == nil`。

### Pitfall 3: bool default 覆盖显式 false
**What goes wrong:** `enabled:false` 加载后被 defaults 改回 true。
**Why it happens:** bool 零值与配置 false 无法区分。
**How to avoid:** Viper `SetDefault` 在 unmarshal 前设置，或 pointer bool。
**Warning signs:** explicit-false test 结果为 true。

### Pitfall 4: acronym 列名漂移
**What goes wrong:** GORM 自动命名的 `EndedByHuaWeAPI` 与预期 SQL 列名不同。
**Why it happens:** acronym splitting 规则不直观。
**How to avoid:** 用 `Migrator().ColumnTypes` 验证；需要稳定名时显式 `column:ended_by_huawei_api`。
**Warning signs:** HasColumn 用字符串预期失败或数据库出现意外列名。

### Pitfall 5: sentinel 文档“看起来同步”但 recognition 漏掉
**What goes wrong:** generator 可从 errors.go 生成条目，但 runtime `IsKnownError` 仍 false。
**Why it happens:** source definition、HTTP switch、knownSentinels 是三个同步面。
**How to avoid:** 每个新 sentinel 同时加入 `knownSentinels` 和测试；显式映射 500。
**Warning signs:** `FirstKnownSentinelName(newErr)` 返回 false。

## Risks & Open Questions

1. **`GetConferenceState` 是否应自己宣称“持续 ≥30s”？**
   - 已知：success criteria 文案把持续识别写在该方法名下，但 Phase 24 明确拥有 poller/watcher。`ROADMAP.md:68-72`、PRD `:104-116` [VERIFIED: codebase read]
   - 风险：若 planner 机械把 timestamp 放进 shared client，会造成并发污染和阶段越界。
   - 建议：Phase 23 定义一次采样 + fallback metadata；测试一个 package-private、无状态判据函数。Phase 24 才完成时间持续 requirement 的行为验收。若 phase gate 必须在 23 独立宣称 DETECT-01 完成，应在 Phase 23 增加纯 `ConferenceStateTracker`（无 goroutine、由 caller 注入 `now`）而不是把状态塞进 client。[ASSUMED]

2. **Huawei mailbox 真实 payload 的字段层级/类型仍需设备 fixture 确认。**
   - 已知：项目 PRD 转述字段在 `state` 中，现有代码也解 `state`；但本次未获得可访问的 Huawei 官方文档 URL。PRD `:84-115` [CITED: local PRD transcription]
   - 风险：固件可能返回数字字符串、null 或字段位于不同层级。
   - 建议：实施前从 TE40 抓一份脱敏真实 response，加入 fixture；没有 fixture 时将该点保留 MEDIUM confidence。

3. **`IsInConf==0` 在字段缺失时是否足以表示“结束”而非“尚未入会”。**
   - 已知：这是锁定需求。`REQUIREMENTS.md:16` [VERIFIED: codebase read]
   - 风险：任务刚启动/临时断会时可 false positive。
   - 建议：同样经过 Phase 24 的 `huawei_persist_s`，不要在单次样本直接 complete。

4. **三个 sentinel 的 HTTP 状态。**
   - 已知：它们是内部录制行为/外部状态 fetch 故障，并无直接 handler API 契约。
   - 建议：统一 500；Huawei 连续失败后的 watcher degradation 是业务降级，不应把单次 fetch 错误映射为 HTTP 503，除非未来 endpoint 直接暴露此调用。[ASSUMED]

5. **`max_extend_count=0` 的语义。**
   - 已知：默认 4；PRD 未定义显式 0。
   - 建议：Phase 23 validation 可允许 0 表示不自动延时，也可要求 `>0`；planner 在编码前锁定。为避免产生未定义行为，本研究倾向 `>0`，总开关已有 `enabled`。[ASSUMED]

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | build/test/generate | ✓ | go1.25.0 windows/amd64 | — |
| SQLite via Go driver | migration tests | ✓（module dependency） | gorm sqlite v1.6.0 | in-memory SQLite |
| Huawei TE40 | real payload validation | 未验证 | — | JSON fixture + later hardware smoke |
| Git | CI-style clean diff | ✓ | repo clean at research start | — |

**Missing dependencies with no fallback:** none for implementation/unit tests.

**Missing dependencies with fallback:** live TE40；使用脱敏 fixture，硬件 smoke 留为人工验证。

## Validation Architecture (Nyquist — test coverage requirements)

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` + testify v1.11.1 |
| Config file | none（standard `go test`） |
| Quick run command | `go test ./internal/huawei ./internal/models ./internal/config ./internal/errors ./cmd/error-doc-gen` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DETECT-01 | new payload parses confState/joinSum; empty sample classified; malformed response errors | unit | `go test ./internal/huawei -run 'Test.*ConferenceState' -count=1` | ❌ Wave 0 additions to existing test |
| DETECT-04 | absent fields selects `IsInConf` fallback; present zero is not “missing” | unit | `go test ./internal/huawei -run 'Test.*ConferenceState.*Fallback' -count=1` | ❌ Wave 0 |
| AUDIT-01 | five columns migrate, defaults persist, read/write round-trip | integration (SQLite memory) | `go test ./internal/models -run 'TestVideoRecordingTaskSmartEndFields' -count=1` | ❌ Wave 0 |
| AUDIT-05 | three errors map to 500, are known by stable name, generated docs clean | unit + generated artifact | `go test ./internal/errors ./cmd/error-doc-gen && go generate ./internal/errors/... && git diff --exit-code -- docs/errors.md` | partial; tests exist but cases missing |
| CFG-01 | all defaults load; explicit false preserved; invalid thresholds rejected | unit | `go test ./internal/config -run 'TestSmartEnd' -count=1` | ❌ Wave 0 |
| CFG-02 | YAML contains exactly 14 keys and loads values | unit/config artifact | `go test ./internal/config -run 'TestSmartEnd.*YAML' -count=1` | ❌ Wave 0 |

### Required Huawei fixture matrix

| Fixture | Expected |
|---------|----------|
| `confState:"rollcall", joinSum:10, isInConf:1` | new fields available; active |
| `confState:"", joinSum:0, isInConf:0` | new fields available; empty candidate |
| no `confState/joinSum`, `isInConf:0` | fallback available; empty candidate |
| no new fields, `isInConf:1` | fallback active |
| `confState:"rollcall", joinSum:0` | active (conservative) |
| missing only one of the two new fields | fallback; do not partially apply new criterion |
| empty Data / malformed nested JSON / wrong field type | wrapped `ErrRecordingHuaWeiStateFetchFailed` |

### Sampling Rate

- **Per task commit:** relevant package quick run.
- **Per wave merge:** `go test ./internal/huawei ./internal/models ./internal/config ./internal/errors ./cmd/error-doc-gen`.
- **Phase gate:** `go generate ./internal/errors/...` clean diff, then `go test -race ./...`, `go vet ./...`; CI build/lint remains authoritative.

### Wave 0 Gaps

- [ ] Extend `internal/huawei/client_test.go` with mailbox fixtures and error tests.
- [ ] Create `internal/models/video_recording_task_test.go` for schema/read-write.
- [ ] Create `internal/config/smart_end_test.go` for defaults, explicit false, exact 14 keys, validation.
- [ ] Extend `internal/errors/mapping_test.go` for three sentinel names/mappings.
- [ ] Obtain or record a sanitized real TE40 `WEB_GetMailboxDataAPI` fixture if hardware is available.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no new auth | reuse existing Huawei SessionID handling |
| V3 Session Management | yes, existing boundary | `GetMailboxData` requires active session and context cancellation |
| V4 Access Control | no new endpoint | no handler/UI added |
| V5 Input Validation | yes | strict nested JSON parse; config range validation; fail closed on malformed H data |
| V6 Cryptography | no new crypto | retain existing TLS/client policy; never change `InsecureSkipVerify` defaults |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed/forged TE40 response causes false early end | Tampering | strict parse, presence detection, persistence in Phase 24, wrapped error on invalid data |
| Session/response content leaks to logs | Information Disclosure | existing Huawei response sanitizer; log only state fields, never cookie/session ID |
| Config values cause tight-loop polling | Denial of Service | positive minimum validation; no zero/negative interval accepted |
| Overly long reason strings grow DB | Denial of Service | reasons are internal constants/snapshots; optionally cap length before Phase 25 writes |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `GetConferenceState` should return a stateless sample and field-presence metadata | Technical Approach | planner may need a separate tracker to satisfy literal phase criterion |
| A2 | pointer/presence-aware mailbox fields are acceptable API design | Technical Approach | may require custom unmarshal if callers depend on value fields |
| A3 | five proposed GORM tags/defaults and explicit Huawei column name | AUDIT-01 | schema naming/default mismatch |
| A4 | three sentinel HTTP classifications should be 500 | AUDIT-05 | future API may prefer 503 for Huawei fetch |
| A5 | Viper `SetDefault` is preferred for true-valued bool defaults | CFG | two-stage Viper rebuild must apply defaults at correct point |
| A6 | proposed config validation ranges | CFG | operator may need broader/zero semantics |
| A7 | no new dependency is needed | Standard Stack | remains true unless test seam proves insufficient |

## Sources

### Primary (HIGH confidence)

- `D:/CODE/ClaudeCode/record_V2/internal/huawei/client.go` — mailbox struct, API request, nested JSON, fallback field.
- `D:/CODE/ClaudeCode/record_V2/internal/models/video_recording_task.go` — task model/tag conventions.
- `D:/CODE/ClaudeCode/record_V2/cmd/server/app.go` — startup AutoMigrate list.
- `D:/CODE/ClaudeCode/record_V2/internal/config/config.go` — Viper load/default patterns.
- `D:/CODE/ClaudeCode/record_V2/internal/errors/errors.go` and `mapping.go` — sentinel source/mapping/recognition.
- `D:/CODE/ClaudeCode/record_V2/cmd/error-doc-gen/main.go` — generated docs behavior.
- `D:/CODE/ClaudeCode/record_V2/.github/workflows/ci.yml` — sync-check and test gate.
- `D:/CODE/ClaudeCode/record_V2/.planning/REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md` — locked scope and requirement mapping.

### Secondary (MEDIUM confidence)

- `D:/CODE/ClaudeCode/record_V2/docs/plans/2026-08-05-smart-meeting-recording-end-design.md` — local PRD transcription of TE40 official API fields/examples; official external URL was not available in this session.

### Tertiary (LOW confidence)

- None; unverified design recommendations are individually tagged `[ASSUMED]`.

## Metadata

**Confidence breakdown:**
- Codebase findings: HIGH — directly read current code.
- GORM/config/error architecture: HIGH — existing patterns are explicit and tested.
- Huawei field semantics: MEDIUM — locked PRD cites official device research, but no live/official external payload was verified here.
- Persistence boundary recommendation: HIGH — follows concurrency ownership and phase split, but API shape remains planner discretion.

**Research date:** 2026-08-06
**Valid until:** 2026-09-05（内部代码稳定；若 TE40 firmware/fixture 更新应立即复核）

## Research Complete

Phase 23 的规划信息已齐备。建议按 Huawei contract → GORM schema → sentinel generation → config/defaults 四个可独立验证的 plan/task group 执行；严格不接入 watcher、scheduler 或 service 行为。
