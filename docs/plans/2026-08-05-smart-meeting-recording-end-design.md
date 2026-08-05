# 智能会议录制结束方案（Smart Recording End）

> 适用项目：Record V2
> 设计者：Claude Code（基于联网资料 + 现有代码探查）
> 状态：待用户审阅
> 最后更新：2026-08-05

---

## 0. 摘要（TL;DR）

**问题**：实际会议时长与预定 `EndTime` 经常不符——超时未结束被截断，提前结束又浪费磁盘/带宽。

**方案**：
- 在 `recorder/coordinator.go` 给 ffmpeg 增加 `silencedetect` 音频过滤器与 `-progress pipe:1` 进度输出。
- 启动一个 **activity watcher goroutine**，并行采集两种信号：
  1. **silencedetect**：静音持续 ≥ 30s
  2. **输出文件大小停滞**：MKV/HLS 在 120s 内累计增长 < 1 KB/s（可配）
- 两个信号**同时满足** → 视为会议已结束 → 通过 `taskEndedCh` 通知 scheduler 立刻 `completeTask`。
- scheduler 的 `monitorTask` 改为**双信号驱动**：
  - 默认按 `EndTime` timer 走；
  - 若 timer 到点前收到"已结束"信号 → 立即 `completeTask`；
  - 若 timer 到点但收到"仍在进行"信号（任一阈值未触发）→ `EndTime += 30min`、扩展次数 ≤ 4（总上限 2 小时）。
- 全部阈值/开关放 `app config` + GORM 字段（`ExtensionCount`、`LastExtensionReason`、`EndedEarly`），可热调可审计。

**效果**：到点未结束自动延 30min（最多 4 次共 2h），提前结束自动收尾并提交转码，无需人工干预。

---

## 1. 现状速查

来自 Explore agent 对 `internal/` 全量扫描：

| 维度 | 现状 |
|------|------|
| 任务模型 | `VideoRecordingTask` 含 `StartTime / EndTime / PreJoinMinutes / RecordDelayMinutes`，状态 `pending → connecting → recording → converting → completed / failed / canceled`（`internal/models/video_recording_task.go:11-56`） |
| 调度 | `internal/scheduler/video_scheduler.go` `monitorTask` 用 `time.NewTimer(remaining)` 阻塞等 EndTime，到点 `completeTask`（行 542-606） |
| 录制 | `internal/recorder/coordinator.go` `monitorProcessWithKey` 监听 ffmpeg 子进程退出，非预期退出走 `attemptReconnect`（仅 stream 类型，最多 3 次，行 220-322） |
| 会议信号 | 华为侧 `info.Status == "ended"` / `IsInConf==1 && Callstate==2` 已存在但**未消费**（`huawei_conference_connector.go:424` 仅用于 `ValidateConference` 前置校验） |
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
| **A. ffmpeg silencedetect 过滤器** | 直接反映"音频消失"；标准 ffmpeg 内置 | 仅音频；纯静音演讲会被误判为结束 | ✅ 主信号之一 |
| **B. 输出文件大小停滞** | 实现简单；兼容所有 input type（usb/rtsp/rtmp/stream） | mux 缓冲可能造成"假停滞"；需 120s 长窗口 | ✅ 主信号之二 |
| **C. ffmpeg -progress pipe:1** | 标准、低开销、可解析 | tee mux 下 total_size 同样有缓冲 | 仅作辅助日志，不作判定 |
| **D. 华为 IsInConference()** | 业务语义权威 | 老设备硬编码返回 idle；接口分钟级；轮询有延迟 | ❌ 本期不接（后续 phase） |
| **E. RTSP TCP 旁路字节** | 最灵敏 | 需额外 ffprobe/旁路进程，对 usb input 不适用 | ❌ 不通用 |
| **F. WebRTC getStats** | 真实反映下行流量 | 仅 WebRTC 场景适用 | ❌ 与本项目无关 |

**决策**：A + B 双信号"**AND**"判定（必须两者都满足才判定结束）。

### 3.2 为什么不是华为 API

虽然 `info.Status == "ended"` 是最权威信号，但：
- `GetTerminalStatus` 在老设备硬编码返回 `"idle"`（`huawei/client.go:886-893`）
- 接口本身是分钟级轮询，有 30-60s 延迟
- 若会议中途"挂断-重呼"，会被误判为结束

→ 留作 **Phase 2**：在已有 `huawei_conference_connector.go` 上加 15s 轮询，作为"加速信号"叠加到本方案的判定上。

---

## 4. 架构

```
┌────────────────────────────────────────────────────────────┐
│ Scheduler (internal/scheduler/video_scheduler.go)          │
│   monitorTask(task)                                         │
│     ├── ticker C (5s loop)                                  │
│     ├── signals:                                            │
│     │   • EndTime timer (existing)                          │
│     │   • taskEndedCh  (NEW — from watcher)                 │
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
│           ├── scanPipe(stderr) → silence_start / end events│
│           ├── ticker(5s) → os.Stat MKV/HLS size            │
│           └── emit on taskEndedCh (decision in §5.1)       │
└────────────────────────────────────────────────────────────┘
```

### 4.1 新增文件

| 路径 | 职责 |
|------|------|
| `internal/recorder/activity_watcher.go` | `ActivityWatcher` struct：合并 silencedetect 解析 + 文件大小采样，发出 `ActivityEvent` |
| `internal/recorder/silence_parser.go` | ffmpeg stderr `silence_start / silence_end` 行解析器（独立便于单测） |
| `internal/config/smart_end.go` | `SmartEndConfig` 结构体 + 默认值，从 `config.yaml` 加载 |

### 4.2 修改文件

| 路径 | 改动 |
|------|------|
| `internal/recorder/coordinator.go` | `buildRecordingCommand` 增加 `-af silencedetect` 与 `-progress pipe:3`；`StartRecordingWithConfig` 返回 `*ActivityWatcher` 与结束 channel |
| `internal/scheduler/video_scheduler.go` | `monitorTask` 改为 select on EndTime + taskEndedCh + updateCh；加 `maybeExtendEndTime(task)` 与 `completeTaskEarly(task, reason)` |
| `internal/services/video_recording_task_service.go` | 加 `UpdateTaskExtension(task, deltaMin, reason)` / `MarkTaskEndedEarly(task, reason)`；写入 audit log |
| `internal/models/video_recording_task.go` | GORM 加字段：`ExtensionCount int`、`LastExtensionReason string`、`EndedEarly bool`、`EndedEarlyReason string`（AutoMigrate 列表同步） |
| `cmd/server/config.yaml` | 新增 `smart_end:` 段 |
| `internal/errors/`（如新增错误码） | `ErrRecordingSmartExtend / ErrRecordingSmartEarlyEnd`（同步更新 `docs/errors.md`） |
| `internal/handlers/video_recording_task_handler.go` | 可选：暴露 `GET /api/v1/tasks/:id/extension-history` 给前端展示 |

---

## 5. 状态机与判定逻辑

### 5.1 ActivityWatcher 判定（`internal/recorder/activity_watcher.go`）

```go
type ActivityEventType int
const (
    EventSilenceStart ActivityEventType = iota
    EventSilenceEnd
    EventFileStall
    EventMeetingEnded  // ← silence AND file stall both hold long enough
)

type ActivityEvent struct {
    Type      ActivityEventType
    Timestamp time.Time
    Snapshot  ActivitySnapshot  // silence duration, file size delta
}

type ActivitySnapshot struct {
    SilenceSince     time.Time   // zero if not silent
    LastFileGrowthAt time.Time   // zero if file never grew
    CurrentFileSize  int64
    FileGrowthBps    float64     // rolling 120s window
}
```

**判定规则**（每个采样 tick 评估）：

```
EmitMeetingEnded iff:
  (Snapshot.SilenceSince 非零 且 now - SilenceSince ≥ silence_duration_s)
  AND
  (now - LastFileGrowthAt ≥ file_stall_s)
```

**采样间隔**：`check_interval_s`（默认 5s），避免 CPU 浪费。

**降级策略**：
- `silencedetect` stderr 解析失败（连续 5 次无有效行） → 切换到 **纯文件停滞** 判定（`silence_duration_s = 0`）
- 文件采样连续 3 次 `os.Stat` 失败 → 视为流死亡，发 `EventMeetingEnded`，reason = `file_stat_failed`

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
        // EndTime 到点：检查活动信号
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
        task.EndedEarlyReason = ev.Snapshot.String()
        svc.MarkTaskEndedEarly(task, reason)
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

### 5.3 完整状态变迁（叠加在现有状态机上）

```
                          (smart_end.enabled=true 时)
   recording ── EndTime 到点 ──► 仍在活动 ──► EndTime += 30min
       │                          │                │
       │                          │           count < 4
       │                          ▼                │
       │                   写入 ExtensionCount+1
       │                          │
       │                          ▼
       │                  4 次后仍活跃 ──► completeTask("max_extend_reached")
       │
       └── ActivityWatcher 命中双信号 ──► completeTask("smart_early_end")
                                          EndedEarly=true
```

---

## 6. 配置项（`config.yaml` 新增）

```yaml
smart_end:
  enabled: true                    # 总开关；false 时退回纯 EndTime 行为
  silence_db: -30                  # silencedetect noise 阈值（dB）
  silence_duration_s: 30           # 持续静音秒数触发判定
  file_stall_s: 120                # 文件大小无增长秒数触发判定
  file_min_growth_bps: 1024        # 增长速率阈值（<1KB/s 算停滞，可配）
  check_interval_s: 5              # watcher 采样间隔
  extend_step_min: 30              # 单次延时分钟数
  max_extend_count: 4              # 单任务最大延时次数（=2h 总上限）
  stat_failure_threshold: 3        # os.Stat 连续失败次数 → 流死亡
  degrade_on_silence_loss: true    # silencedetect 失效时是否降级到纯文件停滞
```

**默认值**与用户在 AskUserQuestion 里的选择一致：30s silence + 120s stall + 30min × 4 次。

---

## 7. 错误处理 & 边界场景

| 场景 | 行为 |
|------|------|
| ffmpeg silencedetect 解析失败 | `degrade_on_silence_loss=true` → 切纯文件停滞；否则忽略 silence 信号，沿用文件停滞 |
| os.Stat 失败 | 累加计数；≥ `stat_failure_threshold` → 视为流死亡，发 `EventMeetingEnded` reason=`file_stat_failed` |
| ffmpeg 进程异常退出（断流） | 走现有 `attemptReconnect`；**不动** watcher 状态（避免被重连误判为"结束"） |
| 用户手动 `StopTask` | 直接 `StopRecording` + `CancelTaskExecution`；watcher 通过 ctx 取消 |
| 用户手动 `UpdateTaskEndTime` 提前 | timer 重置为新 EndTime；`ExtensionCount` 不重置 |
| 同一任务多 input（huawei_auto + usb） | 每个 input 独立 watcher，**任一**判定结束 → 整体结束（保守策略） |
| 静音演讲（PPT 演示+画外音） | 30s 静音阈值 + 120s 文件停滞可容错；如确为静音演示，运维可调 `silence_duration_s=300` |
| 会议中"挂断-重呼" | 本期不识别（华为信号未接入）；用户可通过 `UpdateTaskEndTime` 主动延后 |
| `ExtensionCount` 已满但仍活跃 | 强制 `completeTask("max_extend_reached")`；写 audit `warn`；状态仍为 `completed`（**不**标 failed，避免运维误判） |

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
| 1 | GORM 加字段 + AutoMigrate | `models/video_recording_task.go` | 30min |
| 2 | `SmartEndConfig` + config.yaml | `config/smart_end.go`、`config.yaml` | 20min |
| 3 | `SilenceParser` + 单测 | `recorder/silence_parser.go`、`recorder/silence_parser_test.go` | 1h |
| 4 | `ActivityWatcher` + 单测 | `recorder/activity_watcher.go`、`recorder/activity_watcher_test.go` | 2h |
| 5 | `coordinator.go` 拼 `-af silencedetect` + `-progress pipe:3` + 启 watcher | `recorder/coordinator.go` | 1h |
| 6 | `monitorTask` 改为双信号驱动 | `scheduler/video_scheduler.go` | 2h |
| 7 | service 层 `UpdateTaskExtension` / `MarkTaskEndedEarly` + audit | `services/video_recording_task_service.go` | 1h |
| 8 | errors 加 `ErrRecordingSmartExtend / ErrRecordingSmartEarlyEnd` + 同步 `docs/errors.md` | `internal/errors/*` | 30min |
| 9 | E2E：mock 一段"提前结束"的会议录屏 | `tests/e2e/smart_end_test.go` | 2h |
| 10 | 文档 + CI 更新（`docs/errors.md` sync） | `docs/errors.md`、`.github/workflows/ci.yml` | 30min |

**总计**：约 11.5h，可拆 2 个 phase：phase A（1-5 + 8）跑通单任务 demo，phase B（6-7 + 9-10）跑通端到端。

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
| silencedetect 误杀（静音演讲） | 中 | 录不全 | 默认 30s 已较保守；阈值放 config + 文档建议长会调到 300s |
| 文件停滞误判（mux 缓冲抖动） | 低 | 提前结束 | 120s 长窗口 + 1KB/s 阈值；监控误判率 |
| `ExtensionCount` 用满但会议仍进行 | 中 | 强制截断 | 写 audit `warn`；前端可展示 `ExtensionCount` 让运维知情 |
| watcher goroutine 泄漏 | 低 | 资源泄漏 | ctx 沿用现有 recorder ctx；`StopRecording` 必 close channel |
| 与现有重连逻辑冲突 | 中 | 状态错乱 | watcher 在 `attemptReconnect` 时**不重置**静音/文件计时（仅清 `SilenceSince`，避免断流期间假静音） |

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
| **2.1 华为 API 加速信号** | 在 `huawei_conference_connector.go` 加 15s 轮询 `info.Status / IsInConference`；命中"ended"立即发 `EventMeetingEnded` reason=`huawei_status_ended` |
| **2.2 跨 input 一致性** | 多 input 任务下，若**部分** input 已结束但其他仍活跃，给出"软结束"信号（仅停部分 ffmpeg，不 completeTask） |
| **2.3 前端可视化** | 任务详情页显示 `ExtensionCount / EndedEarlyReason`，让用户直观看到智能延时的发生 |
| **2.4 机器学习预测** | 用历史会议数据训练"提前结束"概率模型，超阈值提前结束（高风险，需更多数据） |

---

## 14. 审阅要点（请重点确认）

1. **检测信号**选 silencedetect + 文件停滞双 AND，是否同意？
2. **延时策略** 30min × 4 次（2h 上限），是否够用？
3. **沉默容忍** 30s silence + 120s file stall，对你日常会议节奏是否合适？
4. **本期不接华为 API**，改为 phase 2，可接受？
5. **强制截断 vs 失败**：`max_extend_reached` 时任务状态用 `completed`（非 `failed`），便于运维不告警；是否同意？

---

## 附：参考资料

- [FFmpeg silencedetect filter 详解（CSDN）](https://blog.csdn.net/lsb2002/article/details/135485520)
- [FFmpeg -progress pipe 解析（CSDN）](https://blog.csdn.net/jm19920911/article/details/122204627)
- [FFmpeg 转码 "Too many packets buffered" 与 max_muxing_queue_size（CSDN）](https://blog.csdn.net/weixin_92288400/article/details/92869284)
- [nielsCodes/go-meeting — Go + Selenium 自动延长会议](https://github.com/nielsCodes/go-meeting)
- [Microsoft Lync SDK — ITMediaControl::Stop](https://learn.microsoft.com/en-us/previous-versions/office/developer/communication-server-2007/bb800439%28v=office.12%29)
- [Zoom Cloud Recording API](https://developers.zoom.us/docs/api/meetings/#tag/Cloud-Recording)
- 项目内：`docs/ci-maintenance.md`、`internal/scheduler/video_scheduler.go`、`internal/recorder/coordinator.go`
