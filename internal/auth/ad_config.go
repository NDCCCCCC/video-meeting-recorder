package auth

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
