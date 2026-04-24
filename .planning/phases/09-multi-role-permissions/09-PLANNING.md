# Phase 9: Multi-Role Permissions & Shared Viewer - Planning Breakdown

## Plan Structure

### Wave 1: Foundation (Database + Models)
- **09-01-PLAN.md**: Database migration and UserRole junction model
- **09-02-PLAN.md**: User and Role model updates (many-to-many relationship)

### Wave 2: Service Layer (Business Logic)
- **09-03-PLAN.md**: Permission checking and data visibility logic
- **09-04-PLAN.md**: UserService updates (AssignRoles, UpdateRoles) and shared_viewer creation

### Wave 3: API + Frontend (Integration)
- **09-05-PLAN.md**: Frontend user management UI updates (multi-select, badges)

## Requirements Coverage

| Decision ID | Plan | Description |
|-------------|------|-------------|
| D-01 | 09-03 | shared_viewer controls visibility only, not permissions |
| D-02 | 09-03 | shared_viewers see all data (skip created_by filter) |
| D-03 | 09-03 | Operation permissions from other roles, not shared_viewer |
| D-04 | 09-04 | shared_viewer role storage and display name |
| D-05 | 09-01, 09-02 | users_roles many-to-many table |
| D-06 | 09-02 | User.Roles []Role, remove RoleID |
| D-07 | 09-03 | HasPermission() OR logic across roles |
| D-08 | 09-01 | Migration preserves existing roles |
| D-09 | 09-03 | Data ownership exists (CreatedBy fields) |
| D-10 | 09-03 | Current behavior: users see own data |
| D-11 | 09-03 | shared_viewers skip created_by filter |
| D-12 | 09-03 | Visibility before permission checks |
| D-13 | 09-04 | Admin-only shared_viewer assignment |
| D-14 | 09-05 | Multi-select role UI |
| D-15 | 09-04 | Audit log for role changes |
| D-16 | 09-01 | Migration steps order |
| D-17 | 09-01 | Rollback plan |
| D-18 | 09-01 | Migration verification |

## Dependency Graph

```
09-01 (Migration)
    ↓
09-02 (Models) → depends on 09-01
    ↓
09-03 (Permissions) → depends on 09-02
09-04 (UserService) → depends on 09-02
    ↓
09-05 (Frontend) → depends on 09-04
```

## Files Modified

### Backend
- `internal/migrations/006_multi_role_migration.go` (NEW)
- `internal/models/user_role.go` (NEW)
- `internal/models/user.go` (MODIFY)
- `internal/models/role.go` (MODIFY)
- `internal/models/permission_constants.go` (MODIFY)
- `internal/services/user_service.go` (MODIFY)
- `internal/services/video_file_service.go` (MODIFY)
- `internal/handlers/user_handler.go` (MODIFY)

### Frontend
- `frontend/src/types/user.ts` (MODIFY)
- `frontend/src/api/user.ts` (MODIFY)
- `frontend/src/pages/system/users/index.tsx` (MODIFY)

## Test Strategy (Wave 0)

Each plan will include test file creation following Phase 8 Wave 0 pattern:
- Unit tests for model changes
- Integration tests for migration
- Unit tests for permission logic
- Frontend component tests for UI changes
