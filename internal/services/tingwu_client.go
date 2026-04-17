package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"go.uber.org/zap"
)

// TingwuTaskStatus represents the status response from Tingwu API
type TingwuTaskStatus struct {
	TaskID       string  `json:"TaskId"`
	Status       string  `json:"TaskStatus"` // Queued, Processing, Completed, Failed
	Progress     float64 `json:"Progress"`   // 0-100
	ErrorMessage string  `json:"ErrorMessage"`
}

// TingwuTaskResult represents the transcription result from Tingwu API
type TingwuTaskResult struct {
	Text     string              `json:"Text"`
	Segments []TingwuTextSegment `json:"Segments"`
}

// TingwuTextSegment represents a text segment with timestamps
type TingwuTextSegment struct {
	Text      string `json:"Text"`
	BeginTime int    `json:"BeginTime"` // Milliseconds
	EndTime   int    `json:"EndTime"`   // Milliseconds
}

// TingwuSubmitResponse represents the response from submitting a task
type TingwuSubmitResponse struct {
	TaskID string `json:"TaskId"`
}

// TingwuClient handles Aliyun Tingwu API operations
type TingwuClient struct {
	appKey    string
	appSecret string
	baseURL   string
	client    *http.Client
	logger    *zap.Logger
	enabled   bool
}

// NewTingwuClient creates a new Tingwu API client
func NewTingwuClient(cfg *config.TingwuConfig, logger *zap.Logger) *TingwuClient {
	if !cfg.Enabled || cfg.AppKey == "" {
		logger.Info("Tingwu服务未启用或AppKey未配置")
		return &TingwuClient{
			logger:  logger,
			enabled: false,
		}
	}

	timeout := time.Duration(cfg.APITimeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &TingwuClient{
		appKey:    cfg.AppKey,
		appSecret: cfg.AppSecret,
		baseURL:   cfg.BaseURL,
		client:    &http.Client{Timeout: timeout},
		logger:    logger,
		enabled:   true,
	}
}

// IsEnabled returns whether Tingwu is configured and enabled
func (c *TingwuClient) IsEnabled() bool {
	return c.enabled
}

// SubmitTask submits a transcription task to Tingwu (per TRAN-01)
func (c *TingwuClient) SubmitTask(ctx context.Context, fileURL string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("Tingwu服务未启用")
	}

	body := map[string]interface{}{
		"file_url": fileURL,
		"version":  "4.0",
	}

	req, err := c.buildRequest(ctx, "POST", "/openapi/tingwu/v4/tasks", body)
	if err != nil {
		return "", fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("提交任务请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Sanitize error: never expose appKey or appSecret in error messages
		return "", fmt.Errorf("提交任务失败: HTTP %d", resp.StatusCode)
	}

	var result TingwuSubmitResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if result.TaskID == "" {
		return "", fmt.Errorf("Tingwu返回空TaskId")
	}

	c.logger.Info("Tingwu任务已提交", zap.String("task_id", result.TaskID))
	return result.TaskID, nil
}

// GetStatus queries the current status of a Tingwu task
func (c *TingwuClient) GetStatus(ctx context.Context, taskID string) (*TingwuTaskStatus, error) {
	if !c.enabled {
		return nil, fmt.Errorf("Tingwu服务未启用")
	}

	path := fmt.Sprintf("/openapi/tingwu/v4/tasks/%s", taskID)
	req, err := c.buildRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询状态请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询状态失败: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data TingwuTaskStatus `json:"Data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析状态响应失败: %w", err)
	}

	return &result.Data, nil
}

// GetResult retrieves the transcription result for a completed task (per TRAN-05)
func (c *TingwuClient) GetResult(ctx context.Context, taskID string) (*TingwuTaskResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("Tingwu服务未启用")
	}

	path := fmt.Sprintf("/openapi/tingwu/v4/tasks/%s/result", taskID)
	req, err := c.buildRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取结果请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取结果失败: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data TingwuTaskResult `json:"Data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析结果响应失败: %w", err)
	}

	return &result.Data, nil
}

// buildRequest builds an HTTP request with HMAC-SHA256 signature per Aliyun ROA spec
func (c *TingwuClient) buildRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	var contentMD5 string
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
		hash := md5.Sum(jsonBody)
		contentMD5 = base64.StdEncoding.EncodeToString(hash[:])
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	signature := c.calculateSignature(method, path, timestamp, contentMD5)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Date", timestamp)
	if contentMD5 != "" {
		req.Header.Set("Content-MD5", contentMD5)
	}
	req.Header.Set("Authorization", fmt.Sprintf("acs %s:%s", c.appKey, signature))

	return req, nil
}

// calculateSignature calculates HMAC-SHA256 signature per Aliyun ROA specification
func (c *TingwuClient) calculateSignature(method, path, timestamp, contentMD5 string) string {
	// StringToSign = METHOD\nContent-MD5\nContent-Type\nDate\nPath
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		strings.ToUpper(method),
		contentMD5,
		"application/json",
		timestamp,
		path,
	)

	h := hmac.New(sha256.New, []byte(c.appSecret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}
