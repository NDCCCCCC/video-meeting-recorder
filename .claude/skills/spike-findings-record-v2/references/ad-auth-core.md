# AD认证核心功能

## Requirements

以下要求来自Spike研究，是实施中不可协商的设计决策：

**技术要求:**
- 使用 `github.com/go-ldap/ldap/v3` 库进行AD认证
- **同时支持LDAP端口389和LDAPS端口636** (兼容内部环境)
- 支持StartTLS升级加密连接
- 前端必须使用SM4加密密码，后端解密后再进行LDAP认证

**安全要求:**
- **前端SM4加密:** 所有场景(本地和AD)必须使用SM4加密
- **端口选择:**
  - **生产/公网环境:** 强制使用LDAPS(端口636)或StartTLS
  - **内部/隔离网络:** 可使用LDAP(端口389)，需在配置中明确确认风险
- **TLS配置:** 启用TLS时强制TLS 1.2或更高版本
- **风险提示:** 使用389端口时需显示安全警告并记录确认

## How to Build It

### 1. 安装依赖

```bash
go get github.com/go-ldap/ldap/v3@latest
```

### 2. 配置结构

```go
type ADConfig struct {
    Server   string `mapstructure:"server" json:"server"`   // ad.example.com:389 或 :636
    BindDN   string `mapstructure:"bind_dn" json:"bind_dn"` // cn=admin,cn=users,dc=example,dc=com
    Password string `mapstructure:"password" json:"password"` // 从环境变量读取
    BaseDN   string `mapstructure:"base_dn" json:"base_dn"` // dc=example,dc=com
    UseTLS   bool   `mapstructure:"use_tls" json:"use_tls"` // false=389, true=636
    
    // 连接池配置
    PoolSize int `mapstructure:"pool_size" json:"pool_size"`
}
```

### 3. 连接模式

```go
// LDAPS模式 (端口636) - 生产环境推荐
l, err := ldap.DialTLS("tcp", "ad.example.com:636", &tls.Config{
    ServerName:         "ad.example.com",
    InsecureSkipVerify: false, // 生产环境必须false
    MinVersion:         tls.VersionTLS12,
})

// LDAP模式 (端口389) - 内部隔离网络
l, err := ldap.Dial("tcp", "ad.example.com:389")
if err == nil {
    err = l.StartTLS(&tls.Config{...}) // 可选：升级为TLS
}
```

### 4. 用户认证流程

```go
// 1. 连接AD服务器
conn, err := connectAD(config)

// 2. 管理员绑定
err = conn.Bind(config.BindDN, config.Password)

// 3. 搜索用户DN
userDN, err := findUserDN(conn, username)

// 4. 使用用户凭据绑定进行认证
err = conn.Bind(userDN, password)

// 5. 获取用户属性
user, err := getUserAttributes(conn, userDN)
```

### 5. 用户搜索

```go
searchRequest := ldap.NewSearchRequest(
    config.BaseDN,
    ldap.ScopeWholeSubtree,
    ldap.NeverDerefAliases,
    0, 0, false,
    fmt.Sprintf("(&(objectClass=user)(sAMAccountName=%s))", ldap.EscapeFilter(username)),
    []string{"dn", "sAMAccountName", "mail", "displayName"},
    nil,
)
```

### 6. 安全配置示例

**生产环境:**
```yaml
auth:
  ad:
    server: "ad.example.com:636"
    use_tls: true
```

**内部网络:**
```yaml
auth:
  ad:
    server: "ad.internal:389"
    use_tls: false
    # 需要前端显示安全警告
```

## What to Avoid

- ❌ 使用已废弃的 `github.com/go-ldap/ldap` (v1/v2)
- ❌ 生产环境使用 `InsecureSkipVerify=true`
- ❌ 生产环境使用LDAP(389)端口
- ⚠️ 内部网络使用LDAP(389)需配置确认和网络隔离
- ❌ 搜索用户名时忘记使用 `ldap.EscapeFilter()` (存在LDAP注入风险)
- ❌ 在日志中记录明文密码或BindDN

## Constraints

**库限制:**
- go-ldap/v3不支持连接池自动管理，需要手动实现
- Windows Server 2019及更早版本LDAP Signing默认禁用

**端口限制:**
- LDAPS: 636 (推荐生产环境)
- LDAP: 389 (仅内部隔离网络)
- StartTLS: 389 (需要升级为TLS)

**AD限制:**
- 搜索需要有效的BaseDN
- 需要管理员账户才能搜索用户
- userAccountControl字段用于判断账户状态

## Origin

Synthesized from spikes: 001, 002
Source files available in: sources/001-go-ldap-ad-auth/, sources/002-ldaps-security/
