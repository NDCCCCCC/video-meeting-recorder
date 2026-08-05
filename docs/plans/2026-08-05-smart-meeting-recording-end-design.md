# 智能会议录制结束方案（Smart Recording End）

> 适用项目：Record V2
> 设计者：Claude Code（基于联网资料 + 现有代码探查 + TE40 V600R019C00 官方 HTTP API 调研）
> 状态：待用户审阅
> 最后更新：2026-08-05（v2：纳入 TE40 邮箱 API 主信号）

---

## 0. 摘要（TL;DR）

**问题**：实际会议时长与预定 `EndTime` 经常不符——超时未结束被截断，提前结束又浪费磁盘/带宽。

**方案（v2 — 加入 TE40 邮箱 API 主信号）**：
- **主信号 H**：在 `recorder/coordinator.go` 启一个 **huawei state poller goroutine**，每 30s 调一次 `HuaweiClient.GetConferenceState()`（底层调 `WEB_GetMailboxDataAPI`）。判据 `confState=="" && joinSum==0` 持续 `huawei_persist_s`（默认 30s） → 视为会议已结束。
- **兜底信号 A + B**：在 `recorder/coordinator.go` 给 ffmpeg 增加 `silencedetect` 过滤器 + 输出文件大小采样。silencedetect 持续 ≥ 30s **且** 文件大小停滞 ≥ 120s → 也判定为结束（华为 API 不可达时仍可工作）。
- 任一信号触发 → 通过 `taskEndedCh` 通知 scheduler 立刻 `completeTask`。
- scheduler 的 `monitorTask` 改为**多信号驱动**：
  - 默认按 `EndTime` timer 走；
  - 若 timer 到点前收到"已结束"信号 → 立即 `completeTask`；
  - 若 timer 到点但收到"仍在进行"信号（所有判定均未触发）→ `EndTime += 30min`、扩展次数 ≤ 4（总上限 2 小时）。
- 全部阈值/开关放 `app config` + GORM 字段（`ExtensionCount`、`LastExtensionReason`、`EndedEarly`、`EndedByHuaWeAPI`），可热调可审计。

**效果**：到点未结束自动延 30min（最多 4 次共 2h），提前结束由 TE40 邮箱 API + 双重前端信号任一触发收尾并提交转码，无需人工干预。

**关键变更（v1 → v2）**：
- v1 仅靠前端 silencedetect + 文件停滞；v2 **升格华为 `WEB_GetMailboxDataAPI` 为主信号**（已有 `HuaweiClient.GetMailboxData()` 函数，仅需扩展字段）。
- v1 列入 Phase 2 的"华为 API 加速信号"在 v2 提前到 Phase 1（**改造成本极低**：零新增基础设施，无需 WebSocket / SSE / 长轮询，零新增依赖）。

---

## 1. 现状速查

来自 Explore agent 对 `internal/` 全量扫描：

| 维度 | 现状 |
|------|------|
| 任务模型 | `VideoRecordingTask` 含 `StartTime / EndTime / PreJoinMinutes / RecordDelayMinutes`，状态 `pending → connecting → recording → converting → completed / failed / canceled`（`internal/models/video_recording_task.go:11-56`） |
| 调度 | `internal/scheduler/video_scheduler.go` `monitorTask` 用 `time.NewTimer(remaining)` 阻塞等 EndTime，到点 `completeTask`（行 542-606） |
| 录制 | `internal/recorder/coordinator.go` `monitorProcessWithKey` 监听 ffmpeg 子进程退出，非预期退出走 `attemptReconnect`（仅 stream 类型，最多 3 次，行 220-322） |
| 会议信号 | 华为侧 `WEB_GetMailboxDataAPI` **已集成**（`huawei/client.go:702-744` `HuaweiClient.GetMailboxData`），`IsInConference` 用 `IsInConf==1 && Callstate==2` 判定；但**只解析 9 个字段**（sitename/speaker/mic/gk/sip/callstate/calltype/conftype/isInConf），**未消费** `confState / joinSum / confLeftTime / siteList / siteInfo` 这几个判定"会议已结束"最关键的字段。`GetConferenceInfo` 是 stub（`huawei/client.go:855-875`） |
| 文件采样 | 现状只在完成时 `os.Stat` 一次性读 `FileSize`，**没有**周期性采样代码 |
| 调度框架 | `github.com/robfig/cron/v3` 秒级 cron；后台 ticker 已用 `time.NewTicker` 模式（hls sweep、rate limiter、audit 等） |

**根因**：收尾只有 2 个触发源——**日历时间**（EndTime timer）+ **ffmpeg 进程死亡**（断流重连）。没有任何"会议真结束"信号。

---

## 2. 目标与非目标

### 2.1 Goals

1. **到点未结束**：自动延长 30 分钟，单任务最多 4 次（2h 累计），不丢内容。
2. **提前结束**：silence + 文件停滞双信号命中 → 立即 `completeTask`，提交转码。
3. **可观测**：每次"延时"和"提前结束"都写 audit log，含检测快照。
4. **可降级**：silencedetect 文件读不出 → 降级只用文件停滞；都失败 → 不影响原有 EndTime 兜底。
5. **可配置**：阈值/开关放 app config（`config.yaml`），运维可调。

### 2.2 Non-Goals（明确不做）

- 不改 `VideoRecordingTaskStatus` 枚举（沿用 `completed / failed / canceled`）。
- 不做"会议提前结束预测"（不做 ML/ASR）。
- 不重写 `monitorProcessWithKey` 现有的断流重连逻辑。
- 不动前端 UI（仅后端 + config）。

---

## 3. 检测信号选型

### 3.1 候选与决策

| 候选 | 优点 | 缺点 | 选不选 |
|------|------|------|--------|
| **A. ffmpeg silencedetect 过滤器** | 直接反映"音频消失"；标准 ffmpeg 内置 | 仅音频；纯静音演讲会被误判为结束 | ✅ **兜底信号 1** |
| **B. 输出文件大小停滞** | 实现简单；兼容所有 input type（usb/rtsp/rtmp/stream） | mux 缓冲可能造成"假停滞"；需 120s 长窗口 | ✅ **兜底信号 2** |
| **C. ffmpeg -progress pipe:1** | 标准、低开销、可解析 | tee mux 下 total_size 同样有缓冲 | 仅作辅助日志，不作判定 |
| **H. TE40 `WEB_GetMailboxDataAPI`** | 业务语义权威；项目**已有** `HuaweiClient.GetMailboxData()`；零新增基础设施；判据 `confState=="" && joinSum==0` 文档原文示例 | 老设备可能无 confState 字段（fallback 到 `IsInConf==0`）；需 30s 持续防瞬时抖动 | ✅ **主信号** |
| ~~D. 华为 IsInConference()（旧）~~ | 业务语义权威 | 老设备硬编码返回 idle；接口分钟级；轮询有延迟 | ❌ 已被 H 替代（升级版用 confState+joinSum 而非 IsInConf） |
| **E. RTSP TCP 旁路字节** | 最灵敏 | 需额外 ffprobe/旁路进程，对 usb input 不适用 | ❌ 不通用 |
| **F. WebRTC getStats** | 真实反映下行流量 | 仅 WebRTC 场景适用 | ❌ 与本项目无关 |
| **G. MSG_CONF_STATE_CHANGE 推送** | 实时性最高（被动推送） | 文档示例是 `Ext.Ajax.request`，底层是 HTTP 长轮询；项目零 websocket 基础设施 | ❌ 本期不接（后续 phase） |

**决策（v2）**：**H 为主信号，A + B 为兜底**——三者任一判定结束即触发收尾（OR 关系，但 H 单独需持续 `huawei_persist_s` 防瞬时抖动；A + B 是 AND）。

### 3.2 主信号 H 的依据（TE40 V600R019C00 官方 API 调研结论）

**接口**：`POST /action.cgi?ActionID=WEB_GetMailboxDataAPI`
**项目内已有实现**：`huawei/client.go:702-744` `HuaweiClient.GetMailboxData()`

**响应结构（文档原文示例，2026-08-05 调研）**：

```json
// 会议进行中
{"joinSum": 10, "unJoinSum": 5, "confState": "rollcall",
 "confLeftTime": 200, "isSupportT140": 0, "isOpenT140": 0,
 "siteList": [...], "siteInfo": [...], ...}

// 会议未召开（文档原文示例）
{"joinSum": 0, "unJoinSum": 0, "confState": "", "confLeftTime": 0,
 "siteList": [], "siteInfo": [], ...}
```

**关键判据**：`confState=="" && joinSum==0` ⇒ 会议已结束（业务级权威信号）。

**实施路径（最小改动）**：
1. 扩展 `MailboxState.State` struct（`client.go:202-214`），新增字段：
   - `ConfState string \`json:"confState"\``
   - `JoinSum int \`json:"joinSum"\``
   - `UnJoinSum int \`json:"unJoinSum"\``
   - `ConfLeftTime int \`json:"confLeftTime"\``
   - `IsSupportT140 int \`json:"isSupportT140"\``
   - `IsOpenT140 int \`json:"isOpenT140"\``
   - `IsInMinimcuConf int \`json:"isInMinimcuConf"\``
   - `IsUnderMCU int \`json:"isUnderMCU"\``
2. 新增 `HuaweiClient.GetConferenceState() (confState, joinSum, confLeftTime, err)` 函数（包装 `GetMailboxData` 并提取上述字段）。
3. 在 `ActivityWatcher` 旁加 **HuaWeiStatePoller goroutine**（或合并到同一 watcher 内），每 30s 调一次，连续 30s 判定为"已结束"则发 `EventMeetingEnded reason="huawei_state_empty"`。

**为什么不用 MSG_CONF_STATE_CHANGE 推送消息**：
- 文档示例 `Ext.Ajax.request` 暗示底层是 HTTP 长轮询，**不是 WebSocket**。
- 项目零 websocket / SSE / long-polling 基础设施（Grip 全文搜索无相关依赖）。
- `WEB_GetMailboxDataAPI` 30s 轮询已能满足"提前结束"判定时效（业务容忍 30-60s 延迟），无必要引入推送通道。
- `MSG_CONF_STATE_CHANGE` 列入 **Phase 2.1**（待项目具备消息推送接入能力时再做）。

---

## 4. 架构

```
┌────────────────────────────────────────────────────────────┐
│ Scheduler (internal/scheduler/video_scheduler.go)          │
│   monitorTask(task)                                         │
│     ├── ticker C (5s loop)                                  │
│     ├── signals:                                            │
│     │   • EndTime timer (existing)                          │
│     │   • taskEndedCh  (NEW — from ActivityWatcher)         │
│     │   • taskUpdateChans (existing — UpdateTaskEndTime)    │
│     └── decision matrix (see §5)                           │
└───────────────────────────┬────────────────────────────────┘
                            │ (existing) StartRecording → TaskHandle
                            ▼
┌────────────────────────────────────────────────────────────┐
│ Recorder (internal/recorder/coordinator.go)                │
│   StartRecordingWithConfig(task)                            │
│     ├── spawn ffmpeg:                                       │
│     │     -af silencedetect=noise=-30dB:d=30 -f s16le ...   │
│     │     -progress pipe:3                                  │
│     └── start ActivityWatcher goroutine (NEW)              │
│           ├── silencedetect parser (stderr) → EventSilenceStart/End │
│           ├── file size sampler (ticker 5s) → EventFileStall │
│           ├── huawei state poller (ticker 30s) → EventMeetingEnded │
│           │     (调 HuaweiClient.GetConferenceState()       │
│           │      判 confState=="" && joinSum==0 持续 30s)   │
│           └── emit on taskEndedCh (decision in §5.1)       │
└───────────────────────────┬────────────────────────────────┘
                            │ (existing) 调用
                            ▼
┌────────────────────────────────────────────────────────────┐
│ Huawei Client (internal/huawei/client.go)                  │
│   GetConferenceState(ctx) (NEW)                             │
│     └── GetMailboxData() → 解析 confState/joinSum/...       │
│         (MailboxState.State 需扩展字段, 见 §4.1)            │
└────────────────────────────────────────────────────────────┘
```

### 4.1 新增 / 修改文件

| 路径 | 类型 | 职责 / 改动 |
|------|------|-------------|
| `internal/recorder/activity_watcher.go` | 新增 | `ActivityWatcher` struct：合并 silencedetect 解析 + 文件大小采样 + **华为状态轮询**，发出 `ActivityEvent` |
| `internal/recorder/silence_parser.go` | 新增 | ffmpeg stderr `silence_start / silence_end` 行解析器（独立便于单测） |
| `internal/config/smart_end.go` | 新增 | `SmartEndConfig` 结构体 + 默认值，从 `config.yaml` 加载 |
| `internal/huawei/client.go` | 修改 | `MailboxState.State` 加 8 个字段（confState/joinSum/unJoinSum/confLeftTime/isSupportT140/isOpenT140/isInMinimcuConf/isUnderMCU）；新增 `GetConferenceState()` 函数 |
| `internal/recorder/coordinator.go` | 修改 | `buildRecordingCommand` 增加 `-af silencedetect` 与 `-progress pipe:3`；`StartRecordingWithConfig` 注入 `HuaweiClient` 引用给 watcher |
| `internal/scheduler/video_scheduler.go` | 修改 | `monitorTask` 改为 select on EndTime + taskEndedCh + updateCh；加 `maybeExtendEndTime(task)` 与 `completeTaskEarly(task, reason)` |
| `internal/services/video_recording_task_service.go` | 修改 | 加 `UpdateTaskExtension(task, deltaMin, reason)` / `MarkTaskEndedEarly(task, reason, byHuaWeiAPI bool)`；写入 audit log |
| `internal/models/video_recording_task.go` | 修改 | GORM 加字段：`ExtensionCount int`、`LastExtensionReason string`、`EndedEarly bool`、`EndedEarlyReason string`、`EndedByHuaWeAPI bool`（AutoMigrate 列表同步） |
| `cmd/server/config.yaml` | 修改 | 新增 `smart_end:` 段（含 `huawei_persist_s: 30` 等新阈值） |
| `internal/errors/` | 修改 | `ErrRecordingSmartExtend / ErrRecordingSmartEarlyEnd / ErrRecordingHuaWeiStateFetchFailed`（同步更新 `docs/errors.md`） |
| `internal/handlers/video_recording_task_handler.go` | 可选 | 暴露 `GET /api/v1/tasks/:id/extension-history` 给前端展示 |

---

## 5. 状态机与判定逻辑

### 5.1 ActivityWatcher 判定（`internal/recorder/activity_watcher.go`）

```go
type ActivityEventType int
const (
    EventSilenceStart ActivityEventType = iota
    EventSilenceEnd
    EventFileStall
    EventHuaWeiStateEmpty   // ← 华为 API 报告 confState=="" && joinSum==0
    EventMeetingEnded       // ← 上述任一信号持续达到阈值
)

type ActivityEvent struct {
    Type      ActivityEventType
    Timestamp time.Time
    Snapshot  ActivitySnapshot
    Reason    string  // e.g. "both_silence_and_stall" / "huawei_state_empty" / "file_stat_failed"
}

type ActivitySnapshot struct {
    SilenceSince       time.Time   // zero if not silent
    LastFileGrowthAt   time.Time   // zero if file never grew
    CurrentFileSize    int64
    FileGrowthBps      float64     // rolling 120s window
    HuaWeiStateEmptySince time.Time // zero if not empty；confState=="" && joinSum==0 起算
    LastHuaWeiState    string      // 最近的 confState 值（debug 用）
    LastJoinSum        int
}
```

**判定规则**（每个采样 tick 评估，**OR 关系**——任一触发即发 `EventMeetingEnded`）：

```
EmitMeetingEnded iff:
  // 主信号 H（华为 API）
  (Snapshot.HuaWeiStateEmptySince 非零
   AND now - HuaWeiStateEmptySince ≥ huawei_persist_s)   // 默认 30s

  OR

  // 兜底 A + B（双 AND）
  (Snapshot.SilenceSince 非零
   AND now - SilenceSince ≥ silence_duration_s           // 默认 30s
   AND now - LastFileGrowthAt ≥ file_stall_s)            // 默认 120s
```

**采样间隔**：
- silencedetect 解析：事件驱动（stderr 读到行即处理）
- 文件大小：`check_interval_s`（默认 5s）
- 华为状态轮询：`huawei_poll_interval_s`（默认 30s）

**降级策略**：
- `silencedetect` stderr 解析失败（连续 5 次无有效行） → 切换到 **纯文件停滞 + 华为信号** 判定（关闭 silence 分支）
- `HuaweiClient.GetConferenceState()` 失败（连续 3 次）→ 降级关闭主信号 H，只用 A + B
- 文件采样连续 3 次 `os.Stat` 失败 → 视为流死亡，发 `EventMeetingEnded`，reason = `file_stat_failed`
- 全部信号都失效 → 不影响原有 EndTime 兜底

### 5.2 scheduler `monitorTask` 决策矩阵

```go
for {
    select {
    case <-EndTime.C:
        if task.ExtensionCount >= cfg.MaxExtendCount {
            log.Warn("max extend reached, force end")
            completeTask(task, "max_extend_reached")
            return
        }
        // EndTime 到点：检查活动信号（任一活跃即延时）
        if watcher.IsActive() {
            task.EndTime = task.EndTime.Add(extendStep)
            task.ExtensionCount++
            task.LastExtensionReason = watcher.Snapshot().String()
            svc.UpdateTaskExtension(task, extendStep, reason)
            log.Info("auto-extend", "task", task.ID, "new_end", task.EndTime, "count", task.ExtensionCount)
            resetTimer(EndTime, task.EndTime.Sub(now))
        } else {
            completeTask(task, "endtime_no_activity")
            return
        }

    case ev := <-watcher.taskEndedCh:
        // 提前结束信号
        task.EndedEarly = true
        task.EndedEarlyReason = ev.Reason
        task.EndedByHuaWeAPI = (ev.Type == EventHuaWeiStateEmpty)
        svc.MarkTaskEndedEarly(task, ev.Reason, task.EndedByHuaWeAPI)
        log.Info("smart_early_end", "task", task.ID, "reason", ev.Reason, "by_huawei_api", task.EndedByHuaWeAPI)
        completeTask(task, "smart_early_end")
        return

    case <-taskUpdateChans[task.ID]:
        // 外部 UpdateTaskEndTime 触发，重置 timer
        resetTimer(EndTime, task.EndTime.Sub(now))
    }
}
```

**关键不变量**：
- EndTime timer 到点**不直接结束**——必须先问 watcher。
- 提前结束信号**永远优先**于 EndTime timer（select 不保证，但 close channel 后 EndTime.C 不再生效即可）。
- `ExtensionCount` 在 `UpdateTaskEndTime`（手动改 EndTime）时**不重置**——避免人为绕开上限。
- H 信号的"持续 30s"由 watcher 内部状态机控制（`HuaWeiStateEmptySince` 时间戳），不是用 channel buffer 偷懒——避免瞬时 confState 抖动误杀。

### 5.3 完整状态变迁（叠加在现有状态机上）

```
                          (smart_end.enabled=true 时)
   recording ── EndTime 到点 ──► 仍在活动（H/A/B 任一非空）──► EndTime += 30min
       │                              │                            │
       │                              │                       count < 4
       │                              ▼                            │
       │                       写入 ExtensionCount+1
       │                              │
       │                              ▼
       │                      4 次后仍活跃 ──► completeTask("max_extend_reached")
       │
       ├── H: HuaWeiStateEmpty 持续 30s  ──► completeTask("smart_early_end", reason=huawei_state_empty)
       │                                    EndedByHuaWeAPI=true
       │
       └── A+B: silencedetect 30s AND file_stall 120s  ──► completeTask("smart_early_end", reason=both_silence_and_stall)
                                                       EndedByHuaWeAPI=false
```

---

## 6. 配置项（`config.yaml` 新增）

```yaml
smart_end:
  enabled: true                    # 总开关；false 时退回纯 EndTime 行为
  # 兜底 A: silencedetect
  silence_db: -30                  # silencedetect noise 阈值（dB）
  silence_duration_s: 30           # 持续静音秒数触发判定
  # 兜底 B: 文件大小
  file_stall_s: 120                # 文件大小无增长秒数触发判定
  file_min_growth_bps: 1024        # 增长速率阈值（<1KB/s 算停滞，可配）
  # 主信号 H: 华为 WEB_GetMailboxDataAPI
  huawei_enabled: true             # 主信号开关；false 时只用兜底 A+B
  huawei_poll_interval_s: 30       # 华为 API 轮询间隔
  huawei_persist_s: 30             # confState=="" && joinSum==0 持续秒数触发判定
  huawei_failure_threshold: 3      # 连续失败次数后降级关闭 H 信号
  # 通用
  check_interval_s: 5              # A/B 信号 watcher 采样间隔
  extend_step_min: 30              # 单次延时分钟数
  max_extend_count: 4              # 单任务最大延时次数（=2h 总上限）
  stat_failure_threshold: 3        # os.Stat 连续失败次数 → 流死亡
  degrade_on_silence_loss: true    # silencedetect 失效时是否降级到 A' + H（仅文件停滞 + 华为）
```

**默认值**与用户在 AskUserQuestion 里的选择一致：30s silence + 120s stall + 30s 华为信号持续 + 30min × 4 次延时。

---

## 7. 错误处理 & 边界场景

| 场景 | 行为 |
|------|------|
| ffmpeg silencedetect 解析失败 | `degrade_on_silence_loss=true` → 切纯文件停滞 + 华为信号；否则忽略 silence 信号 |
| `HuaweiClient.GetConferenceState()` 失败 | 累加计数；≥ `huawei_failure_threshold` → 降级关闭 H 信号，只用 A + B |
| `HuaweiClient.GetConferenceState()` 返回字段缺失（老设备） | fallback 到 `IsInConf==0` 判据（`huawei/client.go:212` 现有字段） |
| `confState` 字段返回非空但 `joinSum==0`（会议类型异常） | 不判定结束（保守策略：可能会议处于"主席离线但仍占用"状态） |
| os.Stat 失败 | 累加计数；≥ `stat_failure_threshold` → 视为流死亡，发 `EventMeetingEnded` reason=`file_stat_failed` |
| ffmpeg 进程异常退出（断流） | 走现有 `attemptReconnect`；**不动** watcher 状态（避免被重连误判为"结束"） |
| 用户手动 `StopTask` | 直接 `StopRecording` + `CancelTaskExecution`；watcher 通过 ctx 取消 |
| 用户手动 `UpdateTaskEndTime` 提前 | timer 重置为新 EndTime；`ExtensionCount` 不重置 |
| 同一任务多 input（huawei_auto + usb） | 每个 input 独立 watcher，**任一**判定结束 → 整体结束（保守策略） |
| 静音演讲（PPT 演示+画外音） | 30s 静音阈值 + 120s 文件停滞可容错；如确为静音演示，运维可调 `silence_duration_s=300` |
| 会议中"挂断-重呼" | H 信号（`confState`+`joinSum`）能识别：挂断后 `joinSum==0` → 触发收尾；若再呼起新会议，本任务已结束（**不会**重新进入录制） |
| `ExtensionCount` 已满但仍活跃 | 强制 `completeTask("max_extend_reached")`；写 audit `warn`；状态仍为 `completed`（**不**标 failed，避免运维误判） |
| H 信号 false positive（误杀活会议） | `huawei_persist_s=30` 持续判定 + 兜底 A+B 可在 H 误判时继续录制；但 H 单独触发时任务立即结束；建议监控 `EndedByHuaWeAPI=true` 比例 |
| 全部信号失效 | 不影响原有 EndTime 兜底；到点正常 `completeTask("endtime_no_activity")` |

---

## 8. 可观测性

### 8.1 audit log（落 `internal/services/audit`）

每次"延时"和"提前结束"事件：

```json
{
  "event": "smart_recording_extend",
  "task_id": 12345,
  "task_name": "...",
  "extension_count": 2,
  "extend_step_min": 30,
  "new_end_time": "2026-08-05T18:30:00+08:00",
  "snapshot": {
    "silence_since": "2026-08-05T17:55:12+08:00",
    "last_file_growth": "2026-08-05T17:54:30+08:00",
    "file_size_bytes": 234567890,
    "file_growth_bps": 0.0
  },
  "ts": "2026-08-05T18:00:05+08:00"
}
```

### 8.2 日志（`internal/logger`）

- `INFO`：`smart_extend task=12345 count=2 new_end=... reason=...`
- `INFO`：`smart_early_end task=12345 reason=both_signals snapshot=...`
- `WARN`：`max_extend_reached task=12345 force_end=true`
- `ERROR`：`activity_watcher_degraded reason=silence_parser_failed`

### 8.3 监控（可选）

- `record_v2_smart_extend_total{task="..."}` counter
- `record_v2_smart_early_end_total{task="..."}` counter
- `record_v2_watcher_degraded_total` counter

（如项目无 prometheus 集成，本期可不做；日志足够排查）

---

## 9. 实施步骤（建议顺序）

| # | 任务 | 涉及文件 | 估计 |
|---|------|----------|------|
| 1 | GORM 加字段 + AutoMigrate（`ExtensionCount` / `LastExtensionReason` / `EndedEarly` / `EndedEarlyReason` / `EndedByHuaWeAPI`） | `models/video_recording_task.go` | 30min |
| 2 | `SmartEndConfig` + config.yaml（含 `huawei_*` 段） | `config/smart_end.go`、`config.yaml` | 20min |
| 3 | **华为 API 改造**：`MailboxState.State` 加 8 字段 + `GetConferenceState()` 函数 + 单测 | `huawei/client.go`、`huawei/client_test.go` | 1h |
| 4 | `SilenceParser` + 单测 | `recorder/silence_parser.go`、`recorder/silence_parser_test.go` | 1h |
| 5 | `ActivityWatcher` + 单测（含 H/A/B 三信号 OR 判定 + 降级逻辑） | `recorder/activity_watcher.go`、`recorder/activity_watcher_test.go` | 3h |
| 6 | `coordinator.go` 拼 `-af silencedetect` + `-progress pipe:3` + 注入 `HuaweiClient` 引用 + 启 watcher | `recorder/coordinator.go` | 1h |
| 7 | `monitorTask` 改为多信号驱动（含 H 触发的 `EndedByHuaWeAPI=true` 写入） | `scheduler/video_scheduler.go` | 2h |
| 8 | service 层 `UpdateTaskExtension` / `MarkTaskEndedEarly(task, reason, byHuaWeiAPI)` + audit | `services/video_recording_task_service.go` | 1h |
| 9 | errors 加 `ErrRecordingSmartExtend / ErrRecordingSmartEarlyEnd / ErrRecordingHuaWeiStateFetchFailed` + 同步 `docs/errors.md` | `internal/errors/*` | 30min |
| 10 | E2E：mock 一段"提前结束"的会议录屏（含 H 信号触发场景） | `tests/e2e/smart_end_test.go` | 3h |
| 11 | 文档 + CI 更新（`docs/errors.md` sync） | `docs/errors.md`、`.github/workflows/ci.yml` | 30min |

**总计**：约 14h，可拆 3 个 phase：
- **phase A**（#1-3 + #9）：华为 API 改造 + 字段迁移 + 错误码（约 3.5h）
- **phase B**（#2 + #4-6）：watcher + 兜底信号 + 录制层接入（约 5.5h）
- **phase C**（#7-8 + #10-11）：scheduler 改造 + service + E2E + CI（约 5h）

---

## 10. 测试策略

### 10.1 单元测试

- `SilenceParser`：用 fixture 文件模拟 ffmpeg stderr，断言事件序列。
- `ActivityWatcher`：mock 文件 stat（注入 `FileStatFn`），断言判定时机。

### 10.2 集成测试

- `coordinator.go` 起 ffmpeg 录制 60s 真实音频（用 `testsrc` + `sine`），验证 watcher 在静音后正确发出事件。

### 10.3 E2E（重点）

**场景 1：提前结束**
```
1. 创建任务 EndTime = now+1h
2. ffmpeg 推 sine 音频 30s 后静音
3. 期望：120s 后 watcher 发 EventMeetingEnded → completeTask("smart_early_end")
4. 断言：task.EndedEarly == true；RecordingDuration ≈ 150s
```

**场景 2：自动延时**
```
1. EndTime = now+5min；silence_duration_s 调成 600（不触发）
2. ffmpeg 持续推流
3. 5min 后期望：EndTime += 30min；ExtensionCount=1
4. 4 次后第 4 次仍延；第 5 次到点 → completeTask("max_extend_reached")
```

**场景 3：降级**
```
1. silencedetect 解析失败（构造畸形 stderr）
2. 期望：watcher 自动切纯文件停滞
3. 日志含 `activity_watcher_degraded`
```

---

## 11. 风险与缓解

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| **H 信号 false positive**（华为 API 报告"会议结束"但实际还在） | 中 | 提前结束 | `huawei_persist_s=30` 持续判定防瞬时抖动；监控 `EndedByHuaWeAPI=true` 比例告警；可临时 `huawei_enabled=false` 回退 A+B |
| H 信号老设备字段缺失 | 中 | H 信号失效 | fallback 到现有 `IsInConf==0` 判据；连续 `huawei_failure_threshold=3` 次失败自动降级关闭 H |
| `confState` 文档未列全所有枚举值 | 高 | 误判/漏判 | 仅用 `confState==""` 作为"已结束"信号（文档明确给出"未召开"示例），其他值不判定结束；保守策略 |
| silencedetect 误杀（静音演讲） | 中 | 录不全 | 默认 30s 已较保守；阈值放 config + 文档建议长会调到 300s |
| 文件停滞误判（mux 缓冲抖动） | 低 | 提前结束 | 120s 长窗口 + 1KB/s 阈值；监控误判率 |
| `ExtensionCount` 用满但会议仍进行 | 中 | 强制截断 | 写 audit `warn`；前端可展示 `ExtensionCount` 让运维知情 |
| watcher goroutine 泄漏 | 低 | 资源泄漏 | ctx 沿用现有 recorder ctx；`StopRecording` 必 close channel；H 信号轮询 goroutine 同 ctx 取消 |
| 与现有重连逻辑冲突 | 中 | 状态错乱 | watcher 在 `attemptReconnect` 时**不重置**静音/文件计时（仅清 `SilenceSince`，避免断流期间假静音）；H 信号在重连期间不轮询（复用 `GetMailboxData` 的 session 状态） |

---

## 12. 与现有 GSD / commit 约定的协调

- 按 `commit-boundary-separation.md`：本方案的 GORM 字段迁移、watcher 实现、scheduler 改造、配置/文档**分 4 个 commit**，便于回滚。
- 按 `migration-automigrate-convention.md`：新字段走 `AutoMigrate`，不进 `runCustomMigrations`。
- 按 `.planning/gitignored.md`：实施阶段的 plan/tasks 仍放 `.planning/phases/`（用 `git add -f`），但**最终设计文档**放 `docs/plans/`（本文件，正常 commit）。
- CI 约束（`docs/ci-maintenance.md`）：新增 `errors/` 条目**必须**同步 `docs/errors.md`，否则 `Verify errors doc sync` step 会失败。

---

## 13. 后续 Phase 候选（本期不做）

| Phase | 内容 |
|-------|------|
| **2.1 MSG_CONF_STATE_CHANGE 推送接入** | 项目具备消息推送能力后，改用 `MSG_CONF_STATE_CHANGE.Param2==0` 替代 30s 轮询；进一步降低判定延迟到 1-2s |
| **2.2 T.140 字幕信号** | TE40 响应里 `isSupportT140=1 && isOpenT140=1` 时，把 T.140 字幕流作为辅助活动信号（避免纯静音会议被 A+B 误杀） |
| **2.3 跨 input 一致性** | 多 input 任务下，若**部分** input 已结束但其他仍活跃，给出"软结束"信号（仅停部分 ffmpeg，不 completeTask） |
| **2.4 前端可视化** | 任务详情页显示 `ExtensionCount / EndedEarlyReason / EndedByHuaWeAPI`，让用户直观看到智能延时的发生 |
| **2.5 机器学习预测** | 用历史会议数据训练"提前结束"概率模型，超阈值提前结束（高风险，需更多数据） |

---

## 14. 审阅要点（请重点确认）

1. **主信号 H（华为 API）**：用 `WEB_GetMailboxDataAPI.confState=="" && joinSum==0` 持续 30s 判定会议结束，是否同意？这是 v2 的核心变更，零新增基础设施、零新增依赖，仅需扩展 8 个 struct 字段。
2. **兜底 A + B**：silencedetect 30s **AND** 文件停滞 120s，作为 H 失效/不可达时的兜底，是否同意？
3. **判定关系**（OR 还是 AND）：v2 采用 **OR 关系**——H 或 A+B 任一触发即结束。是否需要改为 AND 关系（H **且** A+B 触发才结束）以降低误杀率？
4. **延时策略** 30min × 4 次（2h 上限），是否够用？
5. **沉默容忍** 30s silence + 120s file stall + 30s 华为信号持续，对你日常会议节奏是否合适？
6. **强制截断 vs 失败**：`max_extend_reached` 时任务状态用 `completed`（非 `failed`），便于运维不告警；是否同意？
7. **审计字段** `EndedByHuaWeAPI bool`：是否需要在任务详情页/审计日志里明确区分"由华为信号触发的提前结束" vs "由兜底信号触发"，便于排查误杀来源？
8. **配置开关** `huawei_enabled: true`：是否需要在 TE40 设备下线/维护时能单独关闭 H 信号、回退 A+B？

---

## 附：参考资料

- [FFmpeg silencedetect filter 详解（CSDN）](https://blog.csdn.net/lsb2002/article/details/135485520)
- [FFmpeg -progress pipe 解析（CSDN）](https://blog.csdn.net/jm19920911/article/details/122204627)
- [FFmpeg 转码 "Too many packets buffered" 与 max_muxing_queue_size（CSDN）](https://blog.csdn.net/weixin_92288400/article/details/92869284)
- [nielsCodes/go-meeting — Go + Selenium 自动延长会议](https://github.com/nielsCodes/go-meeting)
- [Microsoft Lync SDK — ITMediaControl::Stop](https://learn.microsoft.com/en-us/previous-versions/office/developer/communication-server-2007/bb800439%28v=office.12%29)
- [Zoom Cloud Recording API](https://developers.zoom.us/docs/api/meetings/#tag/Cloud-Recording)
- **TE40 V600R019C00 HTTP API 编程参考**（2026-08-05 调研）：
  - 会议控制类（chapter `eca722a4`）：`WEB_InitSiteListDataAPI` / `WEB_EndConfAPI` / `WEB_ConfTimeDelayAPI`
  - 消息接口 → 会议开始类（chapter `cdfab5b4`）：`MSG_CONF_STATE_CHANGE` / `MSG_RECORD_STATUS_CHANGE`
  - 消息数据的订阅与取消订阅（chapter `445e9964`）
  - 状态数据的订阅与取消订阅（chapter `20325c60`）
  - 枚举定义 → 会议类型 / 呼叫状态（chapter `91bed6c`）
- 项目内：`docs/ci-maintenance.md`、`internal/scheduler/video_scheduler.go`、`internal/recorder/coordinator.go`、`internal/huawei/client.go:702-744`（`GetMailboxData` 已有实现）
