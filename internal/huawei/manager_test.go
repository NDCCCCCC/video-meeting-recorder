package huawei

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// stubDB 是 Manager.DBInterface 的最小内存实现，用于绕开真实数据库，
// 把 createClient 路径（Manager.GetClient → m.db.GetHuaweiConfig）拉到测试里来。
// 桥接测试需要它把 Server/Port 指向 httptest.Server，这样能验证
// Manager.SetTLSServerName → Config.tlsServerName → NewHTTPClient.tls.Config.ServerName
// 的完整透传链。
type stubDB struct {
	cfg *HuaweiConfigDB
}

func (s *stubDB) GetHuaweiConfig(configID uint) (*HuaweiConfigDB, error) {
	if s.cfg == nil {
		return nil, errors.New("stubDB: no config injected")
	}
	return s.cfg, nil
}

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
	c := NewHTTPClient("example.com", 443, 10*time.Second, false, tls.VersionTLS12, nil, "", zap.NewNop())

	tr, ok := c.client.Transport.(*http.Transport)
	assert.True(t, ok, "Transport 应为 *http.Transport")
	tlsCfg := tr.TLSClientConfig
	assert.NotNil(t, tlsCfg)
	assert.Equal(t, uint16(tls.VersionTLS12), tlsCfg.MinVersion, "MinVersion 应为注入的 TLS 1.2")
	assert.False(t, tlsCfg.InsecureSkipVerify)
	assert.Nil(t, tlsCfg.RootCAs, "未传入 caCertPool 时 RootCAs 应为 nil（系统 CA 兜底）")

	for _, cs := range tlsCfg.CipherSuites {
		assert.NotEqual(t, tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA, cs,
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

// ============================================================================
// SEC-003a: 私有 CA bundle 加载 + TLS 注入 — 5 场景回归测试（场景 ①③④⑤）
// ============================================================================

// generateSelfSignedCert 生成一个 self-signed 证书，返回 certPEM / keyPEM。
// 套用 fillCertDefaults 保证 SerialNumber / NotBefore / NotAfter / KeyUsage / BasicConstraints
// 默认值合理，caller 仍可显式覆盖。
func generateSelfSignedCert(t *testing.T, tmpl *x509.Certificate) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	fillCertDefaults(tmpl)

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM
}

// fillCertDefaults 兜底一些常用字段，避免每个 caller 重复设置。
func fillCertDefaults(tmpl *x509.Certificate) {
	if tmpl.SerialNumber == nil {
		tmpl.SerialNumber = big.NewInt(time.Now().UnixNano())
	}
	if tmpl.NotBefore.IsZero() {
		tmpl.NotBefore = time.Now().Add(-1 * time.Hour)
	}
	if tmpl.NotAfter.IsZero() {
		tmpl.NotAfter = time.Now().Add(24 * time.Hour)
	}
	if tmpl.KeyUsage == 0 {
		tmpl.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	}
	if tmpl.IsCA {
		tmpl.BasicConstraintsValid = true
	}
}

// parseFirstCertDER 从 PEM 字节中解析首个 CERTIFICATE 块并返回 DER。
func parseFirstCertDER(t *testing.T, pemBytes []byte) []byte {
	t.Helper()
	block, _ := pem.Decode(pemBytes)
	require.NotNil(t, block, "PEM decode 不应返回 nil block")
	require.Equal(t, "CERTIFICATE", block.Type)
	return block.Bytes
}

// parsePKCS1PrivateKey 从 PEM 字节解析 RSA PRIVATE KEY。
func parsePKCS1PrivateKey(t *testing.T, pemBytes []byte) *rsa.PrivateKey {
	t.Helper()
	block, _ := pem.Decode(pemBytes)
	require.NotNil(t, block)
	require.Equal(t, "RSA PRIVATE KEY", block.Type)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	return key
}

// signServerCert 用 caCert + caKey 签发一张 server 证书，CN + SAN + IP 都满足 httptest localhost:port。
func signServerCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, []byte) {
	t.Helper()
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		Subject:     pkix.Name{CommonName: "localhost"},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	fillCertDefaults(tmpl)
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &serverKey.PublicKey, caKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, serverKey, der
}

// startTLSServer 启动一个 httptest TLS server，持有 serverCert + serverKey。
func startTLSServer(t *testing.T, handler http.Handler, serverCert *x509.Certificate, serverKey *rsa.PrivateKey) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{
			{Certificate: [][]byte{serverCert.Raw}, PrivateKey: serverKey, Leaf: serverCert},
		},
		MinVersion: tls.VersionTLS12,
	}
	srv.StartTLS()
	return srv
}

// fakeHuaweiAPIHandler 返回 success=1 的最小华为 API 响应（用于触发握手成功路径）。
func fakeHuaweiAPIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": 1, "data": ""})
	})
}

// makeHuaweiClient 为本地 TLS server 构造一个 Huawei client，注入 caPool + 开发环境 bypass。
// tlsServerName="" 表示不显式设 ServerName（让 Go x509 verifier 用 dial 地址做 hostname
// 校验）；非空字符串会注入到 tls.Config.ServerName，让 Go 走 dNSName 匹配路径（这是 REDO
// 修复的核心：IP-only dial + DNS-only SAN cert 必须靠 ServerName 完成 hostname 校验）。
func makeHuaweiClient(t *testing.T, srv *httptest.Server, caPool *x509.CertPool, tlsServerName string) *HuaweiClient {
	t.Helper()
	urlStr := strings.TrimPrefix(srv.URL, "https://")
	parts := strings.SplitN(urlStr, ":", 2)
	require.Len(t, parts, 2, "URL 应含端口")
	port, err := strconv.Atoi(parts[1])
	require.NoError(t, err)

	client := NewHuaweiClient(&Config{
		Server:             parts[0],
		Port:               port,
		APITimeout:         5 * time.Second,
		SessionTimeout:     1 * time.Minute,
		KeepAliveInterval:  30 * time.Second,
		InsecureSkipVerify: false,
		MinTLSVersion:      tls.VersionTLS12,
		caCertPool:         caPool,
		tlsServerName:      tlsServerName,
	}, zap.NewNop())
	// 走 development 环境绕过 SEC-013 出站 URL 白名单——本测试不验证白名单。
	client.httpClient.SetOutboundURLAllowlist(nil, "development")
	return client
}

// TestSetCABundle_ValidPEM 场景 ①：valid PEM + httptest self-signed server，握手成功。
// SEC-003a-05 第 ① 条要求：normal PEM + httptest 自签 server → handshake 成功。
func TestSetCABundle_ValidPEM(t *testing.T) {
	dir := t.TempDir()

	// 1) 生成 self-signed CA + 写文件
	caPEM, caKeyPEM := generateSelfSignedCert(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "test-ca-valid"},
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	caPath := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(caPath, caPEM, 0o600))

	// 2) 用该 CA 签 server cert
	caCert, err := x509.ParseCertificate(parseFirstCertDER(t, caPEM))
	require.NoError(t, err)
	caKey := parsePKCS1PrivateKey(t, caKeyPEM)
	serverCert, serverKey, _ := signServerCert(t, caCert, caKey)

	// 3) 启动 httptest TLS server
	srv := startTLSServer(t, fakeHuaweiAPIHandler(), serverCert, serverKey)
	defer srv.Close()

	// 4) SetCABundle 应成功
	m := NewManager(zap.NewNop(), nil)
	require.NoError(t, m.SetCABundle(caPath), "valid PEM 应成功加载")
	require.NotNil(t, m.caCertPool, "caCertPool 应已被发布")
	assert.False(t, m.tlsInsecureSkipVerify, "SetCABundle 不应改写 TLS 策略")

	// 5) 真实握手：InsecureSkipVerify=false + 注入 CA → 必须成功
	client := makeHuaweiClient(t, srv, m.caCertPool, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.httpClient.Post(ctx, "Web_RequestSessionID", nil)
	require.NoError(t, err, "TLS 握手 + HTTP POST 应全部成功（InsecureSkipVerify=false + 注入 CA）")
	assert.Equal(t, 1, resp.Success)
}

// TestSetCABundle_InvalidOrMissing 场景 ②：覆盖文件不存在 / 损坏 PEM / 末尾残留 / 0 证书 / 错类型 / 坏 DER。
// SEC-003a-05 第 ② 条要求：PEM 损坏 / 不存在 → SetCABundle 返回 wrapped error，错误文本含 path。
func TestSetCABundle_InvalidOrMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, data, 0o600))
		return p
	}

	// 缺失文件 → os.PathError
	t.Run("missing_file", func(t *testing.T) {
		missing := filepath.Join(dir, "does-not-exist.pem")
		m := NewManager(zap.NewNop(), nil)
		err := m.SetCABundle(missing)
		require.Error(t, err)
		assert.Contains(t, err.Error(), missing, "错误文本应包含 path")
		var pathErr *os.PathError
		assert.True(t, errors.As(err, &pathErr), "wrapped cause 应保留 os.PathError")
		assert.Nil(t, m.caCertPool)
	})

	// 损坏 PEM → 0 证书
	t.Run("malformed_pem", func(t *testing.T) {
		bad := writeFile("bad.pem", []byte("definitely not a PEM file"))
		m := NewManager(zap.NewNop(), nil)
		err := m.SetCABundle(bad)
		require.Error(t, err)
		assert.Contains(t, err.Error(), bad)
		assert.Nil(t, m.caCertPool)
	})

	// 0 证书文件
	t.Run("zero_certs", func(t *testing.T) {
		empty := writeFile("empty.pem", []byte(""))
		m := NewManager(zap.NewNop(), nil)
		err := m.SetCABundle(empty)
		require.Error(t, err)
		assert.Contains(t, err.Error(), empty)
		assert.Contains(t, err.Error(), "未找到任何 CERTIFICATE 块")
		assert.Nil(t, m.caCertPool)
	})

	// 末尾残留非空白字节
	t.Run("trailing_garbage", func(t *testing.T) {
		caPEM, _ := generateSelfSignedCert(t, &x509.Certificate{
			Subject: pkix.Name{CommonName: "test-ca-trail"},
			IsCA:    true,
		})
		caPEM = append(caPEM, []byte("\nxx leftover junk data")...)
		bad := writeFile("trail.pem", caPEM)
		m := NewManager(zap.NewNop(), nil)
		err := m.SetCABundle(bad)
		require.Error(t, err)
		assert.Contains(t, err.Error(), bad)
		assert.Contains(t, err.Error(), "末尾残留非空白字节")
		assert.Nil(t, m.caCertPool)
	})

	// 错 PEM 块类型（不是 CERTIFICATE）
	t.Run("wrong_pem_type", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		keyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		})
		wrong := writeFile("wrong.pem", keyPEM)
		m := NewManager(zap.NewNop(), nil)
		err = m.SetCABundle(wrong)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "不支持的 PEM 块类型")
		assert.Contains(t, err.Error(), wrong)
		assert.Nil(t, m.caCertPool)
	})

	// 合法 PEM 头但坏 DER 内容
	t.Run("broken_der", func(t *testing.T) {
		bad := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: []byte("not a valid DER sequence"),
		})
		broken := writeFile("broken.pem", bad)
		m := NewManager(zap.NewNop(), nil)
		err := m.SetCABundle(broken)
		require.Error(t, err)
		assert.Contains(t, err.Error(), broken)
		assert.Contains(t, err.Error(), "x509.ParseCertificate 错误")
		assert.Nil(t, m.caCertPool)
	})
}

// TestSetCABundle_EmptyPath 场景 ③：空字符串（含纯空白）→ caCertPool=nil。
// SEC-003a-05 第 ③ 条要求。
func TestSetCABundle_EmptyPath(t *testing.T) {
	m := NewManager(zap.NewNop(), nil)
	// 先假设有 pool
	m.caCertPool = x509.NewCertPool()

	for _, in := range []string{"", "   ", "\t\n"} {
		err := m.SetCABundle(in)
		require.NoError(t, err, "空/空白路径应不报错：input=%q", in)
		assert.Nil(t, m.caCertPool, "空路径后 caCertPool 应被设为 nil：input=%q", in)
	}
}

// TestNewHTTPClient_CertPoolBranches 场景 ⑤：NewHTTPClient 签名变化后两条分支覆盖。
// SEC-003a-05 第 ⑤ 条要求。
func TestNewHTTPClient_CertPoolBranches(t *testing.T) {
	t.Run("nil_pool_means_system_CA", func(t *testing.T) {
		c := NewHTTPClient("example.com", 443, 5*time.Second, false, tls.VersionTLS12, nil, "", zap.NewNop())
		tr := c.client.Transport.(*http.Transport)
		require.NotNil(t, tr.TLSClientConfig)
		assert.Nil(t, tr.TLSClientConfig.RootCAs, "nil pool → RootCAs 必须为 nil（系统 CA 兜底）")
		assert.False(t, tr.TLSClientConfig.InsecureSkipVerify, "InsecureSkipVerify 应保持 false")
	})

	t.Run("non_nil_pool_is_assigned_by_identity", func(t *testing.T) {
		pool := x509.NewCertPool()
		c := NewHTTPClient("example.com", 443, 5*time.Second, false, tls.VersionTLS12, pool, "", zap.NewNop())
		tr := c.client.Transport.(*http.Transport)
		require.NotNil(t, tr.TLSClientConfig)
		require.NotNil(t, tr.TLSClientConfig.RootCAs, "非 nil pool 必须被分配")
		assert.Same(t, pool, tr.TLSClientConfig.RootCAs, "RootCAs 应为传入的指针实例")
	})
}

// TestCABundle_ServerAndRootChain 场景 ④：单一 PEM 文件同时含 server cert +
// 自签根，握手应成功。这是 SEC-003a-05 第 ④ 条要求。
func TestCABundle_ServerAndRootChain(t *testing.T) {
	dir := t.TempDir()

	// 1) 自签 CA
	caPEM, caKeyPEM := generateSelfSignedCert(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "huawei-chain-ca"},
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	caCert, err := x509.ParseCertificate(parseFirstCertDER(t, caPEM))
	require.NoError(t, err)
	caKey := parsePKCS1PrivateKey(t, caKeyPEM)

	// 2) CA 签 server cert
	serverCert, serverKey, _ := signServerCert(t, caCert, caKey)
	serverPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCert.Raw})

	// 3) 拼成 server + root 单一 PEM（模拟 huawei-10.62.10.3-ca.pem 的实际格式）
	combinedPEM := append(append([]byte{}, serverPEM...), caPEM...)
	combinedPath := filepath.Join(dir, "chain.pem")
	require.NoError(t, os.WriteFile(combinedPath, combinedPEM, 0o600))

	// 4) httptest server（其 serverCert 与 combined 中的第一块是一致的）
	srv := startTLSServer(t, fakeHuaweiAPIHandler(), serverCert, serverKey)
	defer srv.Close()

	// 5) SetCABundle 应成功并把两个 CERTIFICATE 都加入 pool
	m := NewManager(zap.NewNop(), nil)
	require.NoError(t, m.SetCABundle(combinedPath), "链式 PEM 应被正确解析")
	require.NotNil(t, m.caCertPool)

	// 6) InsecureSkipVerify=false 时凭 pool 内的 root 信任
	client := makeHuaweiClient(t, srv, m.caCertPool, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.httpClient.Post(ctx, "Web_RequestSessionID", nil)
	require.NoError(t, err, "链式 PEM 加载后握手应成功（InsecureSkipVerify=false）")
	assert.Equal(t, 1, resp.Success)
}

// TestNewHTTPClient_ServerNameSet_HostnameVerifiesByDNS — REDO: 修复 IP-only 拨号
// 场景下的 hostname/SAN 校验。
//
// 模拟生产现场（huawei-10.62.10.3 server cert 实际 SAN 只有 'DNS:*.tp.huawei.com'，
// 无 iPAddress SAN）。客户端 IP-only dial 时若不显式设 ServerName，Go x509 verifier
// 用 dial 地址做 hostname 校验 → 失败（"doesn't contain any IP SANs"）。一旦显式让
// tls.Config.ServerName="vct.tp.huawei.com"，Go 走 dNSName 匹配路径 → *.tp.huawei.com
// 通配符命中 → 握手成功（InsecureSkipVerify=false 与 full chain 校验保持）。
//
// RED 阶段：NewHTTPClient 尚不支持 tlsServerName 入参 → 编译失败，确认行为变更的
// 真实性；GREEN 阶段：注入 tls.Config.ServerName 后握手成功。
func TestNewHTTPClient_ServerNameSet_HostnameVerifiesByDNS(t *testing.T) {
	dir := t.TempDir()

	// 1) 自签 root + 签发 DNS-only SAN server cert（CN / DNS = vct.tp.huawei.com；
	//    无 IPAddresses 字段，模拟生产 cert 结构）。
	rootPEM, rootKeyPEM := generateSelfSignedCert(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "huawei-redo-root"},
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	root, err := x509.ParseCertificate(parseFirstCertDER(t, rootPEM))
	require.NoError(t, err)
	rootKey := parsePKCS1PrivateKey(t, rootKeyPEM)
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	serverTmpl := &x509.Certificate{
		Subject:  pkix.Name{CommonName: "vct.tp.huawei.com"},
		DNSNames: []string{"vct.tp.huawei.com"},
		// 故意不设 IPAddresses —— 钉住"无 IP SAN 时 IP-only dial 必须用 ServerName"。
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	fillCertDefaults(serverTmpl)
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, root, &serverKey.PublicKey, rootKey)
	require.NoError(t, err)
	serverCert, err := x509.ParseCertificate(serverDER)
	require.NoError(t, err)

	// 2) root 写文件 → SetCABundle 注入 pool
	rootPath := filepath.Join(dir, "root.pem")
	require.NoError(t, os.WriteFile(rootPath, rootPEM, 0o600))
	m := NewManager(zap.NewNop(), nil)
	require.NoError(t, m.SetCABundle(rootPath))
	require.NotNil(t, m.caCertPool)

	// 3) 显式注册 ServerName —— 模拟生产 huawei.tls_server_name 配置
	m.SetTLSServerName("vct.tp.huawei.com")

	// 4) httptest TLS server（cert 与 generateServerCert 一致）
	srv := startTLSServer(t, fakeHuaweiAPIHandler(), serverCert, serverKey)
	defer srv.Close()

	// 5) IP-only dial + ServerName="vct.tp.huawei.com" → 握手必须成功（InsecureSkipVerify=false 保持）
	client := makeHuaweiClient(t, srv, m.caCertPool, "vct.tp.huawei.com")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.httpClient.Post(ctx, "Web_RequestSessionID", nil)
	require.NoError(t, err, "DNS SAN match：握手应成功（InsecureSkipVerify=false 保持）")
	assert.Equal(t, 1, resp.Success)
}

// TestNewHTTPClient_NoServerName_IPClient_FailsHostnameSAN — REDO 回归测试：钉住真实
// bug 现状——server cert SAN 仅含 dNSName、客户端 IP-only dial 且未显式设 ServerName
// 时，Go x509 verifier 应在 hostname 校验阶段失败，而非误指 trust anchor
// （"unknown authority"）。这是 PRIOR debug 错误假设的反证：真实失败模式是 hostname
// 层而非 CA chain。后续 SetCABundle 不应再将此类失败误归类为 trust anchor 问题。
//
// 断言只钉**语义指纹**（Go x509 错误栈 + 反证"unknown authority"），不钉易随
// Go 版本变化的错误文案——早期版本的 Go x509 在 IP-only dial + 无 iPAddress SAN 时报
// "doesn't contain any IP SANs"，但 Go 1.20+ 走 dNSName vs IP 路径时报
// "is valid for vct.tp.huawei.com, not 127.0.0.1"。两者都属于 hostname 校验失败，
// 不是 trust anchor 失败，所以保守断言只看错误的归属 + 反证不存在 unknown authority。
func TestNewHTTPClient_NoServerName_IPClient_FailsHostnameSAN(t *testing.T) {
	dir := t.TempDir()

	// 1) 同 TestNewHTTPClient_ServerNameSet_HostnameVerifiesByDNS：self-signed root +
	//    DNS-only SAN server cert（无 IPAddresses）
	rootPEM, rootKeyPEM := generateSelfSignedCert(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "huawei-redo-root-noservername"},
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	root, err := x509.ParseCertificate(parseFirstCertDER(t, rootPEM))
	require.NoError(t, err)
	rootKey := parsePKCS1PrivateKey(t, rootKeyPEM)
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	serverTmpl := &x509.Certificate{
		Subject:     pkix.Name{CommonName: "vct.tp.huawei.com"},
		DNSNames:    []string{"vct.tp.huawei.com"},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	fillCertDefaults(serverTmpl)
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, root, &serverKey.PublicKey, rootKey)
	require.NoError(t, err)
	serverCert, err := x509.ParseCertificate(serverDER)
	require.NoError(t, err)

	rootPath := filepath.Join(dir, "root-noservername.pem")
	require.NoError(t, os.WriteFile(rootPath, rootPEM, 0o600))
	m := NewManager(zap.NewNop(), nil)
	require.NoError(t, m.SetCABundle(rootPath))
	require.NotNil(t, m.caCertPool)
	// 注意：故意不调用 SetTLSServerName —— 模拟回归 bug 现场。

	srv := startTLSServer(t, fakeHuaweiAPIHandler(), serverCert, serverKey)
	defer srv.Close()

	client := makeHuaweiClient(t, srv, m.caCertPool, "") // 不传 ServerName
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.httpClient.Post(ctx, "Web_RequestSessionID", nil)
	require.Error(t, err, "无 ServerName + IP-only dial + 无 IP SAN → 握手必须失败")

	errStr := err.Error()
	// 语义指纹 ①：错误栈来自 Go x509 校验层（cert chain + hostname 都走 x509 包）
	assert.Contains(t, errStr, "x509",
		"错误应来自 x509 校验栈（Go 任何 hostname/chain 校验错误的固定前缀）")
	// 语义指纹 ②（反证）：不是 PRIOR 错误假设的 unknown authority
	// ——bug 的真实断点是 hostname/SAN，CA chain 验证实际 OK。
	// 该断言与 Go 版本无关：unknown authority 文本在 Go x509 多年都是 stable 的。
	assert.NotContains(t, errStr, "unknown authority",
		"REDO 反证：当前 root cause 不是 CA chain trust，而是 hostname 校验；"+
			"旧假设的 unknown authority 文本不应再出现")
}

// TestSetTLSServerName_TrimWhitespace — review follow-up：SetTLSServerName 入参必须
// 走 strings.TrimSpace 归一化，前后空白被剥离；trim 后空字符串视作"清空" opt-out，
// 与 ca_bundle_file 同语义。运维常因 YAML 缩进或复制粘贴引入意外空格，桥接层必须
// 能容忍。如果回归到"原样落库"，会导致 dial 地址 vs ServerName 字面不匹配 → 握手
// 失败，且无明确错误指向，debug 成本最高。
func TestSetTLSServerName_TrimWhitespace(t *testing.T) {
	m := NewManager(zap.NewNop(), nil)

	t.Run("leading_and_trailing_whitespace_trimmed", func(t *testing.T) {
		m.SetTLSServerName("   vct.tp.huawei.com   ")
		assert.Equal(t, "vct.tp.huawei.com", m.tlsServerName,
			"前后空白必须被 TrimSpace 剥离")
	})

	t.Run("whitespace_only_is_cleared", func(t *testing.T) {
		// 先生成一个值证明后续 trim 能清掉
		m.SetTLSServerName("vct.tp.huawei.com")
		require.Equal(t, "vct.tp.huawei.com", m.tlsServerName)

		m.SetTLSServerName("   \t\n  ")
		assert.Equal(t, "", m.tlsServerName,
			"全空白字符串 trim 后视作清空（与 ca_bundle_file 同语义）")
	})

	t.Run("clean_value_passes_through", func(t *testing.T) {
		m.SetTLSServerName("vct.tp.huawei.com")
		assert.Equal(t, "vct.tp.huawei.com", m.tlsServerName,
			"无空白的字面值不应被改动")
	})
}

// TestSetTLSServerName_PropagatesToNewHTTPClientThroughManager — **review [blocking]**
// 桥接回归测试：通过 stubDB 让 Manager.GetClient 走完整 createClient 路径，让真的
// httptest server 与 client 进行 TLS 握手。仅当 Manager.SetTLSServerName →
// createClient 配置构造（tlsServerName: m.tlsServerName）→ NewHuaweiClient →
// NewHTTPClient → tls.Config.ServerName 这条桥接完整正确时，握手才能成功。
//
// 现有 TestNewHTTPClient_ServerNameSet_HostnameVerifiesByDNS 用 makeHuaweiClient
// 直接构造 Config 并把 tlsServerName 当字面量传进去，绕开了 Manager 的桥接 ——
// 若 manager.go createClient 中的 `tlsServerName: m.tlsServerName` 行被误删/改坏，
// 该测试仍保持 GREEN。本测试就是为了补齐这一覆盖漏洞：它会在桥接断点出现时
// 必然 FAIL，是 review [blocking] 项的核心验证。
//
// 敏感性证明（手动切换 RED→GREEN）：
//  1. 把 manager.go 的 `tlsServerName: name,` 改成 `tlsServerName: "",`，
//  2. 跑 `go test -count=1 -run TestSetTLSServerName_PropagatesToNewHTTPClientThroughManager ./internal/huawei/`
//     → 必 FAIL（handshake / InitializeAndStartKeepAlive 在 GetSessionID 阶段返回 hostname error）。
//  3. 恢复原行，同命令 → PASS。
//
// 详细 RED/GREEN 输出见 debug session 的 Evidence 区。
func TestSetTLSServerName_PropagatesToNewHTTPClientThroughManager(t *testing.T) {
	dir := t.TempDir()

	// 1) 自签 root + DNS-only SAN server cert（CN / DNS = vct.tp.huawei.com；无
	//    IPAddresses，模拟生产 cert 结构）。client dial 127.0.0.1 时若不显式设
	//    ServerName 必失败；显式设 ServerName="vct.tp.huawei.com" 后走 dNSName 路径成功。
	rootPEM, rootKeyPEM := generateSelfSignedCert(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "huawei-bridge-root"},
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	root, err := x509.ParseCertificate(parseFirstCertDER(t, rootPEM))
	require.NoError(t, err)
	rootKey := parsePKCS1PrivateKey(t, rootKeyPEM)
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	serverTmpl := &x509.Certificate{
		Subject:  pkix.Name{CommonName: "vct.tp.huawei.com"},
		DNSNames: []string{"vct.tp.huawei.com"},
		// 故意不设 IPAddresses —— 钉住"无 IP SAN 时 IP-only dial 必须靠 ServerName"
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	fillCertDefaults(serverTmpl)
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, root, &serverKey.PublicKey, rootKey)
	require.NoError(t, err)
	serverCert, err := x509.ParseCertificate(serverDER)
	require.NoError(t, err)

	// 2) root 写文件 → SetCABundle 注入 pool
	rootPath := filepath.Join(dir, "bridge-root.pem")
	require.NoError(t, os.WriteFile(rootPath, rootPEM, 0o600))

	// 3) httptest TLS server —— 必须返回 SessionID cookie（无论 login/keep-alive
	//    路径都能拿到），否则 InitializeAndStartKeepAlive 会在 GetSessionID 报
	//    "未能获取到会话ID"，无法验证 TLS 桥接。
	bridgeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "SessionID",
			Value: "bridge-test-session",
			Path:  "/",
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": 1, "data": ""})
	})
	srv := startTLSServer(t, bridgeHandler, serverCert, serverKey)
	defer srv.Close()

	// 4) 解析 server URL 给 stubDB
	urlStr := strings.TrimPrefix(srv.URL, "https://")
	parts := strings.SplitN(urlStr, ":", 2)
	require.Len(t, parts, 2, "URL 应含端口")
	port, err := strconv.Atoi(parts[1])
	require.NoError(t, err)

	// 5) stubDB 把 Manager.GetClient 拉进 httptest server
	db := &stubDB{cfg: &HuaweiConfigDB{
		ID:             1,
		Server:         parts[0],
		Port:           port,
		Username:       "user",
		Password:       "pass",
		TerminalNumber: "tn1",
	}}

	// 6) Manager 全装配 —— 走与生产 cmd/server 启动期一致的 setter 链路
	m := NewManager(zap.NewNop(), db)
	require.NoError(t, m.SetCABundle(rootPath), "root PEM 应被正常加载")
	require.NotNil(t, m.caCertPool)
	m.SetTLSPolicy(false, tls.VersionTLS12, false) // 非生产环境，避免任何 fatal 触发
	m.SetTLSServerName("vct.tp.huawei.com")        // **核心：要测的桥接起点**
	m.SetOutboundURLAllowlist(nil, "development")  // 绕过 SEC-013 出站白名单

	// 7) GetClient → createClient → NewHTTPClient with ServerName → 真实握手
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := m.GetClient(ctx, 1)
	require.NoError(t, err,
		"Manager→Config→NewHTTPClient 桥接必须正确；"+
			"否则 m.tlsServerName 在 createClient 中未被读到 tls.Config.ServerName，"+
			"IP-only dial 在 InitializeAndStartKeepAlive 的 GetSessionID 阶段会 hostname 报错")
	require.NotNil(t, client)

	// 8) 直接断言最终 tls.Config.ServerName 也已注入（更靠近 review 要求的"透传到
	//    NewHTTPClient" 措辞）—— 单元级 fidelity 比单跑握手更稳：即使 fake handler
	//    因为其他原因返回成功，这里也能捕获"字段未被赋值"的退化场景。
	tr, ok := client.httpClient.client.Transport.(*http.Transport)
	require.True(t, ok, "Transport 应为 *http.Transport")
	require.NotNil(t, tr.TLSClientConfig)
	assert.Equal(t, "vct.tp.huawei.com", tr.TLSClientConfig.ServerName,
		"tls.Config.ServerName 必须从 Manager.SetTLSServerName 透传到 tls.Config")
	assert.False(t, tr.TLSClientConfig.InsecureSkipVerify,
		"InsecureSkipVerify 必须保持 false（review hard_constraint）")

	// 9) 关停 keep-alive goroutine，避免 runtime 抖动污染 Go race 测试
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	_ = client.Stop(stopCtx)
}
