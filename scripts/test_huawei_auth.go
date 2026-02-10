package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type APIResponse struct {
	Success int    `json:"success"`
	Data    string `json:"data"`
	Error    struct {
		ID   int    `json:"id"`
		Code int    `json:"code"`
	} `json:"error,omitempty"`
}

type SessionIDResponse struct {
	AcSessionID string `json:"acSessionId"`
	SzTermType  string `json:"szTermType"`
}

func main() {
	baseURL := "https://10.62.10.3:443"
	username := "api"
	password := "Hubei@1992"

	fmt.Println("=== 华为终端认证流程测试 ===")
	fmt.Printf("目标: %s\n", baseURL)
	fmt.Printf("用户名: %s\n\n", username)

	// 创建HTTP客户端（跳过TLS验证，使用正确的密码套件）
	// 华为终端需要 TLS 1.0-1.2 版本范围和特定的密码套件
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS10, // TLS 1.0
				MaxVersion:         tls.VersionTLS12, // 限制最大版本为 TLS 1.2
				// 华为终端支持的密码套件（尽可能多地包含以兼容老设备）
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
				},
				// 使用支持所有密码套件的曲线偏好
				CurvePreferences: []tls.CurveID{
					tls.CurveP256,
					tls.CurveP384,
					tls.CurveP521,
				},
			},
		},
	}

	// 步骤1: 获取会话ID
	fmt.Println("步骤1: 获取会话ID (Web_RequestSessionID)")
	sessionURL := baseURL + "/action.cgi?ActionID=Web_RequestSessionID"
	req1, _ := http.NewRequest("POST", sessionURL, nil)
	req1.Header.Set("userType", "web")

	resp1, err := client.Do(req1)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return
	}
	defer resp1.Body.Close()

	body1, _ := io.ReadAll(resp1.Body)
	fmt.Printf("  响应状态: %d\n", resp1.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(body1))

	// 解析响应
	var resp1Data APIResponse
	json.Unmarshal(body1, &resp1Data)

	if resp1Data.Success != 1 {
		fmt.Printf("  获取会话ID失败\n")
		return
	}

	// 从data字段解析会话ID
	var sessionResp SessionIDResponse
	json.Unmarshal([]byte(resp1Data.Data), &sessionResp)

	var sessionID string
	if sessionResp.AcSessionID != "" {
		sessionID = sessionResp.AcSessionID
		fmt.Printf("  从data获取会话ID: %s\n", maskString(sessionID))
	}

	// 从Cookie中提取会话ID
	if sessionID == "" {
		for _, cookie := range resp1.Cookies() {
			if cookie.Name == "SessionID" {
				sessionID = cookie.Value
				fmt.Printf("  从Cookie获取会话ID: %s\n", maskString(sessionID))
				break
			}
		}
	}

	if sessionID == "" {
		fmt.Printf("  错误: 未能获取到会话ID\n")
		return
	}

	fmt.Printf("  最终会话ID: %s\n", maskString(sessionID))
	fmt.Println()

	// 步骤2: 用户认证
	fmt.Println("步骤2: 用户认证 (WEB_RequestCertificateAPI)")
	authURL := baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI"

	// 注意：华为API需要小写的user和password字段名
	authPayload := fmt.Sprintf(`{"user":"%s","password":"%s"}`, username, password)
	req2, _ := http.NewRequest("POST", authURL, strings.NewReader(authPayload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req2.Header.Set("userType", "web")
	req2.Header.Set("Cookie", fmt.Sprintf("SessionID=%s", sessionID))

	fmt.Printf("  请求Cookie: SessionID=%s\n", maskString(sessionID))

	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("  响应状态: %d\n", resp2.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(body2))

	var resp2Data APIResponse
	json.Unmarshal(body2, &resp2Data)

	if resp2Data.Success == 1 {
		fmt.Printf("  *** 认证成功! ***\n")
	} else {
		fmt.Printf("  认证失败，错误码: %d\n", resp2Data.Error.ID)
	}
	fmt.Println()

	// 步骤3: 替换会话ID
	if resp2Data.Success == 1 {
		fmt.Println("步骤3: 替换会话ID (WEB_ChangeSessionID)")
		changeURL := baseURL + "/action.cgi?ActionID=WEB_ChangeSessionID"

		req3, _ := http.NewRequest("POST", changeURL, nil)
		req3.Header.Set("userType", "web")
		req3.Header.Set("Cookie", fmt.Sprintf("SessionID=%s", sessionID))

		resp3, err := client.Do(req3)
		if err != nil {
			fmt.Printf("  请求失败: %v\n", err)
			return
		}
		defer resp3.Body.Close()

		body3, _ := io.ReadAll(resp3.Body)
		fmt.Printf("  响应状态: %d\n", resp3.StatusCode)
		fmt.Printf("  响应内容: %s\n", string(body3))

		var resp3Data APIResponse
		json.Unmarshal(body3, &resp3Data)

		if resp3Data.Success == 1 {
			fmt.Printf("  *** 替换会话ID成功! ***\n")
		} else {
			fmt.Printf("  替换失败，错误码: %d\n", resp3Data.Error.ID)
		}
	}

	fmt.Println("\n=== 测试完成 ===")
}

// maskString 隐藏敏感信息
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
