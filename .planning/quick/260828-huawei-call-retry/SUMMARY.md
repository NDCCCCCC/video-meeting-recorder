---
status: complete
completed: 2026-08-28
---

# 华为会议连接瞬时失败修复 — 完成摘要

## 问题
生产环境 task_id=116 手动启动录制时连接华为会议失败（scheduler.go:383, sentinel_type=ErrInternal）。
日志时序显示 67ms 内完成"呼叫→失败→解锁"，即 `client.CallConference` 的 HTTP Post 立即失败。
重启服务后恢复正常 → 典型的进程内缓存客户端状态与远端失同步。

## 根因
`manager.GetClient()` 缓存的 `HuaweiClient`，其 `hasSession()` 仅检查本地 `session.IsExpired()`
（基于本地时钟），无法发现远端 MCU 已主动踢掉 session。复用该 client 时
`WEB_CallSiteAPI` 请求立即失败，被包成 ErrInternal 直接令录制任务失败。

## 修复内容
1. **方案二（诊断日志）** `internal/huawei/manager.go` SafeCallConference 失败路径：
   用 `errors.As` 提取 `HuaweiError`，输出 `huawei_error_code` / `huawei_error_message`
   （Code/Message 不含凭据，安全），非华为错误则记 SentinelField 分类。
2. **方案一（自愈重试）**：
   - `manager.go` 新增导出 `RemoveClient(ctx, configID)`（内部调 `removeClient`）。
   - `huawei_conference_connector.go` ConnectToConference：首次 `SafeCallConference`
     失败 → `RemoveClient` 清缓存 → 重试一次（走 createClient 重建会话）；
     重试仍失败才解锁终端并返回错误。

## 效果
- 下次发生同类故障时日志可直接区分"网络层错误"vs"华为侧拒绝（具体错误码）"。
- 会话失同步场景自动重建客户端恢复，无需重启服务。

## 验证
- `go build ./internal/huawei/... ./internal/services/video_recording/...` ✓
- `go test ./internal/huawei/... ./internal/services/video_recording/...` ✓ (2.842s)
- `golangci-lint run` 0 issues
- `go build ./...` cgo/sqlite3 失败为本机预存环境问题（stash 验证原始树同样失败），与改动无关
