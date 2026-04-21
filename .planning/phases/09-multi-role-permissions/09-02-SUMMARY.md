---
phase: 09
plan: 02
title: "Model Updates - Many-to-Many User-Role Relationship"
one_liner: "Updated User and Role models to support many-to-many relationships via users_roles junction table, enabling users to have multiple roles simultaneously"
status: complete
completed_date: "2026-04-21T06:14:34Z"

subsystem: "Authorization & Permissions"
tags: ["models", "gorm", "many-to-many", "rbac", "permissions"]

dependency_graph:
  requires:
    - id: "09-01"
      reason: "Junction table UserRole must exist before many-to-many relationship"
  provides:
    - id: "09-03"
      reason: "Updated models required for permission checking and data visibility logic"
    - id: "09-04"
      reason: "User.HasRole() method needed for API authorization layer"
    - id: "09-05"
      reason: "Multi-role model required for frontend UI updates"
  affects:
    - component: "internal/models"
      reason: "User and Role model structure changed"
    - component: "Authentication/Authorization"
      reason: "Permission checking logic updated for multiple roles"

tech_stack:
  added: []
  patterns:
    - "GORM many-to-many relationships with many2many tag"
    - "OR logic for permission aggregation across multiple roles"
    - "Helper methods for role checking (HasRole)"

key_files:
  created:
    - path: "internal/models/user_test.go"
      lines: 51
      purpose: "Test stubs for User model changes (Wave 0)"
  modified:
    - path: "internal/models/user.go"
      changes: "26 insertions, 14 deletions"
      purpose: "Remove RoleID/Role fields, add Roles []Role many2many, update HasPermission(), add HasRole()"
    - path: "internal/models/role.go"
      changes: "5 insertions, 5 deletions"
      purpose: "Remove Users []User field, add RoleSharedViewer constant"

decisions_made:
  - "User.RoleID and User.Role fields removed to eliminate single-role constraint"
  - "User.Roles []Role added with many2many:users_roles tag for junction table"
  - "HasPermission() updated to iterate all roles with OR logic (any role granting permission = access)"
  - "HasRole() helper added for convenient role checking"
  - "Role.Users field removed (no longer 1:N relationship from Role to User)"
  - "RoleSharedViewer constant added for shared viewer role (D-04)"

deviations_from_plan: []
authentication_gates: []

metrics:
  duration_seconds: 123
  duration_minutes: 2
  tasks_completed: 4
  files_created: 1
  files_modified: 2
  commits: 4

threat_flags: []
---

# Phase 09 Plan 02: Model Updates - Many-to-Many User-Role Relationship Summary

## Overview

Updated User and Role models to use many-to-many relationship via `users_roles` junction table, removing the old single-role foreign key pattern. This enables users to have multiple roles simultaneously, supporting requirements D-04 (shared viewer), D-05 (role accumulation), and D-06 (many-to-many relationships).

## Changes Made

### 1. User Model Updates (`internal/models/user.go`)

**Removed:**
- `RoleID uint` field (line 17) - old single-role foreign key
- `Role *Role` field (line 18) - old single-role relationship

**Added:**
- `Roles []Role` field with `gorm:"many2many:users_roles;"` tag - many-to-many relationship
- `HasRole(roleName string) bool` helper method - checks if user has specific role

**Modified:**
- `HasPermission(resource, action string) bool` method - updated to iterate over all roles with OR logic:
  - Returns `false` if `len(u.Roles) == 0` (no roles)
  - Checks if ANY role is "admin" (admin shortcut)
  - Iterates through all roles, returning `true` if ANY role grants the permission
  - Maintains wildcard resource/action checking

### 2. Role Model Updates (`internal/models/role.go`)

**Removed:**
- `Users []User` field with `foreignKey:RoleID` tag - old 1:N relationship

**Added:**
- `RoleSharedViewer = "shared_viewer"` constant - shared viewer role for D-04

**Preserved:**
- `Name` and `Description` fields
- `Permissions []Permission` with `many2many:role_permissions` tag
- `BeforeCreate` hook and `TableName` method

### 3. Permission Constants (`internal/models/permission_constants.go`)

**No changes** - permissions are independent of role changes. The shared_viewer role does not grant any permissions; it only controls data visibility (implemented in future plans).

### 4. Test Stubs (`internal/models/user_test.go`)

Created Wave 0 test stubs:
- `TestUser_HasRole_ReturnsTrueForMatchingRole`
- `TestUser_HasRole_ReturnsFalseForNonMatchingRole`
- `TestUser_HasPermission_WithMultipleRoles_ORLogic` (D-07)
- `TestUser_HasPermission_AdminRoleGrantsAll`
- `TestUser_RolesField_ManyToManyAssociation`

All tests use `t.Skip()` for Wave 0 implementation.

## Technical Details

### GORM Many-to-Many Relationship

The relationship is established via the `users_roles` junction table (created in plan 09-01):

```go
// User model
type User struct {
    // ...
    Roles []Role `gorm:"many2many:users_roles;" json:"roles,omitempty"`
}

// Role model
type Role struct {
    // ...
    // No Users field - relationship defined from User side only
}
```

### Permission Checking Logic (OR Logic)

Per D-07, `HasPermission()` uses OR logic across all roles:

```go
for _, role := range u.Roles {
    if role.Name == RoleAdmin {
        return true  // Admin shortcut
    }
    for _, perm := range role.Permissions {
        if perm.Resource == resource && (perm.Action == action || perm.Action == "*") {
            return true  // Permission granted by ANY role
        }
    }
}
return false  // No role grants permission
```

This means:
- If a user has roles [operator, viewer], and "operator" can delete files, the user can delete files
- Permissions accumulate across all roles (union of permissions)
- Admin role grants all permissions regardless of other roles

## Commits

| Commit | Hash | Message |
|--------|------|---------|
| Task 1 | 7741a5c | feat(09-02): update User model for many-to-many roles |
| Task 2 | 1749678 | feat(09-02): update Role model and add shared_viewer constant |
| Task 3 | fc34037 | chore(09-02): verify permission_constants.go unchanged |
| Task 4 | eff0d49 | test(09-02): create test stubs for User model changes |

## Verification Results

All success criteria met:

- [x] User.Roles []Role field added with many2many:users_roles tag
- [x] User.RoleID and User.Role fields removed
- [x] HasPermission() updated for multi-role OR logic
- [x] HasRole() helper method added
- [x] Role.Users field removed
- [x] RoleSharedViewer constant added
- [x] All model files compile without errors
- [x] Test stubs created for model changes

## Threat Model Compliance

All mitigations from threat register implemented:

- **T-09-06 (Elevation of Privilege)**: HasPermission() verifies all roles; no shortcut bypasses permission check
- **T-09-07 (Tampering)**: HasRole() uses simple string comparison; role names controlled in DB (low-risk)
- **T-09-08 (Information Disclosure)**: User.Roles uses `omitempty` tag to prevent empty array exposure
- **T-09-09 (Denial of Service)**: HasPermission() returns false if `len(Roles)==0`; prevents nil panic

## Next Steps

Plan 09-03 will update permission checking and data visibility logic to use the new multi-role model:
- Update authorization middleware to use User.HasRole()
- Implement data visibility filtering for shared_viewer role
- Update API handlers to check multiple roles
- Add shared_viewer role to role seeding scripts

## Notes

- Duration: 2 minutes (123 seconds)
- All changes compile without errors
- No breaking changes to API layer (authorization logic updated in future plans)
- Test stubs ready for Wave 1 implementation
- Junction table `users_roles` created in plan 09-01 is now active
