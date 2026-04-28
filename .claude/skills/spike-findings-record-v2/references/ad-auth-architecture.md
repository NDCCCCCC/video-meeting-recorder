# AD认证架构

## Requirements

以下要求来自Spike研究，是实施中不可协商的设计决策：

**架构要求:**
- 使用策略模式实现认证方式切换(local/ad/hybrid)
- 支持配置热加载，无需重启服务
- AD认证后需要在本地创建用户映射记录

**功能要求:**
- 支持三种认证模式：local(本地)、ad(域控)、hybrid(混合)
- AD用户认证成功后自动创建本地用户记录
- 所有认证模式切换记录审计日志

## How to Build It

### 1. 认证器接口

```go
// Authenticator 认证器接口
type Authenticator interface {
    // Login 用户登录
    Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error)
    
    // Logout 用户登出
    Logout(token string) error
    
    // ValidateToken 验证token
    ValidateToken(token string) (*UserDTO, error)
    
    // Name 认证器名称
    Name() string
}
```

### 2. 认证服务

```go
type AuthService struct {
    config       *AuthConfig
    localAuth    Authenticator
    adAuth       Authenticator
    tokenService *TokenService
}

func (s *AuthService) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
    var authenticator Authenticator
    
    // 根据模式选择认证器
    switch s.config.Mode {
    case "local":
        authenticator = s.localAuth
    case "ad":
        authenticator = s.adAuth
    case "hybrid":
        return s.loginHybrid(req, ipAddress, userAgent)
    default:
        return nil, errors.New("无效的认证模式")
    }
    
    return authenticator.Login(req, ipAddress, userAgent)
}
```

### 3. 配置结构

```go
type AuthConfig struct {
    Mode string `mapstructure:"mode" json:"mode"` // local, ad, hybrid
    
    // 本地认证配置
    Local LocalAuthConfig `mapstructure:"local" json:"local"`
    
    // AD认证配置
    AD ADConfig `mapstructure:"ad" json:"ad"`
}
```

### 4. 模式切换流程

```
配置变更请求
    ↓
验证AD配置(如果切换到AD/hybrid)
    ↓
┌───┴────────┐
│           │
通过      失败
│           │
│           ↓
│        回滚，提示错误
│
↓
热加载新配置
    ↓
记录审计日志
    ↓
返回成功
```

### 5. Hybrid模式登录

```go
func (s *AuthService) loginHybrid(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
    // 先尝试本地认证
    resp, err := s.localAuth.Login(req, ipAddress, userAgent)
    if err == nil {
        return resp, nil
    }
    
    // 本地认证失败，记录日志
    s.logger.Info("本地认证失败，尝试AD认证")
    
    // 尝试AD认证
    return s.adAuth.Login(req, ipAddress, userAgent)
}
```

### 6. 配置验证API

```bash
# 测试AD配置
POST /api/admin/auth/ad/validate
Content-Type: application/json

{
  "server": "ad.example.com:636",
  "bind_dn": "cn=admin,...",
  "password": "test_password",
  "base_dn": "dc=example,dc=com",
  "use_tls": true
}
```

## What to Avoid

- ❌ 使用if/else硬编码认证方式 - 难以扩展
- ❌ 切换模式时不验证AD配置
- ❌ 切换后需要重启服务
- ❌ AD用户没有本地映射记录
- ❌ 配置变更不记录审计日志

## Constraints

**模式限制:**
- `local`: 仅本地认证，AD用户无法登录
- `ad`: 仅AD认证，本地用户无法使用密码登录
- `hybrid`: 先本地后AD，兼容性最佳但可能增加登录延迟

**切换限制:**
- 切换到AD/hybrid模式前必须验证AD配置
- AD配置验证失败时保留旧配置
- 现有token在切换后继续有效

**用户限制:**
- AD用户需要在本地有同名记录
- AD用户的本地密码哈希不会被使用(随机生成)
- AD用户禁用状态需要定期同步

## Origin

Synthesized from spikes: 003
Source files available in: sources/003-auth-switch-architecture/
