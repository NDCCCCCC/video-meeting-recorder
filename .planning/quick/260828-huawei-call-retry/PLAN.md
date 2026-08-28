---
title: 修复华为会议连接瞬时失败导致录制任务报错
type: quick-fix
status: in-progress
created: 2026-08-28
---

# 华为会议连接瞬时失败修复

## 问题
task_id=116 启动录制时，连接华为会议失败：sentinel_type=ErrInternal。重启服务后恢复，
符合"进程内缓存客户端死亡但本地 hasSession() 仍为 true"特征。

## 根因推断
manager.go:407 client.CallConference 失败时，clients[configID] 缓存的客户端持有远端已
踢掉的 session，但 hasSession() 仅查本地 IsExpired()，复用后立即失败。

## 方案
- 方案二（先做，0 风险）：manager.go:407 失败路径补日志，输出 HuaweiError.Code，便于运维定位
- 方案一（再做）：huawei_conference_connector.go:78 SafeCallConference 失败后移除客户端
  缓存并重试一次；需在 manager.go 新增导出 RemoveClient

## 涉及文件
- internal/huawei/manager.go（方案二日志 + 方案一 RemoveClient 导出）
- internal/services/video_recording/huawei_conference_connector.go（方案一重试）
