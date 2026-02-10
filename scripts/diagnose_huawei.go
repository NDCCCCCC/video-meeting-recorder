package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"
)

func main() {
	host := "10.62.10.3"
	port := 443

	fmt.Println("=== 华为终端连接诊断 ===")
	fmt.Printf("目标: %s:%d\n\n", host, port)

	// 1. 测试TCP连接
	fmt.Println("1. 测试TCP连接...")
	tcpConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		fmt.Printf("   TCP连接失败: %v\n", err)
		if opErr, ok := err.(*net.OpError); ok {
			if syscallErr, ok := opErr.Err.(*syscall.Errno); ok {
				fmt.Printf("   系统错误码: %d\n", *syscallErr)
			}
		}
		fmt.Println()
		fmt.Println("可能原因:")
		fmt.Println("- 华为终端IP地址不正确")
		fmt.Println("- 端口号不正确（可能不是443）")
		fmt.Println("- 网络不通（防火墙、路由）")
		fmt.Println("- 华为终端未开机或服务未运行")
		return
	}
	tcpConn.Close()
	fmt.Println("   TCP连接成功 ✓")
	fmt.Println()

	// 2. 测试不同TLS版本
	fmt.Println("2. 测试不同TLS版本...")
	tlsVersions := []struct {
		name   string
		version uint16
	}{
		{"TLS 1.0", tls.VersionTLS10},
		{"TLS 1.1", tls.VersionTLS11},
		{"TLS 1.2", tls.VersionTLS12},
		{"TLS 1.3", tls.VersionTLS13},
	}

	for _, tv := range tlsVersions {
		fmt.Printf("   测试 %s... ", tv.name)

		conn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: 5 * time.Second},
			"tcp",
			fmt.Sprintf("%s:%d", host, port),
			&tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tv.version,
				MaxVersion:         tv.version,
			},
		)

		if err != nil {
			fmt.Printf("失败: %v\n", err)
		} else {
			state := conn.ConnectionState()
			fmt.Printf("成功! 实际使用: %x\n", state.Version)
			conn.Close()

			// 如果连接成功，尝试发送HTTP请求
			fmt.Printf("   发送测试请求... ")
			testHTTPRequest(tv.version)
			fmt.Println()
			return // 成功后退出
		}
	}
	fmt.Println()

	// 3. 测试HTTP（非HTTPS）
	fmt.Println("3. 测试HTTP连接（端口80）...")
	httpURL := fmt.Sprintf("http://%s:%d", host, 80)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(httpURL)
	if err == nil {
		fmt.Printf("   HTTP连接成功! 状态码: %d\n", resp.StatusCode)
		resp.Body.Close()
		fmt.Println("   建议: 华为终端可能使用HTTP而非HTTPS")
	} else {
		fmt.Printf("   HTTP连接失败: %v\n", err)
	}
	fmt.Println()

	// 4. 测试常见的替代端口
	fmt.Println("4. 测试其他常见端口...")
	ports := []int{80, 8443, 8888, 9443, 8080}
	for _, p := range ports {
		fmt.Printf("   测试端口 %d... ", p)
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, p), 2*time.Second)
		if err != nil {
			fmt.Printf("失败\n")
		} else {
			fmt.Printf("开放 ✓\n")
			conn.Close()
		}
	}
	fmt.Println()

	fmt.Println("=== 诊断完成 ===")
}

func testHTTPRequest(tlsVersion uint16) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tlsVersion,
				MaxVersion:         tlsVersion,
			},
		},
	}

	req, _ := http.NewRequest("GET", "https://10.62.10.3:443/", nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("状态码: %d, 内容: %s", resp.StatusCode, string(body))
}
