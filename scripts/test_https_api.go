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
	// 尝试不同的HTTPS端口
	ports := []int{443, 8443, 9443}

	for _, port := range ports {
		baseURL := fmt.Sprintf("https://10.62.10.3:%d", port)
		fmt.Printf("=== 测试HTTPS端口 %d ===\n", port)

		// 创建跳过TLS验证的客户端
		client := &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					MinVersion:         tls.VersionTLS10,
				},
			},
		}

		// 测试获取会话ID
		sessionURL := baseURL + "/action.cgi?ActionID=Web_RequestSessionID"

		req, _ := http.NewRequest("POST", sessionURL, nil)
		req.Header.Set("userType", "web")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  连接失败: %v\n\n", err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("  响应: %s\n", string(body))

		var sessionResp Response
		json.Unmarshal(body, &sessionResp)

		if sessionResp.Success == 1 || sessionResp.Exception.ID != 4 {
			fmt.Printf("  *** 端口 %d 可用! ***\n\n", port)
			if sessionResp.Success == 1 {
				testFullLogin(baseURL, client)
				return
			}
		} else {
			fmt.Printf("  错误码: %d\n\n", sessionResp.Exception.ID)
		}
	}

	fmt.Println("所有HTTPS端口测试完成")
}

func testFullLogin(baseURL string, client *http.Client) {
	username := "api"
	password := "Changeme_123"

	fmt.Println("\n=== 完整登录流程测试 ===")

	// 1. 获取会话ID
	fmt.Println("1. 获取会话ID...")
	sessionURL := baseURL + "/action.cgi?ActionID=Web_RequestSessionID"

	req, _ := http.NewRequest("POST", sessionURL, nil)
	req.Header.Set("userType", "web")

	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var sessionResp Response
	json.Unmarshal(body, &sessionResp)

	var sessionID string
	if sessionResp.Success == 1 {
		sessionID = sessionResp.Session
		fmt.Printf("   会话ID: %s ✓\n", sessionID)
	}

	// 2. 登录
	fmt.Println("2. 登录...")
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

	loginResp, _ := client.Do(loginReq)
	loginBody, _ := io.ReadAll(loginResp.Body)
	loginResp.Body.Close()

	fmt.Printf("   响应: %s\n", string(loginBody))

	var loginResult Response
	json.Unmarshal(loginBody, &loginResult)

	if loginResult.Success == 1 {
		fmt.Printf("   登录成功! ✓\n")
	} else {
		fmt.Printf("   登录失败，错误码: %d\n", loginResult.Exception.ID)
	}

	// 3. 获取站点信息
	fmt.Println("3. 获取站点信息...")
	siteURL := baseURL + "/action.cgi?ActionID=WEB_GetCurrentSiteInfo"

	siteReq, _ := http.NewRequest("POST", siteURL, nil)
	siteReq.Header.Set("userType", "web")

	siteResp, _ := client.Do(siteReq)
	siteBody, _ := io.ReadAll(siteResp.Body)
	siteResp.Body.Close()

	fmt.Printf("   响应: %s\n", string(siteBody))

	var siteResult Response
	json.Unmarshal(siteBody, &siteResult)

	if siteResult.Success == 1 {
		fmt.Printf("   *** API调用成功! ***\n")
	}
}
