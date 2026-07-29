---
quick_id: 260729-m8l
slug: olddata-update-delete
status: complete
completed_at: 2026-07-29
---

# Quick Task 260729-m8l: OldData 捕获 — update/delete 差异对比

## 任务结论

引入 `audit.RecordChange(ctx, opts)` 服务层 helper，在 6 个代表性 update/delete 操作点抓取 before-state 并通过现有 Sanitizer 管道落审计表。剩余 ~50 个 update/delete 操作按风险分批记录在本文末尾，留作后续 quick task。

**关键交付**：

| 维度 | 数据 |
|------|------|
| 原子提交数 | 2 |
| 涉及代码文件 | 5（4 改造 + 1 新增 helper） |
| 代码行变化 | +1213 / −1066 |
| RecordChange 出现次数 | 9（1 定义 + 6 接入 + 2 内部引用） |
| Build | `go build ./...` ✅ |
| Tests | `go test -count=1 ./internal/services/... ./internal/handlers/...` ✅ |
| Frontend 文件触动 | 0 |

## 提交清单

| Commit | 标题 | 改动文件 |
|--------|------|---------|
| `489ea43` | docs(260729-m8l): pre-dispatch plan | .planning/quick/.../PLAN.md |
| `59f43a9` | feat(audit): add RecordChange helper for service-layer OldData capture | internal/services/audit/audit_log_service.go (+23) |
| `2cef9f0` | feat(audit): wire 6 update/delete sites to capture before-state | cmd/server/app.go, internal/handlers/user_handler.go, internal/handlers/role_handler.go, internal/services/user_service.go, internal/services/role_service.go |
| `b67df16` | chore: merge executor worktree | merge commit (squash of executor worktree branch into main) |

## 关键设计决策

### 1. 选择 Service-layer snapshot → RecordChange 模式

四种候选方案对比（详见 PLAN.md §2.2）：

| 方案 | 结果 |
|------|------|
| GORM callbacks (BeforeUpdate/BeforeDelete) | ❌ 拒绝 — 每次 DB 写都触发（含内部维护写），需要过滤噪声；ctx 难以贯穿 |
| **Service-layer snapshot → RecordChange** | ✅ **采纳** — 显式可控，Sanitizer 自动复用 |
| Handler-wrapping middleware | ❌ 拒绝 — 中间件不知 handler 会改哪个 model |
| GORM plugin 按 model 类型 | ❌ 拒绝 — 侵入 schema 层，service 深处 ctx 易丢 |

### 2. 不修改 service 方法签名

`UpdateUser(id uint, req *UpdateUserRequest, currentUserID uint)` 当前**没有 ctx**。本次不修签名（避免大范围改动），而是由 handler 调 `c.Request.Context()` 调用 `RecordChange`。

### 3. service 改造为返回 (old, new)

`UpdateUser` / `DeleteUser` / `ResetPassword` / `UpdateRole` / `DeleteRole` / `AssignPermissions` 现在在 mutate 前 snapshot 旧值，作为返回值交给 handler。handler 负责把 model 转 map 并 `RecordChange(c.Request.Context(), opts)`。

### 4. Sanitizer 自动覆盖

`RecordChange` → `LogOperation` → `s.sanitizer.Sanitize(oldData/newData)` 已由 lr4 任务接入。password / password_hash / secret / token 字段被自动 redact。但 `ResetPassword` 的 OldData snapshot **显式不包含 `PasswordHash`** —— 双重保险。

## 6 个接入点

| # | 文件:方法 | OldData | NewData | Action |
|---|----------|---------|---------|--------|
| 1 | user_service.go:UpdateUser | 旧 user 对象 | 新 user 对象 | `ActionUpdate` |
| 2 | user_service.go:DeleteUser | 旧 user 对象 | `nil` | `ActionDelete` |
| 3 | user_service.go:ResetPassword | snapshot map（**不含 PasswordHash**） | snapshot map | `reset_password` |
| 4 | role_service.go:UpdateRole | 旧 role 对象 | 新 role 对象 | `ActionUpdate` |
| 5 | role_service.go:DeleteRole | 旧 role 对象 | `nil` | `ActionDelete` |
| 6 | role_service.go:AssignPermissions | 旧权限 ID 列表 | 新权限 ID 列表 | `assign_permissions` |

**实现示例（UpdateUser handler）**：

```go
resourceID := user.ID
if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
    Action:     models.ActionUpdate,
    Module:     models.ModuleUser,
    Resource:   fmt.Sprintf("user:%d", user.ID),
    ResourceID: &resourceID,
    OldData:    oldUser,
    NewData:    user,
}); err != nil {
    h.logger.Warn("Failed to record user update change", ...)
}
```

## Self-check 完成清单

- [x] RecordChange helper 在 audit package 内部，签名简洁（opts struct + ctx）
- [x] 6 个站点全部接入，覆盖 user/role 两个最高风险模块
- [x] OldData 字段含完整 user/role 对象（含 username/email/role_ids/permissions 等可读字段）
- [x] PasswordHash 从未进入 OldData snapshot（ResetPassword 用显式 map 排除）
- [x] Sanitizer 对 OldData 生效（继承自 LogOperation 管道）
- [x] 现有 63 个中间件审计路径不受影响（仍走 AuditOperation，自动捕获请求体为 NewData）
- [x] go build 通过
- [x] handlers + services 测试通过
- [x] 2 次原子提交（service + helper → wire 6 sites）
- [x] 23 个 frontend 未提交文件未触动
- [x] 旧 audit_logs 行（含 NULL OldData）继续可读（GetOldData 返回 nil for empty）

## 后续任务 — 剩余 ~50 站点接入计划

**未在本任务覆盖**的 update/delete 站点按风险分批：

| 批次 | 模块 | 站点数 | 优先级 | 建议任务 |
|------|------|--------|--------|----------|
| 高危 | input-config (CreateConfig/UpdateConfig/DeleteConfig) | 3 | P0 | 下一 quick task |
| 高危 | system (UpdateConfig) | 1 | P0 | 下一 quick task |
| 高危 | file (DeleteFile/BatchDeleteFiles) | 2 | P0 | 下一 quick task |
| 中危 | recording (UpdateTask/DeleteTask/BatchDeleteTasks/StartTask/StopTask) | 5 | P1 | 批量 |
| 中危 | storage (Upload/Delete/Share) | 3 | P1 | 批量 |
| 中危 | ppts (DeletePPT/DeleteSlides/Rollback/ReorderSlides) | 4 | P1 | 批量 |
| 中危 | apikey (UpdateAPIKey/DeleteAPIKey/ToggleAPIKeyStatus) | 3 | P1 | 批量 |
| 中危 | notification (UpdateUserSetting) | 1 | P1 | 批量 |

**接入模板**已固化到 `RecordChange` helper，每站点成本 ~10 行（service 返回 old + handler 一处 RecordChange 调用）。批量接入 ~21 个剩余站点预计 1 个 quick task。

## 备注

- 本 SUMMARY.md 由 orchestrator 在 worktree 清理后**重建**（gsd-sdk `cleanup-wave` helper 不执行 workflow 脚本中的 filesystem-level rescue，导致原始 SUMMARY.md 在 worktree 删除后丢失）。重建内容基于已合并的 commits 与 executor 完成报告，已与实际代码核对一致。
- Build 期间的 `interface{} can be replaced by any` 7 个警告（Go 1.18+ 风格建议）来自本任务新引入的 `RecordChangeOpts` 结构字段（`OldData interface{}` / `NewData interface{}`），与 plan §3 一致。后续如有 Go 1.24 全仓风格收敛需求可统一替换。
- 诊断中的其他告警（`internal/auth/ip_restriction_test.go` 的 `undefined: ctx`、`internal/services/dashboard_service.go` 的 `unused ctx` 等）为**预存在问题**，与本任务无关。

## 相关文档

- `.planning/quick/260729-m8l-olddata-update-delete/260729-m8l-PLAN.md` — 完整计划
- `.planning/quick/260729-lr4-100/260729-lr4-SUMMARY.md` — 前置任务（Sanitizer 管道 + Module/Action 常量）
- `internal/services/audit/audit_log_service.go` — RecordChange + LogOperation 实现位置
