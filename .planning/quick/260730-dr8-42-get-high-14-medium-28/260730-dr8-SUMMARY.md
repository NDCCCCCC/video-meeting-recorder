---
phase: quick
plan: 260730-dr8
subsystem: audit
completed: 2026-07-30
tags: [audit, get-endpoints, security]
key-files:
  modified:
    - internal/models/audit_log.go
    - cmd/server/app.go
---

# Quick Task 260730-dr8 Summary

为 48 个敏感 GET 路由补齐读取、列表和导出审计，并保留 3 个审计自查询路由不审计以避免递归噪声。

## Commits

1. `4565333` — ActionRead / ActionList 常量
2. `9b13ed0` — HIGH 13 个 GET 路由
3. Task 3 commit — MEDIUM 35 个 GET 路由及本摘要

## 48 Endpoint Mapping

| Module | Action | Endpoints |
|---|---|---|
| User | Read | `/auth/me`, `/users/profile`, `/users/:id` |
| User | List | `/users` |
| Admin | Read | `/admin/auth/config`, `/admin/auth/me` |
| System | Read | `/scheduler/debug`, `/system/config`, `/dashboard/stats` |
| File | Export | `/files/:id/download`, `/files/download/:token`, `/files/share/:token` |
| File | Read | `/storage/quota`, `/files/stats`, `/files/:id` |
| File | List | `/storage`, `/files` |
| PPT | Export | `/ppts/:id/download`, `/ppts/:id/slides/:resolution/:filename` |
| PPT | Read | `/ppts/:id/duplicates` |
| PPT | List | `/videos/:id/ppts`, `/ppts/:id/slides` |
| Task | Export | `/recordings/:id/preview/stream/:file` |
| Task | Read | `/recordings/:id`, `/recordings/:id/conversion-status`, `/recordings/:id/preview`, `/videos/:id/split-status` |
| Task | List | `/recordings`, `/videos/:id/segments` |
| Role | Read | `/roles/:id`, `/roles/:id/permissions` |
| Role | List | `/roles`, `/permissions` |
| APIKey | Read | `/apikeys/:id`, `/apikeys/:id/logs`, `/apikeys/:id/summary` |
| APIKey | List | `/apikeys` |
| InputConfig | Read | `/input-configs/active`, `/input-configs/:id` |
| InputConfig | List | `/input-configs` |
| Transcription | Read | `/videos/:id/transcription-status`, `/videos/:id/transcription-text`, `/transcriptions/:videoFileId/timestamps`, `/transcriptions/batch/:id` |
| Transcription | List | `/transcriptions/active` |
| Notification | Read | `/notifications/unread-count`, `/notifications/settings` |
| Notification | List | `/notifications` |

## Intentionally Skipped Audit Self-Queries

- `/audit/logs`
- `/audit/logs/:id`
- `/audit/statistics`

These endpoints query the audit system itself and remain unaudited to prevent recursive/noisy audit traffic.

## P2 Remaining

- Public health and basic system-stat endpoints remain outside this sensitive GET scope.
- Read-only password validation and PPT batch-check remain intentionally excluded under their existing route comments.
- Future endpoint additions should select `ActionRead`, `ActionList`, or `ActionExport` according to response semantics.

## Verification

- `auditOp(models.` count: 111 (63 baseline + 13 HIGH + 35 MEDIUM).
- String-literal module audit calls: 0.
- Frontend files modified by this worktree: 0.
- `go build ./internal/models/...`: passed.
- Full `go build ./...` and `go test ./internal/... ./cmd/...`: blocked by pre-existing repository/environment failures: tracked/generated `internal/frontend/dist` is absent (`internal/frontend/embed.go:9:12: pattern dist: no matching files found`), and `internal/auth/ip_restriction_test.go` references an undefined `ctx` at eight sites. No frontend or unrelated auth files were modified to work around these out-of-scope failures.

## Deviations from Plan

- The supplied full `EXPECTED_BASE` object was unavailable; the repository's matching pre-dispatch commit resolves to `ac069caa082e6d7107d40e5ff0e1f4bd64f21c7d` and was used.
- Full Go verification could not start due to the missing embedded frontend distribution described above; source-level checks and the models build passed.

## Self-Check: PASSED

All planned source files and this summary exist; the first two task commits are present, and the final task commit contains the remaining implementation and summary.
