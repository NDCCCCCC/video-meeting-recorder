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
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
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
	appKey               string
	appSecret            string
	baseURL              string
	client               *http.Client
	logger               *zap.Logger
	enabled              bool
	outboundURLAllowlist []string
	environment          string
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

// SetOutboundURLAllowlist 注入 SEC-013 出站 URL 白名单与运行环境。
// allowlist 元素为 host 后缀匹配（如 "aliyun.com"）；env=="development" 时
// allowlist 为空也允许所有出站（开发期绕过）；其他环境按白名单严格校验。
func (c *TingwuClient) SetOutboundURLAllowlist(allowlist []string, environment string) {
	c.outboundURLAllowlist = allowlist
	c.environment = environment
}

// guardOutboundURL 校验 baseURL 是否在出站白名单。SEC-013 SSRF 防御。
// Phase 19 D14: 4 URL config 错误全部包装 ErrTranscriptionUnavailable (503);
// 这些是配置/环境问题而非传输问题，应区分于 ErrTranscriptionFailed (500)。
func (c *TingwuClient) guardOutboundURL() error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("baseURL 解析失败: %w", apperrors.ErrTranscriptionUnavailable)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("baseURL 缺少 host: %s: %w", c.baseURL, apperrors.ErrTranscriptionUnavailable)
	}
	// 开发环境绕过白名单
	if c.environment == "development" {
		return nil
	}
	if len(c.outboundURLAllowlist) == 0 {
		return fmt.Errorf("outbound URL allowlist 为空且非开发环境，禁止访问: %s: %w", host, apperrors.ErrTranscriptionUnavailable)
	}
	for _, suffix := range c.outboundURLAllowlist {
		if strings.HasSuffix(host, suffix) {
			return nil
		}
	}
	return fmt.Errorf("URL 不在出站白名单: %s: %w", host, apperrors.ErrTranscriptionUnavailable)
}

// IsEnabled returns whether Tingwu is configured and enabled
func (c *TingwuClient) IsEnabled() bool {
	return c.enabled
}

// SubmitTask submits a transcription task to Tingwu (per TRAN-01)
// Phase 19 D14: 8 散点统一分类——未启用/配置 -> ErrTranscriptionUnavailable;
// HTTP/parse/状态码 -> 包装 ErrTranscriptionFailed (复用 D5)。
func (c *TingwuClient) SubmitTask(ctx context.Context, fileURL string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("Tingwu服务未启用: %w", apperrors.ErrTranscriptionUnavailable)
	}

	body := map[string]interface{}{
		"file_url": fileURL,
		"version":  "4.0",
	}

	req, err := c.buildRequest(ctx, "POST", "/openapi/tingwu/v4/tasks", body)
	if err != nil {
		return "", fmt.Errorf("构建请求失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("提交任务请求失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		// Sanitize error: never expose appKey or appSecret in error messages
		return "", fmt.Errorf("提交任务失败: HTTP %d: %w", resp.StatusCode, apperrors.ErrTranscriptionFailed)
	}

	var result TingwuSubmitResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}

	if result.TaskID == "" {
		return "", fmt.Errorf("Tingwu返回空TaskId: %w", apperrors.ErrTranscriptionFailed)
	}

	c.logger.Info("Tingwu任务已提交", zap.String("task_id", result.TaskID))
	return result.TaskID, nil
}

// GetStatus queries the current status of a Tingwu task
// Phase 19 D14: 6 散点统一分类（与 SubmitTask 同模式）。
func (c *TingwuClient) GetStatus(ctx context.Context, taskID string) (*TingwuTaskStatus, error) {
	if !c.enabled {
		return nil, fmt.Errorf("Tingwu服务未启用: %w", apperrors.ErrTranscriptionUnavailable)
	}

	path := fmt.Sprintf("/openapi/tingwu/v4/tasks/%s", taskID)
	req, err := c.buildRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询状态请求失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询状态失败: HTTP %d: %w", resp.StatusCode, apperrors.ErrTranscriptionFailed)
	}

	var result struct {
		Data TingwuTaskStatus `json:"Data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析状态响应失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}

	return &result.Data, nil
}

// GetResult retrieves the transcription result for a completed task (per TRAN-05)
// Phase 19 D14: 6 散点统一分类（与 SubmitTask 同模式）。
func (c *TingwuClient) GetResult(ctx context.Context, taskID string) (*TingwuTaskResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("Tingwu服务未启用: %w", apperrors.ErrTranscriptionUnavailable)
	}

	path := fmt.Sprintf("/openapi/tingwu/v4/tasks/%s/result", taskID)
	req, err := c.buildRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取结果请求失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取结果失败: HTTP %d: %w", resp.StatusCode, apperrors.ErrTranscriptionFailed)
	}

	var result struct {
		Data TingwuTaskResult `json:"Data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析结果响应失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
	}

	return &result.Data, nil
}

// buildRequest builds an HTTP request with HMAC-SHA256 signature per Aliyun ROA spec
func (c *TingwuClient) buildRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	// SEC-013: 出站 URL 白名单校验（SSRF 防御）
	if err := c.guardOutboundURL(); err != nil {
		return nil, err
	}
	url := c.baseURL + path

	var bodyReader io.Reader
	var contentMD5 string
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
		}
		bodyReader = bytes.NewReader(jsonBody)
		hash := md5.Sum(jsonBody)
		contentMD5 = base64.StdEncoding.EncodeToString(hash[:])
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w: %w", apperrors.ErrTranscriptionFailed, err)
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
