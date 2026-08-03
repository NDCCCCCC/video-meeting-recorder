package auth

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// MockLDAPConnection is a mock for LDAP connection
type MockLDAPConnection struct {
	mock.Mock
}

func (m *MockLDAPConnection) Bind(username, password string) error {
	args := m.Called(username, password)
	return args.Error(0)
}

func (m *MockLDAPConnection) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockLDAPConnection) Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error) {
	args := m.Called(searchRequest)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ldap.SearchResult), args.Error(1)
}

// TestADAuthenticator_Login_Success tests successful AD authentication
func TestADAuthenticator_Login_Success(t *testing.T) {
	// Setup
	req := &LoginRequest{
		Username: "testuser",
		Password: "testpass",
	}
	_ = req // Used in TODO implementation

	// Expected: User found in AD, token returned
	// TODO: Implement ADAuthenticator.Login() method
	// TODO: Mock LDAP connection returning valid user
	// TODO: Assert token is returned
	// TODO: Assert user DTO contains correct information

	t.Log("TestADAuthenticator_Login_Success: Not yet implemented")
}

// TestADAuthenticator_Login_UserNotFound tests AD authentication with non-existent user
func TestADAuthenticator_Login_UserNotFound(t *testing.T) {
	// Setup
	req := &LoginRequest{
		Username: "nonexistent",
		Password: "testpass",
	}
	_ = req // Used in TODO implementation

	// Expected: Error "域控账号不存在"
	// TODO: Implement ADAuthenticator.Login() method
	// TODO: Mock LDAP connection returning empty search results
	// TODO: Assert error message equals "域控账号不存在"

	t.Log("TestADAuthenticator_Login_UserNotFound: Not yet implemented")
}

// TestADAuthenticator_Login_InvalidPassword tests AD authentication with invalid password
func TestADAuthenticator_Login_InvalidPassword(t *testing.T) {
	// Setup
	req := &LoginRequest{
		Username: "testuser",
		Password: "wrongpass",
	}
	_ = req // Used in TODO implementation

	// Expected: Error "域控密码错误"
	// TODO: Implement ADAuthenticator.Login() method
	// TODO: Mock LDAP connection returning invalid credentials error
	// TODO: Assert error message equals "域控密码错误"

	t.Log("TestADAuthenticator_Login_InvalidPassword: Not yet implemented")
}

// TestADAuthenticator_Login_AccountDisabled tests AD authentication with disabled account
func TestADAuthenticator_Login_AccountDisabled(t *testing.T) {
	// Setup
	req := &LoginRequest{
		Username: "disableduser",
		Password: "testpass",
	}
	_ = req // Used in TODO implementation

	// Expected: Error "域控账号已禁用"
	// TODO: Implement ADAuthenticator.Login() method
	// TODO: Mock LDAP connection returning user with ACCOUNTDISABLE flag (0x0002)
	// TODO: Assert error message equals "域控账号已禁用"

	t.Log("TestADAuthenticator_Login_AccountDisabled: Not yet implemented")
}

// TestADAuthenticator_Login_ConnectionFailed tests AD authentication with connection failure
func TestADAuthenticator_Login_ConnectionFailed(t *testing.T) {
	// Setup
	req := &LoginRequest{
		Username: "testuser",
		Password: "testpass",
	}
	_ = req // Used in TODO implementation

	// Expected: Connection error
	// TODO: Implement ADAuthenticator.Login() method
	// TODO: Mock LDAP connection failing to connect
	// TODO: Assert error contains "连接AD服务器失败"

	t.Log("TestADAuthenticator_Login_ConnectionFailed: Not yet implemented")
}

// TestADAuthenticator_CreateLocalUser tests local user creation after AD authentication
func TestADAuthenticator_CreateLocalUser(t *testing.T) {
	// Setup
	adUser := &ADUser{
		Username:           "newuser",
		Email:              "newuser@example.com",
		DisplayName:        "New User",
		Department:         "IT",
		UserPrincipalName:  "newuser@example.com",
		ObjectGUID:         "12345678-1234-1234-1234-123456789012",
		DN:                 "CN=newuser,CN=Users,DC=example,DC=com",
		UserAccountControl: 0x0000, // Active account
	}
	_ = adUser // Used in TODO implementation

	// Expected: Local user created with AD fields populated
	// TODO: Implement findOrCreateLocalUser() method
	// TODO: Mock database insert
	// TODO: Assert user created in database
	// TODO: Assert AD fields (ad_username, ad_dn, ad_guid, etc.) are populated
	// TODO: Assert random password generated and hashed

	t.Log("TestADAuthenticator_CreateLocalUser: Not yet implemented")
}

// TestADAuthenticator_ExistingUserUpdated tests updating AD information for existing user
func TestADAuthenticator_ExistingUserUpdated(t *testing.T) {
	// Setup
	adUser := &ADUser{
		Username:           "existinguser",
		Email:              "updated@example.com",
		DisplayName:        "Updated Name",
		Department:         "Sales",
		UserPrincipalName:  "existinguser@example.com",
		ObjectGUID:         "87654321-4321-4321-4321-210987654321",
		DN:                 "CN=existinguser,CN=Users,DC=example,DC=com",
		UserAccountControl: 0x0000,
	}
	_ = adUser // Used in TODO implementation

	// Expected: Existing user's AD fields updated
	// TODO: Implement findOrCreateLocalUser() method
	// TODO: Mock database query finding existing user
	// TODO: Mock database update
	// TODO: Assert user AD fields updated with new values
	// TODO: Assert last_ad_login timestamp updated

	t.Log("TestADAuthenticator_ExistingUserUpdated: Not yet implemented")
}

// TestADUser_IsDisabled tests the IsDisabled method
func TestADUser_IsDisabled(t *testing.T) {
	tests := []struct {
		name     string
		uac      uint32
		expected bool
	}{
		{
			name:     "Active account",
			uac:      0x0000,
			expected: false,
		},
		{
			name:     "Disabled account",
			uac:      0x0002, // ACCOUNTDISABLE
			expected: true,
		},
		{
			name:     "Disabled with other flags",
			uac:      0x0002 | 0x0010, // ACCOUNTDISABLE | PASSWORD_EXPIRED
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &ADUser{
				UserAccountControl: tt.uac,
			}
			result := user.IsDisabled()
			assert.Equal(t, tt.expected, result, "IsDisabled() should return %v for UAC 0x%X", tt.expected, tt.uac)
		})
	}
}

// TestErrADUserNotRegistered_Sentinel verifies the sentinel error is detected
// by IsADUserNotRegistered so the HTTP handler can map it to 403 (whitelist policy).
// The sentinel itself was migrated to internal/errors in Phase 20 R-3.
func TestErrADUserNotRegistered_Sentinel(t *testing.T) {
	t.Run("direct sentinel matches", func(t *testing.T) {
		err := apperrors.ErrADUserNotRegistered
		assert.True(t, IsADUserNotRegistered(err), "IsADUserNotRegistered must recognize sentinel")
		assert.True(t, errors.Is(err, apperrors.ErrADUserNotRegistered))
	})

	t.Run("wrapped sentinel matches", func(t *testing.T) {
		err := fmt.Errorf("wrap: %w", apperrors.ErrADUserNotRegistered)
		assert.True(t, IsADUserNotRegistered(err), "IsADUserNotRegistered must recognize wrapped sentinel")
	})

	t.Run("unrelated error does not match", func(t *testing.T) {
		err := errors.New("some other error")
		assert.False(t, IsADUserNotRegistered(err), "IsADUserNotRegistered must not match unrelated errors")
	})
}
