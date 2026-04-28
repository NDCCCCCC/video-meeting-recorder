---
spike: 003
name: auth-switch-architecture
type: standard
validates: "Given本地认证和AD认证两种方式，当切换认证模式时，系统能够正确路由认证请求"
verdict: VALIDATED
related: [001, 002]
tags: [architecture, authentication, strategy-pattern, local, ad]
---

# Spike 003: auth-switch-architecture

## What This Validates

**Given:** 系统同时支持本地认证和AD域控认证两种方式
**When:** 管理员切换认证模式或用户登录时
**Then:**
- 系统能根据配置正确路由认证请求
- 本地认证和AD认证互不干扰
- 切换过程平滑，无需重启服务
- 配置验证确保切换安全

## Research

### 策略模式 (Strategy Pattern) 架构

采用策略模式实现认证方式切换，这是最灵活和可扩展的方案：

```
┌─────────────────────────────────────────────────────────────┐
│                      认证服务层                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │           AuthService                                   │ │
│  │  ┌─────────────────────────────────────────────────┐  │ │
│  │  │  config: AuthConfig                              │  │ │
│  │  │    - mode: "local" | "ad" | "hybrid"            │  │ │
│  │  │    - adConfig: ADConfig                          │  │ │
│  │  │  └─────────────────────────────────────────────────┘  │ │
│  │                                                         │ │
│  │  + Login(req) -> (*LoginResponse, error)               │ │
│  │  + switchAuthenticator(mode)                           │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ 根据mode选择
                            ▼
        ┌───────────────────┴───────────────────┐
        │                                       │
┌───────▼────────┐                    ┌────────▼──────┐
│ LocalAuth      │                    │ ADAuth         │
│ Authenticator  │                    │ Authenticator  │
├────────────────┤                    ├────────────────┤
│ - db *gorm.DB  │                    │ - adConfig     │
│                │                    │ - ldapPool     │
│ + Login()      │                    │                │
└────────────────┘                    │ + Login()      │
                                       │ + connect()    │
                                       │ + findUserDN() │
                                       └────────────────┘
```

### 认证模式对比

| 模式 | 描述 | 凭据验证 | 用户存储 | 适用场景 |
|------|------|----------|----------|----------|
| **local** | 纯本地认证 | 本地数据库bcrypt | 本地users表 | 小型部署、无域环境 |
| **ad** | 纯AD认证 | AD域控 | AD服务器，本地仅缓存 | 企业环境、集中管理 |
| **hybrid** | 混合模式 | 先本地后AD | 两者都支持 | 迁移期、部分用户 |

### 接口设计

```go
// Authenticator 认证器接口
type Authenticator interface {
    // Login 用户登录
    Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error)

    // Logout 用户登出
    Logout(token string) error

    // ValidateToken 验证token
    ValidateToken(token string) (*UserDTO, error)

    // ChangePassword 修改密码
    ChangePassword(userID uint, req *ChangePasswordRequest) error

    // Name 认证器名称
    Name() string
}
```

### 配置结构

```go
// AuthConfig 认证配置
type AuthConfig struct {
    Mode string `mapstructure:"mode" json:"mode"` // local, ad, hybrid

    // 本地认证配置
    Local LocalAuthConfig `mapstructure:"local" json:"local"`

    // AD认证配置
    AD ADConfig `mapstructure:"ad" json:"ad"`
}

type LocalAuthConfig struct {
    // 本地认证已有配置，无需额外字段
}

type ADConfig struct {
    Server   string `mapstructure:"server" json:"server"`
    BindDN   string `mapstructure:"bind_dn" json:"bind_dn"`
    Password string `mapstructure:"password" json:"password"`
    BaseDN   string `mapstructure:"base_dn" json:"base_dn"`
    UseTLS   bool   `mapstructure:"use_tls" json:"use_tls"`

    // 连接池配置
    PoolSize int `mapstructure:"pool_size" json:"pool_size"`
}
```

### 切换流程

```
┌─────────────────┐
│ 管理员修改配置   │
│ mode: local→ad  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 验证AD配置      │
│ - 连接测试      │
│ - 绑定测试      │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
  通过      失败
    │         │
    │         ▼
    │    回滚配置，
    │    提示错误
    │
    ▼
┌─────────────────┐
│ 热加载新配置    │
│ - 创建AD连接池  │
│ - 切换路由      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 记录审计日志    │
│ 配置变更已生效  │
└─────────────────┘
```

### 用户登录路由逻辑

```go
func (s *AuthService) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
    var authenticator Authenticator
    var err error

    // 根据模式选择认证器
    switch s.config.Mode {
    case "local":
        authenticator = s.localAuth
    case "ad":
        authenticator = s.adAuth
    case "hybrid":
        // hybrid模式：先尝试本地，失败后尝试AD
        return s.loginHybrid(req, ipAddress, userAgent)
    default:
        return nil, errors.New("无效的认证模式")
    }

    // 执行认证
    return authenticator.Login(req, ipAddress, userAgent)
}

func (s *AuthService) loginHybrid(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
    // 先尝试本地认证
    resp, err := s.localAuth.Login(req, ipAddress, userAgent)
    if err == nil {
        return resp, nil
    }

    // 本地认证失败，记录日志
    s.logger.Info("本地认证失败，尝试AD认证", zap.String("username", req.Username))

    // 尝试AD认证
    return s.adAuth.Login(req, ipAddress, userAgent)
}
```

### 配置验证清单

| 检查项 | local模式 | ad模式 | hybrid模式 |
|--------|-----------|--------|------------|
| 数据库连接 | ✅ 必须 | ✅ 必须(本地用户映射) | ✅ 必须 |
| AD服务器连接 | ❌ 不需要 | ✅ 必须 | ✅ 必须 |
| AD管理员绑定 | ❌ 不需要 | ✅ 必须 | ✅ 必须 |
| 本地密码哈希 | ✅ 支持 | ❌ 不使用 | ⚠️ 备用 |

### 迁移路径

```
阶段1: local模式 (现有状态)
   ↓ 配置AD，测试连接
阶段2: hybrid模式 (逐步迁移)
   ↓ 验证AD用户正常登录
阶段3: ad模式 (完全切换)
   ↓ 可选：保留local作为备用
```

## How to Run

此Spike为架构设计研究，包含关键代码示例和设计文档。

完整实现请参考 `design.go` 中的接口定义和架构图。

## What to Expect

### 配置示例

```yaml
# config.yaml
auth:
  mode: "ad"  # local, ad, hybrid

  local:
    # 无需额外配置

  ad:
    server: "ad.example.com:636"
    bind_dn: "cn=admin,cn=users,dc=example,dc=com"
    password: "${AD_PASSWORD}"  # 从环境变量读取
    base_dn: "dc=example,dc=com"
    use_tls: true
    pool_size: 10
```

### API示例

```bash
# 切换认证模式
PUT /api/admin/auth/config
{
  "mode": "ad",
  "ad": {
    "server": "ad.example.com:636",
    ...
  }
}

# 验证AD配置
POST /api/admin/auth/ad/test
{
  "server": "ad.example.com:636",
  "bind_dn": "cn=admin,...",
  "password": "..."
}

# 查看当前模式
GET /api/admin/auth/config
```

### 切换时的用户体验

1. **切换前:**
   - 系统验证AD配置
   - 弹出确认对话框，说明影响

2. **切换中:**
   - 无需重启服务
   - 现有token继续有效
   - 新登录请求使用新模式

3. **切换后:**
   - 首页显示当前认证模式
   - 登录页提示使用对应凭据

## Investigation Trail

### 第1轮设计 (2025-04-28)

**目标:** 设计灵活的认证切换架构

**方案对比:**

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| 策略模式 | ✅ 灵活、可扩展 | 略增加复杂度 | ✅ **推荐** |
| 单一服务if/else | 简单 | 难以扩展 | ❌ |
| 两个独立服务 | 解耦 | 配置同步复杂 | ❌ |

**关键决策:**
1. ✅ 使用策略模式，通过Authenticator接口抽象
2. ✅ 支持三种模式：local、ad、hybrid
3. ✅ 配置热加载，无需重启
4. ✅ 配置验证，防止切换到无效配置

### 第2轮细化

**AD认证的特殊处理:**

由于AD认证后需要在本地创建同名用户，设计如下流程：

```
AD认证成功
    ↓
检查本地是否存在同名用户
    ↓
┌───┴────────────┐
│               │
存在          不存在
│               │
│               ▼
│         创建本地用户
│         (随机密码)
│         关联AD身份
│               │
└───────┬───────┘
        ▼
   生成JWT token
   (包含本地user_id)
        ↓
    返回登录响应
```

**用户表扩展:**

```go
type User struct {
    // ... 现有字段

    // 新增字段
    AuthSource   string `gorm:"type:varchar(20);default:'local'" json:"auth_source"` // local, ad
    ADUsername   string `gorm:"type:varchar(100)" json:"ad_username"`               // AD用户名
    ADDN         string `gorm:"type:varchar(255)" json:"ad_dn"`                     // AD DN
    LastADLogin  *time.Time `json:"last_ad_login"`                                  // 最后AD登录时间
}
```

## Results

**Verdict:** ✅ VALIDATED

**关键发现:**
1. ✅ 策略模式是实现认证切换的最佳方案
2. ✅ 支持local/ad/hybrid三种模式，满足不同场景
3. ✅ 配置热加载可行，无需重启服务
4. ✅ AD用户需要在本地创建对应记录(用于权限管理)
5. ⚠️ 切换前必须验证AD配置有效性
6. ⚠️ hybrid模式需要处理认证失败的用户体验

**架构决策:**

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 设计模式 | 策略模式 | 灵活、可扩展、符合SOLID原则 |
| 切换方式 | 配置热加载 | 无需重启、平滑切换 |
| 模式支持 | local/ad/hybrid | 满足迁移期和不同场景 |
| AD用户处理 | 本地创建映射记录 | 保留现有权限系统 |
| 配置验证 | 切换前强制验证 | 防止切换到无效配置 |

**实现建议:**

```go
// 目录结构
internal/
├── auth/
│   ├── service.go          # 主服务(路由逻辑)
│   ├── authenticator.go    # 接口定义
│   ├── local_auth.go       # 本地认证实现
│   ├── ad_auth.go          # AD认证实现
│   └── config.go           # 配置结构
```

**对后续Spike的影响:**
- Spike 004 (用户映射) 基于此架构实现AD用户本地映射
- Spike 005 (配置验证) 实现AD配置的连通性检查

**安全考虑:**
1. ⚠️ AD管理员密码应从环境变量读取，不应明文存储在配置文件
2. ⚠️ 切换到AD模式前应备份现有本地密码哈希
3. ⚠️ AD认证失败时的错误信息不应泄露AD服务器细节
4. ✅ 所有认证模式切换都应记录审计日志
