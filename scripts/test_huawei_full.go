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
	Data string `json:"data"`
}

type SessionData struct {
	AcSessionID string `json:"acSessionId"`
	SzTermType  string `json:"szTermType"`
}

type LoginData struct {
	AcSessionID string `json:"acSessionId"`
}

func main() {
	baseURL := "https://10.62.10.3:443"
	username := "api"
	password := "Changeme_123"

	fmt.Println("=== 华为终端HTTPS API完整测试 ===")
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
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
			},
		},
	}

	// 1. 获取会话ID
	fmt.Println("步骤1: 获取会话ID (Web_RequestSessionID)...")
	sessionURL := baseURL + "/action.cgi?ActionID=Web_RequestSessionID"

	req, _ := http.NewRequest("POST", sessionURL, nil)
	req.Header.Set("userType", "web")

	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Printf("  响应: %s\n", string(body))

	var sessionResp Response
	json.Unmarshal(body, &sessionResp)

	var sessionID string
	if sessionResp.Success == 1 && sessionResp.Data != "" {
		var sessionData SessionData
		json.Unmarshal([]byte(sessionResp.Data), &sessionData)
		sessionID = sessionData.AcSessionID
		fmt.Printf("  会话ID: '%s' (长度: %d)\n", sessionID, len(sessionID))
		fmt.Printf("  终端类型: %s ✓\n", sessionData.SzTermType)
	}
	fmt.Println()

	// 2. 尝试不同的登录方式
	fmt.Println("步骤2: 尝试登录...")

	loginMethods := []struct {
		name string
		buildRequest func() (*http.Request, error)
	}{
		{
			"方法1: 表单格式",
			func() (*http.Request, error) {
				loginURL := baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI"
				formData := url.Values{
					"User":     {username},
					"Password": {password},
				}
				if sessionID != "" {
					formData.Set("SessionID", sessionID)
				}
				req, _ := http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set("userType", "web")
				return req, nil
			},
		},
		{
			"方法2: JSON格式",
			func() (*http.Request, error) {
				loginURL := baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI"
				jsonData := fmt.Sprintf(`{"User":"%s","Password":"%s"}`, username, password)
				if sessionID != "" {
					jsonData = fmt.Sprintf(`{"User":"%s","Password":"%s","SessionID":"%s"}`, username, password, sessionID)
				}
				req, _ := http.NewRequest("POST", loginURL, strings.NewReader(jsonData))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("userType", "web")
				return req, nil
			},
		},
		{
			"方法3: Query参数",
			func() (*http.Request, error) {
				loginURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_RequestCertificateAPI&User=%s&Password=%s",
					baseURL, username, password)
				if sessionID != "" {
					loginURL += "&SessionID=" + sessionID
				}
				req, _ := http.NewRequest("POST", loginURL, nil)
				req.Header.Set("userType", "web")
				return req, nil
			},
		},
		{
			"方法4: 无SessionID",
			func() (*http.Request, error) {
				loginURL := baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI"
				formData := url.Values{
					"User":     {username},
					"Password": {password},
				}
				req, _ := http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set("userType", "web")
				return req, nil
			},
		},
	}

	var workingSessionID string
	for _, method := range loginMethods {
		fmt.Printf("  %s...\n", method.name)
		req, err := method.buildRequest()
		if err != nil {
			fmt.Printf("    构建请求失败: %v\n", err)
			continue
		}

		loginResp, err := client.Do(req)
		if err != nil {
			fmt.Printf("    请求失败: %v\n", err)
			continue
		}
		loginBody, _ := io.ReadAll(loginResp.Body)
		loginResp.Body.Close()

		fmt.Printf("    响应: %s\n", string(loginBody))

		var loginResult Response
		json.Unmarshal(loginBody, &loginResult)

		if loginResult.Success == 1 {
			fmt.Printf("    *** 登录成功! ***\n")
			if loginResult.Data != "" {
				var loginData LoginData
				json.Unmarshal([]byte(loginResult.Data), &loginData)
				workingSessionID = loginData.AcSessionID
				fmt.Printf("    新会话ID: %s\n", workingSessionID)
			}
			break
		} else {
			fmt.Printf("    失败，错误码: %d\n", loginResult.Exception.ID)
		}
	}
	fmt.Println()

	// 3. 获取站点信息
	if workingSessionID != "" || sessionID != "" {
		fmt.Println("步骤3: 获取站点信息...")
		siteURL := baseURL + "/action.cgi?ActionID=WEB_GetCurrentSiteInfo"

		siteReq, _ := http.NewRequest("POST", siteURL, nil)
		siteReq.Header.Set("userType", "web")

		siteResp, _ := client.Do(siteReq)
		siteBody, _ := io.ReadAll(siteResp.Body)
		siteResp.Body.Close()

		fmt.Printf("  响应: %s\n", string(siteBody))

		var siteResult Response
		json.Unmarshal(siteBody, &siteResult)

		if siteResult.Success == 1 {
			fmt.Printf("  *** API调用成功! ***\n")
		} else {
			fmt.Printf("  错误码: %d\n", siteResult.Exception.ID)
		}
	}

	fmt.Println("\n=== 测试完成 ===")
}
