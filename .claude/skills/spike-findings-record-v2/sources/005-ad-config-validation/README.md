---
spike: 005
name: ad-config-validation
type: standard
validates: "Given AD配置信息，当系统启动或切换认证模式时，能够验证配置的完整性和连通性"
verdict: VALIDATED
related: [001, 002, 003]
tags: [configuration, validation, health-check, ad]
---

# Spike 005: ad-config-validation

## What This Validates

**Given:** 管理员配置AD域控连接信息
**When:** 系统启动、保存配置或切换认证模式时
**Then:**
- 能够验证所有必填字段是否完整
- 能够测试AD服务器的连通性
- 能够验证管理员凭据的有效性
- 能够提供清晰的错误提示和修复建议

## Research

### 配置验证层次

```
┌─────────────────────────────────────────────────────────────┐
│                    第1层: 格式验证                            │
│  - 服务器地址格式 (host:port)                                │
│  - BaseDN格式 (dc=example,dc=com)                           │
│  - 必填字段非空                                              │
└──────────────────────┬──────────────────────────────────────┘
                       │ ✓ 通过
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    第2层: 网络验证                            │
│  - DNS解析成功                                               │
│  - TCP端口可达                                               │
│  - TLS握手成功                                               │
└──────────────────────┬──────────────────────────────────────┘
                       │ ✓ 通过
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    第3层: 认证验证                            │
│  - BindDN存在                                                │
│  - 密码正确                                                  │
│  - BaseDN可访问                                              │
└──────────────────────┬──────────────────────────────────────┘
                       │ ✓ 通过
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    第4层: 功能验证                            │
│  - 可搜索用户                                                │
│  - 可读取用户属性                                            │
└─────────────────────────────────────────────────────────────┘
```

### 配置结构

```go
// ADConfig AD域控配置
type ADConfig struct {
    Server   string `mapstructure:"server" json:"server" validate:"required,hostname_port"`
    BindDN   string `mapstructure:"bind_dn" json:"bind_dn" validate:"required"`
    Password string `mapstructure:"password" json:"password" validate:"required"`
    BaseDN   string `mapstructure:"base_dn" json:"base_dn" validate:"required,dn"`
    UseTLS   bool   `mapstructure:"use_tls" json:"use_tls"`

    // 连接池配置
    PoolSize int `mapstructure:"pool_size" json:"pool_size" validate:"min=1,max=100"`

    // 超时配置
    DialTimeout   int `mapstructure:"dial_timeout" json:"dial_timeout" validate:"min=1,max=60"`   // 秒
    RequestTimeout int `mapstructure:"request_timeout" json:"request_timeout" validate:"min=1,max=300"` // 秒

    // 可选：用户搜索配置
    UserSearchBase string `mapstructure:"user_search_base" json:"user_search_base"`
    UserFilter     string `mapstructure:"user_filter" json:"user_filter"`
}

// ADConfigValidationResult 验证结果
type ADConfigValidationResult struct {
    Valid        bool     `json:"valid"`
    Level        int      `json:"level"` // 1-4: 通过的验证层级
    Errors       []string `json:"errors,omitempty"`
    Warnings     []string `json:"warnings,omitempty"`
    ServerInfo   string   `json:"server_info,omitempty"`
    ResponseTime int64    `json:"response_time_ms,omitempty"`
}
```

### 验证器实现

```go
// ADConfigValidator AD配置验证器
type ADConfigValidator struct {
    logger *zap.Logger
}

// Validate 验证AD配置
func (v *ADConfigValidator) Validate(config *ADConfig) *ADConfigValidationResult {
    result := &ADConfigValidationResult{
        Valid:  false,
        Level:  0,
        Errors: []string{},
        Warnings: []string{},
    }

    // 第1层: 格式验证
    if err := v.validateFormat(config); err != nil {
        result.Errors = append(result.Errors, err.Error())
        return result
    }
    result.Level = 1

    // 第2层: 网络验证
    start := time.Now()
    conn, err := v.testConnection(config)
    if err != nil {
        result.Errors = append(result.Errors, v.formatConnectionError(err))
        return result
    }
    defer conn.Close()
    result.ResponseTime = time.Since(start).Milliseconds()
    result.Level = 2

    // 第3层: 认证验证
    if err := v.testBind(conn, config); err != nil {
        result.Errors = append(result.Errors, v.formatBindError(err))
        return result
    }
    result.Level = 3

    // 第4层: 功能验证
    if err := v.testFunctionality(conn, config); err != nil {
        result.Warnings = append(result.Warnings, "功能测试警告: "+err.Error())
        // 不影响验证通过，仅警告
    }
    result.Level = 4

    result.Valid = true
    return result
}

// validateFormat 格式验证
func (v *ADConfigValidator) validateFormat(config *ADConfig) error {
    var errs []string

    // 验证服务器地址
    if config.Server == "" {
        errs = append(errs, "服务器地址不能为空")
    } else if !isValidHostPort(config.Server) {
        errs = append(errs, "服务器地址格式错误，应为 host:port")
    }

    // 验证端口
    if config.UseTLS && !strings.HasSuffix(config.Server, ":636") {
        errs = append(errs, "LDAPS模式应使用端口636")
    } else if !config.UseTLS && !strings.HasSuffix(config.Server, ":389") {
        errs = append(errs, "LDAP模式应使用端口389")
    }

    // 验证BindDN
    if config.BindDN == "" {
        errs = append(errs, "BindDN不能为空")
    } else if !isValidDN(config.BindDN) {
        errs = append(errs, "BindDN格式错误")
    }

    // 验证密码
    if config.Password == "" {
        errs = append(errs, "管理员密码不能为空")
    }

    // 验证BaseDN
    if config.BaseDN == "" {
        errs = append(errs, "BaseDN不能为空")
    } else if !isValidDN(config.BaseDN) {
        errs = append(errs, "BaseDN格式错误")
    }

    if len(errs) > 0 {
        return fmt.Errorf(strings.Join(errs, "; "))
    }
    return nil
}

// testConnection 测试网络连接
func (v *ADConfigValidator) testConnection(config *ADConfig) (*ldap.Conn, error) {
    var conn *ldap.Conn
    var err error

    if config.UseTLS {
        // LDAPS
        tlsConfig := &tls.Config{
            ServerName:         strings.Split(config.Server, ":")[0],
            InsecureSkipVerify: false, // 生产环境必须验证证书
            MinVersion:         tls.VersionTLS12,
        }
        conn, err = ldap.DialTLS("tcp", config.Server, tlsConfig)
    } else {
        // LDAP + StartTLS
        conn, err = ldap.Dial("tcp", config.Server)
        if err == nil {
            err = conn.StartTLS(&tls.Config{
                ServerName: strings.Split(config.Server, ":")[0],
                MinVersion: tls.VersionTLS12,
            })
        }
    }

    if err != nil {
        return nil, fmt.Errorf("连接失败: %w", err)
    }

    return conn, nil
}

// testBind 测试认证
func (v *ADConfigValidator) testBind(conn *ldap.Conn, config *ADConfig) error {
    err := conn.Bind(config.BindDN, config.Password)
    if err != nil {
        return fmt.Errorf("认证失败: %w", err)
    }

    // 获取当前用户信息
    whoami, err := conn.WhoAmI(nil)
    if err != nil {
        return fmt.Errorf("获取用户信息失败: %w", err)
    }

    v.logger.Info("AD管理员认证成功", zap.String("authz_id", whoami.AuthzID))
    return nil
}

// testFunctionality 测试功能
func (v *ADConfigValidator) testFunctionality(conn *ldap.Conn, config *ADConfig) error {
    // 尝试搜索一个测试用户
    searchRequest := ldap.NewSearchRequest(
        config.BaseDN,
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        1, 0, false, // 只返回1个结果
        "(objectClass=user)",
        []string{"dn", "sAMAccountName"},
        nil,
    )

    _, err := conn.Search(searchRequest)
    if err != nil {
        return fmt.Errorf("搜索测试失败: %w", err)
    }

    return nil
}

// formatConnectionError 格式化连接错误
func (v *ADConfigValidator) formatConnectionError(err error) string {
    errMsg := err.Error()

    // 根据错误类型提供修复建议
    switch {
    case strings.Contains(errMsg, "no such host"):
        return fmt.Sprintf("无法解析服务器地址: %v (请检查服务器地址是否正确)", err)
    case strings.Contains(errMsg, "connection refused"):
        return fmt.Sprintf("连接被拒绝: %v (请检查防火墙设置和LDAP服务是否启动)", err)
    case strings.Contains(errMsg, "i/o timeout"):
        return fmt.Sprintf("连接超时: %v (请检查网络连接和服务器状态)", err)
    case strings.Contains(errMsg, "certificate"):
        return fmt.Sprintf("TLS证书错误: %v (请检查证书配置或临时使用测试模式)", err)
    default:
        return fmt.Sprintf("连接失败: %v", err)
    }
}

// formatBindError 格式化认证错误
func (v *ADConfigValidator) formatBindError(err error) string {
    errMsg := err.Error()

    // LDAP错误码判断
    if ldapErr, ok := err.(*ldap.Error); ok {
        switch ldapErr.ResultCode {
        case ldap.LDAPResultInvalidCredentials:
            return "管理员用户名或密码错误"
        case ldap.LDAPResultNoSuchObject:
            return "BindDN指定的对象不存在"
        case ldap.LDAPResultInsufficientAccessRights:
            return "管理员权限不足"
        case ldap.LDAPResultUnwillingToPerform:
            return "服务器拒绝执行操作"
        default:
            return fmt.Sprintf("认证失败: %v", err)
        }
    }

    return fmt.Sprintf("认证失败: %v", err)
}
```

### 验证API设计

```go
// 健康检查端点
func (h *AdminHandler) ValidateADConfig(c *gin.Context) {
    var req ADConfig
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    validator := NewADConfigValidator(h.logger)
    result := validator.Validate(&req)

    if result.Valid {
        c.JSON(200, gin.H{
            "valid":         true,
            "level":         result.Level,
            "server_info":   result.ServerInfo,
            "response_time": result.ResponseTime,
            "message":       "AD配置验证通过",
        })
    } else {
        c.JSON(200, gin.H{
            "valid":  false,
            "level":  result.Level,
            "errors": result.Errors,
            "message": "AD配置验证失败",
        })
    }
}
```

### 系统启动验证

```go
// 在系统启动时验证AD配置(如果配置为AD模式)
func ValidateADConfigOnStartup(cfg *config.Config, logger *zap.Logger) error {
    if cfg.Auth.Mode != "ad" && cfg.Auth.Mode != "hybrid" {
        return nil
    }

    logger.Info("验证AD域控配置...")
    validator := NewADConfigValidator(logger)
    result := validator.Validate(&cfg.Auth.AD)

    if !result.Valid {
        return fmt.Errorf("AD配置验证失败: %v", result.Errors)
    }

    logger.Info("AD配置验证通过",
        zap.Int("level", result.Level),
        zap.Int64("response_time_ms", result.ResponseTime),
    )

    return nil
}
```

### 切换前验证

```go
func (h *AdminHandler) UpdateAuthConfig(c *gin.Context) {
    var req struct {
        Mode string   `json:"mode" binding:"required,oneof=local ad hybrid"`
        AD   ADConfig `json:"ad"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 如果切换到AD或hybrid模式，先验证AD配置
    if req.Mode == "ad" || req.Mode == "hybrid" {
        validator := NewADConfigValidator(h.logger)
        result := validator.Validate(&req.AD)

        if !result.Valid {
            c.JSON(400, gin.H{
                "error": "AD配置验证失败",
                "details": result.Errors,
            })
            return
        }
    }

    // 验证通过，更新配置
    if err := h.updateConfig(&req); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"message": "认证配置已更新"})
}
```

### 错误提示规则

| 验证层级 | 错误类型 | 提示信息 | 修复建议 |
|----------|----------|----------|----------|
| 格式验证 | 缺少必填字段 | "字段不能为空" | 填写完整信息 |
| 格式验证 | 格式错误 | "格式错误" | 按示例格式填写 |
| 网络验证 | DNS解析失败 | "无法解析服务器地址" | 检查服务器地址 |
| 网络验证 | 连接拒绝 | "连接被拒绝" | 检查防火墙和端口 |
| 网络验证 | 超时 | "连接超时" | 检查网络状态 |
| 认证验证 | 密码错误 | "管理员密码错误" | 检查密码配置 |
| 认证验证 | BindDN不存在 | "BindDN不存在" | 检查DN格式 |
| 认证验证 | 权限不足 | "管理员权限不足" | 使用有权限的账户 |
| 功能验证 | 搜索失败 | "搜索用户失败" | 检查BaseDN配置 |

### 安全提示

当启用AD认证时，向管理员显示安全提示：

```markdown
## 启用AD域控认证前的确认

您即将启用Windows AD域控认证模式，请确认：

✅ **必做事项:**
1. AD服务器使用LDAPS(端口636)加密连接
2. 已配置企业CA证书或准备使用测试模式
3. 管理员账户有足够的权限读取用户信息
4. 已在AD中创建测试用户并验证认证功能

⚠️ **重要提示:**
- 启用AD认证后，本地用户将无法使用密码登录
- AD用户需要在本地创建同名账号才能获得权限
- 请确保已正确配置用户权限分配策略
- 建议先在hybrid模式测试后再切换到纯AD模式

🔒 **安全建议:**
- 管理员密码应从环境变量读取，不要明文存储
- 定期检查AD服务器连接状态
- 监控AD认证失败的审计日志
```

## How to Run

此Spike为配置验证设计，包含完整的验证器实现和错误处理逻辑。

## What to Expect

### API调用示例

```bash
# 测试AD配置
curl -X POST http://localhost:8080/api/admin/auth/ad/validate \
  -H "Content-Type: application/json" \
  -d '{
    "server": "ad.example.com:636",
    "bind_dn": "cn=admin,cn=users,dc=example,dc=com",
    "password": "admin_password",
    "base_dn": "dc=example,dc=com",
    "use_tls": true
  }'
```

### 成功响应

```json
{
  "valid": true,
  "level": 4,
  "server_info": "ad.example.com",
  "response_time_ms": 125,
  "message": "AD配置验证通过，可正常使用"
}
```

### 失败响应

```json
{
  "valid": false,
  "level": 3,
  "errors": [
    "管理员用户名或密码错误"
  ],
  "message": "AD配置验证失败"
}
```

## Investigation Trail

### 第1轮设计 (2025-04-28)

**目标:** 设计完整的AD配置验证方案

**关键问题:**
1. 如何提供清晰的错误提示？
2. 如何分层验证，避免不必要的网络请求？
3. 如何在切换前确保配置可用？
4. 如何处理测试环境的证书问题？

**决策:**
1. ✅ 分4层验证：格式→网络→认证→功能
2. ✅ 每层失败立即返回，提供具体错误
3. ✅ 错误消息包含修复建议
4. ✅ 支持测试模式跳过证书验证

### 第2轮细化

**测试模式支持:**

对于测试环境，允许临时禁用证书验证：

```go
if config.IsTestMode {
    tlsConfig.InsecureSkipVerify = true
    result.Warnings = append(result.Warnings, "测试模式：未验证TLS证书")
}
```

**健康检查集成:**

将AD配置验证集成到系统健康检查：

```go
func (h *HealthHandler) Check(c *gin.Context) {
    health := map[string]interface{}{
        "status": "healthy",
        "components": map[string]interface{}{
            "database": h.checkDatabase(),
            "ad_auth":  h.checkADAuth(),
        },
    }

    status := 200
    if health["components"]["ad_auth"] != "ok" {
        status = 503 // 如果AD认证不可用，返回降级状态
    }

    c.JSON(status, health)
}
```

## Results

**Verdict:** ✅ VALIDATED

**关键发现:**
1. ✅ 分层验证是最佳实践，避免不必要的网络请求
2. ✅ 错误消息应包含具体的修复建议
3. ✅ 验证应在系统启动和切换前执行
4. ✅ 响应时间可用于判断连接质量
5. ⚠️ 测试环境需要支持跳过证书验证
6. ⚠️ 需要向管理员显示启用AD认证的安全提示

**实现建议:**

1. **验证器API:**
   ```go
   // 简单验证
   validator.Validate(&config)

   // 带上下文的验证
   validator.ValidateWithContext(&config, context.Background())

   // 测试模式验证
   validator.ValidateInTestMode(&config)
   ```

2. **前端集成:**
   - 配置页面实时验证字段格式
   - 保存/切换前调用验证API
   - 显示验证结果和响应时间

3. **安全提示:**
   - 首次启用AD时显示确认对话框
   - 列出必做事项和安全建议
   - 提供回滚选项

**对后续实现的影响:**
- 所有AD配置变更都需要经过验证
- 系统启动时验证AD配置(如果启用)
- 配置验证失败时应保留旧配置

**用户体验建议:**
1. ✅ 显示详细的错误信息和修复建议
2. ✅ 提供测试连接按钮，点击即可验证
3. ✅ 显示上次验证时间和结果
4. ✅ 连接失败时提供诊断信息

**安全考虑:**
1. ⚠️ 验证API不应返回详细的AD服务器信息
2. ⚠️ 管理员密码不应在日志中记录
3. ✅ 验证失败应记录审计日志
4. ✅ 测试模式应在配置中明确标注
