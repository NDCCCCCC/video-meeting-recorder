---
spike: 004
name: ad-user-mapping
type: standard
validates: "Given AD用户认证成功，当查找本地同名用户时，能够正确映射并分配权限"
verdict: VALIDATED
related: [001, 003]
tags: [ldap, user-mapping, permissions, synchronization]
---

# Spike 004: ad-user-mapping

## What This Validates

**Given:** 用户通过AD域控认证成功
**When:** 系统需要获取用户权限和返回登录响应
**Then:**
- 能够在本地users表中找到或创建对应的用户记录
- AD用户的属性正确映射到本地用户字段
- 用户的角色和权限正确分配
- 登录响应包含完整的用户信息

## Research

### 用户映射策略

| 策略 | 描述 | 优点 | 缺点 | 推荐度 |
|------|------|------|------|--------|
| **同名映射** | AD用户名=sAMAccountName=本地username | 简单直观 | 需要手动创建本地用户 | ✅ 推荐 |
| **UPN映射** | userPrincipalName的@前部分=本地username | 更符合企业习惯 | 可能与sAMAccountName不一致 | ⚠️ 可选 |
| **DN映射** | 完整DN作为唯一标识 | 绝对唯一 | 不便于用户记忆 | ❌ 不推荐 |
| **属性映射** | employeeNumber等自定义属性 | 灵活 | 需要AD额外配置 | ⚠️ 企业场景 |

### 推荐方案：同名映射 + 自动创建

```
┌─────────────────────────────────────────────────────────────┐
│                   AD认证成功                                │
│  - username: "john.doe"                                     │
│  - email: "john.doe@example.com"                            │
│  - displayName: "John Doe"                                  │
│  - department: "Engineering"                                │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
            ┌──────────────────────┐
            │ 查询本地users表      │
            │ WHERE username = ?   │
            └──────────┬───────────┘
                       │
            ┌──────────┴──────────┐
            │                     │
         找到                 未找到
            │                     │
            │                     ▼
            │          ┌──────────────────────┐
            │          │ 自动创建本地用户      │
            │          │ - auth_source: "ad"  │
            │          │ - ad_username: "..." │
            │          │ - ad_dn: "..."       │
            │          │ - 随机bcrypt密码     │
            │          └──────────┬───────────┘
            │                     │
            └──────────┬──────────┘
                       ▼
            ┌──────────────────────┐
            │ 验证用户状态          │
            │ - is_active = true   │
            │ - 角色已分配         │
            └──────────┬───────────┘
                       │
                       ▼
            ┌──────────────────────┐
            │ 生成JWT token        │
            │ 返回LoginResponse    │
            └──────────────────────┘
```

### 属性映射表

| AD属性 | 本地User字段 | 说明 | 示例 |
|--------|-------------|------|------|
| sAMAccountName | username | 登录用户名 | john.doe |
| mail | email | 邮箱 | john.doe@example.com |
| displayName | full_name | 显示名称 | John Doe |
| department | - | 部门(扩展字段) | Engineering |
| userPrincipalName | - | UPN(扩展字段) | john.doe@example.com |
| distinguishedName (DN) | ad_dn | AD唯一标识 | CN=John Doe,OU=Users,DC=example,DC=com |
| objectGUID | ad_guid | AD GUID | {550e8c67-a...} |

### 数据库扩展

```sql
-- 扩展users表
ALTER TABLE users ADD COLUMN auth_source VARCHAR(20) DEFAULT 'local';
ALTER TABLE users ADD COLUMN ad_username VARCHAR(100);
ALTER TABLE users ADD COLUMN ad_dn VARCHAR(255);
ALTER TABLE users ADD COLUMN ad_guid CHAR(36);
ALTER TABLE users ADD COLUMN last_ad_login DATETIME;
ALTER TABLE users ADD COLUMN ad_department VARCHAR(100);
ALTER TABLE users ADD COLUMN ad_upn VARCHAR(200);

-- 索引
CREATE INDEX idx_users_auth_source ON users(auth_source);
CREATE UNIQUE INDEX idx_users_ad_guid ON users(ad_guid) WHERE ad_guid IS NOT NULL;
```

```go
// User模型扩展
type User struct {
    // ... 现有字段

    // 新增AD相关字段
    AuthSource   string     `gorm:"type:varchar(20);default:'local'" json:"auth_source"`
    ADUsername   string     `gorm:"type:varchar(100)" json:"ad_username,omitempty"`
    ADDN         string     `gorm:"type:varchar(255)" json:"ad_dn,omitempty"`
    ADGUID       string     `gorm:"type:char(36);index" json:"ad_guid,omitempty"`
    LastADLogin  *time.Time `json:"last_ad_login,omitempty"`
    ADDepartment string     `gorm:"type:varchar(100)" json:"ad_department,omitempty"`
    ADUPN        string     `gorm:"type:varchar(200)" json:"ad_upn,omitempty"`
}

// IsADUser 判断是否为AD用户
func (u *User) IsADUser() bool {
    return u.AuthSource == "ad"
}

// GetADLoginName 获取AD登录名
func (u *User) GetADLoginName() string {
    if u.ADUsername != "" {
        return u.ADUsername
    }
    return u.Username
}
```

### AD认证器登录流程

```go
func (a *ADAuthenticator) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
    // 1. AD认证
    adUser, err := a.authenticateAD(req.Username, req.Password)
    if err != nil {
        return nil, err
    }

    // 2. 查找或创建本地用户
    localUser, err := a.findOrCreateLocalUser(adUser)
    if err != nil {
        return nil, fmt.Errorf("本地用户映射失败: %w", err)
    }

    // 3. 检查用户状态
    if !localUser.IsActive {
        return nil, errors.New("用户已被禁用")
    }

    // 4. 更新最后登录时间和AD信息
    now := time.Now()
    localUser.LastLoginAt = &now
    localUser.LastADLogin = &now
    a.db.Save(localUser)

    // 5. 生成token
    tokenPair, err := a.tokenService.GenerateTokenPair(localUser)
    if err != nil {
        return nil, err
    }

    // 6. 记录session
    a.tokenService.CreateSession(localUser.ID, tokenPair.AccessToken, ipAddress, userAgent, tokenPair.ExpiresAt)

    return &LoginResponse{
        AccessToken:  tokenPair.AccessToken,
        RefreshToken: tokenPair.RefreshToken,
        ExpiresIn:    int64(a.tokenService.expireHours * 3600),
        User:         a.toUserDTO(localUser),
    }, nil
}

func (a *ADAuthenticator) findOrCreateLocalUser(adUser *ADUser) (*models.User, error) {
    // 先尝试通过username查找
    var user models.User
    err := a.db.Where("username = ?", adUser.Username).First(&user).Error

    if err == nil {
        // 找到本地用户，更新AD信息
        a.updateADInfo(&user, adUser)
        return &user, nil
    }

    if !errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, err
    }

    // 未找到，创建新用户
    user = models.User{
        Username:     adUser.Username,
        Email:        adUser.Email,
        FullName:     adUser.DisplayName,
        AuthSource:   "ad",
        ADUsername:   adUser.Username,
        ADDN:         adUser.DN,
        ADGUID:       adUser.ObjectGUID,
        ADDepartment: adUser.Department,
        ADUPN:        adUser.UserPrincipalName,
        IsActive:     true,
    }

    // 生成随机密码(AD用户不会使用)
    randomPassword := utils.GenerateRandomPassword(32)
    if err := user.SetPassword(randomPassword); err != nil {
        return nil, err
    }

    // 分配默认角色(可配置)
    if err := a.assignDefaultRole(&user); err != nil {
        return nil, err
    }

    if err := a.db.Create(&user).Error; err != nil {
        return nil, err
    }

    return &user, nil
}

func (a *ADAuthenticator) updateADInfo(user *models.User, adUser *ADUser) {
    // 仅更新AD相关字段，保留本地配置
    user.Email = adUser.Email
    user.FullName = adUser.DisplayName
    user.ADDN = adUser.DN
    user.ADGUID = adUser.ObjectGUID
    user.ADDepartment = adUser.Department
    user.ADUPN = adUser.UserPrincipalName
}
```

### 权限分配策略

| 策略 | 描述 | 配置 |
|------|------|------|
| **默认角色** | 新AD用户分配默认角色 | config.ad.default_role_id |
| **AD组映射** | AD组→本地角色映射表 | ad_group_roles表 |
| **手动分配** | 管理员手动分配 | 现有角色管理界面 |

```go
// AD组映射表(可选)
type ADGroupRoleMapping struct {
    ADGroupDN  string `gorm:"type:varchar(255);uniqueIndex" json:"ad_group_dn"`
    RoleID     uint   `json:"role_id"`
    Role       Role   `gorm:"foreignKey:RoleID" json:"role"`
}

// 根据AD组成员资格分配角色
func (a *ADAuthenticator) assignRolesByMembership(user *models.User, adUser *ADUser) error {
    // 查询用户所属的AD组
    adGroups, err := a.getUserADGroups(adUser.DN)
    if err != nil {
        return err
    }

    // 查找映射的角色
    var mappings []ADGroupRoleMapping
    a.db.Where("ad_group_dn IN ?", adGroups).Find(&mappings)

    // 分配角色
    for _, mapping := range mappings {
        a.db.Exec("INSERT INTO users_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
            user.ID, mapping.RoleID)
    }

    return nil
}
```

### 用户同步(可选)

对于需要定期同步AD用户信息的场景：

```go
// 同步AD用户信息到本地
func (a *ADAuthenticator) SyncUser(adUsername string) error {
    // 1. 从AD获取最新信息
    adUser, err := a.getUserFromAD(adUsername)
    if err != nil {
        return err
    }

    // 2. 更新本地用户
    var user models.User
    err = a.db.Where("ad_username = ? OR username = ?", adUsername, adUsername).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            // 用户不存在，创建
            _, err = a.findOrCreateLocalUser(adUser)
            return err
        }
        return err
    }

    // 更新AD信息
    a.updateADInfo(&user, adUser)
    return a.db.Save(&user).Error
}
```

## How to Run

此Spike为研究和设计验证，包含关键代码示例。

## What to Expect

### 配置示例

```yaml
auth:
  mode: "ad"
  ad:
    server: "ad.example.com:636"
    bind_dn: "cn=admin,dc=example,dc=com"
    password: "${AD_PASSWORD}"
    base_dn: "dc=example,dc=com"
    use_tls: true

    # AD用户映射配置
    auto_create_users: true         # 自动创建本地用户
    default_role_id: 3              # 默认角色ID
    sync_on_login: true             # 登录时同步信息
    allow_ad_group_mapping: false   # 是否启用AD组映射(高级功能)
```

### 数据库迁移

```bash
# 创建迁移文件
migrate create add_ad_user_fields

# 运行迁移
migrate up
```

### 管理界面扩展

用户管理界面需要显示：
- 认证来源 (本地/AD)
- AD用户名
- AD部门
- 最后AD登录时间

## Investigation Trail

### 第1轮设计 (2025-04-28)

**目标:** 设计AD用户到本地用户的映射方案

**关键问题:**
1. AD用户如何在本地表示？
2. 如何避免重复创建用户？
3. AD用户的权限如何分配？
4. AD信息变更如何同步？

**决策:**
1. ✅ 使用同名映射(sAMAccountName = username)
2. ✅ AD认证成功后自动创建本地用户
3. ✅ 新用户分配默认角色，支持手动调整
4. ✅ 每次登录更新AD信息(邮箱、部门等)

### 第2轮细化

**边界情况处理:**

| 场景 | 处理方式 |
|------|----------|
| AD用户已存在本地同名账户 | 更新AD信息，保留现有权限 |
| AD用户被禁用 | 拒绝登录，提示联系管理员 |
| 本地用户被禁用 | 拒绝登录，即使AD认证成功 |
| AD用户名包含特殊字符 | LDAP搜索时自动转义 |
| AD用户无email | 允许，email字段可为空 |

## Results

**Verdict:** ✅ VALIDATED

**关键发现:**
1. ✅ 同名映射(sAMAccountName→username)是最简单可靠的方案
2. ✅ 自动创建本地用户时使用随机bcrypt密码(不会被使用)
3. ✅ AD用户需要扩展User模型存储AD属性
4. ✅ 权限分配支持默认角色和手动分配两种方式
5. ✅ 每次登录可同步AD最新信息
6. ⚠️ 需要处理AD用户与本地用户的冲突场景

**实现建议:**

1. **最小实现:**
   - AD认证后自动创建本地用户
   - 分配默认角色
   - 存储基本AD信息(DN、GUID)

2. **完整实现:**
   - AD组→角色映射
   - 定期用户同步
   - AD用户管理界面
   - 批量导入AD用户

3. **数据库变更:**
   ```sql
   ALTER TABLE users ADD COLUMN auth_source VARCHAR(20) DEFAULT 'local';
   ALTER TABLE users ADD COLUMN ad_username VARCHAR(100);
   ALTER TABLE users ADD COLUMN ad_dn VARCHAR(255);
   ALTER TABLE users ADD COLUMN ad_guid CHAR(36);
   ALTER TABLE users ADD COLUMN last_ad_login DATETIME;
   ```

**对后续实现的影响:**
- 需要修改用户注册逻辑(AD模式禁止本地注册)
- 需要修改密码修改逻辑(AD用户提示联系域管理员)
- 需要扩展用户管理界面显示AD信息

**安全考虑:**
1. ⚠️ AD用户的随机密码哈希仅用于满足数据库约束，不应被使用
2. ⚠️ AD用户禁用状态应定期同步或实时检查
3. ✅ AD敏感信息(如DN)不应在前端暴露
4. ✅ 审计日志应记录用户来源(本地/AD)
