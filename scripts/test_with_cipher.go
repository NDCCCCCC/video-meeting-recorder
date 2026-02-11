//go:build ignore
package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Response struct {
	Success int `json:"success"`
	Session string `json:"session"`
	Exception struct {
		ID int `json:"id"`
	} `json:"exception"`
	Data json.RawMessage `json:"data,omitempty"`
}

func main() {
	baseURL := "https://10.62.10.3:443"
	username := "api"
	password := "Changeme_123"

	fmt.Println("=== 华为终端HTTPS API测试 (TLSv1.2 + AES256-SHA256) ===")
	fmt.Printf("目标: %s\n\n", baseURL)

	// 创建支持特定密码套件的客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS12,
				// 明确指定密码套件
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				},
			},
		},
	}

	// 1. 获取会话ID
	fmt.Println("步骤1: 获取会话ID (Web_RequestSessionID)...")
	sessionURL := baseURL + "/action.cgi?ActionID=Web_RequestSessionID"

	req, _ := http.NewRequest("POST", sessionURL, nil)
	req.Header.Set("userType", "web")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		fmt.Println("\n尝试使用curl测试...")
		testWithCurl()
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("  响应: %s\n", string(body))

	var sessionResp Response
	json.Unmarshal(body, &sessionResp)

	var sessionID string
	if sessionResp.Success == 1 {
		sessionID = sessionResp.Session
		fmt.Printf("  会话ID: %s ✓\n", sessionID)
	} else {
		fmt.Printf("  获取失败，错误码: %d\n", sessionResp.Exception.ID)
		// 检查cookie
		for _, cookie := range resp.Cookies() {
			fmt.Printf("  Cookie: %s=%s\n", cookie.Name, cookie.Value)
		}
	}
	fmt.Println()

	// 2. 登录
	fmt.Println("步骤2: 登录 (WEB_RequestCertificateAPI)...")
	loginURL := baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI"

	formData := url.Values{
		"User":     {username},
		"Password": {password},
	}
	if sessionID != "" {
		formData.Set("SessionID", sessionID)
	}

	loginReq, _ := http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("userType", "web")

	loginResp, err := client.Do(loginReq)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return
	}
	defer loginResp.Body.Close()

	loginBody, _ := io.ReadAll(loginResp.Body)
	fmt.Printf("  响应: %s\n", string(loginBody))

	var loginResult Response
	json.Unmarshal(loginBody, &loginResult)

	if loginResult.Success == 1 {
		fmt.Printf("  登录成功! ✓\n")
		if loginResult.Session != "" {
			sessionID = loginResult.Session
		}
	} else {
		fmt.Printf("  登录失败，错误码: %d\n", loginResult.Exception.ID)
	}
	fmt.Println()

	// 3. 获取站点信息
	fmt.Println("步骤3: 获取站点信息 (WEB_GetCurrentSiteInfo)...")
	siteURL := baseURL + "/action.cgi?ActionID=WEB_GetCurrentSiteInfo"

	siteReq, _ := http.NewRequest("POST", siteURL, nil)
	siteReq.Header.Set("userType", "web")

	siteResp, err := client.Do(siteReq)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return
	}
	defer siteResp.Body.Close()

	siteBody, _ := io.ReadAll(siteResp.Body)
	fmt.Printf("  响应: %s\n", string(siteBody))

	var siteResult Response
	json.Unmarshal(siteBody, &siteResult)

	if siteResult.Success == 1 {
		fmt.Printf("  *** API调用成功! ***\n")
	} else {
		fmt.Printf("  错误码: %d\n", siteResult.Exception.ID)
	}
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
}

func testWithCurl() {
	fmt.Println("使用curl测试...")
	// 这里可以提示用户使用curl测试
	fmt.Println("请尝试: curl -k -X POST 'https://10.62.10.3:443/action.cgi?ActionID=Web_RequestSessionID' -H 'userType: web'")
}
