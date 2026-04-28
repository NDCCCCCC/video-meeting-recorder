---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/auth/ad_auth.go
  - internal/auth/ad_authenticator_test.go
autonomous: true
requirements:
  - SM4-AD-PASSWORD-DECRYPT

must_haves:
  truths:
    - "AD domain login receives plaintext password after SM4 decryption"
    - "AD login works when frontend sends SM4-encrypted password"
    - "AD login still works when frontend sends plaintext password (backward compatible)"
    - "Decryption failure returns proper error message"
  artifacts:
    - path: "internal/auth/ad_auth.go"
      provides: "SM4 password decryption before AD bind"
      contains: "DecryptPasswordECB"
    - path: "internal/auth/ad_authenticator_test.go"
      provides: "Test coverage for AD SM4 decryption"
      contains: "TestADAuth"
  key_links:
    - from: "internal/auth/ad_auth.go"
      to: "internal/utils/sm4_password.go"
      via: "DecryptPasswordECB call"
      pattern: "utils\\.DecryptPasswordECB"
---

<objective>
Fix AD domain login to decrypt SM4-encrypted passwords before sending to AD server.

Purpose: The frontend encrypts passwords with SM4-ECB before transmission. The local authenticator already decrypts these before checking against the local database, but the AD authenticator passes the encrypted password directly to `conn.Bind(userDN, req.Password)`, which sends the SM4 ciphertext to the AD server as if it were the user's actual password. This causes AD authentication to always fail when the frontend is configured with `VITE_SM4_SECRET`.

Output: AD authenticator that decrypts SM4 passwords before LDAP bind, with test coverage.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

<!-- Key interfaces the executor needs -->

<interfaces>
<!-- From internal/auth/ad_auth.go - the ADAuthenticator struct and its Login method -->
```go
// ADAuthenticator.ADAuthenticator struct fields:
type ADAuthenticator struct {
    adConfig     *config.ADAuthConfig
    db           *gorm.DB
    tokenService *SM4TokenService
    logger       *zap.Logger
}

// Login method signature:
func (a *ADAuthenticator) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error)

// LoginRequest (from service.go):
type LoginRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50"`
    Password string `json:"password" binding:"required"`
}
```

<!-- From internal/utils/sm4_password.go - existing decryption functions -->
```go
// IsEncryptedPassword checks if password has SM4: prefix
func IsEncryptedPassword(password string) bool

// DecryptPasswordECB decrypts SM4-ECB encrypted password
func DecryptPasswordECB(ciphertext string, sm4Secret string) (string, error)

// ValidateSM4Secret validates the SM4 secret key
func ValidateSM4Secret(secret string) error
```

<!-- From internal/config/config.go - ADAuthConfig has no SM4Secret, it lives on AuthConfig -->
```go
type AuthConfig struct {
    SM4Secret string
    Mode      string
    AD        ADAuthConfig
}
```

<!-- From internal/auth/local_auth.go - the exact pattern to follow (lines 69-94) -->
```go
// 3. Validate SM4 key config (if using encrypted password)
if a.cfg != nil && utils.IsEncryptedPassword(req.Password) {
    if a.cfg.SM4Secret != "" {
        if err := utils.ValidateSM4Secret(a.cfg.SM4Secret); err != nil {
            a.logger.Error("Invalid SM4 secret configuration", zap.Error(err))
            return nil, errors.New("system configuration error")
        }
    }
}

// 4. Try to decrypt password (if encrypted)
passwordToCheck := req.Password
if a.cfg != nil && utils.IsEncryptedPassword(req.Password) {
    decrypted, err := utils.DecryptPasswordECB(req.Password, a.cfg.SM4Secret)
    if err != nil {
        // Record decryption failure
        a.logger.Warn("Failed to decrypt password")
        return nil, errors.New("username or password error")
    }
    passwordToCheck = decrypted
    a.logger.Debug("Password decrypted for login")
}
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add SM4 password decryption to AD authenticator</name>
  <files>internal/auth/ad_auth.go</files>
  <behavior>
    - When password has "SM4:" prefix and sm4Secret is configured, AD authenticator decrypts it before LDAP bind
    - When password has no "SM4:" prefix, AD authenticator uses it as-is (backward compatible)
    - When decryption fails, returns error "domain password error" (does not leak decryption failure details)
    - ADAuthenticator needs access to sm4Secret from AuthConfig to decrypt
  </behavior>
  <action>
    1. Add `sm4Secret string` field to `ADAuthenticator` struct in `internal/auth/ad_auth.go`.

    2. Update `NewADAuthenticator` function signature to accept `sm4Secret string` parameter and store it on the struct. The caller in `internal/auth/service.go` line 85 passes `cfg.Auth.SM4Secret`.

    3. In `ADAuthenticator.Login()`, BEFORE Step 5 (`conn.Bind(userDN, req.Password)`), add SM4 decryption logic following the exact pattern from `local_auth.go` lines 69-94:
       - Check if `utils.IsEncryptedPassword(req.Password)` is true
       - If yes, call `utils.DecryptPasswordECB(req.Password, a.sm4Secret)` to get plaintext
       - If decryption fails, log warning and return `errors.New("domain password error")`
       - Use the decrypted password in the `conn.Bind(userDN, ...)` call instead of `req.Password`

    4. Update the caller in `internal/auth/service.go` line 85:
       Change from: `adAuth := NewADAuthenticator(&cfg.Auth.AD, db, tokenService, logger)`
       Change to: `adAuth := NewADAuthenticator(&cfg.Auth.AD, db, tokenService, logger, cfg.Auth.SM4Secret)`

    The decryption must happen AFTER finding the user DN (Step 3-4) but BEFORE binding as the user (Step 5). This is the same position as in local_auth where decryption happens before password verification.
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/record_V2 && go build ./... && go test ./internal/auth/... -run "TestAD" -v -count=1 2>&1 | head -60</automated>
  </verify>
  <done>
    - ADAuthenticator struct has sm4Secret field
    - NewADAuthenticator accepts sm4Secret parameter
    - Login() decrypts SM4 password before conn.Bind
    - service.go updated to pass SM4Secret to NewADAuthenticator
    - All existing tests pass
    - go build succeeds with no errors
  </done>
</task>

<task type="auto">
  <name>Task 2: Write tests for AD SM4 password decryption</name>
  <files>internal/auth/ad_authenticator_test.go</files>
  <action>
    Read the existing test file `internal/auth/ad_authenticator_test.go` first to understand current test patterns.

    Add test cases for the SM4 decryption in the AD authenticator Login flow:

    1. **Test SM4 encrypted password is decrypted before bind**: Create a test that sets up an ADAuthenticator with an sm4Secret, simulates receiving an SM4-encrypted password (using `utils.DecryptPasswordECB` to produce a valid encrypted string), and verifies the decrypted plaintext is what gets passed to the LDAP bind. Since LDAP connection is external, use the existing test pattern from the file (mock or integration-style).

    2. **Test plaintext password passes through unchanged**: When password has no "SM4:" prefix, it should be used as-is without any decryption attempt.

    3. **Test decryption failure returns error**: When password has "SM4:" prefix but decryption fails (wrong key, corrupted data), the Login should return an error without attempting LDAP bind.

    If the existing test file already has a test server mock pattern, reuse it. If tests require actual LDAP server, write unit-level tests that verify the decryption logic path by testing the password preparation step directly (consider extracting a `preparePassword` helper if that makes testing cleaner).
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/record_V2 && go test ./internal/auth/... -v -count=1 2>&1 | tail -40</automated>
  </verify>
  <done>
    - Test cases cover encrypted password decryption, plaintext passthrough, and decryption failure
    - All tests pass
    - No regression in existing AD auth tests
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Frontend to Backend | SM4-encrypted password transmitted over HTTPS |
| Backend to AD Server | Decrypted plaintext password sent via LDAP/LDAPS bind |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-sm4-01 | I (Information Disclosure) | AD password decryption | mitigate | SM4 secret must not be logged; decryption failures return generic error "domain password error" |
| T-sm4-02 | S (Spoofing) | SM4 encrypted payload | accept | SM4-ECB provides confidentiality for password transit; replay protection handled by rate limiter |
| T-sm4-03 | T (Tampering) | SM4 ciphertext in transit | mitigate | HTTPS transport + SM4 encryption provides dual-layer integrity |
</threat_model>

<verification>
1. `go build ./...` succeeds with no errors
2. `go test ./internal/auth/... -v` passes all tests including new SM4 decryption tests
3. AD login with SM4-encrypted password decrypts correctly before LDAP bind
4. AD login with plaintext password continues to work (backward compatible)
5. service.go correctly passes SM4Secret to NewADAuthenticator
</verification>

<success_criteria>
- AD authenticator decrypts SM4-encrypted passwords before sending to AD server
- Backward compatible with plaintext passwords
- Test coverage for the decryption path
- All existing tests continue to pass
</success_criteria>

<output>
After completion, create `.planning/quick/260428-mlh-sm4/260428-mlh-SUMMARY.md`
</output>
