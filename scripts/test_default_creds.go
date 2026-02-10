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
	Data    string `json:"data"`
	Exception struct {
		ID int `json:"id"`
	} `json:"exception"`
}

type SessionData struct {
	AcSessionID string `json:"acSessionId"`
	SzTermType  string `json:"szTermType"`
}

func testLogin(baseURL string, username, password string) bool {
	// 每次创建新的HTTP客户端和Transport
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				},
			},
		},
	}

	loginURL := baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI"
	formData := url.Values{
		"User":     {username},
		"Password": {password},
	}

	req, _ := http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("userType", "web")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("    请求失败: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result Response
	json.Unmarshal(body, &result)

	fmt.Printf("    %s:%s -> ", username, password)
	if result.Success == 1 {
		fmt.Printf("成功! ✓\n")
		if result.Data != "" {
			var data SessionData
			json.Unmarshal([]byte(result.Data), &data)
			if data.AcSessionID != "" {
				fmt.Printf("    会话ID: %s\n", data.AcSessionID)
			}
		}
		return true
	} else {
		fmt.Printf("失败 (错误码: %d)\n", result.Exception.ID)
		return false
	}
}

func main() {
	baseURL := "https://10.62.10.3:443"

	fmt.Println("=== 测试默认账户 ===")
	fmt.Printf("目标: %s\n\n", baseURL)

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				},
			},
		},
	}

	// 先获取会话
	fmt.Println("获取会话...")
	sessionURL := baseURL + "/action.cgi?ActionID=Web_RequestSessionID"
	req, _ := http.NewRequest("POST", sessionURL, nil)
	req.Header.Set("userType", "web")
	resp, err := client.Do(req)
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var sessionResp Response
		json.Unmarshal(body, &sessionResp)
		if sessionResp.Success == 1 {
			var data SessionData
			json.Unmarshal([]byte(sessionResp.Data), &data)
			fmt.Printf("终端类型: %s ✓\n\n", data.SzTermType)
		}
	}

	// 测试常见的默认账户
	credentials := []struct {
		username string
		password string
	}{
		{"api", "Hubei@1992"},  // 用户提供的正确账户
		{"admin", "admin"},
		{"admin", "Changeme_123"},
		{"api", "api"},
		{"api", "admin"},
		{"admin", ""},
		{"api", ""},
		{"admin", "123456"},
		{"admin", "password"},
		{"user", "user"},
	}

	fmt.Println("测试登录...")
	for _, cred := range credentials {
		testLogin(baseURL, cred.username, cred.password)
	}

	fmt.Println("\n提示: 如果所有默认账户都失败，请在华为终端的管理界面中确认API账户配置。")
}
