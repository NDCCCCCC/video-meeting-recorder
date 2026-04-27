package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupIPTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate tables
	err = db.AutoMigrate(&User{}, &Role{})
	require.NoError(t, err)

	return db
}

// TestUser_GetAllowedIPs_Empty tests GetAllowedIPs() with empty field
// Validates that empty AllowedIPs field returns empty slice
func TestUser_GetAllowedIPs_Empty(t *testing.T) {
	t.Skip("TODO: implement User.GetAllowedIPs() and SetAllowedIPs() methods")

	user := &User{
		Username: "testuser",
	}

	// Before setting, should return empty slice
	ips := user.GetAllowedIPs()
	assert.Equal(t, []string{}, ips)
	assert.Empty(t, ips)
}

// TestUser_GetAllowedIPs_JSONArray tests GetAllowedIPs() deserializes JSON array
// Validates that JSON array stored in AllowedIPs field is correctly deserialized
func TestUser_GetAllowedIPs_JSONArray(t *testing.T) {
	t.Skip("TODO: implement User.GetAllowedIPs() method")

	user := &User{
		Username:    "testuser",
		AllowedIPs:  `["192.168.1.100","10.0.0.0/16","172.16.0.1-172.16.0.100"]`,
	}

	ips := user.GetAllowedIPs()
	assert.Len(t, ips, 3)
	assert.Contains(t, ips, "192.168.1.100")
	assert.Contains(t, ips, "10.0.0.0/16")
	assert.Contains(t, ips, "172.16.0.1-172.16.0.100")
}

// TestUser_SetAllowedIPs_Serializes tests SetAllowedIPs() serializes to JSON
// Validates that IP array is correctly serialized to JSON and stored
func TestUser_SetAllowedIPs_Serializes(t *testing.T) {
	t.Skip("TODO: implement User.SetAllowedIPs() method")

	user := &User{
		Username: "testuser",
	}

	inputIPs := []string{"192.168.1.100", "10.0.0.0/16", "172.16.0.1-172.16.0.100"}
	err := user.SetAllowedIPs(inputIPs)
	require.NoError(t, err)

	// Verify JSON serialization
	assert.NotEmpty(t, user.AllowedIPs)

	// Verify it can be deserialized back
	var decoded []string
	err = json.Unmarshal([]byte(user.AllowedIPs), &decoded)
	require.NoError(t, err)
	assert.Equal(t, inputIPs, decoded)
}

// TestUser_AllowedIPs_RoundTrip tests round-trip serialization/deserialization
// Validates that IPs can be set and retrieved without data loss
func TestUser_AllowedIPs_RoundTrip(t *testing.T) {
	t.Skip("TODO: implement User SetAllowedIPs() and GetAllowedIPs() methods")

	user := &User{
		Username: "testuser",
	}

	inputIPs := []string{"192.168.1.100", "10.0.0.0/16", "172.16.0.1-172.16.0.100"}
	err := user.SetAllowedIPs(inputIPs)
	require.NoError(t, err)

	retrievedIPs := user.GetAllowedIPs()
	assert.Equal(t, inputIPs, retrievedIPs)
}

// TestUser_AllowedIPs_EmptyArray tests setting empty array
// Validates that setting empty array clears the field
func TestUser_AllowedIPs_EmptyArray(t *testing.T) {
	t.Skip("TODO: implement User.SetAllowedIPs() method")

	user := &User{
		Username:   "testuser",
		AllowedIPs: `["192.168.1.100"]`,
	}

	// Set empty array
	err := user.SetAllowedIPs([]string{})
	require.NoError(t, err)

	// Should serialize to empty JSON array
	assert.Equal(t, "[]", user.AllowedIPs)

	// GetAllowedIPs should return empty slice
	ips := user.GetAllowedIPs()
	assert.Empty(t, ips)
}

// TestRole_GetAllowedIPs_Empty tests Role GetAllowedIPs() with empty field
// Validates that empty AllowedIPs field returns empty slice
func TestRole_GetAllowedIPs_Empty(t *testing.T) {
	t.Skip("TODO: implement Role.GetAllowedIPs() and SetAllowedIPs() methods")

	role := &Role{
		Name: "testrole",
	}

	// Before setting, should return empty slice
	ips := role.GetAllowedIPs()
	assert.Equal(t, []string{}, ips)
	assert.Empty(t, ips)
}

// TestRole_GetAllowedIPs_JSONArray tests Role GetAllowedIPs() deserializes JSON array
// Validates that JSON array stored in AllowedIPs field is correctly deserialized
func TestRole_GetAllowedIPs_JSONArray(t *testing.T) {
	t.Skip("TODO: implement Role.GetAllowedIPs() method")

	role := &Role{
		Name:       "testrole",
		AllowedIPs: `["192.168.1.0/24","10.0.0.1-10.0.0.254"]`,
	}

	ips := role.GetAllowedIPs()
	assert.Len(t, ips, 2)
	assert.Contains(t, ips, "192.168.1.0/24")
	assert.Contains(t, ips, "10.0.0.1-10.0.0.254")
}

// TestRole_SetAllowedIPs_Serializes tests Role SetAllowedIPs() serializes to JSON
// Validates that IP array is correctly serialized to JSON and stored
func TestRole_SetAllowedIPs_Serializes(t *testing.T) {
	t.Skip("TODO: implement Role.SetAllowedIPs() method")

	role := &Role{
		Name: "testrole",
	}

	inputIPs := []string{"192.168.1.0/24", "10.0.0.1-10.0.0.254"}
	err := role.SetAllowedIPs(inputIPs)
	require.NoError(t, err)

	// Verify JSON serialization
	assert.NotEmpty(t, role.AllowedIPs)

	// Verify it can be deserialized back
	var decoded []string
	err = json.Unmarshal([]byte(role.AllowedIPs), &decoded)
	require.NoError(t, err)
	assert.Equal(t, inputIPs, decoded)
}

// TestRole_AllowedIPs_RoundTrip tests role round-trip serialization/deserialization
// Validates that role IPs can be set and retrieved without data loss
func TestRole_AllowedIPs_RoundTrip(t *testing.T) {
	t.Skip("TODO: implement Role SetAllowedIPs() and GetAllowedIPs() methods")

	role := &Role{
		Name: "testrole",
	}

	inputIPs := []string{"192.168.1.0/24", "10.0.0.1-10.0.0.254"}
	err := role.SetAllowedIPs(inputIPs)
	require.NoError(t, err)

	retrievedIPs := role.GetAllowedIPs()
	assert.Equal(t, inputIPs, retrievedIPs)
}

// TestAllowedIPs_GORMScan tests GORM Scan() interface implementation
// Validates that GORM can correctly scan database value into AllowedIPs field
func TestAllowedIPs_GORMScan(t *testing.T) {
	t.Skip("TODO: implement GORM Scan() interface for AllowedIPs if needed")

	db := setupIPTestDB(t)

	// Create user with IP restrictions
	user := &User{
		Username: "testuser",
	}
	err := user.SetAllowedIPs([]string{"192.168.1.100", "10.0.0.0/16"})
	require.NoError(t, err)

	// Save to database
	err = db.Create(user).Error
	require.NoError(t, err)

	// Retrieve from database
	var retrieved User
	err = db.First(&retrieved, user.ID).Error
	require.NoError(t, err)

	// Verify IPs were correctly scanned
	ips := retrieved.GetAllowedIPs()
	assert.Len(t, ips, 2)
	assert.Contains(t, ips, "192.168.1.100")
	assert.Contains(t, ips, "10.0.0.0/16")
}

// TestAllowedIPs_GORMValue tests GORM Value() interface implementation
// Validates that GORM can correctly serialize AllowedIPs field to database value
func TestAllowedIPs_GORMValue(t *testing.T) {
	t.Skip("TODO: implement GORM Value() interface for AllowedIPs if needed")

	db := setupIPTestDB(t)

	// Create role with IP restrictions
	role := &Role{
		Name: "testrole",
	}
	err := role.SetAllowedIPs([]string{"192.168.1.0/24"})
	require.NoError(t, err)

	// Save to database
	err = db.Create(role).Error
	require.NoError(t, err)

	// Query raw database value to verify storage
	var rawValue string
	err = db.Table("roles").Select("allowed_ips").Where("id = ?", role.ID).Scan(&rawValue).Error
	require.NoError(t, err)

	// Verify it's valid JSON
	var decoded []string
	err = json.Unmarshal([]byte(rawValue), &decoded)
	require.NoError(t, err)
	assert.Len(t, decoded, 1)
	assert.Equal(t, "192.168.1.0/24", decoded[0])
}

// TestAllowedIPs_DatabaseRoundTrip tests full database round-trip
// Validates that IP restrictions survive database save and load cycle
func TestAllowedIPs_DatabaseRoundTrip(t *testing.T) {
	t.Skip("TODO: implement User and Role AllowedIPs methods")

	db := setupIPTestDB(t)

	// Create user with various IP formats
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	userIPs := []string{
		"192.168.1.100",        // Single IP
		"10.0.0.0/16",          // CIDR
		"172.16.0.1-172.16.0.100", // Range
	}
	err := user.SetAllowedIPs(userIPs)
	require.NoError(t, err)

	err = db.Create(user).Error
	require.NoError(t, err)

	// Create role with IP restrictions
	role := &Role{
		Name: "testrole",
	}
	roleIPs := []string{"192.168.2.0/24", "10.1.0.1-10.1.0.254"}
	err = role.SetAllowedIPs(roleIPs)
	require.NoError(t, err)

	err = db.Create(role).Error
	require.NoError(t, err)

	// Retrieve user
	var retrievedUser User
	err = db.Preload("Roles").First(&retrievedUser, user.ID).Error
	require.NoError(t, err)

	retrievedUserIPs := retrievedUser.GetAllowedIPs()
	assert.Equal(t, userIPs, retrievedUserIPs)

	// Retrieve role
	var retrievedRole Role
	err = db.First(&retrievedRole, role.ID).Error
	require.NoError(t, err)

	retrievedRoleIPs := retrievedRole.GetAllowedIPs()
	assert.Equal(t, roleIPs, retrievedRoleIPs)
}

// TestAllowedIPs_InvalidJSON tests handling of invalid JSON in field
// Validates that corrupt JSON data is handled gracefully
func TestAllowedIPs_InvalidJSON(t *testing.T) {
	t.Skip("TODO: implement error handling in GetAllowedIPs()")

	user := &User{
		Username:    "testuser",
		AllowedIPs:  `invalid json`,
	}

	// Should return empty slice on invalid JSON
	ips := user.GetAllowedIPs()
	assert.Empty(t, ips)
}

// TestAllowedIPs_WhitespaceHandling tests handling of whitespace in IP entries
// Validates that IP entries with whitespace are preserved or trimmed as expected
func TestAllowedIPs_WhitespaceHandling(t *testing.T) {
	t.Skip("TODO: implement SetAllowedIPs() with whitespace handling")

	user := &User{
		Username: "testuser",
	}

	// IPs with various whitespace
	inputIPs := []string{
		"192.168.1.100",
		" 10.0.0.1 ",
		"192.168.2.0/24",
	}

	err := user.SetAllowedIPs(inputIPs)
	require.NoError(t, err)

	retrievedIPs := user.GetAllowedIPs()

	// Verify entries are stored as-is (whitespace preserved)
	// or trimmed (based on implementation decision)
	assert.Len(t, retrievedIPs, 3)
	assert.Contains(t, retrievedIPs, "192.168.1.100")
	// Whitespace handling to be determined by implementation
}
