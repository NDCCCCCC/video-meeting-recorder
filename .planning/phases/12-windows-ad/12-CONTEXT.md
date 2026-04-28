# Phase 12: Windows AD域控认证 - Context

**Gathered:** 2026-04-28
**Status:** Ready for planning

<domain>
## Phase Boundary

集成Windows Active Directory域控认证功能，支持LDAP(389)和LDAPS(636)双端口，实现local/ad两种认证模式切换。

**核心功能:**
- AD用户认证（使用go-ldap/v3库）
- 认证模式切换（local/ad，系统级配置）
- 前端SM4密码加密（与本地认证一致）
- AD配置验证和连通性测试
- 配置热加载，无需重启服务

**不属于本阶段:**
- AD组→角色映射（未来扩展）
- 定期用户状态同步（仅登录时同步）
- AD用户单独管理界面（统一用户列表）

</domain>

<decisions>
## Implementation Decisions

### 认证模式设计
- **D-01:** 系统仅支持**local**和**ad**两种认证模式，移除hybrid模式
- **D-02:** 系统默认使用**local模式**（最安全，不与AD交互）
- **D-03:** 认证模式是系统级配置，切换后所有用户使用该模式
- **D-04:** **AD模式不降级**：AD认证失败（账号不存在或密码错误）直接返回错误提示，不尝试本地认证
- **D-05:** **local模式**：仅使用本地账号密码进行认证

### 账号管理策略
- **D-06:** 所有账号统一管理，UI不区分来源显示
- **D-07:** 所有账号都需要设置本地密码（AD用户的本地密码可以是系统生成的随机密码）
- **D-08:** 不需要auth_source字段，完全透明管理

### 首次启用AD认证
- **D-09:** 使用**简单表单式**引导流程（配置页面 + 测试连接按钮）
- **D-10:** 配置字段包括：AD服务器地址、端口、BindDN、密码、BaseDN、TLS选项
- **D-11:** 提供测试连接按钮，调用AD连通性测试API

### 安全提示设计
- **D-12:** 使用LDAP 389端口时，在配置字段旁显示**内联警告图标**（⚠️）
- **D-13:** 风险确认为**被动记录**：在审计日志中记录警告已展示，不需要用户显式确认
- **D-14:** 警告内容：说明389端口明文传输风险，建议使用LDAPS 636端口

### 配置验证
- **D-15:** 配置变更时**自动验证**AD连通性
- **D-16:** 验证失败时**阻止保存**并显示具体错误原因
- **D-17:** 切换到AD模式时必须先验证AD配置成功

### 错误处理
- **D-18:** AD连接失败时显示**友好提示**："无法连接到AD服务器，请检查网络和配置"
- **D-19:** 详细错误信息记录到后端日志（LDAP错误码、堆栈等）
- **D-20:** AD认证失败时明确提示原因：账号不存在 vs 密码错误 vs 服务器连接失败

### 数据库变更（来自Spike验证）
- **D-21:** 扩展users表支持AD属性，但不包含auth_source字段（基于D-08）
- **D-22:** 需要的字段：ad_username, ad_dn, ad_guid, last_ad_login, ad_department, ad_upn
- **D-23:** 所有AD字段可为NULL（本地用户这些字段为空）

### Claude's Discretion
- 具体的错误提示文案（中文友好即可）
- 配置页面的UI布局和样式
- 测试连接API的响应格式
- 审计日志的具体字段和格式

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Spike Findings (已验证的技术方案)
- `.claude/skills/spike-findings-record-v2/SKILL.md` — Spike技能包总览
- `.claude/skills/spike-findings-record-v2/references/ad-auth-core.md` — AD认证核心功能（go-ldap/v3使用、端口配置、安全要求）
- `.claude/skills/spike-findings-record-v2/references/ad-auth-architecture.md` — 认证切换架构（策略模式、配置结构）
- `.claude/skills/spike-findings-record-v2/references/ad-user-management.md` — AD用户映射逻辑
- `.claude/skills/spike-findings-record-v2/references/ad-config-validation.md` — AD配置验证设计

### Spike源文件（完整参考）
- `.planning/spikes/001-go-ldap-ad-auth/` — AD认证验证代码
- `.planning/spikes/002-ldaps-security/` — 安全方案研究
- `.planning/spikes/003-auth-switch-architecture/` — 架构设计
- `.planning/spikes/004-ad-user-mapping/` — 用户映射方案
- `.planning/spikes/005-ad-config-validation/` — 配置验证设计

### 项目现有认证系统
- `internal/auth/service.go` — 现有认证服务（需要扩展支持AD）
- `internal/models/user.go` — User模型定义
- `internal/migrations/` — 数据库迁移文件

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/auth/service.go` — 现有AuthService，需要扩展支持AD认证器
- SM4加密工具 — 前端已有SM4密码加密实现，AD认证同样使用
- 配置管理系统 — 现有配置热加载机制
- 审计日志系统 — 现有审计日志接口，记录认证模式切换

### Established Patterns
- 策略模式 — 用于认证器切换（LocalAuthenticator vs ADAuthenticator）
- 配置热加载 — 通过配置文件变更监听实现
- 服务接口抽象 — Authenticator接口定义统一的认证行为

### Integration Points
- 登录API — `/api/auth/login` 需要支持AD认证路径
- 配置API — 需要新增AD配置管理端点
- 用户管理 — 需要支持AD字段的创建和更新
- 前端登录页 — 已经支持SM4加密，无需修改

</code_context>

<specifics>
## Specific Ideas

- 测试连接API示例：
  ```bash
  POST /api/admin/auth/ad/test-connection
  Body: { "server": "ad.example.com:636", "bind_dn": "...", "password": "...", "base_dn": "...", "use_tls": true }
  Response: { "success": true, "message": "连接成功", "user_found": true }
  ```

- 389端口警告文案： "⚠️ 使用LDAP 389端口时密码将以明文传输，存在安全风险。建议在生产环境使用LDAPS 636端口。"

- AD认证失败提示：
  - 账号不存在："域控账号不存在，请联系管理员确认"
  - 密码错误："域控密码错误，请重试"
  - 连接失败："无法连接到域控服务器，请检查网络和配置"

</specifics>

<deferred>
## Deferred Ideas

- AD组→角色映射（未来扩展功能）
- 定期自动同步AD用户状态（仅登录时同步即可）
- AD用户单独管理界面（统一用户列表即可）
- 支持多个AD服务器配置
- AD用户密码修改（提示联系域管理员）

</deferred>

---

*Phase: 12-windows-ad*
*Context gathered: 2026-04-28*
