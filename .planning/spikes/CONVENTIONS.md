# Spike Conventions

Patterns and stack choices established across spike sessions. New spikes follow these unless the question requires otherwise.

## Stack

### Backend
- **Language:** Go 1.25+
- **Web Framework:** Gin (github.com/gin-gonic/gin)
- **ORM:** GORM (gorm.io/gorm)
- **Database:** SQLite (modernc.org/sqlite)
- **LDAP Library:** github.com/go-ldap/ldap/v3

### AD/LDAP
- **端口支持:** 同时支持LDAP(389)和LDAPS(636)
- **生产环境:** 优先使用LDAPS (port 636)
- **内部网络:** 可使用LDAP (port 389)，需配置确认
- **TLS Version:** 启用TLS时最低TLS 1.2，推荐TLS 1.3
- **前端加密:** 必须使用SM4加密密码传输
- **Authentication:** SASL with Kerberos preferred, NTLM acceptable

## Structure

```
.planning/spikes/
├── MANIFEST.md              # Spike索引和总体要求
├── CONVENTIONS.md           # 此文件，记录约定的模式和栈
├── 001-go-ldap-ad-auth/
│   ├── README.md            # Spike报告(包含Research和Results)
│   ├── main.go              # 验证代码
│   └── go.mod               # Go模块定义
├── 002-ldaps-security/
│   └── README.md            # 纔研究型Spike
├── 003-auth-switch-architecture/
│   └── README.md            # 架构设计Spike
├── 004-ad-user-mapping/
│   └── README.md            # 数据映射设计Spike
└── 005-ad-config-validation/
    └── README.md            # 配置验证设计Spike
```

## Patterns

### README Frontmatter
所有Spike README必须包含YAML frontmatter：

```yaml
---
spike: NNN
name: descriptive-name
type: standard
validates: "Given [precondition], when [action], then [expected outcome]"
verdict: PENDING
related: []
tags: [tag1, tag2]
---
```

### Spike验证状态
- `PENDING` - 研究中
- `VALIDATED` - 验证通过
- `INVALIDATED` - 验证失败
- `PARTIAL` - 部分验证

### 错误处理模式
LDAP错误应分类处理：

```go
if ldapErr, ok := err.(*ldap.Error); ok {
    switch ldapErr.ResultCode {
    case ldap.LDAPResultInvalidCredentials:
        return "用户名或密码错误"
    case ldap.LDAPResultNoSuchObject:
        return "对象不存在"
    // ...其他错误码
    }
}
```

### 配置验证模式
分层验证：格式→网络→认证→功能，每层失败立即返回。

## Tools & Libraries

### 推荐使用的包

| 包名 | 版本 | 用途 |
|------|------|------|
| github.com/go-ldap/ldap/v3 | v3.4.6 | LDAP客户端 |
| github.com/gin-gonic/gin | v1.11.0 | Web框架 |
| gorm.io/gorm | v1.30.0 | ORM |
| go.uber.org/zap | v1.27.0 | 日志 |
| golang.org/x/crypto | v0.49.0 | 密码哈希(bcrypt) |

### 避免使用

- ❌ github.com/go-ldap/ldap (v1/v2) - 已废弃
- ❌ 生产环境的明文LDAP(389) - 安全风险
- ❌ InsecureSkipVerify=true (生产环境) - 安全风险
- ⚠️ 内部网络使用LDAP(389) - 需配置确认和网络隔离

## 安全约定

1. **前端密码加密:** 所有场景(本地和AD)必须使用SM4加密
2. **传输层加密:** HTTPS + LDAPS/LDAP(根据环境)
3. **密码存储:** 本地用户使用bcrypt哈希，AD用户不存储密码
4. **端口选择:**
   - 生产环境: 必须使用LDAPS(636)
   - 内部网络: 可使用LDAP(389)，需配置确认
5. **配置管理:** 敏感信息(如AD密码)从环境变量读取
6. **证书验证:** 生产环境必须验证TLS证书
7. **审计日志:** 所有认证模式切换和AD认证失败必须记录
8. **风险提示:** 使用389端口时需显示安全警告

## 文档约定

### Spike README章节结构
1. What This Validates - 验证目标(Given/When/Then)
2. Research - 研究发现和对比表格
3. How to Run - 运行说明(如适用)
4. What to Expect - 预期输出
5. Investigation Trail - 研究过程记录
6. Results - 验证结论和建议

### 代码注释
- 公开函数必须有注释
- 复杂逻辑需要行内注释
- 安全敏感代码必须有安全警告注释
