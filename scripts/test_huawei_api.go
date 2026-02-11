//go:build ignore
package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func main() {
	// 华为终端配置
	server := "10.62.10.3"
	port := 443
	username := "api"
	password := "Changeme_123"

	baseURL := fmt.Sprintf("https://%s:%d", server, port)

	fmt.Println("=== 华为终端API连接测试 ===")
	fmt.Printf("目标: %s\n", baseURL)
	fmt.Printf("用户名: %s\n", username)
	fmt.Println()

	// 1. 测试基本连接（跳过TLS验证）
	fmt.Println("步骤1: 测试基本连接...")
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS10,
			},
		},
	}

	// 2. 测试获取会话ID
	fmt.Println("步骤2: 测试获取会话ID...")
	sessionURL := fmt.Sprintf("%s/action.cgi?ActionID=Web_RequestSessionID", baseURL)
	req, err := http.NewRequest("POST", sessionURL, nil)
	if err != nil {
		fmt.Printf("  创建请求失败: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		fmt.Println()
		fmt.Println("=== 错误分析 ===")
		fmt.Println("TLS握手失败的可能原因:")
		fmt.Println("1. 华为终端未启用HTTPS或端口不正确")
		fmt.Println("2. 华为终端使用的是自签名证书（已跳过验证）")
		fmt.Println("3. 网络连接问题（防火墙、路由等）")
		fmt.Println("4. 华为终端服务未运行")
		fmt.Println()
		fmt.Println("建议检查:")
		fmt.Println("- 确认华为终端IP地址和端口是否正确")
		fmt.Println("- 使用浏览器访问 https://10.62.10.3:443 检查是否可访问")
		fmt.Println("- 检查华为终端服务状态")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("  读取响应失败: %v\n", err)
		return
	}

	fmt.Printf("  响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(body))
	fmt.Println()

	// 3. 测试登录
	if resp.StatusCode == 200 {
		fmt.Println("步骤3: 测试登录...")
		loginURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_RequestCertificateAPI", baseURL)

		// 构造登录表单
		formData := url.Values{}
		formData.Set("User", username)
		formData.Set("Password", password)
		formData.Set("SessionID", string(body)) // 使用上一步获取的SessionID

		loginReq, err := http.NewRequest("POST", loginURL, nil)
		if err != nil {
			fmt.Printf("  创建登录请求失败: %v\n", err)
			return
		}
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// 将表单数据设置为请求体
		loginReq.Body = io.NopCloser(nil)

		loginResp, err := httpClient.Do(loginReq)
		if err != nil {
			fmt.Printf("  登录请求失败: %v\n", err)
			return
		}
		defer loginResp.Body.Close()

		loginBody, err := io.ReadAll(loginResp.Body)
		if err != nil {
			fmt.Printf("  读取登录响应失败: %v\n", err)
			return
		}

		fmt.Printf("  登录响应状态码: %d\n", loginResp.StatusCode)
		fmt.Printf("  登录响应内容: %s\n", string(loginBody))
		fmt.Println()
	}

	fmt.Println("=== 测试完成 ===")
}
