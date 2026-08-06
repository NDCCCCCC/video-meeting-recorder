---
phase: 24-activitywatcher-silencedetect
plan: 03
type: execute
status: complete
---

# Plan 24-03 Summary — coordinator wiring

## Objective
把 ActivityWatcher 接到 ffmpeg 录制管线:buildRecordingCommand 注入 -af silencedetect 过滤器 (DETECT-02 wiring),RecordingProcess 加 OnReconnect 回调字段 (WATCH-05),StartRecordingWithConfig 构造 per-task ActivityWatcher。

## Tasks Complete (3/3)

### Task 1 — RecordingProcess.OnReconnect + restartRecording 调用
- File: `internal/recorder/coordinator.go`
- `RecordingProcess` struct 新增 `OnReconnect func()` 字段 (WATCH-05)
- `restartRecording` 在 `process.logFile.Close()` 之后 / `cmd.Start` 之前同步调用 `process.OnReconnect()` (RESEARCH.md Pitfall 4 timing)
- nil-safe (Process 默认 nil 时不 panic)

### Task 2 — buildRecordingCommand 注入 -af silencedetect
- File: `internal/recorder/coordinator.go`
- `-flags +global_header` 之后插入 `-af silencedetect=noise=NdB:d=Ns`
- `fmt.Sprintf` 数字字段来自 `cfg.SmartEnd.SilenceDB / SilenceDurationS`
- 仅 `cfg.SmartEnd.Enabled=true` 时注入 (CFG-03 守门,Phase 25 调度不启 watcher 时不增加 ffmpeg 开销)
- T-24-07 threat model: 数字字段受 cfg.Validate 限定范围 (SilenceDB ∈ [-100,0])

### Task 3 — StartRecordingWithConfig 构造 ActivityWatcher + StopRecording 优雅关闭
- File: `internal/recorder/coordinator.go`
- `RecordingProcess` 新增 `ActivityWatcher *ActivityWatcher` + `taskEndedCh chan struct{}` 字段
- `SimpleRecordingCoordinator` 新增 `huaweiCli HuaweiStateClient` 字段 + `SetHuaweiCli(cli)` setter (Phase 25 入口)
- `StartRecordingWithConfig` 在 `c.mu.Unlock()` 之后:若 `cfg.SmartEnd.Enabled` 构造 `NewActivityWatcher(c.config, c.huaweiClient(), mkvPath, logFile, c.logger)` + 启动 4 goroutines + 注册 `process.OnReconnect = watcher.OnReconnect`
- `StopRecording` 在 `process.CancelFunc()` 之前 `process.ActivityWatcher.Stop()` (避免 watcher 在 ffmpeg 已死时读 logFile 触发 EOF 误报)

## Verification (all PASS)
- `go build ./...` → exit 0
- `go vet ./internal/recorder` → exit 0 (no warnings)
- `go test ./internal/recorder -count=1` → PASS (existing tests unbroken)

## Key Greps
- `OnReconnect func()`: 1 行
- `process.OnReconnect()`: 1 行 (restartRecording)
- `silencedetect=noise=`: 1 行 (buildRecordingCommand)
- `NewActivityWatcher`: 1 行 (StartRecordingWithConfig)
- `ActivityWatcher.Stop`: 2 行 (StopRecording 2 处)
- `huaweiCli` / `SetHuaweiCli`: 8 / 3 处

## Delivery Notes
- **Quota-aware execution**: 子代理因 429 配额失败,Plan 24-03 由 orchestrator 内联完成 (single-edit 多处同步应用 + 1 次错误修正:NewActivityWatcher 首参实际为 `*config.Config` 而非 `*config.SmartEndConfig`)。
- **Phase 25 接线责任**: `SetHuaweiCli` 由 cmd/server/app.go 在 Phase 25 调用,本阶段不在 app.go 改动范围。
- **未触动文件**: STATE.md / ROADMAP.md (将由 orchestrator 在 wave 3 close-out 统一更新)。

## Downstream Consumers
- **Plan 24-04 (测试)** — 可基于现有 RecordingProcess + ActivityWatcher 字段写 `TestBuildRecordingCommand_SilenceDetect` + `TestRestartRecording_OnReconnect`
- **Phase 25 (scheduler)** — `<-rec.taskEndedCh` + `watcher.IsActive()` + `watcher.Snapshot()` 直接可用
