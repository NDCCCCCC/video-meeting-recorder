package auth

import "context"

// ADUser represents an Active Directory user
type ADUser struct {
	Username            string
	DN                  string
	ObjectGUID          string
	Email               string
	DisplayName         string
	Department          string
	UserPrincipalName   string
	UserAccountControl  uint32
}

// IsDisabled checks if the AD account is disabled
// ACCOUNTDISABLE flag is 0x0002
func (u *ADUser) IsDisabled() bool {
	return u.UserAccountControl&0x0002 != 0
}

// ADAuthConfig holds AD connection configuration
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

	// AllowAutoCreate controls whether AD users are automatically created on first login
	// If false, only pre-existing AD users in the system can log in (whitelist mode)
	AllowAutoCreate bool `mapstructure:"allow_auto_create" json:"allow_auto_create"` // Default: true
}

// ADConfigValidationResult holds the result of AD configuration validation
type ADConfigValidationResult struct {
	Valid         bool     `json:"valid"`
	Level         int      `json:"level"`          // 0-4: format, network, auth, functionality
	Errors        []string `json:"errors"`
	Warnings      []string `json:"warnings"`
	ServerInfo    string   `json:"server_info"`
	ResponseTime  int64    `json:"response_time"` // milliseconds
}

// ADUserLookupResult holds the result of an AD user lookup
type ADUserLookupResult struct {
	Found      bool   `json:"found"`
	Username   string `json:"username"`
	Email      string `json:"email,omitempty"`
	FullName   string `json:"full_name,omitempty"`
	Department string `json:"department,omitempty"`
	UPN        string `json:"upn,omitempty"`
	DN         string `json:"dn,omitempty"`
	Disabled   bool   `json:"disabled,omitempty"`
	Message    string `json:"message,omitempty"`
}

// Authenticator defines the authentication interface (strategy pattern per Spike 003)
type Authenticator interface {
	// Login authenticates a user and returns a login response.
	// ctx 用于把请求上下文（RequestID/TraceID）传递到审计日志，保证审计与
	// 请求链路可串联；为 nil 时调用方需自行降级处理。
	Login(ctx context.Context, req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error)

	// Logout logs out a user by revoking their token
	Logout(token string) error

	// ValidateToken validates a token and returns the associated user
	ValidateToken(token string) (*UserDTO, error)

	// Name returns the authenticator name
	Name() string
}

