package auth

import (
	"testing"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/stretchr/testify/assert"
)

// TestValidateIP_ValidIP tests valid IPv4 addresses
// Validates that standard IPv4 addresses pass validation per D-06
func TestValidateIP_ValidIP(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{
			name:    "valid private IP",
			ip:      "192.168.1.100",
			wantErr: false,
		},
		{
			name:    "valid public IP",
			ip:      "8.8.8.8",
			wantErr: false,
		},
		{
			name:    "valid localhost",
			ip:      "127.0.0.1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIP(tt.ip)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateIP_InvalidIP tests invalid IP formats
// Validates that malformed IP addresses are rejected
func TestValidateIP_InvalidIP(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{
			name:    "empty string",
			ip:      "",
			wantErr: true,
		},
		{
			name:    "incomplete IP",
			ip:      "192.168.1",
			wantErr: true,
		},
		{
			name:    "out of range octet",
			ip:      "192.168.1.256",
			wantErr: true,
		},
		{
			name:    "non-numeric",
			ip:      "192.168.1.abc",
			wantErr: true,
		},
		{
			name:    "too many octets",
			ip:      "192.168.1.1.1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIP(tt.ip)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateIP_IPv6Rejected tests IPv6 rejection per D-09
// Validates that IPv6 addresses are explicitly rejected as per decision D-09.
// Phase 19 D8: 验证 sentinel 而非 string-match——避免"中文消息变更触发测试回归"
// 这一 anti-pattern;errors.Is(err, ErrInvalidInput) 是稳定的契约。
func TestValidateIP_IPv6Rejected(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name string
		ip   string
	}{
		{
			name: "IPv6 full format",
			ip:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		},
		{
			name: "IPv6 compressed",
			ip:   "2001:db8::1",
		},
		{
			name: "IPv6 loopback",
			ip:   "::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIP(tt.ip)
			assert.Error(t, err)
			assert.True(t, apperrors.Is(err, apperrors.ErrInvalidInput),
				"expected ErrInvalidInput sentinel, got %v", err)
		})
	}
}

// TestValidateCIDR_ValidCIDR tests valid CIDR ranges per D-07
// Validates that CIDR notation is accepted for IP ranges
func TestValidateCIDR_ValidCIDR(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{
			name:    "/24 subnet",
			cidr:    "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "/32 single IP",
			cidr:    "192.168.1.100/32",
			wantErr: false,
		},
		{
			name:    "/16 subnet",
			cidr:    "10.0.0.0/16",
			wantErr: false,
		},
		{
			name:    "/8 large network",
			cidr:    "10.0.0.0/8",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCIDR(tt.cidr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateCIDR_InvalidCIDR tests invalid CIDR formats
// Validates that malformed CIDR notation is rejected
func TestValidateCIDR_InvalidCIDR(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{
			name:    "missing prefix",
			cidr:    "192.168.1.0",
			wantErr: true,
		},
		{
			name:    "invalid prefix length",
			cidr:    "192.168.1.0/33",
			wantErr: true,
		},
		{
			name:    "negative prefix",
			cidr:    "192.168.1.0/-1",
			wantErr: true,
		},
		{
			name:    "invalid IP in CIDR",
			cidr:    "256.168.1.0/24",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCIDR(tt.cidr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateIPRange_ValidRange tests valid IP ranges per D-08
// Validates that start-end IP range notation is accepted
func TestValidateIPRange_ValidRange(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name    string
		ipRange string
		wantErr bool
	}{
		{
			name:    "small range",
			ipRange: "192.168.1.100-192.168.1.200",
			wantErr: false,
		},
		{
			name:    "single IP range",
			ipRange: "192.168.1.100-192.168.1.100",
			wantErr: false,
		},
		{
			name:    "large range",
			ipRange: "10.0.0.1-10.0.0.254",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIPRange(tt.ipRange)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateIPRange_InvalidRange tests invalid IP range formats
// Validates that malformed IP range notation is rejected
func TestValidateIPRange_InvalidRange(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name    string
		ipRange string
		wantErr bool
	}{
		{
			name:    "missing dash",
			ipRange: "192.168.1.100 192.168.1.200",
			wantErr: true,
		},
		{
			name:    "only one IP",
			ipRange: "192.168.1.100",
			wantErr: true,
		},
		{
			name:    "too many dashes",
			ipRange: "192.168.1-100-192.168.1.200",
			wantErr: true,
		},
		{
			name:    "invalid start IP",
			ipRange: "192.168.1.256-192.168.1.200",
			wantErr: true,
		},
		{
			name:    "invalid end IP",
			ipRange: "192.168.1.100-192.168.1.256",
			wantErr: true,
		},
		{
			name:    "end before start",
			ipRange: "192.168.1.200-192.168.1.100",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIPRange(tt.ipRange)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsIPAllowed_SingleIP tests single IP matching
// Validates that exact IP match returns true
func TestIsIPAllowed_SingleIP(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name        string
		clientIP    string
		allowedList []string
		wantAllowed bool
		wantErr     bool
	}{
		{
			name:        "exact match",
			clientIP:    "192.168.1.100",
			allowedList: []string{"192.168.1.100"},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:        "no match",
			clientIP:    "192.168.1.101",
			allowedList: []string{"192.168.1.100"},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:        "match in list",
			clientIP:    "192.168.1.100",
			allowedList: []string{"192.168.1.50", "192.168.1.100", "192.168.1.150"},
			wantAllowed: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := validator.IsIPAllowed(tt.clientIP, tt.allowedList)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantAllowed, allowed)
		})
	}
}

// TestIsIPAllowed_CIDRRange tests CIDR range matching
// Validates that IPs within CIDR ranges are allowed
func TestIsIPAllowed_CIDRRange(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name        string
		clientIP    string
		allowedList []string
		wantAllowed bool
		wantErr     bool
	}{
		{
			name:        "within /24 subnet",
			clientIP:    "192.168.1.50",
			allowedList: []string{"192.168.1.0/24"},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:        "outside /24 subnet",
			clientIP:    "192.168.2.50",
			allowedList: []string{"192.168.1.0/24"},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:        "network address",
			clientIP:    "192.168.1.0",
			allowedList: []string{"192.168.1.0/24"},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:        "broadcast address",
			clientIP:    "192.168.1.255",
			allowedList: []string{"192.168.1.0/24"},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:        "/32 single IP CIDR",
			clientIP:    "192.168.1.100",
			allowedList: []string{"192.168.1.100/32"},
			wantAllowed: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := validator.IsIPAllowed(tt.clientIP, tt.allowedList)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantAllowed, allowed)
		})
	}
}

// TestIsIPAllowed_IPRange tests IP range matching per D-08
// Validates that IPs within start-end ranges are allowed
func TestIsIPAllowed_IPRange(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name        string
		clientIP    string
		allowedList []string
		wantAllowed bool
		wantErr     bool
	}{
		{
			name:        "within range",
			clientIP:    "192.168.1.150",
			allowedList: []string{"192.168.1.100-192.168.1.200"},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:        "below range",
			clientIP:    "192.168.1.50",
			allowedList: []string{"192.168.1.100-192.168.1.200"},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:        "above range",
			clientIP:    "192.168.1.250",
			allowedList: []string{"192.168.1.100-192.168.1.200"},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:        "lower boundary",
			clientIP:    "192.168.1.100",
			allowedList: []string{"192.168.1.100-192.168.1.200"},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:        "upper boundary",
			clientIP:    "192.168.1.200",
			allowedList: []string{"192.168.1.100-192.168.1.200"},
			wantAllowed: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := validator.IsIPAllowed(tt.clientIP, tt.allowedList)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantAllowed, allowed)
		})
	}
}

// TestIsIPAllowed_NoMatch tests IP not in allowed list
// Validates that IPs not matching any rule return false
func TestIsIPAllowed_NoMatch(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name        string
		clientIP    string
		allowedList []string
		wantAllowed bool
		wantErr     bool
	}{
		{
			name:        "different subnet",
			clientIP:    "10.0.0.1",
			allowedList: []string{"192.168.1.0/24"},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:        "multiple rules no match",
			clientIP:    "192.168.2.1",
			allowedList: []string{"192.168.1.0/24", "10.0.0.0/16", "172.16.0.0-172.16.0.100"},
			wantAllowed: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := validator.IsIPAllowed(tt.clientIP, tt.allowedList)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantAllowed, allowed)
		})
	}
}

// TestIsIPAllowed_EmptyList tests empty allowed list behavior
// Validates that empty allowed list permits all IPs (no restriction)
func TestIsIPAllowed_EmptyList(t *testing.T) {
	validator := &IPValidator{}

	tests := []struct {
		name        string
		clientIP    string
		allowedList []string
		wantAllowed bool
		wantErr     bool
	}{
		{
			name:        "empty list allows all",
			clientIP:    "192.168.1.100",
			allowedList: []string{},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:        "nil list allows all",
			clientIP:    "10.0.0.1",
			allowedList: nil,
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:        "list with only empty entries",
			clientIP:    "172.16.0.1",
			allowedList: []string{"", "", ""},
			wantAllowed: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := validator.IsIPAllowed(tt.clientIP, tt.allowedList)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantAllowed, allowed)
		})
	}
}
