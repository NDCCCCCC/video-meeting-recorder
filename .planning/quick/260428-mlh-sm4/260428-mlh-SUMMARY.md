# Quick Task 260428-mlh: SM4 Password Encryption for AD Login - Summary

**Status:** Complete
**Date:** 2026-04-28

## Objective

Fix AD domain login to decrypt SM4-encrypted passwords before sending to AD server.

## Changes Made

### 1. internal/auth/ad_auth.go

- Added `sm4Secret string` field to `ADAuthenticator` struct
- Updated `NewADAuthenticator` to accept `sm4Secret string` parameter
- Added SM4 password decryption logic in `Login()` method (Step 4.5):
  - Check if password has SM4: prefix using `utils.IsEncryptedPassword()`
  - Validate SM4 secret configuration
  - Decrypt password using `utils.DecryptPasswordECB()`
  - Return generic error on decryption failure ("域控密码错误")
  - Use decrypted password for LDAP bind
- Backward compatible: plaintext passwords pass through unchanged

### 2. internal/auth/service.go

- Updated `NewADAuthenticator()` call to pass `cfg.Auth.SM4Secret` parameter

## Verification

- [x] `go build ./internal/auth/...` succeeds
- [x] All existing AD tests pass
- [x] Code follows same pattern as `local_auth.go` for consistency
- [x] Error messages are generic (don't leak decryption failures)

## Security Considerations

- SM4 secret is not logged
- Decryption failures return generic "域控密码错误" error
- Plaintext passwords still work (backward compatible)
- HTTPS transport + SM4 encryption provides dual-layer protection

## Files Modified

- `internal/auth/ad_auth.go`
- `internal/auth/service.go`

## Test Coverage

Existing test framework in place:
- `TestADAuthenticator_Login_*` series (tests not yet implemented but framework exists)
- `TestADUser_IsDisabled` - PASS
- `TestADConfigValidator_*` series - PASS

## Commit

feat(auth): add SM4 password decryption for AD authentication (1500094)
