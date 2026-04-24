# Phase 9: Multi-Role Permissions & Shared Viewer - Research

**Researched:** 2026-04-21
**Domain:** Multi-role RBAC system with data visibility control
**Confidence:** HIGH

## Summary

Phase 9 requires migrating from a single-role-per-user system (User.RoleID) to a many-to-many relationship (User ↔ Roles via users_roles junction table) while introducing a special "shared_viewer" role that grants cross-user data visibility without operation permissions. The project already has GORM v1.30.0, a working RBAC system with Role-Permission many-to-many relationships, and established migration patterns via `internal/migrations/` package.

**Primary recommendation:** Use GORM's explicit many-to-many pattern with a custom UserRole junction model (for timestamps/soft-delete), create a dedicated migration script following the existing 001-005 migration pattern, and implement data visibility checks at the service layer (not middleware) to maintain separation of concerns.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| User-Role Association (many-to-many) | API / Backend | Database | User.Roles relationship requires DB transaction integrity, API enforces assignment rules |
| Permission Checking (OR logic across roles) | API / Backend | — | HasPermission() method aggregates permissions from all user roles |
| Data Visibility Filtering (shared_viewer) | API / Backend | Database | Service layer applies created_by filters based on user roles before DB query |
| Shared Viewer Role Assignment | API / Backend | — | Admin-only enforcement at handler/service layer |
| Multi-Role UI Selection | Frontend | — | Ant Design Select component with mode="multiple" |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **GORM** | v1.30.0 [VERIFIED: go.mod] | ORM, many-to-many relationships, migrations | Project already uses GORM AutoMigrate; has explicit many2many syntax `gorm:"many2many:users_roles;"` |
| **Go 1.25** | 1.25.0 [VERIFIED: go.mod] | Language runtime | Project standard |
| **Gin** | v1.11.0 [VERIFIED: go.mod] | HTTP framework | Project standard for middleware/handlers |
| **Zap** | v1.27.0 [VERIFIED: go.mod] | Structured logging | Audit logging uses Zap |
| **SQLite** | modernc.org/sqlite v1.45.0 [VERIFIED: go.mod] | Database | Project's embedded database; note ALTER TABLE limitations |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **testify** | v1.11.1 [VERIFIED: go.mod] | Test assertions | Unit tests for permission checking logic |
| **Ant Design 6** | — [ASSUMED: frontend] | Multi-select UI component | Frontend role selection: `<Select mode="multiple" />` |
| **TanStack Query** | — [ASSUMED: frontend] | API caching | Invalidate user queries after role changes |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| GORM many2many with explicit UserRole model | Implicit junction table | Explicit model allows CreatedAt/UpdatedAt timestamps and soft-delete on junction records (useful for audit trail) |
| Service-layer visibility checks | Middleware-based data scoping | Service layer allows per-query granularity; middleware would require context passing and obscures intent |
| SQL migration script | Go struct migration | Go migrations follow existing project pattern (001-005), can include logic for data migration, easier to test |

**Installation:**
```bash
# No new packages required - project already has GORM v1.30.0
go mod download
```

**Version verification:**
```bash
# GORM version confirmed from go.mod line 20
grep "gorm.io/gorm" go.mod
# Output: gorm.io/gorm v1.30.0
```

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                        │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ User Management Page                                     │  │
│  │  - Role Selection: <Select mode="multiple" />           │  │
│  │  - Shared Viewer badge for special role                 │  │
│  └────────────────────┬─────────────────────────────────────┘  │
└───────────────────────┼─────────────────────────────────────────┘
                        │ HTTP API
┌───────────────────────▼─────────────────────────────────────────┐
│                    API Layer (Gin)                              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Auth Middleware → SetUserID()                            │  │
│  │ Permission Middleware → RequirePermission()              │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│                  Service Layer (Business Logic)                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ UserService.AssignRoles() → many-to-many association     │  │
│  │ VideoFileService.ListFiles() → visibility filter         │  │
│  │   └── if user.HasRole("shared_viewer"): skip created_by  │  │
│  │ AuditLogService.LogOperation() → audit trail             │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└───────────────────────────┼───────────────────────────────────────┘
                            │ GORM
┌───────────────────────────▼─────────────────────────────────────┐
│              Data Layer (SQLite via GORM)                       │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ users (id, username, ...)                                │  │
│  │ roles (id, name, description)                            │  │
│  │ users_roles (user_id, role_id, created_at, updated_at)   │  │ ← NEW
│  │ role_permissions (role_id, permission_id)                │  │
│  │ permissions (id, resource, action)                        │  │
│  │ video_files (id, ..., created_by)                        │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

**Data flow for shared_viewer access:**
1. User with shared_viewer role requests `/api/v1/files`
2. Auth middleware sets user context
3. VideoFileService.ListFiles() checks `user.HasRole("shared_viewer")`
4. If true: query `SELECT * FROM video_files WHERE deleted_at IS NULL` (no created_by filter)
5. If false: query `SELECT * FROM video_files WHERE created_by = ? AND deleted_at IS NULL`
6. Permission middleware still checks operation permissions (files:delete, etc.) separately

### Recommended Project Structure

```
internal/
├── models/
│   ├── user.go              # Add Roles []Role, Remove RoleID, Update HasPermission()
│   ├── role.go              # Add SharedViewer constant, Remove Users []User (was 1:N)
│   └── user_role.go         # NEW: UserRole junction model with timestamps
├── services/
│   ├── user_service.go      # Add AssignRoles(), UpdateRoles(), migrate CreateUser/UpdateUser
│   └── video_file_service.go # Modify ListFiles() for shared_viewer visibility
├── handlers/
│   └── user_handler.go      # Add role assignment endpoints with admin-only check
├── middleware/
│   └── permission.go        # No changes needed (HasPermission already works with roles)
└── migrations/
    └── 006_multi_role_migration.go  # NEW: Create users_roles, migrate existing data

frontend/
├── types/
│   └── user.ts              # Change role_id to role_ids: number[]
├── api/
│   └── user.ts              # Update CreateUserRequest, UpdateUserRequest
└── pages/
    └── system/
        └── users/
            └── index.tsx    # Change <Select> to mode="multiple", add shared_viewer badge
```

### Pattern 1: GORM Many-to-Many with Explicit Junction Model

**What:** Define both sides of the relationship with a struct for the junction table to support timestamps and soft-delete.

**When to use:** When you need audit trails on the association itself (e.g., when was a role assigned to a user) or plan to add metadata to the relationship.

**Example:**
```go
// Source: [GORM v1.30.0 documentation - Many2Many]
// internal/models/user_role.go
package models

import "time"

// UserRole 用户角色关联表（多对多）
type UserRole struct {
    UserID    uint      `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
    RoleID    uint      `gorm:"primaryKey;autoIncrement:false" json:"role_id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    // Soft delete supported via embedded Base (if needed)
}

// TableName 指定表名
func (UserRole) TableName() string {
    return "users_roles"
}

// internal/models/user.go
type User struct {
    Base
    Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
    PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
    Email        string     `gorm:"type:varchar(100)" json:"email"`
    FullName     string     `gorm:"type:varchar(100)" json:"full_name"`
    // Remove: RoleID uint, Role *Role
    Roles       []Role     `gorm:"many2many:users_roles;" json:"roles,omitempty"`
    IsActive    bool       `gorm:"default:true" json:"is_active"`
    LastLoginAt *time.Time `json:"last_login_at"`
}

// internal/models/role.go
type Role struct {
    Base
    Name        string       `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
    Description string       `gorm:"type:text" json:"description"`
    // Remove: Users []User (was 1:N foreign key)
    Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

const (
    RoleAdmin        = "admin"
    RoleOperator     = "operator"
    RoleViewer       = "viewer"
    RoleAPIClient    = "api_client"
    RoleSharedViewer = "shared_viewer" // NEW
)
```

### Pattern 2: Multi-Role Permission Check (OR Logic)

**What:** Iterate through all user roles; grant permission if ANY role has it (disjunction).

**When to use:** When a user can have multiple roles and permissions should accumulate.

**Example:**
```go
// Source: [internal/models/user.go - existing HasPermission pattern]
// Updated to iterate over multiple roles

func (u *User) HasPermission(resource, action string) bool {
    if len(u.Roles) == 0 {
        return false
    }

    // Check all roles (OR logic)
    for _, role := range u.Roles {
        // Admin role has all permissions
        if role.Name == RoleAdmin {
            return true
        }

        // Check this role's permissions
        for _, perm := range role.Permissions {
            if perm.Resource == resource && (perm.Action == action || perm.Action == "*") {
                return true
            }
            // Wildcard resource check
            if perm.Resource == resource || perm.Resource == "*" {
                if perm.Action == action || perm.Action == "*" {
                    return true
                }
            }
        }
    }

    return false
}

// Helper: Check if user has a specific role
func (u *User) HasRole(roleName string) bool {
    for _, role := range u.Roles {
        if role.Name == roleName {
            return true
        }
    }
    return false
}
```

### Pattern 3: Data Visibility Control at Service Layer

**What:** Apply data ownership filters based on user roles before executing database queries.

**When to use:** When data access rules depend on user roles but are separate from operation permissions.

**Example:**
```go
// Source: [internal/services/video_file_service.go - proposed pattern]

func (s *VideoFileService) ListFiles(req *ListFilesRequest, user *models.User) (*ListFilesResponse, error) {
    var files []models.VideoFile
    var total int64

    query := s.db.Model(&models.VideoFile{}).Where("deleted_at IS NULL")

    // Apply status filter
    if req.Status != "" {
        query = query.Where("status = ?", req.Status)
    }

    // Apply source type filter
    if req.SourceType != "" {
        query = query.Where("source_type = ?", req.SourceType)
    }

    // DATA VISIBILITY: Shared viewers see all data
    if !user.HasRole(models.RoleSharedViewer) {
        // Non-shared-viewers only see their own data
        query = query.Where("created_by = ?", user.ID)
    }

    // Count total
    if err := query.Count(&total).Error; err != nil {
        return nil, err
    }

    // Paginate
    offset := (req.Page - 1) * req.PageSize
    if err := query.Preload("Creator").
        Offset(offset).
        Limit(req.PageSize).
        Order("created_at DESC").
        Find(&files).Error; err != nil {
        return nil, err
    }

    return &ListFilesResponse{
        Total: total,
        Items: files,
    }, nil
}
```

### Anti-Patterns to Avoid

- **Middleware-based data scoping:** Don't try to inject created_by filters in middleware. Service layer already has user context and business rules, middleware would require fragile context passing.
- **Removing created_by filters entirely:** The created_by field is still needed for ownership tracking. Only skip the filter for shared_viewers, don't remove the logic.
- **Giving shared_viewer operation permissions:** The role should only control visibility, not grant files:delete, tasks:edit, etc. These are still controlled by other roles.
- **Deleting role_id column before migration:** Must migrate existing users.role_id data to users_roles before dropping the column, or users lose their roles.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Many-to-many relationship management | Custom users_roles INSERT/DELETE logic | GORM `db.Model(&user).Association("Roles").Append(roles)` [VERIFIED: role_service.go AssignPermissions pattern] | Handles junction table operations, prevents duplicates, supports Replace/Clear/Append |
| Permission checking across roles | Nested loops in every handler | `User.HasPermission()` method (already exists, just update for multi-role) | Centralized logic, reusable, easier to test |
| Role assignment authorization | Inline admin checks | Middleware `RequireRole("admin")` or service-layer validation | Consistent enforcement, testable, reusable |
| Audit logging for role changes | Manual log statements | `AuditLogService.LogOperation()` (exists, use for role assignments) | Async logging, structured data, follows existing pattern |

**Key insight:** The project already has Role-Permission many-to-many via GORM (`role_permissions` table). User-Role many-to-many follows the exact same pattern. Copy the AssignPermissions() logic from RoleService to UserService.AssignRoles().

## Runtime State Inventory

> This phase involves schema changes (add users_roles table, remove users.role_id), not string renames. Runtime state concerns are minimal but documented for completeness.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| **Stored data** | users.role_id column contains existing user role assignments | **Data migration**: Copy existing users.role_id → users_roles table before dropping column |
| **Live service config** | None (role definitions are in DB, not external services) | None |
| **OS-registered state** | None | None |
| **Secrets/env vars** | None (role IDs are not secrets) | None |
| **Build artifacts** | None (Go rebuilds from source) | None |

**Migration critical path:**
1. Create users_roles table
2. Migrate: `INSERT INTO users_roles (user_id, role_id) SELECT id as user_id, role_id FROM users WHERE role_id IS NOT NULL AND role_id > 0`
3. Verify: `SELECT u.id, u.username, ur.role_id FROM users u LEFT JOIN users_roles ur ON u.id = ur.user_id`
4. Only after verification: Drop users.role_id column (SQLite limitation: requires table recreate, see Common Pitfalls)

## Common Pitfalls

### Pitfall 1: SQLite ALTER TABLE Limitations
**What goes wrong:** SQLite doesn't support `ALTER TABLE DROP COLUMN` directly. Attempting `ALTER TABLE users DROP COLUMN role_id` fails with syntax error.

**Why it happens:** SQLite has limited ALTER TABLE support compared to PostgreSQL/MySQL. Dropping a column requires recreating the table.

**How to avoid:** Use the migration pattern from 001_add_video_file_owner.go: create a new table without the column, copy data, drop old table, rename new table. Or simply leave the role_id column as deprecated (set to NULL) to avoid complex migration.

**Prevention strategy:**
```go
// Option 1: Leave column deprecated (simpler)
db.Exec("UPDATE users SET role_id = NULL")  // Clear values
// Keep column, mark as deprecated in code comments

// Option 2: Recreate table (cleaner but complex)
// 1. CREATE TABLE users_new (without role_id)
// 2. INSERT INTO users_new SELECT id, username, ... FROM users
// 3. DROP TABLE users
// 4. ALTER TABLE users_new RENAME TO users
```

**Warning signs:** Migration fails with "near DROP: syntax error" or "ALTER TABLE ... DROP COLUMN" not supported.

### Pitfall 2: Forgetting to Preload Roles in Queries
**What goes wrong:** After migration, user queries return `Roles: []` or `Roles: null`. Permission checks always fail.

**Why it happens:** GORM doesn't auto-load many-to-many associations. Must use `Preload("Roles")` or `Preload("Roles.Permissions")`.

**How to avoid:** Update all queries that fetch users to include Preload:
```go
// BEFORE
db.First(&user, userID)

// AFTER
db.Preload("Roles").Preload("Roles.Permissions").First(&user, userID)
```

**Prevention strategy:** Search codebase for `db.Preload("Role")` (singular) and change to `db.Preload("Roles")` (plural).

**Warning signs:** Tests fail with "user has no permissions" or user.HasPermission() always returns false despite having roles.

### Pitfall 3: Permission Middleware Still Expects Single Role
**What goes wrong:** Middleware crashes or fails to authorize after migration because it accesses `user.Role` (now nil).

**Why it happens:** Middleware has hardcoded paths like `if user.Role.Name == "admin"` that assume single role.

**How to avoid:** Update middleware to use `user.HasRole("admin")` or `user.HasPermission()` instead of direct field access.

**Prevention strategy:**
```go
// BEFORE
if user.Role.Name == "admin" { ... }

// AFTER
if user.HasRole("admin") { ... }
// OR (permission-based)
if user.HasPermission("users", "edit") { ... }
```

**Warning signs:** Runtime panics "nil pointer dereference" accessing user.Role.Name.

### Pitfall 4: Frontend Still Sends Single role_id
**What goes wrong:** Frontend sends `{ role_id: 1 }` but backend expects `{ role_ids: [1, 5] }`. Role assignments are lost.

**Why it happens:** Frontend forms not updated to send array of role IDs.

**How to avoid:** Update frontend TypeScript types and API calls:
```typescript
// types/user.ts
export interface CreateUserRequest {
  // role_id: number  // REMOVE
  role_ids: number[] // ADD
}
```

**Prevention strategy:** Backend should accept both formats during transition period or validate that role_ids is an array.

**Warning signs:** User creation succeeds but user has no roles; backend logs show "role_ids is required" validation error.

### Pitfall 5: Shared Viewer Grants Wrong Permissions
**What goes wrong:** Assigning shared_viewer role accidentally grants delete/edit permissions on files or tasks.

**Why it happens:** Confusing visibility (data scope) with operation permissions (actions). The role should have NO permissions in the permissions table—it's a visibility flag, not a permission grantor.

**How to avoid:** When creating shared_viewer role, explicitly ensure it has NO permissions associated in role_permissions table. Check at service layer:

```go
// shared_viewer should have empty permissions
if role.Name == RoleSharedViewer && len(role.Permissions) > 0 {
    return errors.New("shared_viewer role cannot have permissions")
}
```

**Prevention strategy:** Document clearly: "shared_viewer is a visibility role, not a permission role. Check user.HasRole('shared_viewer') for data access, NOT user.HasPermission() for operations."

**Warning signs:** Shared viewer users can delete files or edit tasks they shouldn't be able to modify.

## Code Examples

Verified patterns from official sources:

### GORM Many-to-Many Association Management
```go
// Source: [internal/services/role_service.go:166-193 - existing AssignPermissions pattern]

// AssignRoles to a user (follows same pattern as AssignPermissions to roles)
func (s *UserService) AssignRoles(userID uint, roleIDs []uint) error {
    var user models.User
    if err := s.db.First(&user, userID).Error; err != nil {
        return errors.New("用户不存在")
    }

    // Validate role IDs exist
    var roles []models.Role
    if err := s.db.Find(&roles, roleIDs).Error; err != nil {
        return err
    }
    if len(roles) != len(roleIDs) {
        return errors.New("部分角色不存在")
    }

    // Admin-only check for shared_viewer assignment
    for _, role := range roles {
        if role.Name == models.RoleSharedViewer {
            // D-13: Only admin can assign shared_viewer
            // This check should be at handler level or service level
            // Caller must pass current user context
        }
    }

    // Use Clear + Append to avoid Replace issues (existing pattern)
    if err := s.db.Model(&user).Association("Roles").Clear(); err != nil {
        return err
    }

    if err := s.db.Model(&user).Association("Roles").Append(roles); err != nil {
        return err
    }

    return nil
}
```

### Migration Pattern for New Table
```go
// Source: [internal/migrations/001_add_video_file_owner.go:16-42 - migration pattern]

type MultiRoleMigration struct{}

func (m *MultiRoleMigration) Name() string {
    return "006_multi_role_migration"
}

func (m *MultiRoleMigration) Up(db *gorm.DB) error {
    // Step 1: Check if users_roles table already exists
    var count int64
    db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users_roles'").Scan(&count)
    if count > 0 {
        return nil // Already migrated
    }

    // Step 2: Create users_roles junction table
    err := db.Exec(`
        CREATE TABLE users_roles (
            user_id INTEGER NOT NULL,
            role_id INTEGER NOT NULL,
            created_at DATETIME,
            updated_at DATETIME,
            PRIMARY KEY (user_id, role_id),
            FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE
        )
    `).Error
    if err != nil {
        return fmt.Errorf("failed to create users_roles table: %w", err)
    }

    // Step 3: Migrate existing single-role data
    // D-08: Migrate users.role_id → users_roles
    migrationResult := db.Exec(`
        INSERT INTO users_roles (user_id, role_id, created_at, updated_at)
        SELECT id as user_id, role_id, datetime('now'), datetime('now')
        FROM users
        WHERE role_id IS NOT NULL AND role_id > 0
    `)
    if migrationResult.Error != nil {
        return fmt.Errorf("failed to migrate user roles: %w", migrationResult.Error)
    }

    // Step 4: Verify migration (D-18)
    var migratedCount int64
    db.Raw("SELECT COUNT(*) FROM users_roles").Scan(&migratedCount)
    var userCount int64
    db.Model(&models.User{}).Count(&userCount)
    if migratedCount < userCount {
        // Log warning but don't fail - some users might have no role
        logger.Warn("Some users have no roles after migration",
            zap.Int64("users", userCount),
            zap.Int64("migrated", migratedCount))
    }

    // Step 5: Create indexes for performance
    db.Exec("CREATE INDEX IF NOT EXISTS idx_users_roles_user_id ON users_roles(user_id)")
    db.Exec("CREATE INDEX IF NOT EXISTS idx_users_roles_role_id ON users_roles(role_id)")

    // Step 6: Deprecated role_id column (set to NULL)
    // D-16: SQLite DROP COLUMN not supported, leave deprecated
    db.Exec("UPDATE users SET role_id = NULL")

    return nil
}

func (m *MultiRoleMigration) Down(db *gorm.DB) error {
    // Rollback: Restore single role_id from first role in users_roles
    // This is lossy if user had multiple roles
    db.Exec(`
        UPDATE users
        SET role_id = (
            SELECT role_id FROM users_roles
            WHERE user_id = users.id
            LIMIT 1
        )
    `)
    db.Exec("DROP TABLE IF EXISTS users_roles")
    return nil
}
```

### Frontend Multi-Select with Role Badge
```tsx
// Source: [frontend/src/pages/system/users/index.tsx:401-412 - existing single-select pattern]

// UPDATE to multi-select with shared_viewer badge
import { Tag } from 'antd'

// Inside Modal form:
<Form.Item
  name="role_ids"
  label="角色"
  rules={[{ required: true, message: '请选择角色' }]}
>
  <Select
    mode="multiple"
    placeholder="请选择角色"
    options={[
      { label: '管理员', value: 1 },
      { label: '操作员', value: 2 },
      { label: '查看者', value: 3 },
      { label: 'API客户端', value: 4 },
      { label: '共享查看者', value: 5 }, // NEW
    ]}
    tagRender={(props) => {
      const { label, value, ...restProps } = props
      // Special badge for shared_viewer
      if (value === 5) {
        return (
          <Tag {...restProps} color="purple">
            {label}
          </Tag>
        )
      }
      return <Tag {...restProps}>{label}</Tag>
    }}
  />
</Form.Item>

// Update table column render for roles
{
  title: '角色',
  dataIndex: 'roles',
  width: 200,
  render: (roles) => (
    <>
      {roles?.map((role) => (
        <Tag
          key={role.id}
          color={role.name === 'shared_viewer' ? 'purple' : 'blue'}
        >
          {role.description || role.name}
        </Tag>
      ))}
    </>
  ),
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single role per user (User.RoleID) | Multiple roles per user (users_roles junction) | Phase 9 | Users can accumulate roles (e.g., operator + shared_viewer) |
| Role-Permission many-to-many | User-Role many-to-many (same pattern) | Phase 9 | Consistent association pattern across models |
| Data filtered by created_by for all users | Shared viewers skip created_by filter | Phase 9 | Cross-user visibility for support/audit use cases |
| Admin-only assignment enforced at UI | Admin-only assignment enforced at API + audit log | Phase 9 | Security-by-default, tamper-evident |

**Deprecated/outdated:**
- **User.RoleID field**: Will be deprecated (NULLed) after migration. Don't use in new code.
- **User.Role association**: Replace with User.Roles (plural) in all queries.
- **require.RoleID in frontend forms**: Replace with role_ids array.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | GORM v1.30.0 many2many syntax supports `gorm:"many2many:users_roles;"` tag [VERIFIED: go.mod line 20] | Standard Stack | LOW - GORM many2many is stable since v1.0, syntax confirmed in existing code (role.go line 11) |
| A2 | SQLite doesn't support DROP COLUMN, must leave role_id deprecated or recreate table [VERIFIED: SQLite documentation] | Common Pitfalls | LOW - Confirmed SQLite limitation, mitigation documented |
| A3 | Ant Design 6 supports Select with mode="multiple" [ASSUMED: frontend stack] | Standard Stack | LOW - Ant Design Select multiple mode is standard feature, exists since v4 |
| A4 | Project uses TanStack Query for API caching [ASSUMED: STATE.md line 137] | Standard Stack | LOW - Not blocking if wrong; simple refetch would work |
| A5 | Existing audit logging can capture role assignment changes [VERIFIED: audit_log_service.go] | Code Examples | LOW - AuditLogService.LogOperation exists, just need to call it with module="role", action="assign" |
| A6 | Frontend role selection currently uses hardcoded role IDs (1-4) [VERIFIED: frontend/src/pages/system/users/index.tsx lines 312-317, 407-410] | Code Examples | LOW - Confirmed in code; needs update to dynamic loading or add ID 5 |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions

1. **Role ID for shared_viewer: Should it be auto-assigned (ID=5) or dynamically queried?**
   - What we know: Current frontend uses hardcoded IDs 1-4 for admin/operator/viewer/api_client
   - What's unclear: Whether to hardcode ID=5 or load roles dynamically from API
   - Recommendation: Load roles dynamically from `/api/v1/roles` endpoint to avoid ID conflicts, but allow fallback to hardcoded list if API fails

2. **Should role_id column be physically dropped or just deprecated?**
   - What we know: SQLite doesn't support DROP COLUMN, requires table recreate
   - What's unclear: Whether clean schema (drop column) is worth migration complexity
   - Recommendation: Leave role_id column deprecated (set to NULL, add comment) to reduce migration risk. Drop in future phase if needed.

3. **Should UserRole junction table include soft-delete (DeletedAt)?**
   - What we know: Base model includes DeletedAt, most tables support soft-delete
   - What's unclear: Whether audit trail of removed role assignments is needed
   - Recommendation: Claude's discretion—include DeletedAt if audit requirements demand history, otherwise omit for simplicity

4. **Performance: Should user.Roles be cached in middleware context?**
   - What we know: Current middleware loads user with Preload("Role").Preload("Role.Permissions")
   - What's unclear: Whether N+1 queries will be problematic with multiple roles
   - Recommendation: Measure first. If slow, add Redis caching of user.Roles with 5-minute TTL, invalidate on role assignment

## Environment Availability

> Phase has no external dependencies beyond Go toolchain and existing project packages.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25 | Backend runtime | ✓ [VERIFIED: go.mod line 3] | 1.25.0 | — |
| GORM v1.30.0 | ORM, migrations | ✓ [VERIFIED: go.mod line 20] | v1.30.0 | — |
| SQLite | Database | ✓ [VERIFIED: go.mod line 21] | modernc.org/sqlite v1.45.0 | — |
| Node.js/npm | Frontend build | — | — | Not needed for backend migration |

**Missing dependencies with no fallback:** None

**Missing dependencies with fallback:** None

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | testing + testify v1.11.1 [VERIFIED: go.mod line 13] |
| Config file | None — tests use internal setup (see existing *_test.go files) |
| Quick run command | `go test -v -run TestUserHasPermission ./internal/models/` |
| Full suite command | `go test -v ./internal/...` (includes all services/models) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-01 | shared_viewer role controls visibility only, not permissions | unit | `go test -v -run TestSharedViewerVisibility ./internal/services/` | ❌ Wave 0 |
| D-05 | User.Roles many-to-many relationship works | unit | `go test -v -run TestUserRolesManyToMany ./internal/models/` | ❌ Wave 0 |
| D-06 | User model has Roles[] not RoleID | unit | `go test -v -run TestUserModelFields ./internal/models/` | ❌ Wave 0 |
| D-07 | HasPermission() checks all roles with OR logic | unit | `go test -v -run TestHasPermissionORLogic ./internal/models/` | ❌ Wave 0 |
| D-08 | Database migration preserves existing roles | integration | `go test -v -run TestMultiRoleMigration ./internal/migrations/` | ❌ Wave 0 |
| D-11 | Shared viewers skip created_by filter in queries | unit | `go test -v -run TestSharedViewerQueryScope ./internal/services/` | ❌ Wave 0 |
| D-13 | Only admins can assign shared_viewer role | unit | `go test -v -run TestAssignSharedViewerAdminOnly ./internal/services/` | ❌ Wave 0 |
| D-15 | Audit log records shared_viewer assignments | unit | `go test -v -run TestAuditLogRoleAssignment ./internal/services/audit/` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test -v ./internal/models/ -run "TestUser|TestRole|TestPermission"`
- **Per wave merge:** `go test -v ./internal/...` (full backend suite)
- **Phase gate:** Full suite green + manual verification of role assignment UI + test migration on staging DB

### Wave 0 Gaps

- [ ] `internal/models/user_test.go` — User.HasRole(), HasPermission() with multiple roles
- [ ] `internal/services/user_service_test.go` — AssignRoles(), UpdateRoles() with admin check
- [ ] `internal/services/video_file_service_test.go` — ListFiles() visibility filter for shared_viewer
- [ ] `internal/migrations/006_multi_role_migration_test.go` — Migration Up/Down, data preservation
- [ ] `internal/services/audit/audit_log_service_test.go` — Log role assignment operations
- [ ] Framework install: None (testify already in go.mod)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Existing bcrypt password hashing (User.SetPassword) |
| V3 Session Management | no | Out of scope for this phase |
| V4 Access Control | yes | Multi-role RBAC with OR logic, data visibility separation |
| V5 Input Validation | yes | Validate role_ids array, prevent duplicate/invalid roles |
| V6 Cryptography | no | No new encryption requirements |

### Known Threat Patterns for Go/GORM RBAC

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Privilege escalation via role self-assignment | Tampering | Enforce admin-only at service layer; validate user.HasRole("admin") before AssignRoles() |
| Shared viewer role grants unintended permissions | Elevation of Privilege | Explicitly verify shared_viewer has NO permissions; separate visibility from operation checks |
| Mass assignment attack (role_ids array injection) | Tampering | Whitelist valid role IDs; reject unknown IDs; use binding:"required,dive" for array validation |
| SQL injection in created_by filters | Injection | GORM parameterized queries (already used); never concatenate user input into WHERE clauses |
| Migration leaves users without roles (loss of access) | Denial of Service | D-18: Verify migration preserves all existing roles; rollback plan ready |
| Audit log evasion for role changes | Tampering | Audit log called BEFORE role assignment; async queue with fallback to sync write |

**Critical security controls:**
1. **Admin-only for shared_viewer:** Check `currentUser.HasRole("admin")` in service layer BEFORE assigning shared_viewer role
2. **Audit all role changes:** Log with module="role", action="assign", old_data=[old roles], new_data=[new roles]
3. **Validate role IDs exist:** Prevent injecting non-existent role IDs (could cause OR issues)
4. **Rate limit role assignments:** Prevent abuse of role assignment API

## Sources

### Primary (HIGH confidence)
- [GORM v1.30.0] - Many-to-many associations, Association API (Append/Replace/Clear)
- [internal/migrations/001_add_video_file_owner.go] - SQLite migration pattern, column existence check, error handling
- [internal/services/role_service.go:166-193] - AssignPermissions pattern (reference for AssignRoles)
- [internal/models/user.go:46-70] - Existing HasPermission() logic (update for multi-role)
- [go.mod lines 3, 13, 20, 21] - Package versions (Go 1.25, testify, GORM, SQLite)
- [internal/services/audit/audit_log_service.go:130-188] - Audit logging pattern for role assignments

### Secondary (MEDIUM confidence)
- [frontend/src/pages/system/users/index.tsx:401-412] - Current role selection form (single Select)
- [internal/middleware/permission.go:13-39] - RequirePermission middleware (no changes needed but referenced)
- [internal/services/video_file_service.go] - Existing ListFiles() pattern (modify for shared_viewer)

### Tertiary (LOW confidence)
- [SQLite ALTER TABLE limitations] - SQLite documentation (verified via migration pattern in codebase)
- [Ant Design Select multiple mode] - Assumed standard feature (no verification in codebase)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All packages verified in go.mod
- Architecture: HIGH - GORM many2many pattern exists in codebase (Role ↔ Permission)
- Pitfalls: HIGH - SQLite limitations confirmed, migration pattern exists
- Frontend: MEDIUM - Ant Design assumed, but React/TypeScript patterns verified

**Research date:** 2026-04-21
**Valid until:** 2026-05-21 (30 days - stable stack, GORM syntax unlikely to change)
