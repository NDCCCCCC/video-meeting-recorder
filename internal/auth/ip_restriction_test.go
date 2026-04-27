package auth

import (
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate tables
	err = db.AutoMigrate(&models.User{}, &models.Role{}, &models.Permission{})
	require.NoError(t, err)

	return db
}

// createTestUserWithRoles creates a test user with specified roles and IP restrictions
func createTestUserWithRoles(t *testing.T, db *gorm.DB, username string, userIPs []string, roleIPs map[string][]string) *models.User {
	user := &models.User{
		Username: username,
		Email:    username + "@test.com",
		FullName: "Test User",
		IsActive: true,
	}

	// Set user IP restrictions
	if len(userIPs) > 0 {
		err := user.SetAllowedIPs(userIPs)
		require.NoError(t, err)
	}

	// Create user
	err := db.Create(user).Error
	require.NoError(t, err)

	// Create roles and associate with user
	for roleName, ips := range roleIPs {
		role := &models.Role{
			Name:        roleName,
			Description: "Test role " + roleName,
		}

		// Set role IP restrictions
		if len(ips) > 0 {
			err := role.SetAllowedIPs(ips)
			require.NoError(t, err)
		}

		// Create role
		err = db.Create(role).Error
		require.NoError(t, err)

		// Associate role with user
		err = db.Model(user).Association("Roles").Append(role)
		require.NoError(t, err)
	}

	// Reload user with roles
	err = db.Preload("Roles").First(user, user.ID).Error
	require.NoError(t, err)

	return user
}

// TestCheckIPRestriction_UserOnly tests user-level IP restrictions only
// Validates that when only user has IP restrictions, role IPs are not considered
func TestCheckIPRestriction_UserOnly(t *testing.T) {

	db := setupTestDB(t)
	service := &Service{db: db}

	// Create user with IP restrictions, role without restrictions
	user := createTestUserWithRoles(t, db, "testuser", []string{"192.168.1.100"}, map[string][]string{
		"operator": {}, // No IP restrictions
	})

	tests := []struct {
		name        string
		clientIP    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "allowed IP",
			clientIP:    "192.168.1.100",
			wantErr:     false,
		},
		{
			name:        "disallowed IP",
			clientIP:    "192.168.1.101",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
		{
			name:        "different subnet",
			clientIP:    "10.0.0.1",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CheckIPRestriction(user, tt.clientIP)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheckIPRestriction_RoleOnly tests role-level IP restrictions only
// Validates that when only role has IP restrictions, user IPs are not considered
func TestCheckIPRestriction_RoleOnly(t *testing.T) {

	db := setupTestDB(t)
	service := &Service{db: db}

	// Create user without IP restrictions, role with restrictions
	user := createTestUserWithRoles(t, db, "testuser", []string{}, map[string][]string{
		"operator": {"192.168.1.0/24"},
	})

	tests := []struct {
		name        string
		clientIP    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "within role CIDR",
			clientIP:    "192.168.1.50",
			wantErr:     false,
		},
		{
			name:        "outside role CIDR",
			clientIP:    "192.168.2.50",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
		{
			name:        "different network",
			clientIP:    "10.0.0.1",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CheckIPRestriction(user, tt.clientIP)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheckIPRestriction_UserAndRole_OR tests OR logic merging user + role IPs per D-02
// Validates that user and role IP restrictions are merged using OR logic
func TestCheckIPRestriction_UserAndRole_OR(t *testing.T) {

	db := setupTestDB(t)
	service := &Service{db: db}

	// Create user with IP restrictions AND role with IP restrictions
	// Per D-02: User_IPs ∪ Role_IPs
	user := createTestUserWithRoles(t, db, "testuser", []string{"10.0.0.1"}, map[string][]string{
		"operator": {"192.168.1.0/24"},
	})

	tests := []struct {
		name        string
		clientIP    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "user's single IP allowed",
			clientIP: "10.0.0.1",
			wantErr:  false,
		},
		{
			name:     "role's CIDR range allowed",
			clientIP: "192.168.1.50",
			wantErr:  false,
		},
		{
			name:        "IP not in user or role lists",
			clientIP:    "172.16.0.1",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
		{
			name:     "boundary of role CIDR",
			clientIP: "192.168.1.255",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CheckIPRestriction(user, tt.clientIP)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheckIPRestriction_NoRestrictions tests empty IP lists (allow all per D-03)
// Validates that when neither user nor roles have IP restrictions, all IPs are allowed
func TestCheckIPRestriction_NoRestrictions(t *testing.T) {

	db := setupTestDB(t)
	service := &Service{db: db}

	// Create user without IP restrictions, role without restrictions
	user := createTestUserWithRoles(t, db, "testuser", []string{}, map[string][]string{
		"operator": {},
	})

	tests := []struct {
		name     string
		clientIP string
		wantErr  bool
	}{
		{
			name:     "any IP allowed - private",
			clientIP: "192.168.1.100",
			wantErr:  false,
		},
		{
			name:     "any IP allowed - public",
			clientIP: "8.8.8.8",
			wantErr:  false,
		},
		{
			name:     "any IP allowed - loopback",
			clientIP: "127.0.0.1",
			wantErr:  false,
		},
		{
			name:     "any IP allowed - different class A",
			clientIP: "10.0.0.1",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CheckIPRestriction(user, tt.clientIP)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheckIPRestriction_IPNotInList tests IP not allowed
// Validates that IPs not matching any restriction rule are rejected
func TestCheckIPRestriction_IPNotInList(t *testing.T) {

	db := setupTestDB(t)
	service := &Service{db: db}

	// Create user with specific IP restrictions
	user := createTestUserWithRoles(t, db, "testuser", []string{"192.168.1.100", "10.0.0.0/16"}, map[string][]string{})

	tests := []struct {
		name        string
		clientIP    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "close but not exact",
			clientIP:    "192.168.1.101",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
		{
			name:        "different subnet entirely",
			clientIP:    "172.16.0.1",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
		{
			name:        "just outside CIDR range",
			clientIP:    "10.1.0.1",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CheckIPRestriction(user, tt.clientIP)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheckIPRestriction_MultiRoleMerge tests multiple roles' IPs merged correctly
// Validates that IP restrictions from multiple roles are all merged (OR logic)
func TestCheckIPRestriction_MultiRoleMerge(t *testing.T) {

	db := setupTestDB(t)
	service := &Service{db: db}

	// Create user with multiple roles, each with different IP restrictions
	user := createTestUserWithRoles(t, db, "testuser", []string{}, map[string][]string{
		"operator":  {"192.168.1.0/24"},
		"viewer":    {"10.0.0.0/16"},
		"admin":     {"172.16.0.1-172.16.0.100"},
	})

	tests := []struct {
		name        string
		clientIP    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "IP from first role CIDR",
			clientIP: "192.168.1.50",
			wantErr:  false,
		},
		{
			name:     "IP from second role CIDR",
			clientIP: "10.0.50.1",
			wantErr:  false,
		},
		{
			name:     "IP from third role range",
			clientIP: "172.16.0.50",
			wantErr:  false,
		},
		{
			name:        "IP not in any role",
			clientIP:    "8.8.8.8",
			wantErr:     true,
			errContains: "不在允许列表中",
		},
		{
			name:     "boundary of first role",
			clientIP: "192.168.1.255",
			wantErr:  false,
		},
		{
			name:     "boundary of second role",
			clientIP: "10.0.255.255",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CheckIPRestriction(user, tt.clientIP)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheckIPRestriction_InvalidClientIP tests invalid client IP handling
// Validates that malformed client IPs return appropriate errors
func TestCheckIPRestriction_InvalidClientIP(t *testing.T) {

	db := setupTestDB(t)
	service := &Service{db: db}

	// Create user with IP restrictions
	user := createTestUserWithRoles(t, db, "testuser", []string{"192.168.1.100"}, map[string][]string{})

	tests := []struct {
		name        string
		clientIP    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty IP",
			clientIP:    "",
			wantErr:     true,
			errContains: "IP地址验证失败",
		},
		{
			name:        "malformed IP",
			clientIP:    "not-an-ip",
			wantErr:     true,
			errContains: "IP地址验证失败",
		},
		{
			name:        "IPv6 address",
			clientIP:    "::1",
			wantErr:     true,
			errContains: "IP地址验证失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CheckIPRestriction(user, tt.clientIP)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheckIPRestriction_AuditLogOnFailure tests audit logging on IP check failure
// Validates that IP restriction failures are logged to audit log per D-14
func TestCheckIPRestriction_AuditLogOnFailure(t *testing.T) {

	db := setupTestDB(t)
	service := &Service{db: db}

	// Create user with IP restrictions
	user := createTestUserWithRoles(t, db, "testuser", []string{"192.168.1.100"}, map[string][]string{})

	// This test will need audit logger setup
	// For now, just verify the error is returned correctly
	err := service.CheckIPRestriction(user, "192.168.1.101")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不在允许列表中")

	// TODO: Verify audit log entry was created
	// TODO: Assert log contains: UserID, Username, Action=ip_restriction_failed, ClientIP, Status=failure
}
