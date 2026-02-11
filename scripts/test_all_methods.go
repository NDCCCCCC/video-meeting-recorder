//go:build ignore
package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func main() {
	baseURL := "http://10.62.10.3:80"

	fmt.Println("=== 华为终端API调用方式测试 ===")
	fmt.Printf("目标: %s\n\n", baseURL)

	client := &http.Client{Timeout: 5 * time.Second}

	// 测试不同的请求方式
	tests := []struct {
		name string
		method string
		url string
		contentType string
		body string
	}{
		// 1. GET请求测试
		{"GET请求", "GET", baseURL + "/action.cgi?ActionID=Web_RequestSessionID", "", ""},
		{"GET不带ActionID", "GET", baseURL + "/action.cgi", "", ""},
		{"根路径GET", "GET", baseURL + "/", "", ""},

		// 2. POST请求测试
		{"POST action.cgi", "POST", baseURL + "/action.cgi?ActionID=Web_RequestSessionID", "application/json", ""},
		{"POST application/x-www-form-urlencoded", "POST", baseURL + "/action.cgi?ActionID=Web_RequestSessionID", "application/x-www-form-urlencoded", ""},
		{"POST text/plain", "POST", baseURL + "/action.cgi?ActionID=Web_RequestSessionID", "text/plain", ""},

		// 3. 不同ActionID格式
		{"ActionID小写", "POST", baseURL + "/action.cgi?ActionID=web_requestsessionid", "application/json", ""},
		{"ActionID大写", "POST", baseURL + "/action.cgi?actionid=WEB_REQUESTSESSIONID", "application/json", ""},
		{"action参数", "POST", baseURL + "/action.cgi?action=Web_RequestSessionID", "application/json", ""},

		// 4. 带SessionID的请求
		{"带空SessionID", "POST", baseURL + "/action.cgi?ActionID=Web_RequestSessionID&SessionID=", "application/json", ""},
		{"带随机SessionID", "POST", baseURL + "/action.cgi?ActionID=Web_RequestSessionID&SessionID=123456", "application/json", ""},

		// 5. 尝试登录API
		{"登录-JSON格式", "POST", baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI", "application/json", `{"User":"api","Password":"Changeme_123"}`},
		{"登录-表单格式", "POST", baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI", "application/x-www-form-urlencoded", "User=api&Password=Changeme_123"},
		{"登录-Query参数", "POST", baseURL + "/action.cgi?ActionID=WEB_RequestCertificateAPI&User=api&Password=Changeme_123", "application/json", ""},

		// 6. 尝试其他常见API
		{"获取版本", "POST", baseURL + "/action.cgi?ActionID=WEB_GetVersion", "application/json", ""},
		{"获取能力", "POST", baseURL + "/action.cgi?ActionID=WEB_GetCapability", "application/json", ""},
		{"获取网络", "POST", baseURL + "/action.cgi?ActionID=WEB_GetNetworkParam", "application/json", ""},

		// 7. 尝试不同的路径
		{"api/action.cgi", "POST", baseURL + "/api/action.cgi?ActionID=Web_RequestSessionID", "application/json", ""},
		{"cgi-bin/action.cgi", "POST", baseURL + "/cgi-bin/action.cgi?ActionID=Web_RequestSessionID", "application/json", ""},
	}

	for _, test := range tests {
		fmt.Printf("[%s]\n", test.name)

		var bodyReader io.Reader
		if test.body != "" {
			bodyReader = strings.NewReader(test.body)
		}

		req, err := http.NewRequest(test.method, test.url, bodyReader)
		if err != nil {
			fmt.Printf("  创建请求失败: %v\n\n", err)
			continue
		}

		if test.contentType != "" {
			req.Header.Set("Content-Type", test.contentType)
		}

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  请求失败: %v\n\n", err)
			continue
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		elapsed := time.Since(start)

		// 截断过长的响应
		bodyStr := string(respBody)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}

		fmt.Printf("  状态: %d, 耗时: %v\n", resp.StatusCode, elapsed)
		fmt.Printf("  响应: %s\n", bodyStr)

		// 检查是否有成功标志
		if strings.Contains(string(respBody), `"success":1`) || strings.Contains(string(respBody), `"success": 1`) {
			fmt.Printf("  *** 成功! ***\n")
		}
		fmt.Println()
	}

	// 最后尝试：用浏览器方式访问
	fmt.Println("\n=== 测试用浏览器User-Agent ===")
	req, _ := http.NewRequest("GET", baseURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("状态: %d\n", resp.StatusCode)
		fmt.Printf("响应: %s\n", string(body))
	}
}
