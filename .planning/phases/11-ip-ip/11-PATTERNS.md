# Phase 11: IP地址登录限制 - Pattern Map

**Mapped:** 2026-04-27
**Files analyzed:** 12
**Analogs found:** 12 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/user.go` | model | CRUD | `internal/models/user.go` | exact (modify existing) |
| `internal/models/role.go` | model | CRUD | `internal/models/role.go` | exact (modify existing) |
| `internal/auth/ip_validator.go` | utility | transform | `internal/models/api_key.go` | role-match (IP validation) |
| `internal/auth/service.go` | service | request-response | `internal/auth/service.go` | exact (modify existing) |
| `internal/handlers/auth_handler.go` | handler | request-response | `internal/handlers/auth_handler.go` | exact (modify existing) |
| `internal/middleware/audit.go` | middleware | event-driven | `internal/middleware/audit.go` | exact (modify existing) |
| `internal/migrations/011_add_ip_restrictions.go` | migration | batch | `internal/migrations/006_multi_role_migration.go` | role-match |
| `frontend/src/types/user.ts` | types | transform | `frontend/src/types/user.ts` | exact (modify existing) |
| `frontend/src/types/role.ts` | types | transform | `frontend/src/types/role.ts` | exact (modify existing) |
| `frontend/src/api/user.ts` | api-client | request-response | `frontend/src/api/user.ts` | exact (modify existing) |
| `frontend/src/api/role.ts` | api-client | request-response | `frontend/src/api/role.ts` | exact (modify existing) |
| `frontend/src/pages/system/users/index.tsx` | component | request-response | `frontend/src/pages/system/users/index.tsx` | exact (modify existing) |

## Pattern Assignments

### `internal/models/user.go` (model, CRUD - modify existing)

**Analog:** `internal/models/user.go` (existing file)

**Add AllowedIPs field** (after line 17):
```go
type User struct {
    Base
    Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
    PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
    Email        string     `gorm:"type:varchar(100)" json:"email"`
    FullName     string     `gorm:"type:varchar(100)" json:"full_name"`
    Roles        []Role     `gorm:"many2many:users_roles;" json:"roles,omitempty"`
    AllowedIPs   string     `gorm:"type:text" json:"allowed_ips"` // NEW: JSON数组字符串
    IsActive     bool       `gorm:"default:true" json:"is_active"`
    LastLoginAt  *time.Time `json:"last_login_at"`
    APIKeys      []APIKey   `gorm:"foreignKey:UserID" json:"api_keys,omitempty"`
}
```

**Add helper methods** (after line 72):
```go
// GetAllowedIPs 获取IP限制列表
func (u *User) GetAllowedIPs() []string {
    if u.AllowedIPs == "" {
        return []string{}
    }
    var ips []string
    _ = json.Unmarshal([]byte(u.AllowedIPs), &ips)
    return ips
}

// SetAllowedIPs 设置IP限制列表
func (u *User) SetAllowedIPs(ips []string) error {
    data, err := json.Marshal(ips)
    if err != nil {
        return err
    }
    u.AllowedIPs = string(data)
    return nil
}
```

---

### `internal/models/role.go` (model, CRUD - modify existing)

**Analog:** `internal/models/role.go` (existing file)

**Add AllowedIPs field** (after line 10):
```go
type Role struct {
    Base
    Name        string       `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
    Description string       `gorm:"type:text" json:"description"`
    Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
    AllowedIPs  string       `gorm:"type:text" json:"allowed_ips"` // NEW: JSON数组字符串
}
```

**Add helper methods** (after line 31):
```go
// GetAllowedIPs 获取IP限制列表
func (r *Role) GetAllowedIPs() []string {
    if r.AllowedIPs == "" {
        return []string{}
    }
    var ips []string
    _ = json.Unmarshal([]byte(r.AllowedIPs), &ips)
    return ips
}

// SetAllowedIPs 设置IP限制列表
func (r *Role) SetAllowedIPs(ips []string) error {
    data, err := json.Marshal(ips)
    if err != nil {
        return err
    }
    r.AllowedIPs = string(data)
    return nil
}
```

---

### `internal/auth/ip_validator.go` (utility, transform)

**Analog:** `internal/models/api_key.go` (lines 126-150)

**IP validation pattern** (source: api_key.go `IsIPAllowed` method):
```go
// Source: internal/models/api_key.go lines 127-150
package auth

import (
    "encoding/json"
    "errors"
    "net"
    "strings"
)

// IPValidator IP地址验证器
type IPValidator struct{}

// ValidateIP 验证单个IP地址
func (v *IPValidator) ValidateIP(ipStr string) error {
    ip := net.ParseIP(ipStr)
    if ip == nil {
        return errors.New("invalid IP address")
    }
    // Reject IPv6 per D-09
    if ip.To4() == nil {
        return errors.New("IPv6 is not supported")
    }
    return nil
}

// ValidateCIDR 验证CIDR范围
func (v *IPValidator) ValidateCIDR(cidr string) error {
    _, _, err := net.ParseCIDR(cidr)
    if err != nil {
        return errors.New("invalid CIDR range")
    }
    return nil
}

// ValidateIPRange 验证IP范围 (e.g., "192.168.1.100-192.168.1.200")
func (v *IPValidator) ValidateIPRange(rangeStr string) error {
    parts := strings.Split(rangeStr, "-")
    if len(parts) != 2 {
        return errors.New("invalid IP range format")
    }
    startIP := net.ParseIP(strings.TrimSpace(parts[0]))
    endIP := net.ParseIP(strings.TrimSpace(parts[1]))
    if startIP == nil || endIP == nil {
        return errors.New("invalid IP addresses in range")
    }
    if startIP.To4() == nil || endIP.To4() == nil {
        return errors.New("IPv6 is not supported")
    }
    return nil
}

// IsIPAllowed 检查客户端IP是否在允许列表中
func (v *IPValidator) IsIPAllowed(clientIP string, allowedList []string) (bool, error) {
    clientAddr := net.ParseIP(clientIP)
    if clientAddr == nil {
        return false, errors.New("invalid client IP")
    }

    for _, allowed := range allowedList {
        // Single IP
        if strings.Contains(allowed, "/") == false && strings.Contains(allowed, "-") == false {
            if clientIP == allowed {
                return true, nil
            }
            continue
        }

        // CIDR range
        if strings.Contains(allowed, "/") {
            _, ipNet, err := net.ParseCIDR(allowed)
            if err != nil {
                continue // Skip invalid CIDR
            }
            if ipNet.Contains(clientAddr) {
                return true, nil
            }
            continue
        }

        // IP range
        if strings.Contains(allowed, "-") {
            parts := strings.Split(allowed, "-")
            startIP := net.ParseIP(strings.TrimSpace(parts[0]))
            endIP := net.ParseIP(strings.TrimSpace(parts[1]))
            // Compare IPs byte by byte
            if bytes.Compare(clientAddr, startIP) >= 0 && bytes.Compare(clientAddr, endIP) <= 0 {
                return true, nil
            }
            continue
        }
    }

    return false, nil
}
```

**Required imports:**
```go
import (
    "bytes"
    "encoding/json"
    "errors"
    "net"
    "strings"
)
```

---

### `internal/auth/service.go` (service, request-response - modify existing)

**Analog:** `internal/auth/service.go` (existing file, lines 88-177 for Login pattern)

**Add CheckIPRestriction method** (after line 177):
```go
// CheckIPRestriction 检查用户IP限制
func (s *Service) CheckIPRestriction(user *models.User, clientIP string) error {
    validator := &IPValidator{}

    // Collect all allowed IPs from user and all roles
    allowedIPs := make([]string, 0)

    // Add user's IP restrictions
    if len(user.GetAllowedIPs()) > 0 {
        allowedIPs = append(allowedIPs, user.GetAllowedIPs()...)
    }

    // Add IP restrictions from all roles (OR logic per D-02)
    for _, role := range user.Roles {
        if len(role.GetAllowedIPs()) > 0 {
            allowedIPs = append(allowedIPs, role.GetAllowedIPs()...)
        }
    }

    // If no restrictions, allow all IPs
    if len(allowedIPs) == 0 {
        return nil
    }

    // Check if client IP is allowed
    allowed, err := validator.IsIPAllowed(clientIP, allowedIPs)
    if err != nil {
        s.logger.Warn("IP validation failed",
            zap.String("client_ip", clientIP),
            zap.Uint("user_id", user.ID),
            zap.Error(err),
        )
        return errors.New("IP地址验证失败")
    }

    if !allowed {
        return errors.New("您的IP地址不在允许列表中")
    }

    return nil
}
```

**Modify Login method** (insert after line 152, before token generation):
```go
// After line 152 (after user.IsActive check):
// 5. 检查IP限制 (D-16, D-17)
if err := s.CheckIPRestriction(&user, ipAddress); err != nil {
    // IP限制检查失败，记录审计日志
    s.auditLogger.Log(c.Request.Context(), &audit.LogOperationRequest{
        UserID:    user.ID,
        Username:  user.Username,
        Action:    models.ActionIPRestrictionFailed,
        Module:    models.ModuleUser,
        IPAddress: ipAddress,
        Status:    models.StatusFailure,
        ErrorMsg:  err.Error(),
    })
    return nil, err
}

// 6. 生成Token (原步骤5)
```

---

### `internal/handlers/auth_handler.go` (handler, request-response - modify existing)

**Analog:** `internal/handlers/auth_handler.go` (existing file)

**No changes needed** - Login() already passes `ipAddress` to `authService.Login()` (line 46). IP check will be done in service layer.

---

### `internal/middleware/audit.go` (middleware, event-driven - modify existing)

**Analog:** `internal/middleware/audit.go` (existing file)

**Add IP restriction action constant** (after line 76 in models/audit_log.go):
```go
const (
    ActionLogin               = "login"
    ActionLogout              = "logout"
    ActionPasswordChange      = "password_change"
    ActionIPRestrictionFailed = "ip_restriction_failed" // NEW
)
```

---

### `internal/migrations/011_add_ip_restrictions.go` (migration, batch)

**Analog:** `internal/migrations/006_multi_role_migration.go` (migration pattern)

**Database migration pattern** (source: 006_multi_role_migration.go):
```go
package migrations

import (
    "fmt"
    "log"

    "gorm.io/gorm"
)

// AddIPRestrictionsMigration 为用户和角色添加IP限制字段
type AddIPRestrictionsMigration struct{}

func (m *AddIPRestrictionsMigration) Name() string {
    return "011_add_ip_restrictions"
}

func (m *AddIPRestrictionsMigration) Up(db *gorm.DB) error {
    // Step 1: Add allowed_ips column to users table
    err := db.Exec("ALTER TABLE users ADD COLUMN allowed_ips TEXT").Error
    if err != nil {
        return fmt.Errorf("failed to add allowed_ips column to users: %w", err)
    }

    // Step 2: Add allowed_ips column to roles table
    err = db.Exec("ALTER TABLE roles ADD COLUMN allowed_ips TEXT").Error
    if err != nil {
        return fmt.Errorf("failed to add allowed_ips column to roles: %w", err)
    }

    log.Println("INFO: IP restrictions migration completed")
    return nil
}

func (m *AddIPRestrictionsMigration) Down(db *gorm.DB) error {
    // SQLite doesn't support DROP COLUMN, leave deprecated per multi-role pattern
    log.Println("WARN: Rolling back IP restrictions migration: columns will remain deprecated")
    return nil
}
```

---

### `frontend/src/types/user.ts` (types, transform - modify existing)

**Analog:** `frontend/src/types/user.ts` (existing file)

**Add allowed_ips to UserInfo** (after line 28):
```typescript
export interface UserInfo {
  id: number
  created_at: string
  updated_at: string
  username: string
  email: string
  full_name: string
  roles: Array<Role>
  allowed_ips?: string[] // NEW: IP限制列表
  is_active: boolean
  last_login_at: string | null
}
```

**Add allowed_ips to UpdateUserRequest** (after line 52):
```typescript
export interface UpdateUserRequest {
  email?: string
  full_name?: string
  role_ids?: number[]
  allowed_ips?: string[] // NEW: IP限制列表
  is_active?: boolean
}
```

**Add allowed_ips to CreateUserRequest** (after line 44):
```typescript
export interface CreateUserRequest {
  username: string
  password: string
  email?: string
  full_name?: string
  role_ids: number[]
  allowed_ips?: string[] // NEW: IP限制列表
  is_active: boolean
}
```

---

### `frontend/src/types/role.ts` (types, transform - modify existing)

**Analog:** `frontend/src/types/role.ts` (existing file)

**Add allowed_ips to RoleInfo** (after line 23):
```typescript
export interface RoleInfo {
  id: number
  created_at: string
  updated_at: string
  name: string
  description: string
  allowed_ips?: string[] // NEW: IP限制列表
  permissions?: Permission[]
}
```

**Add allowed_ips to UpdateRoleRequest** (after line 44):
```typescript
export interface UpdateRoleRequest {
  description?: string
  allowed_ips?: string[] // NEW: IP限制列表
}
```

**Add allowed_ips to CreateRoleRequest** (after line 39):
```typescript
export interface CreateRoleRequest {
  name: string
  description?: string
  allowed_ips?: string[] // NEW: IP限制列表
}
```

---

### `frontend/src/api/user.ts` (api-client, request-response - modify existing)

**Analog:** `frontend/src/api/user.ts` (existing file)

**No changes needed** - `allowed_ips` field will be automatically included in requests/responses due to type definitions.

---

### `frontend/src/api/role.ts` (api-client, request-response - modify existing)

**Analog:** `frontend/src/api/role.ts` (existing file)

**No changes needed** - `allowed_ips` field will be automatically included in requests/responses due to type definitions.

---

### `frontend/src/pages/system/users/index.tsx` (component, request-response - modify existing)

**Analog:** `frontend/src/pages/system/users/index.tsx` (existing file, lines 419-453 for form pattern)

**Add IP restriction form field** (after line 443, after role_ids Form.Item):
```tsx
<Form.Item
  name="allowed_ips"
  label="IP地址限制"
  extra="每行一个IP地址，支持格式：192.168.1.100 或 192.168.1.0/24 或 192.168.1.100-192.168.1.200"
>
  <Input.TextArea
    placeholder="例如：&#10;192.168.1.100&#10;192.168.1.0/24&#10;192.168.1.100-192.168.1.200"
    rows={4}
    onChange={(e) => {
      // Convert textarea lines to array
      const lines = e.target.value.split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0)
      form.setFieldsValue({ allowed_ips: lines })
    }}
  />
</Form.Item>
```

**Form initialization** (modify line 94-100):
```tsx
if (user) {
  form.setFieldsValue({
    username: user.username,
    email: user.email,
    full_name: user.full_name,
    role_ids: user.roles?.map(r => r.id) || [],
    allowed_ips: user.allowed_ips || [],
    is_active: user.is_active,
  })
}
```

---

## Shared Patterns

### JSON Field Storage (Text + Manual Marshal/Unmarshal)

**Source:** `internal/models/api_key.go` (lines 25-26, 86-124)

**Apply to:** `internal/models/user.go`, `internal/models/role.go`

```go
// Model field definition
AllowedIPs string `gorm:"type:text" json:"allowed_ips"`

// Helper methods
func (m *Model) GetAllowedIPs() []string {
    if m.AllowedIPs == "" {
        return []string{}
    }
    var ips []string
    _ = json.Unmarshal([]byte(m.AllowedIPs), &ips)
    return ips
}

func (m *Model) SetAllowedIPs(ips []string) error {
    data, err := json.Marshal(ips)
    if err != nil {
        return err
    }
    m.AllowedIPs = string(data)
    return nil
}
```

---

### Multi-Role OR Logic Pattern

**Source:** `internal/models/user.go` (lines 44-72, HasPermission method)

**Apply to:** `internal/auth/service.go` (CheckIPRestriction method)

```go
// Check all roles (OR logic per D-07)
for _, role := range user.Roles {
    // Check role's IP restrictions
    if len(role.GetAllowedIPs()) > 0 {
        allowedIPs = append(allowedIPs, role.GetAllowedIPs()...)
    }
}
```

---

### IP Validation Pattern

**Source:** `internal/models/api_key.go` (lines 126-150)

**Apply to:** `internal/auth/ip_validator.go`

```go
// Single IP match
if allowed == clientIP {
    return true, nil
}

// CIDR range match
if strings.Contains(allowed, "/") {
    _, ipNet, err := net.ParseCIDR(allowed)
    if err != nil {
        continue
    }
    if ipNet.Contains(clientAddr) {
        return true, nil
    }
}

// IP range match
if strings.Contains(allowed, "-") {
    parts := strings.Split(allowed, "-")
    startIP := net.ParseIP(strings.TrimSpace(parts[0]))
    endIP := net.ParseIP(strings.TrimSpace(parts[1]))
    if bytes.Compare(clientAddr, startIP) >= 0 && bytes.Compare(clientAddr, endIP) <= 0 {
        return true, nil
    }
}
```

---

### Error Response Pattern

**Source:** `pkg/response/response.go` (lines 117-140)

**Apply to:** All error returns in IP restriction logic

```go
// Use Chinese error messages for user-facing errors
if !allowed {
    return errors.New("您的IP地址不在允许列表中")
}

// Use response.GinError for handler errors
response.GinError(c, response.CodeInvalidRequest, err.Error())
```

---

### Audit Logging Pattern

**Source:** `internal/middleware/audit.go` (lines 111-149, AuditLogin method)

**Apply to:** IP restriction failure logging in auth service

```go
s.auditLogger.Log(c.Request.Context(), &audit.LogOperationRequest{
    UserID:    user.ID,
    Username:  user.Username,
    Action:    models.ActionIPRestrictionFailed,
    Module:    models.ModuleUser,
    IPAddress: ipAddress,
    Status:    models.StatusFailure,
    ErrorMsg:  err.Error(),
})
```

---

### Frontend TextArea to Array Pattern

**Source:** Frontend form patterns (lines 366-367 in roles/index.tsx)

**Apply to:** `frontend/src/pages/system/users/index.tsx`, `frontend/src/pages/system/roles/index.tsx`

```tsx
<Input.TextArea
  placeholder="支持格式：192.168.1.100 或 192.168.1.0/24 或 192.168.1.100-192.168.1.200"
  rows={4}
  onChange={(e) => {
    const lines = e.target.value.split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)
    form.setFieldsValue({ allowed_ips: lines })
  }}
/>
```

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| N/A | — | — | All files have analogs in existing codebase |

---

## Metadata

**Analog search scope:**
- `internal/models/` - User, Role, APIKey models
- `internal/auth/` - Service, handler patterns
- `internal/middleware/` - Audit middleware
- `internal/migrations/` - Migration patterns
- `pkg/response/` - Error handling patterns
- `frontend/src/types/` - TypeScript type definitions
- `frontend/src/api/` - API client patterns
- `frontend/src/pages/system/` - Form patterns

**Files scanned:** 15
**Pattern extraction date:** 2026-04-27

**Key patterns identified:**
1. **JSON field storage**: Use `gorm:"type:text"` + manual `json.Marshal/Unmarshal` (from APIKey model)
2. **Multi-role OR logic**: Iterate all roles and merge IP lists (from User.HasPermission)
3. **IP validation**: Use Go `net.ParseIP()`, `net.ParseCIDR()`, `bytes.Compare()` (from APIKey.IsIPAllowed)
4. **Client IP extraction**: Use `c.ClientIP()` (from auth_handler.go line 42)
5. **Audit logging**: Add new action constant, log with audit service (from audit middleware)
6. **Migration pattern**: Add columns with ALTER TABLE, handle rollback gracefully (from 006 migration)
7. **Frontend IP input**: TextArea with line-by-line conversion to string array (form patterns)
