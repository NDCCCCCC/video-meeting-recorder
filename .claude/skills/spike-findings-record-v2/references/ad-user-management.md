# AD用户管理

## Requirements

以下要求来自Spike研究，是实施中不可协商的设计决策：

**用户映射要求:**
- 使用同名映射(sAMAccountName → username)
- AD认证成功后自动创建本地用户记录
- 本地用户记录存储AD属性用于权限管理

**配置验证要求:**
- 系统启动时验证AD配置(如果启用AD模式)
- 切换认证模式前验证AD配置
- 分层验证：格式→网络→认证→功能

## How to Build It

### 1. 用户模型扩展

```go
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

func (u *User) IsADUser() bool {
    return u.AuthSource == "ad"
}
```

### 2. 数据库迁移

```sql
ALTER TABLE users ADD COLUMN auth_source VARCHAR(20) DEFAULT 'local';
ALTER TABLE users ADD COLUMN ad_username VARCHAR(100);
ALTER TABLE users ADD COLUMN ad_dn VARCHAR(255);
ALTER TABLE users ADD COLUMN ad_guid CHAR(36);
ALTER TABLE users ADD COLUMN last_ad_login DATETIME;
ALTER TABLE users ADD COLUMN ad_department VARCHAR(100);
ALTER TABLE users ADD COLUMN ad_upn VARCHAR(200);

CREATE INDEX idx_users_auth_source ON users(auth_source);
CREATE UNIQUE INDEX idx_users_ad_guid ON users(ad_guid) WHERE ad_guid IS NOT NULL;
```

### 3. AD用户映射流程

```go
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
    
    // 分配默认角色
    if err := a.assignDefaultRole(&user); err != nil {
        return nil, err
    }
    
    if err := a.db.Create(&user).Error; err != nil {
        return nil, err
    }
    
    return &user, nil
}
```

### 4. 属性映射表

| AD属性 | 本地User字段 | 说明 |
|--------|-------------|------|
| sAMAccountName | username | 登录用户名 |
| mail | email | 邮箱 |
| displayName | full_name | 显示名称 |
| department | ad_department | 部门(扩展字段) |
| userPrincipalName | ad_upn | UPN(扩展字段) |
| distinguishedName (DN) | ad_dn | AD唯一标识 |
| objectGUID | ad_guid | AD GUID |

### 5. 配置验证器

```go
type ADConfigValidator struct {
    logger *zap.Logger
}

func (v *ADConfigValidator) Validate(config *ADConfig) *ADConfigValidationResult {
    result := &ADConfigValidationResult{
        Valid:  false,
        Level:  0,
        Errors: []string{},
    }
    
    // 第1层: 格式验证
    if err := v.validateFormat(config); err != nil {
        result.Errors = append(result.Errors, err.Error())
        return result
    }
    result.Level = 1
    
    // 第2层: 网络验证
    conn, err := v.testConnection(config)
    if err != nil {
        result.Errors = append(result.Errors, v.formatConnectionError(err))
        return result
    }
    defer conn.Close()
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
    }
    result.Level = 4
    
    result.Valid = true
    return result
}
```

### 6. 验证API

```bash
# 验证AD配置
POST /api/admin/auth/ad/validate
{
  "server": "ad.example.com:636",
  "bind_dn": "cn=admin,dc=example,dc=com",
  "password": "admin_password",
  "base_dn": "dc=example,dc=com",
  "use_tls": true
}
```

### 7. 用户管理界面扩展

显示：
- 认证来源 (本地/AD)
- AD用户名
- AD部门
- 最后AD登录时间

操作：
- AD用户禁用密码修改功能(提示联系域管理员)
- 同步AD用户信息按钮

## What to Avoid

- ❌ AD用户没有本地记录 - 无法管理权限
- ❌ 使用UPN作为本地username - 格式可能不一致
- ❌ 不区分本地用户和AD用户 - 密码修改逻辑混乱
- ❌ 切换到AD模式前不验证配置 - 导致系统不可用
- ❌ 配置验证失败时返回详细错误信息给前端 - 安全风险

## Constraints

**用户名限制:**
- sAMAccountName最大长度: 20个字符
- 只能包含: 字母、数字、特殊字符(!#$%&'()*+,-./:;<=>?@[\]^_`{|}~)
- 不区分大小写

**AD属性限制:**
- mail可能为空
- displayName可能为空
- userAccountControl是位掩码，需要正确解析

**配置验证限制:**
- 格式验证不检查网络连通性
- 网络验证不检查AD服务器配置
- 认证验证需要有效的管理员凭据

## Origin

Synthesized from spikes: 004, 005
Source files available in: sources/004-ad-user-mapping/, sources/005-ad-config-validation/
