//go:build ignore
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func main() {
	// 华为终端配置 (使用HTTP)
	server := "10.62.10.3"
	port := 80
	username := "api"
	password := "Changeme_123"

	baseURL := fmt.Sprintf("http://%s:%d", server, port)

	fmt.Println("=== 华为终端HTTP API测试 ===")
	fmt.Printf("目标: %s\n", baseURL)
	fmt.Printf("用户名: %s\n", username)
	fmt.Println()

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 1. 测试获取会话ID
	fmt.Println("步骤1: 测试获取会话ID (Web_RequestSessionID)...")
	sessionURL := fmt.Sprintf("%s/action.cgi?ActionID=Web_RequestSessionID", baseURL)
	req, _ := http.NewRequest("POST", sessionURL, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("  读取响应失败: %v\n", err)
		return
	}

	sessionID := string(body)
	fmt.Printf("  响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("  会话ID: %s\n", sessionID)
	fmt.Println()

	// 2. 测试登录
	fmt.Println("步骤2: 测试登录 (WEB_RequestCertificateAPI)...")
	loginURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_RequestCertificateAPI", baseURL)

	// 构造登录表单数据
	formData := url.Values{}
	formData.Set("User", username)
	formData.Set("Password", password)
	formData.Set("SessionID", sessionID)

	loginReq, _ := http.NewRequest("POST", loginURL, nil)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 将表单数据作为请求体发送
	loginReq.Body = io.NopCloser(nil)

	loginResp, err := httpClient.PostForm(loginURL, formData)
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

	// 3. 测试获取会议信息
	fmt.Println("步骤3: 测试获取会议信息 (WEB_GetCurrentSiteInfo)...")
	infoURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_GetCurrentSiteInfo", baseURL)
	infoReq, _ := http.NewRequest("POST", infoURL, nil)

	infoResp, err := httpClient.Do(infoReq)
	if err != nil {
		fmt.Printf("  获取会议信息失败: %v\n", err)
		return
	}
	defer infoResp.Body.Close()

	infoBody, _ := io.ReadAll(infoResp.Body)
	fmt.Printf("  响应状态码: %d\n", infoResp.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(infoBody))
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
}
