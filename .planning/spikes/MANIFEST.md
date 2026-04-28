# Spike Manifest

## Idea
Windows AD域控认证功能可行性研究 - 评估在现有Go项目中集成Windows Active Directory域控认证的可行性，验证技术方案并提供实现建议。

## Requirements

基于Spike研究发现的实现要求：

**技术要求:**
- 使用 `github.com/go-ldap/ldap/v3` 库进行AD认证
- **同时支持LDAP端口389和LDAPS端口636** (兼容内部环境)
- 支持StartTLS升级加密连接
- 使用策略模式实现认证方式切换(local/ad/hybrid)
- AD用户需要在本地创建映射记录用于权限管理

**安全要求:**
- **密码传输安全:** 前端使用SM4加密密码，后端解密后再进行LDAP认证（与本地认证一致）
- **端口选择:**
  - **生产/公网环境:** 强制使用LDAPS(端口636)或StartTLS
  - **内部/隔离网络:** 可使用LDAP(端口389)，需在配置中明确确认风险
- **TLS配置:** 启用TLS时强制TLS 1.2或更高版本
- **配置安全:** 管理员密码应从环境变量读取
- **配置验证:** 切换认证模式前必须验证AD配置
- **风险提示:** 使用389端口时需显示安全警告并记录确认

**功能要求:**
- 支持三种认证模式：local(本地)、ad(域控)、hybrid(混合)
- AD用户认证成功后自动创建本地用户记录
- 配置热加载，无需重启服务
- 提供AD配置连通性测试API
- 所有认证模式切换记录审计日志

**用户体验要求:**
- 启用AD认证前显示安全提示和确认对话框
- 切换模式时验证配置并显示清晰的错误信息
- AD用户密码修改提示联系域管理员
- 管理界面显示用户认证来源

## Spikes

| # | Name | Type | Validates | Verdict | Tags |
|---|------|------|-----------|---------|------|
| 001 | go-ldap-ad-auth | standard | Given Go LDAP库，当连接Windows AD服务器并验证用户凭据时，则能够成功认证并获取用户属性 | ✅ VALIDATED | ldap, ad, authentication, golang |
| 002 | ldaps-security | standard | Given AD服务器配置，当使用LDAPS/StartTLS连接时，则密码传输经过加密且安全 | ✅ VALIDATED | ldap, ldaps, security, tls, encryption |
| 003 | auth-switch-architecture | standard | Given本地认证和AD认证两种方式，当切换认证模式时，系统能够正确路由认证请求 | ✅ VALIDATED | architecture, authentication, strategy-pattern, local, ad |
| 004 | ad-user-mapping | standard | Given AD用户认证成功，当查找本地同名用户时，能够正确映射并分配权限 | ✅ VALIDATED | ldap, user-mapping, permissions, synchronization |
| 005 | ad-config-validation | standard | Given AD配置信息，当系统启动或切换认证模式时，能够验证配置的完整性和连通性 | ✅ VALIDATED | configuration, validation, health-check, ad |
