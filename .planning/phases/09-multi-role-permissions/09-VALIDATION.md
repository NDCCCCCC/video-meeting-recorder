---
phase: 9
slug: multi-role-permissions
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-21
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | testing + testify v1.11.1 |
| **Config file** | None — tests use internal setup (see existing *_test.go files) |
| **Quick run command** | `go test -v -run TestUserHasPermission ./internal/models/` |
| **Full suite command** | `go test -v ./internal/...` (includes all services/models) |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -v ./internal/models/ -run "TestUser|TestRole|TestPermission"`
- **After every plan wave:** Run `go test -v ./internal/...` (full backend suite)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | D-08, D-16, D-18 | — | Migration preserves existing user roles | integration | `go test -v -run TestMultiRoleMigration ./internal/migrations/` | ❌ W0 | ⬜ pending |
| 09-01-02 | 01 | 1 | D-17 | — | Migration rollback safe | integration | `go test -v -run TestMultiRoleMigrationRollback ./internal/migrations/` | ❌ W0 | ⬜ pending |
| 09-01-03 | 01 | 1 | — | — | Test stubs created | unit | `go test -v -run TestUserRoleModel ./internal/models/` | ❌ W0 | ⬜ pending |
| 09-02-01 | 02 | 1 | D-05, D-06 | — | User.Roles many-to-many relationship works | unit | `go test -v -run TestUserRolesManyToMany ./internal/models/` | ❌ W0 | ⬜ pending |
| 09-02-02 | 02 | 1 | D-07 | — | HasPermission() checks all roles with OR logic | unit | `go test -v -run TestHasPermissionORLogic ./internal/models/` | ❌ W0 | ⬜ pending |
| 09-02-03 | 02 | 1 | D-04 | — | RoleSharedViewer constant exists | unit | `go test -v -run TestRoleSharedViewerConstant ./internal/models/` | ❌ W0 | ⬜ pending |
| 09-02-04 | 02 | 1 | — | — | Test stubs created | unit | `go test -v -run TestPermissionService ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-03-01 | 03 | 2 | D-01, D-02, D-11, D-12 | T-09-22 | Shared viewers skip created_by filter in queries | unit | `go test -v -run TestSharedViewerQueryScope ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-03-02 | 03 | 2 | D-03 | T-09-23 | Shared viewer has no operation permissions | unit | `go test -v -run TestSharedViewerNoOpPermissions ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-03-03 | 03 | 2 | — | — | Test stubs created | unit | `go test -v -run TestVideoFileService ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-04-01 | 04 | 2 | D-13 | T-09-19 | Only admins can assign shared_viewer role | unit | `go test -v -run TestAssignSharedViewerAdminOnly ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-04-02 | 04 | 2 | D-15 | — | Audit log records role assignment operations | unit | `go test -v -run TestAuditLogRoleAssignment ./internal/services/audit/` | ❌ W0 | ⬜ pending |
| 09-04-03 | 04 | 2 | D-05 | — | AssignRoles() uses GORM Association API | unit | `go test -v -run TestAssignRolesAssociation ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-04-04 | 04 | 2 | D-04 | — | shared_viewer role creation is idempotent | unit | `go test -v -run TestSharedViewerRoleSeed ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-04-05 | 04 | 2 | D-05, D-06 | — | CreateUser/UpdateUser use AssignRoles | unit | `go test -v -run TestUserCRUDWithRoles ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-04-06 | 04 | 2 | — | — | Test stubs created | unit | `go test -v -run TestUserService ./internal/services/` | ❌ W0 | ⬜ pending |
| 09-05-01 | 05 | 3 | D-14 | — | TypeScript types use role_ids array | N/A | Manual: `npx tsc --noEmit` | ✅ | ⬜ pending |
| 09-05-02 | 05 | 3 | D-14 | — | API functions send role_ids array | N/A | Manual: `grep role_ids frontend/src/api/user.ts` | ✅ | ⬜ pending |
| 09-05-03 | 05 | 3 | D-14 | — | Role form uses multi-select | N/A | Manual: `grep 'mode="multiple"' frontend/src/pages/system/users/index.tsx` | ✅ | ⬜ pending |
| 09-05-04 | 05 | 3 | D-04 | — | Shared viewer badge displays in purple | N/A | Manual: `grep "color=\"purple\"" frontend/src/pages/system/users/index.tsx` | ✅ | ⬜ pending |
| 09-05-05 | 05 | 3 | D-13, D-14 | T-09-19 | Admin-only check prevents assignment | N/A | Manual: Human verification checkpoint | ✅ | ⬜ pending |
| 09-05-06 | 05 | 3 | D-14 | — | Role filter includes shared_viewer | N/A | Manual: UI verification | ✅ | ⬜ pending |
| 09-05-07 | 05 | 3 | All | — | Human verification of multi-role UI | checkpoint | Human: `Type "approved" if UI works` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/models/user_test.go` — User.HasRole(), HasPermission() with multiple roles
- [ ] `internal/services/user_service_test.go` — AssignRoles(), UpdateRoles() with admin check
- [ ] `internal/services/video_file_service_test.go` — ListFiles() visibility filter for shared_viewer
- [ ] `internal/migrations/006_multi_role_migration_test.go` — Migration Up/Down, data preservation
- [ ] `internal/services/audit/audit_log_service_test.go` — Log role assignment operations
- [ ] Framework install: None (testify already in go.mod)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Multi-select role UI | D-14 | Frontend UI component requires visual verification | 1. Navigate to /system/users 2. Click "新建用户" 3. Verify role dropdown shows checkboxes 4. Select multiple roles 5. Verify shared_viewer tag displays in purple |
| Admin-only shared_viewer assignment | D-13, D-14 | Client-side security behavior requires user interaction | 1. As non-admin, attempt to assign shared_viewer 2. Verify error message displays 3. As admin, assign shared_viewer 4. Verify success |
| TypeScript compilation | D-14 | Type system validation | Run `npx tsc --noEmit` in frontend directory |

*Note: Backend behaviors have automated unit tests. Frontend requires manual verification for UI interactions.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (6 test files to be created)
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter (pending Wave 0 completion)

**Approval:** pending
