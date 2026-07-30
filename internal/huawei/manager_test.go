package huawei

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestNewManager_DefaultTLSPolicy 验证 SEC-003a：NewManager 默认 TLS 策略为
// MinTLSVersion=tls.VersionTLS12、InsecureSkipVerify=false（不再有 0x0301 / true 硬编码）。
func TestNewManager_DefaultTLSPolicy(t *testing.T) {
	m := NewManager(zap.NewNop(), nil)
	assert.Equal(t, uint16(tls.VersionTLS12), m.tlsMinVersion, "默认最低 TLS 版本应为 1.2")
	assert.False(t, m.tlsInsecureSkipVerify, "默认 InsecureSkipVerify 应为 false")
}

// TestSetTLSPolicy 验证 SEC-003a：显式策略被正确存储，minTLSVersion=0 归一化为 TLS 1.2。
func TestSetTLSPolicy(t *testing.T) {
	m := NewManager(zap.NewNop(), nil)
	m.SetTLSPolicy(false, tls.VersionTLS13, false)
	assert.Equal(t, uint16(tls.VersionTLS13), m.tlsMinVersion)
	assert.False(t, m.tlsInsecureSkipVerify)

	// 0 归一化为 TLS 1.2
	m.SetTLSPolicy(false, 0, false)
	assert.Equal(t, uint16(tls.VersionTLS12), m.tlsMinVersion)
}

// TestSetTLSPolicy_ProductionInsecureFatal 验证 SEC-003a：生产环境 InsecureSkipVerify=true → Fatal。
func TestSetTLSPolicy_ProductionInsecureFatal(t *testing.T) {
	original := fatalFunc
	defer func() { fatalFunc = original }()
	fatalFunc = func(l *zap.Logger, msg string, fields ...zap.Field) { panic(msg) }

	m := NewManager(zap.NewNop(), nil)
	assert.Panics(t, func() {
		m.SetTLSPolicy(true, 0, true)
	}, "生产环境 InsecureSkipVerify=true 应触发 Fatal")

	// 非生产环境允许 true（便于内网自签证书调试）
	assert.NotPanics(t, func() {
		m.SetTLSPolicy(true, 0, false)
	})
}

// TestNewHTTPClient_No3DESCipher 验证 SEC-003a：HTTPClient 的 CipherSuites 不含 3DES（SWEET32）。
func TestNewHTTPClient_No3DESCipher(t *testing.T) {
	c := NewHTTPClient("example.com", 443, 10*time.Second, false, tls.VersionTLS12, zap.NewNop())

	tr, ok := c.client.Transport.(*http.Transport)
	assert.True(t, ok, "Transport 应为 *http.Transport")
	tlsCfg := tr.TLSClientConfig
	assert.NotNil(t, tlsCfg)
	assert.Equal(t, uint16(tls.VersionTLS12), tlsCfg.MinVersion, "MinVersion 应为注入的 TLS 1.2")
	assert.False(t, tlsCfg.InsecureSkipVerify)

	for _, cs := range tlsCfg.CipherSuites {
		assert.NotEqual(t, uint16(tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA), uint16(cs),
			"密码套件不应包含 3DES_EDE_CBC_SHA（SWEET32）")
	}
}

// TestParseMinTLSVersion 验证 SEC-003a：字符串版本号解析为 tls 常量；TLS 1.0 被强制提升为 1.2。
func TestParseMinTLSVersion(t *testing.T) {
	cases := []struct {
		in   string
		want uint16
	}{
		{"1.2", tls.VersionTLS12},
		{"1.3", tls.VersionTLS13},
		{"1.1", tls.VersionTLS11},
		{"1.0", tls.VersionTLS12}, // 显式拒绝 1.0，归一化为 1.2
		{"", 0},                   // 空串返回 0，由调用方归一化
		{"unknown", 0},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ParseMinTLSVersion(tc.in), "input=%q", tc.in)
	}
}
