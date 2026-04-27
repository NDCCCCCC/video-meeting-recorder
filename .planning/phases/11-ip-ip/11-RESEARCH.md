# Phase 11: IP地址登录限制 - Research

**Researched:** 2026-04-27
**Domain:** Go IP Validation & Access Control (Gin/GORM/React)
**Confidence:** HIGH

## Summary

Phase 11 adds IP address-based access control to the authentication system, supporting both user-level and role-level IP restrictions. The implementation uses Go's standard `net` package for IP validation, GORM's JSON field storage for IP lists, and integrates with the existing login flow and audit logging system.

**Primary recommendation:** Use Go's `net.ParseCIDR()` and `net.ParseIP()` for validation with JSON field storage in SQLite, following the established multi-role OR logic pattern from Phase 09.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| IP validation logic | API / Backend | — | Server-side verification prevents client bypass |
| IP restriction storage | Database | — | SQLite with GORM JSON field storage |
| IP restriction enforcement | API / Backend | — | Applied during login authentication flow |
| IP management UI | Frontend | — | React forms for admins to configure restrictions |
| Audit logging | API / Backend | — | Server-side recording of IP restriction violations |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go `net` package | 1.25.0 | IP parsing, CIDR validation, range matching | [VERIFIED: go.mod] Standard library, no external dependencies |
| GORM | 1.30.0 | ORM with JSON field support | [VERIFIED: go.mod] Existing project standard |
| Gin | 1.11.0 | Web framework (c.ClientIP()) | [VERIFIED: go.mod] Existing project standard |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Ant Design 6 | ^6.0.0 | IP input UI components (Input.TextArea, Tag) | [VERIFIED: frontend/package.json] Existing project standard |
| React 19 | ^19.2.0 | Form state management | [VERIFIED: frontend/package.json] Existing project standard |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go `net` package | Third-party IP libraries | Standard library is sufficient, well-tested, zero dependencies |
| JSON field storage | Separate IP address table | JSON is simpler, no complex joins needed for read-only IP lists |
| `c.ClientIP()` | Manual X-Forwarded-For parsing | Gin's built-in handles trusted proxies correctly |

**Installation:**
```bash
# No additional packages needed - using existing Go standard library and GORM
# Verify Go version:
go version  # Expected: go1.25.0
```

**Version verification:** All packages are from existing `go.mod` (verified 2026-04-27).

## Architecture Patterns

### System Architecture Diagram

```
User Login Request
        ↓
[Login Handler - auth_handler.go]
        ↓
Extract Client IP (c.ClientIP())
        ↓
[Auth Service - auth/service.go]
        ↓
┌─────────────────────────────────┐
│ 1. Password Validation          │
│ 2. User Status Check            │
│ 3. IP Restriction Check (NEW)   │
│    ├─ Load User IPs             │
│    ├─ Load All Role IPs         │
│    ├─ Merge IPs (OR logic)      │
│    └─ Validate Client IP        │
│ 4. Generate Token               │
│ 5. Create Session               │
└─────────────────────────────────┘
        ↓
IP Check Failed?
        ↓
    ┌───┴───┐
    │       │
   Yes     No
    │       │
    ↓       ↓
[Audit Log]  [Return Token]
(IP Restriction Failed)
    ↓
Return Error
```

### Recommended Project Structure

```
internal/
├── models/
│   ├── user.go              # Add AllowedIPs field
│   ├── role.go              # Add AllowedIPs field
│   └── ip_restriction.go    # NEW: IP validation types and methods
├── auth/
│   ├── service.go           # Add CheckIPRestriction() method
│   └── ip_validator.go      # NEW: IP validation logic
├── handlers/
│   └── auth_handler.go      # Add IP check call in Login()
├── middleware/
│   └── audit.go             # Add ip_restriction_failed action
└── migrations/
    └── 011_add_ip_restrictions.go  # Database migration

frontend/src/
├── pages/system/
│   ├── users/index.tsx      # Add IP restriction form field
│   └── roles/index.tsx      # Add IP restriction form field
├── types/
│   ├── user.ts              # Add allowed_ips type
│   └── role.ts              # Add allowed_ips type
└── api/
    ├── user.ts              # Add allowed_ips to API types
    └── role.ts              # Add allowed_ips to API types
```

### Pattern 1: IP Restriction Field Storage (JSON)

**What:** Store IP address lists as JSON arrays in SQLite using GORM's `type:json` tag

**When to use:** When storing collections of IP addresses (single IPs, CIDR ranges, IP ranges) that don't require complex queries or joins

**Example:**

```go
// Source: [Existing pattern from internal/models/user.go - Roles field]
type User struct {
    Base
    Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
    // ... existing fields ...
    Roles        []Role     `gorm:"many2many:users_roles;" json:"roles,omitempty"`
    AllowedIPs   AllowedIPs `gorm:"type:json" json:"allowed_ips"` // NEW
}

// Source: [Existing pattern from internal/models/role.go]
type Role struct {
    Base
    Name        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
    // ... existing fields ...
    Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
    AllowedIPs  AllowedIPs   `gorm:"type:json" json:"allowed_ips"` // NEW
}

// IP list type implementing Scanner and Valuer for GORM
type AllowedIPs []string

func (ips *AllowedIPs) Scan(value interface{}) error {
    // Implement JSON deserialization
    if value == nil {
        *ips = make([]string, 0)
        return nil
    }
    bytes, ok := value.([]byte)
    if !ok {
        return errors.New("type assertion to []byte failed")
    }
    return json.Unmarshal(bytes, ips)
}

func (ips AllowedIPs) Value() (driver.Value, error) {
    // Implement JSON serialization
    if len(ips) == 0 {
        return nil, nil
    }
    return json.Marshal(ips)
}
```

### Pattern 2: IP Validation with Go `net` Package

**What:** Use Go's standard library `net.ParseIP()`, `net.ParseCIDR()`, and `net.IPNet.Contains()` for IP validation and matching

**When to use:** For all IP address validation, CIDR range matching, and IP range comparisons

**Example:**

```go
// Source: [Go standard library - verified via go.mod 1.25.0]
package auth

import (
    "net"
    "strings"
)

type IPValidator struct{}

// ValidateIP validates a single IP address
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

// ValidateCIDR validates a CIDR range
func (v *IPValidator) ValidateCIDR(cidr string) error {
    _, _, err := net.ParseCIDR(cidr)
    if err != nil {
        return errors.New("invalid CIDR range")
    }
    return nil
}

// ValidateIPRange validates an IP range (e.g., "192.168.1.100-192.168.1.200")
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

// IsIPAllowed checks if a client IP is in the allowed list
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
            if bytes.Compare(clientAddr, startIP) >= 0 && bytes.Compare(clientAddr, endIP) <= 0 {
                return true, nil
            }
            continue
        }
    }
    
    return false, nil
}
```

### Pattern 3: Multi-Role IP Restriction OR Logic

**What:** Merge user-level and role-level IP restrictions using OR logic (user can login if IP matches user's list OR any of their roles' lists)

**When to use:** When implementing IP restrictions for users with multiple roles (Phase 09 multi-role pattern)

**Example:**

```go
// Source: [Existing pattern from internal/models/user.go - HasPermission()]
func (u *User) HasPermission(resource, action string) bool {
    if len(u.Roles) == 0 {
        return false
    }

    // Check all roles (OR logic per D-07)
    for _, role := range u.Roles {
        // ... permission check logic ...
    }
    return false
}

// NEW: Apply same OR logic for IP restrictions
func (s *Service) CheckIPRestriction(user *models.User, clientIP string) error {
    // Collect all allowed IPs from user and all roles
    allowedIPs := make([]string, 0)
    
    // Add user's IP restrictions
    if len(user.AllowedIPs) > 0 {
        allowedIPs = append(allowedIPs, user.AllowedIPs...)
    }
    
    // Add IP restrictions from all roles (OR logic)
    for _, role := range user.Roles {
        if len(role.AllowedIPs) > 0 {
            allowedIPs = append(allowedIPs, role.AllowedIPs...)
        }
    }
    
    // If no restrictions, allow all IPs
    if len(allowedIPs) == 0 {
        return nil
    }
    
    // Check if client IP is allowed
    validator := &IPValidator{}
    allowed, err := validator.IsIPAllowed(clientIP, allowedIPs)
    if err != nil {
        s.logger.Warn("IP validation failed", 
            zap.String("client_ip", clientIP),
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

### Anti-Patterns to Avoid

- **Don't use regex for IP validation**: Go's `net.ParseIP()` is more reliable and handles edge cases
- **Don't store IPs as comma-separated strings**: JSON arrays are more maintainable and less error-prone
- **Don't implement IP range checking manually**: Use `net.IPNet.Contains()` for CIDR matching
- **Don't forget IPv6 rejection**: Explicitly check `ip.To4() == nil` to reject IPv6 addresses per D-09
- **Don't bypass IP checks for admins**: All users (including admins) must respect IP restrictions per D-15
- **Don't use X-Forwarded-For directly**: Use Gin's `c.ClientIP()` which handles trusted proxies correctly per D-10

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| IP validation | Custom regex parsers | Go `net.ParseIP()`, `net.ParseCIDR()` | Standard library handles edge cases (octal, hex, etc.) |
| CIDR matching | Manual subnet math | `net.IPNet.Contains()` | Correctly handles subnet masks and boundaries |
| JSON serialization | Custom marshal/unmarshal | GORM `Scanner`/`Valuer` interfaces | Proven pattern from existing codebase |
| IP range comparison | Byte-by-byte comparison | `bytes.Compare()` | Standard library is clear and efficient |
| Client IP extraction | Manual header parsing | Gin `c.ClientIP()` | Handles X-Forwarded-For chain correctly |

**Key insight:** IP address validation is surprisingly complex (octal formats, IPv4-mapped IPv6, etc.). The Go standard library has years of battle-tested code for this — don't reinvent it.

## Runtime State Inventory

> Not applicable — this is a greenfield feature phase, not a rename/refactor phase.

## Common Pitfalls

### Pitfall 1: CIDR Range Validation Errors
**What goes wrong:** Storing invalid CIDR ranges (e.g., "192.168.1.0/33") causes runtime panics in `net.ParseCIDR()`

**Why it happens:** CIDR validation only happens at login time, not at storage time

**How to avoid:** Validate all IP entries when saving user/role records; return validation errors to admin user

**Warning signs:** Login attempts causing "panic: invalid CIDR address" errors

### Pitfall 2: IP Restriction Deadlock
**What goes wrong:** Admin sets IP restriction for themselves, gets locked out of the system

**Why it happens:** No special exemption for admins per D-15 ("所有用户（包括管理员角色）都必须遵守IP限制")

**How to avoid:** 
1. Warn admin when setting IP restrictions that would exclude their current IP
2. Provide database rollback mechanism in emergencies
3. Document emergency procedure (direct DB update)

**Warning signs:** Admin can't login after setting IP restrictions

### Pitfall 3: JSON Array Serialization Failures
**What goes wrong:** GORM fails to save/load `AllowedIPs` field with "unsupported scan" errors

**Why it happens:** Missing `Scanner` and `Valuer` interface implementations for custom type

**How to avoid:** Implement both interfaces on `AllowedIPs` type following the pattern from `AuditLogData`

**Warning signs:** Database saves succeed but IP lists are empty/null on load

### Pitfall 4: Incorrect OR Logic for Multi-Role Users
**What goes wrong:** User with 3 roles gets denied even though one role allows their IP

**Why it happens:** Implementing AND logic instead of OR logic for merging role IP restrictions

**How to avoid:** Follow the pattern from `User.HasPermission()` which uses OR logic across all roles

**Warning signs:** Users report "I have role X that allows this IP but still get denied"

### Pitfall 5: IP Range Format Confusion
**What goes wrong:** IP ranges stored as "192.168.1.100-200" instead of "192.168.1.100-192.168.1.200"

**Why it happens:** Inconsistent format documentation and lack of input validation

**How to avoid:** 
1. Document exact format: "START_IP-END_IP" (both full IPs)
2. Provide client-side validation
3. Show examples in UI placeholder text

**Warning signs:** "invalid IP range format" errors when parsing ranges

## Code Examples

Verified patterns from official sources:

### IP Address Validation (Go standard library)

```go
// Source: [Go 1.25.0 standard library - net package]
package main

import (
    "fmt"
    "net"
)

func main() {
    // Validate single IP
    ip := net.ParseIP("192.168.1.100")
    if ip != nil {
        fmt.Printf("Valid IP: %s\n", ip)
    }
    
    // Validate CIDR
    _, ipNet, err := net.ParseCIDR("192.168.1.0/24")
    if err == nil {
        fmt.Printf("Valid CIDR: %s\n", ipNet.String())
    }
    
    // Check if IP is in CIDR range
    clientIP := net.ParseIP("192.168.1.50")
    if ipNet.Contains(clientIP) {
        fmt.Println("IP is in range")
    }
    
    // Check for IPv4 (reject IPv6)
    if ip.To4() != nil {
        fmt.Println("This is IPv4")
    }
}
```

### GORM JSON Field Storage Pattern

```go
// Source: [Existing pattern from internal/models/audit_log.go - AuditLogData]
type AuditLogData struct {
    OldData interface{} `json:"old_data,omitempty"`
    NewData interface{} `json:"new_data,omitempty"`
    Diff    interface{} `json:"diff,omitempty"`
}

func (a *AuditLogData) MarshalJSON() ([]byte, error) {
    return json.Marshal(struct {
        OldData interface{} `json:"old_data,omitempty"`
        NewData interface{} `json:"new_data,omitempty"`
        Diff    interface{} `json:"diff,omitempty"`
    }{
        OldData: a.OldData,
        NewData: a.NewData,
        Diff:    a.Diff,
    })
}

// NEW: Apply same pattern to AllowedIPs
type AllowedIPs []string

func (ips *AllowedIPs) Scan(value interface{}) error {
    if value == nil {
        *ips = make([]string, 0)
        return nil
    }
    bytes, ok := value.([]byte)
    if !ok {
        return fmt.Errorf("type assertion to []byte failed for AllowedIPs")
    }
    return json.Unmarshal(bytes, ips)
}

func (ips AllowedIPs) Value() (driver.Value, error) {
    if len(ips) == 0 {
        return "[]", nil  // Empty JSON array
    }
    return json.Marshal(ips)
}
```

### Multi-Role OR Logic Pattern

```go
// Source: [Existing pattern from internal/models/user.go - HasPermission()]
func (u *User) HasPermission(resource, action string) bool {
    if len(u.Roles) == 0 {
        return false
    }

    // Check all roles (OR logic per D-07)
    for _, role := range u.Roles {
        // ... permission check ...
    }
    return false
}

// NEW: Apply same pattern for IP restrictions
func (s *Service) CheckIPRestriction(user *models.User, clientIP string) error {
    // Merge IPs from user + all roles (OR logic)
    allowedIPs := make([]string, 0)
    
    if len(user.AllowedIPs) > 0 {
        allowedIPs = append(allowedIPs, user.AllowedIPs...)
    }
    
    for _, role := range user.Roles {
        if len(role.AllowedIPs) > 0 {
            allowedIPs = append(allowedIPs, role.AllowedIPs...)
        }
    }
    
    // Empty list = no restrictions
    if len(allowedIPs) == 0 {
        return nil
    }
    
    // Validate client IP against merged list
    // ... validation logic ...
}
```

### Audit Logging Pattern

```go
// Source: [Existing pattern from internal/middleware/audit.go]
const (
    ActionLogin          = "login"
    ActionLogout         = "logout"
    ActionPasswordChange = "password_change"
    // ... other actions ...
)

// NEW: Add IP restriction failure action
const (
    ActionIPRestrictionFailed = "ip_restriction_failed"
)

// Usage in auth service:
if err := s.CheckIPRestriction(&user, ipAddress); err != nil {
    // Log audit event
    s.auditLogger.Log(c, &models.AuditLog{
        Action:     models.ActionIPRestrictionFailed,
        Module:     models.ModuleUser,
        UserID:     user.ID,
        Username:   user.Username,
        IPAddress:  ipAddress,
        Status:     models.StatusFailure,
        ErrorMsg:   err.Error(),
    })
    return err
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No IP restrictions | User + Role IP restrictions with OR logic | Phase 11 (current) | Adds network-based access control layer |
| Manual header parsing | Gin `c.ClientIP()` with trusted proxies | Existing | Secure IP extraction behind proxies |
| Password-only auth | Password + IP dual-factor | Phase 11 (current) | Enhanced security for privileged accounts |

**Deprecated/outdated:**
- Direct `X-Forwarded-For` header reading: Use Gin's `c.ClientIP()` instead
- IP address string splitting: Use Go `net` package parsers
- Comma-separated IP storage: Use JSON arrays with GORM

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Go 1.25.0 `net` package supports all required IP operations (ParseIP, ParseCIDR, IPNet.Contains) | Standard Stack | LOW - Go stdlib is stable and well-documented |
| A2 | GORM SQLite JSON field storage works reliably for `[]string` arrays | Architecture Patterns | LOW - Existing codebase uses JSON fields successfully |
| A3 | Gin `c.ClientIP()` correctly handles X-Forwarded-For for trusted proxies | Code Examples | LOW - Verified in existing code (auth_handler.go:42) |
| A4 | Ant Design 6 Input.TextArea is suitable for multi-line IP input | Standard Stack | LOW - Standard form component, widely used |
| A5 | Existing audit logging infrastructure can handle new `ip_restriction_failed` action | Code Examples | LOW - Audit system is action-agnostic |

**All assumptions are LOW risk** - they're based on verified existing patterns in the codebase or stable standard library features.

## Open Questions (RESOLVED)

1. **Should we validate IP addresses on form submit or only on login?**
   - **Resolution:** Validate on save (better UX) AND login (security defense-in-depth)
   - **Plan reference:** Plan 11-01 Task 1 validates IP formats in backend, Plan 11-04 provides frontend input

2. **What's the maximum number of IP addresses allowed per user/role?**
   - **Resolution:** Claude's Discretion - no explicit limit set
   - **Plan reference:** Left as implementation discretion; large lists would be caught by testing

3. **Should we show the current admin's IP when they're setting restrictions?**
   - **Resolution:** Yes - show current IP and warn if excluding it
   - **Plan reference:** Plan 11-04 Task 4 includes checkpoint test for admin lockout warning

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25.0 | IP validation (net package) | ✓ | 1.25.0 | — |
| GORM 1.30.0 | JSON field storage | ✓ | 1.30.0 | — |
| Gin 1.11.0 | c.ClientIP() | ✓ | 1.11.0 | — |
| SQLite | JSON storage backend | ✓ | (via GORM) | — |
| React 19 | Frontend forms | ✓ | 19.2.0 | — |
| Ant Design 6 | UI components | ✓ | 6.0.0 | — |

**All dependencies available** - no blocking issues.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify/assert |
| Config file | None (standard Go test) |
| Quick run command | `go test -v ./internal/auth/... -run TestIP` |
| Full suite command | `go test -v ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| IP-01 | User-level IP restrictions work | unit | `go test -v ./internal/auth/... -run TestCheckIPRestriction` | ❌ Wave 0 |
| IP-02 | Role-level IP restrictions work | unit | `go test -v ./internal/auth/... -run TestCheckIPRestrictionRole` | ❌ Wave 0 |
| IP-03 | Multi-role OR logic merges IPs correctly | unit | `go test -v ./internal/auth/... -run TestMultiRoleIPMerge` | ❌ Wave 0 |
| IP-04 | IP validation rejects invalid formats | unit | `go test -v ./internal/auth/... -run TestValidateIPInvalid` | ❌ Wave 0 |
| IP-05 | CIDR range matching works correctly | unit | `go test -v ./internal/auth/... -run TestCIDRMatching` | ❌ Wave 0 |
| IP-06 | IP range matching works correctly | unit | `go test -v ./internal/auth/... -run TestIPRangeMatching` | ❌ Wave 0 |
| IP-07 | IPv6 addresses are rejected | unit | `go test -v ./internal/auth/... -run TestRejectIPv6` | ❌ Wave 0 |
| IP-08 | Empty IP list = no restrictions | unit | `go test -v ./internal/auth/... -run TestEmptyIPList` | ❌ Wave 0 |
| IP-09 | Audit logs record IP restriction failures | integration | `go test -v ./internal/auth/... -run TestIPRestrictionAuditLog` | ❌ Wave 0 |
| IP-10 | Frontend IP input validation | unit | `npm test -- IPValidator` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -v ./internal/auth/... -run TestIP`
- **Per wave merge:** `go test -v ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/auth/ip_validator_test.go` — IP validation logic tests
- [ ] `internal/auth/ip_restriction_test.go` — IP restriction integration tests
- [ ] `internal/models/ip_restriction_test.go` — GORM JSON field tests
- [ ] `frontend/src/components/__tests__/IPInput.test.tsx` — Frontend IP input component tests
- [ ] Test fixtures for user/role with IP restrictions

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | IP restriction adds network-level factor |
| V3 Session Management | yes | IP check at login (session creation) |
| V4 Access Control | yes | IP-based access restriction enforcement |
| V5 Input Validation | yes | Go `net.ParseIP()`, `net.ParseCIDR()` for IP format validation |
| V6 Cryptography | no | N/A (no crypto for IP validation) |

### Known Threat Patterns for Go IP Validation

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| IP spoofing | Spoofing | Server-side validation only; no client-side trust |
| X-Forwarded-For injection | Tampering | Use Gin `c.ClientIP()` with trusted proxy config |
| CIDR bypass (e.g., /0) | Tampering | Validate CIDR prefix length (/8-/32 for IPv4) |
| IPv6 bypass to skip IPv4 filter | Spoofing | Explicitly reject IPv6 (check `ip.To4() == nil`) |
| Admin lockout (DoS) | Denial of Service | Warn when excluding current IP; provide DB rollback procedure |

### Security Validation Requirements

1. **IP spoofing protection:** Client IP must be extracted server-side using `c.ClientIP()`
2. **Trusted proxy configuration:** Gin must have `SetTrustedProxies()` configured correctly
3. **IPv6 rejection:** Explicitly check `ip.To4() == nil` per D-09
4. **Admin lockout prevention:** UI warning when admin excludes their current IP
5. **Audit trail:** All IP restriction failures logged with username, IP, timestamp

## Sources

### Primary (HIGH confidence)
- [Go 1.25.0 standard library] - `net` package (ParseIP, ParseCIDR, IPNet, IP)
- [GORM 1.30.0 documentation] - JSON field storage with Scanner/Valuer interfaces
- [Gin 1.11.0 documentation] - `c.ClientIP()` method and trusted proxy configuration
- [Existing codebase] - `internal/models/user.go`, `internal/models/role.go`, `internal/handlers/auth_handler.go`

### Secondary (MEDIUM confidence)
- [Existing codebase patterns] - Multi-role OR logic (Phase 09), audit logging (Phase 04)
- [Ant Design 6 documentation] - Input.TextArea component for multi-line IP input

### Tertiary (LOW confidence)
- [ASSUMED] Go `net` package performance for large IP lists (should benchmark if > 100 IPs)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All packages verified in go.mod and existing codebase
- Architecture: HIGH - Patterns from existing code (multi-role OR logic, JSON fields)
- Pitfalls: HIGH - Based on common IP validation and access control mistakes

**Research date:** 2026-04-27
**Valid until:** 2026-05-27 (30 days - Go stdlib and GORM are stable)
