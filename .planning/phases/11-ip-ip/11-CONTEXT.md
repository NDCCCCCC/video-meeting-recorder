# Phase 11: IP地址登录限制 - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 11 为用户和角色添加IP地址组功能，限制只有允许的IP地址才能登录系统。这是一种基于网络位置的访问控制机制，用于增强系统安全性。

系统支持用户级别和角色级别的IP限制，最终允许的IP地址通过OR逻辑合并。如果用户和角色都没有设置IP限制，则对IP地址无限制。

</domain>

<decisions>
## Implementation Decisions

### IP限制粒度
- **D-01:** 采用混合模式：支持用户级别和角色级别IP限制
- **D-02:** 用户最终允许的IP地址 = 用户IP组 ∪ 所有角色的IP组（OR逻辑）
- **D-03:** IP限制合并规则：
  - 用户有IP限制 + 角色无IP限制 → 使用用户的IP限制
  - 用户无IP限制 + 角色有IP限制 → 使用角色的IP组
  - 用户有IP限制 + 角色有IP限制 → 使用OR逻辑合并（User_IPs ∪ Role_IPs）
  - 用户无IP限制 + 角色无IP限制 → 无IP限制

### IP地址组存储方式
- **D-04:** 采用内嵌字段方式：IP列表直接存储在用户和角色模型的字段中
- **D-05:** 不创建独立的IP地址组资源，简化管理

### 支持的IP地址格式
- **D-06:** 单个IPv4地址：如 `192.168.1.100`
- **D-07:** IPv4 CIDR范围：如 `192.168.1.0/24`
- **D-08:** IPv4地址段：如 `192.168.1.100-192.168.1.200`
- **D-09:** 不支持IPv6（当前仅支持IPv4）

### IP地址获取方式
- **D-10:** 使用Gin的 `c.ClientIP()` 方法获取客户端真实IP
- **D-11:** 现有代码已在使用此方法（`auth_handler.go` 第42行）
- **D-12:** 系统为直接部署场景（无反向代理）

### 登录失败处理
- **D-13:** 当用户IP不在允许列表时，明确告知用户"您的IP地址不在允许列表中"
- **D-14:** IP限制检查失败应记录审计日志（敏感操作）
- **D-15:** 所有用户（包括管理员角色）都必须遵守IP限制，无豁免

### IP限制检查时机
- **D-16:** IP限制检查在用户登录时进行（`auth_handler.go` 的 `Login()` 函数中）
- **D-17:** IP限制检查在密码验证通过后、Token生成前进行

### Claude's Discretion
- IP地址字段的数据结构（JSON数组存储、逗号分隔字符串或独立关联表）
- IP地址验证的具体实现（正则表达式、IP库）
- CIDR范围匹配算法
- IP段匹配算法
- IP地址段配置错误时的处理
- 前端IP地址管理UI的具体实现

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project requirements
- `.planning/PROJECT.md` — 技术栈（Go 1.24/Gin, React 19/Ant Design 6, SQLite/GORM）
- `.planning/REQUIREMENTS.md` — RBAC权限系统已验证，审计日志已实现
- `.planning/STATE.md` — 项目进度和决策历史

### Existing authentication system
- `internal/auth/service.go` — 认证服务，Login() 方法
- `internal/auth/sm4_token.go` — SM4-GCM Token 认证
- `internal/handlers/auth_handler.go` — 认证处理器，Login() 处理函数（第42行使用c.ClientIP()）
- `internal/middleware/auth.go` — 认证中间件，SM4Auth()

### Existing models
- `internal/models/user.go` — User 模型，多角色关系
- `internal/models/role.go` — Role 模型，预定义角色常量
- `internal/models/user_role.go` — UserRole 关联表（Phase 9 实现）

### Prior phases context
- `.planning/phases/09-multi-role-permissions/09-CONTEXT.md` — 多角色权限系统，用户可以有多个角色
- `.planning/phases/04-cloud-services/04-CONTEXT.md` — 审计日志系统

### Audit logging
- `internal/models/audit_log.go` — AuditLog 模型
- `internal/middleware/audit.go` — 审计中间件

No external specs — requirements are fully captured in decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Login Flow** (`internal/handlers/auth_handler.go`): Login() 函数已获取客户端IP（`c.ClientIP()`），可在此处添加IP检查
- **Auth Service** (`internal/auth/service.go`): 认证服务，可添加IP限制检查逻辑
- **User Model** (`internal/models/user.go`): 需要添加IP限制字段
- **Role Model** (`internal/models/role.go`): 需要添加IP限制字段
- **Audit Logging**: 审计日志系统已实现，IP限制检查失败需要记录

### Established Patterns
- **多对多关系**: User ↔ Role 使用 `gorm:"many2many:users_roles;"` 模式
- **JSON字段存储**: 使用 `gorm:"type:json"` 存储复杂数据结构
- **审计日志**: 敏感操作通过中间件自动记录
- **权限检查**: User.HasPermission() 遍历所有角色（OR逻辑）

### Integration Points
- **User Model**: 添加 `AllowedIPs` 字段（JSON格式，存储IP列表）
- **Role Model**: 添加 `AllowedIPs` 字段（JSON格式，存储IP列表）
- **Auth Service**: 添加 `CheckIPRestriction()` 方法，验证用户IP是否在允许列表中
- **Login Handler**: 在密码验证后、Token生成前调用IP检查
- **审计日志**: 记录IP限制检查失败事件（操作类型：ip_restriction_failed）

</code_context>

<specifics>
## Specific Ideas

- IP地址验证应使用Go标准库 `net` 包的 `ParseCIDR()` 和 `ParseIP()` 函数
- IP匹配检查应考虑性能：先检查单IP，再检查CIDR，最后检查IP段
- 前端IP输入应提供友好的UI：支持输入多个IP、CIDR和范围，用换行或逗号分隔
- IP限制配置错误（如无效IP格式）应在保存时验证并提示
- 考虑在用户登录失败时记录客户端IP，方便管理员排查问题

</specifics>

<deferred>
## Deferred Ideas

- IP地址访问历史记录（哪些用户从哪些IP登录过）
- 动态IP地址限制（基于时间段、地理位置）
- IP地址黑名单（明确禁止某些IP访问，与白名单相反）
- IPv6支持（未来可能需要）
- 临时IP访问令牌（为受信任用户生成临时访问链接）

---

*Phase: 11-ip-ip*
*Context gathered: 2026-04-27*
