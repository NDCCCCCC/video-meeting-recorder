//go:build ignore
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type APIResponse struct {
	Success   int `json:"success"`
	Exception struct {
		ID int `json:"id"`
	} `json:"exception"`
}

type SessionIDResponse struct {
	Success int `json:"success"`
	Session string `json:"session"`
	Exception struct {
		ID int `json:"id"`
	} `json:"exception"`
}

type LoginResponse struct {
	Success int `json:"success"`
	Session string `json:"session"`
	Exception struct {
		ID int `json:"id"`
	} `json:"exception"`
}

func main() {
	// 华为终端配置 (使用HTTP)
	server := "10.62.10.3"
	port := 80
	username := "api"
	password := "Changeme_123"

	baseURL := fmt.Sprintf("http://%s:%d", server, port)

	fmt.Println("=== 华为终端HTTP API测试 (修正版) ===")
	fmt.Printf("目标: %s\n", baseURL)
	fmt.Printf("用户名: %s\n", username)
	fmt.Println()

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 1. 测试获取会话ID
	fmt.Println("步骤1: 获取会话ID (Web_RequestSessionID)...")
	sessionURL := fmt.Sprintf("%s/action.cgi?ActionID=Web_RequestSessionID", baseURL)

	resp, err := httpClient.Post(sessionURL, "application/json", nil)
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

	fmt.Printf("  响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(body))

	var sessionResp SessionIDResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		fmt.Printf("  解析JSON失败: %v\n", err)
		return
	}

	var sessionID string
	if sessionResp.Success == 0 && sessionResp.Session != "" {
		sessionID = sessionResp.Session
		fmt.Printf("  会话ID: %s ✓\n", sessionID)
	} else {
		fmt.Printf("  获取会话ID失败，错误码: %d\n", sessionResp.Exception.ID)
		// 使用默认会话ID继续
		sessionID = ""
	}
	fmt.Println()

	// 2. 测试登录 (使用POST表单)
	fmt.Println("步骤2: 登录 (WEB_RequestCertificateAPI)...")
	loginURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_RequestCertificateAPI", baseURL)

	formData := url.Values{
		"User":     {username},
		"Password": {password},
	}
	if sessionID != "" {
		formData.Set("SessionID", sessionID)
	}

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

	fmt.Printf("  响应状态码: %d\n", loginResp.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(loginBody))

	var loginResult LoginResponse
	if err := json.Unmarshal(loginBody, &loginResult); err == nil {
		if loginResult.Success == 1 {
			fmt.Printf("  登录成功! ✓\n")
			if loginResult.Session != "" {
				sessionID = loginResult.Session
			}
		} else {
			fmt.Printf("  登录失败，错误码: %d\n", loginResult.Exception.ID)
		}
	}
	fmt.Println()

	// 3. 测试获取会议信息
	fmt.Println("步骤3: 获取会议信息 (WEB_GetCurrentSiteInfo)...")
	infoURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_GetCurrentSiteInfo", baseURL)

	infoResp, err := httpClient.Post(infoURL, "application/json", nil)
	if err != nil {
		fmt.Printf("  获取会议信息失败: %v\n", err)
		return
	}
	defer infoResp.Body.Close()

	infoBody, _ := io.ReadAll(infoResp.Body)
	fmt.Printf("  响应状态码: %d\n", infoResp.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(infoBody))
	fmt.Println()

	// 4. 测试获取终端状态
	fmt.Println("步骤4: 获取终端状态 (WEB_GetCurrentSiteStatus)...")
	statusURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_GetCurrentSiteStatus", baseURL)

	statusResp, err := httpClient.Post(statusURL, "application/json", nil)
	if err != nil {
		fmt.Printf("  获取终端状态失败: %v\n", err)
		return
	}
	defer statusResp.Body.Close()

	statusBody, _ := io.ReadAll(statusResp.Body)
	fmt.Printf("  响应状态码: %d\n", statusResp.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(statusBody))
	fmt.Println()

	// 5. 尝试测试会议号 (模拟呼叫会议)
	fmt.Println("步骤5: 测试会议号验证...")
	confNumber := "521270003"
	confURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_DialInConfNew&ConfID=%s&SiteURI=%s",
		baseURL, confNumber, confNumber)

	confResp, err := httpClient.Post(confURL, "application/json", nil)
	if err != nil {
		fmt.Printf("  呼叫会议请求失败: %v\n", err)
		return
	}
	defer confResp.Body.Close()

	confBody, _ := io.ReadAll(confResp.Body)
	fmt.Printf("  响应状态码: %d\n", confResp.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(confBody))
	fmt.Println()

	// 6. 测试带JSON体的请求
	fmt.Println("步骤6: 测试JSON格式请求...")
	jsonURL := fmt.Sprintf("%s/action.cgi?ActionID=WEB_GetSiteList", baseURL)
	jsonData := map[string]interface{}{
		"ConfID": confNumber,
	}
	jsonBytes, _ := json.Marshal(jsonData)

	jsonResp, err := httpClient.Post(jsonURL, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		fmt.Printf("  JSON请求失败: %v\n", err)
		return
	}
	defer jsonResp.Body.Close()

	jsonResponseBody, _ := io.ReadAll(jsonResp.Body)
	fmt.Printf("  响应状态码: %d\n", jsonResp.StatusCode)
	fmt.Printf("  响应内容: %s\n", string(jsonResponseBody))
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
}
