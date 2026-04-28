---
status: root_cause_found
trigger: Phase 09 多角色迁移后出现一系列问题，包括数据库迁移失败、AD 用户无 sidebar 等
created: 2026-04-28T14:00:00+08:00
updated: 2026-04-28T15:30:00+08:00
---

# Multi-Role Migration Issues

## Symptoms

### Expected Behavior
- Phase 09 实现多对多用户角色关系后，系统应正常运行
- AD 用户登录后应能看到基于其角色的 sidebar 菜单
- GORM AutoMigrate 应能正常处理 schema 变更

### Actual Behavior
1. **GORM AutoMigrate 失败**: `constraint failed: NOT NULL constraint failed: users__temp.password_hash`
2. **AD 用户登录成功但无 sidebar**: 域控账号能登录但看不到任何菜单
3. **新建用户失败**: `constraint failed: NOT NULL constraint failed: users.role_id`

### Error Messages
```
constraint failed: NOT NULL constraint failed: users__temp.password_hash (1299)
constraint failed: NOT NULL constraint failed: users.role_id (1299)
```

### Timeline
- Phase 09 实现了 users_roles 多对多关系表
- Migration 006 迁移现有用户的 role_id 到 users_roles
- 之后出现一系列 bug 并进行了多次修复
- Phase 12 添加 AD 域控功能后问题继续暴露

### Reproduction
1. 启动服务 → GORM AutoMigrate 失败
2. 创建新用户 → role_id 约束失败
3. AD 用户登录 → 登录成功但无 sidebar

## Current Focus

**hypothesis**: 多角色迁移后，users 表的 role_id 列处理不一致，导致 GORM AutoMigrate、用户创建和 AD 认证之间产生冲突

**next_action**: ROOT CAUSE FOUND - Migration 012 保留了 password_hash 的 NOT NULL 约束，但 AD 用户不需要密码

## Evidence

- timestamp: 2026-04-28T15:15:00+08:00
  source: "internal/migrations/012_drop_legacy_role_id.go:57"
  finding: |
    Migration 012 创建新表时保留 `password_hash VARCHAR(255) NOT NULL` 约束
    但 AD 用户通过域控认证，不需要本地密码，导致约束冲突

- timestamp: 2026-04-28T15:18:00+08:00
  source: "internal/models/user.go:15"
  finding: |
    User 模型定义 `PasswordHash stringgorm:"type:varchar(255);not null"`
    与 AD 认证流程冲突 - AD 用户在 findOrCreateLocalUser 中
    虽然设置了随机密码，但 GORM AutoMigrate 尝试创建临时表时
    会因 NOT NULL 约束失败

- timestamp: 2026-04-28T15:20:00+08:00
  source: "internal/auth/ad_auth.go:190-206"
  finding: |
    AD 用户创建逻辑正确：生成随机密码并调用 user.SetPassword()
    但问题在于 Migration 012 的 schema 定义与实际需求不匹配

- timestamp: 2026-04-28T15:22:00+08:00
  source: "internal/auth/ad_auth.go:276-311"
  finding: |
    toUserDTO 方法正确处理了多角色权限聚合（OR logic）
    AD 用户无 sidebar 的根本原因是创建时可能失败
    或者角色关联失败，导致 user.Roles 为空

## Eliminated

- timestamp: 2026-04-28T15:10:00+08:00
  hypothesis: "Migration 006 未正确迁移 role_id"
  evidence: "Migration 006 正确创建了 users_roles 表并迁移数据"
  reason: "Migration 006 逻辑正确，问题不在这里"

- timestamp: 2026-04-28T15:12:00+08:00
  hypothesis: "User 模型未正确配置 many2many 关系"
  evidence: "User 模型正确定义了 Roles []Rolegorm:'many2many:users_roles'"
  reason: "GORM 关系定义正确"

## Resolution

**root_cause**: |
  Migration 012 在重建 users 表时保留了 password_hash 的 NOT NULL 约束，
  但这与 AD 认证流程冲突。AD 用户通过域控认证，理论上不需要本地密码字段。
  虽然 AD 用户创建时会设置随机密码，但 GORM AutoMigrate 在 schema 变更时
  会尝试创建临时表，此时 NOT NULL 约束会导致失败。

**specialist_hint**: go
**technical_context**: |
  涉及 GORM AutoMigrate、SQLite 约束处理、AD 认证集成
  需要理解 GORM 的临时表创建机制和 SQLite 的约束重建逻辑

**proposed_fix**: |
  1. 修改 Migration 012：将 password_hash 改为 nullable
  2. 更新 User 模型：PasswordHash 改为 *string (指针类型)
  3. 添加数据迁移：为现有的 NULL password_hash 设置随机密码
  4. 确保 AD 用户创建逻辑健壮性

**fix_status**: pending_approval
