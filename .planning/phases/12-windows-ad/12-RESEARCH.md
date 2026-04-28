# Phase 12: Windows AD域控认证 - Research

**Researched:** 2026-04-28
**Domain:** Windows Active Directory LDAP authentication integration
**Confidence:** HIGH

## Summary

This phase integrates Windows Active Directory domain authentication into the existing Record V2 system, supporting both LDAP (port 389) and LDAPS (port 636) connections with a configurable local/AD authentication mode switch. The implementation builds on 5 validated spikes covering go-ldap library usage, LDAPS security, authentication switch architecture, AD user mapping, and configuration validation. The system uses the strategy pattern to support local and AD authenticators with configuration hot-reloading, maintaining the existing SM4 password encryption for frontend compatibility.

**Primary recommendation:** Implement authentication switching using the strategy pattern with `github.com/go-ldap/ldap/v3` library, extending the User model with AD fields while maintaining backward compatibility with local authentication.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** System supports only **local** and **ad** authentication modes, removing hybrid mode
- **D-02:** System defaults to **local mode** (safest, no AD interaction)
- **D-03:** Authentication mode is system-level configuration, switching affects all users
- **D-04:** **AD mode does NOT degrade** - AD authentication failure returns error directly, no fallback to local auth
- **D-05:** **local mode** uses only local account password authentication
- **D-06:** All accounts managed uniformly, UI does not display source distinction
- **D-07:** All accounts require local password (AD users can have system-generated random password)
- **D-08:** No auth_source field needed, completely transparent management
- **D-09:** Use **simple form-based** guidance flow (configuration page + test connection button)
- **D-10:** Configuration fields: AD server address, port, BindDN, password, BaseDN, TLS options
- **D-11:** Provide test connection button calling AD connectivity test API
- **D-12:** Display **inline warning icon** (⚠️) when using LDAP port 389
- **D-13:** Risk confirmation is **passive logging**: record warning display in audit log, no explicit user confirmation needed
- **D-14:** Warning content: explain port 389 plaintext transmission risk, recommend LDAPS port 636
- **D-15:** Configuration changes **automatically validate** AD connectivity
- **D-16:** Validation failure **blocks save** and displays specific error reason
- **D-17:** Switching to AD mode requires successful AD configuration validation first
- **D-18:** Display **friendly prompt** on AD connection failure: "Unable to connect to AD server, please check network and configuration"
- **D-19:** Detailed error information logged to backend (LDAP error codes, stack traces)
- **D-20:** AD authentication failure clearly indicates reason: account not exists vs password error vs server connection failed
- **D-21:** Extend users table with AD attributes, excluding auth_source field (per D-08)
- **D-22:** Required fields: ad_username, ad_dn, ad_guid, last_ad_login, ad_department, ad_upn
- **D-23:** All AD fields nullable (empty for local users)

### Claude's Discretion
- Specific error prompt wording (Chinese-friendly acceptable)
- Configuration page UI layout and styling
- Test connection API response format
- Audit log specific fields and format

### Deferred Ideas (OUT OF SCOPE)
- AD group→role mapping (future extension)
- Periodic user status sync (sync on login only)
- Separate AD user management interface (unified user list)
- Support multiple AD server configurations
- AD user password modification (prompt to contact domain administrator)

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| D-01 | System supports only local and AD modes | Spike 003 validates strategy pattern for mode switching |
| D-02 | Default to local mode | Config structure in spike 003 shows default mode setting |
| D-03 | System-level authentication mode | Spike 003 architecture shows config-based mode switching |
| D-04 | AD mode no degradation | Spike 003 shows direct authenticator routing without fallback |
| D-05 | Local mode uses local passwords | Existing AuthService in internal/auth/service.go already implements |
| D-06 | Unified account management | Spike 004 shows transparent user management without auth_source display |
| D-07 | All accounts require local password | Spike 004 shows AD users with random generated passwords |
| D-08 | No auth_source field needed | Spike 004 validates unified user model without source distinction |
| D-09 | Simple form-based guidance | Spike 005 provides configuration validation API design |
| D-10 | Configuration fields defined | Spike 003 ADConfig structure defines all required fields |
| D-11 | Test connection button | Spike 005 validation API provides test-connection endpoint |
| D-12 | Port 389 warning icon | Spike 002 documents security risks of port 389 plaintext transmission |
| D-13 | Passive risk logging | Existing audit service in internal/services/audit/ supports logging |
| D-14 | Warning content defined | Spike 002 security research provides warning text |
| D-15 | Automatic AD validation | Spike 005 provides 4-layer validation architecture |
| D-16 | Validation blocks save | Spike 005 shows validation-blocking pattern in config updates |
| D-17 | AD validation before mode switch | Spike 003 architecture requires validation before switching |
| D-18 | Friendly connection failure prompts | Spike 005 provides error formatting with user-friendly messages |
| D-19 | Detailed backend error logging | Spike 005 shows detailed error logging with LDAP codes |
| D-20 | Specific AD authentication failure reasons | Spike 001 shows different error types for user not found, bad password, connection failure |
| D-21 | Extend users table with AD attributes | Spike 004 provides User model extension SQL migration |
| D-22 | AD field names defined | Spike 004 user mapping defines exact field names and types |
| D-23 | AD fields nullable | Spike 004 migration SQL shows nullable AD fields |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| User authentication (local) | API / Backend | Database | Existing AuthService handles bcrypt password validation |
| User authentication (AD) | API / Backend | External AD Server | AD authenticator connects to external AD server via LDAP/LDAPS |
| Authentication mode routing | API / Backend | — | Strategy pattern in AuthService routes to appropriate authenticator |
| User session management | API / Backend | Database | JWT tokens stored in database sessions table |
| Configuration management | API / Backend | File System | Config hot-reloading from config.yaml with file watcher |
| Password encryption (SM4) | Browser / Client | API / Backend | Frontend encrypts with SM4, backend decrypts before auth |
| AD connectivity validation | API / Backend | External AD Server | Validation API tests connection to AD server |
| User permissions | Database | API / Backend | Local users_roles table stores permission mappings |
| Audit logging | API / Backend | Database | All auth mode switches logged to audit_logs table |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/go-ldap/ldap/v3` | v3.4.6 [VERIFIED: go.mod verification] | AD LDAP authentication | Official Go LDAP library, actively maintained, used by HashiCorp Vault and Kubernetes |
| `github.com/gin-gonic/gin` | v1.11.0 | HTTP routing and API handlers | Existing project framework |
| `gorm.io/gorm` | v1.30.0 | ORM for database operations | Existing project ORM |
| `gorm.io/driver/sqlite` | v1.6.0 | SQLite database driver | Existing project database |
| `github.com/spf13/viper` | v1.19.0 | Configuration management | Existing config system supports hot-reloading |
| `go.uber.org/zap` | v1.27.0 | Structured logging | Existing project logger |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/tjfoc/gmsm` | v1.4.1 | SM4 encryption for passwords | Existing SM4 implementation, used for both local and AD auth |
| `golang.org/x/crypto` | v0.49.0 | Bcrypt password hashing | Existing local password hashing |

### Frontend
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `antd` | v6.0.0 | UI components for configuration page | Existing UI framework |
| `sm-crypto` | v0.4.0 | SM4 password encryption on frontend | Existing SM4 client-side encryption |
| `axios` | v1.7.0 | HTTP client for API calls | Existing API client |
| `react` | v19.2.0 | Frontend framework | Existing framework |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-ldap/v3 | No actively maintained alternatives | v2/v1 deprecated, v3 is the only viable option |

**Installation:**
```bash
# Backend dependencies (most already installed)
go get github.com/go-ldap/ldap/v3@latest

# Frontend dependencies (already installed)
npm install sm-crypto@0.4.0
```

**Version verification:** Verified against go.mod showing Go 1.25.0 with existing dependencies. The `github.com/go-ldap/ldap/v3` library at v3.4.6 is the current stable release [VERIFIED: Web search 2025].

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React 19)                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Login Page (SM4 encryption)                              │  │
│  │  - Encrypts password with SM4 before sending             │  │
│  │  - Uses existing sm-crypto library                       │  │
│  └────────────────────┬─────────────────────────────────────┘  │
│                       │                                         │
└───────────────────────┼─────────────────────────────────────────┘
                        │ POST /api/auth/login
                        │ {username, encrypted_password}
┌───────────────────────▼─────────────────────────────────────────┐
│                    Backend API (Gin)                            │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Auth Handler                                             │  │
│  │  - Decrypts SM4 password                                 │  │
│  │  - Calls AuthService.Login()                             │  │
│  └────────────────────┬─────────────────────────────────────┘  │
│                       │                                         │
│  ┌────────────────────▼─────────────────────────────────────┐  │
│  │  AuthService (Strategy Pattern)                          │  │
│  │  - Reads auth.mode from config                           │  │
│  │  - Routes to: LocalAuthenticator OR ADAuthenticator      │  │
│  └────────────┬────────────────────────────────┬─────────────┘  │
│               │                                │                  │
│      ┌────────▼─────────┐          ┌─────────▼────────────┐    │
│      │ LocalAuthenticator│          │  ADAuthenticator     │    │
│      ├──────────────────┤          ├──────────────────────┤    │
│      │ - Validates bcrypt│          │ - Connects to AD     │    │
│      │   password hash   │          │ - Binds with user    │    │
│      │ - Checks user     │          │   credentials        │    │
│      │   status          │          │ - Fetches user attrs │    │
│      └────────┬─────────┘          │ - Creates/updates    │    │
│               │                    │   local user mapping  │    │
│               │                    └──────────┬───────────┘    │
│               │                               │                  │
│  ┌────────────▼───────────────────────────────▼─────────────┐  │
│  │  Database (SQLite + GORM)                                 │  │
│  │  - users table (extended with AD fields)                 │  │
│  │  - users_roles table (permissions)                       │  │
│  │  - sessions table (JWT tokens)                           │  │
│  │  - audit_logs table (auth mode switches)                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│               │                                                │
│               │ (AD authenticator only)                        │
│  ┌────────────▼───────────────────────────────────────────┐  │
│  │  External Active Directory Server                        │  │
│  │  - LDAP port 389 (with StartTLS)                        │  │
│  │  - LDAPS port 636 (recommended)                         │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
internal/
├── auth/
│   ├── service.go              # Main auth service (modify for AD routing)
│   ├── authenticator.go        # Authenticator interface (NEW)
│   ├── local_auth.go           # Local auth implementation (extract existing)
│   ├── ad_auth.go              # AD auth implementation (NEW)
│   ├── ad_config.go            # AD configuration structures (NEW)
│   ├── ad_validator.go         # AD config validator (NEW)
│   ├── ip_validator.go         # Existing IP validator
│   ├── ip_restriction.go       # Existing IP restriction
│   └── token_service.go        # Existing JWT token service
├── models/
│   ├── user.go                 # Extend with AD fields
│   ├── audit_log.go            # Existing audit log model
│   └── ...                     # Other existing models
├── handlers/
│   ├── auth_handler.go         # Add AD config endpoints
│   └── admin_handler.go        # Add auth mode switching
├── migrations/
│   ├── 013_add_ad_fields.go    # Add AD fields to users table
│   └── helpers.go              # Existing migration helpers
├── services/
│   ├── audit/
│   │   └── audit_log_service.go # Existing audit service
│   └── ...                      # Other existing services
└── config/
    ├── config.go               # Add ADConfig to AuthConfig
    └── ...                     # Other existing config

frontend/src/
├── pages/
│   ├── system/
│   │   ├── auth-config/        # NEW: AD configuration page
│   │   │   └── index.tsx
│   │   ├── users/              # Modify: no auth source display
│   │   └── ...
│   └── auth/
│       └── Login.tsx           # No changes needed (SM4 already works)
├── api/
│   ├── auth.ts                 # Add AD config APIs
│   └── ...
└── ...
```

### Pattern 1: Strategy Pattern for Authentication Switching

**What:** Define an Authenticator interface that both LocalAuthenticator and ADAuthenticator implement, allowing AuthService to route authentication requests based on configuration without coupling to specific implementations.

**When to use:** When you need multiple authentication methods that can be switched at runtime without code changes.

**Example:**
```go
// Source: Spike 003 auth-switch-architecture
// internal/auth/authenticator.go

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

// AuthService 使用策略模式路由认证请求
type AuthService struct {
    config       *config.AuthConfig
    localAuth    Authenticator
    adAuth       Authenticator
    tokenService *TokenService
    logger       *zap.Logger
}

func (s *AuthService) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
    var authenticator Authenticator
    
    // 根据配置模式选择认证器
    switch s.config.Mode {
    case "local":
        authenticator = s.localAuth
    case "ad":
        authenticator = s.adAuth
    default:
        return nil, errors.New("无效的认证模式")
    }
    
    return authenticator.Login(req, ipAddress, userAgent)
}
```

### Pattern 2: Four-Layer AD Configuration Validation

**What:** Validate AD configuration in four progressive layers (format → network → authentication → functionality), failing fast at each layer to provide clear error messages.

**When to use:** When validating external service configurations that require network connectivity and authentication testing.

**Example:**
```go
// Source: Spike 005 ad-config-validation
// internal/auth/ad_validator.go

type ADConfigValidator struct {
    logger *zap.Logger
}

func (v *ADConfigValidator) Validate(config *ADConfig) *ADConfigValidationResult {
    result := &ADConfigValidationResult{
        Valid:  false,
        Level:  0,
        Errors: []string{},
    }
    
    // Layer 1: Format validation (no network calls)
    if err := v.validateFormat(config); err != nil {
        result.Errors = append(result.Errors, err.Error())
        return result
    }
    result.Level = 1
    
    // Layer 2: Network validation (TCP connection)
    conn, err := v.testConnection(config)
    if err != nil {
        result.Errors = append(result.Errors, v.formatConnectionError(err))
        return result
    }
    defer conn.Close()
    result.Level = 2
    
    // Layer 3: Authentication validation (bind test)
    if err := v.testBind(conn, config); err != nil {
        result.Errors = append(result.Errors, v.formatBindError(err))
        return result
    }
    result.Level = 3
    
    // Layer 4: Functionality validation (user search)
    if err := v.testFunctionality(conn, config); err != nil {
        result.Warnings = append(result.Warnings, "功能测试警告: "+err.Error())
    }
    result.Level = 4
    
    result.Valid = true
    return result
}
```

### Pattern 3: AD User Mapping with Local Shadow Records

**What:** Create local user records for AD-authenticated users to maintain existing permission and role systems while storing AD attributes for synchronization.

**When to use:** When integrating external authentication with local authorization systems.

**Example:**
```go
// Source: Spike 004 ad-user-management
// internal/auth/ad_auth.go

func (a *ADAuthenticator) findOrCreateLocalUser(adUser *ADUser) (*models.User, error) {
    // First, try to find existing user by username
    var user models.User
    err := a.db.Where("username = ?", adUser.Username).First(&user).Error
    
    if err == nil {
        // Found existing user, update AD information
        a.updateADInfo(&user, adUser)
        return &user, nil
    }
    
    if !errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, err
    }
    
    // Not found, create new user
    user = models.User{
        Username:     adUser.Username,
        Email:        adUser.Email,
        FullName:     adUser.DisplayName,
        // No auth_source field per D-08
        ADUsername:   adUser.Username,
        ADDN:         adUser.DN,
        ADGUID:       adUser.ObjectGUID,
        ADDepartment: adUser.Department,
        ADUPN:        adUser.UserPrincipalName,
        IsActive:     true,
    }
    
    // Generate random password (AD users won't use it)
    randomPassword := utils.GenerateRandomPassword(32)
    if err := user.SetPassword(randomPassword); err != nil {
        return nil, err
    }
    
    // Assign default role
    if err := a.assignDefaultRole(&user); err != nil {
        return nil, err
    }
    
    if err := a.db.Create(&user).Error; err != nil {
        return nil, err
    }
    
    return &user, nil
}
```

### Anti-Patterns to Avoid
- **Hardcoded authentication logic with if/else chains:** Difficult to extend and test. Use strategy pattern instead.
- **Storing AD admin password in config files:** Security risk. Use environment variables.
- **Skipping AD configuration validation before mode switch:** Can render system unusable. Always validate before switching.
- **Creating AD users without local shadow records:** Breaks existing permission system. Always create local user mapping.
- **Logging sensitive AD information in plaintext:** Security risk. Sanitize logs and avoid logging passwords or BindDNs.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| LDAP protocol implementation | Custom LDAP client | `github.com/go-ldap/ldap/v3` | LDAP v3 is complex, includes binary encoding, BER/DER, security edge cases |
| Password encryption for frontend | Custom crypto | Existing SM4 implementation (`sm-crypto`, `github.com/tjfoc/gmsm`) | Already integrated, tested, and working for local auth |
| Configuration hot-reloading | Custom file watcher | Viper's built-in config watching | Viper already supports automatic config reloading |
| JWT token management | Custom token logic | Existing SM4TokenService | Already handles JWT generation, validation, revocation |
| Audit logging | Custom audit system | Existing AuditLogService | Already handles structured logging, sanitization, async queue |
| Input validation | Custom validators | Existing validation patterns | Project already has validation patterns for IP, CIDR, etc. |

**Key insight:** The existing codebase already provides most infrastructure (JWT, audit, config, encryption). Only the AD-specific logic (LDAP connection, user mapping, config validation) needs to be built.

## Common Pitfalls

### Pitfall 1: LDAP Injection Through Unescaped User Input
**What goes wrong:** User-provided usernames are directly interpolated into LDAP search filters without escaping, allowing injection attacks that bypass authentication or leak information.

**Why it happens:** Developers treat LDAP filters like simple strings, not realizing special characters (`*`, `(`, `)`, `\`, etc.) have meaning in LDAP search syntax.

**How to avoid:** Always use `ldap.EscapeFilter()` for user-provided input in search filters:
```go
// WRONG
filter := fmt.Sprintf("(sAMAccountName=%s)", username)

// CORRECT
filter := fmt.Sprintf("(sAMAccountName=%s)", ldap.EscapeFilter(username))
```

**Warning signs:** Username contains special characters and authentication behaves unexpectedly or returns wrong user information.

### Pitfall 2: Insecure TLS Configuration for LDAPS
**What goes wrong:** LDAPS connections either fail with certificate errors or use insecure settings like `InsecureSkipVerify=true` in production.

**Why it happens:** Self-signed certificates in dev environments, missing CA certificates, or copying dev config to production.

**How to avoid:** 
- Use `tls.VersionTLS12` as minimum TLS version
- Properly configure CA certificates in production
- Use environment-specific configs (dev can skip verify, prod must verify)
- Document certificate requirements for setup

**Warning signs:** TLS handshake failures, x509 certificate errors, or successful connections only when `InsecureSkipVerify=true`.

### Pitfall 3: Authentication Mode Switching Without Validation
**What goes wrong:** Admin switches to AD mode with invalid configuration, rendering all users unable to log in.

**Why it happens:** Configuration update doesn't validate AD connectivity before applying the change.

**How to avoid:** 
- Always validate AD config before switching to AD mode (Spike 005 pattern)
- Use transaction-style config updates (validate → apply → rollback on failure)
- Provide "test connection" button before mode switch
- Keep old config in memory until validation succeeds

**Warning signs:** Mode switch succeeds but login immediately fails for all users.

### Pitfall 4: Forgetting to Create Local User Records for AD Users
**What goes wrong:** AD users can authenticate but have no permissions, roles, or appear missing from user management interface.

**Why it happens:** AD authenticator returns success without creating/finding local shadow record for permission mapping.

**How to avoid:** 
- Always call `findOrCreateLocalUser()` after successful AD authentication
- Store AD attributes (DN, GUID, department, UPN) in local user record
- Assign default role to newly created AD users

**Warning signs:** AD-authenticated users can log in but see blank pages or "access denied" everywhere.

### Pitfall 5: Breaking Existing Local Authentication During AD Integration
**What goes wrong:** Local users can no longer log in after AD integration due to authentication service refactoring mistakes.

**Why it happens:** Refactoring breaks existing LocalAuthenticator logic or SM4 password decryption.

**How to avoid:** 
- Extract existing auth logic to LocalAuthenticator without changing behavior
- Write tests for local authentication before and after refactoring
- Keep local auth path completely separate from AD path
- Test both modes end-to-end before committing

**Warning signs:** Existing local user logins fail after AD integration changes.

### Pitfall 6: Not Handling AD User Account Status
**What goes wrong:** Disabled or expired AD accounts can still authenticate through the system.

**Why it happens:** AD authenticator doesn't check `userAccountControl` attribute or account expiration.

**How to avoid:** 
- Parse `userAccountControl` flag (check ACCOUNTDISABLE bit 0x0002)
- Check account expiration if AD enforces password expiration
- Return specific error for disabled accounts
- Log authentication attempts for disabled accounts

**Warning signs:** Users whose AD accounts are disabled can still log in to the system.

### Pitfall 7: SM4 Password Decryption Failures
**What goes wrong:** Legitimate login attempts fail with "decrypt failure" errors, especially for AD authentication.

**Why it happens:** SM4 secret mismatch between frontend and backend, or missing SM4 secret in AD authentication flow.

**How to avoid:** 
- Use same SM4 secret for both local and AD authentication
- Validate SM4 secret configuration at startup
- Handle decryption failures gracefully with clear error messages
- Test SM4 encryption/decryption with AD authentication flow

**Warning signs:** "decrypt failure" errors increase in logs after AD integration.

### Pitfall 8: Database Migration Failures on Existing Systems
**What goes wrong:** Migration to add AD columns fails because columns already exist or constraints conflict.

**Why it happens:** Migration doesn't check for existing columns before adding them.

**How to avoid:** 
- Use `columnExists()` helper before adding columns (existing pattern)
- Make migrations idempotent (safe to run multiple times)
- Log migration steps clearly for debugging
- Test migration on copy of production database

**Warning signs:** Migration errors on column already exists or constraint violations.

## Code Examples

### AD Connection with LDAPS
```go
// Source: Spike 001 go-ldap-ad-auth + Spike 002 ldaps-security
// internal/auth/ad_auth.go

import (
    "crypto/tls"
    "github.com/go-ldap/ldap/v3"
)

func (a *ADAuthenticator) connectAD() (*ldap.Conn, error) {
    var conn *ldap.Conn
    var err error
    
    if a.config.UseTLS {
        // LDAPS mode (port 636) - recommended for production
        tlsConfig := &tls.Config{
            ServerName:         extractHostname(a.config.Server),
            InsecureSkipVerify: a.config.InsecureSkipVerify, // false in production
            MinVersion:         tls.VersionTLS12,
        }
        conn, err = ldap.DialTLS("tcp", a.config.Server, tlsConfig)
    } else {
        // LDAP mode (port 389) - internal networks only
        conn, err = ldap.Dial("tcp", a.config.Server)
        if err == nil {
            // Upgrade to TLS with StartTLS
            err = conn.StartTLS(&tls.Config{
                ServerName: extractHostname(a.config.Server),
                MinVersion: tls.VersionTLS12,
            })
        }
    }
    
    if err != nil {
        return nil, fmt.Errorf("AD连接失败: %w", err)
    }
    
    return conn, nil
}
```

### AD User Authentication Flow
```go
// Source: Spike 001 go-ldap-ad-auth
// internal/auth/ad_auth.go

func (a *ADAuthenticator) AuthenticateUser(username, password string) (*ADUser, error) {
    // Step 1: Connect to AD server
    conn, err := a.connectAD()
    if err != nil {
        return nil, fmt.Errorf("连接AD服务器失败: %w", err)
    }
    defer conn.Close()
    
    // Step 2: Bind as admin to search for user
    err = conn.Bind(a.config.BindDN, a.config.Password)
    if err != nil {
        return nil, fmt.Errorf("管理员绑定失败: %w", err)
    }
    
    // Step 3: Search for user DN (prevent LDAP injection)
    searchRequest := ldap.NewSearchRequest(
        a.config.BaseDN,
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        1, 0, false,
        fmt.Sprintf("(&(objectClass=user)(sAMAccountName=%s))", ldap.EscapeFilter(username)),
        []string{"dn", "sAMAccountName", "mail", "displayName", "userAccountControl", "objectGUID", "department", "userPrincipalName"},
        nil,
    )
    
    sr, err := conn.Search(searchRequest)
    if err != nil {
        return nil, fmt.Errorf("搜索用户失败: %w", err)
    }
    
    if len(sr.Entries) == 0 {
        return nil, errors.New("域控账号不存在")
    }
    
    userDN := sr.Entries[0].DN
    adUser := a.parseLDAPEntry(sr.Entries[0])
    
    // Step 4: Check if account is disabled
    if adUser.IsDisabled() {
        return nil, errors.New("域控账号已禁用")
    }
    
    // Step 5: Bind as user to authenticate
    err = conn.Bind(userDN, password)
    if err != nil {
        return nil, fmt.Errorf("域控密码错误")
    }
    
    return adUser, nil
}
```

### Configuration Extension for AD
```go
// Source: Spike 003 auth-switch-architecture
// internal/config/config.go

type AuthConfig struct {
    // Existing fields
    SM4Secret              string        `mapstructure:"sm4_secret" json:"sm4_secret"`
    AccessTokenDuration    time.Duration `mapstructure:"access_token_duration" json:"access_token_duration"`
    // ... other existing fields
    
    // NEW: Authentication mode
    Mode string `mapstructure:"mode" json:"mode"` // local, ad
    
    // NEW: AD configuration
    AD ADAuthConfig `mapstructure:"ad" json:"ad"`
}

type ADAuthConfig struct {
    Server   string `mapstructure:"server" json:"server"`         // ad.example.com:636
    BindDN   string `mapstructure:"bind_dn" json:"bind_dn"`       // cn=admin,cn=users,dc=example,dc=com
    Password string `mapstructure:"password" json:"password"`     // From environment variable
    BaseDN   string `mapstructure:"base_dn" json:"base_dn"`       // dc=example,dc=com
    UseTLS   bool   `mapstructure:"use_tls" json:"use_tls"`       // true for LDAPS (636), false for LDAP (389)
    
    // Connection pool settings
    PoolSize int `mapstructure:"pool_size" json:"pool_size"`     // Default: 10
    
    // Timeout settings
    DialTimeout    int `mapstructure:"dial_timeout" json:"dial_timeout"`        // Default: 10 seconds
    RequestTimeout int `mapstructure:"request_timeout" json:"request_timeout"`  // Default: 30 seconds
    
    // Test mode (for development only)
    InsecureSkipVerify bool `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify"` // Default: false
}
```

### Database Migration for AD Fields
```go
// Source: Spike 004 ad-user-management
// internal/migrations/013_add_ad_fields.go

package migrations

import (
    "log"
    
    "gorm.io/gorm"
)

// AddADFieldsMigration 添加AD相关字段到users表
type AddADFieldsMigration struct{}

func (m *AddADFieldsMigration) Name() string {
    return "013_add_ad_fields"
}

func (m *AddADFieldsMigration) Up(db *gorm.DB) error {
    // Add AD fields (nullable for local users)
    fields := []struct {
        column string
        typ    string
    }{
        {"ad_username", "VARCHAR(100)"},
        {"ad_dn", "VARCHAR(255)"},
        {"ad_guid", "CHAR(36)"},
        {"ad_department", "VARCHAR(100)"},
        {"ad_upn", "VARCHAR(200)"},
        {"last_ad_login", "DATETIME"},
    }
    
    for _, field := range fields {
        exists, err := columnExists(db, "users", field.column)
        if err != nil {
            return err
        }
        if !exists {
            if err := db.Exec("ALTER TABLE users ADD COLUMN " + field.column + " " + field.typ).Error; err != nil {
                return err
            }
            log.Println("INFO: Added column " + field.column + " to users table")
        }
    }
    
    // Create index on ad_guid for faster AD user lookups
    db.Exec("CREATE INDEX IF NOT EXISTS idx_users_ad_guid ON users(ad_guid)")
    
    log.Println("INFO: AD fields migration completed")
    return nil
}

func (m *AddADFieldsMigration) Down(db *gorm.DB) error {
    // SQLite doesn't support DROP COLUMN, leave deprecated
    log.Println("WARN: Rolling back AD fields migration: columns will remain deprecated")
    return nil
}
```

### AD Configuration Validation API
```go
// Source: Spike 005 ad-config-validation
// internal/handlers/admin_handler.go

func (h *AdminHandler) ValidateADConfig(c *gin.Context) {
    var req ADConfig
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "请求参数错误: " + err.Error()})
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

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded single authentication method | Strategy pattern with pluggable authenticators | Phase 12 | Enables runtime mode switching without code changes |
| Manual account creation for AD users | Automatic user provisioning on first login | Phase 12 | Reduces admin overhead, improves UX |
| Static configuration requiring restart | Hot-reloading configuration via Viper | Phase 12 | Zero-downtime authentication mode changes |
| Basic error messages | Layered validation with specific error messages | Phase 12 | Faster troubleshooting, better UX |

**Deprecated/outdated:**
- `github.com/go-ldap/ldap` (v1/v2): Use v3 only
- Direct LDAP queries without escaping: Always use `ldap.EscapeFilter()`
- Storing AD admin passwords in config files: Use environment variables

## Assumptions Log

> List all claims tagged `[ASSUMED]` in this research. The planner and discuss-phase use this section to identify decisions that need user confirmation before execution.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Frontend SM4 encryption works without modification for AD authentication | Standard Stack | If SM4 encryption is tightly coupled to local auth, may need frontend changes |
| A2 | Existing audit service can log authentication mode switches without modification | Architecture Patterns | If audit schema doesn't support new action types, migration needed |
| A3 | Viper config hot-reloading works for nested structures like AuthConfig | Architecture Patterns | If hot-reload has bugs with nested structs, may need restart for config changes |
| A4 | SQLite performance is adequate for AD user lookup queries | Architecture Patterns | If AD user base is large (>10,000), may need performance optimization |
| A5 | Go 1.25.0 is compatible with go-ldap/v3 without build issues | Standard Stack | If there's a version conflict, may need Go version upgrade or library version pinning |
| A6 | Existing session management works for AD-authenticated users | Architecture Patterns | If sessions are tied to local auth, may need refactoring for AD users |
| A7 | Role-based permission system works identically for AD and local users | Architecture Patterns | If permissions are coupled to auth source, additional changes needed |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions

1. **AD Server Certificate Management**
   - What we know: LDAPS requires TLS certificates, production should validate certificates
   - What's unclear: How customers will handle self-signed certificates in internal environments
   - Recommendation: Support both validated and self-signed certificates with configuration flag and appropriate warnings

2. **Default Role Assignment for New AD Users**
   - What we know: AD users need local shadow records with roles for permissions
   - What's unclear: Which default role should be assigned to newly created AD users
   - Recommendation: Make default role configurable, default to a read-only role for security

3. **AD User Synchronization Frequency**
   - What we know: User attributes sync on login per deferred ideas
   - What's unclear: Whether admins need manual "sync now" functionality or scheduled sync
   - Recommendation: Implement manual sync button in user management, defer scheduled sync to future phase

4. **Password Change UI for AD Users**
   - What we know: AD users cannot change passwords through the system per deferred ideas
   - What's unclear: How to handle password change requests from AD users in the UI
   - Recommendation: Hide password change for AD users or show message "Please contact your domain administrator"

5. **Migration Path for Existing Local Users**
   - What we know: System will support both local and AD modes
   - What's unclear: Whether existing local users need AD accounts created or can coexist
   - Recommendation: Support coexistence - local users can still login in local mode, AD users created on first login

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25.0 | Backend | ✓ | 1.25.0 | — |
| SQLite | Database | ✓ | Modern version via modernc.org/sqlite | — |
| Active Directory Server | AD Authentication | ✗ | — | Use local mode for development/testing |
| Network connectivity to AD | AD Authentication | ✗ | — | Use local mode for development/testing |
| TLS certificate (for LDAPS) | Production AD | ✗ | — | Use StartTLS or test mode for development |

**Missing dependencies with no fallback:**
- Active Directory server for production AD authentication
- Network connectivity to AD server
- TLS certificate for production LDAPS

**Missing dependencies with fallback:**
- Development/testing can use local authentication mode without AD server
- Test mode allows skipping certificate validation for development

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `github.com/stretchr/testify` v1.11.1 + Go testing |
| Config file | None - uses Go built-in testing |
| Quick run command | `go test ./internal/auth/... -v -run TestAD` |
| Full suite command | `go test ./... -v` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-01 | System supports local and AD modes | unit | `go test ./internal/auth/... -run TestAuthService_ModeSwitch` | ❌ Wave 0 |
| D-04 | AD mode does not degrade on failure | unit | `go test ./internal/auth/... -run TestADAuth_NoFallback` | ❌ Wave 0 |
| D-15 | Config validates AD connectivity | integration | `go test ./internal/auth/... -run TestADConfigValidator` | ❌ Wave 0 |
| D-16 | Validation blocks save on failure | unit | `go test ./internal/handlers/... -run TestAdminHandler_ValidationBlocks` | ❌ Wave 0 |
| D-21 | Users table extended with AD fields | unit | `go test ./internal/migrations/... -run TestAddADFieldsMigration` | ❌ Wave 0 |
| D-23 | AD fields are nullable | unit | `go test ./internal/models/... -run TestUser_ADFieldsNullable` | ❌ Wave 0 |
| D-20 | AD auth shows specific failure reasons | unit | `go test ./internal/auth/... -run TestADAuth_SpecificErrors` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/auth/... -v`
- **Per wave merge:** `go test ./... -v`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/auth/ad_auth_test.go` - AD authentication logic tests
- [ ] `internal/auth/ad_validator_test.go` - AD configuration validation tests
- [ ] `internal/auth/local_auth_test.go` - Extracted local authenticator tests
- [ ] `internal/auth/authenticator_test.go` - Authenticator interface tests
- [ ] `internal/migrations/013_test.go` - Migration tests for AD fields
- [ ] `internal/models/user_ad_test.go` - User model AD field tests
- [ ] `internal/handlers/admin_handler_ad_test.go` - Admin AD configuration API tests
- [ ] Framework install: Already installed (`go test` is built-in)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | go-ldap/v3 library with TLS 1.2+ |
| V3 Session Management | yes | Existing JWT token service (SM4TokenService) |
| V4 Access Control | yes | Existing role-based permission system (users_roles table) |
| V5 Input Validation | yes | `ldap.EscapeFilter()` for LDAP queries, existing password validator |
| V6 Cryptography | yes | TLS 1.2+ for LDAPS, existing SM4 for frontend encryption, bcrypt for local passwords |

### Known Threat Patterns for Windows AD Authentication

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| LDAP injection | Tampering | Use `ldap.EscapeFilter()` for all user input in LDAP queries |
| Credential theft (plaintext LDAP) | Information Disclosure | Enforce LDAPS (port 636) or StartTLS, warn on port 389 use |
| Man-in-the-middle (TLS downgrade) | Tampering | Force TLS 1.2+, validate certificates in production |
| Password replay attacks | Spoofing | Use JWT tokens with short expiration, existing session management |
| Unauthorized AD access | Elevation of Privilege | Validate AD configuration before mode switch, audit log all auth attempts |
| DoS via slow AD response | Denial of Service | Implement connection timeouts, connection pooling, rate limiting |
| Information disclosure via error messages | Information Disclosure | Return generic errors to frontend, log detailed errors to backend |

### Additional Security Considerations

1. **Password Security:**
   - Frontend: SM4 encryption (existing implementation, reused for AD)
   - Backend: Decrypt SM4, then send to AD over TLS (no plaintext storage)
   - Local: Bcrypt hashing for shadow records (existing implementation)

2. **TLS Configuration:**
   - Minimum TLS version: 1.2
   - Certificate validation: Required in production, skippable in test mode
   - Cipher suites: Use Go's default secure cipher suites

3. **Logging Security:**
   - Never log plaintext passwords or BindDN credentials
   - Sanitize AD server details in user-facing error messages
   - Log all authentication mode switches with admin user ID
   - Record detailed LDAP errors in backend logs only

4. **Configuration Security:**
   - AD admin password from environment variable (not in config file)
   - Warn on port 389 use (plaintext transmission risk)
   - Validate configuration before applying mode switch
   - Support test mode with clear warnings

## Sources

### Primary (HIGH confidence)
- [Spike 001 - go-ldap-ad-auth](D:/CODE/ClaudeCode/record_V2/.planning/spikes/001-go-ldap-ad-auth/README.md) - Complete go-ldap/v3 usage guide with code examples
- [Spike 002 - ldaps-security](D:/CODE/ClaudeCode/record_V2/.planning/spikes/002-ldaps-security/README.md) - LDAPS security best practices and TLS configuration
- [Spike 003 - auth-switch-architecture](D:/CODE/ClaudeCode/record_V2/.planning/spikes/003-auth-switch-architecture/README.md) - Strategy pattern for authentication switching
- [Spike 004 - ad-user-management](D:/CODE/ClaudeCode/record_V2/.planning/spikes/004-ad-user-management/README.md) - AD user mapping and local shadow records
- [Spike 005 - ad-config-validation](D:/CODE/ClaudeCode/record_V2/.planning/spikes/005-ad-config-validation/README.md) - Four-layer AD configuration validation
- [go-ldap/v3 package documentation](https://pkg.go.dev/github.com/go-ldap/ldap/v3) - Official Go LDAP library documentation
- [Spike findings SKILL.md](D:/CODE/ClaudeCode/record_V2/.claude/skills/spike-findings-record-v2/SKILL.md) - Synthesized spike findings
- [Existing codebase](D:/CODE/ClaudeCode/record_V2/) - AuthService, User model, migrations, audit service

### Secondary (MEDIUM confidence)
- [go-ldap/ldap GitHub Releases](https://github.com/go-ldap/ldap/releases) - Version history and current release (v3.4.6)
- [LDAP signing for Active Directory - Microsoft Learn](https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/ldap-signing) - Official Microsoft LDAP security guidance
- [Enforcing TLS 1.2+ for LDAPS - DSInternals](https://www.dsinternals.com/en/active-directory-domain-controller-tls-ldaps/) - TLS configuration for AD domain controllers
- [Active Directory security best practices - Specops Software](https://specopssoft.com/blog/active-directory-security-best-practices/) - Current AD security recommendations

### Tertiary (LOW confidence)
- [Web Search: go-ldap v3 2025](https://web.archive.org/web/20250101000000/https://pkg.go.dev/github.com/go-ldap/ldap/v3) - Confirms current version and active maintenance
- Various Stack Overflow discussions on LDAP injection prevention and TLS configuration

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Verified against go.mod, spike research, and official documentation
- Architecture: HIGH - Based on 5 validated spikes with working code examples
- Pitfalls: HIGH - Derived from spike investigation trails and common LDAP integration issues
- Security: HIGH - Based on official Microsoft documentation and security research

**Research date:** 2026-04-28
**Valid until:** 2026-05-28 (30 days - Go ecosystem stable, AD authentication mature technology)

---
*Phase: 12-windows-ad*
*Research completed: 2026-04-28*
