# Phase 9: Multi-Role Permissions & Shared Viewer - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-21
**Phase:** 9 - Multi-Role Permissions & Shared Viewer
**Areas discussed:** 共享查看者角色定义, 多角色存储方式, 数据所有权隔离, 共享查看者分配, 权限检查逻辑, 角色命名

---

## 共享查看者角色定义

| Option | Description | Selected |
|--------|-------------|----------|
| 所有数据只读访问 | 可以查看所有用户的录制任务、视频文件、PPT结果，但不能编辑或删除 | |
| 只读 + 自己的编辑权限 | 可以查看所有数据，但只能编辑/删除自己创建的内容 | |
| 仅查看，不能下载 | 完全只读，甚至不能下载文件 | |

**User's choice:** (Free text input)
只控制能看到的范围，读写等等只要能看到就有权限，所以才需要将用户-角色修改为多对多关系，共享者让用户能够看都所有数据，其他角色控制具体能够执行什么操作

**Notes:** 用户澄清了核心设计意图——共享查看者角色仅控制数据可见性（scope of visibility），操作权限仍由其他角色决定。这是一个"可见性提升器"而非"操作权限授予者"角色。

---

## 多角色存储方式

| Option | Description | Selected |
|--------|-------------|----------|
| 新用户-角色关联表 | 从单角色字段迁移到多对多关系表（users_roles），更灵活 | ✓ |
| 主角色 + 附加角色字段 | 保留 RoleID 主角色，添加 SecondaryRoleIDs JSON 字段存储附加角色 | |
| 将 RoleID 改为数组 | 直接将 RoleID 改为 JSON 数组字段，简单但不规范 | |

**User's choice:** 新用户-角色关联表

**Notes:** 标准的多对多关系设计，与现有 Role ↔ Permission 模式一致。

---

## 数据所有权隔离

| Option | Description | Selected |
|--------|-------------|----------|
| 基于所有者的数据隔离 | 在 VideoFile、RecordingTask 等表添加 CreatedBy 字段，用户只能看到自己的数据（除非有共享查看者角色） | ✓ |
| 无数据隔离，仅权限控制 | 所有有查看权限的用户可以看到所有数据，仅通过权限控制，不按所有者隔离 | |
| 部分数据隔离 | 仅对特定资源（如 PPT 结果）进行所有者隔离，其他资源共享 | |

**User's choice:** (Free text input)
当前不是已经有数据隔离了吗？不是选项1吗？

**Notes:** 用户指出数据所有权隔离已存在（VideoFile.CreatedBy 和 VideoRecordingTask.CreatedBy），此阶段只需添加共享查看者的"可见性绕过"逻辑。

---

## 共享查看者分配方式

| Option | Description | Selected |
|--------|-------------|----------|
| 仅管理员可分配 | 仅系统管理员可以将共享查看者角色分配给用户 | ✓ |
| 用户申请+管理员审批 | 用户可以自主申请共享查看者角色，需要管理员审批 | |
| 管理员分配+用户申请 | 管理员可以直接分配，用户也可以申请，双渠道 | |

**User's choice:** 仅管理员可分配

**Notes:** 敏感角色，仅限管理员分配。审计日志应记录此类操作。

---

## 权限检查逻辑

| Option | Description | Selected |
|--------|-------------|----------|
| 任一角色有权限即可（OR） | 用户只要有任一角色有权限即可。共享查看者角色有 files:view 就能看所有文件，不需要其他角色也有 files:view | |
| 所有角色都需要权限（AND） | 用户的所有角色权限取交集。必须所有角色都有该权限才能执行 | |
| 共享查看者只管可见 | 共享查看者角色特殊处理：只对数据可见性生效，操作权限由其他角色决定 | ✓ |

**User's choice:** 共享查看者只管可见

**Notes:** 确认共享查看者角色的特殊定位——它不是普通权限角色，而是一个"可见性开关"。操作权限检查仍按 OR 逻辑遍历其他角色。

---

## 角色命名

| Option | Description | Selected |
|--------|-------------|----------|
| shared_viewer | 英文名称，简洁 | |
| 共享查看者 | 中文名称，符合现有角色命名风格 | |
| 英文名+中文显示名 | shared_viewer，显示名称为"共享查看者" - 存储英文，显示中文 | ✓ |

**User's choice:** 英文名+中文显示名

**Notes:** 存储名：shared_viewer；显示名：共享查看者。符合国际化最佳实践。

---

## Claude's Discretion

None — 所有决策均由用户明确指定。

---

## Deferred Ideas

- 用户可以"申请"共享查看者角色的审批流程（未来增强）
- 更细粒度的数据共享（如按部门、项目共享，而非全系统）
- 临时共享查看者权限（有时限的访问提升）
