package main

import (
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

type LoginInfoResponse struct {
	Success int `json:"success"`
	Data struct {
		LoginURL string `json:"loginUrl"`
		WebType  string `json:"webType"`
	} `json:"data"`
	Exception struct {
		ID int `json:"id"`
	} `json:"exception"`
}

func main() {
	baseURL := "http://10.62.10.3:80"
	username := "api"
	password := "Changeme_123"

	fmt.Println("=== 华为终端API测试 (带正确Header) ===")
	fmt.Printf("目标: %s\n\n", baseURL)

	client := &http.Client{Timeout: 10 * time.Second}

	// 1. 获取会话ID（带userType header）
	fmt.Println("步骤1: 获取会话ID (Web_RequestSessionID)...")
	sessionURL := baseURL + "/action.cgi?ActionID=Web_RequestSessionID"

	req, _ := http.NewRequest("POST", sessionURL, nil)
	req.Header.Set("userType", "web")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
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
		// 尝试从cookie中获取
		for _, cookie := range resp.Cookies() {
			if strings.Contains(cookie.Name, "session") || strings.Contains(cookie.Name, "Session") {
				sessionID = cookie.Value
				fmt.Printf("  从Cookie获取会话ID: %s\n", sessionID)
				break
			}
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

	// 3. 切换会话
	if sessionID != "" {
		fmt.Println("步骤3: 切换会话 (WEB_ChangeSessionID)...")
		changeURL := baseURL + "/action.cgi?ActionID=WEB_ChangeSessionID"

		changeReq, _ := http.NewRequest("POST", changeURL, nil)
		changeReq.Header.Set("Cookie", "SessionID="+sessionID)
		changeReq.Header.Set("userType", "web")

		changeResp, err := client.Do(changeReq)
		if err != nil {
			fmt.Printf("  请求失败: %v\n", err)
		} else {
			defer changeResp.Body.Close()
			changeBody, _ := io.ReadAll(changeResp.Body)
			fmt.Printf("  响应: %s\n", string(changeBody))
		}
		fmt.Println()
	}

	// 4. 获取登录信息
	fmt.Println("步骤4: 获取登录信息 (WEB_GetLoginInfo)...")
	infoURL := baseURL + "/action.cgi?ActionID=WEB_GetLoginInfo"

	infoReq, _ := http.NewRequest("POST", infoURL, nil)
	infoReq.Header.Set("userType", "web")

	infoResp, err := client.Do(infoReq)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return
	}
	defer infoResp.Body.Close()

	infoBody, _ := io.ReadAll(infoResp.Body)
	fmt.Printf("  响应: %s\n", string(infoBody))

	var loginInfo LoginInfoResponse
	json.Unmarshal(infoBody, &loginInfo)
	fmt.Printf("  成功: %d\n", loginInfo.Success)
	fmt.Println()

	// 5. 获取当前站点信息
	fmt.Println("步骤5: 获取站点信息 (WEB_GetCurrentSiteInfo)...")
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
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
}
