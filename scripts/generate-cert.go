//go:build ignore
// +build ignore

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

func main() {
	// 生成 ECDSA 私钥（P-256）
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Printf("Failed to generate private key: %v\n", err)
		os.Exit(1)
	}

	// 设置证书有效期（10年）
	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour)

	// 生成序列号
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		fmt.Printf("Failed to generate serial number: %v\n", err)
		os.Exit(1)
	}

	// 解析 IP 地址
	ip127 := net.ParseIP("127.0.0.1")
	ipLocalhost := net.ParseIP("::1")

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Record V2 Internal"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		// 添加多个 DNS 名称和 IP 地址
		DNSNames:    []string{"localhost", "*.localhost", "record.local"},
		IPAddresses: []net.IP{ip127, ipLocalhost},
	}

	// 自签名证书
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		fmt.Printf("Failed to create certificate: %v\n", err)
		os.Exit(1)
	}

	// 创建证书目录
	certDir := "./certs"
	if err := os.MkdirAll(certDir, 0755); err != nil {
		fmt.Printf("Failed to create certs directory: %v\n", err)
		os.Exit(1)
	}

	// 写入证书文件
	certOut, err := os.Create(fmt.Sprintf("%s/server.crt", certDir))
	if err != nil {
		fmt.Printf("Failed to create cert file: %v\n", err)
		os.Exit(1)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		fmt.Printf("Failed to write cert: %v\n", err)
		os.Exit(1)
	}

	// 写入私钥文件
	keyOut, err := os.OpenFile(fmt.Sprintf("%s/server.key", certDir), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Printf("Failed to create key file: %v\n", err)
		os.Exit(1)
	}
	defer keyOut.Close()
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		fmt.Printf("Failed to marshal private key: %v\n", err)
		os.Exit(1)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		fmt.Printf("Failed to write key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 自签名证书生成成功!")
	fmt.Printf("📁 证书目录: %s\n", certDir)
	fmt.Println("   - server.crt (证书)")
	fmt.Println("   - server.key (私钥)")
	fmt.Println("\n⚠️  注意: 浏览器会显示安全警告，这是正常的（自签名证书）")
	fmt.Println("   在浏览器中选择「高级 → 继续访问」即可")
}
