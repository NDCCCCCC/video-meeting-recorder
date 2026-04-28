---
name: spike-findings-record-v2
description: Implementation blueprint from spike experiments. Requirements, proven patterns, and verified knowledge for Windows AD domain authentication integration. Auto-loaded during implementation work.
---

<context>
## Project: record_v2

Windows AD域控认证功能可行性研究 - 评估在现有Go项目中集成Windows Active Directory域控认证的可行性，验证技术方案并提供实现建议。

Spike sessions wrapped: 2025-04-28
</context>

<requirements>
## Requirements

基于Spike研究发现的实现要求：

**技术要求:**
- 使用 `github.com/go-ldap/ldap/v3` 库进行AD认证
- **同时支持LDAP端口389和LDAPS端口636** (兼容内部环境)
- 支持StartTLS升级加密连接
- 使用策略模式实现认证方式切换(local/ad/hybrid)
- AD用户需要在本地创建映射记录用于权限管理

**安全要求:**
- **前端SM4加密:** 所有场景(本地和AD)必须使用SM4加密
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

</requirements>

<findings_index>
## Feature Areas

| Area | Reference | Key Finding |
|------|-----------|-------------|
| AD认证核心 | references/ad-auth-core.md | go-ldap/v3库完整支持AD认证，支持389/636双端口 |
| AD认证架构 | references/ad-auth-architecture.md | 策略模式实现认证切换，支持local/ad/hybrid三种模式 |
| AD用户管理 | references/ad-user-management.md | 同名映射自动创建本地用户，分层验证配置 |

## Source Files

原始Spike源文件保存在 `sources/` 目录中供完整参考：
- sources/001-go-ldap-ad-auth/ - AD认证验证代码
- sources/002-ldaps-security/ - 安全方案研究
- sources/003-auth-switch-architecture/ - 架构设计
- sources/004-ad-user-mapping/ - 用户映射方案
- sources/005-ad-config-validation/ - 配置验证设计

</findings_index>

<metadata>
## Processed Spikes

- 001-go-ldap-ad-auth
- 002-ldaps-security
- 003-auth-switch-architecture
- 004-ad-user-mapping
- 005-ad-config-validation
</metadata>
