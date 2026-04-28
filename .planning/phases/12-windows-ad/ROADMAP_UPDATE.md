### Phase 12: Windows AD域控认证 - 集成Windows Active Directory域控认证，支持LDAP/LDAPS双端口，实现local/ad两种认证模式切换

**Goal:** 集成Windows Active Directory域控认证，支持LDAP(389)和LDAPS(636)双端口，实现local/ad两种认证模式切换（移除hybrid模式），确保密码传输安全（前端SM4加密+LDAPS），AD用户自动映射到本地用户记录用于权限管理。

**Requirements:**
- **Spike验证:** 5个spike已验证通过（go-ldap-ad-auth, ldaps-security, auth-switch-architecture, ad-user-mapping, ad-config-validation）
- **技术要求:** go-ldap/v3库，策略模式认证切换，配置热加载，AD连通性测试API
- **安全要求:** 前端SM4密码加密，生产推荐LDAPS(636)，内网可用LDAP(389)需风险确认，TLS 1.2+，管理员密码环境变量
- **功能要求:** local/ad两种模式（移除hybrid），AD用户自动创建本地记录，审计日志，安全提示警告
- **数据库变更:** 扩展users表（ad_username, ad_dn, ad_guid, last_ad_login, ad_department, ad_upn） - 无auth_source字段（透明管理）

**Depends on:** Phase 11
**Plans:** 6 plans ready

**Spike Findings:** `.planning/spikes/` - 5个验证通过的spike，包含代码验证和架构设计
**Skill Reference:** `./.claude/skills/spike-findings-record-v2/` - Spike技能包包含实施蓝图、约束、注意事项

Plans:
- [ ] 12-00-PLAN.md — Wave 0: Test infrastructure and AD types (6 test files, 35 test stubs) — Wave 0
- [ ] 12-01-PLAN.md — Database migration, User model, Config extension (local/ad modes) — Wave 1
- [ ] 12-02-PLAN.md — Strategy pattern: Authenticator interface, Local/AD authenticators, refactored AuthService — Wave 2
- [ ] 12-03-PLAN.md — AD config validation (4-layer), admin API endpoints, test connection API — Wave 2
- [ ] 12-04-PLAN.md — Frontend config page with security warnings, API client, TypeScript types (with checkpoint) — Wave 3
- [ ] 12-05-PLAN.md — Testing documentation, administrator guide, troubleshooting, README update (with checkpoint) — Wave 4

**Wave Structure:**
- Wave 0 (sequential): 12-00 (test infrastructure + go-ldap/v3 dependency)
- Wave 1 (sequential): 12-01 (database migration + models + config foundation)
- Wave 2 (parallel): 12-02 (authenticators + strategy pattern), 12-03 (validation + admin APIs)
- Wave 3 (sequential): 12-04 (frontend config page with checkpoint)
- Wave 4 (sequential): 12-05 (testing + docs + verification checkpoint)

**Key Decisions:**
- D-01 to D-05: 认证模式设计（仅local/ad，无hybrid，AD不降级）
- D-06 to D-08: 账号管理策略（统一管理，无auth_source，透明）
- D-09 to D-11: 首次启用AD认证（表单式配置，连通性测试）
- D-12 to D-14: 安全提示设计（389端口内联警告，被动记录）
- D-15 to D-17: 配置验证（自动验证，失败阻止保存，切换前验证）
- D-18 to D-20: 错误处理（友好提示，详细后端日志，明确失败原因）
- D-21 to D-23: 数据库变更（AD字段扩展，无auth_source，全可空）
